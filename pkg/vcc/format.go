package vcc

import "strings"

// RECALL_NOTE is appended once to a merged summary to mark that a previous
// summary was retained (algo spec summarize.ts:179-191).
const RECALL_NOTE = "[RECALL] This summary continues from a previous compaction."

// section renders a single summary block: a heading followed by "- " prefixed
// entries. An empty line set yields an empty string so callers can skip it.
func section(title string, lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("## ")
	b.WriteString(title)
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
// Context, User Preferences. Empty sections are omitted. BriefTranscript is
// rendered elsewhere (capBrief/brief.go), not here.
func formatSummary(d SectionData) string {
	var parts []string
	if s := section("Session Goal", d.SessionGoal); s != "" {
		parts = append(parts, s)
	}
	if s := section("Files And Changes", d.FilesAndChanges); s != "" {
		parts = append(parts, s)
	}
	if s := section("Commits", d.Commits); s != "" {
		parts = append(parts, s)
	}
	if s := section("Outstanding Context", d.OutstandingContext); s != "" {
		parts = append(parts, s)
	}
	if s := section("User Preferences", d.UserPreferences); s != "" {
		parts = append(parts, s)
	}
	return strings.Join(parts, "\n")
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

// capBrief returns at most maxLines of the brief transcript lines.
func capBrief(lines []string, maxLines int) []string {
	if maxLines < 0 {
		maxLines = 0
	}
	if len(lines) <= maxLines {
		return lines
	}
	return lines[:maxLines]
}
