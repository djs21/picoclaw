package vcc

import (
	"strings"
	"testing"
)

func ptrInt(i int) *int { return &i }

func TestBRIEF_MAX_LINES(t *testing.T) {
	if BRIEF_MAX_LINES != 120 {
		t.Errorf("BRIEF_MAX_LINES = %d, want 120", BRIEF_MAX_LINES)
	}
}

func TestToolOneLinerRead(t *testing.T) {
	got := toolOneLiner("Read", map[string]any{"file_path": "/tmp/foo.txt"})
	want := `* Read "/tmp/foo.txt"`
	if got != want {
		t.Errorf("toolOneLiner(Read) = %q, want %q", got, want)
	}
}

func TestToolOneLinerWrite(t *testing.T) {
	got := toolOneLiner("Write", map[string]any{"file_path": "/tmp/bar.txt"})
	want := `* Write "/tmp/bar.txt"`
	if got != want {
		t.Errorf("toolOneLiner(Write) = %q, want %q", got, want)
	}
}

func TestToolOneLinerBash(t *testing.T) {
	got := toolOneLiner("bash", map[string]any{"command": "echo hello"})
	want := `* bash "echo hello"`
	if got != want {
		t.Errorf("toolOneLiner(bash) = %q, want %q", got, want)
	}
}

func TestToolOneLinerGrepQuery(t *testing.T) {
	got := toolOneLiner("Grep", map[string]any{"query": "error pattern"})
	want := `* Grep "error pattern"`
	if got != want {
		t.Errorf("toolOneLiner(Grep) = %q, want %q", got, want)
	}
}

func TestToolOneLinerUnknown(t *testing.T) {
	got := toolOneLiner("UnknownTool", map[string]any{})
	want := `* UnknownTool`
	if got != want {
		t.Errorf("toolOneLiner(UnknownTool) = %q, want %q", got, want)
	}
}

func TestCompressBash(t *testing.T) {
	got := compressBash("ls -la")
	want := "ls -la"
	if got != want {
		t.Errorf("compressBash = %q, want %q", got, want)
	}
}

func TestHeredocCloseIndex(t *testing.T) {
	lines := []string{"cat > f <<EOF", "body line", "EOF", "after"}
	got := heredocCloseIndex(lines, 0)
	want := 2
	if got != want {
		t.Errorf("heredocCloseIndex = %d, want %d", got, want)
	}
}

func TestHeredocCloseIndexNotFound(t *testing.T) {
	lines := []string{"echo << 10", "not a heredoc terminator"}
	got := heredocCloseIndex(lines, 0)
	want := -1
	if got != want {
		t.Errorf("heredocCloseIndex = %d, want %d", got, want)
	}
}

func TestBuildBriefSectionsUserWithRef(t *testing.T) {
	si := ptrInt(5)
	blocks := []NormalizedBlock{
		{Kind: BlockKindUser, Text: "hello there", SourceIndex: si},
	}
	sections := buildBriefSections(blocks)
	if len(sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(sections))
	}
	if sections[0].Header != "[user]" {
		t.Errorf("header = %q, want [user]", sections[0].Header)
	}
	if len(sections[0].Lines) != 1 || sections[0].Lines[0] != "hello there (#5)" {
		t.Errorf("lines = %#v, want [hello there (#5)]", sections[0].Lines)
	}
}

func TestBuildBriefSectionsUserNoiseSkipped(t *testing.T) {
	blocks := []NormalizedBlock{
		{Kind: BlockKindUser, Text: "   "},
		{Kind: BlockKindUser, Text: ""},
	}
	sections := buildBriefSections(blocks)
	if len(sections) != 0 {
		t.Errorf("expected 0 sections (noise skipped), got %d: %#v", len(sections), sections)
	}
}

func TestBuildBriefSectionsBash(t *testing.T) {
	si := ptrInt(7)
	blocks := []NormalizedBlock{
		{Kind: BlockKindBash, Command: "ls -la", SourceIndex: si},
	}
	sections := buildBriefSections(blocks)
	if len(sections) != 1 || sections[0].Header != "[user]" {
		t.Fatalf("bad section: %+v", sections)
	}
	if len(sections[0].Lines) != 1 || sections[0].Lines[0] != "$ ls -la (#7)" {
		t.Errorf("lines = %#v, want [$ ls -la (#7)]", sections[0].Lines)
	}
}

