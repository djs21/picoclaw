package vcc

import (
	"strings"
	"testing"
)

func mkSection(header string, lines ...string) string {
	b := new(strings.Builder)
	b.WriteString("[" + header + "]")
	for _, l := range lines {
		b.WriteString("\n- " + l)
	}
	return b.String()
}

func TestMergePreviousStickySectionsDedupPrevThenFresh(t *testing.T) {
	prev := mkSection("Session Goal", "goal-a", "goal-b")
	fresh := mkSection("Session Goal", "goal-b", "goal-c")
	got := mergePrevious(prev, fresh)
	if !strings.Contains(got, "[Session Goal]") {
		t.Fatalf("expected [Session Goal] block, got:\n%s", got)
	}
	block := blockOf(t, got, "Session Goal")
	// line-level dedup prev then fresh: goal-a, goal-b, goal-c
	want := []string{"goal-a", "goal-b", "goal-c"}
	if !eq(block, want) {
		t.Fatalf("Session Goal dedup = %v, want %v", block, want)
	}
}

func TestMergePreviousStickySectionsCapped(t *testing.T) {
	var prev, fresh []string
	for i := 0; i < 10; i++ {
		prev = append(prev, "prev-goal-"+itoa(i))
	}
	for i := 0; i < 10; i++ {
		fresh = append(fresh, "fresh-goal-"+itoa(i))
	}
	got := mergePrevious(mkSection("Session Goal", prev...), mkSection("Session Goal", fresh...))
	block := blockOf(t, got, "Session Goal")
	// Session Goal cap 8, keep last 8 of deduped prev+fresh
	if len(block) != sessionGoalCap {
		t.Fatalf("Session Goal block len = %d, want %d (cap)", len(block), sessionGoalCap)
	}
	if block[0] != "fresh-goal-2" || block[len(block)-1] != "fresh-goal-9" {
		t.Fatalf("expected last 8 fresh goals, got %v", block)
	}
}

func TestMergePreviousOutstandingContextVolatileFreshOnly(t *testing.T) {
	prev := mkSection("Outstanding Context", "old-ctx")
	fresh := mkSection("Outstanding Context", "new-ctx")
	got := mergePrevious(prev, fresh)
	block := blockOf(t, got, "Outstanding Context")
	if !eq(block, []string{"new-ctx"}) {
		t.Fatalf("Outstanding Context = %v, want fresh-only [new-ctx]", block)
	}
}

func TestMergePreviousOutstandingContextFreshEmpty(t *testing.T) {
	prev := mkSection("Outstanding Context", "old-ctx")
	fresh := mkSection("Outstanding Context")
	got := mergePrevious(prev, fresh)
	if strings.Contains(got, "[Outstanding Context]") {
		t.Fatalf("volatile Outstanding Context with empty fresh should be omitted, got:\n%s", got)
	}
}

func TestMergePreviousEmptyPrevYieldsFresh(t *testing.T) {
	fresh := mkSection("Commits", "commit-1", "commit-2")
	got := mergePrevious("", fresh)
	block := blockOf(t, got, "Commits")
	if !eq(block, []string{"commit-1", "commit-2"}) {
		t.Fatalf("empty prev -> fresh = %v, want [commit-1 commit-2]", block)
	}
}

func TestMergePreviousEmptyFreshYieldsPrev(t *testing.T) {
	prev := mkSection("User Preferences", "pref-1")
	got := mergePrevious(prev, "")
	block := blockOf(t, got, "User Preferences")
	if !eq(block, []string{"pref-1"}) {
		t.Fatalf("empty fresh -> prev = %v, want [pref-1]", block)
	}
}

