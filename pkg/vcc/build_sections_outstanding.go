package vcc

import (
	"regexp"
	"strings"
)

// BLOCKER_RE matches lines that signal outstanding work or problems
// (algo spec §3.3, src/core/build-sections.ts:17).
var BLOCKER_RE = regexp.MustCompile(`(?i)\b(fail(ed|s|ure|ing)?|broken|cannot|can't|won't work|does not work|doesn't work|still (broken|failing|wrong)|blocked|blocker|not (fixed|resolved|working)|crash(es|ed|ing)?)\b`)

// extractOutstandingContext scans the tail-20 blocks for blocker-signal lines
// and returns up to 5 clipped items (algo spec §3.3).
func extractOutstandingContext(blocks []NormalizedBlock) []string {
	var items []string
	tail := blocks
	if len(blocks) > 20 {
		tail = blocks[len(blocks)-20:]
	}
	seen := make(map[string]bool)
	for _, b := range tail {
		if b.Kind != BlockKindAssistant && b.Kind != BlockKindUser {
			continue
		}
		for _, line := range nonEmptyLines(b.Text) {
			if !BLOCKER_RE.MatchString(line) {
				continue
			}
			if len(line) < 15 {
				continue
			}
			// Skip continuation fragments (sub-bullets, parentheticals, dangling clauses)
			if len(line) > 0 && (line[0] == '-' || line[0] == '*' || line[0] == '+' || line[0] == '>') {
				continue
			}
			if len(line) > 0 && line[0] == '(' {
				continue
			}
			// Require sentence-like start: capital letter, code identifier, or quote
			trimmed := strings.TrimSpace(line)
			if len(trimmed) == 0 {
				continue
			}
			first := trimmed[0]
			if !(first >= 'A' && first <= 'Z') && first != '`' && first != '"' && first != '\'' && first != '*' && first != '_' {
				continue
			}
			var clipped string
			if b.Kind == BlockKindUser {
				clipped = "[user] " + clipSentence(line, 150)
			} else {
				clipped = clipSentence(line, 150)
			}
			if !seen[clipped] {
				seen[clipped] = true
				items = append(items, clipped)
			}
			break
		}
	}
	if len(items) > 5 {
		items = items[:5]
	}
	return items
}
