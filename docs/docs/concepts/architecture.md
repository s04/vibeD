---
sidebar_position: 1
---

# Architecture

vibeD is a single Go binary that serves three concerns:

1. **MCP Server** - Protocol endpoint for AI coding tools
2. **REST API** - JSON API for the dashboard
3. **Web Dashboard** - React SPA for monitoring artifacts

## System Architecture

```
┌─────────────────────────────────────┐
│            AI Coding Tool           │
│     (Claude, Gemini, ChatGPT)       │
└──────────────┬──────────────────────┘
               │ MCP Protocol
               v
┌──────────────────────────────────────┐
│              vibeD                   │
│                                      │
│  ┌──────────┐  ┌──────────────────┐  │
│  │ MCP      │  │ REST API         │  │
│  │ Server   │  │ + Dashboard      │  │
│  └────┬─────┘  └────────┬────────┘  │
│       └───────┬──────────┘           │
│               v                      │
│       ┌──────────────┐               │
│       │ Orchestrator │               │
│       └──┬───┬───┬───┘               │
│          │   │   │                   │
│   ┌──────┘   │   └──────┐           │
│   v          v          v            │
│ ┌──────┐ ┌──────┐ ┌─────────┐       │
│ │Store │ │Build │ │Deployer │       │
│ │      │ │(ah)  │ │Factory  │       │
│ └──────┘ └──────┘ └─┬───┬──┘       │
│                      │   │           │
│              ┌───────┘   └───────┐   │
│              v                   v   │
│         ┌─────────┐      ┌─────────┐│
│         │Knative  │      │   K8s   ││
│         └─────────┘      └─────────┘│
└──────────────────────────────────────┘
```

## Key Design Principles

- **Interface-driven** - Every subsystem (Builder, Deployer, Storage, Store) is behind a Go interface with 2+ implementations
- **Single binary** - No microservices, no sidecars, one container
- **Environment detection** - Auto-discovers available deployment targets by checking CRDs
- **Fail-safe** - If Knative isn't available, falls back to plain Kubernetes

## Subsystems

| Subsystem | Interface | Implementations |
|-----------|-----------|-----------------|
| **Store** | `ArtifactStore` | In-memory, ConfigMap (both support owner-scoped listing) |
| **Storage** | `Storage` | Local filesystem, GitHub, GitLab, UserStorageRouter (per-user routing) |
| **Builder** | `Builder` | Buildah (K8s Jobs) — auto-generates Dockerfiles per language |
| **Deployer** | `Deployer` | Knative, Kubernetes, wasmCloud |
| **Registry** | `Registry` | Any OCI-compatible registry |
