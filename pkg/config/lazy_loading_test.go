package config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLazyLoadingConfig_Defaults(t *testing.T) {
	cfg := DefaultConfig()
	require.NotNil(t, cfg)

	lazy := cfg.Tools.LazyLoading
	assert.False(t, lazy.Enabled, "LazyLoading should be disabled by default for backward compatibility")
	assert.Equal(t, "hybrid", lazy.Mode)
	assert.Equal(t, 3, lazy.DefaultTTL)
	assert.True(t, lazy.AutoIntercept)
	assert.Equal(t, 5, lazy.MaxSearchResults)
	assert.Equal(t, []string{"read_file", "edit_file", "exec"}, lazy.CoreTools)
}

func TestLazyLoadingConfig_DefaultsDropLegacyTools(t *testing.T) {
	cfg := DefaultConfig()
	require.NotNil(t, cfg)

	for _, legacy := range []string{"write_file", "append_file", "list_dir", "subagent", "spawn_status"} {
		assert.NotContains(t, cfg.Tools.LazyLoading.CoreTools, legacy,
			"%s was consolidated or removed and must not be a default core tool", legacy)
	}
}

func TestLazyLoadingConfig_UnmarshalJSON(t *testing.T) {
	input := `{
		"tools": {
			"lazy_loading": {
				"enabled": true,
				"mode": "hybrid",
				"default_ttl": 5,
				"auto_intercept": true,
				"max_search_results": 10,
				"core_tools": ["read_file", "message", "load_tool"]
			}
		}
	}`

	var cfg Config
	err := json.Unmarshal([]byte(input), &cfg)
	require.NoError(t, err)

	assert.True(t, cfg.Tools.LazyLoading.Enabled)
	assert.Equal(t, "hybrid", cfg.Tools.LazyLoading.Mode)
	assert.Equal(t, 5, cfg.Tools.LazyLoading.DefaultTTL)
	assert.True(t, cfg.Tools.LazyLoading.AutoIntercept)
	assert.Equal(t, 10, cfg.Tools.LazyLoading.MaxSearchResults)
	assert.Equal(t, []string{"read_file", "message", "load_tool"}, cfg.Tools.LazyLoading.CoreTools)
}

func TestLazyLoadingConfig_HelperMethods(t *testing.T) {
	lazy := LazyLoadingConfig{
		Enabled:   true,
		CoreTools: []string{"read_file", "edit_file", "load_tool"},
	}

	assert.True(t, lazy.IsCoreTool("read_file"))
	assert.True(t, lazy.IsCoreTool("READ_FILE"))
	assert.True(t, lazy.IsCoreTool("  edit_file  "))
	assert.True(t, lazy.IsCoreTool("load_tool"))
	assert.True(t, lazy.IsCoreTool("tool_search_tool_regex"))
	assert.True(t, lazy.IsCoreTool("tool_search_tool_bm25"))
	assert.False(t, lazy.IsCoreTool("spi"))
	assert.False(t, lazy.IsCoreTool("web_fetch"))
	assert.False(t, lazy.IsCoreTool(""))
}

func TestLazyLoadingConfig_EffectiveCoreToolsFallsBackToDefaults(t *testing.T) {
	unset := LazyLoadingConfig{Enabled: true}
	assert.Equal(t, DefaultLazyCoreTools, unset.EffectiveCoreTools())
	assert.True(t, unset.IsCoreTool("read_file"))
	assert.False(t, unset.IsCoreTool("web_search"))
	assert.False(t, unset.IsCoreTool("web_fetch"))
	assert.False(t, unset.IsCoreTool("write_file"))

	explicit := LazyLoadingConfig{Enabled: true, CoreTools: []string{"message"}}
	assert.Equal(t, []string{"message"}, explicit.EffectiveCoreTools())
	assert.True(t, explicit.IsCoreTool("message"))
	assert.False(t, explicit.IsCoreTool("read_file"))
}
