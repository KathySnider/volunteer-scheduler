package middleware

import (
	"context"
	"net/http"
)

// These key are "private" to avoid collisions with other packages.
type contextKey string

const (
	volunteerIDContextKey   contextKey = "volunteerID"
	volunteerRoleContextKey contextKey = "volunteerRole"
	responseWriterKey       contextKey = "httpResponseWriter"
	httpRequestKey          contextKey = "httpRequest"
)

// ContextWithVolunteerId stores the authenticated volunteer id in the context.
func ContextWithVolunteerId(ctx context.Context, volId int) context.Context {
	return context.WithValue(ctx, volunteerIDContextKey, volId)
}

// VolunteerIdFromContext retrieves the authenticated volunteer id from the context.
func VolunteerIdFromContext(ctx context.Context) (int, bool) {
	volId, ok := ctx.Value(volunteerIDContextKey).(int)
	return volId, ok
}

// ContextWithVolunteerRole stores the authenticated volunteer role in the context.
func ContextWithVolunteerRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, volunteerRoleContextKey, role)
}

// VolunteerRoleFromContext retrieves the authenticated volunteer role from the context.
func VolunteerRoleFromContext(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(volunteerRoleContextKey).(string)
	return role, ok
}

// HTTP keys
// HTTPresponse writer key - so the resolver can call SetCookie() when the user logs in or out.

// ContextWithResponseWriter stores the response writer in the context.
func ContextWithResponseWriter(ctx context.Context, writer http.ResponseWriter) context.Context {
	return context.WithValue(ctx, responseWriterKey, writer)
}
func ResponseWriterFromContext(ctx context.Context) (http.ResponseWriter, bool) {
	writer, ok := ctx.Value(responseWriterKey).(http.ResponseWriter)
	return writer, ok
}

// HTTP request key - so the resolver can call SetCookie("session") to read the session cookie. This the token is not needed
// on the frontend at all.

// ContextWithRequest stores the request in the context.
func ContextWithRequest(ctx context.Context, request *http.Request) context.Context {
	return context.WithValue(ctx, httpRequestKey, request)
}
func RequestFromContext(ctx context.Context) (*http.Request, bool) {
	request, ok := ctx.Value(httpRequestKey).(*http.Request)
	return request, ok
}

// WithHTTPContext wraps an http.Handler to inject the ResponseWriter and
// Request into the context so resolvers can read and set cookies.
// Used on the /graphql/auth endpoint (login/logout). The authenticated
// endpoints get the same injection via RequireAuth.
func WithHTTPContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := ContextWithResponseWriter(r.Context(), w)
		ctx = ContextWithRequest(ctx, r)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
