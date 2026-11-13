package agent

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

// ---------------------------------------------------------------------------
// VCC ContextManager factory / registration tests
// ---------------------------------------------------------------------------

func TestVCCContextManager_RegistersAndResolves(t *testing.T) {
	cleanup := resetCMRegistry()
	defer cleanup()

	if err := RegisterContextManager("vcc", newVCCContextManager); err != nil {
		t.Fatalf("register vcc: %v", err)
	}
	f, ok := lookupContextManager("vcc")
	if !ok {
		t.Fatal("expected vcc factory to be registered")
	}
	if f == nil {
		t.Fatal("expected non-nil vcc factory")
	}

	cfg := testConfig(t)
	cfg.Agents.Defaults.ContextManager = "vcc"
	al := newCMTestAgentLoop(cfg)
	if _, ok := al.contextManager.(*vccContextManager); !ok {
		t.Fatalf("expected *vccContextManager, got %T", al.contextManager)
	}
}

func TestVCCContextManager_UnknownFallsBackToLegacy(t *testing.T) {
	cleanup := resetCMRegistry()
	defer cleanup()

	cfg := testConfig(t)
	cfg.Agents.Defaults.ContextManager = "unknown_cm"
	al := newCMTestAgentLoop(cfg)
	if _, ok := al.contextManager.(*legacyContextManager); !ok {
		t.Fatalf("expected fallback to *legacyContextManager, got %T", al.contextManager)
	}
}

func TestVCCContextManager_FactoryErrorFallsBackToLegacy(t *testing.T) {
	cleanup := resetCMRegistry()
	defer cleanup()

	factory := func(cfg json.RawMessage, al *AgentLoop) (ContextManager, error) {
		return nil, os.ErrPermission
	}
	if err := RegisterContextManager("broken_vcc", factory); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	cfg := testConfig(t)
	cfg.Agents.Defaults.ContextManager = "broken_vcc"
	al := newCMTestAgentLoop(cfg)
	if _, ok := al.contextManager.(*legacyContextManager); !ok {
		t.Fatalf("expected fallback to *legacyContextManager on factory error, got %T", al.contextManager)
	}
}

// ---------------------------------------------------------------------------
// VCC ContextManager behavior tests
// ---------------------------------------------------------------------------

func TestVCCCompact_ProducesSummary(t *testing.T) {
	cleanup := resetCMRegistry()
	defer cleanup()
	if err := RegisterContextManager("vcc", newVCCContextManager); err != nil {
		t.Fatalf("register vcc: %v", err)
	}

	cfg := testConfig(t)
	cfg.Agents.Defaults.ContextManager = "vcc"
	al := newCMTestAgentLoop(cfg)
	cm, ok := al.contextManager.(*vccContextManager)
	if !ok {
		t.Fatalf("expected *vccContextManager, got %T", al.contextManager)
	}

	sessionKey := "vcc-session-1"
	if err := cm.Ingest(context.Background(), &IngestRequest{
		SessionKey: sessionKey,
		Message: providers.Message{Role: "user", Content: "Fix the login bug"},
	}); err != nil {
		t.Fatalf("ingest user: %v", err)
	}
	if err := cm.Ingest(context.Background(), &IngestRequest{
		SessionKey: sessionKey,
		Message: providers.Message{Role: "assistant", Content: "I will fix it now"},
	}); err != nil {
		t.Fatalf("ingest assistant: %v", err)
	}

	if err := cm.Compact(context.Background(), &CompactRequest{
		SessionKey: sessionKey,
		Reason:     ContextCompressReasonSummarize,
	}); err != nil {
		t.Fatalf("compact: %v", err)
	}

	resp, err := cm.Assemble(context.Background(), &AssembleRequest{
		SessionKey: sessionKey,
	})
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	if resp.Summary == "" {
		t.Fatal("expected non-empty summary after compaction")
	}
	if !strings.Contains(resp.Summary, "[Session Goal]") && !strings.Contains(resp.Summary, "Use vcc_recall") {
		t.Fatalf("summary missing [Session Goal] or Use vcc_recall:\n%s", resp.Summary)
	}
}

func TestVCCClear_EmptiesSession(t *testing.T) {
	cleanup := resetCMRegistry()
	defer cleanup()
	if err := RegisterContextManager("vcc", newVCCContextManager); err != nil {
		t.Fatalf("register vcc: %v", err)
	}

	cfg := testConfig(t)
	cfg.Agents.Defaults.ContextManager = "vcc"
	al := newCMTestAgentLoop(cfg)
	cm, ok := al.contextManager.(*vccContextManager)
	if !ok {
		t.Fatalf("expected *vccContextManager, got %T", al.contextManager)
	}

	sessionKey := "vcc-session-clear"
	if err := cm.Ingest(context.Background(), &IngestRequest{
		SessionKey: sessionKey,
		Message: providers.Message{Role: "user", Content: "remember this"},
	}); err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if err := cm.Clear(context.Background(), sessionKey); err != nil {
		t.Fatalf("clear: %v", err)
	}

	resp, err := cm.Assemble(context.Background(), &AssembleRequest{
		SessionKey: sessionKey,
	})
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	if len(resp.History) != 0 {
		t.Fatalf("expected empty history after clear, got %d messages", len(resp.History))
	}
	if resp.Summary != "" {
		t.Fatalf("expected empty summary after clear, got %q", resp.Summary)
	}
}
