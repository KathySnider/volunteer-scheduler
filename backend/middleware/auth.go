package middleware

import (
	"net/http"
	"strings"
	"volunteer-scheduler/services"
)

// RequireAuth returns an HTTP middleware that validates the session token.
// If the token is missing or invalid, it returns a 401 Unauthorized response.
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

		// Validate the session token.
		volId, err := magicLinkService.ValidateSessionToken(r.Context(), token)
		if err != nil {
			http.Error(w, `{"errors":[{"message":"invalid or expired session"}]}`, http.StatusUnauthorized)
			return
		}

		// Store the volunteer id in the request context for use by resolvers.
		ctx := ContextWithVolunteerId(r.Context(), volId)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