func TestMergeFileLinesByCategory(t *testing.T) {
	prev := []string{
		"- Modified: src/a.go",
		"- Created: src/b.go",
		"- Read: src/c.go (+1 more)",
	}
	fresh := []string{
		"- Modified: src/b.go",
		"- Modified: src/d.go",
		"- Read: src/e.go",
	}
	got := mergeFileLines(prev, fresh)
	// Modified wins over Created: src/b.go appears only under Modified.
	// union: Modified=[a,b,d], Created=[], Read=[c,e]
	want := []string{
		"- Modified: src/a.go",
		"- Modified: src/b.go",
		"- Modified: src/d.go",
		"- Read: src/c.go",
		"- Read: src/e.go",
	}
	if !eq(got, want) {
		t.Fatalf("mergeFileLines = %v, want %v", got, want)
	}
}

func TestMergeFileLinesCappedWithMoreSuffix(t *testing.T) {
	var prev, fresh []string
	for i := 0; i < 12; i++ {
		prev = append(prev, "- Modified: src/m"+itoa(i)+".go")
	}
	got := mergeFileLines(prev, fresh)
	// cap each category at 10 with "(+N more)" suffix
	if len(got) != fileCatCap {
		t.Fatalf("Modified category len = %d, want %d (cap)", len(got), fileCatCap)
	}
	if !strings.HasSuffix(got[len(got)-1], " (+2 more)") {
		t.Fatalf("expected cap suffix, got last line: %s", got[len(got)-1])
	}
}

func TestStripRecallNote(t *testing.T) {
	body := "hello world\n"
	with := body + RECALL_NOTE
	got := stripRecallNote(with)
	// trailing RECALL_NOTE and surrounding separators/whitespace are removed
	if got != "hello world" {
		t.Fatalf("stripRecallNote = %q, want %q", got, "hello world")
	}
	// no recall note present -> unchanged
	if stripRecallNote(body) != body {
		t.Fatalf("stripRecallNote should leave text without RECALL_NOTE unchanged")
	}
}

func TestMergePreviousAppendsRecallNoteOnce(t *testing.T) {
	prev := mkSection("Session Goal", "goal")
	fresh := mkSection("Session Goal", "goal")
	got := mergePrevious(prev, fresh)
	count := strings.Count(got, RECALL_NOTE)
	if count != 1 {
		t.Fatalf("RECALL_NOTE count = %d, want 1", count)
	}
	if !strings.HasSuffix(strings.TrimSpace(got), RECALL_NOTE) {
		t.Fatalf("RECALL_NOTE not appended at end:\n%s", got)
	}
	// previous summary containing RECALL_NOTE is stripped before merge (not restacked)
	prevWithNote := mkSection("Session Goal", "goal") + "\n\n" + RECALL_NOTE
	got2 := mergePrevious(prevWithNote, fresh)
	if strings.Count(got2, RECALL_NOTE) != 1 {
		t.Fatalf("RECALL_NOTE count with noted prev = %d, want 1", strings.Count(got2, RECALL_NOTE))
	}
}

func blockOf(t *testing.T, text, header string) []string {
	t.Helper()
	token := "[" + header + "]"
	idx := strings.Index(text, token)
	if idx == -1 {
		t.Fatalf("header [%s] not found in:\n%s", header, text)
	}
	rest := text[idx+len(token):]
	// cut at next header or end
	next := -1
	for _, h := range []string{"Session Goal", "Files And Changes", "Commits", "Outstanding Context", "User Preferences"} {
		if h == header {
			continue
		}
		if j := strings.Index(rest, "\n["+h+"]"); j != -1 && (next == -1 || j < next) {
			next = j
		}
	}
	if next != -1 {
		rest = rest[:next]
	}
	var lines []string
	for _, raw := range strings.Split(rest, "\n") {
		line := strings.TrimSpace(raw)
		if !strings.HasPrefix(line, "- ") {
			continue
		}
		if strings.Contains(line, "<skill") || strings.Contains(line, "</skill") {
			continue
		}
		lines = append(lines, strings.TrimPrefix(line, "- "))
	}
	return lines
}

func eq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
