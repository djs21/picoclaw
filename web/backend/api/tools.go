package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
	picotools "github.com/sipeed/picoclaw/pkg/tools"
)

type toolCatalogEntry struct {
	Name        string
	Description string
	Category    string
	ConfigKey   string
}

type toolSupportItem struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	ConfigKey   string `json:"config_key"`
	Status      string `json:"status"`
	ReasonCode  string `json:"reason_code,omitempty"`
}

type toolSupportResponse struct {
	Tools []toolSupportItem `json:"tools"`
}

type toolStateRequest struct {
	Enabled bool `json:"enabled"`
}

type webSearchProviderOption struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	Configured   bool   `json:"configured"`
	Current      bool   `json:"current"`
	RequiresAuth bool   `json:"requires_auth"`
}

type webSearchProviderConfig struct {
	Enabled    bool     `json:"enabled"`
	MaxResults int      `json:"max_results"`
	BaseURL    string   `json:"base_url,omitempty"`
	APIKey     string   `json:"api_key,omitempty"`
	APIKeys    []string `json:"api_keys,omitempty"`
	Model      string   `json:"model,omitempty"`
	APIKeySet  bool     `json:"api_key_set,omitempty"`
}

type webSearchConfigResponse struct {
	Provider       string                             `json:"provider"`
	CurrentService string                             `json:"current_service"`
	PreferNative   bool                               `json:"prefer_native"`
	Proxy          string                             `json:"proxy,omitempty"`
	Providers      []webSearchProviderOption          `json:"providers"`
	Settings       map[string]webSearchProviderConfig `json:"settings"`
}

type webSearchConfigRequest struct {
	Provider     string                             `json:"provider"`
	PreferNative bool                               `json:"prefer_native"`
	Proxy        string                             `json:"proxy"`
	Settings     map[string]webSearchProviderConfig `json:"settings"`
}

var toolCatalog = []toolCatalogEntry{
	{
		Name:        "read_file",
		Description: "Read file content from the workspace or explicitly allowed paths.",
		Category:    "filesystem",
		ConfigKey:   "read_file",
	},
	{
		Name:        "list_dir",
		Description: "Inspect directories and enumerate files available to the agent.",
		Category:    "filesystem",
		ConfigKey:   "list_dir",
	},
	{
		Name:        "edit_file",
		Description: "Create, overwrite, append to, or patch files within the writable workspace scope.",
		Category:    "filesystem",
		ConfigKey:   "edit_file",
	},
	{
		Name:        "exec",
		Description: "Run shell commands inside the configured workspace sandbox.",
		Category:    "filesystem",
		ConfigKey:   "exec",
	},
	{
		Name:        "cron",
		Description: "Schedule one-time or recurring reminders, jobs, and shell commands.",
		Category:    "automation",
		ConfigKey:   "cron",
	},
	{
		Name:        "web_search",
		Description: "Search the web using the configured providers.",
		Category:    "web",
		ConfigKey:   "web",
	},
	{
		Name:        "web_fetch",
		Description: "Fetch and summarize the contents of a webpage.",
		Category:    "web",
		ConfigKey:   "web_fetch",
	},
	{
		Name:        "message",
		Description: "Send a follow-up message or file attachment back to the active user or chat.",
		Category:    "communication",
		ConfigKey:   "message",
	},
	{
		Name:        "skills",
		Description: "Browse and install skills from the configured registries.",
		Category:    "skills",
		ConfigKey:   "skills",
	},
	{
		Name: "spawn",
		Description: "Delegate work to a subagent: spawn a background or synchronous " +
			"task, or inspect spawned task status.",
		Category:  "agents",
		ConfigKey: "spawn",
	},
}

func (h *Handler) registerToolRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/tools", h.handleListTools)
	mux.HandleFunc("PUT /api/tools/{name}/state", h.handleUpdateToolState)
	mux.HandleFunc("GET /api/tools/web-search-config", h.handleGetWebSearchConfig)
	mux.HandleFunc("PUT /api/tools/web-search-config", h.handleUpdateWebSearchConfig)
}

