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

## Work Guidance

## Verification
`go test -tags goolm,stdjson ./pkg/tools/...`

## Child DOX Index
(none — `pkg/tools/fs`, `pkg/tools/integration`, `pkg/tools/shared` are owned
directly by this doc)
