---
sidebar_position: 3
---

# Container Registry

vibeD can push built container images to an OCI-compatible registry.

## Configuration

```yaml
registry:
  enabled: true
  url: "ghcr.io/myorg/vibed-artifacts"
```

## Supported Registries

Any OCI-compatible registry works:

- **GitHub Container Registry** (`ghcr.io`)
- **Docker Hub** (`docker.io`)
- **Google Container Registry** (`gcr.io`)
- **Amazon ECR** (`*.dkr.ecr.*.amazonaws.com`)
- **Azure Container Registry** (`*.azurecr.io`)

## Authentication

vibeD uses the standard Docker credential chain:

1. `~/.docker/config.json`
2. Docker credential helpers
3. Cloud provider credential helpers (ECR, GCR, ACR)

For GitHub Container Registry:

```bash
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin
```
