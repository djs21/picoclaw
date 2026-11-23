package agent

import (
	"context"
	"encoding/json"
	"fmt"

	runtimeevents "github.com/sipeed/picoclaw/pkg/events"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/vcc"
)

// vccContextManager adapts the deterministic VCC compiler to agent.ContextManager.
// It persists through SessionStore and never invokes an LLM.
type vccContextManager struct {
	al       *AgentLoop
	settings vcc.Settings
}

// newVCCContextManager creates a VCC-backed ContextManager.
func newVCCContextManager(cfg json.RawMessage, al *AgentLoop) (ContextManager, error) {
	if al == nil {
		return nil, fmt.Errorf("vcc: AgentLoop is required")
	}
	settings := vcc.DEFAULT_SETTINGS
	if len(cfg) > 0 {
		parsed, err := vcc.ParseSettings(cfg)
		if err != nil {
			return nil, fmt.Errorf("vcc: parse settings: %w", err)
		}
		settings = parsed
	}
	return &vccContextManager{
		al:       al,
		settings: settings,
	}, nil
}

func (m *vccContextManager) sessionsFor(key string) (*AgentInstance, error) {
	agent := m.al.agentForSession(key)
	if agent == nil || agent.Sessions == nil {
		return nil, fmt.Errorf("vcc: sessions not initialized")
	}
	return agent, nil
}

// Assemble returns the session history and summary from SessionStore. VCC never
// invokes an LLM, so this is just a pass-through of the persistent state.
func (m *vccContextManager) Assemble(_ context.Context, req *AssembleRequest) (*AssembleResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("vcc assemble: nil request")
	}
	agent, err := m.sessionsFor(req.SessionKey)
	if err != nil {
		return nil, err
	}
	return &AssembleResponse{
		History: agent.Sessions.GetHistory(req.SessionKey),
		Summary: agent.Sessions.GetSummary(req.SessionKey),
	}, nil
}

// Compact compiles the persisted session history into a deterministic VCC
// summary, persists it, clears the raw history, and emits both compress and
// session-summarize events.
func (m *vccContextManager) Compact(_ context.Context, req *CompactRequest) error {
	if req == nil {
		return nil
	}
	agent, err := m.sessionsFor(req.SessionKey)
	if err != nil {
		return err
	}

	history := agent.Sessions.GetHistory(req.SessionKey)
	if len(history) == 0 {
		return nil
	}

	summary, err := compileVCC(history, agent.Sessions.GetSummary(req.SessionKey))
	if err != nil {
		return err
	}
	if summary == "" {
		return nil
	}

	if req.Reason == ContextCompressReasonRetry && req.Budget > 0 {
		summary = fitSummaryToBudget(summary, history, agent.Sessions.GetSummary(req.SessionKey), req.Budget)
	}

	agent.Sessions.SetSummary(req.SessionKey, summary)
	agent.Sessions.SetHistory(req.SessionKey, []providers.Message{})
	if err := agent.Sessions.Save(req.SessionKey); err != nil {
		return fmt.Errorf("vcc compact: save session: %w", err)
	}
	dropped := len(history)

	m.al.emitEvent(
		runtimeevents.KindAgentContextCompress,
		m.al.newTurnEventScope(agent.ID, req.SessionKey, nil).meta(0, "vccCompact", "turn.context.compress"),
		ContextCompressPayload{
			Reason:            req.Reason,
			DroppedMessages:   dropped,
			RemainingMessages: 0,
		},
	)
	m.al.emitEvent(
		runtimeevents.KindAgentSessionSummarize,
		m.al.newTurnEventScope(agent.ID, req.SessionKey, nil).meta(0, "vccCompact", "turn.session.summarize"),
		SessionSummarizePayload{
			SummarizedMessages: dropped,
			KeptMessages:       0,
			SummaryLen:         len(summary),
			OmittedOversized:   false,
		},
	)
	return nil
}

// Ingest is a no-op because the pipeline persists each message to SessionStore
// before calling Ingest (same contract as the legacy ContextManager).
func (m *vccContextManager) Ingest(_ context.Context, _ *IngestRequest) error {
	return nil
}

// Clear removes the session history and summary from persistent storage.
func (m *vccContextManager) Clear(_ context.Context, sessionKey string) error {
	agent, err := m.sessionsFor(sessionKey)
	if err != nil {
		return err
	}
	agent.Sessions.SetHistory(sessionKey, []providers.Message{})
	agent.Sessions.SetSummary(sessionKey, "")
	return agent.Sessions.Save(sessionKey)
}

func init() {
	if err := RegisterContextManager("vcc", newVCCContextManager); err != nil {
		panic(fmt.Sprintf("register vcc context manager: %v", err))
	}
}

func providerMessagesToMaps(msgs []providers.Message) ([]map[string]any, error) {
	raw, err := json.Marshal(msgs)
	if err != nil {
		return nil, fmt.Errorf("vcc compact: marshal messages: %w", err)
	}
	var out []map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("vcc compact: unmarshal messages: %w", err)
	}
	return out, nil
}

func compileVCC(history []providers.Message, previous string) (string, error) {
	msgs, err := providerMessagesToMaps(history)
	if err != nil {
		return "", err
	}
	return vcc.Compile(vcc.CompileInput{
		Messages:        msgs,
		PreviousSummary: previous,
	}), nil
}

func fitSummaryToBudget(summary string, history []providers.Message, previous string, budget int) string {
	estimate := func(s string) int {
		return vcc.EstimateTokensFromChars(len([]rune(s)))
	}

	workingHistory := append([]providers.Message(nil), history...)
	workingSummary := summary
	for len(workingHistory) > 1 && estimate(workingSummary) > budget {
		workingHistory = workingHistory[1:]
		next, err := compileVCC(workingHistory, previous)
		if err != nil || next == "" {
			break
		}
		workingSummary = next
		// Once the message set changed, the previous summary no longer needs to
		// merge with the already-summarized portion we just removed.
		if len(workingHistory) > 1 {
			previous = ""
		}
	}

	if estimate(workingSummary) <= budget {
		return workingSummary
	}
	// Hard deterministic fallback: cut the summary to fit the budget.
	maxRunes := budget*4 + 4
	runes := []rune(workingSummary)
	if len(runes) <= maxRunes {
		return workingSummary
	}
	return string(runes[:maxRunes])
}
