package mcp

import (
	"context"
	"fmt"

	"github.com/maxkorbacher/vibed/internal/orchestrator"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type shareArtifactInput struct {
	ArtifactID string   `json:"artifact_id" jsonschema:"ID of the artifact to share"`
	UserIDs    []string `json:"user_ids" jsonschema:"List of user IDs to grant read-only access to"`
}

type shareArtifactOutput struct {
	ArtifactID string `json:"artifact_id"`
	Message    string `json:"message"`
}

func registerShareTool(server *mcp.Server, orch *orchestrator.Orchestrator) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "share_artifact",
		Description: "Share a deployed artifact with other users, granting them read-only access (view status, logs, URL). Only the artifact owner or an admin can share.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input shareArtifactInput) (*mcp.CallToolResult, *shareArtifactOutput, error) {
		if err := orch.ShareArtifact(ctx, input.ArtifactID, input.UserIDs); err != nil {
			return nil, nil, err
		}
		return nil, &shareArtifactOutput{
			ArtifactID: input.ArtifactID,
			Message:    fmt.Sprintf("Artifact shared with %d user(s).", len(input.UserIDs)),
		}, nil
	})
}

type unshareArtifactInput struct {
	ArtifactID string   `json:"artifact_id" jsonschema:"ID of the artifact to unshare"`
	UserIDs    []string `json:"user_ids" jsonschema:"List of user IDs to revoke access from"`
}

type unshareArtifactOutput struct {
	ArtifactID string `json:"artifact_id"`
	Message    string `json:"message"`
}

func registerUnshareTool(server *mcp.Server, orch *orchestrator.Orchestrator) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "unshare_artifact",
		Description: "Revoke read-only access to a deployed artifact from specific users. Only the artifact owner or an admin can unshare.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input unshareArtifactInput) (*mcp.CallToolResult, *unshareArtifactOutput, error) {
		if err := orch.UnshareArtifact(ctx, input.ArtifactID, input.UserIDs); err != nil {
			return nil, nil, err
		}
		return nil, &unshareArtifactOutput{
			ArtifactID: input.ArtifactID,
			Message:    fmt.Sprintf("Access revoked for %d user(s).", len(input.UserIDs)),
		}, nil
	})
}
