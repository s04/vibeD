---
sidebar_position: 3
---

# First Deployment

Deploy your first artifact using vibeD's MCP tools.

## Using Claude Desktop

### Option A: Stdio Transport (Direct)

Run vibeD as a local process that Claude Desktop launches directly:

```json
{
  "mcpServers": {
    "vibed": {
      "command": "/path/to/vibed",
      "args": ["--config", "/path/to/vibed.yaml"]
    }
  }
}
```

### Option B: HTTP Transport (Remote / In-Cluster)

If vibeD runs as a service (e.g. deployed to your Kind cluster), use `mcp-remote` to bridge Claude Desktop's stdio to vibeD's HTTP endpoint:

```json
{
  "mcpServers": {
    "vibed": {
      "command": "npx",
      "args": ["mcp-remote", "http://localhost:8080/mcp/"]
    }
  }
}
```

This requires a port-forward to the vibeD service:

```bash
kubectl port-forward svc/vibed 8080:8080 -n vibed-system
```

### Deploy Your First Artifact

Ask Claude to deploy a simple website:

> "Create a simple portfolio website with my name and deploy it using vibeD"

Claude will use the `deploy_artifact` tool automatically. Static HTML/CSS/JS files deploy instantly via ConfigMap (no container build needed). More complex apps (Node.js, Python, Go) are built via Buildah.

## Using MCP Inspector

For testing, use the [MCP Inspector](https://github.com/modelcontextprotocol/inspector):

```bash
npx @modelcontextprotocol/inspector ./bin/vibed --config vibed.yaml
```

Then call the `deploy_artifact` tool with:

```json
{
  "name": "hello-world",
  "files": {
    "index.html": "<!DOCTYPE html><html><body><h1>Hello from vibeD!</h1></body></html>"
  }
}
```

## Using the HTTP API

If vibeD is running in HTTP mode, you can call the MCP endpoint directly via Streamable HTTP:

```bash
# 1. Initialize session
curl -X POST http://localhost:8080/mcp/ \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{
    "protocolVersion":"2025-03-26","capabilities":{},
    "clientInfo":{"name":"curl","version":"1.0"}}}'

# 2. Use the Mcp-Session-Id header from the response for subsequent calls
```

## Check the Dashboard

After deploying, open `http://localhost:8080` to see your artifact in the dashboard with its status and access URL.
