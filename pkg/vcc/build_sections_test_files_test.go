package vcc

import (
	"strings"
	"testing"
)

// ── extractFiles / formatFileActivity ──

func TestFormatFileActivityModifiedWinsOverCreated(t *testing.T) {
	// Use tool calls in blocks since FileOps is write-only (cannot retrieve paths).
	blocks := []NormalizedBlock{
		{Kind: BlockKindToolCall, Name: "Edit", Args: map[string]any{"file_path": "/src/a.txt"}},
		{Kind: BlockKindToolCall, Name: "Write", Args: map[string]any{"file_path": "/src/a.txt"}},
		{Kind: BlockKindToolCall, Name: "Write", Args: map[string]any{"file_path": "/src/b.txt"}},
	}
	got := formatFileActivity(blocks, nil)

	joined := strings.Join(got, " ")
	// a.txt should be in Modified, not Created (Modified wins over Created)
	if strings.Contains(joined, "Created: a.txt") || strings.Contains(joined, "Created: /src/a.txt") {
		t.Errorf("Modified wins over Created: a.txt should not appear in Created, got %v", got)
	}
	if !strings.Contains(joined, "Modified: a.txt") {
		t.Errorf("expected a.txt in Modified, got %v", got)
	}
	if !strings.Contains(joined, "b.txt") {
		t.Errorf("expected b.txt in Modified (Write is both write+create), got %v", got)
	}
}

func TestFormatFileActivityCapsAt10PerCategory(t *testing.T) {
	var blocks []NormalizedBlock
	for i := 0; i < 15; i++ {
		blocks = append(blocks, NormalizedBlock{
			Kind: BlockKindToolCall, Name: "Edit",
			Args: map[string]any{"file_path": "/src/proj" + string(rune('a'+i)) + "/file.txt"},
		})
	}
	for i := 0; i < 12; i++ {
		blocks = append(blocks, NormalizedBlock{
			Kind: BlockKindToolCall, Name: "Write",
			Args: map[string]any{"file_path": "/src/proj" + string(rune('a'+i)) + "/new.txt"},
		})
	}

	got := formatFileActivity(blocks, nil)

	// Check Modified category cap
	for _, line := range got {
		if strings.HasPrefix(line, "Modified:") {
			paths := strings.Split(strings.TrimPrefix(line, "Modified: "), ", ")
			// Strip the "(+N more)" suffix if present
			if len(paths) > 0 {
				last := paths[len(paths)-1]
				if strings.HasSuffix(last, "more)") {
					paths = paths[:len(paths)-1]
				}
			}
			if len(paths) > 10 {
				t.Errorf("Modified category has %d entries, want <= 10", len(paths))
			}
		}
		if strings.HasPrefix(line, "Created:") {
			paths := strings.Split(strings.TrimPrefix(line, "Created: "), ", ")
			if len(paths) > 0 {
				last := paths[len(paths)-1]
				if strings.HasSuffix(last, "more)") {
					paths = paths[:len(paths)-1]
				}
			}
			if len(paths) > 10 {
				t.Errorf("Created category has %d entries, want <= 10", len(paths))
			}
		}
	}
}

func TestFormatFileActivityReadCategory(t *testing.T) {
	blocks := []NormalizedBlock{
		{Kind: BlockKindToolCall, Name: "Read", Args: map[string]any{"file_path": "/src/readme.txt"}},
	}
	got := formatFileActivity(blocks, nil)

	joined := strings.Join(got, " ")
	if !strings.Contains(joined, "Read: /src/readme.txt") {
		t.Errorf("expected Read category, got %v", got)
	}
}

func TestExtractFilesFromToolCalls(t *testing.T) {
	blocks := []NormalizedBlock{
		{Kind: BlockKindToolCall, Name: "Read", Args: map[string]any{"file_path": "/tmp/a.txt"}},
		{Kind: BlockKindToolCall, Name: "Edit", Args: map[string]any{"file_path": "/tmp/b.txt"}},
		{Kind: BlockKindToolCall, Name: "Write", Args: map[string]any{"file_path": "/tmp/c.txt"}},
	}
	got := extractFiles(blocks, nil)

	joined := strings.Join(got, " ")
	if !strings.Contains(joined, "a.txt") {
		t.Errorf("expected a.txt (Read), got %v", got)
	}
	if !strings.Contains(joined, "b.txt") {
		t.Errorf("expected b.txt (Edit/modified), got %v", got)
	}
	if !strings.Contains(joined, "c.txt") {
		t.Errorf("expected c.txt (Write/created), got %v", got)
	}
}

