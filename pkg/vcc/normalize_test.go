package vcc

import (
	"testing"
)

func TestNormalizeUserMessage(t *testing.T) {
	msgs := []map[string]interface{}{
		{"role": "user", "content": "hello"},
	}
	blocks := normalize(msgs)
	if len(blocks) != 1 {
		t.Fatalf("normalize(user) = %d blocks, want 1", len(blocks))
	}
	if blocks[0].Kind != BlockKindUser {
		t.Errorf("block[0].Kind = %q, want %q", blocks[0].Kind, BlockKindUser)
	}
	if blocks[0].Text != "hello" {
		t.Errorf("block[0].Text = %q, want %q", blocks[0].Text, "hello")
	}
}

func TestNormalizeUserMessageWithSourceIndex(t *testing.T) {
	msgs := []map[string]interface{}{
		{"role": "user", "content": "hello"},
	}
	blocks := normalize(msgs, 42)
	if len(blocks) != 1 {
		t.Fatalf("normalize = %d blocks, want 1", len(blocks))
	}
	if blocks[0].SourceIndex == nil || *blocks[0].SourceIndex != 42 {
		t.Errorf("block[0].SourceIndex = %v, want 42", blocks[0].SourceIndex)
	}
}

func TestNormalizeNoSourceIndex(t *testing.T) {
	msgs := []map[string]interface{}{
		{"role": "user", "content": "hello"},
	}
	blocks := normalize(msgs)
	if len(blocks) != 1 {
		t.Fatalf("normalize = %d blocks, want 1", len(blocks))
	}
	if blocks[0].SourceIndex != nil {
		t.Errorf("block[0].SourceIndex = %v, want nil", blocks[0].SourceIndex)
	}
}

func TestNormalizeAssistantString(t *testing.T) {
	msgs := []map[string]interface{}{
		{"role": "assistant", "content": "response"},
	}
	blocks := normalize(msgs)
	if len(blocks) != 1 {
		t.Fatalf("normalize(assistant) = %d blocks, want 1", len(blocks))
	}
	if blocks[0].Kind != BlockKindAssistant {
		t.Errorf("block[0].Kind = %q, want %q", blocks[0].Kind, BlockKindAssistant)
	}
	if blocks[0].Text != "response" {
		t.Errorf("block[0].Text = %q, want %q", blocks[0].Text, "response")
	}
}

func TestNormalizeAssistantArray(t *testing.T) {
	msgs := []map[string]interface{}{
		{"role": "assistant", "content": []interface{}{
			map[string]interface{}{"text": "part1"},
			map[string]interface{}{"text": "part2"},
		}},
	}
	blocks := normalize(msgs)
	if len(blocks) != 2 {
		t.Fatalf("normalize(assistant array) = %d blocks, want 2", len(blocks))
	}
	if blocks[0].Text != "part1" {
		t.Errorf("block[0].Text = %q, want %q", blocks[0].Text, "part1")
	}
	if blocks[1].Text != "part2" {
		t.Errorf("block[1].Text = %q, want %q", blocks[1].Text, "part2")
	}
}

func TestNormalizeBashExecution(t *testing.T) {
	msgs := []map[string]interface{}{
		{"role": "bashExecution", "command": "ls", "output": "file1\nfile2", "exitCode": 0},
	}
	blocks := normalize(msgs)
	if len(blocks) != 1 {
		t.Fatalf("normalize(bash) = %d blocks, want 1", len(blocks))
	}
	if blocks[0].Kind != BlockKindBash {
		t.Errorf("block[0].Kind = %q, want %q", blocks[0].Kind, BlockKindBash)
	}
	if blocks[0].Command != "ls" {
		t.Errorf("block[0].Command = %q, want %q", blocks[0].Command, "ls")
	}
	if blocks[0].Output != "file1\nfile2" {
		t.Errorf("block[0].Output = %q, want %q", blocks[0].Output, "file1\nfile2")
	}
	if blocks[0].ExitCode == nil || *blocks[0].ExitCode != 0 {
		t.Errorf("block[0].ExitCode = %v, want 0", blocks[0].ExitCode)
	}
}

