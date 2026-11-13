package vcc

import (
	"regexp"
	"strings"
)

// BRIEF_MAX_LINES caps the rendered brief transcript (algo spec §3.4).
const BRIEF_MAX_LINES = 120

// BriefLine is a section of the brief transcript: a header like "[user]" or
// "[assistant]" plus its content lines.
type BriefLine struct {
	Header string
	Lines  []string
}

// ── noise filtering ──

// isNoiseUser reports whether user text carries no durable information.
func isNoiseUser(text string) bool {
	return strings.TrimSpace(text) == ""
}

// ── skill collapse ──

var skillBlockRe = regexp.MustCompile(`<skill\s+name="([^"]+)"[^>]*>[\s\S]*?(?:<\/skill>|$)`)

// collapseSkillText replaces <skill name="X">...</skill> blocks with [skill: X].
func collapseSkillText(text string) string {
	return skillBlockRe.ReplaceAllStringFunc(text, func(match string) string {
		name := skillBlockRe.FindStringSubmatch(match)[1]
		return "[skill: " + name + "]"
	})
}

// ── truncation ──

var stopWords = map[string]bool{
	"a": true, "an": true, "the": true, "is": true, "are": true, "was": true, "were": true, "be": true, "been": true, "being": true,
	"have": true, "has": true, "had": true, "do": true, "does": true, "did": true, "will": true, "would": true, "could": true,
	"should": true, "may": true, "might": true, "shall": true, "can": true, "need": true, "must": true,
	"to": true, "of": true, "in": true, "for": true, "on": true, "with": true, "at": true, "by": true, "from": true, "as": true,
	"into": true, "through": true, "during": true, "before": true, "after": true, "above": true, "below": true,
	"between": true, "under": true, "over": true,
	"and": true, "but": true, "or": true, "nor": true, "not": true, "so": true, "yet": true, "both": true, "either": true,
	"neither": true, "each": true, "every": true, "all": true, "any": true, "few": true, "more": true, "most": true,
	"other": true, "some": true, "such": true, "no": true,
	"that": true, "this": true, "these": true, "those": true, "it": true, "its": true,
	"i": true, "me": true, "my": true, "we": true, "our": true, "you": true, "your": true, "he": true, "him": true, "his": true,
	"she": true, "her": true, "they": true, "them": true, "their": true, "who": true, "which": true, "what": true,
	"if": true, "then": true, "than": true, "when": true, "where": true, "how": true, "just": true, "also": true,
}

var wordRe = regexp.MustCompile(`[\p{L}\p{N}]`)

// isWord reports whether a segment counts as a word (alphanumeric content).
func isWord(seg string) bool {
	return wordRe.MatchString(seg)
}

// normalizeForTokenBudget normalizes whitespace/newlines like the TS port.
func normalizeForTokenBudget(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	re := regexp.MustCompile(`[^\S\n]+`)
	text = re.ReplaceAllString(text, " ")
	re2 := regexp.MustCompile(`\n{3,}`)
	text = re2.ReplaceAllString(text, "\n\n")
	return strings.TrimSpace(text)
}

// truncateTokens keeps the first `limit` non-stop words, appending a truncation
// marker when the budget is exceeded.
func truncateTokens(text string, limit int) string {
	flat := normalizeForTokenBudget(text)
	words := strings.Split(flat, " ")
	count := 0
	lastEnd := 0
	pos := 0
	for _, w := range words {
		if isWord(w) && !stopWords[strings.ToLower(w)] {
			count++
			if count > limit {
				return flat[:lastEnd] + "...(truncated)"
			}
		}
		lastEnd = pos + len(w)
		pos += len(w) + 1 // account for the space we split on
	}
	return flat
}

type wordSpan struct {
	start int
	end   int
}

// significantWordSpans returns byte spans of non-stop words in flat text.
func significantWordSpans(flat string) []wordSpan {
	var words []wordSpan
	re := regexp.MustCompile(`\S+`)
	for _, m := range re.FindAllStringIndex(flat, -1) {
		seg := flat[m[0]:m[1]]
		if !isWord(seg) {
			continue
		}
		if stopWords[strings.ToLower(seg)] {
			continue
		}
		words = append(words, wordSpan{start: m[0], end: m[1]})
	}
	return words
}

// truncateTokensHeadTail keeps headLimit significant words from the start and
// tailLimit from the end, eliding the middle.
func truncateTokensHeadTail(text string, headLimit, tailLimit int) string {
	flat := normalizeForTokenBudget(text)
	if headLimit <= 0 || tailLimit <= 0 {
		return flat
	}
	words := significantWordSpans(flat)
	if len(words) <= headLimit+tailLimit {
		return flat
	}
	head := strings.TrimRight(flat[:words[headLimit-1].end], " ")
	tail := strings.TrimLeft(flat[words[len(words)-tailLimit].start:], " ")
	return head + "\n...(middle truncated)...\n" + tail
}

// ── self-talk prefix ──

var selfTalkPrefixRe = regexp.MustCompile(`(?i)^\s*(?:hmm|wait|actually|oh|okay|ok|well|so)[,.!\s-]+`)

// stripSelfTalk removes up to two leading self-reflective assistant prefixes.
func stripSelfTalk(raw string) string {
	for i := 0; i < 2; i++ {
		stripped := selfTalkPrefixRe.ReplaceAllString(raw, "")
		if stripped == raw {
			break
		}
		raw = stripped
	}
	return raw
}

// ── segment closing ──

