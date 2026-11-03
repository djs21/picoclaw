package agent

import (
	"context"
	"testing"

	agenttools "github.com/sipeed/picoclaw/pkg/tools"
)

type allowlistTestTool struct {
	name string
}

func (t *allowlistTestTool) Name() string { return t.name }

func (t *allowlistTestTool) Description() string { return "test tool" }

func (t *allowlistTestTool) Parameters() map[string]any {
	return map[string]any{"type": "object"}
}

func (t *allowlistTestTool) Execute(
	_ context.Context,
	_ map[string]any,
) *agenttools.ToolResult {
	return agenttools.NewToolResult("ok")
}

func TestUnknownAgentToolNames(t *testing.T) {
	workspace := setupWorkspace(t, map[string]string{
		"AGENT.md": `---
tools: [read_file, web_serach]
---
# Agent
`,
	})
	defer cleanupWorkspace(t, workspace)

	registry := agenttools.NewToolRegistry()
	registry.Register(&allowlistTestTool{name: "read_file"})
	registry.Register(&allowlistTestTool{name: "web_search"})

	unknown := unknownAgentToolNames(registry, loadAgentDefinition(workspace))
	if len(unknown) != 1 || unknown[0] != "web_serach" {
		t.Fatalf("unknownAgentToolNames() = %v, want [web_serach]", unknown)
	}
}

func TestUnknownAgentToolNamesUsesRegisteredRuntimeTools(t *testing.T) {
	workspace := setupWorkspace(t, map[string]string{
		"AGENT.md": `---
tools: [serial, reaction, send_tts, load_image, delegate, made_up]
---
# Agent
`,
	})
	defer cleanupWorkspace(t, workspace)

	registry := agenttools.NewToolRegistry()
	for _, name := range []string{"serial", "reaction", "send_tts", "load_image", "delegate"} {
		registry.Register(&allowlistTestTool{name: name})
	}

	unknown := unknownAgentToolNames(registry, loadAgentDefinition(workspace))
	if len(unknown) != 1 || unknown[0] != "made_up" {
		t.Fatalf("unknownAgentToolNames() = %v, want [made_up]", unknown)
	}
}

func TestResolveAgentToolAllowlistDistinguishesMissingAndEmptyToolsField(t *testing.T) {
	tests := []struct {
		name      string
		agentMD   string
		wantNil   bool
		wantEmpty bool
	}{
		{
			name: "missing tools field allows all tools",
			agentMD: `---
name: pico
---
# Agent
`,
			wantNil: true,
		},
		{
			name: "explicit empty tools list blocks all tools",
			agentMD: `---
tools: []
---
# Agent
`,
			wantEmpty: true,
		},
		{
			name: "blank tools field blocks all tools",
			agentMD: `---
tools:
---
# Agent
`,
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workspace := setupWorkspace(t, map[string]string{
				"AGENT.md": tt.agentMD,
			})
			defer cleanupWorkspace(t, workspace)

			allowlist := resolveAgentToolAllowlist(loadAgentDefinition(workspace))

			if tt.wantNil {
				if allowlist != nil {
					t.Fatalf("resolveAgentToolAllowlist() = %v, want nil", allowlist)
				}
				return
			}

			if allowlist == nil {
				t.Fatal("resolveAgentToolAllowlist() = nil, want explicit empty allowlist")
			}
			if len(allowlist) != 0 {
				t.Fatalf("resolveAgentToolAllowlist() = %v, want empty allowlist", allowlist)
			}
		})
	}
}


