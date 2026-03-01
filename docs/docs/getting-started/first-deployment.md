---
sidebar_position: 3
---

# First Deployment

Deploy your first artifact using vibeD's MCP tools.

## Using Claude Desktop

1. Add vibeD as an MCP server in your Claude Desktop config:

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

2. Ask Claude to deploy a simple website:

> "Create a simple portfolio website with my name and deploy it using vibeD"

Claude will use the `deploy_artifact` tool automatically to build and deploy the site.

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

If vibeD is running in HTTP mode, you can also use the MCP HTTP endpoint:

```bash
curl -X POST http://localhost:8080/mcp/ \
  -H "Content-Type: application/json" \
  -d '...'
```

## Check the Dashboard

After deploying, open `http://localhost:8080` to see your artifact in the dashboard with its status and access URL.
