package vcc

import (
	"regexp"
	"strings"
)

// NOISE_TOOLS lists tool names whose calls/results are always dropped (algo spec §3.2).
var NOISE_TOOLS = []string{
	"TodoWrite",
	"TodoRead",
	"ToolSearch",
	"WebSearch",
	"AskUser",
	"ExitSpecMode",
	"GenerateDroid",
}

// NOISE_STRINGS lists user text strings that mark a block as noise (algo spec §3.2).
var NOISE_STRINGS = []string{
	"Continue from where you left off.",
	"No response requested.",
	"IMPORTANT: TodoWrite was not called yet.",
}

// xmlWrapperTags lists the system XML wrapper tag names (algo spec §3.2).
var xmlWrapperTags = []string{
	"system-reminder",
	"ide_opened_file",
	"command-message",
	"context-window-usage",
}

// xmlWrapperREs compiles one regex per XML wrapper tag type (Go regexp lacks
// backreferences, so we can't use a single pattern with \1).
var xmlWrapperREs = func() []*regexp.Regexp {
	var res []*regexp.Regexp
	for _, tag := range xmlWrapperTags {
		res = append(res, regexp.MustCompile(`<`+tag+`[^>]*>[\s\S]*?</`+tag+`>`))
	}
	return res
}()

// isNoiseTool reports whether a tool name is in NOISE_TOOLS.
func isNoiseTool(name string) bool {
	for _, nt := range NOISE_TOOLS {
		if name == nt {
			return true
		}
	}
	return false
}

// isNoiseUserString reports whether trimmed text contains any NOISE_STRINGS.
func isNoiseUserString(text string) bool {
	trimmed := strings.TrimSpace(text)
	for _, ns := range NOISE_STRINGS {
		if strings.Contains(trimmed, ns) {
			return true
		}
	}
	return false
}

// cleanUserText removes XML wrappers and trims the text (algo spec §3.2).
func cleanUserText(text string) string {
	cleaned := text
	for _, re := range xmlWrapperREs {
		cleaned = re.ReplaceAllString(cleaned, "")
	}
	return strings.TrimSpace(cleaned)
}

// filterNoise drops noise tool calls/results, noise user strings, XML wrapper
// tags, and empty-after-clean user blocks (algo spec §3.2,
// src/core/filter-noise.ts:26).
func filterNoise(blocks []NormalizedBlock) []NormalizedBlock {
	var result []NormalizedBlock
	for _, b := range blocks {
		switch b.Kind {
		case BlockKindToolCall, BlockKindToolResult:
			if isNoiseTool(b.Name) {
				continue
			}
			result = append(result, b)
		case BlockKindUser:
			if isNoiseUserString(b.Text) {
				continue
			}
			cleaned := cleanUserText(b.Text)
			if cleaned == "" {
				continue
			}
			b.Text = cleaned
			result = append(result, b)
		default:
			result = append(result, b)
		}
	}
	return result
}
