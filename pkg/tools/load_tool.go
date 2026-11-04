package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/logger"
)

const LoadToolName = "load_tool"

// LoadTool allows the LLM to dynamically load deferred tools into its context window on-demand.
type LoadTool struct {
	registry *ToolRegistry
	ttl      int
}

// NewLoadTool creates a new LoadTool instance.
func NewLoadTool(registry *ToolRegistry, ttl int) *LoadTool {
	if ttl <= 0 {
		ttl = 3
	}
	return &LoadTool{
		registry: registry,
		ttl:      ttl,
	}
}

func (t *LoadTool) Name() string {
	return LoadToolName
}

func (t *LoadTool) Description() string {
	return "Load one or more deferred tools on-demand to inspect their full parameters and make them available for direct calling in your subsequent turns."
}

func (t *LoadTool) PromptMetadata() PromptMetadata {
	return PromptMetadata{
		Layer:  ToolPromptLayerCapability,
		Slot:   ToolPromptSlotTooling,
		Source: ToolPromptSourceDiscovery,
	}
}

func (t *LoadTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"tools": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "string",
				},
				"description": "List of deferred tool names to activate and load into context",
			},
		},
		"required": []string{"tools"},
	}
}

func (t *LoadTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	rawTools, ok := args["tools"]
	if !ok {
		return ErrorResult("Missing required parameter: 'tools'")
	}

	var toolNames []string
	switch v := rawTools.(type) {
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				toolNames = append(toolNames, strings.TrimSpace(s))
			}
		}
	case []string:
		for _, s := range v {
			if strings.TrimSpace(s) != "" {
				toolNames = append(toolNames, strings.TrimSpace(s))
			}
		}
	case string:
		if strings.TrimSpace(v) != "" {
			toolNames = append(toolNames, strings.TrimSpace(v))
		}
	}

	if len(toolNames) == 0 {
		return ErrorResult("No valid tool names provided in 'tools' argument.")
	}

	var missing []string
	var loaded []string

	for _, name := range toolNames {
		if !t.registry.HasRegistered(name) {
			missing = append(missing, name)
			continue
		}
		t.registry.PromoteTools([]string{name}, t.ttl)
		loaded = append(loaded, name)
	}

	if len(missing) > 0 && len(loaded) == 0 {
		return ErrorResult(fmt.Sprintf("Tools not found: %s", strings.Join(missing, ", ")))
	}

	logger.InfoCF("tools", "Loaded tools on demand", map[string]any{
		"loaded":  loaded,
		"missing": missing,
		"ttl":     t.ttl,
	})

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("SUCCESS: Activated %d tool(s): %s (TTL: %d turns).\n", len(loaded), strings.Join(loaded, ", "), t.ttl))
	msg.WriteString("Their full parameter schemas are now unlocked. You can call them directly in your next response.")
	if len(missing) > 0 {
		msg.WriteString(fmt.Sprintf("\nWarning: Some requested tools were not found: %s", strings.Join(missing, ", ")))
	}

	return SilentResult(msg.String())
}
