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

func TestVCCContextManager_ParsesSettings(t *testing.T) {
	al := newCMTestAgentLoop(testConfig(t))

	cm, err := newVCCContextManager(json.RawMessage(`{"smartKeepTail":true}`), al)
	if err != nil {
		t.Fatalf("newVCCContextManager: %v", err)
	}
	vcm, ok := cm.(*vccContextManager)
	if !ok {
		t.Fatalf("expected *vccContextManager, got %T", cm)
	}
	if !vcm.settings.SmartKeepTail {
		t.Fatal("expected SmartKeepTail parsed=true")
	}

	if _, err := newVCCContextManager(json.RawMessage(`{"bad":`), al); err == nil {
		t.Fatal("expected parse error for invalid JSON")
	}
}

// ---------------------------------------------------------------------------
// VCC ContextManager behavior tests
// ---------------------------------------------------------------------------

func vccStoreSession(t *testing.T, al *AgentLoop, key string) {
	t.Helper()
	agent := al.registry.GetDefaultAgent()
	if agent == nil || agent.Sessions == nil {
		t.Fatal("default agent sessions not initialized")
	}
	agent.Sessions.AddFullMessage(key, providers.Message{Role: "user", Content: "Fix the login bug"})
	agent.Sessions.AddFullMessage(key, providers.Message{Role: "assistant", Content: "I will fix it now"})
}

func TestVCCCompact_ProducesSummaryAndPersists(t *testing.T) {
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
	vccStoreSession(t, al, sessionKey)

	if err := cm.Compact(context.Background(), &CompactRequest{
		SessionKey: sessionKey,
		Reason:     ContextCompressReasonSummarize,
	}); err != nil {
		t.Fatalf("compact: %v", err)
	}

	resp, err := cm.Assemble(context.Background(), &AssembleRequest{SessionKey: sessionKey})
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	if resp.Summary == "" {
		t.Fatal("expected non-empty summary after compaction")
	}
	if !strings.Contains(resp.Summary, "[Session Goal]") && !strings.Contains(resp.Summary, "Use vcc_recall") {
		t.Fatalf("summary missing [Session Goal] or Use vcc_recall:\n%s", resp.Summary)
	}
	if len(resp.History) != 0 {
		t.Fatalf("expected raw history to be cleared after compaction, got %d messages", len(resp.History))
	}

	agent := al.registry.GetDefaultAgent()
	if got := agent.Sessions.GetSummary(sessionKey); got == "" {
		t.Fatal("expected summary persisted in SessionStore")
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
	vccStoreSession(t, al, sessionKey)

	if err := cm.Clear(context.Background(), sessionKey); err != nil {
		t.Fatalf("clear: %v", err)
	}

	resp, err := cm.Assemble(context.Background(), &AssembleRequest{SessionKey: sessionKey})
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
