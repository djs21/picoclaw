## Context

Contract-map anchors (all read-only, verified):
- `ContextManager` interface + 4 request/response structs + `ContextManagerFactory` + registry: `pkg/agent/context_manager.go:15-94`.
- Selection `resolveContextManager()`: `pkg/agent/turn_coord.go:342-363`; assembly call site `turn_coord.go:406-414`; init `pkg/agent/agent_init.go:99`.
- `RegisterContextManager` call sites: `context_manager.go:70` (def), `context_seahorse.go:300` (init), `context_seahorse_unsupported.go:17` (stub init).
- Config fields: `pkg/config/config.go:445-446` (`ContextManager string`, `ContextManagerConfig json.RawMessage`). No validation/enum; `defaults.go` sets nothing; `config.example.json` has no entry.
- `session.SessionStore` interface: `pkg/session/session_store.go:13-33` (fire-and-forget writes); `memory.Store`: `pkg/memory/store.go:11-45`; `JSONLBackend`: `pkg/session/jsonl_backend.go`.
- Events: `pkg/events/kind.go:19,21`; payloads `pkg/agent/event_payloads.go:90-114`.
- Tests/seams: `pkg/agent/context_manager_test.go` — `resetCMRegistry()` line 843, `newCMTestAgentLoop` line 873, `noopContextManager` line 787, `trackingContextManager` line 797.
- Build tags: `Makefile:38-39` (`goolm,stdjson`); seahorse is platform-gated (`context_seahorse.go:1`, `context_seahorse_unsupported.go:1`).

Source algorithm reference: `/tmp/picoclaw-vcc-algo-spec.md` (pi-vcc 0.8.0) — pipeline normalize → filterNoise → buildSections → formatSummary → mergePrevious → append RECALL_NOTE; RECALL_NOTE appended once after merge to avoid compounding (`summarize.ts:179-191`).

## Goals / Non-Goals

**Goals:**
- Pure-Go deterministic compiler package `pkg/vcc/` producing `SectionData` (goals, files/changes, commits, outstanding context, preferences, brief transcript).
- Thin `ContextManager` adapter `pkg/agent/context_vcc.go` registered as `"vcc"`, selected via existing config fields, optional (never replaces legacy/seahorse).
- Every production/test file < 400 LOC (measurable LOC budget); zero-overlap file ownership with git worktree isolation per worker and serialized merge gates so ≥3 workers implement in parallel.
- `-tags goolm,stdjson` compatible; no C/SQLite deps; no platform build gate.
- Dashboard-compatible event emission (`KindAgentContextCompress`, `KindAgentSessionSummarize`).
- Reuse existing `session.SessionStore` for history/summary reads/writes; no new storage backend.

**Non-Goals:**
- Make VCC the default or remove legacy/seahorse.
- Semantic recall / local-model rerank (`vcc_recall` phase 2 per algo spec §3.8).
- Config enum/validation for `ContextManager`.
- New database or storage backend beyond `SessionStore`.

## Decisions

**D1 — Package split.** Pure compiler in `pkg/vcc/` (no picoclaw imports); thin adapter `pkg/agent/context_vcc.go` in package `agent` (needs unexported `AgentLoop` helpers: `emitEvent`, `newTurnEventScope`, `registry`, `agent.Sessions`, `agent.Provider`). Adapter imports `pkg/vcc` for compilation only.

**D2 — Opt-in, not replacement.** `"vcc"` is a third option selected by `ContextManager: "vcc"`. Legacy remains the default/fallback (`turn_coord.go:344`). No behavior change for existing users.

**D3 — Self-registration via `init()`.** Adapter calls `RegisterContextManager("vcc", newVCCContextManager)` in `init()` (mirrors seahorse `context_seahorse.go:299-302`). Registered before `NewAgentLoop` runs via package import in the agent binary.

**D4 — Pure Go, no build gate.** Unlike seahorse (platform-gated, requires SQLite/olm), VCC is pure Go with no `goolm` dependency. Works under `goolm,stdjson` and without any extra tag.

**D5 — Storage via existing SessionStore.** VCC `Assemble` reads history via `GetHistory`/`GetSummary`; `Compact` writes `SetSummary`+`TruncateHistory`+`Save`. Writes are fire-and-forget (`session_store.go:10-12`); VCC treats `Save` error as log-only, never fails the turn.

**D6 — Config passthrough.** Factory receives `ContextManagerConfig json.RawMessage` (may be nil) and parses it defensively into `vcc.Settings` (algo spec §4: `overrideDefaultCompaction`, `smartKeepTail`, `continueAfterThresholdCompact` MVP-critical). No config enum change.

**D7 — Event emission.** VCC emits `KindAgentContextCompress` on force compression and `KindAgentSessionSummarize` after summarization, using `al.emitEvent` + `al.newTurnEventScope(...)`, mirroring legacy (`context_legacy.go:44-52, 251-260`).

**D8 — No validation/enum.** `ContextManager` stays a free string; unknown/failing `"vcc"` factory degrades to legacy with a warning (`turn_coord.go:348-360`). No `Config.Validate()` change.

**D9 — File size cap.** Each production/test file strictly < 400 LOC (measurable LOC budget); estimated pi-vcc core ≈ 1.5k LOC split across ~9 files.

