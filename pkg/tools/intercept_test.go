package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolRegistry_ExecuteWithContext_AutoIntercept(t *testing.T) {
	r := NewToolRegistry()
	tool := &mockTestTool{name: "deferred_tool", desc: "A deferred tool"}
	r.RegisterHidden(tool)

	// Tool is registered as hidden with TTL = 0.
	_, ok := r.Get("deferred_tool")
	require.False(t, ok)

	// Enable auto-intercept with default TTL = 3
	r.SetAutoIntercept(true, 3)

	// Execute should succeed because auto-intercept promotes it
	res := r.ExecuteWithContext(context.Background(), "deferred_tool", map[string]any{}, "channel1", "chat1", nil)
	require.NotNil(t, res)
	assert.False(t, res.IsError)
	assert.Equal(t, "ok", res.ContentForLLM())

	// Now tool should have TTL = 3 and be callable
	tInstance, ok := r.Get("deferred_tool")
	assert.True(t, ok)
	assert.Equal(t, "deferred_tool", tInstance.Name())
	entry, _ := r.GetEntry("deferred_tool")
	assert.Equal(t, 3, entry.TTL)
}

func TestToolRegistry_ExecuteWithContext_AutoIntercept_Disabled(t *testing.T) {
	r := NewToolRegistry()
	tool := &mockTestTool{name: "deferred_tool", desc: "A deferred tool"}
	r.RegisterHidden(tool)

	// Auto-intercept disabled
	r.SetAutoIntercept(false, 3)

	res := r.ExecuteWithContext(context.Background(), "deferred_tool", map[string]any{}, "channel1", "chat1", nil)
	require.NotNil(t, res)
	assert.True(t, res.IsError)
	assert.Contains(t, res.ContentForLLM(), "not found")
}
