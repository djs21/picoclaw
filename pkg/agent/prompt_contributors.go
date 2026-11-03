package agent

import (
	"context"
	"strings"

	"github.com/sipeed/picoclaw/pkg/tools"
)

type toolDiscoveryPromptContributor struct {
	useBM25  bool
	useRegex bool
}

func (c toolDiscoveryPromptContributor) PromptSource() PromptSourceDescriptor {
	return PromptSourceDescriptor{
		ID:              PromptSourceToolDiscovery,
		Owner:           "tools",
		Description:     "Tool discovery instructions",
		Allowed:         []PromptPlacement{{Layer: PromptLayerCapability, Slot: PromptSlotTooling}},
		StableByDefault: true,
	}
}

func (c toolDiscoveryPromptContributor) ContributePrompt(
	_ context.Context,
	req PromptBuildRequest,
) ([]PromptPart, error) {
	if req.SuppressToolUseRule {
		return nil, nil
	}
	useBM25 := c.useBM25 && promptAllowsTool(req, tools.BM25SearchToolName)
	useRegex := c.useRegex && promptAllowsTool(req, tools.RegexSearchToolName)
	if !useBM25 && !useRegex {
		return nil, nil
	}
	content := formatToolDiscoveryRule(useBM25, useRegex)
	if strings.TrimSpace(content) == "" {
		return nil, nil
	}

	return []PromptPart{
		{
			ID:      "capability.tool_discovery",
			Layer:   PromptLayerCapability,
			Slot:    PromptSlotTooling,
			Source:  PromptSource{ID: PromptSourceToolDiscovery, Name: "tool_registry:discovery"},
			Title:   "tool discovery",
			Content: content,
			Stable:  true,
			Cache:   PromptCacheEphemeral,
		},
	}, nil
}


type agentDiscoveryPromptContributor struct {
	agentID  string
	discover func(agentID string) []AgentDescriptor
}

func (c agentDiscoveryPromptContributor) PromptSource() PromptSourceDescriptor {
	return PromptSourceDescriptor{
		ID:              PromptSourceAgentDiscovery,
		Owner:           "agent",
		Description:     "Structured multi-agent discovery registry",
		Allowed:         []PromptPlacement{{Layer: PromptLayerCapability, Slot: PromptSlotTooling}},
		StableByDefault: false,
	}
}

func (c agentDiscoveryPromptContributor) ContributePrompt(
	_ context.Context,
	req PromptBuildRequest,
) ([]PromptPart, error) {
	if req.SuppressToolUseRule {
		return nil, nil
	}
	if !promptAllowsTool(req, "spawn") {
		return nil, nil
	}
	if c.discover == nil {
		return nil, nil
	}
	content := formatAgentDiscoverySection(c.discover(c.agentID))
	if strings.TrimSpace(content) == "" {
		return nil, nil
	}

	return []PromptPart{
		{
			ID:      "capability.agent_discovery",
			Layer:   PromptLayerCapability,
			Slot:    PromptSlotTooling,
			Source:  PromptSource{ID: PromptSourceAgentDiscovery, Name: "agent:discovery"},
			Title:   "agent discovery",
			Content: content,
			Stable:  false,
			Cache:   PromptCacheNone,
		},
	}, nil
}


func promptAllowsTool(req PromptBuildRequest, name string) bool {
	if len(req.AllowedTools) == 0 {
		return true
	}
	allowed := cleanAllowedSet(req.AllowedTools)
	_, ok := allowed[strings.ToLower(strings.TrimSpace(name))]
	return ok
}

