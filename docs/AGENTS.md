# Project Instructions

## Purpose
Project documentation: `architecture`, `channels`, `design`, `guides`,
`migration`, `operations`, `project`, `reference`, `security`, with multi-language
variants (`.fr`, `.ja`, `.zh`, `.pt-br`, `.vi`, `.ms`, `.id`, `.it`, `.ko`).

## Ownership
`docs/` and all `*.md` files beneath it. Root AGENTS.md references `docs/` for
cross-cutting changes (e.g. tool removal).

## Local Contracts
- Translations live beside their English source as `<name>.<locale>.md` (enforced
  by `scripts/lint-docs.sh`); no locale-named subdirectories under `docs/`;
  translated project entry docs go under `docs/project/`.

## Work Guidance

- Config docs must match the JSON struct tags in `pkg/config` and the defaults in
  `pkg/config/defaults.go`; never document a default from memory.
- `env:` / `envPrefix:` struct tags in `pkg/config` are declarations only — there
  is no reflection-based env binder. Only present a `PICOCLAW_*` variable as
  working when a matching `os.LookupEnv` call exists.
- Tool names in docs must match currently registered tools. Retired tools and
  their replacements are recorded in `applyToolState` in
  `web/backend/api/tools.go`; grep that switch before documenting a tool name.

## Verification
`scripts/lint-docs.sh` (runs from the git toplevel)

## Child DOX Index
(none)
