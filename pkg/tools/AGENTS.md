# Project Instructions

## Purpose
Builtin tool packages (`cron`, `fs`, `integration`, `shell`, `spawn`, `subagent`,
`search`, `load_tool`, `normalization`, `validate`, `toolloop`, `shared`, ...)
and the tool registry.

## Ownership
Parent: `pkg/AGENTS.md`.

## Local Contracts
- Tool set must match `pkg/agent/agent_init.go` registration and the dashboard
  catalog in `web/backend/api/tools.go`.
- Removing a tool is cross-cutting: tool package, registration, config fields
  and defaults, `config/config.example.json`, dashboard catalog and its tests,
  `workspace/skills/` docs, `docs/`, and any i18n reason codes.
- Media type decisions must mirror `inferMediaType` in
  `pkg/agent/agent_utils.go`: SVG (`image/svg*` or `.svg`) is sent as `file`,
  never `image`, because raster-only endpoints such as Telegram `SendPhoto`
  reject vectors. `normalizeMessageMediaType` in
  `pkg/tools/integration/message.go` re-implements this rule and must not
  drift; an explicit `media[].type` hint overrides detection. Delegation from
  `pkg/tools` to `pkg/agent` is impossible (import cycle) — move shared helpers
  down instead.
- Enabled is not visible. With `tools.lazy_loading.enabled`, non-core tools are
  registered through `RegisterHidden` and their schemas are dropped from the
  provider payload while the names stay callable. A schema that is additionally
  gated by config (e.g. `message.file_path` exists only when
  `tools.message.media_enabled` is true) leaves the model unable to call the
  tool correctly, so document every such gate in `docs/`.

## Work Guidance

## Verification
`go test -tags goolm,stdjson ./pkg/tools/...`

## Child DOX Index
(none — `pkg/tools/fs`, `pkg/tools/integration`, `pkg/tools/shared` are owned
directly by this doc)
