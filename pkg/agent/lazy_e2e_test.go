package agent_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type e2eMockProvider struct {
	providers.LLMProvider
}

func (m *e2eMockProvider) Name() string { return "e2e_mock" }

func TestLazyLoading_E2E_TokenEfficiencyAndAutoPromotion(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Setup eager configuration (All tools enabled, LazyLoading disabled)
	cfgEager := config.DefaultConfig()
	cfgEager.Agents.Defaults.Workspace = tmpDir
	cfgEager.Tools.LazyLoading.Enabled = false
	cfgEager.Tools.ReadFile.Enabled = true
	cfgEager.Tools.WriteFile.Enabled = true
	cfgEager.Tools.AppendFile.Enabled = true
	cfgEager.Tools.EditFile.Enabled = true
	cfgEager.Tools.ListDir.Enabled = true

	provider := &e2eMockProvider{}
	agentEager := agent.NewAgentInstance(nil, &cfgEager.Agents.Defaults, cfgEager, provider)

	eagerDefinitions := agentEager.Tools.ToProviderDefs()
	eagerJSON, err := json.Marshal(eagerDefinitions)
	require.NoError(t, err)
	eagerBytes := len(eagerJSON)

	// 2. Setup lazy configuration (LazyLoading enabled, hybrid mode)
	cfgLazy := config.DefaultConfig()
	cfgLazy.Agents.Defaults.Workspace = tmpDir
	cfgLazy.Tools.LazyLoading.Enabled = true
	cfgLazy.Tools.LazyLoading.Mode = "hybrid"
	cfgLazy.Tools.LazyLoading.DefaultTTL = 2
	cfgLazy.Tools.LazyLoading.AutoIntercept = true
	// Only read_file and load_tool are core tools
	cfgLazy.Tools.LazyLoading.CoreTools = []string{"read_file", "load_tool"}
	cfgLazy.Tools.ReadFile.Enabled = true
	cfgLazy.Tools.WriteFile.Enabled = true
	cfgLazy.Tools.AppendFile.Enabled = true
	cfgLazy.Tools.EditFile.Enabled = true
	cfgLazy.Tools.ListDir.Enabled = true

	agentLazy := agent.NewAgentInstance(nil, &cfgLazy.Agents.Defaults, cfgLazy, provider)

	lazyDefinitions := agentLazy.Tools.ToProviderDefs()
	lazyJSON, err := json.Marshal(lazyDefinitions)
	require.NoError(t, err)
	lazyBytes := len(lazyJSON)

	// Verify significant reduction in schema bytes sent to LLM
	t.Logf("Eager schema size: %d bytes across %d tools", eagerBytes, len(eagerDefinitions))
	t.Logf("Lazy schema size: %d bytes across %d tools", lazyBytes, len(lazyDefinitions))
	assert.Less(t, lazyBytes, eagerBytes, "Lazy loading should drastically reduce initial schema size")
	reductionPercent := float64(eagerBytes-lazyBytes) / float64(eagerBytes) * 100
	t.Logf("Schema size reduction: %.2f%%", reductionPercent)
	assert.Greater(t, reductionPercent, 25.0, "Expected >25% reduction in tool definition payload")

	// 3. Test Auto-Intercept Promotion of a deferred tool (write_file)
	writeEntry, exists := agentLazy.Tools.GetEntry("write_file")
	require.True(t, exists)
	assert.False(t, writeEntry.IsCore)
	assert.Equal(t, 0, writeEntry.TTL)

	// write_file shouldn't be in provider defs yet
	_, callableBefore := agentLazy.Tools.Get("write_file")
	assert.False(t, callableBefore)

	// Calling the deferred tool via Execute should auto-promote it
	ctx := context.Background()
	res := agentLazy.Tools.Execute(ctx, "write_file", map[string]any{
		"path":    "hello.txt",
		"content": "hello world",
	})
	require.NotNil(t, res)
	assert.False(t, res.IsError)

	// After auto-promotion, write_file should have active TTL and appear in ToProviderDefs()
	writeEntryAfter, _ := agentLazy.Tools.GetEntry("write_file")
	assert.Greater(t, writeEntryAfter.TTL, 0)
	_, callableAfter := agentLazy.Tools.Get("write_file")
	assert.True(t, callableAfter)

	activeDefs := agentLazy.Tools.ToProviderDefs()
	var foundInActive bool
	for _, def := range activeDefs {
		if def.Function.Name == "write_file" {
			foundInActive = true
			break
		}
	}
	assert.True(t, foundInActive, "Auto-promoted tool should now be in ToProviderDefs()")

	// Step turn (decrements TTL via TickTTL)
	agentLazy.Tools.TickTTL() // TTL decreases: 2 -> 1
	_, callableTick1 := agentLazy.Tools.Get("write_file")
	assert.True(t, callableTick1)

	agentLazy.Tools.TickTTL() // TTL decreases: 1 -> 0, demoted!
	_, callableTick2 := agentLazy.Tools.Get("write_file")
	assert.False(t, callableTick2, "Tool should be demoted after TTL expires")
}
