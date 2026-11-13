# VCC ContextManager — Implementation Tasks

Change: `vcc-context-manager` (optional third ContextManager, never replaces legacy/seahorse).
Every task = one atomic commit. TDD direction: write the failing test first, then minimal code, then pass.
Wave grouping enables ≥3 workers in parallel with **zero-overlap file ownership**, executed through **git worktree isolation per worker plus serialized merge gates** (see Execution Model below).

## Wave 1 — Types, Schemas, Config (Worker A) — files: pkg/vcc/types.go, pkg/vcc/config.go, pkg/vcc/settings.go

**1.1 Types/Schemas** (TDD: `pkg/vcc/types_test.go` first)
- Files: Create `pkg/vcc/types.go`, `pkg/vcc/types_test.go`
- Steps:
  - [ ] Write failing `types_test.go` asserting `BlockKind` constants (`user`/`assistant`/`tool_call`/`tool_result`/`bash`) and `NormalizedBlock`/`SectionData` struct tags (algo spec §2.1, §2.2).
  - [ ] Implement `pkg/vcc/types.go` with `BlockKind`, `NormalizedBlock`, `SectionData`, `FileOps`.
  - [ ] Run `go test -tags goolm,stdjson ./pkg/vcc/` → types tests pass.
  - [ ] Commit: `feat(vcc): add types and section data model`

**1.2 Config parsing** (TDD: `pkg/vcc/config_test.go` first)
- Files: Create `pkg/vcc/config.go`, `pkg/vcc/config_test.go`
- Steps:
  - [ ] Write failing `config_test.go` asserting `ParseSettings(json.RawMessage)` handles nil/raw/invalid JSON and coerces bad arrays to `[]`.
  - [ ] Implement `ParseSettings` in `pkg/vcc/config.go` (algo spec §4: `overrideDefaultCompaction`, `smartKeepTail`, `continueAfterThresholdCompact` MVP-critical).
  - [ ] Run tests pass.
  - [ ] Commit: `feat(vcc): add config parsing for ContextManagerConfig`

**1.3 Settings scaffolding** (TDD: `pkg/vcc/settings_test.go` first)
- Files: Create `pkg/vcc/settings.go`, `pkg/vcc/settings_test.go`
- Steps:
  - [ ] Write failing `settings_test.go` asserting `DEFAULT_SETTINGS`, `loadSettings`, `scaffoldSettings` (never clobber invalid keys).
  - [ ] Implement `pkg/vcc/settings.go`.
  - [ ] Run tests pass.
  - [ ] Commit: `feat(vcc): add settings load/scaffold`

## Wave 2 — Parsing/Validation + Core Domain Logic (Worker B) — files: pkg/vcc/content.go, pkg/vcc/normalize.go, pkg/vcc/filter_noise.go

**2.1 Content primitives** (TDD: `pkg/vcc/content_test.go` first)
- Files: Create `pkg/vcc/content.go`, `pkg/vcc/content_test.go`
- Steps:
  - [ ] Write failing tests for `textOf`, `sanitize` (CRLF/CR→LF, ANSI stripping), `clip`/`clipSentence`, `nonEmptyLines` (algo spec §3.1).
  - [ ] Implement `pkg/vcc/content.go`.
  - [ ] Run tests pass.
  - [ ] Commit: `feat(vcc): add content primitives`

**2.2 Normalize** (TDD: `pkg/vcc/normalize_test.go` first)
- Files: Create `pkg/vcc/normalize.go`, `pkg/vcc/normalize_test.go`
- Steps:
  - [ ] Write failing tests for `normalize`/`normalizeOne` including sourceIndex mapping, image blocks `[image: mime]`, empty-text fallback user block (algo spec §3.1).
  - [ ] Implement `pkg/vcc/normalize.go`.
  - [ ] Run tests pass.
  - [ ] Commit: `feat(vcc): add message→block normalization`

**2.3 Filter noise** (TDD: `pkg/vcc/filter_noise_test.go` first)
- Files: Create `pkg/vcc/filter_noise.go`, `pkg/vcc/filter_noise_test.go`
- Steps:
  - [ ] Write failing tests for `NOISE_TOOLS` drop, `NOISE_STRINGS` drop, `XML_WRAPPER_RE` stripping, empty-after-clean drop (algo spec §3.2).
  - [ ] Implement `pkg/vcc/filter_noise.go`.
  - [ ] Run tests pass.
  - [ ] Commit: `feat(vcc): add noise filtering`

## Wave 3 — Core Domain Logic (Worker C) — files: pkg/vcc/build_sections.go, pkg/vcc/brief.go, pkg/vcc/format.go, pkg/vcc/merge.go

