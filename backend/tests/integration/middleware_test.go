package integration

import (
	"testing"
)

// A minimal query that any authenticated user can call (shared by both
// volunteer and admin schemas).
const queryLookupValues = `
	query {
		lookupValues {
			regions {
				id
				name
			}
		}
	}`

// ============================================================================
// RequireAuth middleware (/graphql/volunteer)
// ============================================================================

// TestRequireAuth_NoToken verifies that hitting the volunteer endpoint without
// an Authorization header is rejected.
func TestRequireAuth_NoToken(t *testing.T) {
	resp := gqlPost(t, "/graphql/volunteer", "", queryLookupValues, nil)

	if !hasGQLErrors(resp) {
		t.Error("expected auth error with no token, got none")
	}
}

// TestRequireAuth_ValidVolunteerToken verifies that a valid volunteer session
// token is accepted by the volunteer endpoint.
func TestRequireAuth_ValidVolunteerToken(t *testing.T) {
	email := uniqueEmail(t)
	volID := seedVolunteer(t, email, "Eve", "Test", "VOLUNTEER")
	token := seedSession(t, email, volID, "VOLUNTEER", "vol-auth-token-"+email)

	resp := gqlPost(t, "/graphql/volunteer", token, queryLookupValues, nil)

	if hasGQLErrors(resp) {
		t.Errorf("expected no errors with valid volunteer token, got: %v", resp.Errors)
	}
}

// TestRequireAuth_ExpiredToken verifies that an expired session token is rejected.
func TestRequireAuth_ExpiredToken(t *testing.T) {
	email := uniqueEmail(t)
	volID := seedVolunteer(t, email, "Frank", "Test", "VOLUNTEER")
	token := seedExpiredSession(t, email, volID, "VOLUNTEER", "expired-vol-token-"+email)

	resp := gqlPost(t, "/graphql/volunteer", token, queryLookupValues, nil)

	if !hasGQLErrors(resp) {
		t.Error("expected auth error with expired token, got none")
	}
}

// TestRequireAuth_InvalidToken verifies that a made-up token is rejected.
func TestRequireAuth_InvalidToken(t *testing.T) {
	resp := gqlPost(t, "/graphql/volunteer", "not-a-real-token", queryLookupValues, nil)

	if !hasGQLErrors(resp) {
		t.Error("expected auth error with invalid token, got none")
	}
}

// ============================================================================
// RequireAdmin middleware (/graphql/admin)
// ============================================================================

// TestRequireAdmin_NoToken verifies that the admin endpoint rejects requests
// with no token.
func TestRequireAdmin_NoToken(t *testing.T) {
	resp := gqlPost(t, "/graphql/admin", "", queryLookupValues, nil)

	if !hasGQLErrors(resp) {
		t.Error("expected auth error with no token on admin endpoint, got none")
	}
}

// TestRequireAdmin_VolunteerTokenRejected verifies that a valid volunteer
// session token is NOT accepted by the admin endpoint.
func TestRequireAdmin_VolunteerTokenRejected(t *testing.T) {
	email := uniqueEmail(t)
	volID := seedVolunteer(t, email, "Grace", "Test", "VOLUNTEER")
	token := seedSession(t, email, volID, "VOLUNTEER", "vol-tries-admin-"+email)

	resp := gqlPost(t, "/graphql/admin", token, queryLookupValues, nil)

	if !hasGQLErrors(resp) {
		t.Error("expected admin endpoint to reject volunteer token, but got no errors")
	}
}

// TestRequireAdmin_AdminTokenAccepted verifies that a valid administrator
// session token is accepted by the admin endpoint.
func TestRequireAdmin_AdminTokenAccepted(t *testing.T) {
	email := uniqueEmail(t)
	volID := seedVolunteer(t, email, "Hank", "Admin", "ADMINISTRATOR")
	token := seedSession(t, email, volID, "ADMINISTRATOR", "admin-auth-token-"+email)

	resp := gqlPost(t, "/graphql/admin", token, queryLookupValues, nil)

	if hasGQLErrors(resp) {
		t.Errorf("expected admin token to be accepted, got errors: %v", resp.Errors)
	}
}

// TestRequireAdmin_ExpiredAdminToken verifies that an expired admin token is
// rejected even though the role is correct.
func TestRequireAdmin_ExpiredAdminToken(t *testing.T) {
	email := uniqueEmail(t)
	volID := seedVolunteer(t, email, "Iris", "Admin", "ADMINISTRATOR")
	token := seedExpiredSession(t, email, volID, "ADMINISTRATOR", "expired-admin-token-"+email)

	resp := gqlPost(t, "/graphql/admin", token, queryLookupValues, nil)

	if !hasGQLErrors(resp) {
		t.Error("expected expired admin token to be rejected, got no errors")
	}
}
