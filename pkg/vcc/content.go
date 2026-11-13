package vcc

import (
	"regexp"
	"strings"
)

// textOf extracts plain text from message content, which may be a string or an
// array of parts (algo spec §3.1, src/core/content.ts:49).
func textOf(content interface{}) string {
	switch c := content.(type) {
	case string:
		return c
	case []interface{}:
		var parts []string
		for _, p := range c {
			if m, ok := p.(map[string]interface{}); ok {
				if t, ok := m["text"].(string); ok {
					parts = append(parts, t)
				}
			}
		}
		return strings.Join(parts, "\n")
	default:
		return ""
	}
}

// sanitize normalizes CRLF/CR → LF, strips ANSI escape sequences and control
// characters (algo spec §3.1, src/core/sanitize.ts:4).
func sanitize(s string) string {
	// CRLF → LF
	s = strings.ReplaceAll(s, "\r\n", "\n")
	// lone CR → LF
	s = strings.ReplaceAll(s, "\r", "\n")
	// ANSI escape sequences: ESC [ ... letter
	ansiRe := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	s = ansiRe.ReplaceAllString(s, "")
	// strip control characters except \n and \t
	var b strings.Builder
	for _, r := range s {
		if r == '\n' || r == '\t' || r >= 0x20 {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// clip truncates s to at most max runes (algo spec §3.1).
func clip(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max <= 0 {
		return ""
	}
	return string(runes[:max])
}

// clipSentence truncates s to at most max runes, preferring a sentence
// boundary (. ! ?) within the limit (algo spec §3.1).
func clipSentence(s string, max int) string {
	if len(s) <= max {
		return s
	}
	// Look for sentence-ending punctuation within the first max runes.
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	limit := max
	if limit > len(runes) {
		limit = len(runes)
	}
	for i := limit - 1; i >= 0; i-- {
		switch runes[i] {
		case '.', '!', '?':
			return string(runes[:i+1])
		}
	}
	// No sentence boundary found; hard clip.
	return string(runes[:max])
}

// nonEmptyLines returns the non-empty (trimmed) lines of s (algo spec §3.1).
func nonEmptyLines(s string) []string {
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			lines = append(lines, trimmed)
		}
	}
	return lines
}