func TestNormalizeToolResult(t *testing.T) {
	msgs := []map[string]interface{}{
		{"role": "toolResult", "toolName": "Read", "content": "file content"},
	}
	blocks := normalize(msgs)
	if len(blocks) != 1 {
		t.Fatalf("normalize(tool_result) = %d blocks, want 1", len(blocks))
	}
	if blocks[0].Kind != BlockKindToolResult {
		t.Errorf("block[0].Kind = %q, want %q", blocks[0].Kind, BlockKindToolResult)
	}
	if blocks[0].Name != "Read" {
		t.Errorf("block[0].Name = %q, want %q", blocks[0].Name, "Read")
	}
	if blocks[0].Text != "file content" {
		t.Errorf("block[0].Text = %q, want %q", blocks[0].Text, "file content")
	}
}

func TestNormalizeUnknownRole(t *testing.T) {
	msgs := []map[string]interface{}{
		{"role": "system", "content": "ignored"},
	}
	blocks := normalize(msgs)
	if len(blocks) != 0 {
		t.Errorf("normalize(unknown) = %d blocks, want 0", len(blocks))
	}
}

func TestNormalizeEmptyTextUserFallback(t *testing.T) {
	msgs := []map[string]interface{}{
		{"role": "user", "content": ""},
	}
	blocks := normalize(msgs)
	if len(blocks) != 1 {
		t.Fatalf("normalize(empty user) = %d blocks, want 1", len(blocks))
	}
	if blocks[0].Kind != BlockKindUser {
		t.Errorf("block[0].Kind = %q, want %q", blocks[0].Kind, BlockKindUser)
	}
	if blocks[0].Text != "" {
		t.Errorf("block[0].Text = %q, want %q", blocks[0].Text, "")
	}
}

func TestNormalizeUserImageBlock(t *testing.T) {
	msgs := []map[string]interface{}{
		{"role": "user", "content": []interface{}{
			map[string]interface{}{"text": "see image"},
			map[string]interface{}{"image": map[string]interface{}{"mimeType": "image/png"}},
		}},
	}
	blocks := normalize(msgs)
	if len(blocks) != 2 {
		t.Fatalf("normalize(user with image) = %d blocks, want 2", len(blocks))
	}
	if blocks[0].Kind != BlockKindUser {
		t.Errorf("block[0].Kind = %q, want %q", blocks[0].Kind, BlockKindUser)
	}
	if blocks[0].Text != "see image" {
		t.Errorf("block[0].Text = %q, want %q", blocks[0].Text, "see image")
	}
	if blocks[1].Kind != BlockKindUser {
		t.Errorf("block[1].Kind = %q, want %q", blocks[1].Kind, BlockKindUser)
	}
	if blocks[1].Text != "[image: image/png]" {
		t.Errorf("block[1].Text = %q, want %q", blocks[1].Text, "[image: image/png]")
	}
}

func TestNormalizeMultipleMessages(t *testing.T) {
	msgs := []map[string]interface{}{
		{"role": "user", "content": "first"},
		{"role": "assistant", "content": "second"},
		{"role": "user", "content": "third"},
	}
	blocks := normalize(msgs, 1, 2, 3)
	if len(blocks) != 3 {
		t.Fatalf("normalize(multi) = %d blocks, want 3", len(blocks))
	}
	if blocks[0].Text != "first" || *blocks[0].SourceIndex != 1 {
		t.Errorf("block[0] = %+v, want text=first sourceIndex=1", blocks[0])
	}
	if blocks[1].Text != "second" || *blocks[1].SourceIndex != 2 {
		t.Errorf("block[1] = %+v, want text=second sourceIndex=2", blocks[1])
	}
	if blocks[2].Text != "third" || *blocks[2].SourceIndex != 3 {
		t.Errorf("block[2] = %+v, want text=third sourceIndex=3", blocks[2])
	}
}
