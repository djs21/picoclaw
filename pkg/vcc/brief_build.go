package vcc

import (
	"regexp"
	"strings"
)

// ── tool summary ──

var toolSummaryFields = map[string]string{
	"Read":  "file_path",
	"Edit":  "file_path",
	"Write": "file_path",
	"read":  "file_path",
	"edit":  "file_path",
	"write": "file_path",
	"Glob":  "pattern",
	"Grep":  "pattern",
}

// extractPath returns the first path-like argument value, or "" if none.
func extractPath(args map[string]any) string {
	for _, key := range []string{"path", "file_path", "filePath", "file"} {
		if v, ok := args[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return ""
}

// toolOneLiner renders a single tool call as a compact summary line.
func toolOneLiner(name string, args map[string]any) string {
	field, ok := toolSummaryFields[name]
	if ok {
		if v, ok := args[field]; ok {
			if s, ok := v.(string); ok {
				return `* ` + name + ` "` + s + `"`
			}
		}
	}
	if path := extractPath(args); path != "" {
		return `* ` + name + ` "` + path + `"`
	}
	if name == "bash" || name == "Bash" {
		raw := ""
		if v, ok := args["command"]; ok {
			if s, ok := v.(string); ok {
				raw = s
			}
		}
		if raw == "" {
			if v, ok := args["description"]; ok {
				if s, ok := v.(string); ok {
					raw = s
				}
			}
		}
		cmd := compressBash(raw)
		return `* ` + name + ` "` + cmd + `"`
	}
	if v, ok := args["query"]; ok {
		if s, ok := v.(string); ok {
			return `* ` + name + ` "` + clip(s, 60) + `"`
		}
	}
	return `* ` + name
}

// ── buildBriefSections ──

func pushSection(sections []BriefLine, lastHeader string, header string, line string) ([]BriefLine, string) {
	if header == lastHeader && len(sections) > 0 {
		sections[len(sections)-1].Lines = append(sections[len(sections)-1].Lines, line)
		return sections, lastHeader
	}
	sections = append(sections, BriefLine{Header: header, Lines: []string{line}})
	return sections, header
}

func pushText(sections []BriefLine, lastHeader string, header string, text string, ref string) ([]BriefLine, string) {
	lines := strings.Split(text, "\n")
	if ref != "" && len(lines) > 0 {
		lines[len(lines)-1] = lines[len(lines)-1] + ref
	}
	for _, line := range lines {
		sections, lastHeader = pushSection(sections, lastHeader, header, line)
	}
	return sections, lastHeader
}

// buildBriefSections builds BriefLine sections from NormalizedBlocks.
func buildBriefSections(blocks []NormalizedBlock) []BriefLine {
	var sections []BriefLine
	lastHeader := ""

	for blockIndex := 0; blockIndex < len(blocks); blockIndex++ {
		b := blocks[blockIndex]
		switch b.Kind {
		case BlockKindUser:
			if isNoiseUser(b.Text) {
				break
			}
			text := truncateTokens(collapseSkillText(b.Text), 256)
			if text != "" {
				ref := ""
				if b.SourceIndex != nil {
					ref = " (#" + itoa(*b.SourceIndex) + ")"
				}
				sections, lastHeader = pushText(sections, lastHeader, "[user]", text, ref)
			}
			lastHeader = "[user]"

		case BlockKindBash:
			cmd := compressBash(b.Command)
			ref := ""
			if b.SourceIndex != nil {
				ref = " (#" + itoa(*b.SourceIndex) + ")"
			}
			if cmd != "" {
				sections, lastHeader = pushSection(sections, lastHeader, "[user]", "$ "+cmd+ref)
			}
			lastHeader = "[user]"

		case BlockKindAssistant:
			raw := b.Text
			raw = stripSelfTalk(raw)
			var text string
			if isSegmentClosingAssistant(blocks, blockIndex) {
				text = truncateTokensHeadTail(raw, 120, 120)
			} else {
				text = truncateTokensHeadTail(raw, 80, 120)
			}
			if text != "" {
				ref := ""
				if b.SourceIndex != nil {
					ref = " (#" + itoa(*b.SourceIndex) + ")"
				}
				sections, lastHeader = pushText(sections, lastHeader, "[assistant]", text, ref)
			}

		case BlockKindToolCall:
			if b.Name == "" || strings.TrimSpace(b.Name) == "" {
				break
			}
			ref := ""
			if b.SourceIndex != nil {
				ref = " (#" + itoa(*b.SourceIndex) + ")"
			}
			summary := toolOneLiner(b.Name, b.Args) + ref
			sections, lastHeader = pushSection(sections, lastHeader, "[assistant]", summary)

		case BlockKindToolResult:
			// Tool result bodies are intentionally omitted from compact briefs.
		}
	}

	// Collapse consecutive identical tool lines (same text, different #ref).
	for secIdx := range sections {
		sec := &sections[secIdx]
		if sec.Header != "[assistant]" {
			continue
		}
		var out []string
		for _, line := range sec.Lines {
			if !strings.HasPrefix(line, "* ") {
				out = append(out, line)
				continue
			}
			ref := extractRef(line)
			if ref == "" {
				out = append(out, line)
				continue
			}
			base := strings.TrimRight(line[:len(line)-(len(ref)+3)], " ")
			last := ""
			if len(out) > 0 {
				last = out[len(out)-1]
			}
			if m := collapseGroupRe.FindStringSubmatch(last); m != nil && m[1] == base {
				out[len(out)-1] = base + " (" + m[2] + ", #" + ref + ") x" + itoa(atoi(m[3])+1)
			} else if singleRefRe.MatchString(last) {
				prevRef := extractRef(last)
				prevBase := strings.TrimRight(last[:len(last)-(len(prevRef)+3)], " ")
				if prevBase == base {
					out[len(out)-1] = base + " (#" + prevRef + ", #" + ref + ") x2"
				} else {
					out = append(out, line)
				}
			} else {
				out = append(out, line)
			}
		}
		sec.Lines = out
	}

	// Cap tool calls per [assistant] turn — keep tail.
	const toolCallsPerTurn = 8
	for secIdx := range sections {
		sec := &sections[secIdx]
		if sec.Header != "[assistant]" {
			continue
		}
		toolIdxs := make([]int, 0)
		for i, l := range sec.Lines {
			if strings.HasPrefix(l, "* ") {
				toolIdxs = append(toolIdxs, i)
			}
		}
		if len(toolIdxs) <= toolCallsPerTurn {
			continue
		}
		dropCount := len(toolIdxs) - toolCallsPerTurn
		dropSet := make(map[int]bool)
		for _, idx := range toolIdxs[:dropCount] {
			dropSet[idx] = true
		}
		firstKeptToolIdx := toolIdxs[dropCount]
		var next []string
		inserted := false
		for i := 0; i < len(sec.Lines); i++ {
			if dropSet[i] {
				continue
			}
			if !inserted && i == firstKeptToolIdx {
				next = append(next, "* ("+itoa(dropCount)+" earlier tool-call entries omitted)")
				inserted = true
			}
			next = append(next, sec.Lines[i])
		}
		sec.Lines = next
	}

	return sections
}

var collapseGroupRe = regexp.MustCompile(`^(.*) \((#[\d, #]+)\) x(\d+)$`)
var singleRefRe = regexp.MustCompile(`\(#\d+\)$`)

func extractRef(line string) string {
	m := singleRefRe.FindStringSubmatch(line)
	if len(m) < 1 {
		return ""
	}
	inner := m[0]
	inner = strings.TrimPrefix(inner, "(#")
	inner = strings.TrimSuffix(inner, ")")
	return inner
}

// ── stringifyBrief ──

// stringifyBrief joins sections into text, suppressing blank lines between
// consecutive all-tool assistant sections.
func stringifyBrief(sections []BriefLine) string {
	var out []string
	for i, sec := range sections {
		if i > 0 {
			prev := sections[i-1]
			prevIsTools := prev.Header == "[assistant]" && allToolLines(prev.Lines)
			curIsTools := sec.Header == "[assistant]" && allToolLines(sec.Lines)
			if !(prevIsTools && curIsTools) {
				out = append(out, "")
			}
		}
		out = append(out, sec.Header)
		for _, line := range sec.Lines {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

func allToolLines(lines []string) bool {
	if len(lines) == 0 {
		return false
	}
	for _, l := range lines {
		if !strings.HasPrefix(l, "* ") {
			return false
		}
	}
	return true
}

// ── compileBrief ──

// compileBrief builds sections from blocks and stringifies to text.
func compileBrief(blocks []NormalizedBlock) string {
	return stringifyBrief(buildBriefSections(blocks))
}

// itoa converts a non-negative integer to its decimal string.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// atoi parses a decimal string to int (assumes valid input from refs).
func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}
