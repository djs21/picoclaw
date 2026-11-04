package agent

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/tools"
)

const (
	// leakedToolCallReply reproduces the observed 9router/Gemini leak: a
	// chain-of-thought header plus a pseudo tool invocation printed as text.
	leakedToolCallReply = "Analyzing the Git Command: A Stream of Consciousness\n" +
		"Alright, let's break this down. First, I will run:\n" +
		"call:default_api:echo_text{\"text\":\"hi\"}"

	// leakRecoveryMarker is the fragment the shim must inject so the model
	// switches back to native tool calling instead of narrating the call.
	leakRecoveryMarker = "structured tool call interface"
)

// leakRecoveryProvider scripts one response per Chat call so tests can model
// "model leaks first, then behaves" without depending on iteration ordering
// elsewhere in the pipeline.
type leakRecoveryProvider struct {
	mu        sync.Mutex
	calls     int
	snapshots [][]providers.Message
	respond   func(call int, messages []providers.Message) *providers.LLMResponse
}

func (p *leakRecoveryProvider) Chat(
	_ context.Context,
	messages []providers.Message,
	_ []providers.ToolDefinition,
	_ string,
	_ map[string]any,
) (*providers.LLMResponse, error) {
	p.mu.Lock()
	p.calls++
	call := p.calls
	p.snapshots = append(p.snapshots, append([]providers.Message(nil), messages...))
	respond := p.respond
	p.mu.Unlock()

	if respond == nil {
		return &providers.LLMResponse{Content: "no scripted response", FinishReason: "stop"}, nil
	}
	return respond(call, messages), nil
}

func (p *leakRecoveryProvider) GetDefaultModel() string {
	return "leak-recovery-provider"
}

func (p *leakRecoveryProvider) callCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.calls
}

func (p *leakRecoveryProvider) snapshot(call int) []providers.Message {
	p.mu.Lock()
	defer p.mu.Unlock()
	if call < 1 || call > len(p.snapshots) {
		return nil
	}
	return p.snapshots[call-1]
}

// leakRecoveryRecordingTool records invocations so tests can prove the recovered
// tool call was actually executed, not merely mentioned.
type leakRecoveryRecordingTool struct {
	mu      sync.Mutex
	inputs  []string
	results []*tools.ToolResult
}

func (t *leakRecoveryRecordingTool) Name() string {
	return "echo_text"
}

func (t *leakRecoveryRecordingTool) Description() string {
	return "echo a text argument"
}

func (t *leakRecoveryRecordingTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"text": map[string]any{"type": "string"},
		},
	}
}

func (t *leakRecoveryRecordingTool) Execute(_ context.Context, args map[string]any) *tools.ToolResult {
	text, _ := args["text"].(string)
	result := tools.SilentResult(text)

	t.mu.Lock()
	t.inputs = append(t.inputs, text)
	t.results = append(t.results, result)
	t.mu.Unlock()

	return result
}

func (t *leakRecoveryRecordingTool) callCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.inputs)
}

func newLeakRecoveryAgentLoop(
	t *testing.T,
	provider *leakRecoveryProvider,
	tool *leakRecoveryRecordingTool,
) *AgentLoop {
	t.Helper()

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         t.TempDir(),
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}
	al := NewAgentLoop(cfg, bus.NewMessageBus(), provider)
	al.RegisterTool(tool)
	return al
}

