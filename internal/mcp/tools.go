package mcp

import (
	"github.com/vibed-project/vibeD/internal/config"
	"github.com/vibed-project/vibeD/internal/orchestrator"
	"github.com/vibed-project/vibeD/internal/store"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RegisterTools registers all vibeD MCP tools with the server.
func RegisterTools(server *mcp.Server, orch *orchestrator.Orchestrator, limits config.LimitsConfig, userStore store.UserStore) {
	registerArtifactOperations(server, orch, limits)
	if userStore != nil {
		registerListUsersTool(server, userStore)
		registerGetUserTool(server, userStore)
		registerListDepartmentsTool(server, userStore)
		registerCreateDepartmentTool(server, userStore)
	}
}
