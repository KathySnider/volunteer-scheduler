package middleware

import (
	"net/http"
	"strings"
	"volunteer-scheduler/models"
	"volunteer-scheduler/services"
)

// RequireAuth returns an HTTP middleware that validates the session token.
// If the token is missing or invalid, it returns a 401 Unauthorized response.
// On success, stores the volunteer ID, role, ResponseWriter, and Request in
// the request context (the latter two so resolvers can set/clear cookies).
func RequireAuth(magicLinkService *services.MagicLinkService, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// 1. Try the HttpOnly session cookie first.
		var token string
		if cookie, err := r.Cookie("session"); err == nil {
			token = cookie.Value
		}

		// 2. Fall back to the Authorization: Bearer header (keeps API clients working).
		if token == "" {
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
					token = parts[1]
				}
			}
		}

		if token == "" {
			http.Error(w, `{"errors":[{"message":"unauthorized"}]}`, http.StatusUnauthorized)
			return
		}

		// Validate the session token — returns volunteer ID and roles.
		volId, roles, err := magicLinkService.ValidateSessionToken(r.Context(), token)
		if err != nil {
			http.Error(w, `{"errors":[{"message":"invalid or expired session"}]}`, http.StatusUnauthorized)
			return
		}

		// Store volunteer ID, roles, ResponseWriter, and Request in the context
		// so resolvers can read the session cookie and set/clear it on login/logout.
		ctx := ContextWithVolunteerId(r.Context(), volId)
		ctx = ContextWithVolunteerRoles(ctx, roles)
		ctx = ContextWithResponseWriter(ctx, w)
		ctx = ContextWithRequest(ctx, r)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAdmin wraps RequireAuth and additionally enforces that the caller
// has the ADMINISTRATOR role. Returns 403 Forbidden if they do not.
func RequireAdmin(magicLinkService *services.MagicLinkService, next http.Handler) http.Handler {
	return RequireAuth(magicLinkService, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		roles, ok := VolunteerRolesFromContext(r.Context())
		if !ok || !hasRole(roles, string(models.RoleAdministrator)) {
			http.Error(w, `{"errors":[{"message":"forbidden"}]}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}))
}

// hasRole returns true when role is present in the roles slice.
func hasRole(roles []string, role string) bool {
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}
