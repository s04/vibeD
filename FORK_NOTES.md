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

- PR #1: Added an initial PR-loop workflow script and introduced first-pass REST deploy/update endpoints for artifacts (`POST /api/artifacts`, `PUT /api/artifacts/{id}`), along with matching OpenAPI updates and handler tests. This was the fork's first step toward making deployment usable without relying only on MCP.
- PR #2: Tightened the PR-loop workflow so staged files define the PR boundary. This keeps future pull requests small and reviewable instead of committing every local change at once.
- PR #3: Added this `FORK_NOTES.md` file, a README pointer to it, and a pull request template reminding contributors to record fork-specific changes when they materially affect behavior or direction.
- PR #4: Added a shared `internal/operations` foundation package that defines the overlapping artifact lifecycle operations in one canonical registry. This is groundwork for reducing duplication between REST and MCP without changing deployment behavior yet.
- PR #7: Made the artifact API operations the canonical definition for the overlapping artifact lifecycle surface, then derived the artifact MCP tool registrations from those same API operations. This removes the separate API-vs-MCP implementation paths for artifact deploy/list/status/share behavior while preserving the existing MCP tool names and descriptions.
- PR #8: Simplified the artifact MCP adapter by replacing the remaining repetitive output-shaping glue with one generic registration path plus a small set of explicit projection helpers. This keeps the API-first structure from PR #7, but makes the MCP side easier to read and cheaper to extend.
- PR #9: Replaced the remaining explicit artifact MCP registration calls with a small declarative binding table. The artifact API remains canonical, but the MCP adapter is now more obviously data-driven and leaves only the genuinely custom projections as code.
- PR #10: Renamed `internal/frontend` to `internal/api` so the package name matches what it now does. The directory still serves the SPA and related HTTP endpoints, but the API is no longer treated like a sidecar to “frontend” code, so the new name is a better architectural fit.
- PR #11: Added regression coverage for paginated artifact listing. The fork already had the `offset`/`limit` store, API, MCP, and dashboard behavior from upstream issue #9, so this PR makes that support explicit and safer to upstream by testing the default page size, offset handling, and max-limit clamping instead of reimplementing the feature.
- PR #12: Added outbound webhook notifications on top of the existing EventBus. Artifact lifecycle events can now be posted to external systems with per-target event filters, optional HMAC signing, request timeouts, and retries, while keeping the deploy pipeline non-blocking and reusing the same internal event stream that powers SSE.
- PR #13: Added deploy-from-repo support as a first-class artifact operation. `vibeD` can now clone a Git repository, optionally pin a branch or commit, load files from a subdirectory, and hand the result to the normal deploy pipeline through both REST and MCP, which makes the project much more usable for existing codebases and CI-driven workflows than inline file maps alone.
