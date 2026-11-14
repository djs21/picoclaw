## Why

picoclaw's `ContextManager` is pluggable (`pkg/agent/context_manager.go:15`) but ships only two implementations: `legacy` (hardcoded fallback, `pkg/agent/context_legacy.go`) and `seahorse` (SQLite-backed, platform-gated, `pkg/agent/context_seahorse.go`). There is no deterministic, pure-Go, config-selectable third option. Operators who want structured, auditable summaries (session goals, files changed, commits, outstanding context, user preferences, brief transcript) without C/SQLite dependencies or platform build gates have no supported path.

`context_manager` is a free-form string with no enum/validation (`pkg/config/config.go:445`), and `resolveContextManager` silently falls back to `legacy` on unknown/failing names (`pkg/agent/turn_coord.go:342-363`). This makes adding `"vcc"` safe: a typo degrades to legacy instead of breaking the turn.

## What Changes

Add an optional third `ContextManager` named `"vcc"`, selected through the existing `ContextManager: "vcc"` config field plus `ContextManagerConfig` passthrough (`turn_coord.go:354`). VCC never replaces `legacy` or `seahorse`; it is opt-in only.

New code:
- `pkg/vcc/` — pure-Go deterministic compiler: types, normalize, filter-noise, sections, brief/format, merge/token-estimate, settings. No picoclaw imports; no C/SQLite deps; `-tags goolm,stdjson` compatible.
- `pkg/agent/context_vcc.go` — thin adapter implementing `ContextManager` + `RegisterContextManager("vcc", ...)` (in package `agent` so it can call unexported `AgentLoop` helpers: `emitEvent`, `newTurnEventScope`, `registry`).

No changes to legacy/seahorse, no config enum/validation, no new storage backend (reads/writes history & summary via the existing `session.SessionStore`).

## Capabilities

### New Capabilities
- `vcc-context-manager`: deterministic context summarization (SectionData → bracketed-heading summary) exposed as a `"vcc"` ContextManager via the existing registry and config fields.

### Modified Capabilities
- (none)

## Impact

- New files only; zero existing source modified.
- Runtime: selecting `context_manager: "vcc"` routes `Assemble`/`Compact`/`Ingest`/`Clear` through VCC; unknown names still fall back to legacy (`turn_coord.go:347-352`).
- Events: VCC emits `KindAgentContextCompress` / `KindAgentSessionSummarize` (`pkg/events/kind.go:19,21`) for dashboard parity with legacy (`context_legacy.go:45,252`).
- Build: pure Go; no `goolm` C dependency, no platform gate (unlike seahorse `context_seahorse.go:1` / `context_seahorse_unsupported.go:1`).
