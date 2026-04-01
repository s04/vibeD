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

## Next PR

- Next PR: Reuse the shared `internal/operations` registry as the source of truth for MCP tool names and descriptions for the overlapping artifact lifecycle operations. This continues the effort to reduce duplicated edge-layer metadata while preserving the original tool wording.

## 2026-04-01

- PR #1: Added an initial PR-loop workflow script and introduced first-pass REST deploy/update endpoints for artifacts (`POST /api/artifacts`, `PUT /api/artifacts/{id}`), along with matching OpenAPI updates and handler tests. This was the fork's first step toward making deployment usable without relying only on MCP.
- PR #2: Tightened the PR-loop workflow so staged files define the PR boundary. This keeps future pull requests small and reviewable instead of committing every local change at once.
- PR #3: Added this `FORK_NOTES.md` file, a README pointer to it, and a pull request template reminding contributors to record fork-specific changes when they materially affect behavior or direction.
- PR #4: Added a shared `internal/operations` foundation package that defines the overlapping artifact lifecycle operations in one canonical registry. This is groundwork for reducing duplication between REST and MCP without changing deployment behavior yet.
