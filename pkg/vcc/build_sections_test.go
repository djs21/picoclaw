package vcc

import (
	"strings"
	"testing"
)

// testFileOps implements FileOps for testing.
type testFileOps struct {
	reads    []string
	modifies []string
	creates  []string
}

func (f *testFileOps) Read(path string)     { f.reads = append(f.reads, path) }
func (f *testFileOps) Modified(path string) { f.modifies = append(f.modifies, path) }
func (f *testFileOps) Created(path string)  { f.creates = append(f.creates, path) }

// ── extractGoals ──

func TestExtractGoalsEmpty(t *testing.T) {
	got := extractGoals([]NormalizedBlock{})
	if len(got) != 0 {
		t.Errorf("extractGoals(empty) = %v, want empty", got)
	}
}

func TestExtractGoalsUserBlock(t *testing.T) {
	blocks := []NormalizedBlock{
		{Kind: BlockKindUser, Text: "implement the login page\nfix the navbar bug\nadd dark mode toggle"},
	}
	got := extractGoals(blocks)
	if len(got) != 3 {
		t.Fatalf("extractGoals = %v, want 3 goals", got)
	}
	if got[0] != "implement the login page" {
		t.Errorf("goal[0] = %q, want %q", got[0], "implement the login page")
	}
}

func TestExtractGoalsCapsAt8(t *testing.T) {
	var blocks []NormalizedBlock
	for i := 0; i < 12; i++ {
		blocks = append(blocks, NormalizedBlock{
			Kind: BlockKindUser,
			Text: strings.Join([]string{"goal", string(rune('a' + i))}, " "),
		})
	}
	got := extractGoals(blocks)
	if len(got) > 8 {
		t.Errorf("extractGoals returned %d goals, want <= 8", len(got))
	}
	for _, g := range got {
		if len(g) > 200 {
			t.Errorf("goal %q exceeds 200 chars (%d)", g, len(g))
		}
	}
}

func TestExtractGoalsFirstBlockGetsSix(t *testing.T) {
	blocks := []NormalizedBlock{
		{Kind: BlockKindUser, Text: "one\ntwo\nthree\nfour\nfive\nsix\nseven\neight"},
	}
	got := extractGoals(blocks)
	// First user block yields up to 6 goals
	if len(got) > 6 {
		t.Errorf("first block yielded %d goals, want <= 6", len(got))
	}
}

func TestExtractGoalsNonUserBlocksIgnored(t *testing.T) {
	blocks := []NormalizedBlock{
		{Kind: BlockKindAssistant, Text: "implement something"},
		{Kind: BlockKindToolCall, Name: "Read", Args: map[string]interface{}{"file_path": "/tmp/a.txt"}},
		{Kind: BlockKindUser, Text: "real goal here"},
	}
	got := extractGoals(blocks)
	if len(got) != 1 || got[0] != "real goal here" {
		t.Errorf("extractGoals = %v, want [real goal here]", got)
	}
}

// ── extractOutstandingContext ──

func TestExtractOutstandingContextEmpty(t *testing.T) {
	got := extractOutstandingContext([]NormalizedBlock{})
	if len(got) != 0 {
		t.Errorf("extractOutstandingContext(empty) = %v, want empty", got)
	}
}

func TestExtractOutstandingContextBlockerRe(t *testing.T) {
	// Blocker lines must start with an uppercase letter or special char
	// (sentence-like start required by the algorithm).
	blocks := []NormalizedBlock{
		{Kind: BlockKindUser, Text: "The build is broken and needs fixing"},
		{Kind: BlockKindAssistant, Text: "I will look into it"},
		{Kind: BlockKindUser, Text: "Blocked by missing API keys"},
	}
	got := extractOutstandingContext(blocks)
	if len(got) == 0 {
		t.Fatalf("extractOutstandingContext = %v, want blocker items", got)
	}
	for _, item := range got {
		if len(item) < 15 {
			t.Errorf("outstanding context item too short: %q", item)
		}
	}
}

func TestExtractOutstandingContextCapsAt5(t *testing.T) {
	var blocks []NormalizedBlock
	for i := 0; i < 10; i++ {
		blocks = append(blocks, NormalizedBlock{
			Kind: BlockKindUser,
			Text: "Blocker issue number " + string(rune('a'+i)) + " needs attention now",
		})
	}
	got := extractOutstandingContext(blocks)
	if len(got) > 5 {
		t.Errorf("extractOutstandingContext returned %d items, want <= 5", len(got))
	}
}

func TestExtractOutstandingContextTail20Scan(t *testing.T) {
	// Build more than 20 blocks; only the tail-20 should be scanned.
	var blocks []NormalizedBlock
	for i := 0; i < 30; i++ {
		kind := BlockKindAssistant
		if i == 5 {
			kind = BlockKindUser
		}
		blocks = append(blocks, NormalizedBlock{Kind: kind, Text: "ordinary text"})
	}
	// Add a blocker in block index 5 (outside tail-20 when total is 30+1)
	blocks[5] = NormalizedBlock{Kind: BlockKindUser, Text: "Early blocker that should not appear"}
	// Add a blocker in the last block (inside tail-20)
	last := len(blocks) - 1
	blocks[last] = NormalizedBlock{Kind: BlockKindUser, Text: "Late blocker that should appear"}

	got := extractOutstandingContext(blocks)
	foundLate := false
	for _, item := range got {
		if strings.Contains(item, "Late blocker") {
			foundLate = true
		}
	}
	if !foundLate {
		t.Errorf("expected late blocker in outstanding context, got %v", got)
	}
}

func TestExtractOutstandingContextSkipsContinuationFragments(t *testing.T) {
	blocks := []NormalizedBlock{
		{Kind: BlockKindUser, Text: "- this is a sub-bullet\nBroken build needs fix"},
	}
	got := extractOutstandingContext(blocks)
	// The sub-bullet starting with "-" should be skipped; "Broken build..." matches
	found := false
	for _, item := range got {
		if strings.Contains(item, "Broken build") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected blocker line in outstanding context, got %v", got)
	}
}
