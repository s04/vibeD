package operations

import "testing"

func TestArtifactOperations_MCPMetadataIsConsistent(t *testing.T) {
	ops := ArtifactOperations()
	if len(ops) == 0 {
		t.Fatal("expected artifact operations")
	}

	seenIDs := map[string]struct{}{}
	seenTools := map[string]struct{}{}

	for _, op := range ops {
		if op.ID == "" {
			t.Fatal("operation ID must not be empty")
		}
		if op.Method == "" {
			t.Fatalf("operation %s is missing method", op.ID)
		}
		if op.Path == "" {
			t.Fatalf("operation %s is missing path", op.ID)
		}
		if _, ok := seenIDs[op.ID]; ok {
			t.Fatalf("duplicate operation ID: %s", op.ID)
		}
		seenIDs[op.ID] = struct{}{}

		if op.Summary == "" {
			t.Fatalf("operation %s is missing summary", op.ID)
		}
		if op.Description == "" {
			t.Fatalf("operation %s is missing description", op.ID)
		}

		if op.MCP == nil || op.MCP.ToolName == "" {
			t.Fatalf("operation %s is missing MCP tool metadata", op.ID)
		}
		if _, ok := seenTools[op.MCP.ToolName]; ok {
			t.Fatalf("duplicate MCP tool name: %s", op.MCP.ToolName)
		}
		seenTools[op.MCP.ToolName] = struct{}{}
	}
}

func TestMustArtifactOperation(t *testing.T) {
	op := MustArtifactOperation("artifacts.deploy")
	if op.MCP == nil || op.MCP.ToolName != "deploy_artifact" {
		t.Fatal("expected deploy MCP metadata")
	}
	if op.Method != "POST" || op.Path != "/api/artifacts" {
		t.Fatal("expected deploy API metadata")
	}
}

func TestMustArtifactOperation_DeployFromRepo(t *testing.T) {
	op := MustArtifactOperation("artifacts.deploy-from-repo")
	if op.MCP == nil || op.MCP.ToolName != "deploy_from_repo" {
		t.Fatal("expected deploy_from_repo MCP metadata")
	}
	if op.Method != "POST" || op.Path != "/api/artifacts/from-repo" {
		t.Fatal("expected deploy-from-repo API metadata")
	}
}
