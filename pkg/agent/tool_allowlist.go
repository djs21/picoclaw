package agent

import (
	"sort"
	"strings"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/tools"
)

func warnOnUnknownAgentToolDeclarations(
	agentID, workspace string,
	definition AgentContextDefinition,
	registry *tools.ToolRegistry,
) {
	if registry == nil || frontmatterParseFailed(definition) {
		return
	}

	if unknownTools := unknownAgentToolNames(registry, definition); len(unknownTools) > 0 {
		logger.WarnCF("agent", "AGENT.md declares unregistered tool names",
			map[string]any{
				"agent_id":  agentID,
				"workspace": workspace,
				"tools":     unknownTools,
			})
	}
}

func unknownAgentToolNames(
	registry *tools.ToolRegistry,
	definition AgentContextDefinition,
) []string {
	if definition.Agent == nil || definition.Agent.Frontmatter.Tools == nil {
		return nil
	}

	known := registeredRuntimeToolNames(registry)
	unknown := make(map[string]struct{})
	for _, raw := range definition.Agent.Frontmatter.Tools {
		name := strings.ToLower(strings.TrimSpace(raw))
		if name == "" {
			continue
		}
		if _, ok := known[name]; ok {
			continue
		}
		unknown[name] = struct{}{}
	}

	return sortedKeys(unknown)
}

func registeredRuntimeToolNames(registry *tools.ToolRegistry) map[string]struct{} {
	known := make(map[string]struct{})
	if registry == nil {
		return known
	}
	for _, raw := range registry.List() {
		name := strings.ToLower(strings.TrimSpace(raw))
		if name == "" {
			continue
		}
		known[name] = struct{}{}
	}
	return known
}

func sortedKeys(values map[string]struct{}) []string {
	if len(values) == 0 {
		return nil
	}

	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func resolveAgentToolAllowlist(definition AgentContextDefinition) []string {
	if frontmatterParseFailed(definition) {
		return []string{}
	}
	if definition.Agent == nil || !frontmatterDeclaresField(definition, "tools") {
		return nil
	}

	allowlist := make(map[string]struct{}, len(definition.Agent.Frontmatter.Tools))
	for _, raw := range definition.Agent.Frontmatter.Tools {
		trimmed := strings.ToLower(strings.TrimSpace(raw))
		if trimmed == "" {
			continue
		}
		allowlist[trimmed] = struct{}{}
	}

	if len(allowlist) == 0 {
		return []string{}
	}

	return sortedKeys(allowlist)
}

func frontmatterDeclaresField(definition AgentContextDefinition, field string) bool {
	if definition.Agent == nil || definition.Agent.Frontmatter.Fields == nil {
		return false
	}
	_, ok := definition.Agent.Frontmatter.Fields[field]
	return ok
}

func frontmatterParseFailed(definition AgentContextDefinition) bool {
	if definition.Agent == nil {
		return false
	}
	if strings.TrimSpace(definition.Agent.RawFrontmatter) == "" {
		return false
	}
	return strings.TrimSpace(definition.Agent.FrontmatterErr) != ""
}
