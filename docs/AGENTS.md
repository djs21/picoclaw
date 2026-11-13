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

## Verification
`scripts/lint-docs.sh` (runs from the git toplevel)

## Child DOX Index
(none)
