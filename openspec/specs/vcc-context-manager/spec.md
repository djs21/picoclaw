# vcc-context-manager Specification

## Purpose

Add a deterministic, optional `"vcc"` ContextManager to picoclaw that compiles conversation history into a structured summary (session goals, files/changes, commits, outstanding context, user preferences, brief transcript) and exposes it through the existing pluggable `ContextManager` registry and config fields. New capability; deletes nothing.

## Requirements

### Requirement: REQ-1 Selection via existing config
**REQ-1 SHALL** select `ContextManager` named `"vcc"` through the existing `ContextManager` string field (`pkg/config/config.go:445`) with `ContextManagerConfig` passthrough (`config.go:446`), without adding validation/enum.

#### Scenario: Select vcc by name
- **WHEN** `ContextManager: "vcc"` is set in config
- **THEN** `resolveContextManager()` (`pkg/agent/turn_coord.go:342-363`) looks up the `"vcc"` factory in `cmRegistry` and invokes it with `ContextManagerConfig` and `*AgentLoop` (`turn_coord.go:354`)

#### Scenario: Unknown name still falls back to legacy
- **WHEN** `ContextManager` is unset or names an unregistered manager
- **THEN** `resolveContextManager()` returns `&legacyContextManager{al: al}` with a warning (`turn_coord.go:344-352`)

#### Scenario: Factory error falls back to legacy
- **WHEN** the `"vcc"` factory returns an error
- **THEN** `resolveContextManager()` logs a warning and returns `&legacyContextManager{al: al}` (`turn_coord.go:355-360`)

### Requirement: REQ-2 Registration
**REQ-2 SHALL** register the `"vcc"` factory under the name `"vcc"` via `RegisterContextManager("vcc", factory)` (`pkg/agent/context_manager.go:70`) before `NewAgentLoop` runs.

#### Scenario: Self-registration via init
- **WHEN** the `context_vcc` package is imported into the agent binary
- **THEN** its `init()` calls `RegisterContextManager("vcc", newVCCContextManager)` (mirrors `pkg/agent/context_seahorse.go:299-302`)

#### Scenario: Duplicate registration rejected
- **WHEN** `RegisterContextManager("vcc", ...)` is called twice
- **THEN** the second call returns an "already registered" error (`context_manager.go:81-83`)

### Requirement: REQ-3 Assemble (deterministic summary)
**REQ-3 SHALL** read session history/summary via `Assemble` and return `AssembleResponse{History, Summary}` (`context_manager.go:41-44`), built by the deterministic VCC compiler.

#### Scenario: Assemble returns history + summary
- **WHEN** `Assemble(ctx, &AssembleRequest{SessionKey, Budget, MaxTokens})` is called
- **THEN** returns non-nil `*AssembleResponse` with `History []providers.Message` and `Summary string` (`context_manager.go:41-44`)

#### Scenario: Assemble respects budget
- **WHEN** `Budget` and `MaxTokens` are provided
- **THEN** effective budget = `Budget - MaxTokens` (mirroring `context_seahorse.go:102-110`) and `estimateTokensFromChars(Summary + History)` is ≤ `effectiveBudget`.

#### Scenario: Nil request rejected
- **WHEN** `Assemble` receives a nil `*AssembleRequest`
- **THEN** returns a non-nil error (mirrors seahorse `context_seahorse.go:93-95`)

### Requirement: REQ-4 Compact (summarize + truncate)
**REQ-4 SHALL** compress conversation history via `Compact`: build a fresh summary via the VCC compiler, merge with any previous summary, persist summary + truncated history via `session.SessionStore`.

#### Scenario: Compact writes summary and truncates history
- **WHEN** `Compact(ctx, &CompactRequest{SessionKey, Reason, Budget})` runs
- **THEN** calls `SetSummary`, `TruncateHistory`, and `Save` on the session store (`session_store.go:23,27,29`)

#### Scenario: Compact emits summarize event
- **WHEN** summarization completes
- **THEN** emits `KindAgentSessionSummarize` with `SessionSummarizePayload` (`pkg/events/kind.go:21`; `pkg/agent/event_payloads.go:109-114`)

#### Scenario: Compact on retry uses aggressive budget
- **WHEN** `Reason == ContextCompressReasonRetry` (`event_payloads.go:96`) and `Budget > 0`
- **THEN** compacts aggressively to fit under budget (mirrors seahorse `CompactUntilUnder` path, `context_seahorse.go:136-138`)

#### Scenario: Nil compact request is a no-op
- **WHEN** `Compact` receives a nil `*CompactRequest`
- **THEN** returns nil (mirrors seahorse `context_seahorse.go:130-132`)