// ── extractCommits / formatCommits ──

func TestExtractCommitsEmpty(t *testing.T) {
	got := extractCommits([]NormalizedBlock{})
	if len(got) != 0 {
		t.Errorf("extractCommits(empty) = %v, want empty", got)
	}
}

func TestExtractCommitsGitCommit(t *testing.T) {
	blocks := []NormalizedBlock{
		{Kind: BlockKindToolCall, Name: "bash", Command: `git commit -m "initial commit"`, Args: map[string]any{"command": `git commit -m "initial commit"`}},
		{Kind: BlockKindToolResult, Name: "bash", Text: "[main abc1234] initial commit"},
	}
	got := extractCommits(blocks)
	if len(got) != 1 {
		t.Fatalf("extractCommits = %v, want 1 commit", got)
	}
	if !strings.Contains(got[0], "initial commit") {
		t.Errorf("commit message missing: %q", got[0])
	}
}

func TestFormatCommitsCapsAt8(t *testing.T) {
	var commits []string
	for i := 0; i < 12; i++ {
		commits = append(commits, "commit message "+string(rune('a'+i)))
	}
	got := formatCommits(commits)
	if len(got) > 8 {
		t.Errorf("formatCommits returned %d items, want <= 8", len(got))
	}
}

// ── extractPreferences / dedupPreferencesAgainstGoals ──

func TestExtractPreferencesEmpty(t *testing.T) {
	got := extractPreferences([]NormalizedBlock{})
	if len(got) != 0 {
		t.Errorf("extractPreferences(empty) = %v, want empty", got)
	}
}

func TestExtractPreferencesFromUserBlocks(t *testing.T) {
	blocks := []NormalizedBlock{
		{Kind: BlockKindUser, Text: "I prefer using Go for backend services"},
		{Kind: BlockKindUser, Text: "don't want any external databases"},
		{Kind: BlockKindAssistant, Text: "prefer something else"}, // assistant prefs ignored
	}
	got := extractPreferences(blocks)
	if len(got) != 2 {
		t.Errorf("extractPreferences = %v, want 2 prefs", got)
	}
}

func TestExtractPreferencesCapsAt10(t *testing.T) {
	var blocks []NormalizedBlock
	for i := 0; i < 15; i++ {
		blocks = append(blocks, NormalizedBlock{
			Kind: BlockKindUser,
			Text: "Always use option " + string(rune('a'+i)) + " for everything",
		})
	}
	got := extractPreferences(blocks)
	if len(got) > 10 {
		t.Errorf("extractPreferences returned %d items, want <= 10", len(got))
	}
}

func TestDedupPreferencesAgainstGoals(t *testing.T) {
	prefs := []string{"use Go for backend", "use PostgreSQL", "use Go for backend"}
	goals := []string{"use Go for backend", "implement login"}
	got := dedupPreferencesAgainstGoals(prefs, goals)
	if len(got) != 1 || got[0] != "use PostgreSQL" {
		t.Errorf("dedupPreferencesAgainstGoals = %v, want [use PostgreSQL]", got)
	}
}

func TestDedupPreferencesCaseInsensitive(t *testing.T) {
	prefs := []string{"USE Go for backend"}
	goals := []string{"use go for backend"}
	got := dedupPreferencesAgainstGoals(prefs, goals)
	if len(got) != 0 {
		t.Errorf("dedupPreferencesAgainstGoals (case-insensitive) = %v, want empty", got)
	}
}

// ── buildSections ──

