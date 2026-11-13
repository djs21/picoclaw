# Project Instructions

## Purpose
LLM provider implementations (`openai_compat`, `anthropic`, `anthropic_messages`,
`azure`, `bedrock`, `cli`, `httpapi`, `openai_responses_common`) plus factory,
fallback, ratelimiter, error classifier, and tool schema transform.

## Ownership
Parent: `pkg/AGENTS.md`.

## Local Contracts
- Providers constructed via `pkg/providers/factory.go`; catalog in
  `pkg/providers/provider_catalog.go`.
- Dashboard model catalog (`web/backend/api/model_catalog.go`, `models.go`) must
  stay consistent with providers.

## Work Guidance

## Verification
`go test -tags goolm,stdjson ./pkg/providers/...`

## Child DOX Index
(none)
