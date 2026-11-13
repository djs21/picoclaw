# Project Instructions

## Purpose
Embedded builtin skills, prompts, and workspace templates — the only tracked
source, embedded into the binary via `onboard_workspace_embed.go`.

## Ownership
`workspace/` (`AGENT.md`, `SOUL.md`, `USER.md`, `memory/`, `skills/`).
Root AGENTS.md names `workspace/` as the source of truth; there is no second copy
under `cmd/`.

## Local Contracts
- `workspace/` is embedded into the binary; no second copy under `cmd/` to update.
- Skills are copied into `~/.picoclaw/workspace/` at onboarding; `make install`
  does not prune stale copies — removing a builtin skill also requires deleting
  the stale copy in the live workspace.

## Work Guidance

## Verification
(none specific)

## Child DOX Index
- `workspace/skills/AGENTS.md` — embedded builtin skills.
