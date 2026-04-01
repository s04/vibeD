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

type artifactToolBinding func(*mcp.Server, *orchestrator.Orchestrator, config.LimitsConfig)

var artifactToolBindings = []artifactToolBinding{
	identityArtifactBinding("artifacts.deploy", func(ctx context.Context, orch *orchestrator.Orchestrator, limits config.LimitsConfig, input operations.DeployArtifactRequest) (*orchestrator.DeployResult, error) {
		return operations.DeployArtifact(ctx, orch, limits, input)
	}),
	identityArtifactBinding("artifacts.update", func(ctx context.Context, orch *orchestrator.Orchestrator, limits config.LimitsConfig, input operations.UpdateArtifactRequest) (*orchestrator.DeployResult, error) {
		return operations.UpdateArtifact(ctx, orch, limits, input)
	}),
	identityArtifactBinding("artifacts.list", func(ctx context.Context, orch *orchestrator.Orchestrator, _ config.LimitsConfig, input operations.ListArtifactsRequest) (*operations.ListArtifactsResponse, error) {
		return operations.ListArtifacts(ctx, orch, input)
	}),
	identityArtifactBinding("artifacts.status", func(ctx context.Context, orch *orchestrator.Orchestrator, _ config.LimitsConfig, input operations.GetArtifactStatusRequest) (*api.Artifact, error) {
		return operations.GetArtifactStatus(ctx, orch, input)
	}),
	projectedArtifactBinding("artifacts.delete", func(ctx context.Context, orch *orchestrator.Orchestrator, _ config.LimitsConfig, input operations.DeleteArtifactRequest) (*operations.DeleteArtifactResponse, error) {
		return operations.DeleteArtifact(ctx, orch, input)
	}, func(_ context.Context, _ *orchestrator.Orchestrator, _ config.LimitsConfig, _ operations.DeleteArtifactRequest, result *operations.DeleteArtifactResponse) (*messageOutput, error) {
		return &messageOutput{Message: fmt.Sprintf("Artifact %q deleted successfully.", result.ID)}, nil
	}),
	projectedArtifactBinding("artifacts.logs", func(ctx context.Context, orch *orchestrator.Orchestrator, limits config.LimitsConfig, input operations.GetArtifactLogsRequest) (*operations.GetArtifactLogsResponse, error) {
		return operations.GetArtifactLogs(ctx, orch, limits, input)
	}, func(_ context.Context, _ *orchestrator.Orchestrator, _ config.LimitsConfig, _ operations.GetArtifactLogsRequest, result *operations.GetArtifactLogsResponse) (*logsOutput, error) {
		return &logsOutput{Logs: strings.Join(result.Logs, "\n")}, nil
	}),
	projectedArtifactBinding("artifacts.targets", func(ctx context.Context, orch *orchestrator.Orchestrator, _ config.LimitsConfig, _ struct{}) (*operations.ListTargetsResponse, error) {
		targets, err := operations.ListTargets(ctx, orch)
		if err != nil {
			return nil, err
		}
		return &targets, nil
	}, func(_ context.Context, _ *orchestrator.Orchestrator, _ config.LimitsConfig, _ struct{}, result *operations.ListTargetsResponse) (*targetsOutput, error) {
		return &targetsOutput{Targets: []api.TargetInfo(*result)}, nil
	}),
	identityArtifactBinding("artifacts.versions.list", func(ctx context.Context, orch *orchestrator.Orchestrator, _ config.LimitsConfig, input operations.ListVersionsRequest) (*operations.ListVersionsResponse, error) {
		return operations.ListVersions(ctx, orch, input)
	}),
	projectedArtifactBinding("artifacts.rollback", func(ctx context.Context, orch *orchestrator.Orchestrator, _ config.LimitsConfig, input operations.RollbackArtifactRequest) (*orchestrator.DeployResult, error) {
		return operations.RollbackArtifact(ctx, orch, input)
	}, func(ctx context.Context, orch *orchestrator.Orchestrator, _ config.LimitsConfig, input operations.RollbackArtifactRequest, result *orchestrator.DeployResult) (*rollbackArtifactOutput, error) {
		return projectRollbackArtifactResult(ctx, orch, input, result)
	}),
	projectedArtifactBinding("artifacts.share", func(ctx context.Context, orch *orchestrator.Orchestrator, _ config.LimitsConfig, input operations.ShareArtifactRequest) (*operations.ShareArtifactResponse, error) {
		return operations.ShareArtifact(ctx, orch, input)
	}, func(_ context.Context, _ *orchestrator.Orchestrator, _ config.LimitsConfig, _ operations.ShareArtifactRequest, result *operations.ShareArtifactResponse) (*artifactMessageOutput, error) {
		return &artifactMessageOutput{
			ArtifactID: result.ArtifactID,
			Message:    fmt.Sprintf("Artifact shared with %d user(s).", len(result.SharedWith)),
		}, nil
	}),
	projectedArtifactBinding("artifacts.unshare", func(ctx context.Context, orch *orchestrator.Orchestrator, _ config.LimitsConfig, input operations.UnshareArtifactRequest) (*operations.UnshareArtifactResponse, error) {
		return operations.UnshareArtifact(ctx, orch, input)
	}, func(_ context.Context, _ *orchestrator.Orchestrator, _ config.LimitsConfig, _ operations.UnshareArtifactRequest, result *operations.UnshareArtifactResponse) (*artifactMessageOutput, error) {
		return &artifactMessageOutput{
			ArtifactID: result.ArtifactID,
			Message:    fmt.Sprintf("Access revoked for %d user(s).", len(result.Removed)),
		}, nil
	}),
	identityArtifactBinding("artifacts.share-links.create", func(ctx context.Context, orch *orchestrator.Orchestrator, _ config.LimitsConfig, input operations.CreateShareLinkRequest) (*api.ShareLink, error) {
		return operations.CreateShareLink(ctx, orch, input)
	}),
	projectedArtifactBinding("artifacts.share-links.list", func(ctx context.Context, orch *orchestrator.Orchestrator, _ config.LimitsConfig, input operations.ListShareLinksRequest) (*[]api.ShareLink, error) {
		links, err := operations.ListShareLinks(ctx, orch, input)
		if err != nil {
			return nil, err
		}
		return &links, nil
	}, func(_ context.Context, _ *orchestrator.Orchestrator, _ config.LimitsConfig, _ operations.ListShareLinksRequest, result *[]api.ShareLink) (*shareLinksOutput, error) {
		return &shareLinksOutput{Links: *result}, nil
	}),
	identityArtifactBinding("artifacts.share-links.revoke", func(ctx context.Context, orch *orchestrator.Orchestrator, _ config.LimitsConfig, input operations.RevokeShareLinkRequest) (*operations.RevokeShareLinkResponse, error) {
		return operations.RevokeShareLink(ctx, orch, input)
	}),
}

