package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers"
)

// MockLLMProvider is a test implementation of LLMProvider.
type MockLLMProvider struct {
	lastOptions map[string]any
}

func (m *MockLLMProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	options map[string]any,
) (*providers.LLMResponse, error) {
	m.lastOptions = options
	// Find the last user message to generate a response.
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			return &providers.LLMResponse{
				Content: "Task completed: " + messages[i].Content,
			}, nil
		}
	}
	return &providers.LLMResponse{Content: "No task provided"}, nil
}

func (m *MockLLMProvider) GetDefaultModel() string {
	return "test-model"
}

func (m *MockLLMProvider) SupportsTools() bool {
	return false
}

func (m *MockLLMProvider) GetContextWindow() int {
	return 4096
}

// mockSpawner implements SubTurnSpawner for testing.
type mockSpawner struct {
	mu         sync.Mutex
	lastConfig SubTurnConfig
	done       chan struct{}
	closeOnce  sync.Once
	result     *ToolResult
	err        error
}

func (m *mockSpawner) SpawnSubTurn(ctx context.Context, cfg SubTurnConfig) (*ToolResult, error) {
	m.mu.Lock()
	m.lastConfig = cfg
	m.mu.Unlock()

	if m.done != nil {
		m.closeOnce.Do(func() { close(m.done) })
	}

	if m.err != nil {
		return nil, m.err
	}
	if m.result != nil {
		return m.result, nil
	}

	// Extract task from system prompt for response.
	task := cfg.SystemPrompt
	if strings.Contains(task, "Task: ") {
		parts := strings.Split(task, "Task: ")
		if len(parts) > 1 {
			task = parts[1]
		}
	}
	return &ToolResult{
		ForLLM:  "Task completed: " + task,
		ForUser: "Task completed",
	}, nil
}

func (m *mockSpawner) config() SubTurnConfig {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastConfig
}

// waitForTaskStatus polls the manager until the task reaches the wanted status.
func waitForTaskStatus(t *testing.T, manager *SubagentManager, id, want string) SubagentTask {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		task, ok := manager.GetTaskCopy(id)
		if ok && task.Status == want {
			return task
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for task %s status %q (found=%v, task=%+v)", id, want, ok, task)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func newTestSpawnTool(t *testing.T) (*SpawnTool, *SubagentManager, *mockSpawner) {
	t.Helper()
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "test-model", t.TempDir())
	tool := NewSpawnTool(manager)
	spawner := &mockSpawner{done: make(chan struct{})}
	tool.SetSpawner(spawner)
	return tool, manager, spawner
}

func TestSpawnTool_Name(t *testing.T) {
	tool := NewSpawnTool(nil)
	if tool.Name() != "spawn" {
		t.Errorf("Expected name 'spawn', got '%s'", tool.Name())
	}
}

func TestSpawnTool_Parameters(t *testing.T) {
	tool := NewSpawnTool(nil)

	params := tool.Parameters()
	if params["type"] != "object" {
		t.Errorf("Expected type 'object', got: %v", params["type"])
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("Expected 'properties' to be a map")
	}
	for _, name := range []string{"task", "agent_id", "sync", "action", "task_id"} {
		if _, ok := props[name]; !ok {
			t.Errorf("Expected %q parameter in properties", name)
		}
	}

	action, ok := props["action"].(map[string]any)
	if !ok {
		t.Fatal("action parameter should be a map")
	}
	enum, ok := action["enum"].([]string)
	if !ok {
		t.Fatalf("action enum should be []string, got %T", action["enum"])
	}
	wantActions := []string{"spawn", "status", "list"}
	if len(enum) != len(wantActions) {
		t.Fatalf("action enum = %v, want %v", enum, wantActions)
	}
	for i, want := range wantActions {
		if enum[i] != want {
			t.Errorf("action enum[%d] = %q, want %q", i, enum[i], want)
		}
	}

	required, ok := params["required"].([]string)
	if !ok {
		t.Fatalf("Required should be a string array")
	}
	if len(required) != 0 {
		t.Errorf("Required should be empty because it depends on action, got: %v", required)
	}
}