**3.1 Section extraction** (TDD: `pkg/vcc/build_sections_test.go` first)
- Files: Create `pkg/vcc/build_sections.go`, `pkg/vcc/build_sections_test.go`
- Steps:
  - [ ] Write failing tests for `buildSections` producing `SectionData`: goals (≤8, ≤200 chars), outstanding context (≤5, blocker-regex, tail-20 scan), files/changes (≤10/category, Modified-wins-over-Created), commits (≤8), preferences (≤10, dedup vs goals) (algo spec §3.3).
  - [ ] Implement `pkg/vcc/build_sections.go`.
  - [ ] Run tests pass.
  - [ ] Commit: `feat(vcc): add section extraction`

**3.2 Brief transcript** (TDD: `pkg/vcc/brief_test.go` first)
- Files: Create `pkg/vcc/brief.go`, `pkg/vcc/brief_test.go`
- Steps:
  - [ ] Write failing tests for `buildBriefSections`/`stringifyBrief`: `[user]`/`[assistant]` headers, `(#N)` refs, bash compression (`compressBash`), `toolOneLiner`, tool-call cap 8/turn, identical-tool collapse `(#A,#B) xN`, `BRIEF_MAX_LINES` 120 (algo spec §3.4).
  - [ ] Implement `pkg/vcc/brief.go`.
  - [ ] Run tests pass.
  - [ ] Commit: `feat(vcc): add brief transcript rendering`

**3.3 Format summary** (TDD: `pkg/vcc/format_test.go` first)
- Files: Create `pkg/vcc/format.go`, `pkg/vcc/format_test.go`
- Steps:
  - [ ] Write failing tests for `formatSummary` (section order: Session Goal, Files And Changes, Commits, Outstanding Context, User Preferences), `RECALL_NOTE` constant, `wrapLongLines` (120 chars), `capBrief` (algo spec §3.5).
  - [ ] Implement `pkg/vcc/format.go`.
  - [ ] Run tests pass.
  - [ ] Commit: `feat(vcc): add summary formatting`

**3.4 Merge semantics** (TDD: `pkg/vcc/merge_test.go` first)
- Files: Create `pkg/vcc/merge.go`, `pkg/vcc/merge_test.go`
- Steps:
  - [ ] Write failing tests for `mergePrevious`: sticky sections dedup prev+fresh, volatile Outstanding Context fresh-only, empty prev→fresh / empty fresh→prev, `mergeFileLines` by category, `stripRecallNote`, RECALL_NOTE appended once after merge (algo spec §3.6).
  - [ ] Implement `pkg/vcc/merge.go`.
  - [ ] Run tests pass.
  - [ ] Commit: `feat(vcc): add summary merge semantics`

**3.5 Token estimate** (TDD: `pkg/vcc/token_estimate_test.go` first) — Worker D can start here
- Files: Create `pkg/vcc/token_estimate.go`, `pkg/vcc/token_estimate_test.go`
- Steps:
  - [ ] Write failing tests for `estimateTokensFromChars`, calibration clamp [2,6], `estimateMessageContentChars` (image=4800) (algo spec §3.7).
  - [ ] Implement `pkg/vcc/token_estimate.go`.
  - [ ] Run tests pass.
  - [ ] Commit: `feat(vcc): add token estimation`

## Wave 4 — Handlers/Pipeline Integration + Wiring/Lifecycle (Worker A) — file: pkg/agent/context_vcc.go

**4.1 VCC ContextManager adapter** (TDD: `pkg/agent/context_vcc_test.go` first)
- Files: Create `pkg/agent/context_vcc.go`, `pkg/agent/context_vcc_test.go`
- Steps:
  - [ ] Write failing tests using `resetCMRegistry()` seam (line 843) + `newCMTestAgentLoop` (line 873): register `"vcc"` factory, assert `al.contextManager` is VCC type; assert unknown/failing factory falls back to legacy (mirrors `context_manager_test.go:175-231`).
  - [ ] Implement `pkg/agent/context_vcc.go`: `newVCCContextManager(cfg json.RawMessage, al *AgentLoop)`, `Assemble`/`Compact`/`Ingest`/`Clear`, `init()` → `RegisterContextManager("vcc", ...)`, event emission (`KindAgentContextCompress`/`KindAgentSessionSummarize`) via `al.emitEvent` + `al.newTurnEventScope(...)` (design D7).
  - [ ] Run `go test -tags goolm,stdjson ./pkg/agent/ -run VCC` → pass.
  - [ ] Commit: `feat(vcc): add ContextManager adapter and registration`

## Wave 5 — Docs (Worker A)

