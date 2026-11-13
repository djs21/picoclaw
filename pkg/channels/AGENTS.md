# Project Instructions

## Purpose
Chat channel implementations: base interfaces, manager, registry, and per-channel
packages (`matrix`, `irc`, `telegram`, `discord`, `slack`, `whatsapp`, `wechat`/
`weixin`/`wecom`, `qq`, `line`, `feishu`, `dingtalk`, `mqtt`, `onebot`, `pico`,
`vk`, `deltachat`, `maixcam`, `slack_webhook`, `teams_webhook`, `webhook`).

## Ownership
Parent: `pkg/AGENTS.md`.

## Local Contracts
- Channel packages implement interfaces in `pkg/channels/interfaces.go` /
  `pkg/channels/base.go`.
- Registered via `pkg/channels/registry.go` and `pkg/channels/manager.go`.
- Channel docs live under `docs/channels/`.

## Work Guidance

## Verification
`go test -tags goolm,stdjson ./pkg/channels/...`

## Child DOX Index
(none)
