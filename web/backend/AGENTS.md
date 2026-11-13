# Project Instructions

## Purpose
Go dashboard API server: `api`, `middleware`, `model`, `dashboardauth`,
`launcherconfig`, `utils`, `systray`, `embed`.

## Ownership
Parent: `web/AGENTS.md`. Root owns build tags.

## Local Contracts
- `web/backend/api/tools.go` must match the `pkg` tool set.
- `embed.go` embeds the frontend `dist`; `i18n.go` handles translations.

## Work Guidance

## Verification
`go test -tags goolm,stdjson ./web/backend/...`

## Child DOX Index
(none)