func TestBuildSectionsEmpty(t *testing.T) {
	input := BuildSectionsInput{Blocks: []NormalizedBlock{}}
	got := buildSections(input)
	if len(got.SessionGoal) != 0 {
		t.Errorf("SessionGoal = %v, want empty", got.SessionGoal)
	}
	if len(got.OutstandingContext) != 0 {
		t.Errorf("OutstandingContext = %v, want empty", got.OutstandingContext)
	}
	if len(got.FilesAndChanges) != 0 {
		t.Errorf("FilesAndChanges = %v, want empty", got.FilesAndChanges)
	}
	if len(got.Commits) != 0 {
		t.Errorf("Commits = %v, want empty", got.Commits)
	}
	if len(got.UserPreferences) != 0 {
		t.Errorf("UserPreferences = %v, want empty", got.UserPreferences)
	}
}

func TestBuildSectionsAllCaps(t *testing.T) {
	var blocks []NormalizedBlock

	// 12 goals (should cap at 8)
	for i := 0; i < 12; i++ {
		blocks = append(blocks, NormalizedBlock{
			Kind: BlockKindUser,
			Text: "Goal " + string(rune('a'+i)) + " that needs doing",
		})
	}
	// 15 files modified (should cap at 10)
	for i := 0; i < 15; i++ {
		blocks = append(blocks, NormalizedBlock{
			Kind: BlockKindToolCall, Name: "Edit",
			Args: map[string]any{"file_path": "/src/file" + string(rune('a'+i)) + ".go"},
		})
	}
	// 12 commits (should cap at 8)
	for i := 0; i < 12; i++ {
		blocks = append(blocks, NormalizedBlock{
			Kind: BlockKindToolCall, Name: "bash",
			Command: `git commit -m "msg ` + string(rune('a'+i)) + `"`,
			Args:    map[string]any{"command": `git commit -m "msg ` + string(rune('a'+i)) + `"`},
		})
		if i < 12 {
			blocks = append(blocks, NormalizedBlock{
				Kind: BlockKindToolResult, Name: "bash",
				Text: "[main abc" + string(rune('a'+i)) + "] msg",
			})
		}
	}
	// 15 preferences (should cap at 10)
	for i := 0; i < 15; i++ {
		blocks = append(blocks, NormalizedBlock{
			Kind: BlockKindUser,
			Text: "Prefer option " + string(rune('a'+i)) + " always",
		})
	}

	input := BuildSectionsInput{Blocks: blocks}
	got := buildSections(input)

	if len(got.SessionGoal) > 8 {
		t.Errorf("SessionGoal has %d entries, want <= 8", len(got.SessionGoal))
	}
	for _, g := range got.SessionGoal {
		if len(g) > 200 {
			t.Errorf("SessionGoal entry exceeds 200 chars: %q", g)
		}
	}
	if len(got.OutstandingContext) > 5 {
		t.Errorf("OutstandingContext has %d entries, want <= 5", len(got.OutstandingContext))
	}
	if len(got.Commits) > 8 {
		t.Errorf("Commits has %d entries, want <= 8", len(got.Commits))
	}
	if len(got.UserPreferences) > 10 {
		t.Errorf("UserPreferences has %d entries, want <= 10", len(got.UserPreferences))
	}
}

func TestBuildSectionsModifiedWinsOverCreated(t *testing.T) {
	// Same path edited (modified) and created → Modified wins
	// Note: "Write" tool is both a write AND create tool, so Written files
	// appear in Modified (which wins over Created per the algorithm).
	blocks := []NormalizedBlock{
		{Kind: BlockKindToolCall, Name: "Edit", Args: map[string]any{"file_path": "/src/shared.go"}},
		{Kind: BlockKindToolCall, Name: "Write", Args: map[string]any{"file_path": "/src/shared.go"}},
		{Kind: BlockKindToolCall, Name: "Read", Args: map[string]any{"file_path": "/src/readonly.go"}},
	}
	input := BuildSectionsInput{Blocks: blocks}
	got := buildSections(input)

	joined := strings.Join(got.FilesAndChanges, " ")
	if strings.Contains(joined, "Created: shared.go") {
		t.Errorf("Modified wins over Created: shared.go should not be in Created")
	}
	if !strings.Contains(joined, "Modified: shared.go") {
		t.Errorf("expected shared.go in Modified, got %v", got.FilesAndChanges)
	}
	if !strings.Contains(joined, "Read: readonly.go") {
		t.Errorf("expected readonly.go in Read, got %v", got.FilesAndChanges)
	}
}
