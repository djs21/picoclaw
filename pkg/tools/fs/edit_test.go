package fstools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestEditTool_EditFile_Success verifies successful file editing
func TestEditTool_EditFile_Success(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("Hello World\nThis is a test"), 0o644)

	tool := NewEditFileTool(tmpDir, true)
	ctx := context.Background()
	args := map[string]any{
		"path":     testFile,
		"old_text": "World",
		"new_text": "Universe",
	}

	result := tool.Execute(ctx, args)

	// Success should not be an error
	if result.IsError {
		t.Errorf("Expected success, got IsError=true: %s", result.ForLLM)
	}

	// Successful edits should surface a diff to the user.
	if result.Silent {
		t.Errorf("Expected Silent=false for EditFile, got true")
	}

	if result.ForUser == "" {
		t.Fatal("Expected ForUser to contain the diff preview")
	}

	if result.ForLLM == result.ForUser {
		t.Fatalf("Expected ForLLM to be a compact summary, got identical outputs %q", result.ForLLM)
	}
	if result.ForLLM != fmt.Sprintf("File edited: %s", testFile) {
		t.Fatalf("Expected compact ForLLM summary, got %q", result.ForLLM)
	}

	diffPath := strings.TrimLeft(filepath.ToSlash(testFile), "/")
	for _, want := range []string{
		fmt.Sprintf("File edited: %s", testFile),
		"```diff",
		"--- a/" + diffPath,
		"+++ b/" + diffPath,
		"-Hello World",
		"+Hello Universe",
	} {
		if !strings.Contains(result.ForUser, want) {
			t.Fatalf("Expected edit diff to contain %q, got:\n%s", want, result.ForUser)
		}
	}

	// Verify file was actually edited
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read edited file: %v", err)
	}
	contentStr := string(content)
	if !strings.Contains(contentStr, "Hello Universe") {
		t.Errorf("Expected file to contain 'Hello Universe', got: %s", contentStr)
	}
	if strings.Contains(contentStr, "Hello World") {
		t.Errorf("Expected 'Hello World' to be replaced, got: %s", contentStr)
	}
}

// TestEditTool_EditFile_NotFound verifies error handling for non-existent file
func TestEditTool_EditFile_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "nonexistent.txt")

	tool := NewEditFileTool(tmpDir, true)
	ctx := context.Background()
	args := map[string]any{
		"path":     testFile,
		"old_text": "old",
		"new_text": "new",
	}

	result := tool.Execute(ctx, args)

	// Should return error result
	if !result.IsError {
		t.Errorf("Expected error for non-existent file")
	}

	// Should mention file not found
	if !strings.Contains(result.ForLLM, "not found") && !strings.Contains(result.ForUser, "not found") {
		t.Errorf("Expected 'file not found' message, got ForLLM: %s", result.ForLLM)
	}
}

// TestEditTool_EditFile_OldTextNotFound verifies error when old_text doesn't exist
func TestEditTool_EditFile_OldTextNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("Hello World"), 0o644)

	tool := NewEditFileTool(tmpDir, true)
	ctx := context.Background()
	args := map[string]any{
		"path":     testFile,
		"old_text": "Goodbye",
		"new_text": "Hello",
	}

	result := tool.Execute(ctx, args)

	// Should return error result
	if !result.IsError {
		t.Errorf("Expected error when old_text not found")
	}

	// Should mention old_text not found
	if !strings.Contains(result.ForLLM, "not found") && !strings.Contains(result.ForUser, "not found") {
		t.Errorf("Expected 'not found' message, got ForLLM: %s", result.ForLLM)
	}
}

