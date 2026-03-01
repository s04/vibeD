package mcp

import (
	"github.com/maxkorbacher/vibed/internal/orchestrator"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RegisterTools registers all vibeD MCP tools with the server.
func RegisterTools(server *mcp.Server, orch *orchestrator.Orchestrator) {
	registerDeployTool(server, orch)
	registerListTool(server, orch)
	registerStatusTool(server, orch)
	registerDeleteTool(server, orch)
	registerLogsTool(server, orch)
	registerTargetsTool(server, orch)
	registerUpdateTool(server, orch)
}
