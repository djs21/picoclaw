package vcc

import (
	"regexp"
	"strings"
)

const (
	// TUI_SAFE_LINE_CHARS is the default soft-wrap width for terminal-safe
	// summaries (algo spec format.ts).
	TUI_SAFE_LINE_CHARS = 120

	// SECTION_SEPARATOR separates summary sections in the rendered output
	// (algo spec format.ts).
	SECTION_SEPARATOR = "\n\n---\n\n"
)

// section renders a single summary block: a heading followed by "- " prefixed
// entries. An empty line set yields an empty string so callers can skip it.
func section(title string, lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("[")
	b.WriteString(title)
	b.WriteString("]")
	b.WriteByte('\n')
	for _, line := range lines {
		b.WriteString("- ")
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

// formatSummary renders SectionData as plain structured text. Sections appear
// in fixed order: Session Goal, Files And Changes, Commits, Outstanding
// Context, User Preferences, followed by the capped brief transcript. The
// result is soft-wrapped to TUI_SAFE_LINE_CHARS columns (algo spec format.ts).
func formatSummary(d SectionData) string {
	headerSections := []string{
		section("Session Goal", d.SessionGoal),
		section("Files And Changes", d.FilesAndChanges),
		section("Commits", d.Commits),
		section("Outstanding Context", d.OutstandingContext),
		section("User Preferences", d.UserPreferences),
	}
	var headerParts []string
	for _, s := range headerSections {
		if s != "" {
			headerParts = append(headerParts, s)
		}
	}
	var parts []string
	if len(headerParts) > 0 {
		parts = append(parts, strings.Join(headerParts, "\n\n"))
	}
	if d.BriefTranscript != "" {
		parts = append(parts, capBrief(d.BriefTranscript, BRIEF_MAX_LINES))
	}
	if len(parts) == 0 {
		return ""
	}
	return wrapLongLines(strings.Join(parts, SECTION_SEPARATOR), TUI_SAFE_LINE_CHARS)
}

// wrapLongLines wraps every line of s so no line exceeds max chars. Lines are
// broken at word boundaries; a single word longer than max is hard-cut.
func wrapLongLines(s string, max int) string {
	if max <= 0 {
		return s
	}
	var out strings.Builder
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if i > 0 {
			out.WriteByte('\n')
		}
		out.WriteString(wrapLine(line, max))
	}
	return out.String()
}

func wrapLine(line string, max int) string {
	if len(line) <= max {
		return line
	}
	var out strings.Builder
	for len(line) > max {
		cut := max
		if idx := strings.LastIndex(line[:max], " "); idx > 0 {
			cut = idx
		}
		out.WriteString(line[:cut])
		out.WriteByte('\n')
		line = strings.TrimLeft(line[cut:], " ")
	}
	out.WriteString(line)
	return out.String()
}

var briefHeaderRe = regexp.MustCompile(`^\[.+\]`)

// capBrief keeps at most maxLines of the brief transcript, preserving the
// newest lines and avoiding a cut immediately before a section header.
func capBrief(text string, maxLines int) string {
	if text == "" || maxLines <= 0 {
		return ""
	}
	lines := strings.Split(text, "\n")
	if len(lines) <= maxLines {
		return text
	}
	omitted := len(lines) - maxLines
	kept := lines[len(lines)-maxLines:]
	if firstHeader := findFirstHeader(kept); firstHeader > 0 {
		kept = kept[firstHeader:]
	}
	return "...(" + itoa(omitted) + " earlier lines omitted)\n\n" + strings.Join(kept, "\n")
}

func findFirstHeader(lines []string) int {
	for i, line := range lines {
		if briefHeaderRe.MatchString(line) {
			return i
		}
	}
	return -1
}
