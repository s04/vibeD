package operations

import "fmt"

// Surface indicates which external interfaces an operation should be exposed on.
type Surface string

const (
	SurfaceREST Surface = "rest"
	SurfaceMCP  Surface = "mcp"
)

// MCPMetadata captures the MCP-specific projection of a shared operation.
type MCPMetadata struct {
	ToolName string
}

// Operation describes one externally exposed capability of vibeD.
// It is intentionally metadata-only for now; runtime adapters can attach handlers later.
type Operation struct {
	ID          string
	Surfaces    []Surface
	Summary     string
	Description string
	MCP         *MCPMetadata
}

func (o Operation) ExposesOn(surface Surface) bool {
	for _, s := range o.Surfaces {
		if s == surface {
			return true
		}
	}
	return false
}

var artifactOperations = []Operation{
	{
		ID:       "artifacts.deploy",
		Surfaces: []Surface{SurfaceREST, SurfaceMCP},
		Summary:  "Deploy a web artifact (website, web app) to the cluster.",
		Description: "Deploy a web artifact (website, web app) to the cluster. Provide source files and vibeD handles " +
			"building a container image and deploying it. Returns immediately with status \"building\" and an artifact_id " +
			"— use get_artifact_status to poll until status is \"running\". Knative is used when available (auto-scaling, " +
			"clean URLs), otherwise falls back to plain Kubernetes. Set target explicitly to override. For Go apps, go.mod " +
			"is auto-generated if not provided.",
		MCP: &MCPMetadata{ToolName: "deploy_artifact"},
	},
	{
		ID:       "artifacts.update",
		Surfaces: []Surface{SurfaceREST, SurfaceMCP},
		Summary:  "Update an existing deployed artifact with new source files.",
		Description: "Update an existing deployed artifact with new source files. Triggers a rebuild and redeployment. " +
			"Returns immediately with status \"building\" and the artifact_id — use get_artifact_status to poll until " +
			"status is \"running\".",
		MCP: &MCPMetadata{ToolName: "update_artifact"},
	},
	{
		ID:          "artifacts.list",
		Surfaces:    []Surface{SurfaceREST, SurfaceMCP},
		Summary:     "List all deployed artifacts with their status, deployment target, and access URLs.",
		Description: "List all deployed artifacts with their status, deployment target, and access URLs.",
		MCP:         &MCPMetadata{ToolName: "list_artifacts"},
	},
	{
		ID:          "artifacts.status",
		Surfaces:    []Surface{SurfaceREST, SurfaceMCP},
		Summary:     "Get detailed status information for a specific deployed artifact, including URL, deployment target, image reference, and any errors.",
		Description: "Get detailed status information for a specific deployed artifact, including URL, deployment target, image reference, and any errors.",
		MCP:         &MCPMetadata{ToolName: "get_artifact_status"},
	},
	{
		ID:          "artifacts.delete",
		Surfaces:    []Surface{SurfaceREST, SurfaceMCP},
		Summary:     "Stop and remove a deployed artifact.",
		Description: "Stop and remove a deployed artifact. This deletes the deployment, stored source code, and all associated resources.",
		MCP:         &MCPMetadata{ToolName: "delete_artifact"},
	},
	{
		ID:          "artifacts.logs",
		Surfaces:    []Surface{SurfaceREST, SurfaceMCP},
		Summary:     "Retrieve recent log lines from a deployed artifact for debugging purposes.",
		Description: "Retrieve recent log lines from a deployed artifact for debugging purposes.",
		MCP:         &MCPMetadata{ToolName: "get_artifact_logs"},
	},
	{
		ID:          "artifacts.targets",
		Surfaces:    []Surface{SurfaceREST, SurfaceMCP},
		Summary:     "Show which deployment backends (Knative, Kubernetes) are available in the current cluster.",
		Description: "Show which deployment backends (Knative, Kubernetes) are available in the current cluster.",
		MCP:         &MCPMetadata{ToolName: "list_deployment_targets"},
	},
	{
		ID:          "artifacts.versions.list",
		Surfaces:    []Surface{SurfaceREST, SurfaceMCP},
		Summary:     "List all version snapshots for a deployed artifact, ordered by version number.",
		Description: "List all version snapshots for a deployed artifact, ordered by version number. Each version includes the image reference, status, URL, and who created it.",
		MCP:         &MCPMetadata{ToolName: "list_versions"},
	},
	{
		ID:          "artifacts.rollback",
		Surfaces:    []Surface{SurfaceREST, SurfaceMCP},
		Summary:     "Roll back a deployed artifact to a previous version.",
		Description: "Roll back a deployed artifact to a previous version. This redeploys the artifact using the image and configuration from the specified version snapshot. A new version entry is created for the rollback (history is not rewritten).",
		MCP:         &MCPMetadata{ToolName: "rollback_artifact"},
	},
	{
		ID:          "artifacts.share",
		Surfaces:    []Surface{SurfaceREST, SurfaceMCP},
		Summary:     "Share a deployed artifact with other users, granting them read-only access (view status, logs, URL).",
		Description: "Share a deployed artifact with other users, granting them read-only access (view status, logs, URL). Only the artifact owner or an admin can share.",
		MCP:         &MCPMetadata{ToolName: "share_artifact"},
	},
	{
		ID:          "artifacts.unshare",
		Surfaces:    []Surface{SurfaceREST, SurfaceMCP},
		Summary:     "Revoke read-only access to a deployed artifact from specific users.",
		Description: "Revoke read-only access to a deployed artifact from specific users. Only the artifact owner or an admin can unshare.",
		MCP:         &MCPMetadata{ToolName: "unshare_artifact"},
	},
	{
		ID:          "artifacts.share-links.create",
		Surfaces:    []Surface{SurfaceREST, SurfaceMCP},
		Summary:     "Create a public shareable link for an artifact.",
		Description: "Create a public shareable link for an artifact. Anyone with the link (and optional password) can view the artifact's status and URL without a vibeD account.",
		MCP:         &MCPMetadata{ToolName: "create_share_link"},
	},
	{
		ID:          "artifacts.share-links.list",
		Surfaces:    []Surface{SurfaceREST, SurfaceMCP},
		Summary:     "List all share links for an artifact.",
		Description: "List all share links for an artifact. Only the artifact owner or admin can see these.",
		MCP:         &MCPMetadata{ToolName: "list_share_links"},
	},
	{
		ID:          "artifacts.share-links.revoke",
		Surfaces:    []Surface{SurfaceREST, SurfaceMCP},
		Summary:     "Revoke a share link so it can no longer be used.",
		Description: "Revoke a share link so it can no longer be used. The link will return 404 after revocation.",
		MCP:         &MCPMetadata{ToolName: "revoke_share_link"},
	},
}

// ArtifactOperations returns the canonical shared artifact operation definitions.
func ArtifactOperations() []Operation {
	out := make([]Operation, len(artifactOperations))
	copy(out, artifactOperations)
	return out
}

// MustArtifactOperation returns the artifact operation with the given ID.
// It panics if the ID is unknown because IDs are static code-level references.
func MustArtifactOperation(id string) Operation {
	for _, op := range artifactOperations {
		if op.ID == id {
			return op
		}
	}
	panic(fmt.Sprintf("unknown artifact operation: %s", id))
}
