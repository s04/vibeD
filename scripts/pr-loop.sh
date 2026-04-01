#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  scripts/pr-loop.sh "PR title" --body "PR description"
  scripts/pr-loop.sh "PR title" --body-file path/to/body.md
  echo "PR description" | scripts/pr-loop.sh "PR title"

Behavior:
  1. Verifies the current repo is on main
  2. Fast-forwards local main from origin/main
  3. Creates a branch from the title
  4. Commits all current changes with the PR title as the commit message
  5. Pushes the branch
  6. Opens a PR against main
  7. Merges the PR
  8. Deletes the remote branch and fast-forwards local main

Options:
  --body TEXT         Inline PR body
  --body-file PATH    Read PR body from a file
  --merge-method ARG  merge | squash | rebase (default: squash)
  --dry-run           Print planned actions without changing anything
  -h, --help          Show this help
EOF
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required command: $1" >&2
    exit 1
  fi
}

slugify() {
  printf '%s' "$1" \
    | tr '[:upper:]' '[:lower:]' \
    | sed -E 's/[^a-z0-9]+/-/g; s/^-+//; s/-+$//; s/-{2,}/-/g'
}

run() {
  if [[ "$DRY_RUN" == "true" ]]; then
    printf '+ %q' "$1"
    shift
    for arg in "$@"; do
      printf ' %q' "$arg"
    done
    printf '\n'
    return 0
  fi
  "$@"
}

require_clean_main_branch() {
  local current_branch
  current_branch="$(git branch --show-current)"
  if [[ "$current_branch" != "main" ]]; then
    echo "This script must be started from local main. Current branch: $current_branch" >&2
    exit 1
  fi
}

TITLE=""
BODY=""
BODY_FILE=""
MERGE_METHOD="squash"
DRY_RUN="false"

if [[ $# -eq 0 ]]; then
  usage
  exit 1
fi

TITLE="$1"
shift

while [[ $# -gt 0 ]]; do
  case "$1" in
    --body)
      BODY="${2:-}"
      shift 2
      ;;
    --body-file)
      BODY_FILE="${2:-}"
      shift 2
      ;;
    --merge-method)
      MERGE_METHOD="${2:-}"
      shift 2
      ;;
    --dry-run)
      DRY_RUN="true"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      usage
      exit 1
      ;;
  esac
done

require_cmd git
require_cmd gh

if [[ -z "$TITLE" ]]; then
  echo "PR title is required." >&2
  exit 1
fi

if [[ -n "$BODY" && -n "$BODY_FILE" ]]; then
  echo "Use either --body or --body-file, not both." >&2
  exit 1
fi

if [[ -n "$BODY_FILE" ]]; then
  if [[ ! -f "$BODY_FILE" ]]; then
    echo "Body file not found: $BODY_FILE" >&2
    exit 1
  fi
  BODY="$(cat "$BODY_FILE")"
elif [[ -z "$BODY" && ! -t 0 ]]; then
  BODY="$(cat)"
fi

if [[ -z "$BODY" ]]; then
  BODY=$'No PR description provided.\n'
fi

case "$MERGE_METHOD" in
  merge|squash|rebase) ;;
  *)
    echo "Invalid merge method: $MERGE_METHOD" >&2
    exit 1
    ;;
esac

require_clean_main_branch

if [[ -z "$(git status --short)" ]]; then
  echo "No local changes to commit." >&2
  exit 1
fi

timestamp="$(date +%Y%m%d-%H%M%S)"
slug="$(slugify "$TITLE")"
if [[ -z "$slug" ]]; then
  slug="change"
fi
branch="pr/${timestamp}-${slug}"

tmp_body="$(mktemp)"
trap 'rm -f "$tmp_body"' EXIT
printf '%s\n' "$BODY" >"$tmp_body"

run git fetch origin main
run git pull --ff-only origin main
run git switch -c "$branch"
run git add -A
run git commit -m "$TITLE"
run git push -u origin "$branch"
run gh pr create --base main --head "$branch" --title "$TITLE" --body-file "$tmp_body"
run gh pr merge "$branch" "--${MERGE_METHOD}" --delete-branch
run git switch main
run git pull --ff-only origin main

echo "Merged PR for branch $branch and synced local main."
