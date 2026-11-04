package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockTestTool struct {
	name string
	desc string
}

func (m *mockTestTool) Name() string { return m.name }
func (m *mockTestTool) Description() string { return m.desc }
func (m *mockTestTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"param1": map[string]any{"type": "string"},
		},
	}
}
func (m *mockTestTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	return NewToolResult("ok")
}

func TestToolRegistry_GetEntry_And_AutoPromote(t *testing.T) {
	r := NewToolRegistry()
	tool := &mockTestTool{name: "deferred_tool", desc: "A deferred tool"}
	r.RegisterHidden(tool)

	// Initially TTL is 0, so Get returns false
	_, ok := r.Get("deferred_tool")
	assert.False(t, ok)

	// GetEntry retrieves the raw entry
	entry, exists := r.GetEntry("deferred_tool")
	require.True(t, exists)
	assert.False(t, entry.IsCore)
	assert.Equal(t, 0, entry.TTL)

	// AutoPromote sets TTL if entry exists
	promoted := r.AutoPromote("deferred_tool", 3)
	assert.True(t, promoted)

	// Now Get returns true because TTL > 0
	tInstance, ok := r.Get("deferred_tool")
	assert.True(t, ok)
	assert.Equal(t, "deferred_tool", tInstance.Name())

	entry, _ = r.GetEntry("deferred_tool")
	assert.Equal(t, 3, entry.TTL)
}

func TestToolRegistry_SnapshotManifest(t *testing.T) {
	r := NewToolRegistry()
	r.Register(&mockTestTool{name: "core_tool", desc: "Core tool description"})
	r.RegisterHidden(&mockTestTool{name: "web_search", desc: "Search web pages"})
	r.RegisterHidden(&mockTestTool{name: "i2c", desc: "Interact with I2C bus"})

	manifest := r.SnapshotManifest()
	require.Len(t, manifest, 2) // Only hidden/deferred tools

	// Deterministically sorted by name
	assert.Equal(t, "i2c", manifest[0].Name)
	assert.Equal(t, "Interact with I2C bus", manifest[0].Description)
	assert.Equal(t, "web_search", manifest[1].Name)
	assert.Equal(t, "Search web pages", manifest[1].Description)
}

func TestLoadTool_Execute(t *testing.T) {
	r := NewToolRegistry()
	r.RegisterHidden(&mockTestTool{name: "i2c", desc: "Interact with I2C bus"})
	r.RegisterHidden(&mockTestTool{name: "spi", desc: "Interact with SPI bus"})

	loadTool := NewLoadTool(r, 4)
	assert.Equal(t, "load_tool", loadTool.Name())

	// Execute load_tool with tool names
	ctx := context.Background()
	res := loadTool.Execute(ctx, map[string]any{
		"tools": []any{"i2c", "spi"},
	})
	assert.False(t, res.IsError)
	assert.Contains(t, res.ContentForLLM(), "SUCCESS")
	assert.Contains(t, res.ContentForLLM(), "i2c")
	assert.Contains(t, res.ContentForLLM(), "spi")

	// Both tools should now be callable and in ToProviderDefs
	_, okI2C := r.Get("i2c")
	_, okSPI := r.Get("spi")
	assert.True(t, okI2C)
	assert.True(t, okSPI)

	defs := r.ToProviderDefs()
	names := make([]string, len(defs))
	for i, d := range defs {
		names[i] = d.Function.Name
	}
	assert.Contains(t, names, "i2c")
	assert.Contains(t, names, "spi")
}

func TestLoadTool_InvalidOrMissingTool(t *testing.T) {
	r := NewToolRegistry()
	loadTool := NewLoadTool(r, 3)

	ctx := context.Background()
	res := loadTool.Execute(ctx, map[string]any{
		"tools": []any{"non_existent_tool"},
	})
	assert.True(t, res.IsError)
	assert.Contains(t, res.ContentForLLM(), "not found")
}
