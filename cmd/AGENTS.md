# Project Instructions

## Purpose
Go entry points for the picoclaw binary and the membench benchmark.
`cmd/picoclaw/main.go` is the binary entry point; `cmd/picoclaw/internal/` holds
internal packages not importable outside `cmd/picoclaw`. `cmd/membench/` is a
standalone benchmark main package.

## Ownership
- `cmd/picoclaw/` — picoclaw binary entry point + internal packages
- `cmd/membench/` — memory/embedding benchmark tool
Owned by root: `go.mod`, build tags, and the root-level build/test rules.

## Local Contracts
- Build with `-tags goolm,stdjson` (per root AGENTS.md "Build and Test"). Plain
  `go build ./...` fails: `olm/olm.h: No such file or directory`.
- Business logic lives in `pkg/`; `cmd/` entry points stay thin.
- `cmd/picoclaw/internal/` packages are imported only within `cmd/picoclaw`.

## Work Guidance
Keep entry points minimal; do not put reusable library code under `cmd/`.

## Verification
`go build -tags goolm,stdjson ./cmd/...`
`go test -tags goolm,stdjson ./cmd/...`

## Child DOX Index
- `cmd/picoclaw/internal/AGENTS.md` — internal packages of the picoclaw binary.
