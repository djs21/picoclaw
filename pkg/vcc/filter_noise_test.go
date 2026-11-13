package vcc

import (
	"testing"
)

func TestFilterNoiseDropsNoiseToolCall(t *testing.T) {
	blocks := []NormalizedBlock{
		{Kind: BlockKindToolCall, Name: "TodoWrite", SourceIndex: ptr(1)},
		{Kind: BlockKindToolCall, Name: "Write", SourceIndex: ptr(2)},
	}
	got := filterNoise(blocks)
	if len(got) != 1 {
		t.Fatalf("filterNoise = %d blocks, want 1", len(got))
	}
	if got[0].Name != "Write" {
		t.Errorf("got[0].Name = %q, want %q", got[0].Name, "Write")
	}
}

func TestFilterNoiseDropsNoiseToolResult(t *testing.T) {
	blocks := []NormalizedBlock{
		{Kind: BlockKindToolResult, Name: "WebSearch", Text: "results", SourceIndex: ptr(1)},
		{Kind: BlockKindToolResult, Name: "Read", Text: "content", SourceIndex: ptr(2)},
	}
	got := filterNoise(blocks)
	if len(got) != 1 {
		t.Fatalf("filterNoise = %d blocks, want 1", len(got))
	}
	if got[0].Name != "Read" {
		t.Errorf("got[0].Name = %q, want %q", got[0].Name, "Read")
	}
}

func TestFilterNoiseDropsNoiseUserString(t *testing.T) {
	blocks := []NormalizedBlock{
		{Kind: BlockKindUser, Text: "Continue from where you left off.", SourceIndex: ptr(1)},
		{Kind: BlockKindUser, Text: "hello there", SourceIndex: ptr(2)},
	}
	got := filterNoise(blocks)
	if len(got) != 1 {
		t.Fatalf("filterNoise = %d blocks, want 1", len(got))
	}
	if got[0].Text != "hello there" {
		t.Errorf("got[0].Text = %q, want %q", got[0].Text, "hello there")
	}
}

func TestFilterNoiseStripsXMLWrapper(t *testing.T) {
	blocks := []NormalizedBlock{
		{
			Kind:        BlockKindUser,
			Text:        "some text <system-reminder>do this</system-reminder> more text",
			SourceIndex: ptr(1),
		},
	}
	got := filterNoise(blocks)
	if len(got) != 1 {
		t.Fatalf("filterNoise = %d blocks, want 1", len(got))
	}
	if got[0].Text != "some text  more text" {
		t.Errorf("got[0].Text = %q, want %q", got[0].Text, "some text  more text")
	}
}

func TestFilterNoiseDropsEmptyAfterClean(t *testing.T) {
	blocks := []NormalizedBlock{
		{Kind: BlockKindUser, Text: "<system-reminder>only wrapper</system-reminder>", SourceIndex: ptr(1)},
		{Kind: BlockKindUser, Text: "real content", SourceIndex: ptr(2)},
	}
	got := filterNoise(blocks)
	if len(got) != 1 {
		t.Fatalf("filterNoise = %d blocks, want 1", len(got))
	}
	if got[0].Text != "real content" {
		t.Errorf("got[0].Text = %q, want %q", got[0].Text, "real content")
	}
}

func TestFilterNoisePassesNonNoiseBlocks(t *testing.T) {
	blocks := []NormalizedBlock{
		{Kind: BlockKindUser, Text: "hello", SourceIndex: ptr(1)},
		{Kind: BlockKindAssistant, Text: "response", SourceIndex: ptr(2)},
		{Kind: BlockKindBash, Command: "ls", SourceIndex: ptr(3)},
	}
	got := filterNoise(blocks)
	if len(got) != 3 {
		t.Fatalf("filterNoise = %d blocks, want 3", len(got))
	}
}

func TestFilterNoiseNoiseToolsList(t *testing.T) {
	want := []string{"TodoWrite", "TodoRead", "ToolSearch", "WebSearch", "AskUser", "ExitSpecMode", "GenerateDroid"}
	if len(NOISE_TOOLS) != len(want) {
		t.Fatalf("NOISE_TOOLS = %d items, want %d", len(NOISE_TOOLS), len(want))
	}
	for i := range want {
		if NOISE_TOOLS[i] != want[i] {
			t.Errorf("NOISE_TOOLS[%d] = %q, want %q", i, NOISE_TOOLS[i], want[i])
		}
	}
}

func TestFilterNoiseNoiseStringsList(t *testing.T) {
	want := []string{"Continue from where you left off.", "No response requested.", "IMPORTANT: TodoWrite was not called yet."}
	if len(NOISE_STRINGS) != len(want) {
		t.Fatalf("NOISE_STRINGS = %d items, want %d", len(NOISE_STRINGS), len(want))
	}
	for i := range want {
		if NOISE_STRINGS[i] != want[i] {
			t.Errorf("NOISE_STRINGS[%d] = %q, want %q", i, NOISE_STRINGS[i], want[i])
		}
	}
}

func TestFilterNoiseStripsEachXMLWrapper(t *testing.T) {
	cases := []struct {
		input    string
		contains string
	}{
		{"text <system-reminder>hello</system-reminder> more", "text  more"},
		{"<ide_opened_file>data</ide_opened_file> only", "only"},
		{"<command-message>cmd</command-message>", ""},
		{"<context-window-usage>usage</context-window-usage> done", "done"},
	}
	for _, c := range cases {
		got := cleanUserText(c.input)
		if got != c.contains {
			t.Errorf("cleanUserText(%q) = %q, want %q", c.input, got, c.contains)
		}
	}
}

func TestFilterNoiseNonXMLTagPassesThrough(t *testing.T) {
	got := cleanUserText("<other>data</other>")
	if got != "<other>data</other>" {
		t.Errorf("cleanUserText(non-XML) = %q, want unchanged", got)
	}
}

func TestFilterNoiseEmptyInput(t *testing.T) {
	got := filterNoise(nil)
	if len(got) != 0 {
		t.Errorf("filterNoise(nil) = %d blocks, want 0", len(got))
	}
}

func TestFilterNoiseToolCallNotInNoiseList(t *testing.T) {
	blocks := []NormalizedBlock{
		{Kind: BlockKindToolCall, Name: "Edit", SourceIndex: ptr(1)},
	}
	got := filterNoise(blocks)
	if len(got) != 1 {
		t.Fatalf("filterNoise = %d blocks, want 1", len(got))
	}
	if got[0].Name != "Edit" {
		t.Errorf("got[0].Name = %q, want %q", got[0].Name, "Edit")
	}
}

func ptr(i int) *int {
	return &i
}
