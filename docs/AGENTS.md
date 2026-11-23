# Project Instructions

## Purpose
Project documentation: `architecture`, `channels`, `design`, `guides`,
`migration`, `operations`, `project`, `reference`, `security`.

## Ownership
`docs/` and all `*.md` files beneath it. Root AGENTS.md references `docs/` for
cross-cutting changes (e.g. tool removal).

## Local Contracts
- Documentation is English-only: one canonical English file per document. Do not
  add localized variants (`<name>.<locale>.md`, `<name>_<locale>.md`) or
  locale-named subdirectories under `docs/`; `scripts/lint-docs.sh` rejects both.
- `docs/design/` holds two Chinese-only documents with no English source
  (`hook-system-design.zh.md`, `issue-783-investigation-and-fix-plan.zh.md`);
  they await a translate-or-remove decision. Every other localized doc fails
  lint.
- Directory type comes first: `docs/<type>/<name>.md`. Only `docs/README.md` and
  the DOX `docs/AGENTS.md` may sit directly under `docs/`.

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
