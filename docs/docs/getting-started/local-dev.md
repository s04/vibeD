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

## Accessing Deployed Artifacts

With sslip.io DNS, artifacts deployed via Knative get URLs like:

```
http://my-app.default.127.0.0.1.sslip.io
```

These resolve to `127.0.0.1` and route through Kourier on port 80.

### Port-Forward Access

If your Kind cluster doesn't expose port 80 to the host (common with Podman), use a port-forward:

```bash
kubectl port-forward svc/kourier 8081:80 -n kourier-system
```

Then access artifacts by appending the port to the URL:

```
http://my-app.default.127.0.0.1.sslip.io:8081
```

### DNS Troubleshooting

If your browser shows `DNS_PROBE_FINISHED_NXDOMAIN`, your router's DNS may not resolve sslip.io domains. Fix by switching to a public DNS resolver:

**macOS**: System Settings → Network → Wi-Fi → Details → DNS → add `8.8.8.8`

Or add entries manually to `/etc/hosts`:

```bash
sudo sh -c 'echo "127.0.0.1 my-app.default.127.0.0.1.sslip.io" >> /etc/hosts'
```

## Teardown

```bash
make teardown
```
