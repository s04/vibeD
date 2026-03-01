package mcp

import (
	"context"

	"github.com/maxkorbacher/vibed/internal/orchestrator"
	"github.com/maxkorbacher/vibed/pkg/api"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type listTargetsInput struct{}

type listTargetsOutput struct {
	Targets []api.TargetInfo `json:"targets"`
}

func registerTargetsTool(server *mcp.Server, orch *orchestrator.Orchestrator) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_deployment_targets",
		Description: "Show which deployment backends (Knative, Kubernetes, wasmCloud) are available in the current cluster.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, _ listTargetsInput) (*mcp.CallToolResult, *listTargetsOutput, error) {
		targets := orch.ListTargets()
		return nil, &listTargetsOutput{Targets: targets}, nil
	})
}
