# Project Instructions

## Purpose
Internal packages of the picoclaw binary: `agent`, `auth`, `cliui`, `config`,
`cron`, `gateway`, `helpers`, `migrate`, `model`, `onboard`, `skills`, `status`,
`version`. Not importable outside `cmd/picoclaw`.

## Ownership
Parent: `cmd/AGENTS.md`. Root owns `go.mod` and build tags.

## Local Contracts
- Internal packages only imported within `cmd/picoclaw`.
- `onboard` copies builtin skills from `workspace/` into `~/.picoclaw/workspace/`
  at onboarding; `make install` does not prune stale copies.

## Work Guidance

## Verification
`go build -tags goolm,stdjson ./cmd/picoclaw/...`
`go test -tags goolm,stdjson ./cmd/picoclaw/...`

## Child DOX Index
(none)
