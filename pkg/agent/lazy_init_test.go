package agent

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockDummyProvider struct {
	providers.LLMProvider
}

func (m *mockDummyProvider) Name() string { return "dummy" }

func TestNewAgentInstance_LazyLoading_RegistersCoreAndDeferred(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tools.LazyLoading.Enabled = true
	cfg.Tools.LazyLoading.Mode = "hybrid"
	cfg.Tools.LazyLoading.DefaultTTL = 4
	cfg.Tools.LazyLoading.AutoIntercept = true
	// Set CoreTools explicitly
	cfg.Tools.LazyLoading.CoreTools = []string{"read_file", "edit_file", "load_tool"}

	// Enable edit_file and list_dir (which should now be deferred/hidden)
	cfg.Tools.ReadFile.Enabled = true
	cfg.Tools.EditFile.Enabled = true
	cfg.Tools.ListDir.Enabled = true

	defaults := &cfg.Agents.Defaults
	provider := &mockDummyProvider{}

	instance := NewAgentInstance(nil, defaults, cfg, provider)
	require.NotNil(t, instance)

	// Core tool: read_file and edit_file should be callable (IsCore = true)
	rEntry, rOk := instance.Tools.GetEntry("read_file")
	require.True(t, rOk)
	assert.True(t, rEntry.IsCore)

	eEntry, eOk := instance.Tools.GetEntry("edit_file")
	require.True(t, eOk)
	assert.True(t, eEntry.IsCore)

	// load_tool should also be registered as Core tool
	loadEntry, loadOk := instance.Tools.GetEntry("load_tool")
	require.True(t, loadOk)
	assert.True(t, loadEntry.IsCore)

	// Deferred tools: list_dir should be registered as hidden (IsCore = false, TTL = 0)
	lEntry, lOk := instance.Tools.GetEntry("list_dir")
	require.True(t, lOk)
	assert.False(t, lEntry.IsCore)
	assert.Equal(t, 0, lEntry.TTL)

	// ContextBuilder should have the manifest contributor attached
	msgs := instance.ContextBuilder.BuildMessagesFromPrompt(PromptBuildRequest{CurrentMessage: "hi"})
	require.NotEmpty(t, msgs)
	var foundManifest bool
	for _, p := range msgs[0].SystemParts {
		if p.PromptSource == string(PromptSourceToolManifest) {
			foundManifest = true
			assert.Contains(t, p.Text, "list_dir")
			break
		}
	}
	assert.True(t, foundManifest, "expected system prompt to contain deferred tool manifest")
}

func TestNewAgentInstance_LazyLoading_UsesDefaultCoreToolsWhenUnset(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tools.LazyLoading.Enabled = true
	cfg.Tools.LazyLoading.CoreTools = nil
	cfg.Tools.ReadFile.Enabled = true
	cfg.Tools.EditFile.Enabled = true
	cfg.Tools.ListDir.Enabled = true

	instance := NewAgentInstance(nil, &cfg.Agents.Defaults, cfg, &mockDummyProvider{})
	require.NotNil(t, instance)

	for _, core := range []string{"read_file", "edit_file"} {
		entry, ok := instance.Tools.GetEntry(core)
		require.True(t, ok, "%s should be registered", core)
		assert.True(t, entry.IsCore, "%s should be core by default", core)
	}

	listEntry, ok := instance.Tools.GetEntry("list_dir")
	require.True(t, ok)
	assert.False(t, listEntry.IsCore, "list_dir should stay deferred")

	// write_file/append_file no longer exist as separate tools.
	for _, legacy := range []string{"write_file", "append_file"} {
		_, exists := instance.Tools.GetEntry(legacy)
		assert.False(t, exists, "%s must not be registered", legacy)
	}
}
