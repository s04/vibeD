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

		if op.ExposesOn(SurfaceMCP) {
			if op.MCP == nil || op.MCP.ToolName == "" {
				t.Fatalf("operation %s exposes MCP but has no tool metadata", op.ID)
			}
			if _, ok := seenTools[op.MCP.ToolName]; ok {
				t.Fatalf("duplicate MCP tool name: %s", op.MCP.ToolName)
			}
			seenTools[op.MCP.ToolName] = struct{}{}
		}
	}
}

func TestOperation_ExposesOn(t *testing.T) {
	op := Operation{
		ID:       "example",
		Surfaces: []Surface{SurfaceREST, SurfaceMCP},
	}

	if !op.ExposesOn(SurfaceREST) {
		t.Fatal("expected REST exposure")
	}
	if !op.ExposesOn(SurfaceMCP) {
		t.Fatal("expected MCP exposure")
	}
	if op.ExposesOn(Surface("other")) {
		t.Fatal("did not expect other surface exposure")
	}
}

func TestMustArtifactOperation(t *testing.T) {
	op := MustArtifactOperation("artifacts.deploy")
	if op.MCP == nil || op.MCP.ToolName != "deploy_artifact" {
		t.Fatal("expected deploy MCP metadata")
	}
}
