---
sidebar_position: 3
---

# Artifact Lifecycle

Every artifact goes through a well-defined lifecycle from creation to deletion.

## States

| State | Description |
|-------|-------------|
| `pending` | Artifact record created, waiting for processing |
| `building` | Source code stored, container image being built |
| `deploying` | Image built, deploying to cluster |
| `running` | Successfully deployed, accessible via URL |
| `failed` | Build or deployment failed (check error field) |
| `deleted` | Removed from cluster and store |

## Flow

```
deploy_artifact called
        │
        ▼
    ┌─────────┐
    │ pending  │
    └────┬────┘
         │ Store source files
         ▼
    ┌──────────┐
    │ building │  ← Buildpacks create container image
    └────┬─────┘
         │ Image ready
         ▼
    ┌───────────┐
    │ deploying │  ← Apply manifest to cluster
    └─────┬─────┘
          │ Deployment successful
          ▼
    ┌─────────┐
    │ running │  ← URL available, serving traffic
    └─────────┘
```

If any step fails, the artifact transitions to `failed` with an error message explaining what went wrong.

## Update Flow

Calling `update_artifact` on a running artifact:
1. Stores new source files (overwrites previous)
2. Rebuilds the container image
3. Updates the deployment (new revision for Knative)
4. Returns the new URL

## Delete Flow

Calling `delete_artifact`:
1. Removes the deployment from the cluster
2. Deletes stored source code and manifests
3. Removes the artifact record from the store
