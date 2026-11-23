package vcc

import "strings"

// normalize converts raw messages into NormalizedBlock[] (algo spec §3.1,
// src/core/normalize.ts:70). sourceIndices (optional, variadic) supplies the
// session-global index per message position; omitted or missing → nil SourceIndex.
func normalize(messages []map[string]any, sourceIndices ...int) []NormalizedBlock {
	var blocks []NormalizedBlock
	for i, msg := range messages {
		role, _ := msg["role"].(string)
		var srcIndex *int
		if i < len(sourceIndices) {
			v := sourceIndices[i]
			srcIndex = &v
		}
		blocks = append(blocks, normalizeOne(role, msg, srcIndex)...)
	}
	return blocks
}

// normalizeOne converts a single message into zero or more NormalizedBlock
// (algo spec §3.1, src/core/normalize.ts:6).
func normalizeOne(role string, msg map[string]any, sourceIndex *int) []NormalizedBlock {
	switch role {
	case "user":
		return normalizeOneUser(msg, sourceIndex)
	case "assistant":
		return normalizeOneAssistant(msg, sourceIndex)
	case "bashExecution":
		return normalizeOneBash(msg, sourceIndex)
	case "toolResult":
		return normalizeOneToolResult(msg, sourceIndex)
	default:
		return nil
	}
}

func normalizeOneUser(msg map[string]any, sourceIndex *int) []NormalizedBlock {
	content := msg["content"]

	var blocks []NormalizedBlock
	var imageBlocks []NormalizedBlock
	var textParts []string

	if parts, ok := content.([]any); ok {
		for _, p := range parts {
			m, ok := p.(map[string]any)
			if !ok {
				continue
			}
			// Image part → collect [image: mimeType] block
			if img, ok := m["image"].(map[string]any); ok {
				if mime, ok := img["mimeType"].(string); ok {
					imageBlocks = append(imageBlocks, NormalizedBlock{
						Kind:        BlockKindUser,
						Text:        "[image: " + mime + "]",
						SourceIndex: sourceIndex,
					})
				}
			}
			// Text part → collect for later
			if t, ok := m["text"].(string); ok {
				textParts = append(textParts, t)
			}
		}
		if len(textParts) > 0 {
			text := sanitize(strings.Join(textParts, "\n"))
			if text != "" {
				blocks = append(blocks, NormalizedBlock{
					Kind:        BlockKindUser,
					Text:        text,
					SourceIndex: sourceIndex,
				})
			}
		}
		// Append image blocks after text block
		blocks = append(blocks, imageBlocks...)
		// Empty-text fallback: if no blocks at all, push a single empty-text user block.
		if len(blocks) == 0 {
			blocks = append(blocks, NormalizedBlock{
				Kind:        BlockKindUser,
				Text:        "",
				SourceIndex: sourceIndex,
			})
		}
		return blocks
	}

	// Plain string content
	text := sanitize(textOf(content))
	if text != "" {
		blocks = append(blocks, NormalizedBlock{
			Kind:        BlockKindUser,
			Text:        text,
			SourceIndex: sourceIndex,
		})
	}

	// Empty-text fallback: if no blocks at all, push a single empty-text user block.
	if len(blocks) == 0 {
		blocks = append(blocks, NormalizedBlock{
			Kind:        BlockKindUser,
			Text:        "",
			SourceIndex: sourceIndex,
		})
	}

	return blocks
}

func normalizeOneAssistant(msg map[string]any, sourceIndex *int) []NormalizedBlock {
	content := msg["content"]

	switch c := content.(type) {
	case string:
		if c == "" {
			return nil
		}
		return []NormalizedBlock{{
			Kind:        BlockKindAssistant,
			Text:        sanitize(c),
			SourceIndex: sourceIndex,
		}}
	case []any:
		var blocks []NormalizedBlock
		for _, p := range c {
			m, ok := p.(map[string]any)
			if !ok {
				continue
			}
			if t, ok := m["text"].(string); ok && t != "" {
				blocks = append(blocks, NormalizedBlock{
					Kind:        BlockKindAssistant,
					Text:        sanitize(t),
					SourceIndex: sourceIndex,
				})
			}
			if _, ok := m["toolCall"]; ok {
				name, _ := m["name"].(string)
				if name == "" {
					continue
				}
				args, _ := m["args"].(map[string]any)
				blocks = append(blocks, NormalizedBlock{
					Kind:        BlockKindToolCall,
					Name:        name,
					Args:        args,
					SourceIndex: sourceIndex,
				})
			}
		}
		return blocks
	default:
		return nil
	}
}

func normalizeOneBash(msg map[string]any, sourceIndex *int) []NormalizedBlock {
	command, _ := msg["command"].(string)
	output, _ := msg["output"].(string)
	var exitCode *int
	switch ec := msg["exitCode"].(type) {
	case float64:
		v := int(ec)
		exitCode = &v
	case int:
		v := ec
		exitCode = &v
	}
	return []NormalizedBlock{{
		Kind:        BlockKindBash,
		Command:     command,
		Output:      output,
		ExitCode:    exitCode,
		SourceIndex: sourceIndex,
	}}
}

func normalizeOneToolResult(msg map[string]any, sourceIndex *int) []NormalizedBlock {
	name, _ := msg["toolName"].(string)
	text := sanitize(textOf(msg["content"]))
	return []NormalizedBlock{{
		Kind:        BlockKindToolResult,
		Name:        name,
		Text:        text,
		SourceIndex: sourceIndex,
	}}
}