func (h *Handler) handleListTools(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toolSupportResponse{
		Tools: buildToolSupport(cfg),
	})
}

func (h *Handler) handleUpdateToolState(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	var req toolStateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if err := applyToolState(cfg, r.PathValue("name"), req.Enabled); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func buildToolSupport(cfg *config.Config) []toolSupportItem {
	items := make([]toolSupportItem, 0, len(toolCatalog))
	for _, entry := range toolCatalog {
		status := "disabled"
		reasonCode := ""

		switch entry.Name {
		case "spawn":
			if cfg.Tools.IsToolEnabled(entry.ConfigKey) {
				if cfg.Tools.IsToolEnabled("subagent") {
					status = "enabled"
				} else {
					status = "blocked"
					reasonCode = "requires_subagent"
				}
			}
		case "web_search":
			status, reasonCode = resolveWebSearchToolSupport(cfg)
		default:
			if cfg.Tools.IsToolEnabled(entry.ConfigKey) {
				status = "enabled"
			}
		}

		items = append(items, toolSupportItem{
			Name:        entry.Name,
			Description: entry.Description,
			Category:    entry.Category,
			ConfigKey:   entry.ConfigKey,
			Status:      status,
			ReasonCode:  reasonCode,
		})
	}
	return items
}

func resolveWebSearchToolSupport(cfg *config.Config) (string, string) {
	if !cfg.Tools.IsToolEnabled("web") {
		return "disabled", ""
	}
	return "enabled", ""
}

func applyToolState(cfg *config.Config, toolName string, enabled bool) error {
	switch toolName {
	case "read_file":
		cfg.Tools.ReadFile.Enabled = enabled
	case "list_dir":
		cfg.Tools.ListDir.Enabled = enabled
	case "edit_file":
		cfg.Tools.EditFile.Enabled = enabled
	case "exec":
		cfg.Tools.Exec.Enabled = enabled
	case "cron":
		cfg.Tools.Cron.Enabled = enabled
	case "web_search":
		cfg.Tools.Web.Enabled = enabled
	case "web_fetch":
		cfg.Tools.WebFetch.Enabled = enabled
	case "message":
		cfg.Tools.Message.Enabled = enabled
	case "send_file":
		return fmt.Errorf("tool %q was consolidated into message", toolName)
	case "skills":
		cfg.Tools.Skills.Enabled = enabled
	case "find_skills", "install_skill":
		// Skills are installed through the CLI workflow now; these tools no
		// longer exist as agent tools and cannot be toggled.
		return fmt.Errorf("tool %q was removed", toolName)
	case "spawn":
		cfg.Tools.Spawn.Enabled = enabled
		if enabled {
			cfg.Tools.Subagent.Enabled = true
		}
	case "spawn_status", "subagent":
		// spawn_status was folded into the unified spawn tool and subagent is
		// an implicit runtime dependency of it, so neither is separately
		// togglable from the tool catalog.
		return fmt.Errorf("tool %q was consolidated into spawn", toolName)
	default:
		return fmt.Errorf("tool %q cannot be updated", toolName)
	}
	return nil
}

func (h *Handler) handleGetWebSearchConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(buildWebSearchConfigResponse(cfg)); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (h *Handler) handleUpdateWebSearchConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	var req webSearchConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	provider := normalizeWebSearchProvider(req.Provider)
	if provider == "" {
		http.Error(w, "invalid web search provider", http.StatusBadRequest)
		return
	}

	cfg.Tools.Web.Provider = provider
	cfg.Tools.Web.PreferNative = req.PreferNative
	cfg.Tools.Web.Proxy = strings.TrimSpace(req.Proxy)

	if settings, ok := req.Settings["sogou"]; ok {
		cfg.Tools.Web.Sogou.Enabled = settings.Enabled
		cfg.Tools.Web.Sogou.MaxResults = settings.MaxResults
	}
	if settings, ok := req.Settings["duckduckgo"]; ok {
		cfg.Tools.Web.DuckDuckGo.Enabled = settings.Enabled
		cfg.Tools.Web.DuckDuckGo.MaxResults = settings.MaxResults
	}
	if settings, ok := req.Settings["gemini"]; ok {
		cfg.Tools.Web.Gemini.Enabled = settings.Enabled
		cfg.Tools.Web.Gemini.MaxResults = settings.MaxResults
		cfg.Tools.Web.Gemini.Model = strings.TrimSpace(settings.Model)
		if key := strings.TrimSpace(settings.APIKey); key != "" {
			cfg.Tools.Web.Gemini.APIKey = *config.NewSecureString(key)
		}
	}
	if settings, ok := req.Settings["brave"]; ok {
		cfg.Tools.Web.Brave.Enabled = settings.Enabled
		cfg.Tools.Web.Brave.MaxResults = settings.MaxResults
		if keys, ok := normalizeWebSearchAPIKeys(settings.APIKeys, settings.APIKey); ok {
			cfg.Tools.Web.Brave.SetAPIKeys(keys)
		}
	}
	if settings, ok := req.Settings["tavily"]; ok {
		cfg.Tools.Web.Tavily.Enabled = settings.Enabled
		cfg.Tools.Web.Tavily.MaxResults = settings.MaxResults
		cfg.Tools.Web.Tavily.BaseURL = strings.TrimSpace(settings.BaseURL)
		if keys, ok := normalizeWebSearchAPIKeys(settings.APIKeys, settings.APIKey); ok {
			cfg.Tools.Web.Tavily.SetAPIKeys(keys)
		}
	}
	if settings, ok := req.Settings["kagi"]; ok {
		cfg.Tools.Web.Kagi.Enabled = settings.Enabled
		cfg.Tools.Web.Kagi.MaxResults = settings.MaxResults
		cfg.Tools.Web.Kagi.BaseURL = strings.TrimSpace(settings.BaseURL)
		if keys, ok := normalizeWebSearchAPIKeys(settings.APIKeys, settings.APIKey); ok {
			cfg.Tools.Web.Kagi.SetAPIKeys(keys)
		}
	}
	if settings, ok := req.Settings["perplexity"]; ok {
		cfg.Tools.Web.Perplexity.Enabled = settings.Enabled
		cfg.Tools.Web.Perplexity.MaxResults = settings.MaxResults
		if keys, ok := normalizeWebSearchAPIKeys(settings.APIKeys, settings.APIKey); ok {
			cfg.Tools.Web.Perplexity.APIKeys = config.SimpleSecureStrings(keys...)
		}
	}
	if settings, ok := req.Settings["searxng"]; ok {
		cfg.Tools.Web.SearXNG.Enabled = settings.Enabled
		cfg.Tools.Web.SearXNG.MaxResults = settings.MaxResults
		cfg.Tools.Web.SearXNG.BaseURL = strings.TrimSpace(settings.BaseURL)
	}
	if settings, ok := req.Settings["glm_search"]; ok {
		cfg.Tools.Web.GLMSearch.Enabled = settings.Enabled
		cfg.Tools.Web.GLMSearch.MaxResults = settings.MaxResults
		cfg.Tools.Web.GLMSearch.BaseURL = strings.TrimSpace(settings.BaseURL)
		if key := strings.TrimSpace(settings.APIKey); key != "" {
			cfg.Tools.Web.GLMSearch.APIKey = *config.NewSecureString(key)
		}
	}
	if settings, ok := req.Settings["baidu_search"]; ok {
		cfg.Tools.Web.BaiduSearch.Enabled = settings.Enabled
		cfg.Tools.Web.BaiduSearch.MaxResults = settings.MaxResults
		cfg.Tools.Web.BaiduSearch.BaseURL = strings.TrimSpace(settings.BaseURL)
		if key := strings.TrimSpace(settings.APIKey); key != "" {
			cfg.Tools.Web.BaiduSearch.APIKey = *config.NewSecureString(key)
		}
	}

	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(buildWebSearchConfigResponse(cfg)); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func normalizeWebSearchProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "", "auto":
		return "auto"
	case "sogou",
		"brave",
		"tavily",
		"kagi",
		"duckduckgo",
		"gemini",
		"perplexity",
		"searxng",
		"glm_search",
		"baidu_search":
		return strings.ToLower(strings.TrimSpace(provider))
	default:
		return ""
	}
}