func TestBuildBriefSectionsToolResultOmitted(t *testing.T) {
	blocks := []NormalizedBlock{
		{Kind: BlockKindToolCall, Name: "Read", Args: map[string]any{"file_path": "/tmp/a.txt"}},
		{Kind: BlockKindToolResult, Output: "file contents here"},
	}
	sections := buildBriefSections(blocks)
	if len(sections) != 1 || sections[0].Header != "[assistant]" {
		t.Fatalf("bad sections: %+v", sections)
	}
	if len(sections[0].Lines) != 1 || sections[0].Lines[0] != `* Read "/tmp/a.txt"` {
		t.Errorf("lines = %#v, want [* Read \"/tmp/a.txt\"]", sections[0].Lines)
	}
}

func TestBuildBriefSectionsAssistantWithRef(t *testing.T) {
	si := ptrInt(9)
	blocks := []NormalizedBlock{
		{Kind: BlockKindAssistant, Text: "I am helping you", SourceIndex: si},
	}
	sections := buildBriefSections(blocks)
	if len(sections) != 1 || sections[0].Header != "[assistant]" {
		t.Fatalf("bad section: %+v", sections)
	}
	if len(sections[0].Lines) != 1 || sections[0].Lines[0] != "I am helping you (#9)" {
		t.Errorf("lines = %#v, want [I am helping you (#9)]", sections[0].Lines)
	}
}

func TestBuildBriefSectionsIdenticalToolCollapse(t *testing.T) {
	s1 := ptrInt(1)
	s2 := ptrInt(2)
	blocks := []NormalizedBlock{
		{Kind: BlockKindAssistant, Text: "reading"},
		{Kind: BlockKindToolCall, Name: "Read", Args: map[string]any{"file_path": "/tmp/a.txt"}, SourceIndex: s1},
		{Kind: BlockKindToolCall, Name: "Read", Args: map[string]any{"file_path": "/tmp/a.txt"}, SourceIndex: s2},
	}
	sections := buildBriefSections(blocks)
	if len(sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(sections))
	}
	if len(sections[0].Lines) != 2 {
		t.Errorf("expected 2 lines (text + collapsed tool), got %d: %#v", len(sections[0].Lines), sections[0].Lines)
	}
	if sections[0].Lines[1] != `* Read "/tmp/a.txt" (#1, #2) x2` {
		t.Errorf("line[1] = %q, want collapsed", sections[0].Lines[1])
	}
}

func TestBuildBriefSectionsToolCap8(t *testing.T) {
	blocks := []NormalizedBlock{
		{Kind: BlockKindAssistant, Text: "doing many things"},
	}
	for i := 0; i < 9; i++ {
		blocks = append(blocks, NormalizedBlock{
			Kind: BlockKindToolCall, Name: "Read",
			Args: map[string]any{"file_path": "/tmp/file.txt"},
		})
	}
	sections := buildBriefSections(blocks)
	if len(sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(sections))
	}
	var toolLines []string
	for _, l := range sections[0].Lines {
		if strings.HasPrefix(l, "* Read ") {
			toolLines = append(toolLines, l)
		}
	}
	if len(toolLines) != 8 {
		t.Errorf("expected 8 tool lines after cap, got %d: %#v", len(toolLines), toolLines)
	}
	foundOmission := false
	for _, l := range sections[0].Lines {
		if l == "* (1 earlier tool-call entries omitted)" {
			foundOmission = true
		}
	}
	if !foundOmission {
		t.Errorf("expected omission note, got lines: %#v", sections[0].Lines)
	}
}

func TestStringifyBrief(t *testing.T) {
	sections := []BriefLine{
		{Header: "[user]", Lines: []string{"hello (#1)"}},
		{Header: "[assistant]", Lines: []string{"hi there (#2)"}},
	}
	got := stringifyBrief(sections)
	want := "[user]\nhello (#1)\n\n[assistant]\nhi there (#2)"
	if got != want {
		t.Errorf("stringifyBrief = %q, want %q", got, want)
	}
}

func TestStringifyBriefConsecutiveToolSectionsNoBlank(t *testing.T) {
	sections := []BriefLine{
		{Header: "[assistant]", Lines: []string{`* Read "/tmp/a.txt"`}},
		{Header: "[assistant]", Lines: []string{`* Read "/tmp/b.txt"`}},
	}
	got := stringifyBrief(sections)
	want := "[assistant]\n* Read \"/tmp/a.txt\"\n[assistant]\n* Read \"/tmp/b.txt\""
	if got != want {
		t.Errorf("stringifyBrief = %q, want %q", got, want)
	}
}

func TestCompileBrief(t *testing.T) {
	blocks := []NormalizedBlock{
		{Kind: BlockKindUser, Text: "hello"},
	}
	got := compileBrief(blocks)
	want := "[user]\nhello"
	if got != want {
		t.Errorf("compileBrief = %q, want %q", got, want)
	}
}