// TestEditTool_EditFile_MultipleMatches verifies error when old_text appears multiple times
func TestEditTool_EditFile_MultipleMatches(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("test test test"), 0o644)

	tool := NewEditFileTool(tmpDir, true)
	ctx := context.Background()
	args := map[string]any{
		"path":     testFile,
		"old_text": "test",
		"new_text": "done",
	}

	result := tool.Execute(ctx, args)

	// Should return error result
	if !result.IsError {
		t.Errorf("Expected error when old_text appears multiple times")
	}

	// Should mention multiple occurrences
	if !strings.Contains(result.ForLLM, "times") && !strings.Contains(result.ForUser, "times") {
		t.Errorf("Expected 'multiple times' message, got ForLLM: %s", result.ForLLM)
	}
}

// TestEditTool_EditFile_OutsideAllowedDir verifies error when path is outside allowed directory
func TestEditTool_EditFile_OutsideAllowedDir(t *testing.T) {
	tmpDir := t.TempDir()
	otherDir := t.TempDir()
	testFile := filepath.Join(otherDir, "test.txt")
	os.WriteFile(testFile, []byte("content"), 0o644)

	tool := NewEditFileTool(tmpDir, true) // Restrict to tmpDir
	ctx := context.Background()
	args := map[string]any{
		"path":     testFile,
		"old_text": "content",
		"new_text": "new",
	}

	result := tool.Execute(ctx, args)

	// Should return error result
	assert.True(t, result.IsError, "Expected error when path is outside allowed directory")

	// Should mention outside allowed directory
	// Note: ErrorResult only sets ForLLM by default, so ForUser might be empty.
	// We check ForLLM as it's the primary error channel.
	assert.True(
		t,
		strings.Contains(result.ForLLM, "outside") || strings.Contains(result.ForLLM, "access denied") ||
			strings.Contains(result.ForLLM, "escapes"),
		"Expected 'outside allowed' or 'access denied' message, got ForLLM: %s",
		result.ForLLM,
	)
}

// TestEditTool_EditFile_MissingPath verifies error handling for missing path
func TestEditTool_EditFile_MissingPath(t *testing.T) {
	tool := NewEditFileTool("", false)
	ctx := context.Background()
	args := map[string]any{
		"old_text": "old",
		"new_text": "new",
	}

	result := tool.Execute(ctx, args)

	// Should return error result
	if !result.IsError {
		t.Errorf("Expected error when path is missing")
	}
}

// TestEditTool_EditFile_OmittedOldText_OverwritesOrCreates verifies that omitting old_text
// is allowed and performs a full file write/create.
func TestEditTool_EditFile_OmittedOldText_OverwritesOrCreates(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "omitted_old.txt")
	tool := NewEditFileTool(tmpDir, true)
	ctx := context.Background()
	args := map[string]any{
		"path":     testFile,
		"new_text": "created via omitted old_text",
	}

	result := tool.Execute(ctx, args)
	if result.IsError {
		t.Errorf("Expected success when old_text is omitted, got error: %s", result.ForLLM)
	}
	data, err := os.ReadFile(testFile)
	assert.NoError(t, err)
	assert.Equal(t, "created via omitted old_text", string(data))
}

// TestEditTool_EditFile_MissingNewText verifies error handling for missing new_text
func TestEditTool_EditFile_MissingNewText(t *testing.T) {
	tool := NewEditFileTool("", false)
	ctx := context.Background()
	args := map[string]any{
		"path":     "/tmp/test.txt",
		"old_text": "old",
	}

	result := tool.Execute(ctx, args)

	// Should return error result
	if !result.IsError {
		t.Errorf("Expected error when new_text is missing")
	}
}


