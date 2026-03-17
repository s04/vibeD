// Package auth provides authentication for vibeD's HTTP endpoints.
//
// It supports three modes:
//   - API key authentication: simple bearer tokens validated against a configured list
//   - External OAuth: tokens verified by an external OAuth gateway/proxy
//   - OIDC: direct JWT validation against an OIDC provider (Keycloak, Azure Entra, etc.)
//
// The implementation uses the MCP SDK's auth.RequireBearerToken middleware,
// which automatically binds sessions to users and prevents session hijacking.
package auth

import (
	"context"
	"crypto/subtle"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"

	"github.com/vibed-project/vibeD/internal/config"
	"github.com/vibed-project/vibeD/internal/store"
	"github.com/vibed-project/vibeD/pkg/api"
)

// Middleware creates the MCP-compatible auth middleware from the config.
// It wraps the SDK's auth.RequireBearerToken with a custom TokenVerifier
// that validates against configured API keys, external OAuth, or OIDC JWTs.
// The userStore parameter is optional (may be nil when auth is disabled or mode is not OIDC).
func Middleware(cfg config.AuthConfig, userStore store.UserStore, logger *slog.Logger) (func(http.Handler) http.Handler, error) {
	if !cfg.Enabled {
		return func(next http.Handler) http.Handler {
			return next
		}, nil
	}

	var verifier mcpauth.TokenVerifier

	switch cfg.Mode {
	case "apikey", "":
		if len(cfg.APIKeys) == 0 {
			return nil, fmt.Errorf("auth.mode is 'apikey' but no API keys are configured")
		}
		verifier = apiKeyVerifier(cfg.APIKeys, logger)

	case "oauth":
		verifier = oauthPassthroughVerifier(logger)

	case "oidc":
		v, err := newOIDCVerifier(cfg.OIDC, userStore, logger)
		if err != nil {
			return nil, fmt.Errorf("initializing OIDC verifier: %w", err)
		}
		verifier = v

	default:
		return nil, fmt.Errorf("unknown auth.mode: %q (must be 'apikey', 'oauth', or 'oidc')", cfg.Mode)
	}

	opts := &mcpauth.RequireBearerTokenOptions{}
	middleware := mcpauth.RequireBearerToken(verifier, opts)
	logger.Info("authentication enabled", "mode", cfg.Mode)

	return middleware, nil
}

// SkipAuthPaths wraps an auth middleware to skip authentication for certain paths
// (health checks, metrics, static frontend assets).
func SkipAuthPaths(authMiddleware func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		authed := authMiddleware(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			// Skip auth for health, metrics, API docs, well-known endpoints, and frontend static assets
			if path == "/healthz" || path == "/readyz" || path == "/metrics" ||
				strings.HasPrefix(path, "/api/docs") ||
				strings.HasPrefix(path, "/api/share/") ||
				strings.HasPrefix(path, "/.well-known/") {
				next.ServeHTTP(w, r)
				return
			}
			// Protect MCP and API endpoints
			if strings.HasPrefix(path, "/mcp") || strings.HasPrefix(path, "/api/") {
				authed.ServeHTTP(w, r)
				return
			}
			// Frontend static files: no auth needed
			next.ServeHTTP(w, r)
		})
	}
}

// apiKeyVerifier returns a TokenVerifier that validates tokens against configured API keys.
func apiKeyVerifier(keys []config.APIKeyConf, logger *slog.Logger) mcpauth.TokenVerifier {
	return func(ctx context.Context, token string, req *http.Request) (*mcpauth.TokenInfo, error) {
		for _, key := range keys {
			resolvedKey := resolveKeyValue(key.Key)
			if subtle.ConstantTimeCompare([]byte(token), []byte(resolvedKey)) == 1 {
				logger.Debug("API key authenticated",
					"name", key.Name,
					"path", req.URL.Path,
				)
				return &mcpauth.TokenInfo{
					Scopes:     key.Scopes,
					Expiration: time.Now().Add(24 * time.Hour), // API keys don't expire per-request
					UserID:     key.Name,
				}, nil
			}
		}

		logger.Warn("authentication failed: invalid API key",
			"path", req.URL.Path,
			"remote", req.RemoteAddr,
		)
		return nil, mcpauth.ErrInvalidToken
	}
}

// oauthPassthroughVerifier accepts any Bearer token and passes it through.
// This is intended for setups where an external gateway/proxy validates OAuth tokens
// and vibeD trusts the proxy's authentication.
func oauthPassthroughVerifier(logger *slog.Logger) mcpauth.TokenVerifier {
	return func(ctx context.Context, token string, req *http.Request) (*mcpauth.TokenInfo, error) {
		if token == "" {
			return nil, mcpauth.ErrInvalidToken
		}

		// Extract user ID from X-Forwarded-User header if set by the proxy
		userID := req.Header.Get("X-Forwarded-User")
		if userID == "" {
			userID = "oauth-user"
		}

		logger.Debug("OAuth passthrough authenticated",
			"user", userID,
			"path", req.URL.Path,
		)

		return &mcpauth.TokenInfo{
			UserID:     userID,
			Expiration: time.Now().Add(1 * time.Hour),
		}, nil
	}
}

// BuildRoleMap creates a mapping from user ID (APIKey Name) to role.
// Users without an explicit role default to "user".
func BuildRoleMap(keys []config.APIKeyConf) map[string]string {
	m := make(map[string]string, len(keys))
	for _, k := range keys {
		role := k.Role
		if role == "" {
			role = "user"
		}
		m[k.Name] = role
	}
	return m
}

