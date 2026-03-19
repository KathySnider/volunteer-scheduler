package middleware

import "context"

type contextKey string

const volunteerIDContextKey contextKey = "volunteerID"
const volunteerRoleContextKey contextKey = "volunteerRole"

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
