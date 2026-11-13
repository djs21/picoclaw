# Project Instructions

## Purpose
Go library packages — the shared runtime and tooling used by `cmd/` and `web/`.
Includes agent core, builtin tools, chat channels, LLM providers, and utility
packages.

## Ownership
All `pkg/*` packages. Root owns `go.mod`, build tags, and the cross-cutting
tool-removal procedure (reference commit `d3ba5737`, i2c/spi/serial prune).

## Local Contracts
- Builtin tool set defined in `pkg/agent/agent_init.go` (`registerSharedTools`)
  and per-tool packages under `pkg/tools/`. The dashboard catalog in
  `web/backend/api/tools.go` must list the same tools.
- Single Go module (`go.mod` at repo root). Plain `go build ./...` fails without
  `-tags goolm,stdjson`.
- Removing a tool is cross-cutting: tool package, registration, config fields
  and defaults, `config/config.example.json`, dashboard catalog and its tests,
  `workspace/skills/` docs, `docs/`, and any i18n reason codes.

## Work Guidance

## Verification
`go build -tags goolm,stdjson ./...`
`go test -tags goolm,stdjson $(go list ./... | grep -v "github.com/sipeed/picoclaw/web/")`
`go test -tags goolm,stdjson $(go list ./... | grep "picoclaw/web/")`

## Child DOX Index
- `pkg/agent/AGENTS.md` — agent core (lifecycle, pipelines, context, hooks, steering).
- `pkg/tools/AGENTS.md` — builtin tool packages and tool registry.
- `pkg/channels/AGENTS.md` — chat channel implementations.
- `pkg/providers/AGENTS.md` — LLM provider implementations.