// TestPipeline_RepromptsWhenModelLeaksToolCallAsText is the core regression for
// the freeze: the first LLM reply contains a pseudo tool call instead of a
// structured one. The pipeline must re-prompt once, then execute the tool call
// the model issues on the retry, and finally deliver the model's real answer.
func TestPipeline_RepromptsWhenModelLeaksToolCallAsText(t *testing.T) {
	provider := &leakRecoveryProvider{}
	provider.respond = func(call int, _ []providers.Message) *providers.LLMResponse {
		switch call {
		case 1:
			return &providers.LLMResponse{Content: leakedToolCallReply, FinishReason: "stop"}
		case 2:
			return &providers.LLMResponse{
				Content: "calling the tool properly",
				ToolCalls: []providers.ToolCall{{
					ID:        "call_1",
					Name:      "echo_text",
					Arguments: map[string]any{"text": "hi"},
				}},
				FinishReason: "tool_calls",
			}
		default:
			return &providers.LLMResponse{Content: "final answer", FinishReason: "stop"}
		}
	}
	tool := &leakRecoveryRecordingTool{}
	al := newLeakRecoveryAgentLoop(t, provider, tool)

	response, err := al.runAgentLoop(
		context.Background(),
		al.GetRegistry().GetDefaultAgent(),
		processOptions{
			SessionKey:      "agent:default:test-leak-recovery",
			UserMessage:     "please check git status",
			DefaultResponse: defaultResponse,
		},
	)
	if err != nil {
		t.Fatalf("runAgentLoop() error = %v", err)
	}
	if response != "final answer" {
		t.Fatalf("response = %q, want %q", response, "final answer")
	}
	if got := provider.callCount(); got != 3 {
		t.Fatalf("provider calls = %d, want 3 (leak, recovered tool call, final answer)", got)
	}
	if got := tool.callCount(); got != 1 {
		t.Fatalf("tool invocations = %d, want 1", got)
	}

	secondCall := provider.snapshot(2)
	if len(secondCall) == 0 {
		t.Fatalf("second provider call captured no messages")
	}
	var recovery *providers.Message
	for i := range secondCall {
		msg := secondCall[i]
		if msg.Role == "user" && strings.Contains(msg.Content, leakRecoveryMarker) {
			recovery = &secondCall[i]
			break
		}
	}
	if recovery == nil {
		t.Fatalf("second provider call missing leak-recovery user message: %#v", secondCall)
	}
	if !strings.Contains(recovery.Content, "previous reply") {
		t.Fatalf("recovery message must reference the previous reply: %q", recovery.Content)
	}

	thirdCall := provider.snapshot(3)
	var toolResultSeen bool
	for _, msg := range thirdCall {
		if msg.Role == "tool" && strings.Contains(msg.Content, "hi") {
			toolResultSeen = true
			break
		}
	}
	if !toolResultSeen {
		t.Fatalf("third provider call missing tool result: %#v", thirdCall)
	}
}

// TestPipeline_LeakRecoveryIsBoundedToSingleReprompt ensures a model that keeps
// leaking cannot spin the turn: exactly one recovery attempt is made, then the
// content is delivered as-is instead of hanging until MaxIterations.
func TestPipeline_LeakRecoveryIsBoundedToSingleReprompt(t *testing.T) {
	provider := &leakRecoveryProvider{}
	provider.respond = func(int, []providers.Message) *providers.LLMResponse {
		return &providers.LLMResponse{Content: leakedToolCallReply, FinishReason: "stop"}
	}
	tool := &leakRecoveryRecordingTool{}
	al := newLeakRecoveryAgentLoop(t, provider, tool)

	response, err := al.runAgentLoop(
		context.Background(),
		al.GetRegistry().GetDefaultAgent(),
		processOptions{
			SessionKey:      "agent:default:test-leak-recovery-bounded",
			UserMessage:     "please check git status",
			DefaultResponse: defaultResponse,
		},
	)
	if err != nil {
		t.Fatalf("runAgentLoop() error = %v", err)
	}
	if got := provider.callCount(); got != 2 {
		t.Fatalf("provider calls = %d, want 2 (initial + single recovery attempt)", got)
	}
	if got := tool.callCount(); got != 0 {
		t.Fatalf("tool invocations = %d, want 0", got)
	}
	if response != leakedToolCallReply {
		t.Fatalf("response = %q, want the leaked content delivered as-is", response)
	}
}

// TestPipeline_DoesNotRepromptWhenToolSyntaxAppearsInCodeFence keeps the shim
// from firing on documentation: explaining the syntax inside a fenced block is
// a legitimate answer and must not trigger a recovery round trip.
func TestPipeline_DoesNotRepromptWhenToolSyntaxAppearsInCodeFence(t *testing.T) {
	documented := "A leaked call looks like this:\n" +
		"```text\n" +
		"call:default_api:echo_text{\"text\":\"hi\"}\n" +
		"```\n" +
		"PicoClaw rejects it and asks the model to call the tool properly."

	provider := &leakRecoveryProvider{}
	provider.respond = func(int, []providers.Message) *providers.LLMResponse {
		return &providers.LLMResponse{Content: documented, FinishReason: "stop"}
	}
	tool := &leakRecoveryRecordingTool{}
	al := newLeakRecoveryAgentLoop(t, provider, tool)

	response, err := al.runAgentLoop(
		context.Background(),
		al.GetRegistry().GetDefaultAgent(),
		processOptions{
			SessionKey:      "agent:default:test-leak-recovery-fenced",
			UserMessage:     "how does the leak look?",
			DefaultResponse: defaultResponse,
		},
	)
	if err != nil {
		t.Fatalf("runAgentLoop() error = %v", err)
	}
	if got := provider.callCount(); got != 1 {
		t.Fatalf("provider calls = %d, want 1 (no recovery for fenced documentation)", got)
	}
	if response != documented {
		t.Fatalf("response = %q, want the documented answer unchanged", response)
	}
}
