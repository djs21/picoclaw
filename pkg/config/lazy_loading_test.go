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
	assert.Contains(t, lazy.CoreTools, "read_file")
	assert.Contains(t, lazy.CoreTools, "write_file")
	assert.Contains(t, lazy.CoreTools, "exec")
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
		CoreTools: []string{"read_file", "write_file", "load_tool"},
	}

	assert.True(t, lazy.IsCoreTool("read_file"))
	assert.True(t, lazy.IsCoreTool("READ_FILE"))
	assert.True(t, lazy.IsCoreTool("  write_file  "))
	assert.False(t, lazy.IsCoreTool("spi"))
	assert.False(t, lazy.IsCoreTool("web_search"))
}
