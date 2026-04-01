# CLAUDE.md

Repository-specific working rules for AI-assisted changes in this fork.

## Pull Request Workflow

- Keep changes incremental and reviewable.
- Use staged files as the PR boundary.
- Do not bundle unrelated changes into the same PR.

## Fork Notes

- Every PR that materially changes the fork's behavior, API surface, workflow, architecture, or project direction must update `FORK_NOTES.md`.
- Write entries so they are readable to someone without chat context and understandable to the original upstream author.
- If the PR does not exist yet, add or update a short `Next PR` note in `FORK_NOTES.md`, then replace `Next PR` with the actual PR number once it exists.

## Style

- Preserve the intent and wording of existing public-facing descriptions where possible, especially MCP tool descriptions, unless there is a deliberate reason to change them.
