# Project Instructions

## Purpose
Dashboard: Go backend (`web/backend`) and pnpm React/TypeScript frontend
(`web/frontend`).

## Ownership
`web/backend/` and `web/frontend/`. Root owns build tags for backend Go code.

## Local Contracts
- Backend built with `-tags goolm,stdjson`; `web/backend/api/tools.go` must list
  the same tools as `pkg/agent` + `pkg/tools`.
- Frontend needs `pnpm install` before `pnpm lint`. Locale files under
  `web/frontend/src/i18n/locales/` must stay valid JSON (checked by parsing, not
  by ESLint).

## Work Guidance

## Verification
Backend: `go test -tags goolm,stdjson $(go list ./... | grep "picoclaw/web/")`
Frontend: `pnpm lint` (after `pnpm install`)

## Child DOX Index
- `web/backend/AGENTS.md` — Go dashboard API server.
- `web/frontend/AGENTS.md` — pnpm React/TypeScript frontend and i18n locales.
