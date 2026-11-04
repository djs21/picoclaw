// PicoClaw - Ultra-lightweight personal AI agent

package agent

import (
	"regexp"
	"strings"

	"github.com/sipeed/picoclaw/pkg/providers"
)

// Leak reasons reported by detectLeakedToolCall. They exist so callers (and
// logs) can distinguish a forged tool envelope from a raw chain-of-thought
// spill; both break the turn because neither produces a structured tool call.
const (
	leakReasonPseudoToolCall     = "pseudo_tool_call"
	leakReasonReasoningMonologue = "reasoning_monologue"
)

// maxToolLeakRecoveryAttempts bounds how many times a single turn may be
// re-prompted after the model narrates a tool call instead of issuing one. One
// retry covers reasoning models that need a nudge without letting a model that
// is stuck in a leak loop spin until MaxIterations.
const maxToolLeakRecoveryAttempts = 1

// toolLeakRecoveryPrompt is injected as a user-role steering message when the
// previous assistant reply leaked tool syntax as plain text.
const toolLeakRecoveryPrompt = "Your previous reply described a tool call as plain text instead of invoking it, " +
	"so no action was executed. Do not print tool syntax such as `call:default_api:name(...)` in your reply. " +
	"Re-issue the action now through the structured tool call interface, or answer directly without tool syntax."

// pseudoToolCallPattern matches a forged tool envelope such as
// `call:default_api:read_file{"path": "main.go"}`. It requires a namespace and a
// tool name, so a bare namespace mention ("call:default_api") does not match.
var pseudoToolCallPattern = regexp.MustCompile(
	`(?i)\bcall\s*:\s*[a-z0-9_]+\s*[.:]\s*[a-z0-9_]+`,
)

// reasoningMonologuePattern matches chain-of-thought headers reasoning models
// leak into visible output on some proxies.
var reasoningMonologuePattern = regexp.MustCompile(
	`(?i)(stream of consciousness|filtered through my expert understanding)`,
)

// codeFencePattern matches whole fenced code blocks, including the fence lines.
var codeFencePattern = regexp.MustCompile("(?s)```.*?```")

// detectLeakedToolCall reports whether content contains tool call syntax or
// reasoning spill that should never reach the user as a final answer.
func detectLeakedToolCall(content string) (bool, string) {
	if strings.TrimSpace(content) == "" {
		return false, ""
	}

	// Documentation legitimately quotes the syntax inside fenced code blocks.
	scannable := codeFencePattern.ReplaceAllString(content, "")

	if pseudoToolCallPattern.MatchString(scannable) {
		return true, leakReasonPseudoToolCall
	}
	if reasoningMonologuePattern.MatchString(scannable) {
		return true, leakReasonReasoningMonologue
	}
	return false, ""
}

// toolLeakRecoveryMessage builds the steering message used to recover a turn.
func toolLeakRecoveryMessage() providers.Message {
	return providers.Message{Role: "user", Content: toolLeakRecoveryPrompt}
}
