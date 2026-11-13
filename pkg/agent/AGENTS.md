# Project Instructions

## Purpose
Agent core: lifecycle, pipelines, context, hooks, steering, events, instance,
turn coordination, and the shared tool registration (`registerSharedTools`).

## Ownership
Parent: `pkg/AGENTS.md`.

## Local Contracts
- `pkg/agent/agent_init.go` (`registerSharedTools`) registers builtin tools; keep
  in sync with `pkg/tools/` and `web/backend/api/tools.go`.
- `pkg/tools/` subdirs (`fs`, `integration`, `shared`) are owned directly by
  `pkg/tools/AGENTS.md`.

## Work Guidance

## Verification
`go test -tags goolm,stdjson ./pkg/agent/...`

## Child DOX Index
(none)
