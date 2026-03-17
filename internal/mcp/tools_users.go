package mcp

import (
	"context"
	"fmt"

	vibedauth "github.com/vibed-project/vibeD/internal/auth"
	"github.com/vibed-project/vibeD/internal/store"
	"github.com/vibed-project/vibeD/pkg/api"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type listUsersInput struct{}

type listUsersOutput struct {
	Users []api.User `json:"users"`
}

func registerListUsersTool(server *mcp.Server, userStore store.UserStore) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_users",
		Description: "List all vibeD users. Requires admin role.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input listUsersInput) (*mcp.CallToolResult, *listUsersOutput, error) {
		if !vibedauth.IsAdmin(ctx) {
			return nil, nil, fmt.Errorf("admin access required")
		}
		users, err := userStore.ListUsers(ctx)
		if err != nil {
			return nil, nil, err
		}
		return nil, &listUsersOutput{Users: users}, nil
	})
}

type getUserInput struct {
	UserID string `json:"user_id" jsonschema:"ID of the user to retrieve"`
}

func registerGetUserTool(server *mcp.Server, userStore store.UserStore) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_user",
		Description: "Get details of a specific vibeD user. Admins can view any user; regular users can only view themselves.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input getUserInput) (*mcp.CallToolResult, *api.User, error) {
		callerID := vibedauth.UserIDFromContext(ctx)
		if !vibedauth.IsAdmin(ctx) && callerID != input.UserID {
			return nil, nil, fmt.Errorf("user not found")
		}
		user, err := userStore.GetUser(ctx, input.UserID)
		if err != nil {
			return nil, nil, err
		}
		return nil, user, nil
	})
}
