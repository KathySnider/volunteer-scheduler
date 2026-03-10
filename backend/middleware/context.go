package middleware

import "context"

type contextKey string

const volunteerIDContextKey contextKey = "volunteerID"

// ContextWithVolunteerId stores the authenticated volunteer id in the context.
func ContextWithVolunteerId(ctx context.Context, volId int) context.Context {
	return context.WithValue(ctx, volunteerIDContextKey, volId)
}

// VolunteerIdFromContext retrieves the authenticated volunteer id from the context.
func VolunteerIdFromContext(ctx context.Context) (int, bool) {
	volId, ok := ctx.Value(volunteerIDContextKey).(int)
	return volId, ok
}
