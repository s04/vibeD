# Fork Notes

This fork tracks upstream `vibeD` and records changes that are specific to `s04/vibeD`.

## Why This Exists

This fork is being explored as an incremental evolution of `vibeD`, not a ground-up rewrite.
The focus is on:

- improving non-MCP ergonomics
- exploring API-first and agent-identity-friendly workflows
- keeping the existing orchestrator core intact while reducing duplicated adapter code

## Fork Delta

Add short, high-signal entries here when a PR materially changes the fork's direction, API surface, operator workflow, or product positioning.

Format:

```md
## YYYY-MM-DD

- PR #123: short description of what changed and why it matters
```

## Current Entries

## 2026-04-01

- PR #1: Added `scripts/pr-loop.sh` for incremental branch -> PR -> merge workflow.
- PR #2: Tightened the PR loop so staged files define the PR boundary.
