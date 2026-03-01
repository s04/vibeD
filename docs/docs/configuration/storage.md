---
sidebar_position: 2
---

# Storage Backends

vibeD stores artifact source code and deployment manifests using pluggable storage backends.

## Local Filesystem (Default)

Stores files on disk. Best for development and single-node setups.

```yaml
storage:
  backend: "local"
  local:
    basePath: "/data/vibed/artifacts"
```

Directory structure:
```
/data/vibed/artifacts/
├── {artifact-id}/
│   ├── src/           # Source files
│   │   ├── index.html
│   │   └── style.css
│   └── manifests/     # Deployment manifests
│       └── knative-service.yaml
```

## GitHub Repository

Stores artifacts in a GitHub repository for versioning and collaboration.

```yaml
storage:
  backend: "github"
  github:
    owner: "myorg"
    repo: "vibed-artifacts"
    branch: "main"
```

Requires the `GITHUB_TOKEN` environment variable. Each artifact gets a folder in the repo:

```
vibed-artifacts/
├── artifacts/
│   ├── my-portfolio/
│   │   ├── src/
│   │   └── manifests/
│   └── chat-app/
│       ├── src/
│       └── manifests/
```

Files are committed atomically using the Git Trees API.