// nextRenderableBlock returns the next block after index that is not a tool_result.
func nextRenderableBlock(blocks []NormalizedBlock, index int) *NormalizedBlock {
	for i := index + 1; i < len(blocks); i++ {
		if blocks[i].Kind != BlockKindToolResult {
			return &blocks[i]
		}
	}
	return nil
}

// isSegmentClosingAssistant reports whether an assistant block is followed by
// nothing renderable or by a user block (a segment boundary).
func isSegmentClosingAssistant(blocks []NormalizedBlock, index int) bool {
	if blocks[index].Kind != BlockKindAssistant {
		return false
	}
	next := nextRenderableBlock(blocks, index)
	return next == nil || next.Kind == BlockKindUser
}

// ── bash command compression ──

const bashCap = 240

var pipeTailRe = regexp.MustCompile(`\s*\|\s*(?:head|tail|sort|wc|column|tr|cut|awk|uniq|python3|node|bun)(?:\s[^|]*)?$`)

var trivialLineRe = regexp.MustCompile(`^(?:set\s+[-+]|cd\s+\S+$|export\s+\w+=|(?:source|\.)\s+\S+|pwd$|true$|:$|#)`)

// heredocOpenRe matches a heredoc opener. Callers must verify the preceding
// char is not a word char or `)` (to avoid mistaking shift ops for openers).
var heredocOpenRe = regexp.MustCompile(`<<-?\s*["']?([A-Za-z_]\w*)["']?`)

func isWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

var filewriterHeredocRe = regexp.MustCompile(`(?:^|[|&;]\s*)(?:cat|tee|dd)\b`)

const heredocBodyCap = 80

var bodyNoiseRe = regexp.MustCompile(`^(?:import\s|from\s+\S+\s+import|require\(|const\s+\w+\s*=\s*require|#|\/\/|""\"|'''|"use strict"|use\s+strict|<\?php)`)

// heredocCloseIndex returns the terminator line index for a heredoc opened at
// lines[i], or -1 if no matching terminator appears downstream.
func heredocCloseIndex(lines []string, i int) int {
	line := lines[i]
	loc := heredocOpenRe.FindStringSubmatchIndex(line)
	if loc == nil {
		return -1
	}
	// A real heredoc opener is not preceded by a word char or `)`.
	start := loc[0]
	if start > 0 {
		prev := line[start-1]
		if isWordChar(prev) || prev == ')' {
			return -1
		}
	}
	term := line[loc[2]:loc[3]]
	for j := i + 1; j < len(lines); j++ {
		if strings.TrimSpace(lines[j]) == term {
			return j
		}
	}
	return -1
}

func stripCdPrefix(line string) string {
	return strings.TrimSpace(regexp.MustCompile(`^cd\s+\S+\s*&&\s*`).ReplaceAllString(line, ""))
}

func stripPipeTail(line string) string {
	c := line
	for i := 0; i < 3; i++ {
		stripped := pipeTailRe.ReplaceAllString(c, "")
		if stripped == c {
			break
		}
		c = stripped
	}
	return strings.TrimSpace(c)
}

// compressBash semantically compresses a (possibly multi-line) bash command.
func compressBash(raw string) string {
	rawLines := strings.Split(raw, "\n")
	withoutHeredocBodies := make([]string, 0, len(rawLines))
	for i := 0; i < len(rawLines); i++ {
		line := rawLines[i]
		withoutHeredocBodies = append(withoutHeredocBodies, line)
		close := heredocCloseIndex(rawLines, i)
		if close == -1 {
			continue
		}
		wantPreview := !filewriterHeredocRe.MatchString(line)
		var preview string
		if wantPreview {
			for j := i + 1; j < close; j++ {
				t := strings.TrimSpace(rawLines[j])
				if t != "" && !bodyNoiseRe.MatchString(t) {
					preview = t
					break
				}
			}
		}
		if preview != "" {
			clipped := preview
			if len(clipped) > heredocBodyCap-1 {
				clipped = clipped[:heredocBodyCap-1] + "\u2026"
			}
			last := len(withoutHeredocBodies) - 1
			withoutHeredocBodies[last] = strings.TrimSpace(line) + " " + clipped
		}
		i = close // loop's i++ skips past the terminator line
	}

	lines := make([]string, 0, len(withoutHeredocBodies))
	for _, l := range withoutHeredocBodies {
		trimmed := strings.TrimSpace(l)
		if trimmed != "" {
			lines = append(lines, trimmed)
		}
	}
	if len(lines) == 0 {
		return strings.TrimSpace(raw)
	}

	meaningful := make([]string, 0, len(lines))
	for _, l := range lines {
		if trivialLineRe.MatchString(l) {
			continue
		}
		meaningful = append(meaningful, stripPipeTail(stripCdPrefix(l)))
	}
	filtered := make([]string, 0, len(meaningful))
	for _, l := range meaningful {
		if l != "" {
			filtered = append(filtered, l)
		}
	}

	chosen := filtered
	if len(chosen) == 0 {
		first := stripPipeTail(stripCdPrefix(lines[0]))
		if first != "" {
			chosen = []string{first}
		}
	}

	cmd := strings.Join(chosen, "; ")
	if len(cmd) > bashCap {
		return cmd[:bashCap-3] + "..."
	}
	return cmd
}

// ── tool summary ──

var toolSummaryFields = map[string]string{
	"Read":     "file_path",
	"Edit":     "file_path",
	"Write":    "file_path",
	"read":     "file_path",
	"edit":     "file_path",
	"write":    "file_path",
	"Glob":     "pattern",
	"Grep":     "pattern",
}

// extractPath returns the first path-like argument value, or "" if none.
func extractPath(args map[string]interface{}) string {
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
func toolOneLiner(name string, args map[string]interface{}) string {
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
