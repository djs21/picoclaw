package fstools

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"strings"
)

// EditFileTool edits a file by replacing old_text with new_text.
// The old_text must exist exactly in the file.
type EditFileTool struct {
	fs fileSystem
}

// NewEditFileTool creates a new EditFileTool with optional directory restriction.
func NewEditFileTool(workspace string, restrict bool, allowPaths ...[]*regexp.Regexp) *EditFileTool {
	var patterns []*regexp.Regexp
	if len(allowPaths) > 0 {
		patterns = allowPaths[0]
	}
	return &EditFileTool{fs: buildFs(workspace, restrict, patterns)}
}

func (t *EditFileTool) Name() string {
	return "edit_file"
}

func (t *EditFileTool) Description() string {
	return "Edit a file by replacing old_text with new_text, creating a new file, or replacing entire content. If old_text is omitted or empty, or if create_if_missing is true and the file does not exist, new_text will be written as the file's content. Standard JSON escaping applies: \\n for newline and \\\\n for literal backslash-n."
}

func (t *EditFileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The file path to edit or create",
			},
			"old_text": map[string]any{
				"type":        "string",
				"description": "The exact text to find and replace. Leave empty or omit if creating a new file or replacing entire file content. Standard JSON escaping applies: \\n for newline and \\\\n for literal backslash-n.",
			},
			"new_text": map[string]any{
				"type":        "string",
				"description": "The replacement text, or initial/full content for new or overwritten files. Standard JSON escaping applies: \\n for newline and \\\\n for literal backslash-n.",
			},
			"create_if_missing": map[string]any{
				"type":        "boolean",
				"description": "If true, creates the file if it does not already exist (defaults to true if old_text is empty).",
			},
		},
		"required": []string{"path", "new_text"},
	}
}

func (t *EditFileTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	path, ok := args["path"].(string)
	if !ok || strings.TrimSpace(path) == "" {
		return ErrorResult("path is required")
	}

	newText, ok := args["new_text"].(string)
	if !ok {
		return ErrorResult("new_text is required")
	}

	var oldText string
	if ot, hasOld := args["old_text"].(string); hasOld {
		oldText = ot
	}

	createIfMissing := oldText == ""
	if cm, ok := args["create_if_missing"].(bool); ok {
		createIfMissing = cm
	}

	beforeContent, afterContent, isNew, err := editOrCreateFile(t.fs, path, oldText, newText, createIfMissing)
	if err != nil {
		return ErrorResult(err.Error())
	}
	if isNew {
		return SilentResult(fmt.Sprintf("File created: %s", path))
	}
	return DiffResult(path, beforeContent, afterContent)
}


// editOrCreateFile reads the file via sysFs, performs the replacement, or creates/overwrites the file.
func editOrCreateFile(sysFs fileSystem, path, oldText, newText string, createIfMissing bool) ([]byte, []byte, bool, error) {
	content, err := sysFs.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) || strings.Contains(err.Error(), "no such file") || strings.Contains(err.Error(), "not found") {
			if createIfMissing {
				newBytes := []byte(newText)
				if writeErr := sysFs.WriteFile(path, newBytes); writeErr != nil {
					return nil, nil, false, writeErr
				}
				return nil, newBytes, true, nil
			}
			return nil, nil, false, fmt.Errorf("file not found: %s", path)
		}
		return nil, nil, false, err
	}

	if oldText == "" {
		// Replace entire content
		newBytes := []byte(newText)
		if writeErr := sysFs.WriteFile(path, newBytes); writeErr != nil {
			return nil, nil, false, writeErr
		}
		return content, newBytes, false, nil
	}

	newContent, err := replaceEditContent(content, oldText, newText)
	if err != nil {
		return nil, nil, false, err
	}

	if writeErr := sysFs.WriteFile(path, newContent); writeErr != nil {
		return nil, nil, false, writeErr
	}

	return content, newContent, false, nil
}

// editFile reads the file via sysFs, performs the replacement, and writes back.
// It uses a fileSystem interface, allowing the same logic for both restricted and unrestricted modes.
func editFile(sysFs fileSystem, path, oldText, newText string) ([]byte, []byte, error) {
	before, after, _, err := editOrCreateFile(sysFs, path, oldText, newText, false)
	return before, after, err
}

// appendFile reads the existing content (if any) via sysFs, appends new content, and writes back.
func appendFile(sysFs fileSystem, path, appendContent string) error {
	content, err := sysFs.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	newContent := append(content, []byte(appendContent)...)
	return sysFs.WriteFile(path, newContent)
}

// replaceEditContent handles the core logic of finding and replacing a single occurrence of oldText.
func replaceEditContent(content []byte, oldText, newText string) ([]byte, error) {
	contentStr := string(content)

	if !strings.Contains(contentStr, oldText) {
		return nil, fmt.Errorf("old_text not found in file. Make sure it matches exactly")
	}

	count := strings.Count(contentStr, oldText)
	if count > 1 {
		return nil, fmt.Errorf("old_text appears %d times. Please provide more context to make it unique", count)
	}

	newContent := strings.Replace(contentStr, oldText, newText, 1)
	return []byte(newContent), nil
}
