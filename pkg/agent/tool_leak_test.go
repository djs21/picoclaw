package agent

import (
	"strings"
	"testing"
)

// TestDetectLeakedToolCall_FlagsPseudoCallSyntax covers the failure mode where a
// reasoning model narrates a tool invocation as text instead of emitting a
// structured tool call. PicoClaw must be able to recognise these envelopes so
// the pipeline can recover instead of delivering the pseudo-call to the user.
func TestDetectLeakedToolCall_FlagsPseudoCallSyntax(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "json arguments",
			content: "Sure, doing it now.\ncall:default_api:echo_text{\"text\":\"hi\"}",
		},
		{
			name:    "parenthesized arguments",
			content: "call:default_api:execute_shell_command(command=\"git status\")",
		},
		{
			name:    "newline separated arguments",
			content: "call:default_api:read_file\n{\"path\": \"main.go\"}",
		},
		{
			name:    "trailing colon envelope",
			content: "call:default_api:web_search: {\"query\": \"picoclaw\"}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, reason := detectLeakedToolCall(tt.content)
			if !matched {
				t.Fatalf("detectLeakedToolCall(%q) matched=false, want true", tt.content)
			}
			if reason != leakReasonPseudoToolCall {
				t.Fatalf("reason = %q, want %q", reason, leakReasonPseudoToolCall)
			}
		})
	}
}

// TestDetectLeakedToolCall_FlagsReasoningMonologue covers the second half of the
// observed freeze: the model streams a raw chain-of-thought header into the
// reply, which stalls the turn because no tool call is ever issued.
func TestDetectLeakedToolCall_FlagsReasoningMonologue(t *testing.T) {
	content := "Analyzing the Git Command: A Stream of Consciousness\n" +
		"Alright, let's break this down. First, the command I intend to run is git status."

	matched, reason := detectLeakedToolCall(content)
	if !matched {
		t.Fatalf("detectLeakedToolCall() matched=false, want true")
	}
	if reason != leakReasonReasoningMonologue {
		t.Fatalf("reason = %q, want %q", reason, leakReasonReasoningMonologue)
	}
}

// TestDetectLeakedToolCall_IgnoresConversationalAnswers guards against false
// positives: normal replies, namespace mentions without a tool name, and
// documentation snippets inside fenced code blocks must pass through untouched.
func TestDetectLeakedToolCall_IgnoresConversationalAnswers(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "plain answer",
			content: "The weather in Jakarta is 31 degrees Celsius with light rain.",
		},
		{
			name:    "empty reply",
			content: "",
		},
		{
			name:    "namespace mention without tool name",
			content: "Tool calls use the call:default_api namespace under the hood.",
		},
		{
			name: "documentation inside fenced code block",
			content: "Here is the shape of a leaked call:\n" +
				"```text\n" +
				"call:default_api:echo_text{\"text\":\"hi\"}\n" +
				"```\n" +
				"PicoClaw never prints this to users.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, reason := detectLeakedToolCall(tt.content)
			if matched {
				t.Fatalf(
					"detectLeakedToolCall(%q) matched=true (reason=%q), want false",
					tt.content,
					reason,
				)
			}
		})
	}
}

// TestLeakRecoveryPromptMentionsStructuredInterface pins the wording contract
// used by the recovery shim: the re-prompt must tell the model to re-issue the
// action through the structured tool call interface and to stop printing tool
// syntax as plain text.
func TestLeakRecoveryPromptMentionsStructuredInterface(t *testing.T) {
	for _, want := range []string{
		"previous reply",
		"structured tool call interface",
		"plain text",
	} {
		if !strings.Contains(toolLeakRecoveryPrompt, want) {
			t.Fatalf("toolLeakRecoveryPrompt missing %q: %q", want, toolLeakRecoveryPrompt)
		}
	}
}