func registerArtifactOperations(server *mcp.Server, orch *orchestrator.Orchestrator, limits config.LimitsConfig) {
	for _, bind := range artifactToolBindings {
		bind(server, orch, limits)
	}
}

func identityArtifactBinding[In any, Out any](
	operationID string,
	handler func(context.Context, *orchestrator.Orchestrator, config.LimitsConfig, In) (*Out, error),
) artifactToolBinding {
	return projectedArtifactBinding(operationID, handler, func(_ context.Context, _ *orchestrator.Orchestrator, _ config.LimitsConfig, _ In, result *Out) (*Out, error) {
		return result, nil
	})
}

func projectedArtifactBinding[In any, APIOut any, MCPOut any](
	operationID string,
	handler func(context.Context, *orchestrator.Orchestrator, config.LimitsConfig, In) (*APIOut, error),
	project func(context.Context, *orchestrator.Orchestrator, config.LimitsConfig, In, *APIOut) (*MCPOut, error),
) artifactToolBinding {
	return func(server *mcp.Server, orch *orchestrator.Orchestrator, limits config.LimitsConfig) {
		registerArtifactTool(server, operationID, func(ctx context.Context, input In) (*APIOut, error) {
			return handler(ctx, orch, limits, input)
		}, func(ctx context.Context, input In, result *APIOut) (*MCPOut, error) {
			return project(ctx, orch, limits, input, result)
		})
	}
}

func registerArtifactTool[In any, APIOut any, MCPOut any](
	server *mcp.Server,
	operationID string,
	handler func(context.Context, In) (*APIOut, error),
	project func(context.Context, In, *APIOut) (*MCPOut, error),
) {
	op := operations.MustArtifactOperation(operationID)
	mcp.AddTool(server, &mcp.Tool{
		Name:        op.MCP.ToolName,
		Description: op.Description,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input In) (*mcp.CallToolResult, *MCPOut, error) {
		result, err := handler(ctx, input)
		if err != nil {
			return nil, nil, err
		}

		projected, err := project(ctx, input, result)
		if err != nil {
			return nil, nil, err
		}
		return nil, projected, nil
	})
}

func projectRollbackArtifactResult(ctx context.Context, orch *orchestrator.Orchestrator, input operations.RollbackArtifactRequest, result *orchestrator.DeployResult) (*rollbackArtifactOutput, error) {
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
}

type messageOutput struct {
	Message string `json:"message"`
}

type logsOutput struct {
	Logs string `json:"logs"`
}

type targetsOutput struct {
	Targets []api.TargetInfo `json:"targets"`
}

type artifactMessageOutput struct {
	ArtifactID string `json:"artifact_id"`
	Message    string `json:"message"`
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

type shareLinksOutput struct {
	Links []api.ShareLink `json:"links"`
}
