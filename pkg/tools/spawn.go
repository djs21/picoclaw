package tools

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

// SpawnTool is the unified subagent tool. It can spawn a subagent either in
// the background (default) or synchronously, and it can report the status of
// previously spawned subagents via the task registry owned by SubagentManager.
//
// The three legacy tools (spawn, subagent, spawn_status) all map onto this
// single tool:
//
//	action=spawn  (default)  spawn a new subagent
//	action=status            inspect one task via task_id
//	action=list              list tasks visible to the current conversation
//
// The sync flag only applies to action=spawn.
type SpawnTool struct {
	spawner        SubTurnSpawner
	manager        *SubagentManager
	defaultModel   string
	maxTokens      int
	temperature    float64
	allowlistCheck func(targetAgentID string) bool

	spawnActionEnabled  bool
	statusActionEnabled bool
}

// Compile-time check: SpawnTool implements AsyncExecutor.
var _ AsyncExecutor = (*SpawnTool)(nil)

const (
	spawnActionSpawn  = "spawn"
	spawnActionStatus = "status"
	spawnActionList   = "list"
)

// NewSpawnTool creates a unified spawn tool. The manager supplies model
// defaults and the task registry used by the status/list actions.
func NewSpawnTool(manager *SubagentManager) *SpawnTool {
	tool := &SpawnTool{
		manager:             manager,
		spawnActionEnabled:  true,
		statusActionEnabled: true,
	}
	if manager == nil {
		return tool
	}
	tool.defaultModel = manager.defaultModel
	tool.maxTokens = manager.maxTokens
	tool.temperature = manager.temperature
	return tool
}

// SetSpawner sets the SubTurnSpawner for direct sub-turn execution.
func (t *SpawnTool) SetSpawner(spawner SubTurnSpawner) {
	t.spawner = spawner
}

// SetAllowlistChecker sets the agent spawn allowlist checker.
func (t *SpawnTool) SetAllowlistChecker(check func(targetAgentID string) bool) {
	t.allowlistCheck = check
}

// SetEnabledActions controls which actions the tool advertises and accepts.
// Status covers both the "status" and "list" actions.
func (t *SpawnTool) SetEnabledActions(spawn, status bool) {
	t.spawnActionEnabled = spawn
	t.statusActionEnabled = status
}

func (t *SpawnTool) Name() string {
	return "spawn"
}

func (t *SpawnTool) Description() string {
	return "Spawn and monitor subagents. " +
		"Use action \"spawn\" (default) to delegate a task to a subagent, either in the " +
		"background (sync=false) or synchronously (sync=true). " +
		"Use action \"status\" with task_id to inspect a specific subagent task, or " +
		"action \"list\" to list the tasks visible to the current conversation with " +
		"their state (running, completed, failed, or canceled)."
}

func (t *SpawnTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task": map[string]any{
				"type": "string",
				"description": "Task for the subagent to complete. Required when " +
					"action is \"spawn\" (the default) and ignored otherwise.",
			},
			"label": map[string]any{
				"type":        "string",
				"description": "Optional short label for the task (for display).",
			},
			"agent_id": map[string]any{
				"type":        "string",
				"description": "Optional target agent ID to delegate the task to.",
			},
			"sync": map[string]any{
				"type": "boolean",
				"description": "When true, wait for the subagent to finish and return its " +
					"result directly. When false (default), run in the background and " +
					"return a task ID. Only applies to action \"spawn\".",
			},
			"action": map[string]any{
				"type": "string",
				"enum": []string{spawnActionSpawn, spawnActionStatus, spawnActionList},
				"description": "Action to perform. \"spawn\" (default) starts a subagent, " +
					"\"status\" inspects one task via task_id, and \"list\" lists visible tasks.",
			},
			"task_id": map[string]any{
				"type": "string",
				"description": "Task ID (e.g. \"subagent-1\") to inspect. Required when " +
					"action is \"status\".",
			},
		},
		"required": []string{},
	}
}

func (t *SpawnTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	return t.execute(ctx, args, nil)
}

// ExecuteAsync implements AsyncExecutor. The callback is passed through to the
// background subagent goroutine as a call parameter, never stored on the tool.
func (t *SpawnTool) ExecuteAsync(
	ctx context.Context,
	args map[string]any,
	cb AsyncCallback,
) *ToolResult {
	return t.execute(ctx, args, cb)
}

func (t *SpawnTool) execute(
	ctx context.Context,
	args map[string]any,
	cb AsyncCallback,
) *ToolResult {
	if t.manager == nil {
		return ErrorResult("Subagent manager not configured").WithError(
			fmt.Errorf("subagent manager not set"),
		)
	}

	action, err := t.parseAction(args)
	if err != nil {
		return ErrorResult(err.Error()).WithError(err)
	}

	switch action {
	case spawnActionStatus:
		if !t.statusActionEnabled {
			return ErrorResult("spawn tool action \"status\" is disabled")
		}
		return t.executeStatus(ctx, args)
	case spawnActionList:
		if !t.statusActionEnabled {
			return ErrorResult("spawn tool action \"list\" is disabled")
		}
		return t.executeList(ctx)
	default:
		if !t.spawnActionEnabled {
			return ErrorResult("spawn tool action \"spawn\" is disabled")
		}
		return t.executeSpawn(ctx, args, cb)
	}
}