**D10 — RFC 2119 wording policy.** All requirement statements in `specs/vcc-context-manager/specs/vcc-context-manager/spec.md` use normative `SHALL`/`MUST` language; no requirement is vacuous or optionally testable (e.g. REQ-3 budget is asserted via `estimateTokensFromChars(Summary + History) ≤ effectiveBudget`; REQ-5 Ingest appends to the per-session working set used by Assemble).

**D11 — One settings type.** `vcc.Settings` (types.go) is the single runtime struct produced by `config.ParseSettings(json.RawMessage)`; `settings.go` is only the MVP scaffold helper providing `DEFAULT_SETTINGS` and `scaffoldSettings` for tests/tooling. `PiVccSettings` is not a separate type.

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| Silent fallback to legacy on config typo (no enum) | Document `context_manager: "vcc"`; tests assert registration + selection (`context_manager_test.go:175-202` seam). |
| `AgentLoop` surface coupling | Adapter uses only the factory-passed `*AgentLoop` + public-ish helpers; pure `pkg/vcc/` has zero picoclaw coupling. |
| Summary format must match `BuildMessages` system-prompt embedding | VCC `AssembleResponse.Summary` is plain structured text; legacy emits plain text (`context_legacy.go:32`), seahorse emits XML (`context_seahorse.go:121-122`). VCC chooses plain bracketed-heading text. |
| Fire-and-forget writes may lose summary on crash | Same contract as legacy/seahorse today; acceptable for MVP. |
| RECALL_NOTE compounding across compactions | Append once after merge (`summarize.ts:179-191`); strip before merge (`stripRecallNote`). |

## File Decomposition (zero-overlap, <400 LOC each)

| File | Role | Est. LOC | Owner | Depends on |
|------|------|----------|-------|------------|
| `pkg/vcc/types.go` | `BlockKind`, `NormalizedBlock`, `SectionData`, `FileOps`, `Settings` (single runtime settings struct) | ~90 | A | — |
| `pkg/vcc/config.go` | `ParseSettings(json.RawMessage)`, defaults, coercion | ~110 | A | types.go |
| `pkg/vcc/content.go` | `textOf`, `sanitize`, `clip`, `clipSentence`, `nonEmptyLines` | ~100 | B | — |
| `pkg/vcc/normalize.go` | `normalize`, `normalizeOne` (sourceIndex mapping) | ~130 | B | types.go, content.go |
| `pkg/vcc/filter_noise.go` | `NOISE_TOOLS`, `NOISE_STRINGS`, `XML_WRAPPER_RE`, `filterNoise` | ~110 | B | types.go, content.go |
| `pkg/vcc/build_sections.go` | `buildSections` + extraction (goals, outstanding, files, commits, prefs) | ~190 | C | types.go, content.go |
| `pkg/vcc/brief.go` | `buildBriefSections`, `stringifyBrief`, `compressBash`, `toolOneLiner`, `capBrief` | ~190 | C | types.go, content.go |
| `pkg/vcc/format.go` | `section`, `formatSummary`, `RECALL_NOTE`, `wrapLongLines` | ~110 | C | types.go |
| `pkg/vcc/merge.go` | `mergePrevious`, `mergeHeaderSection`, `mergeFileLines`, `stripRecallNote` | ~140 | C | types.go |
| `pkg/vcc/token_estimate.go` | calibration, `estimateTokensFromChars`, `estimateMessageContentChars` | ~90 | D | — |
| `pkg/vcc/settings.go` | `DEFAULT_SETTINGS`, `loadSettings`, `scaffoldSettings` scaffold helper (`Settings` type lives in types.go) | ~90 | D | types.go |
| `pkg/agent/context_vcc.go` | `ContextManager` impl + factory + `init()` registration | ~170 | A | vcc types/config, agent loop |
| `pkg/vcc/e2e_test.go` | End-to-end pipeline smoke test (replaces loose `vcc_test.go` idea) | ~200 | C | all vcc pkg |
| `pkg/vcc/sections_test.go` / `brief_test.go` / `merge_test.go` | Pure-function unit tests | ~120 each | B/C | respective files |
| `pkg/agent/context_vcc_test.go` | Adapter selection + factory-error fallback tests | ~150 | A | context_manager_test.go seams |

**Parallel waves — worktree isolation + serialized merge gates:**
Because `pkg/vcc` is one Go package, file-disjointness is not compile-disjointness; `go test ./pkg/vcc/` compiles every `.go` file and the TDD red phase breaks the package for every worker. Workers therefore execute in isolated git worktrees and merge serially in dependency order:

- Wave 1 (Worker A): `pkg/vcc/types.go`, `config.go`, `settings.go` — merge first so the package compiles.
- Wave 2 (Worker B): `pkg/vcc/content.go`, `normalize.go`, `filter_noise.go` — merge after Wave 1.
- Wave 3 (Worker C): `pkg/vcc/build_sections.go`, `brief.go`, `format.go`, `merge.go`, `token_estimate.go` — merge after Wave 2.
- Wave 4 (Worker A): `pkg/agent/context_vcc.go` — merge last (imports `pkg/vcc`).

Run package tests only after each merge. No two workers touch the same file.
