package vcc

import (
	"strings"
	"testing"
)

const (
	wrapMax  = 120
	briefCap = 120
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

func TestFormatSummaryBriefTranscriptRendered(t *testing.T) {
	d := SectionData{
		BriefTranscript: "[user]\nbrief line one",
		SessionGoal:     []string{"a goal"},
	}
	got := formatSummary(d)
	if !strings.Contains(got, "brief line one") {
		t.Errorf("formatSummary should render BriefTranscript:\n%s", got)
	}
	if !strings.Contains(got, "## Session Goal\n- a goal") {
		t.Errorf("formatSummary missing goal:\n%s", got)
	}
	if !strings.Contains(got, "---") {
		t.Errorf("formatSummary should separate sections with ---:\n%s", got)
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
	text := "[user]\na\n[assistant]\nb"
	got := capBrief(text, briefCap)
	if got != text {
		t.Errorf("capBrief(under) = %q, want %q", got, text)
	}
}

func TestCapBriefOverLimit(t *testing.T) {
	lines := make([]string, briefCap+10)
	for i := range lines {
		lines[i] = "line" + itoa(i)
	}
	text := strings.Join(lines, "\n")
	got := capBrief(text, briefCap)
	if !strings.HasPrefix(got, "...(10 earlier lines omitted)\n\n") {
		t.Errorf("capBrief(over) missing omission note: %q", got)
	}
	body := strings.Split(got, "\n")
	if len(body) > briefCap+3 {
		t.Errorf("capBrief(over) returned too many lines: %d", len(body))
	}
}

func TestCapBriefZeroLimit(t *testing.T) {
	if got := capBrief("a\nb", 0); got != "" {
		t.Errorf("capBrief(zero) = %q, want empty", got)
	}
}

func TestCapBriefEmpty(t *testing.T) {
	if got := capBrief("", briefCap); got != "" {
		t.Errorf("capBrief(empty) = %q, want empty", got)
	}
}
