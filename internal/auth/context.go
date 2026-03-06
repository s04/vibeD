package auth

import (
	"context"

	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
)

type contextKey string

const roleKey contextKey = "vibed-role"

// UserIDFromContext extracts the authenticated user's ID from the request context.
// The MCP SDK stores TokenInfo in context via auth.TokenInfoFromContext().
// Returns "" when auth is disabled or no user is set.
func UserIDFromContext(ctx context.Context) string {
	info := mcpauth.TokenInfoFromContext(ctx)
	if info == nil {
		return ""
	}
	return info.UserID
}

// WithRole adds the user's role to the context.
func WithRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, roleKey, role)
}

// RoleFromContext extracts the user's role from the context.
// Returns "user" as the default when no role is set.
func RoleFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(roleKey).(string); ok && v != "" {
		return v
	}
	return "user"
}

// IsAdmin returns true if the current user has the admin role.
func IsAdmin(ctx context.Context) bool {
	return RoleFromContext(ctx) == "admin"
}
