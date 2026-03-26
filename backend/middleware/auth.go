package middleware

import (
	"net/http"
	"strings"
	"volunteer-scheduler/services"
)

// RequireAuth returns an HTTP middleware that validates the session token.
// If the token is missing or invalid, it returns a 401 Unauthorized response.
// On success, stores both the volunteer ID and role in the request context.
func RequireAuth(magicLinkService *services.MagicLinkService, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Extract the token from the Authorization header.
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"errors":[{"message":"unauthorized"}]}`, http.StatusUnauthorized)
			return
		}

		// Expect "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			http.Error(w, `{"errors":[{"message":"invalid authorization header"}]}`, http.StatusUnauthorized)
			return
		}

		token := parts[1]

		// Validate the session token — returns volunteer ID and role.
		volId, role, err := magicLinkService.ValidateSessionToken(r.Context(), token)
		if err != nil {
			http.Error(w, `{"errors":[{"message":"invalid or expired session"}]}`, http.StatusUnauthorized)
			return
		}

		// Store both the volunteer ID and role in the request context.
		ctx := ContextWithVolunteerId(r.Context(), volId)
		ctx = ContextWithVolunteerRole(ctx, role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAdmin wraps RequireAuth and additionally enforces that the caller
// has the ADMINISTRATOR role. Returns 403 Forbidden if they do not.
func RequireAdmin(magicLinkService *services.MagicLinkService, next http.Handler) http.Handler {
	return RequireAuth(magicLinkService, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, ok := VolunteerRoleFromContext(r.Context())
		if !ok || role != "ADMINISTRATOR" {
			http.Error(w, `{"errors":[{"message":"forbidden"}]}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}))
}
