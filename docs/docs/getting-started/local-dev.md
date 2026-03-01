---
sidebar_position: 2
---

# Local Development Setup

vibeD uses **Podman + Kind** for local development. This guide sets up a complete environment.

## One-Command Setup

```bash
make dev
```

This creates a Kind cluster, installs Knative Serving with Kourier, and builds vibeD.

## Manual Setup

### 1. Create Kind Cluster

```bash
make setup-cluster
```

This creates a Kind cluster named `vibed-dev` with port mappings:
- Port 80 → Kourier ingress (for Knative services)
- Port 443 → Kourier ingress (HTTPS)

### 2. Install Knative Serving

```bash
make install-knative
```

This installs:
- Knative Serving CRDs and core components
- Kourier as the ingress layer (NodePort mode)
- sslip.io DNS for automatic URL resolution

### 3. Build and Run vibeD

```bash
# Build
make build

# Run in HTTP mode (dashboard + MCP endpoint)
make run-http
```

### 4. Access the Dashboard

Open `http://localhost:8080` in your browser. You should see the vibeD dashboard with deployment target status.

## Deployed Artifact URLs

With sslip.io DNS, artifacts deployed via Knative get URLs like:

```
http://my-app.default.127.0.0.1.sslip.io
```

These resolve to localhost and route through Kourier on port 80.

## Teardown

```bash
make teardown
```
