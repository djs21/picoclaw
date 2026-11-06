package tools

import (
	"context"
	"strings"
	"testing"
)

func TestSpawnAgentID_RoutesToNamedPeer(t *testing.T) {
	tool, manager, spawner := newTestSpawnTool(t)
	tool.SetAllowlistChecker(func(target string) bool { return target == "worker" })

	result := tool.Execute(context.Background(), map[string]any{
		"task":     "Do the peer task",
		"label":    "peer-label",
		"agent_id": "worker",
		"sync":     true,
	})
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "Task completed:") {
		t.Fatalf("sync result should contain subturn output, got: %s", result.ForLLM)
	}
	cfg := spawner.config()
	if cfg.TargetAgentID != "worker" {
		t.Errorf("TargetAgentID = %q, want worker (peer route)", cfg.TargetAgentID)
	}
	// Peer route must let the subturn layer use peer identity/model/workspace:
	// spawn.go must not override them with the parent's generic subagent model.
	if cfg.Model != "" {
		t.Errorf("Model = %q, want empty so peer agent model is used", cfg.Model)
	}
	if cfg.Tools != nil {
		t.Errorf("Tools = %v, want nil so peer tool registry is inherited", cfg.Tools)
	}
	task, ok := manager.GetTaskCopy("subagent-1")
	if !ok {
		t.Fatal("expected task registered")
	}
	if task.AgentID != "worker" {
		t.Errorf("task AgentID = %q, want worker", task.AgentID)
	}
	if task.Status != "completed" {
		t.Errorf("task status = %q, want completed", task.Status)
	}
}

func TestSpawnAgentID_AllowlistRejected(t *testing.T) {
	tool, _, spawner := newTestSpawnTool(t)
	tool.SetAllowlistChecker(func(target string) bool { return false })

	result := tool.Execute(context.Background(), map[string]any{
		"task":     "Do the peer task",
		"agent_id": "forbidden",
		"sync":     true,
	})
	if !result.IsError {
		t.Fatalf("expected allowlist error, got: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "not allowed") {
		t.Errorf("error should mention not allowed, got: %s", result.ForLLM)
	}
	if spawner.config().TargetAgentID != "" {
		t.Error("spawner must not run when allowlist rejects")
	}
}

func TestSpawnAgentID_AsyncReturnsTaskIDAndStatusList(t *testing.T) {
	tool, _, spawner := newTestSpawnTool(t)
	tool.SetAllowlistChecker(func(target string) bool { return target == "worker" })

	result := tool.Execute(context.Background(), map[string]any{
		"task":     "Async peer task",
		"agent_id": "worker",
		"sync":     false,
	})
	if result.IsError {
		t.Fatalf("expected async success, got: %s", result.ForLLM)
	}
	if !result.Async {
		t.Error("expected async result")
	}
	if !strings.Contains(result.ForLLM, "subagent-1") {
		t.Errorf("async result should contain task ID, got: %s", result.ForLLM)
	}
	select {
	case <-spawner.done:
	case <-context.Background().Done():
	}
	cfg := spawner.config()
	if cfg.TargetAgentID != "worker" {
		t.Errorf("async TargetAgentID = %q, want worker", cfg.TargetAgentID)
	}
	if cfg.Model != "" {
		t.Errorf("async Model = %q, want empty for peer route", cfg.Model)
	}

	status := tool.Execute(context.Background(), map[string]any{
		"action":  "status",
		"task_id": "subagent-1",
	})
	if status.IsError {
		t.Fatalf("status should succeed, got: %s", status.ForLLM)
	}
	if !strings.Contains(status.ForLLM, "agent=worker") {
		t.Errorf("status should show peer agent, got: %s", status.ForLLM)
	}

	list := tool.Execute(context.Background(), map[string]any{"action": "list"})
	if list.IsError {
		t.Fatalf("list should succeed, got: %s", list.ForLLM)
	}
	if !strings.Contains(list.ForLLM, "subagent-1") || !strings.Contains(list.ForLLM, "agent=worker") {
		t.Errorf("list should show peer task, got: %s", list.ForLLM)
	}
}
