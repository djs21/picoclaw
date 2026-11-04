# Project Instructions

## Task Tracking with sbeads

This repository uses sbeads for lightweight, dependency-aware local issue tracking and multi-agent coordination.

- **Check ready tasks**: Run `sbeads ready` to view unblocked tasks.
- **Claim a task**: Run `sbeads claim <id> --agent <name>` before working.
- **Release a claim**: Run `sbeads unclaim <id>` if aborted or blocked.
- **Inspect details**: Run `sbeads show <id>` to view computed status and dependency edges.
- **Close completed work**: Run `sbeads close <id>` after tests pass.
- **Mutations & safety**: Use `sbeads update`, `sbeads reopen`, `sbeads delete`, and `sbeads dep add/rm`. Never modify database SQLite files directly.

## Build and Test

- **Never force the toolchain.** `go.mod` pins the language version, and the Go
  command resolves the matching toolchain itself (distro Go 1.24 + `go.mod` 1.25
  means Go downloads the newer toolchain). Do not reintroduce
  `GOTOOLCHAIN=local` in the Makefile or scripts; that only produces
  `go.mod requires go >= ...` failures on distro Go packages.
- **Build tags matter.** `make` targets pass `-tags goolm,stdjson`. Plain
  `go build ./...` fails with `olm/olm.h: No such file or directory` because the
  Matrix channel needs the pure-Go `goolm` build. Use the Makefile, or pass the
  same tags explicitly.
- **Core suite** (mirrors `make test`, web packages excluded on purpose):

  ```bash
  go build -tags goolm,stdjson ./...
  go test -tags goolm,stdjson $(go list ./... | grep -v "github.com/sipeed/picoclaw/web/")
  go test -tags goolm,stdjson $(go list ./... | grep "picoclaw/web/")
  ```

- **Frontend** lives in `web/frontend` and needs `pnpm install` before
  `pnpm lint`. Locale files under `web/frontend/src/i18n/locales/` must stay
  valid JSON; they are checked by parsing, not by ESLint.
- **Binary install**: `make install` writes to `$INSTALL_PREFIX/bin/picoclaw`
  (default `~/.local/bin`). Skills embedded from `workspace/` are only copied
  into `~/.picoclaw/workspace/` at onboarding, so removing a builtin skill also
  requires deleting the stale copy in the live workspace (`make install` does
  not prune it).

## Tool and Workspace Sources of Truth

- `workspace/` at the repository root is embedded into the binary
  (`onboard_workspace_embed.go`) and is the only tracked source of builtin
  skills, prompts, and workspace templates. There is no second copy under
  `cmd/` to update.
- The builtin tool set is defined in code: `pkg/agent/agent_init.go`
  (`registerSharedTools`) and the per-tool packages under `pkg/tools/`. The
  dashboard catalog in `web/backend/api/tools.go` must list the same tools.
- Removing a tool is a cross-cutting change: tool package, registration, config
  fields and defaults, `config/config.example.json`, dashboard catalog and its
  tests, `workspace/skills/` docs, `docs/`, and any i18n reason codes. Follow
  commit `d3ba5737` (i2c/spi/serial prune) as the reference shape.

## Workflow Conventions

- One sbeads task equals one atomic commit. Claim before editing, close after
  the suite passes, and keep history and docs in the same commit as the change
  they describe.
- Prefer test-first: write the failing test, make it pass, then refactor.
- Verify with real commands before claiming done. Report the command and its
  result rather than assumptions; a grep that finds nothing is evidence, an
  unrun grep is not.
- Keep replies in Indonesian for this repository's own sessions, and keep
  documentation in English unless the file is an existing localized variant.
