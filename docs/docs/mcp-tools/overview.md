---
sidebar_position: 1
---

# MCP Tools Overview

vibeD exposes 7 MCP tools that AI coding tools can call to deploy and manage artifacts.

## Available Tools

| Tool | Description |
|------|-------------|
| [`deploy_artifact`](./deploy-artifact) | Deploy source files as a web artifact |
| `update_artifact` | Update an existing artifact with new files |
| `list_artifacts` | List all deployed artifacts |
| `get_artifact_status` | Get detailed status for one artifact |
| `get_artifact_logs` | Retrieve pod logs for debugging |
| `delete_artifact` | Stop and remove an artifact |
| `list_deployment_targets` | Show available deployment backends |

## Transport Modes

vibeD supports three transport modes for MCP:

- **stdio** - Standard input/output (for CLI integration like Claude Desktop)
- **http** - HTTP/SSE endpoint at `/mcp/` (for networked access)
- **both** - Runs both simultaneously

Configure via `server.transport` in `vibed.yaml` or the `--transport` CLI flag.
