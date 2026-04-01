package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/vibed-project/vibeD/internal/config"
	"github.com/vibed-project/vibeD/internal/operations"
	"github.com/vibed-project/vibeD/internal/orchestrator"
	"github.com/vibed-project/vibeD/pkg/api"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerArtifactOperations(server *mcp.Server, orch *orchestrator.Orchestrator, limits config.LimitsConfig) {
	registerArtifactTool(server, "artifacts.deploy", func(ctx context.Context, input operations.DeployArtifactRequest) (*orchestrator.DeployResult, error) {
		return operations.DeployArtifact(ctx, orch, limits, input)
	})

	registerArtifactTool(server, "artifacts.update", func(ctx context.Context, input operations.UpdateArtifactRequest) (*orchestrator.DeployResult, error) {
		return operations.UpdateArtifact(ctx, orch, limits, input)
	})

	registerArtifactTool(server, "artifacts.list", func(ctx context.Context, input operations.ListArtifactsRequest) (*operations.ListArtifactsResponse, error) {
		return operations.ListArtifacts(ctx, orch, input)
	})

	registerArtifactTool(server, "artifacts.status", func(ctx context.Context, input operations.GetArtifactStatusRequest) (*api.Artifact, error) {
		return operations.GetArtifactStatus(ctx, orch, input)
	})

	registerArtifactTool(server, "artifacts.delete", func(ctx context.Context, input operations.DeleteArtifactRequest) (*deleteArtifactOutput, error) {
		result, err := operations.DeleteArtifact(ctx, orch, input)
		if err != nil {
			return nil, err
		}
		return &deleteArtifactOutput{
			Message: fmt.Sprintf("Artifact %q deleted successfully.", result.ID),
		}, nil
	})

	registerArtifactTool(server, "artifacts.logs", func(ctx context.Context, input operations.GetArtifactLogsRequest) (*getArtifactLogsOutput, error) {
		result, err := operations.GetArtifactLogs(ctx, orch, limits, input)
		if err != nil {
			return nil, err
		}
		return &getArtifactLogsOutput{Logs: strings.Join(result.Logs, "\n")}, nil
	})

	registerArtifactTool(server, "artifacts.targets", func(ctx context.Context, _ struct{}) (*listTargetsOutput, error) {
		targets, err := operations.ListTargets(ctx, orch)
		if err != nil {
			return nil, err
		}
		return &listTargetsOutput{Targets: []api.TargetInfo(targets)}, nil
	})

	registerArtifactTool(server, "artifacts.versions.list", func(ctx context.Context, input operations.ListVersionsRequest) (*operations.ListVersionsResponse, error) {
		return operations.ListVersions(ctx, orch, input)
	})

	registerArtifactTool(server, "artifacts.rollback", func(ctx context.Context, input operations.RollbackArtifactRequest) (*rollbackArtifactOutput, error) {
		result, err := operations.RollbackArtifact(ctx, orch, input)
		if err != nil {
			return nil, err
		}

		artifact, _ := operations.GetArtifactStatus(ctx, orch, operations.GetArtifactStatusRequest{ArtifactID: result.ArtifactID})
		newVersion := 0
		if artifact != nil {
			newVersion = artifact.Version
		}

		return &rollbackArtifactOutput{
			ArtifactID: result.ArtifactID,
			Name:       result.Name,
			URL:        result.URL,
			Target:     result.Target,
			Status:     result.Status,
			ImageRef:   result.ImageRef,
			NewVersion: newVersion,
			Message:    fmt.Sprintf("Successfully rolled back to version %d. New version is v%d.", input.Version, newVersion),
		}, nil
	})

	registerArtifactTool(server, "artifacts.share", func(ctx context.Context, input operations.ShareArtifactRequest) (*shareArtifactOutput, error) {
		result, err := operations.ShareArtifact(ctx, orch, input)
		if err != nil {
			return nil, err
		}
		return &shareArtifactOutput{
			ArtifactID: result.ArtifactID,
			Message:    fmt.Sprintf("Artifact shared with %d user(s).", len(result.SharedWith)),
		}, nil
	})

	registerArtifactTool(server, "artifacts.unshare", func(ctx context.Context, input operations.UnshareArtifactRequest) (*unshareArtifactOutput, error) {
		result, err := operations.UnshareArtifact(ctx, orch, input)
		if err != nil {
			return nil, err
		}
		return &unshareArtifactOutput{
			ArtifactID: result.ArtifactID,
			Message:    fmt.Sprintf("Access revoked for %d user(s).", len(result.Removed)),
		}, nil
	})

	registerArtifactTool(server, "artifacts.share-links.create", func(ctx context.Context, input operations.CreateShareLinkRequest) (*api.ShareLink, error) {
		return operations.CreateShareLink(ctx, orch, input)
	})

	registerArtifactTool(server, "artifacts.share-links.list", func(ctx context.Context, input operations.ListShareLinksRequest) (*listShareLinksOutput, error) {
		links, err := operations.ListShareLinks(ctx, orch, input)
		if err != nil {
			return nil, err
		}
		return &listShareLinksOutput{Links: links}, nil
	})

	registerArtifactTool(server, "artifacts.share-links.revoke", func(ctx context.Context, input operations.RevokeShareLinkRequest) (*operations.RevokeShareLinkResponse, error) {
		return operations.RevokeShareLink(ctx, orch, input)
	})
}

func registerArtifactTool[In any, Out any](server *mcp.Server, operationID string, handler func(context.Context, In) (*Out, error)) {
	op := operations.MustArtifactOperation(operationID)
	mcp.AddTool(server, &mcp.Tool{
		Name:        op.MCP.ToolName,
		Description: op.Description,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input In) (*mcp.CallToolResult, *Out, error) {
		result, err := handler(ctx, input)
		if err != nil {
			return nil, nil, err
		}
		return nil, result, nil
	})
}

type deleteArtifactOutput struct {
	Message string `json:"message"`
}

type getArtifactLogsOutput struct {
	Logs string `json:"logs"`
}

type listTargetsOutput struct {
	Targets []api.TargetInfo `json:"targets"`
}

type rollbackArtifactOutput struct {
	ArtifactID string `json:"artifact_id"`
	Name       string `json:"name"`
	URL        string `json:"url"`
	Target     string `json:"target"`
	Status     string `json:"status"`
	ImageRef   string `json:"image_ref"`
	NewVersion int    `json:"new_version"`
	Message    string `json:"message"`
}

type shareArtifactOutput struct {
	ArtifactID string `json:"artifact_id"`
	Message    string `json:"message"`
}

type unshareArtifactOutput struct {
	ArtifactID string `json:"artifact_id"`
	Message    string `json:"message"`
}

type listShareLinksOutput struct {
	Links []api.ShareLink `json:"links"`
}
