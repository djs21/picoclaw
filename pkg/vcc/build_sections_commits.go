package vcc

import (
	"regexp"
	"strings"
)

// ── Commits ──

// commitInfo mirrors the TS CommitInfo (hash + message).
type commitInfo struct {
	hash    string
	message string
}

var commitMsgRe = regexp.MustCompile(`git\s+commit[^\n]*?-m\s+(?:"((?:[^"\\]|\\.)*)"|'((?:[^'\\]|\\.)*)'|\$?'((?:[^'\\]|\\.)*)')`)
var hashRe = regexp.MustCompile(`\b([0-9a-f]{7,12})\b`)

func firstLineOf(text string) string {
	idx := strings.IndexAny(text, "\n\r")
	if idx != -1 {
		return strings.TrimSpace(text[:idx])
	}
	return strings.TrimSpace(text)
}

func cleanMessage(msg string) string {
	msg = strings.ReplaceAll(msg, `\"`, `"`)
	msg = strings.ReplaceAll(msg, `\'`, `'`)
	return strings.TrimSpace(msg)
}

// extractCommits extracts git commits from bash tool calls and returns
// formatted strings ("hash: message" or just "message") (algo spec §3.3).
func extractCommits(blocks []NormalizedBlock) []string {
	var commits []commitInfo
	seen := make(map[string]bool)

	bracketRe := regexp.MustCompile(`\[\S+\s+([0-9a-f]{7,12})\]`)
	rangeRe := regexp.MustCompile(`\b([0-9a-f]{7,12})\.\.([0-9a-f]{7,12})\b`)

	for i := 0; i < len(blocks); i++ {
		b := blocks[i]
		if b.Kind != BlockKindToolCall || b.Name != "bash" {
			continue
		}
		cmd := b.Command
		if cmd == "" {
			if v, ok := b.Args["command"]; ok {
				if s, ok := v.(string); ok {
					cmd = s
				}
			}
		}
		if !strings.Contains(cmd, "git commit") {
			continue
		}
		m := commitMsgRe.FindStringSubmatch(cmd)
		if m == nil {
			continue
		}
		message := ""
		for _, group := range m[1:] {
			if group != "" {
				message = cleanMessage(firstLineOf(group))
				break
			}
		}
		if message == "" {
			continue
		}

		hash := ""
		// Look at next up to 2 tool_result blocks for hash
		for j := i + 1; j < len(blocks) && j < i+3; j++ {
			r := blocks[j]
			if r.Kind != BlockKindToolResult {
				continue
			}
			if bracket := bracketRe.FindStringSubmatch(r.Text); bracket != nil {
				hash = bracket[1]
				break
			}
			if rangeMatch := rangeRe.FindStringSubmatch(r.Text); rangeMatch != nil {
				hash = rangeMatch[2]
				break
			}
			if plain := hashRe.FindStringSubmatch(r.Text); plain != nil {
				hash = plain[1]
				break
			}
		}

		key := hash + "::" + message
		if !seen[key] {
			seen[key] = true
			commits = append(commits, commitInfo{hash: hash, message: message})
		}
	}

	// Format as strings
	result := make([]string, len(commits))
	for i, c := range commits {
		if c.hash != "" {
			result[i] = c.hash + ": " + c.message
		} else {
			result[i] = c.message
		}
	}
	return result
}

// formatCommits caps the commit list at 8, keeping the most recent (algo spec §3.3).
func formatCommits(commits []string) []string {
	const limit = 8
	if len(commits) <= limit {
		return commits
	}
	return commits[len(commits)-limit:]
}
