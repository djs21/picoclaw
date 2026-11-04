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
	cfg.Tools.LazyLoading.CoreTools = []string{"read_file", "write_file", "load_tool"}

	// Enable append_file and list_dir (which should now be deferred/hidden)
	cfg.Tools.ReadFile.Enabled = true
	cfg.Tools.WriteFile.Enabled = true
	cfg.Tools.AppendFile.Enabled = true
	cfg.Tools.ListDir.Enabled = true

	defaults := &cfg.Agents.Defaults
	provider := &mockDummyProvider{}

	instance := NewAgentInstance(nil, defaults, cfg, provider)
	require.NotNil(t, instance)

	// Core tool: read_file and write_file should be callable (IsCore = true)
	rEntry, rOk := instance.Tools.GetEntry("read_file")
	require.True(t, rOk)
	assert.True(t, rEntry.IsCore)

	wEntry, wOk := instance.Tools.GetEntry("write_file")
	require.True(t, wOk)
	assert.True(t, wEntry.IsCore)

	// load_tool should also be registered as Core tool
	loadEntry, loadOk := instance.Tools.GetEntry("load_tool")
	require.True(t, loadOk)
	assert.True(t, loadEntry.IsCore)

	// Deferred tools: append_file and list_dir should be registered as hidden (IsCore = false, TTL = 0)
	aEntry, aOk := instance.Tools.GetEntry("append_file")
	require.True(t, aOk)
	assert.False(t, aEntry.IsCore)
	assert.Equal(t, 0, aEntry.TTL)

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
			assert.Contains(t, p.Text, "append_file")
			assert.Contains(t, p.Text, "list_dir")
			break
		}
	}
	assert.True(t, foundManifest, "expected system prompt to contain deferred tool manifest")
}
