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

---

# DOX framework

- DOX is highly performant AGENTS.md hierarchy installed here
- Agent must follow DOX instructions across any edits

## Core Contract

- AGENTS.md files are binding work contracts for their subtrees
- Work products, source materials, instructions, records, assets, and durable docs must stay understandable from the nearest applicable AGENTS.md plus every parent AGENTS.md above it

## Read Before Editing

1. Read the root AGENTS.md
2. Identify every file or folder you expect to touch
3. Walk from the repository root to each target path
4. Read every AGENTS.md found along each route
5. If a parent AGENTS.md lists a child AGENTS.md whose scope contains the path, read that child and continue from there
6. Use the nearest AGENTS.md as the local contract and parent docs for repo-wide rules
7. If docs conflict, the closer doc controls local work details, but no child doc may weaken DOX

Do not rely on memory. Re-read the applicable DOX chain in the current session before editing.

## Update After Editing

Every meaningful change requires a DOX pass before the task is done.

Update the closest owning AGENTS.md when a change affects:

- purpose, scope, ownership, or responsibilities
- durable structure, contracts, workflows, or operating rules
- required inputs, outputs, permissions, constraints, side effects, or artifacts
- user preferences about behavior, communication, process, organization, or quality
- AGENTS.md creation, deletion, move, rename, or index contents

Update parent docs when parent-level structure, ownership, workflow, or child index changes. Update child docs when parent changes alter local rules. Remove stale or contradictory text immediately. Small edits that do not change behavior or contracts may leave docs unchanged, but the DOX pass still must happen.

## Hierarchy

- Root AGENTS.md is the DOX rail: project-wide instructions, global preferences, durable workflow rules, and the top-level Child DOX Index
- Child AGENTS.md files own domain-specific instructions and their own Child DOX Index
- Each parent explains what its direct children cover and what stays owned by the parent
- The closer a doc is to the work, the more specific and practical it must be

## Child Doc Shape

- Create a child AGENTS.md when a folder becomes a durable boundary with its own purpose, rules, responsibilities, workflow, materials, or quality standards
- Work Guidance must reflect the current standards of the project or user instructions; if there are no specific standards or instructions yet, leave it empty
- Verification must reflect an existing check; if no verification framework exists yet, leave it empty and update it when one exists

Default section order:
- Purpose
- Ownership
- Local Contracts
- Work Guidance
- Verification
- Child DOX Index

## Style

- Keep docs concise, current, and operational
- Document stable contracts, not diary entries
- Put broad rules in parent docs and concrete details in child docs
- Prefer direct bullets with explicit names
- Do not duplicate rules across many files unless each scope needs a local version
- Delete stale notes instead of explaining history
- Trim obvious statements, repeated rules, misplaced detail, and warnings for risks that no longer exist

## Closeout

1. Re-check changed paths against the DOX chain
2. Update nearest owning docs and any affected parents or children
3. Refresh every affected Child DOX Index
4. Remove stale or contradictory text
5. Run existing verification when relevant
6. Report any docs intentionally left unchanged and why

## User Preferences

When the user requests a durable behavior change, record it here or in the relevant child AGENTS.md

## Child DOX Index

Direct child AGENTS.md files:
- `cmd/AGENTS.md` — Go entry points: `picoclaw` binary and `membench` benchmark.
- `pkg/AGENTS.md` — Go library packages under `pkg/`.
- `web/AGENTS.md` — Dashboard backend (`web/backend`) and frontend (`web/frontend`).
- `docs/AGENTS.md` — Project documentation (multi-language).
- `workspace/AGENTS.md` — Embedded builtin skills, prompts, and workspace templates.
- `scripts/AGENTS.md` — Repo tooling scripts (build, lint, integration tests).

Owned by root (not delegated to a child):
- sbeads task tracking; build/test toolchain rules (Go toolchain, `-tags goolm,stdjson`);
  top-level repo files (`go.mod`, `Makefile`, `README.md`, `ROADMAP.md`,
  `CONTRIBUTING.md`); `config/`, `docker/`, `assets/`, `examples/`,
  `integration/`, `.github/`, `.sbeads/`.