func normalizeWebSearchAPIKeys(apiKeys []string, apiKey string) ([]string, bool) {
	if apiKeys != nil {
		keys := make([]string, 0, len(apiKeys))
		seen := make(map[string]struct{}, len(apiKeys))
		for _, key := range apiKeys {
			trimmed := strings.TrimSpace(key)
			if trimmed == "" {
				continue
			}
			if _, ok := seen[trimmed]; ok {
				continue
			}
			seen[trimmed] = struct{}{}
			keys = append(keys, trimmed)
		}
		return keys, true
	}

	if trimmed := strings.TrimSpace(apiKey); trimmed != "" {
		return []string{trimmed}, true
	}

	return nil, false
}

func buildWebSearchConfigResponse(cfg *config.Config) webSearchConfigResponse {
	opts := picotools.WebSearchToolOptionsFromConfig(cfg)
	current := resolveCurrentWebSearchProvider(cfg)
	settings := map[string]webSearchProviderConfig{
		"sogou": {
			Enabled:    cfg.Tools.Web.Sogou.Enabled,
			MaxResults: cfg.Tools.Web.Sogou.MaxResults,
		},
		"duckduckgo": {
			Enabled:    cfg.Tools.Web.DuckDuckGo.Enabled,
			MaxResults: cfg.Tools.Web.DuckDuckGo.MaxResults,
		},
		"gemini": {
			Enabled:    cfg.Tools.Web.Gemini.Enabled,
			MaxResults: cfg.Tools.Web.Gemini.MaxResults,
			Model:      cfg.Tools.Web.Gemini.Model,
			APIKeySet:  cfg.Tools.Web.Gemini.APIKey.String() != "",
		},
		"brave": {
			Enabled:    cfg.Tools.Web.Brave.Enabled,
			MaxResults: cfg.Tools.Web.Brave.MaxResults,
			APIKeySet:  len(cfg.Tools.Web.Brave.APIKeys.Values()) > 0,
		},
		"tavily": {
			Enabled:    cfg.Tools.Web.Tavily.Enabled,
			MaxResults: cfg.Tools.Web.Tavily.MaxResults,
			BaseURL:    cfg.Tools.Web.Tavily.BaseURL,
			APIKeySet:  len(cfg.Tools.Web.Tavily.APIKeys.Values()) > 0,
		},
		"kagi": {
			Enabled:    cfg.Tools.Web.Kagi.Enabled,
			MaxResults: cfg.Tools.Web.Kagi.MaxResults,
			BaseURL:    cfg.Tools.Web.Kagi.BaseURL,
			APIKeySet:  len(cfg.Tools.Web.Kagi.APIKeys.Values()) > 0,
		},
		"perplexity": {
			Enabled:    cfg.Tools.Web.Perplexity.Enabled,
			MaxResults: cfg.Tools.Web.Perplexity.MaxResults,
			APIKeySet:  len(cfg.Tools.Web.Perplexity.APIKeys.Values()) > 0,
		},
		"searxng": {
			Enabled:    cfg.Tools.Web.SearXNG.Enabled,
			MaxResults: cfg.Tools.Web.SearXNG.MaxResults,
			BaseURL:    cfg.Tools.Web.SearXNG.BaseURL,
		},
		"glm_search": {
			Enabled:    cfg.Tools.Web.GLMSearch.Enabled,
			MaxResults: cfg.Tools.Web.GLMSearch.MaxResults,
			BaseURL:    cfg.Tools.Web.GLMSearch.BaseURL,
			APIKeySet:  cfg.Tools.Web.GLMSearch.APIKey.String() != "",
		},
		"baidu_search": {
			Enabled:    cfg.Tools.Web.BaiduSearch.Enabled,
			MaxResults: cfg.Tools.Web.BaiduSearch.MaxResults,
			BaseURL:    cfg.Tools.Web.BaiduSearch.BaseURL,
			APIKeySet:  cfg.Tools.Web.BaiduSearch.APIKey.String() != "",
		},
	}

	providers := []webSearchProviderOption{
		{
			ID:         "auto",
			Label:      "Auto",
			Configured: current != "",
			Current: cfg.Tools.Web.Provider == "" ||
				cfg.Tools.Web.Provider == "auto",
		},
		{
			ID:         "sogou",
			Label:      "Sogou",
			Configured: picotools.WebSearchProviderReady(opts, "sogou"),
			Current:    current == "sogou",
		},
		{
			ID:         "duckduckgo",
			Label:      "DuckDuckGo",
			Configured: picotools.WebSearchProviderReady(opts, "duckduckgo"),
			Current:    current == "duckduckgo",
		},
		{
			ID:           "gemini",
			Label:        "Gemini (Google Search)",
			Configured:   picotools.WebSearchProviderReady(opts, "gemini"),
			Current:      current == "gemini",
			RequiresAuth: true,
		},
		{
			ID:           "brave",
			Label:        "Brave Search",
			Configured:   picotools.WebSearchProviderReady(opts, "brave"),
			Current:      current == "brave",
			RequiresAuth: true,
		},
		{
			ID:           "tavily",
			Label:        "Tavily",
			Configured:   picotools.WebSearchProviderReady(opts, "tavily"),
			Current:      current == "tavily",
			RequiresAuth: true,
		},
		{
			ID:           "kagi",
			Label:        "Kagi Search",
			Configured:   picotools.WebSearchProviderReady(opts, "kagi"),
			Current:      current == "kagi",
			RequiresAuth: true,
		},
		{
			ID:           "perplexity",
			Label:        "Perplexity",
			Configured:   picotools.WebSearchProviderReady(opts, "perplexity"),
			Current:      current == "perplexity",
			RequiresAuth: true,
		},
		{
			ID:         "searxng",
			Label:      "SearXNG",
			Configured: picotools.WebSearchProviderReady(opts, "searxng"),
			Current:    current == "searxng",
		},
		{
			ID:           "glm_search",
			Label:        "GLM Search",
			Configured:   picotools.WebSearchProviderReady(opts, "glm_search"),
			Current:      current == "glm_search",
			RequiresAuth: true,
		},
		{
			ID:           "baidu_search",
			Label:        "Baidu Search",
			Configured:   picotools.WebSearchProviderReady(opts, "baidu_search"),
			Current:      current == "baidu_search",
			RequiresAuth: true,
		},
	}

	provider := cfg.Tools.Web.Provider
	if provider == "" {
		provider = "auto"
	}

	return webSearchConfigResponse{
		Provider:       provider,
		CurrentService: current,
		PreferNative:   cfg.Tools.Web.PreferNative,
		Proxy:          cfg.Tools.Web.Proxy,
		Providers:      providers,
		Settings:       settings,
	}
}

func resolveCurrentWebSearchProvider(cfg *config.Config) string {
	if cfg == nil || !cfg.Tools.IsToolEnabled("web") {
		return ""
	}
	selected, err := picotools.ResolveWebSearchProviderName(picotools.WebSearchToolOptionsFromConfig(cfg), "")
	if err != nil {
		return ""
	}
	return selected
}
