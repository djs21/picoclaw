# Project Instructions

## Purpose
Repo tooling scripts: `build-macos-app.sh`, `lint-docs.sh`,
`run-integration-tests.sh`, `test-docker-mcp.sh`, `test-irc.sh`, `copydir.go`,
`setup.iss`.

## Ownership
`scripts/`. Root owns the `Makefile` targets that may invoke these scripts.

## Local Contracts
- `run-integration-tests.sh` drives integration suites under `integration/`.
- `lint-docs.sh` enforces the documentation layout: English-only filenames
  repo-wide, no locale directories under `docs/`, typed directories, and only
  `docs/README.md` plus the DOX `docs/AGENTS.md` directly under `docs/`.

## Work Guidance

## Verification
(none specific)

## Child DOX Index
(none)