// TestReplaceEditContent verifies the helper function replaceEditContent
func TestReplaceEditContent(t *testing.T) {
	tests := []struct {
		name        string
		content     []byte
		oldText     string
		newText     string
		expected    []byte
		expectError bool
	}{
		{
			name:        "successful replacement",
			content:     []byte("hello world"),
			oldText:     "world",
			newText:     "universe",
			expected:    []byte("hello universe"),
			expectError: false,
		},
		{
			name:        "old text not found",
			content:     []byte("hello world"),
			oldText:     "golang",
			newText:     "rust",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "multiple matches found",
			content:     []byte("test text test"),
			oldText:     "test",
			newText:     "done",
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := replaceEditContent(tt.content, tt.oldText, tt.newText)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}


// TestEditFileTool_Restricted_InPlaceEdit verifies that EditFileTool in restricted mode
// correctly edits a file using the single-open editFileInRoot path.
func TestEditFileTool_Restricted_InPlaceEdit(t *testing.T) {
	workspace := t.TempDir()
	testFile := "edit_target.txt"
	err := os.WriteFile(filepath.Join(workspace, testFile), []byte("Hello World"), 0o644)
	assert.NoError(t, err)

	tool := NewEditFileTool(workspace, true)
	ctx := context.Background()
	args := map[string]any{
		"path":     testFile,
		"old_text": "World",
		"new_text": "Go",
	}

	result := tool.Execute(ctx, args)
	assert.False(t, result.IsError, "Expected success, got: %s", result.ForLLM)
	assert.False(t, result.Silent)
	assert.Equal(t, "File edited: edit_target.txt", result.ForLLM)
	assert.Contains(t, result.ForUser, "```diff")
	assert.Contains(t, result.ForUser, "--- a/edit_target.txt")
	assert.Contains(t, result.ForUser, "+++ b/edit_target.txt")
	assert.Contains(t, result.ForUser, "-Hello World")
	assert.Contains(t, result.ForUser, "+Hello Go")

	data, err := os.ReadFile(filepath.Join(workspace, testFile))
	assert.NoError(t, err)
	assert.Equal(t, "Hello Go", string(data))
}

// TestEditFileTool_Restricted_FileNotFound verifies that editFileInRoot returns a proper
// error message when the target file does not exist and create_if_missing is false.
func TestEditFileTool_Restricted_FileNotFound(t *testing.T) {
	workspace := t.TempDir()
	tool := NewEditFileTool(workspace, true)
	ctx := context.Background()
	args := map[string]any{
		"path":              "no_such_file.txt",
		"old_text":          "old",
		"new_text":          "new",
		"create_if_missing": false,
	}

	result := tool.Execute(ctx, args)
	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "not found")
}

// TestEditFileTool_CreateIfMissing_NewFile verifies that EditFileTool creates a new file
// when the target does not exist and create_if_missing is true.
func TestEditFileTool_CreateIfMissing_NewFile(t *testing.T) {
	workspace := t.TempDir()
	tool := NewEditFileTool(workspace, true)
	ctx := context.Background()
	args := map[string]any{
		"path":              "created.txt",
		"new_text":          "Brand new file content\nLine 2",
		"create_if_missing": true,
	}

	result := tool.Execute(ctx, args)
	assert.False(t, result.IsError, "Expected success, got: %s", result.ForLLM)
	assert.Contains(t, result.ForLLM, "created.txt")

	data, err := os.ReadFile(filepath.Join(workspace, "created.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "Brand new file content\nLine 2", string(data))
}

// TestEditFileTool_OverwriteWholeFile verifies that passing empty old_text or omitting old_text
// replaces the entire file content.
func TestEditFileTool_OverwriteWholeFile(t *testing.T) {
	workspace := t.TempDir()
	testFile := "existing.txt"
	err := os.WriteFile(filepath.Join(workspace, testFile), []byte("Old full content"), 0o644)
	assert.NoError(t, err)

	tool := NewEditFileTool(workspace, true)
	ctx := context.Background()
	args := map[string]any{
		"path":     testFile,
		"old_text": "",
		"new_text": "Replaced entire content",
	}

	result := tool.Execute(ctx, args)
	assert.False(t, result.IsError, "Expected success, got: %s", result.ForLLM)

	data, err := os.ReadFile(filepath.Join(workspace, testFile))
	assert.NoError(t, err)
	assert.Equal(t, "Replaced entire content", string(data))
}

