package agent

import (
	"context"
	"testing"

	"github.com/sipeed/picoclaw/pkg/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockManifestTool struct {
	name string
	desc string
}

func (m *mockManifestTool) Name() string { return m.name }
func (m *mockManifestTool) Description() string { return m.desc }
func (m *mockManifestTool) Parameters() map[string]any {
	return map[string]any{"type": "object"}
}
func (m *mockManifestTool) Execute(ctx context.Context, args map[string]any) *tools.ToolResult {
	return tools.NewToolResult("ok")
}

func TestContextBuilder_WithLazyToolManifest(t *testing.T) {
	registry := tools.NewToolRegistry()
	registry.RegisterHidden(&mockManifestTool{name: "web_search", desc: "Search the web for up-to-date information"})
	registry.RegisterHidden(&mockManifestTool{name: "spi", desc: "Linux SPI bus communication"})

	cb := NewContextBuilder(t.TempDir()).WithLazyToolManifest(func() []tools.ToolManifestDoc {
		return registry.SnapshotManifest()
	})

	messages := cb.BuildMessagesFromPrompt(PromptBuildRequest{CurrentMessage: "hello"})
	require.NotEmpty(t, messages)
	system := messages[0]

	var manifestFound bool
	var manifestContent string
	for _, part := range system.SystemParts {
		if part.PromptSource == string(PromptSourceToolManifest) {
			manifestFound = true
			manifestContent = part.Text
			break
		}
	}

	require.True(t, manifestFound, "expected PromptSourceToolManifest in system.SystemParts")
	assert.Contains(t, manifestContent, "Available Deferred Tools")
	assert.Contains(t, manifestContent, "load_tool")
	assert.Contains(t, manifestContent, "- `spi`: Linux SPI bus communication")
	assert.Contains(t, manifestContent, "- `web_search`: Search the web for up-to-date information")
}

func TestContextBuilder_WithLazyToolManifest_EmptyWhenNoTools(t *testing.T) {
	cb := NewContextBuilder(t.TempDir()).WithLazyToolManifest(func() []tools.ToolManifestDoc {
		return nil
	})

	messages := cb.BuildMessagesFromPrompt(PromptBuildRequest{CurrentMessage: "hello"})
	require.NotEmpty(t, messages)
	system := messages[0]

	for _, part := range system.SystemParts {
		assert.NotEqual(t, string(PromptSourceToolManifest), part.PromptSource)
	}
}
