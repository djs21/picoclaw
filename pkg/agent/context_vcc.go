// Package agent provides the VCC (Verbal Context Compression) ContextManager
// adapter. VCC is deterministic and never invokes an LLM.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	runtimeevents "github.com/sipeed/picoclaw/pkg/events"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/vcc"
)

// vccSession holds the per-session in-memory state for the VCC ContextManager.
type vccSession struct {
	mu       sync.Mutex
	messages []providers.Message
	summary  string
}

// vccContextManager adapts the deterministic VCC compiler to agent.ContextManager.
// It keeps raw messages per session in memory and compacts them into a summary
// via vcc.Compile. VCC never invokes an LLM.
type vccContextManager struct {
	mu      sync.Mutex
	al      *AgentLoop
	sessions map[string]*vccSession
}

// newVCCContextManager creates a VCC-backed ContextManager.
func newVCCContextManager(_ json.RawMessage, al *AgentLoop) (ContextManager, error) {
	if al == nil {
		return nil, fmt.Errorf("vcc: AgentLoop is required")
	}
	return &vccContextManager{
		al:      al,
		sessions: make(map[string]*vccSession),
	}, nil
}

// session returns the per-session state for sessionKey, creating it on first use.
func (m *vccContextManager) session(sessionKey string) *vccSession {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sessions == nil {
		m.sessions = make(map[string]*vccSession)
	}
	s, ok := m.sessions[sessionKey]
	if !ok {
		s = &vccSession{}
		m.sessions[sessionKey] = s
	}
	return s
}

// Assemble returns the stored summary for the session. When no summary exists
// yet, it returns the stored raw messages as History. VCC never invokes an LLM.
func (m *vccContextManager) Assemble(ctx context.Context, req *AssembleRequest) (*AssembleResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("vcc assemble: nil request")
	}
	s := m.session(req.SessionKey)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.summary != "" {
		return &AssembleResponse{
			Summary: s.summary,
		}, nil
	}
	history := make([]providers.Message, len(s.messages))
	copy(history, s.messages)
	return &AssembleResponse{
		History: history,
	}, nil
}

// Compact compresses the session's stored messages into a summary via vcc.Compile.
// On success the summary is stored and the raw messages are cleared.
func (m *vccContextManager) Compact(ctx context.Context, req *CompactRequest) error {
	if req == nil {
		return nil
	}
	s := m.session(req.SessionKey)
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.messages) == 0 {
		return nil
	}
	raw, err := json.Marshal(s.messages)
	if err != nil {
		return fmt.Errorf("vcc compact: marshal messages: %w", err)
	}
	var msgs []map[string]interface{}
	if err := json.Unmarshal(raw, &msgs); err != nil {
		return fmt.Errorf("vcc compact: unmarshal messages: %w", err)
	}
	summary := vcc.Compile(vcc.CompileInput{
		Messages:         msgs,
		PreviousSummary:  s.summary,
	})
	if summary == "" {
		return nil
	}
	dropped := len(s.messages)
	s.summary = summary
	s.messages = nil
	m.al.emitEvent(
		runtimeevents.KindAgentContextCompress,
		m.al.newTurnEventScope("", req.SessionKey, nil).meta(0, "vccCompact", "turn.context.compress"),
		ContextCompressPayload{
			Reason:            req.Reason,
			DroppedMessages:   dropped,
			RemainingMessages: 0,
		},
	)
	return nil
}

// Ingest appends a message to the session's in-memory store.
func (m *vccContextManager) Ingest(ctx context.Context, req *IngestRequest) error {
	if req == nil {
		return nil
	}
	s := m.session(req.SessionKey)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append(s.messages, req.Message)
	return nil
}

// Clear removes all stored context for the session.
func (m *vccContextManager) Clear(ctx context.Context, sessionKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, sessionKey)
	return nil
}

func init() {
	if err := RegisterContextManager("vcc", newVCCContextManager); err != nil {
		panic(fmt.Sprintf("register vcc context manager: %v", err))
	}
}