func (t *SpawnTool) parseAction(args map[string]any) (string, error) {
	raw, ok := args["action"]
	if !ok || raw == nil {
		return spawnActionSpawn, nil
	}
	action, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("action must be a string")
	}
	switch strings.TrimSpace(action) {
	case "", spawnActionSpawn:
		return spawnActionSpawn, nil
	case spawnActionStatus:
		return spawnActionStatus, nil
	case spawnActionList:
		return spawnActionList, nil
	default:
		return "", fmt.Errorf(
			"invalid action %q: expected one of %q, %q, %q",
			action, spawnActionSpawn, spawnActionStatus, spawnActionList,
		)
	}
}

func (t *SpawnTool) executeSpawn(
	ctx context.Context,
	args map[string]any,
	cb AsyncCallback,
) *ToolResult {
	task, ok := args["task"].(string)
	if !ok || strings.TrimSpace(task) == "" {
		return ErrorResult("task is required and must be a non-empty string")
	}

	label, _ := args["label"].(string)
	agentID, _ := args["agent_id"].(string)
	targetAgentID := strings.TrimSpace(agentID)

	sync, err := parseBoolArg(args, "sync")
	if err != nil {
		return ErrorResult(err.Error()).WithError(err)
	}

	// Check the agent allowlist before creating any task state.
	if targetAgentID != "" && t.allowlistCheck != nil {
		if !t.allowlistCheck(targetAgentID) {
			return ErrorResult(fmt.Sprintf("not allowed to spawn agent '%s'", targetAgentID))
		}
	}

	if t.spawner == nil {
		return ErrorResult("Subagent manager not configured").WithError(
			fmt.Errorf("spawner not set"),
		)
	}

	registered := t.manager.CreateTask(task, label, targetAgentID, ToolChannel(ctx), ToolChatID(ctx))
	cfg := SubTurnConfig{
		Model:         t.defaultModel,
		Tools:         nil, // Inherited from the parent via context.
		SystemPrompt:  buildSubagentSystemPrompt(task, label, sync),
		MaxTokens:     t.maxTokens,
		Temperature:   t.temperature,
		Async:         !sync,
		Critical:      !sync, // Background spawn should survive parent turn completion.
		TargetAgentID: targetAgentID,
	}

	if sync {
		result, err := t.spawner.SpawnSubTurn(ctx, cfg)
		if err != nil {
			t.manager.FinishTask(registered.ID, taskStatusForError(ctx, err), fmt.Sprintf("Error: %v", err))
			return ErrorResult(fmt.Sprintf("Subagent execution failed: %v", err)).WithError(err)
		}
		if result == nil {
			t.manager.FinishTask(registered.ID, "failed", "Error: subagent returned no result")
			return ErrorResult("Subagent execution returned no result")
		}
		t.manager.FinishTask(registered.ID, taskStatusCompleted, result.ForLLM)
		return formatSyncSpawnResult(registered.ID, label, result)
	}

	go func() {
		result, err := t.spawner.SpawnSubTurn(ctx, cfg)
		if err != nil {
			t.manager.FinishTask(registered.ID, taskStatusForError(ctx, err), fmt.Sprintf("Error: %v", err))
			result = ErrorResult(fmt.Sprintf("Spawn failed: %v", err)).WithError(err)
		} else if result == nil {
			t.manager.FinishTask(registered.ID, "failed", "Error: subagent returned no result")
			result = ErrorResult("Spawn failed: subagent returned no result")
		} else {
			t.manager.FinishTask(registered.ID, taskStatusCompleted, result.ForLLM)
		}

		if cb != nil {
			cb(ctx, result)
		}
	}()

	message := fmt.Sprintf("Spawned subagent (task ID: %s) for task: %s", registered.ID, task)
	if label != "" {
		message = fmt.Sprintf("Spawned subagent '%s' (task ID: %s) for task: %s", label, registered.ID, task)
	}
	return AsyncResult(message)
}

func (t *SpawnTool) executeStatus(ctx context.Context, args map[string]any) *ToolResult {
	rawTaskID, ok := args["task_id"]
	if !ok || rawTaskID == nil {
		return ErrorResult("task_id is required for action \"status\"")
	}
	taskIDStr, ok := rawTaskID.(string)
	if !ok {
		return ErrorResult("task_id must be a string")
	}
	taskID := strings.TrimSpace(taskIDStr)
	if taskID == "" {
		return ErrorResult("task_id is required for action \"status\"")
	}

	// GetTaskCopy returns a consistent snapshot under the manager lock.
	task, ok := t.manager.GetTaskCopy(taskID)
	if !ok || !taskVisibleToContext(ctx, &task) {
		return ErrorResult(fmt.Sprintf("No subagent found with task ID: %s", taskID))
	}

	return NewToolResult(formatSubagentTask(&task))
}

