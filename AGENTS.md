# Project Instructions

## Task Tracking with sbeads

This repository uses sbeads for lightweight, dependency-aware local issue tracking and multi-agent coordination.

- **Check ready tasks**: Run `sbeads ready` to view unblocked tasks.
- **Claim a task**: Run `sbeads claim <id> --agent <name>` before working.
- **Release a claim**: Run `sbeads unclaim <id>` if aborted or blocked.
- **Inspect details**: Run `sbeads show <id>` to view computed status and dependency edges.
- **Close completed work**: Run `sbeads close <id>` after tests pass.
- **Mutations & safety**: Use `sbeads update`, `sbeads reopen`, `sbeads delete`, and `sbeads dep add/rm`. Never modify database SQLite files directly.
