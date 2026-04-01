---
sidebar_position: 3
---

# deploy_from_repo

Deploy an artifact directly from a Git repository instead of passing source files inline.

## Input Schema

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `repo_url` | string | Yes | Git repository URL to clone |
| `name` | string | Yes | Unique DNS-safe artifact name |
| `path` | string | No | Subdirectory within the repository to deploy |
| `branch` | string | No | Branch or tag to clone |
| `commit` | string | No | Specific commit SHA to fetch and check out |
| `auth_token` | string | No | HTTPS auth token for private repositories |
| `language` | string | No | Language hint (nodejs, python, go, static) |
| `target` | string | No | Deployment target (auto, knative, kubernetes) |
| `env_vars` | object | No | Environment variables for the artifact |
| `secret_refs` | object | No | Map of env var name to K8s Secret reference (`secret-name:key`) |
| `port` | number | No | Port the app listens on (auto-detected) |

## Example

```json
{
  "repo_url": "https://github.com/example/project",
  "path": "app",
  "branch": "main",
  "name": "project-app",
  "target": "auto"
}
```

## What Happens

1. **Clones** the repository into a temporary directory
2. **Checks out** the requested branch or commit if provided
3. **Reads** files from the selected subdirectory, respecting Git ignore rules
4. **Applies** the same file-count and total-size limits as `deploy_artifact`
5. **Starts** the normal asynchronous deploy flow and returns the artifact ID immediately

## Notes

- `auth_token` is only used for `https://` repository URLs
- `path` is treated as the deployment root, so returned files are relative to that subdirectory
- The repository checkout is removed after files are loaded, even on failure
