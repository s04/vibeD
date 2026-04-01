package operations

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/vibed-project/vibeD/internal/config"
	"github.com/vibed-project/vibeD/internal/orchestrator"
	"github.com/vibed-project/vibeD/pkg/api"
)

// MCPMetadata captures the MCP-specific projection of a canonical API operation.
type MCPMetadata struct {
	ToolName string
}

// Operation describes one canonical artifact API operation.
type Operation struct {
	ID          string
	Method      string
	Path        string
	Summary     string
	Description string
	MCP         *MCPMetadata
}

type DeployArtifactRequest struct {
	Name       string            `json:"name"`
	Files      map[string]string `json:"files"`
	Language   string            `json:"language,omitempty"`
	Target     string            `json:"target,omitempty"`
	EnvVars    map[string]string `json:"env_vars,omitempty"`
	SecretRefs map[string]string `json:"secret_refs,omitempty"`
	Port       int               `json:"port,omitempty"`
}

type UpdateArtifactRequest struct {
	ArtifactID string            `json:"artifact_id"`
	Files      map[string]string `json:"files"`
	EnvVars    map[string]string `json:"env_vars,omitempty"`
	SecretRefs map[string]string `json:"secret_refs,omitempty"`
}

type ListArtifactsRequest struct {
	Status string `json:"status,omitempty"`
	Offset int    `json:"offset,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

type ListArtifactsResponse struct {
	Artifacts []api.ArtifactSummary `json:"artifacts"`
	Total     int                   `json:"total"`
	Offset    int                   `json:"offset"`
	Limit     int                   `json:"limit"`
}

type GetArtifactStatusRequest struct {
	ArtifactID string `json:"artifact_id"`
}

type DeleteArtifactRequest struct {
	ArtifactID string `json:"artifact_id"`
}

type DeleteArtifactResponse struct {
	Status string `json:"status"`
	ID     string `json:"id"`
}

type GetArtifactLogsRequest struct {
	ArtifactID string `json:"artifact_id"`
	Lines      int    `json:"lines,omitempty"`
}

type GetArtifactLogsResponse struct {
	ArtifactID string   `json:"artifact_id"`
	Logs       []string `json:"logs"`
}

type ListTargetsResponse []api.TargetInfo

type ListVersionsRequest struct {
	ArtifactID string `json:"artifact_id"`
}

type ListVersionsResponse struct {
	ArtifactID string                `json:"artifact_id"`
	Versions   []api.ArtifactVersion `json:"versions"`
}

type RollbackArtifactRequest struct {
	ArtifactID string `json:"artifact_id"`
	Version    int    `json:"version"`
}

type ShareArtifactRequest struct {
	ArtifactID string   `json:"artifact_id"`
	UserIDs    []string `json:"user_ids"`
}

type ShareArtifactResponse struct {
	ArtifactID string   `json:"artifact_id"`
	SharedWith []string `json:"shared_with"`
	Status     string   `json:"status"`
}

type UnshareArtifactRequest struct {
	ArtifactID string   `json:"artifact_id"`
	UserIDs    []string `json:"user_ids"`
}

type UnshareArtifactResponse struct {
	ArtifactID string   `json:"artifact_id"`
	Removed    []string `json:"removed"`
	Status     string   `json:"status"`
}

type CreateShareLinkRequest struct {
	ArtifactID string `json:"artifact_id"`
	Password   string `json:"password,omitempty"`
	ExpiresIn  string `json:"expires_in,omitempty"`
}

type ListShareLinksRequest struct {
	ArtifactID string `json:"artifact_id"`
}

type RevokeShareLinkRequest struct {
	Token string `json:"token"`
}

type RevokeShareLinkResponse struct {
	Status string `json:"status"`
}

var artifactOperations = []Operation{
	{
		ID:          "artifacts.deploy",
		Method:      "POST",
		Path:        "/api/artifacts",
		Summary:     "Deploy a web artifact (website, web app) to the cluster.",
		Description: "Deploy a web artifact (website, web app) to the cluster. Provide source files and vibeD handles building a container image and deploying it. Returns immediately with status \"building\" and an artifact_id — use get_artifact_status to poll until status is \"running\". Knative is used when available (auto-scaling, clean URLs), otherwise falls back to plain Kubernetes. Set target explicitly to override. For Go apps, go.mod is auto-generated if not provided.",
		MCP:         &MCPMetadata{ToolName: "deploy_artifact"},
	},
	{
		ID:          "artifacts.update",
		Method:      "PUT",
		Path:        "/api/artifacts/{id}",
		Summary:     "Update an existing deployed artifact with new source files.",
		Description: "Update an existing deployed artifact with new source files. Triggers a rebuild and redeployment. Returns immediately with status \"building\" and the artifact_id — use get_artifact_status to poll until status is \"running\".",
		MCP:         &MCPMetadata{ToolName: "update_artifact"},
	},
	{
		ID:          "artifacts.list",
		Method:      "GET",
		Path:        "/api/artifacts",
		Summary:     "List all deployed artifacts with their status, deployment target, and access URLs.",
		Description: "List all deployed artifacts with their status, deployment target, and access URLs.",
		MCP:         &MCPMetadata{ToolName: "list_artifacts"},
	},
	{
		ID:          "artifacts.status",
		Method:      "GET",
		Path:        "/api/artifacts/{id}",
		Summary:     "Get detailed status information for a specific deployed artifact, including URL, deployment target, image reference, and any errors.",
		Description: "Get detailed status information for a specific deployed artifact, including URL, deployment target, image reference, and any errors.",
		MCP:         &MCPMetadata{ToolName: "get_artifact_status"},
	},
	{
		ID:          "artifacts.delete",
		Method:      "DELETE",
		Path:        "/api/artifacts/{id}",
		Summary:     "Stop and remove a deployed artifact.",
		Description: "Stop and remove a deployed artifact. This deletes the deployment, stored source code, and all associated resources.",
		MCP:         &MCPMetadata{ToolName: "delete_artifact"},
	},
	{
		ID:          "artifacts.logs",
		Method:      "GET",
		Path:        "/api/artifacts/{id}/logs",
		Summary:     "Retrieve recent log lines from a deployed artifact for debugging purposes.",
		Description: "Retrieve recent log lines from a deployed artifact for debugging purposes.",
		MCP:         &MCPMetadata{ToolName: "get_artifact_logs"},
	},
	{
		ID:          "artifacts.targets",
		Method:      "GET",
		Path:        "/api/targets",
		Summary:     "Show which deployment backends (Knative, Kubernetes) are available in the current cluster.",
		Description: "Show which deployment backends (Knative, Kubernetes) are available in the current cluster.",
		MCP:         &MCPMetadata{ToolName: "list_deployment_targets"},
	},
	{
		ID:          "artifacts.versions.list",
		Method:      "GET",
		Path:        "/api/artifacts/{id}/versions",
		Summary:     "List all version snapshots for a deployed artifact, ordered by version number.",
		Description: "List all version snapshots for a deployed artifact, ordered by version number. Each version includes the image reference, status, URL, and who created it.",
		MCP:         &MCPMetadata{ToolName: "list_versions"},
	},
	{
		ID:          "artifacts.rollback",
		Method:      "POST",
		Path:        "/api/artifacts/{id}/rollback",
		Summary:     "Roll back a deployed artifact to a previous version.",
		Description: "Roll back a deployed artifact to a previous version. This redeploys the artifact using the image and configuration from the specified version snapshot. A new version entry is created for the rollback (history is not rewritten).",
		MCP:         &MCPMetadata{ToolName: "rollback_artifact"},
	},
	{
		ID:          "artifacts.share",
		Method:      "POST",
		Path:        "/api/artifacts/{id}/share",
		Summary:     "Share a deployed artifact with other users, granting them read-only access (view status, logs, URL).",
		Description: "Share a deployed artifact with other users, granting them read-only access (view status, logs, URL). Only the artifact owner or an admin can share.",
		MCP:         &MCPMetadata{ToolName: "share_artifact"},
	},
	{
		ID:          "artifacts.unshare",
		Method:      "POST",
		Path:        "/api/artifacts/{id}/unshare",
		Summary:     "Revoke read-only access to a deployed artifact from specific users.",
		Description: "Revoke read-only access to a deployed artifact from specific users. Only the artifact owner or an admin can unshare.",
		MCP:         &MCPMetadata{ToolName: "unshare_artifact"},
	},
	{
		ID:          "artifacts.share-links.create",
		Method:      "POST",
		Path:        "/api/artifacts/{id}/share-link",
		Summary:     "Create a public shareable link for an artifact.",
		Description: "Create a public shareable link for an artifact. Anyone with the link (and optional password) can view the artifact's status and URL without a vibeD account.",
		MCP:         &MCPMetadata{ToolName: "create_share_link"},
	},
	{
		ID:          "artifacts.share-links.list",
		Method:      "GET",
		Path:        "/api/artifacts/{id}/share-links",
		Summary:     "List all share links for an artifact.",
		Description: "List all share links for an artifact. Only the artifact owner or admin can see these.",
		MCP:         &MCPMetadata{ToolName: "list_share_links"},
	},
	{
		ID:          "artifacts.share-links.revoke",
		Method:      "DELETE",
		Path:        "/api/share-links/{token}",
		Summary:     "Revoke a share link so it can no longer be used.",
		Description: "Revoke a share link so it can no longer be used. The link will return 404 after revocation.",
		MCP:         &MCPMetadata{ToolName: "revoke_share_link"},
	},
}

// ArtifactOperations returns the canonical artifact API operations.
func ArtifactOperations() []Operation {
	out := make([]Operation, len(artifactOperations))
	copy(out, artifactOperations)
	return out
}

// MustArtifactOperation returns the artifact operation with the given ID.
func MustArtifactOperation(id string) Operation {
	for _, op := range artifactOperations {
		if op.ID == id {
			return op
		}
	}
	panic(fmt.Sprintf("unknown artifact operation: %s", id))
}

func ValidateFileLimits(files map[string]string, limits config.LimitsConfig) error {
	if limits.MaxFileCount > 0 && len(files) > limits.MaxFileCount {
		return &api.ErrInvalidInput{
			Field:   "files",
			Message: fmt.Sprintf("too many files: %d exceeds maximum of %d", len(files), limits.MaxFileCount),
		}
	}

	total := 0
	for _, content := range files {
		total += len(content)
		if limits.MaxTotalFileSize > 0 && total > limits.MaxTotalFileSize {
			return &api.ErrInvalidInput{
				Field:   "files",
				Message: fmt.Sprintf("total file size exceeds maximum of %d bytes (%d MB)", limits.MaxTotalFileSize, limits.MaxTotalFileSize/(1024*1024)),
			}
		}
	}

	return nil
}

func ClampLogLines(requested int, limits config.LimitsConfig) int {
	if requested <= 0 {
		return 50
	}
	if limits.MaxLogLines > 0 && requested > limits.MaxLogLines {
		return limits.MaxLogLines
	}
	return requested
}

func DeployArtifact(ctx context.Context, orch *orchestrator.Orchestrator, limits config.LimitsConfig, req DeployArtifactRequest) (*orchestrator.DeployResult, error) {
	if err := ValidateFileLimits(req.Files, limits); err != nil {
		return nil, err
	}
	return orch.AsyncDeploy(ctx, orchestrator.DeployRequest{
		Name:       req.Name,
		Files:      req.Files,
		Language:   req.Language,
		Target:     req.Target,
		EnvVars:    req.EnvVars,
		SecretRefs: req.SecretRefs,
		Port:       req.Port,
	})
}

func UpdateArtifact(ctx context.Context, orch *orchestrator.Orchestrator, limits config.LimitsConfig, req UpdateArtifactRequest) (*orchestrator.DeployResult, error) {
	if err := ValidateFileLimits(req.Files, limits); err != nil {
		return nil, err
	}
	return orch.AsyncUpdate(ctx, orchestrator.UpdateRequest{
		ArtifactID: req.ArtifactID,
		Files:      req.Files,
		EnvVars:    req.EnvVars,
		SecretRefs: req.SecretRefs,
	})
}

func ListArtifacts(ctx context.Context, orch *orchestrator.Orchestrator, req ListArtifactsRequest) (*ListArtifactsResponse, error) {
	result, err := orch.List(ctx, req.Status, req.Offset, req.Limit)
	if err != nil {
		return nil, err
	}
	return &ListArtifactsResponse{
		Artifacts: result.Artifacts,
		Total:     result.Total,
		Offset:    req.Offset,
		Limit:     clampArtifactLimit(req.Limit),
	}, nil
}

func GetArtifactStatus(ctx context.Context, orch *orchestrator.Orchestrator, req GetArtifactStatusRequest) (*api.Artifact, error) {
	artifact, err := orch.Status(ctx, req.ArtifactID)
	if err != nil {
		return nil, err
	}
	artifact.EnvVars = nil
	artifact.StorageRef = ""
	return artifact, nil
}

func DeleteArtifact(ctx context.Context, orch *orchestrator.Orchestrator, req DeleteArtifactRequest) (*DeleteArtifactResponse, error) {
	if err := orch.Delete(ctx, req.ArtifactID); err != nil {
		return nil, err
	}
	return &DeleteArtifactResponse{
		Status: "deleted",
		ID:     req.ArtifactID,
	}, nil
}

func GetArtifactLogs(ctx context.Context, orch *orchestrator.Orchestrator, limits config.LimitsConfig, req GetArtifactLogsRequest) (*GetArtifactLogsResponse, error) {
	logs, err := orch.Logs(ctx, req.ArtifactID, ClampLogLines(req.Lines, limits))
	if err != nil {
		return nil, err
	}
	return &GetArtifactLogsResponse{
		ArtifactID: req.ArtifactID,
		Logs:       logs,
	}, nil
}

func ListTargets(_ context.Context, orch *orchestrator.Orchestrator) (ListTargetsResponse, error) {
	return ListTargetsResponse(orch.ListTargets()), nil
}

func ListVersions(ctx context.Context, orch *orchestrator.Orchestrator, req ListVersionsRequest) (*ListVersionsResponse, error) {
	versions, err := orch.ListVersions(ctx, req.ArtifactID)
	if err != nil {
		return nil, err
	}
	return &ListVersionsResponse{
		ArtifactID: req.ArtifactID,
		Versions:   versions,
	}, nil
}

func RollbackArtifact(ctx context.Context, orch *orchestrator.Orchestrator, req RollbackArtifactRequest) (*orchestrator.DeployResult, error) {
	return orch.Rollback(ctx, req.ArtifactID, req.Version)
}

func ShareArtifact(ctx context.Context, orch *orchestrator.Orchestrator, req ShareArtifactRequest) (*ShareArtifactResponse, error) {
	if err := orch.ShareArtifact(ctx, req.ArtifactID, req.UserIDs); err != nil {
		return nil, err
	}
	return &ShareArtifactResponse{
		ArtifactID: req.ArtifactID,
		SharedWith: req.UserIDs,
		Status:     "shared",
	}, nil
}

func UnshareArtifact(ctx context.Context, orch *orchestrator.Orchestrator, req UnshareArtifactRequest) (*UnshareArtifactResponse, error) {
	if err := orch.UnshareArtifact(ctx, req.ArtifactID, req.UserIDs); err != nil {
		return nil, err
	}
	return &UnshareArtifactResponse{
		ArtifactID: req.ArtifactID,
		Removed:    req.UserIDs,
		Status:     "unshared",
	}, nil
}

func CreateShareLink(ctx context.Context, orch *orchestrator.Orchestrator, req CreateShareLinkRequest) (*api.ShareLink, error) {
	var expiresIn time.Duration
	if req.ExpiresIn != "" {
		s := req.ExpiresIn
		if strings.HasSuffix(s, "d") {
			if d, err := time.ParseDuration(strings.TrimSuffix(s, "d") + "h"); err == nil {
				expiresIn = d * 24
			}
		} else {
			expiresIn, _ = time.ParseDuration(s)
		}
	}
	return orch.CreateShareLink(ctx, req.ArtifactID, req.Password, expiresIn)
}

func ListShareLinks(ctx context.Context, orch *orchestrator.Orchestrator, req ListShareLinksRequest) ([]api.ShareLink, error) {
	links, err := orch.ListShareLinks(ctx, req.ArtifactID)
	if err != nil {
		return nil, err
	}
	if links == nil {
		links = []api.ShareLink{}
	}
	return links, nil
}

func RevokeShareLink(ctx context.Context, orch *orchestrator.Orchestrator, req RevokeShareLinkRequest) (*RevokeShareLinkResponse, error) {
	if err := orch.RevokeShareLink(ctx, req.Token); err != nil {
		return nil, err
	}
	return &RevokeShareLinkResponse{Status: "revoked"}, nil
}

func clampArtifactLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	if limit > 200 {
		return 200
	}
	return limit
}