// RoleMiddleware creates middleware that injects the authenticated user's role into the context.
// It checks the roleMap first (for API key users), then falls back to the user store (for OIDC users).
func RoleMiddleware(roleMap map[string]string, userStore store.UserStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := UserIDFromContext(r.Context())
			if userID != "" {
				role := roleMap[userID]
				if role == "" && userStore != nil {
					if u, err := userStore.GetUser(r.Context(), userID); err == nil {
						role = u.Role
					}
				}
				if role == "" {
					role = "user"
				}
				ctx := WithRole(r.Context(), role)
				r = r.WithContext(ctx)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// resolveKeyValue resolves an API key value using the shared config.ResolveSecret helper.
// Supports "env:VAR_NAME" and "file:/path" patterns, or literal values.
func resolveKeyValue(key string) string {
	resolved, err := config.ResolveSecret(key)
	if err != nil || resolved == "" {
		return key // Fall back to literal if resolution fails
	}
	return resolved
}

// newOIDCVerifier creates a TokenVerifier that validates JWTs against an OIDC provider.
// It auto-provisions user records in the store on first login.
func newOIDCVerifier(cfg config.OIDCConfig, userStore store.UserStore, logger *slog.Logger) (mcpauth.TokenVerifier, error) {
	provider, err := oidc.NewProvider(context.Background(), cfg.Issuer)
	if err != nil {
		return nil, fmt.Errorf("discovering OIDC provider %q: %w", cfg.Issuer, err)
	}

	verifierConfig := &oidc.Config{
		ClientID: cfg.Audience,
	}
	if cfg.Audience == "" {
		verifierConfig.SkipClientIDCheck = true
	}
	idTokenVerifier := provider.Verifier(verifierConfig)

	usernameClaim := cfg.UsernameClaim
	if usernameClaim == "" {
		usernameClaim = "preferred_username"
	}
	emailClaim := cfg.EmailClaim
	if emailClaim == "" {
		emailClaim = "email"
	}
	roleClaim := cfg.RoleClaim
	if roleClaim == "" {
		roleClaim = "realm_access.roles"
	}
	adminRole := cfg.AdminRole
	if adminRole == "" {
		adminRole = "vibed-admin"
	}

	logger.Info("OIDC authentication configured",
		"issuer", cfg.Issuer,
		"audience", cfg.Audience,
		"usernameClaim", usernameClaim,
		"adminRole", adminRole,
	)

	return func(ctx context.Context, token string, req *http.Request) (*mcpauth.TokenInfo, error) {
		idToken, err := idTokenVerifier.Verify(ctx, token)
		if err != nil {
			logger.Debug("OIDC token verification failed", "error", err, "path", req.URL.Path)
			return nil, mcpauth.ErrInvalidToken
		}

		// Extract claims
		var claims map[string]interface{}
		if err := idToken.Claims(&claims); err != nil {
			logger.Warn("OIDC claims extraction failed", "error", err)
			return nil, mcpauth.ErrInvalidToken
		}

		sub := idToken.Subject
		username := extractStringClaim(claims, usernameClaim)
		if username == "" {
			username = sub
		}
		email := extractStringClaim(claims, emailClaim)

		// Determine role from claims
		role := "user"
		roles := extractRoleClaims(claims, roleClaim)
		for _, r := range roles {
			if r == adminRole {
				role = "admin"
				break
			}
		}

		// Auto-provision user if store is available
		if userStore != nil {
			if _, err := userStore.GetUser(ctx, sub); err != nil {
				// User doesn't exist — create
				now := time.Now()
				newUser := &api.User{
					ID:        sub,
					Name:      username,
					Email:     email,
					Role:      role,
					Status:    "active",
					Provider:  "oidc",
					CreatedAt: now,
					UpdatedAt: now,
				}
				if createErr := userStore.CreateUser(ctx, newUser); createErr != nil {
					// May fail on duplicate name — try updating instead
					logger.Debug("auto-provision user failed (may already exist)", "sub", sub, "error", createErr)
				} else {
					logger.Info("auto-provisioned OIDC user", "sub", sub, "name", username, "role", role)
				}
			}
		}

		logger.Debug("OIDC authenticated", "sub", sub, "name", username, "role", role, "path", req.URL.Path)

		return &mcpauth.TokenInfo{
			UserID:     sub,
			Expiration: idToken.Expiry,
		}, nil
	}, nil
}

// extractStringClaim extracts a string value from a claims map.
func extractStringClaim(claims map[string]interface{}, key string) string {
	if v, ok := claims[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// extractRoleClaims extracts role strings from nested claims.
// Supports dot-separated paths like "realm_access.roles".
func extractRoleClaims(claims map[string]interface{}, path string) []string {
	parts := strings.Split(path, ".")
	var current interface{} = claims

	for _, part := range parts[:len(parts)-1] {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[part]
		} else {
			return nil
		}
	}

	lastKey := parts[len(parts)-1]
	if m, ok := current.(map[string]interface{}); ok {
		if arr, ok := m[lastKey].([]interface{}); ok {
			var roles []string
			for _, v := range arr {
				if s, ok := v.(string); ok {
					roles = append(roles, s)
				}
			}
			return roles
		}
	}
	return nil
}