**5.1 Document config option**
- Files: Create/update docs (do not modify existing source). Add `context_manager: "vcc"` note to relevant config docs; update `openspec/changes/vcc-context-manager/` already done.
- Steps:
  - [ ] Add VCC config documentation entry describing `context_manager` + `context_manager_config`.
  - [ ] Commit: `docs(vcc): document context_manager vcc option`

## Wave 6 — E2E/Stress (Worker C)

**6.1 End-to-end compile smoke test**
- Files: Create `pkg/vcc/e2e_test.go` (replaces the loose `vcc_test.go` idea from the design table; no new production file)
- Steps:
  - [ ] Write a realistic multi-message conversation, run full pipeline `normalize→filterNoise→buildSections→formatSummary→mergePrevious→wrapLongLines+RECALL_NOTE`, assert deterministic output and RECALL_NOTE appears once.
  - [ ] Run `go test -tags goolm,stdjson ./pkg/vcc/ -run E2E` → pass.
  - [ ] Commit: `test(vcc): add end-to-end compile smoke test`

## Wave 7 — Quality Gates / Closeout (Worker A)

**7.1 Build + vet + full package tests**
- Files: none (verification only)
- Steps:
  - [ ] Run `go build -tags goolm,stdjson ./...` → success.
  - [ ] Run `go vet -tags goolm,stdjson ./pkg/vcc/ ./pkg/agent/` → clean.
  - [ ] Run `go test -tags goolm,stdjson ./pkg/vcc/ ./pkg/agent/` → all pass (including existing legacy/seahorse tests, confirming no regression).
  - [ ] Run `openspec validate vcc-context-manager` → pass (deltas present in specs/).
  - [ ] Commit: `chore(vcc): quality gates and closeout`

## Parallelization Wave Table + Worker File Ownership

| Wave | Worker | Files created (owner) | Overlap |
|------|--------|----------------------|---------|
| 1 | A | `pkg/vcc/types.go`, `pkg/vcc/config.go`, `pkg/vcc/settings.go` (+ tests) | none |
| 2 | B | `pkg/vcc/content.go`, `pkg/vcc/normalize.go`, `pkg/vcc/filter_noise.go` (+ tests) | none |
| 3 | C | `pkg/vcc/build_sections.go`, `pkg/vcc/brief.go`, `pkg/vcc/format.go`, `pkg/vcc/merge.go` (+ tests) | none |
| 3/5 | D | `pkg/vcc/token_estimate.go` (+ test) | none |
| 4 | A | `pkg/agent/context_vcc.go` (+ test) | none (imports pkg/vcc only) |
| 5/6/7 | A/C | docs, e2e test, verification | none |

Rules:
- No two workers edit the same file.
- Each worker runs in its own git branch and isolated worktree; merges are **serialized** in dependency order: Wave 1 (package compiles) → Wave 2 → Wave 3 + token estimate → Wave 4 (`pkg/agent` depends on `pkg/vcc`).
- Run `go test ./pkg/vcc/` only after each merge — never in the shared worktree while another worker's red-phase files are present.
- `pkg/vcc/` is pure Go: zero picoclaw imports; `-tags goolm,stdjson` compatible; no build tag gate.
- `pkg/agent/context_vcc.go` is the only file touching `AgentLoop` internals.
- All tasks map to atomic commits listed above.

## Execution Model — Worktree Isolation + Serialized Merge Gates

Waves touch disjoint files but compile as **ONE Go package** (`pkg/vcc`). File-disjointness is not compile-disjointness: `go test ./pkg/vcc/` compiles every `.go` file in the package, and the TDD red phase leaves the package uncompilable for every worker, not just the one who is red. Therefore:

- Each worker runs on its own git branch and isolated worktree.
- Merge serially in dependency order: Wave 1 (package compiles) → Wave 2 → Wave 3 + token estimate → Wave 4 (`pkg/agent` depends on `pkg/vcc`).
- Run package tests only after each merge; never run `go test ./pkg/vcc/` in the shared worktree while another worker's red-phase files are present.
- TDD direction (test first) and atomic commit-per-task mapping are preserved within each worktree.

## OpenSpec Validation

The change includes `specs/vcc-context-manager/spec.md` with `## ADDED Requirements` and `#### Scenario:` blocks, satisfying OpenSpec delta validation (no `skip_specs` needed).

Validation commands the Orc must run later:
```
cd /home/djs/project/picoclaw
openspec validate vcc-context-manager
go build -tags goolm,stdjson ./...
go vet -tags goolm,stdjson ./pkg/vcc/ ./pkg/agent/
go test -tags goolm,stdjson ./pkg/vcc/ ./pkg/agent/
```
