package vcc

import (
	"strings"
	"testing"
)

const (
	wrapMax    = 120
	briefCap   = 120
)

func TestRECALL_NOTEConstant(t *testing.T) {
	if RECALL_NOTE == "" {
		t.Error("RECALL_NOTE must be a non-empty constant")
	}
	if strings.Contains(RECALL_NOTE, "\n") {
		t.Errorf("RECALL_NOTE must be a single line, got %q", RECALL_NOTE)
	}
}

func TestFormatSummarySectionOrder(t *testing.T) {
	d := SectionData{
		SessionGoal:        []string{"goal one", "goal two"},
		FilesAndChanges:    []string{"modified pkg/vcc/format.go"},
		Commits:            []string{"abc1234 fix summary"},
		OutstandingContext: []string{"finish vcc compiler"},
		UserPreferences:    []string{"theme=dark"},
	}
	got := formatSummary(d)

	order := []string{
		"## Session Goal",
		"## Files And Changes",
		"## Commits",
		"## Outstanding Context",
		"## User Preferences",
	}
	pos := -1
	for _, heading := range order {
		idx := strings.Index(got, heading)
		if idx < 0 {
			t.Errorf("formatSummary missing heading %q", heading)
			continue
		}
		if idx <= pos {
			t.Errorf("formatSummary heading %q out of order (at %d, previous at %d)", heading, idx, pos)
		}
		pos = idx
	}
}

func TestFormatSummaryEntriesPrefixed(t *testing.T) {
	d := SectionData{
		SessionGoal: []string{"improve summary"},
	}
	got := formatSummary(d)
	if !strings.Contains(got, "## Session Goal\n- improve summary") {
		t.Errorf("formatSummary entry not prefixed with '- ':\n%s", got)
	}
}

func TestFormatSummaryOmitsEmptySections(t *testing.T) {
	d := SectionData{
		Commits: []string{"abc123"},
	}
	got := formatSummary(d)
	if strings.Contains(got, "Session Goal") {
		t.Errorf("formatSummary should omit empty sections:\n%s", got)
	}
	if !strings.Contains(got, "## Commits\n- abc123") {
		t.Errorf("formatSummary missing commit entry:\n%s", got)
	}
}

func TestFormatSummaryEmptyInput(t *testing.T) {
	got := formatSummary(SectionData{})
	if got != "" {
		t.Errorf("formatSummary(empty) = %q, want empty", got)
	}
}

func TestFormatSummaryBriefTranscriptNotRendered(t *testing.T) {
	d := SectionData{
		BriefTranscript:    "brief line one\nbrief line two",
		SessionGoal:        []string{"a goal"},
	}
	got := formatSummary(d)
	if strings.Contains(got, "brief line one") {
		t.Errorf("formatSummary should not render BriefTranscript:\n%s", got)
	}
	if !strings.Contains(got, "## Session Goal\n- a goal") {
		t.Errorf("formatSummary missing goal:\n%s", got)
	}
}

func TestWrapLongLinesShortUnchanged(t *testing.T) {
	got := wrapLongLines("short line", wrapMax)
	if got != "short line" {
		t.Errorf("wrapLongLines(short) = %q, want %q", got, "short line")
	}
}

func TestWrapLongLinesExactLimitUnchanged(t *testing.T) {
	s := strings.Repeat("a", wrapMax)
	got := wrapLongLines(s, wrapMax)
	if got != s {
		t.Errorf("wrapLongLines(exact) = %q, want %q", got, s)
	}
}

func TestWrapLongLinesOverLimit(t *testing.T) {
	s := strings.Repeat("word ", 40) // ~200 chars
	got := wrapLongLines(s, wrapMax)
	for _, line := range strings.Split(got, "\n") {
		if len(line) > wrapMax {
			t.Errorf("wrapLongLines produced line longer than %d: %q", wrapMax, line)
		}
	}
	if strings.Contains(got, "  ") {
		t.Errorf("wrapLongLines should not collapse words with double spaces:\n%s", got)
	}
}

func TestWrapLongLinesHardCutOversizedWord(t *testing.T) {
	s := strings.Repeat("x", wrapMax+50)
	got := wrapLongLines(s, wrapMax)
	parts := strings.Split(got, "\n")
	if len(parts) < 2 {
		t.Errorf("wrapLongLines(oversized) should split into multiple lines, got %q", got)
	}
	for _, line := range parts {
		if len(line) > wrapMax {
			t.Errorf("wrapLongLines hard-cut line longer than %d: %q", wrapMax, line)
		}
	}
}

func TestWrapLongLinesPreservesLineBreaks(t *testing.T) {
	s := "line one\n" + strings.Repeat("z", wrapMax+20) + "\nline three"
	got := wrapLongLines(s, wrapMax)
	parts := strings.Split(got, "\n")
	if parts[0] != "line one" {
		t.Errorf("wrapLongLines first line = %q, want %q", parts[0], "line one")
	}
	if parts[len(parts)-1] != "line three" {
		t.Errorf("wrapLongLines last line = %q, want %q", parts[len(parts)-1], "line three")
	}
}

func TestCapBriefUnderLimit(t *testing.T) {
	lines := []string{"a", "b", "c"}
	got := capBrief(lines, briefCap)
	if len(got) != len(lines) {
		t.Errorf("capBrief(under) = %v, want %v", got, lines)
	}
}

func TestCapBriefOverLimit(t *testing.T) {
	lines := make([]string, briefCap+10)
	for i := range lines {
		lines[i] = string(rune('a' + i%26))
	}
	got := capBrief(lines, briefCap)
	if len(got) != briefCap {
		t.Errorf("capBrief(over) len = %d, want %d", len(got), briefCap)
	}
}

func TestCapBriefZeroLimit(t *testing.T) {
	got := capBrief([]string{"a", "b"}, 0)
	if len(got) != 0 {
		t.Errorf("capBrief(zero) = %v, want empty", got)
	}
}

func TestCapBriefNil(t *testing.T) {
	got := capBrief(nil, briefCap)
	if got != nil {
		t.Errorf("capBrief(nil) = %v, want nil", got)
	}
}