func (t *SpawnTool) executeList(ctx context.Context) *ToolResult {
	// ListTaskCopies returns consistent snapshots under the manager lock.
	copies := t.manager.ListTaskCopies()
	if len(copies) == 0 {
		return NewToolResult("No subagents have been spawned yet.")
	}

	tasks := make([]*SubagentTask, 0, len(copies))
	for i := range copies {
		if taskVisibleToContext(ctx, &copies[i]) {
			tasks = append(tasks, &copies[i])
		}
	}
	if len(tasks) == 0 {
		return NewToolResult("No subagents found for this conversation.")
	}

	// Order by creation time so spawning order is preserved. Fall back to the
	// ID string for tasks created in the same millisecond.
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].Created != tasks[j].Created {
			return tasks[i].Created < tasks[j].Created
		}
		return tasks[i].ID < tasks[j].ID
	})

	counts := map[string]int{}
	for _, task := range tasks {
		counts[task.Status]++
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Subagent status report (%d total):\n", len(tasks)))
	for _, status := range []string{"running", "completed", "failed", "canceled"} {
		if n := counts[status]; n > 0 {
			label := strings.ToUpper(status[:1]) + status[1:] + ":"
			sb.WriteString(fmt.Sprintf("  %-10s %d\n", label, n))
		}
	}
	sb.WriteString("\n")

	for _, task := range tasks {
		sb.WriteString(formatSubagentTask(task))
		sb.WriteString("\n\n")
	}

	return NewToolResult(strings.TrimRight(sb.String(), "\n"))
}

const (
	taskStatusCompleted = "completed"
	taskStatusFailed    = "failed"
	taskStatusCanceled  = "canceled"
)

func taskStatusForError(ctx context.Context, err error) string {
	if ctx.Err() != nil {
		return taskStatusCanceled
	}
	return taskStatusFailed
}

func parseBoolArg(args map[string]any, name string) (bool, error) {
	raw, ok := args[name]
	if !ok || raw == nil {
		return false, nil
	}
	value, ok := raw.(bool)
	if !ok {
		return false, fmt.Errorf("%s must be a boolean", name)
	}
	return value, nil
}

func buildSubagentSystemPrompt(task, label string, sync bool) string {
	behavior := "running in the background"
	closing := "report back when done"
	if sync {
		behavior = "running synchronously"
		closing = "provide a clear, concise result"
	}

	if label != "" {
		return fmt.Sprintf(
			`You are a spawned subagent labeled "%s" %s. Complete the given task independently and %s.

Task: %s`,
			label, behavior, closing, task,
		)
	}
	return fmt.Sprintf(
		`You are a spawned subagent %s. Complete the given task independently and %s.

Task: %s`,
		behavior, closing, task,
	)
}

func formatSyncSpawnResult(taskID, label string, result *ToolResult) *ToolResult {
	userContent := result.ForLLM
	if result.ForUser != "" {
		userContent = result.ForUser
	}
	const maxUserLen = 500
	if len(userContent) > maxUserLen {
		userContent = userContent[:maxUserLen] + "..."
	}

	labelStr := label
	if labelStr == "" {
		labelStr = "(unnamed)"
	}

	return &ToolResult{
		ForLLM: fmt.Sprintf(
			"Subagent task completed:\nTask ID: %s\nLabel: %s\nResult: %s",
			taskID, labelStr, result.ForLLM,
		),
		ForUser: userContent,
		Silent:  false,
		IsError: result.IsError,
		Async:   false,
	}
}

// taskVisibleToContext reports whether task belongs to the conversation that
// issued the tool call. Empty caller or task context is treated as visible so
// direct programmatic calls keep working.
func taskVisibleToContext(ctx context.Context, task *SubagentTask) bool {
	callerChannel := ToolChannel(ctx)
	callerChatID := ToolChatID(ctx)
	if callerChannel != "" && task.OriginChannel != "" && task.OriginChannel != callerChannel {
		return false
	}
	if callerChatID != "" && task.OriginChatID != "" && task.OriginChatID != callerChatID {
		return false
	}
	return true
}

// formatSubagentTask renders a single SubagentTask as a human-readable block.
func formatSubagentTask(task *SubagentTask) string {
	var sb strings.Builder

	header := fmt.Sprintf("[%s] status=%s", task.ID, task.Status)
	if task.Label != "" {
		header += fmt.Sprintf("  label=%q", task.Label)
	}
	if task.AgentID != "" {
		header += fmt.Sprintf("  agent=%s", task.AgentID)
	}
	if task.Created > 0 {
		created := time.UnixMilli(task.Created).UTC().Format("2006-01-02 15:04:05 UTC")
		header += fmt.Sprintf("  created=%s", created)
	}
	sb.WriteString(header)

	if task.Task != "" {
		sb.WriteString(fmt.Sprintf("\n  task:   %s", task.Task))
	}
	if task.Result != "" {
		result := task.Result
		const maxResultLen = 300
		runes := []rune(result)
		if len(runes) > maxResultLen {
			result = string(runes[:maxResultLen]) + "…"
		}
		sb.WriteString(fmt.Sprintf("\n  result: %s", result))
	}

	return sb.String()
}