### Requirement: REQ-5 Ingest (record message)
**REQ-5 SHALL** record a newly persisted message into VCC's per-session working set used by Assemble.

#### Scenario: Ingest records the message
- **WHEN** `Ingest(ctx, &IngestRequest{SessionKey, Message})` is called after a message is persisted to session JSONL
- **THEN** VCC SHALL append the provider message to its in-memory per-session working set used by Assemble, or explicitly return nil with documented rationale that Assemble reads live history

#### Scenario: Nil ingest request is a no-op
- **WHEN** `Ingest` receives a nil `*IngestRequest`
- **THEN** returns nil (mirrors seahorse `context_seahorse.go:151-153`)

### Requirement: REQ-6 Clear (reset session context)
**REQ-6 SHALL** remove all stored context for a session via `Clear`.

#### Scenario: Clear resets history and summary
- **WHEN** `Clear(ctx, sessionKey)` is called (e.g. `/clear` or `/reset`)
- **THEN** calls `SetHistory(sessionKey, {})`, `SetSummary(sessionKey, "")`, and `Save(sessionKey)` (`session_store.go:25,23,29`) mirroring legacy `Clear` (`context_legacy.go:65-75`)

### Requirement: REQ-7 Deterministic sections
**REQ-7 SHALL** produce `SectionData` with deterministic extraction rules (goals, files/changes, commits, outstanding context, preferences, brief transcript) via the VCC compiler.

#### Scenario: SectionData has all six fields
- **WHEN** the compiler runs over a message set
- **THEN** `SectionData` contains `SessionGoal`, `FilesAndChanges`, `Commits`, `OutstandingContext`, `UserPreferences`, `BriefTranscript` (algo spec §2.2)

#### Scenario: Section caps are respected
- **WHEN** extracting goals / commits / preferences
- **THEN** goals ≤ 8 entries (≤200 chars each), commits ≤ 8, preferences ≤ 10 (algo spec §5 constants)

#### Scenario: Outstanding context is volatile (fresh-only on merge)
- **WHEN** merging a previous summary with a fresh one
- **THEN** `OutstandingContext` comes from fresh only (algo spec `summarize.ts:56`)

#### Scenario: Brief transcript capped
- **WHEN** rendering `BriefTranscript`
- **THEN` capped at `BRIEF_MAX_LINES` (120) lines (`algo spec format.ts:9`)

### Requirement: REQ-8 Merge semantics
**REQ-8 SHALL** merge a previous summary with a fresh one: preserve sticky sections, take fresh-only for volatile sections, dedup line-level, and append RECALL_NOTE once.

#### Scenario: Sticky sections merge prev + fresh
- **WHEN** merging Session Goal / Commits / User Preferences / Files And Changes
- **THEN** result = line-level dedup of prev then fresh, capped per section (algo spec `summarize.ts:145`)

#### Scenario: RECALL_NOTE appended once after merge
- **WHEN** producing the final summary
- **THEN** `RECALL_NOTE` is appended once after merge-with-previous, not inside the brief transcript (algo spec `summarize.ts:179-191`)

#### Scenario: Previous RECALL_NOTE stripped before merge
- **WHEN** a previous summary contains a trailing RECALL_NOTE
- **THEN** it is stripped via `stripRecallNote` before merging (algo spec `summarize.ts:207`)

### Requirement: REQ-9 Event emission
**REQ-9 SHALL** emit dashboard-compatible events for compress and summarize.

#### Scenario: Compress event emitted on force compression
- **WHEN** emergency compression runs during Compact
- **THEN** emits `KindAgentContextCompress` with `ContextCompressPayload{Reason, DroppedMessages, RemainingMessages}` (`pkg/events/kind.go:19`; `event_payloads.go:102-106`)

#### Scenario: Summarize event emitted after summarization
- **WHEN** post-turn summarization completes
- **THEN** emits `KindAgentSessionSummarize` with `SessionSummarizePayload` (`pkg/events/kind.go:21`; `event_payloads.go:109-114`)

### Requirement: REQ-10 Error and fallback behavior
**REQ-10 SHALL** never hard-fail a turn; storage errors are logged, not returned as fatal.

#### Scenario: Save error is log-only
- **WHEN** `session.SessionStore.Save` returns an error during Compact/Clear
- **THEN** VCC logs it and does not fail the turn (fire-and-forget contract, `session_store.go:10-12`)

#### Scenario: Unknown factory degrades to legacy
- **WHEN** `"vcc"` factory is not registered
- **THEN** turn proceeds with legacy context manager and a warning (`turn_coord.go:348-352`)