func TestSpawnTool_Execute_EmptyTask(t *testing.T) {
	tool, _, _ := newTestSpawnTool(t)
	ctx := context.Background()

	tests := []struct {
		name string
		args map[string]any
	}{
		{"empty string", map[string]any{"task": ""}},
		{"whitespace only", map[string]any{"task": "   "}},
		{"tabs and newlines", map[string]any{"task": "\t\n  "}},
		{"missing task key", map[string]any{"label": "test"}},
		{"wrong type", map[string]any{"task": 123}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tool.Execute(ctx, tt.args)
			if result == nil {
				t.Fatal("Result should not be nil")
			}
			if !result.IsError {
				t.Error("Expected error for invalid task parameter")
			}
			if !strings.Contains(result.ForLLM, "task is required") {
				t.Errorf("Error message should mention 'task is required', got: %s", result.ForLLM)
			}
		})
	}
}

func TestSpawnTool_AsyncSpawnReturnsTaskID(t *testing.T) {
	tool, manager, spawner := newTestSpawnTool(t)
	ctx := context.Background()

	result := tool.Execute(ctx, map[string]any{
		"task":     "Write a haiku about coding",
		"label":    "haiku-task",
		"agent_id": "research",
	})

	if result == nil {
		t.Fatal("Result should not be nil")
	}
	if result.IsError {
		t.Fatalf("Expected success for valid task, got error: %s", result.ForLLM)
	}
	if !result.Async {
		t.Error("Async spawn should return an async result")
	}
	if !strings.Contains(result.ForLLM, "subagent-1") {
		t.Errorf("Async result should contain the task ID, got: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "haiku-task") {
		t.Errorf("Async result should contain the label, got: %s", result.ForLLM)
	}

	select {
	case <-spawner.done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for async spawn")
	}

	cfg := spawner.config()
	if !cfg.Async {
		t.Error("Async spawn should use Async=true")
	}
	if !cfg.Critical {
		t.Error("Background spawn should be critical")
	}
	if cfg.TargetAgentID != "research" {
		t.Errorf("TargetAgentID = %q, want research", cfg.TargetAgentID)
	}

	task := waitForTaskStatus(t, manager, "subagent-1", "completed")
	if task.Label != "haiku-task" {
		t.Errorf("task label = %q, want haiku-task", task.Label)
	}
	if task.AgentID != "research" {
		t.Errorf("task agent ID = %q, want research", task.AgentID)
	}
}

func TestSpawnTool_SyncSpawnReturnsResult(t *testing.T) {
	tool, manager, spawner := newTestSpawnTool(t)
	ctx := context.Background()

	result := tool.Execute(ctx, map[string]any{
		"task":  "Summarize the repository",
		"label": "summary",
		"sync":  true,
	})

	if result == nil {
		t.Fatal("Result should not be nil")
	}
	if result.IsError {
		t.Fatalf("Expected success, got error: %s", result.ForLLM)
	}
	if result.Async {
		t.Error("Sync spawn should not return an async result")
	}
	if !strings.Contains(result.ForLLM, "Task completed:") {
		t.Errorf("ForLLM should contain the subagent result, got: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "summary") {
		t.Errorf("ForLLM should contain the label, got: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "subagent-1") {
		t.Errorf("ForLLM should contain the task ID, got: %s", result.ForLLM)
	}
	if result.ForUser == "" {
		t.Error("ForUser should contain a result summary")
	}

	cfg := spawner.config()
	if cfg.Async {
		t.Error("Sync spawn should use Async=false")
	}
	if cfg.Critical {
		t.Error("Sync spawn should not be critical")
	}

	task, ok := manager.GetTaskCopy("subagent-1")
	if !ok {
		t.Fatal("Expected sync spawn to register a task in the manager")
	}
	if task.Status != "completed" {
		t.Errorf("task status = %q, want completed", task.Status)
	}
}

func TestSpawnTool_SyncSpawnFailureRecordsTask(t *testing.T) {
	tool, manager, spawner := newTestSpawnTool(t)
	spawner.err = errors.New("boom")

	result := tool.Execute(context.Background(), map[string]any{
		"task": "Will fail",
		"sync": true,
	})

	if !result.IsError {
		t.Fatalf("Expected error result, got: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "boom") {
		t.Errorf("Expected error message to include cause, got: %s", result.ForLLM)
	}

	task, ok := manager.GetTaskCopy("subagent-1")
	if !ok {
		t.Fatal("Expected failed spawn to register a task in the manager")
	}
	if task.Status != "failed" {
		t.Errorf("task status = %q, want failed", task.Status)
	}
}

func TestSpawnTool_ActionStatus(t *testing.T) {
	tool, manager, _ := newTestSpawnTool(t)

	manager.mu.Lock()
	manager.tasks["subagent-42"] = &SubagentTask{
		ID:      "subagent-42",
		Task:    "Specific task",
		Label:   "my-task",
		Status:  "failed",
		Result:  "Something went wrong",
		Created: time.Now().UnixMilli(),
	}
	manager.mu.Unlock()

	result := tool.Execute(context.Background(), map[string]any{
		"action":  "status",
		"task_id": "subagent-42",
	})

	if result.IsError {
		t.Fatalf("Expected success, got error: %s", result.ForLLM)
	}
	for _, want := range []string{"subagent-42", "failed", "Something went wrong", "my-task"} {
		if !strings.Contains(result.ForLLM, want) {
			t.Errorf("Expected %q in output, got: %s", want, result.ForLLM)
		}
	}
}

func TestSpawnTool_ActionStatus_RequiresTaskID(t *testing.T) {
	tool, _, _ := newTestSpawnTool(t)

	result := tool.Execute(context.Background(), map[string]any{"action": "status"})
	if !result.IsError {
		t.Fatalf("Expected error when task_id is missing, got: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "task_id") {
		t.Errorf("Expected task_id in error message, got: %s", result.ForLLM)
	}
}

func TestSpawnTool_ActionStatus_NotFound(t *testing.T) {
	tool, _, _ := newTestSpawnTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"action":  "status",
		"task_id": "nonexistent-999",
	})
	if !result.IsError {
		t.Errorf("Expected error for nonexistent task, got: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "nonexistent-999") {
		t.Errorf("Expected task ID in error message, got: %s", result.ForLLM)
	}
}

func TestSpawnTool_ActionStatus_TaskIDNonString(t *testing.T) {
	tool, _, _ := newTestSpawnTool(t)

	for _, badVal := range []any{42, 3.14, true, map[string]any{"x": 1}, []string{"a"}} {
		result := tool.Execute(context.Background(), map[string]any{
			"action":  "status",
			"task_id": badVal,
		})
		if !result.IsError {
			t.Errorf("Expected error for task_id=%T(%v), got success: %s", badVal, badVal, result.ForLLM)
		}
		if !strings.Contains(result.ForLLM, "task_id must be a string") {
			t.Errorf("Expected type-error message, got: %s", result.ForLLM)
		}
	}
}

func TestSpawnTool_ActionList(t *testing.T) {
	tool, manager, _ := newTestSpawnTool(t)

	now := time.Now().UnixMilli()
	manager.mu.Lock()
	manager.tasks["subagent-1"] = &SubagentTask{
		ID: "subagent-1", Task: "Do task A", Label: "task-a", Status: "running", Created: now,
	}
	manager.tasks["subagent-2"] = &SubagentTask{
		ID: "subagent-2", Task: "Do task B", Label: "task-b", Status: "completed",
		Result: "Done successfully", Created: now + 1,
	}
	manager.tasks["subagent-3"] = &SubagentTask{
		ID: "subagent-3", Task: "Do task C", Status: "failed",
		Result: "Error: something went wrong", Created: now + 2,
	}
	manager.mu.Unlock()

	result := tool.Execute(context.Background(), map[string]any{"action": "list"})
	if result.IsError {
		t.Fatalf("Expected success, got error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "3 total") {
		t.Errorf("Expected total count in header, got: %s", result.ForLLM)
	}
	for _, id := range []string{"subagent-1", "subagent-2", "subagent-3"} {
		if !strings.Contains(result.ForLLM, id) {
			t.Errorf("Expected task %s in output, got:\n%s", id, result.ForLLM)
		}
	}
	for _, status := range []string{"running", "completed", "failed"} {
		if !strings.Contains(result.ForLLM, status) {
			t.Errorf("Expected status %q in output, got:\n%s", status, result.ForLLM)
		}
	}
	if !strings.Contains(result.ForLLM, "Done successfully") {
		t.Errorf("Expected result text in output, got:\n%s", result.ForLLM)
	}
}

func TestSpawnTool_ActionList_Empty(t *testing.T) {
	tool, _, _ := newTestSpawnTool(t)

	result := tool.Execute(context.Background(), map[string]any{"action": "list"})
	if result.IsError {
		t.Fatalf("Expected success, got error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "No subagents") {
		t.Errorf("Expected 'No subagents' message, got: %s", result.ForLLM)
	}
}

func TestSpawnTool_ActionList_StatusCounts(t *testing.T) {
	tool, manager, _ := newTestSpawnTool(t)

	manager.mu.Lock()
	for i, status := range []string{"running", "running", "completed", "failed", "canceled"} {
		id := fmt.Sprintf("subagent-%d", i+1)
		manager.tasks[id] = &SubagentTask{ID: id, Task: "t", Status: status}
	}
	manager.mu.Unlock()

	result := tool.Execute(context.Background(), map[string]any{"action": "list"})
	if result.IsError {
		t.Fatalf("Unexpected error: %s", result.ForLLM)
	}
	for _, want := range []string{"Running:", "Completed:", "Failed:", "Canceled:"} {
		if !strings.Contains(result.ForLLM, want) {
			t.Errorf("Expected %q in summary, got:\n%s", want, result.ForLLM)
		}
	}
}

func TestSpawnTool_ActionList_SortByCreatedTimestamp(t *testing.T) {
	tool, manager, _ := newTestSpawnTool(t)

	now := time.Now().UnixMilli()
	manager.mu.Lock()
	// subagent-2 was spawned first, subagent-10 second.
	manager.tasks["subagent-10"] = &SubagentTask{
		ID: "subagent-10", Task: "second", Status: "running", Created: now + 1,
	}
	manager.tasks["subagent-2"] = &SubagentTask{
		ID: "subagent-2", Task: "first", Status: "running", Created: now,
	}
	manager.mu.Unlock()

	result := tool.Execute(context.Background(), map[string]any{"action": "list"})
	if result.IsError {
		t.Fatalf("Unexpected error: %s", result.ForLLM)
	}

	pos2 := strings.Index(result.ForLLM, "subagent-2")
	pos10 := strings.Index(result.ForLLM, "subagent-10")
	if pos2 < 0 || pos10 < 0 {
		t.Fatalf("Both task IDs should appear in output:\n%s", result.ForLLM)
	}
	if pos2 > pos10 {
		t.Errorf("Expected subagent-2 (created first) before subagent-10, got:\n%s", result.ForLLM)
	}
}

func TestSpawnTool_ActionList_ChannelFiltering(t *testing.T) {
	tool, manager, _ := newTestSpawnTool(t)

	manager.mu.Lock()
	manager.tasks["subagent-1"] = &SubagentTask{
		ID: "subagent-1", Task: "mine", Status: "running",
		OriginChannel: "telegram", OriginChatID: "chat-A",
	}
	manager.tasks["subagent-2"] = &SubagentTask{
		ID: "subagent-2", Task: "other user", Status: "running",
		OriginChannel: "telegram", OriginChatID: "chat-B",
	}
	manager.mu.Unlock()

	ctx := WithToolContext(context.Background(), "telegram", "chat-A")
	result := tool.Execute(ctx, map[string]any{"action": "list"})
	if result.IsError {
		t.Fatalf("Unexpected error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "subagent-1") {
		t.Errorf("Expected own task in output, got:\n%s", result.ForLLM)
	}
	if strings.Contains(result.ForLLM, "subagent-2") {
		t.Errorf("Should NOT see other chat's task, got:\n%s", result.ForLLM)
	}
}

func TestSpawnTool_ActionStatus_ChannelFiltering(t *testing.T) {
	tool, manager, _ := newTestSpawnTool(t)

	manager.mu.Lock()
	manager.tasks["subagent-99"] = &SubagentTask{
		ID: "subagent-99", Task: "secret", Status: "completed", Result: "private data",
		OriginChannel: "slack", OriginChatID: "room-Z",
	}
	manager.mu.Unlock()

	ctx := WithToolContext(context.Background(), "slack", "room-OTHER")
	result := tool.Execute(ctx, map[string]any{"action": "status", "task_id": "subagent-99"})
	if !result.IsError {
		t.Errorf("Expected error (cross-chat lookup blocked), got: %s", result.ForLLM)
	}
}

func TestSpawnTool_ResultTruncation(t *testing.T) {
	tool, manager, _ := newTestSpawnTool(t)

	longResult := strings.Repeat("X", 500)
	manager.mu.Lock()
	manager.tasks["subagent-1"] = &SubagentTask{
		ID: "subagent-1", Task: "Long task", Status: "completed", Result: longResult,
	}
	manager.mu.Unlock()

	result := tool.Execute(context.Background(), map[string]any{"action": "status", "task_id": "subagent-1"})
	if result.IsError {
		t.Fatalf("Unexpected error: %s", result.ForLLM)
	}
	if len(result.ForLLM) >= len(longResult) {
		t.Errorf("Expected result to be truncated, but ForLLM is %d chars", len(result.ForLLM))
	}
	if !strings.Contains(result.ForLLM, "…") {
		t.Errorf("Expected truncation indicator '…' in output, got: %s", result.ForLLM)
	}
}

func TestSpawnTool_ResultTruncationUnicode(t *testing.T) {
	tool, manager, _ := newTestSpawnTool(t)

	cjkChar := string(rune(0x5b57))
	longResult := strings.Repeat(cjkChar, 400)
	manager.mu.Lock()
	manager.tasks["subagent-1"] = &SubagentTask{
		ID: "subagent-1", Task: "Unicode task", Status: "completed", Result: longResult,
	}
	manager.mu.Unlock()

	result := tool.Execute(context.Background(), map[string]any{"action": "status", "task_id": "subagent-1"})
	if result.IsError {
		t.Fatalf("Unexpected error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "…") {
		t.Error("Expected truncation indicator in output")
	}
	if !strings.Contains(result.ForLLM, cjkChar) {
		t.Error("Expected CJK runes to appear intact in output")
	}
}

func TestSpawnTool_InvalidAction(t *testing.T) {
	tool, _, _ := newTestSpawnTool(t)

	for _, args := range []map[string]any{
		{"action": "bogus", "task": "x"},
		{"action": 42, "task": "x"},
	} {
		result := tool.Execute(context.Background(), args)
		if !result.IsError {
			t.Errorf("Expected error for action=%v, got: %s", args["action"], result.ForLLM)
		}
		if !strings.Contains(result.ForLLM, "action") {
			t.Errorf("Expected action in error message, got: %s", result.ForLLM)
		}
	}
}

func TestSpawnTool_DisabledActions(t *testing.T) {
	tool, manager, _ := newTestSpawnTool(t)
	tool.SetEnabledActions(false, true)

	result := tool.Execute(context.Background(), map[string]any{"task": "x"})
	if !result.IsError || !strings.Contains(result.ForLLM, "spawn") {
		t.Errorf("Expected spawn to be disabled, got: %s", result.ForLLM)
	}

	manager.mu.Lock()
	manager.tasks["subagent-1"] = &SubagentTask{ID: "subagent-1", Status: "running"}
	manager.mu.Unlock()

	result = tool.Execute(context.Background(), map[string]any{"action": "list"})
	if result.IsError {
		t.Errorf("Expected list to remain enabled, got error: %s", result.ForLLM)
	}

	tool.SetEnabledActions(true, false)
	result = tool.Execute(context.Background(), map[string]any{"action": "list"})
	if !result.IsError || !strings.Contains(result.ForLLM, "disabled") {
		t.Errorf("Expected status actions to be disabled, got: %s", result.ForLLM)
	}
	result = tool.Execute(context.Background(), map[string]any{
		"action":  "status",
		"task_id": "subagent-1",
	})
	if !result.IsError || !strings.Contains(result.ForLLM, "disabled") {
		t.Errorf("Expected status action to be disabled, got: %s", result.ForLLM)
	}
}

func TestSpawnTool_AllowlistBlocked(t *testing.T) {
	tool, _, _ := newTestSpawnTool(t)
	tool.SetAllowlistChecker(func(targetAgentID string) bool { return false })

	result := tool.Execute(context.Background(), map[string]any{
		"task":     "x",
		"agent_id": "forbidden",
	})
	if !result.IsError {
		t.Fatalf("Expected allowlist error, got: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "not allowed") {
		t.Errorf("Expected allowlist message, got: %s", result.ForLLM)
	}
}

func TestSpawnTool_Execute_NilManager(t *testing.T) {
	tool := NewSpawnTool(nil)

	result := tool.Execute(context.Background(), map[string]any{"task": "test task"})
	if !result.IsError {
		t.Error("Expected error for nil manager")
	}
	if !strings.Contains(result.ForLLM, "Subagent manager not configured") {
		t.Errorf("Error message should mention manager not configured, got: %s", result.ForLLM)
	}

	result = tool.Execute(context.Background(), map[string]any{"action": "list"})
	if !result.IsError || !strings.Contains(result.ForLLM, "Subagent manager not configured") {
		t.Errorf("Expected manager error for list, got: %s", result.ForLLM)
	}
}

func TestSubagentManager_SetLLMOptions_AppliesToRunToolLoop(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "test-model", "/tmp/test")
	manager.SetLLMOptions(2048, 0.6)

	if manager.maxTokens != 2048 {
		t.Errorf("manager.maxTokens = %d, want 2048", manager.maxTokens)
	}
	if manager.temperature != 0.6 {
		t.Errorf("manager.temperature = %f, want 0.6", manager.temperature)
	}
	if !manager.hasMaxTokens {
		t.Error("manager.hasMaxTokens should be true")
	}
	if !manager.hasTemperature {
		t.Error("manager.hasTemperature should be true")
	}
}
