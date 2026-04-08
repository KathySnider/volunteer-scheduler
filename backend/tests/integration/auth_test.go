package integration

import (
	"testing"
	"time"
)

// ============================================================================
// Shared GraphQL mutation strings
// ============================================================================

const mutRequestMagicLink = `
	mutation RequestMagicLink($email: String!) {
		requestMagicLink(email: $email) {
			success
			message
			email
		}
	}`

const mutConsumeMagicLink = `
	mutation ConsumeMagicLink($token: String!) {
		consumeMagicLink(token: $token) {
			success
			message
			email
			sessionToken
		}
	}`

const mutLogout = `
	mutation Logout($token: String!) {
		logout(token: $token) {
			success
		}
	}`

// ============================================================================
// requestMagicLink
// ============================================================================

// TestRequestMagicLink_Success verifies that a magic link request for a known
// active volunteer returns success and echoes the email address.
func TestRequestMagicLink_Success(t *testing.T) {
	email := uniqueEmail(t)
	seedVolunteer(t, email, "Alice", "Test", "VOLUNTEER")

	resp := gqlPost(t, "/graphql/auth", "", mutRequestMagicLink, map[string]any{
		"email": email,
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GraphQL errors: %v", resp.Errors)
	}

	var result struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Email   string `json:"email"`
	}
	unmarshalField(t, resp, "requestMagicLink", &result)

	if !result.Success {
		t.Errorf("expected success=true, got false; message: %s", result.Message)
	}
	if result.Email != email {
		t.Errorf("expected email=%q, got %q", email, result.Email)
	}
}

// TestRequestMagicLink_UnknownEmail verifies that a magic link request for an
// email not in the database returns success=false.
func TestRequestMagicLink_UnknownEmail(t *testing.T) {
	email := uniqueEmail(t)

	resp := gqlPost(t, "/graphql/auth", "", mutRequestMagicLink, map[string]any{
		"email": email,
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GraphQL errors: %v", resp.Errors)
	}

	var result struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	unmarshalField(t, resp, "requestMagicLink", &result)

	if result.Success {
		t.Error("expected success=false for unknown email, got true")
	}
}

// TestRequestMagicLink_InactiveAccount verifies that a magic link request for
// an inactive volunteer's email returns success=false.
func TestRequestMagicLink_InactiveAccount(t *testing.T) {
	email := uniqueEmail(t)
	seedInactiveVolunteer(t, email, "Inactive", "Volunteer", "VOLUNTEER")

	resp := gqlPost(t, "/graphql/auth", "", mutRequestMagicLink, map[string]any{
		"email": email,
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GraphQL errors: %v", resp.Errors)
	}

	var result struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	unmarshalField(t, resp, "requestMagicLink", &result)

	if result.Success {
		t.Error("expected success=false for inactive account, got true")
	}
}

// TestRequestMagicLink_RateLimit verifies that more than 5 requests for the
// same email within an hour are rejected.
func TestRequestMagicLink_RateLimit(t *testing.T) {
	email := uniqueEmail(t)
	seedVolunteer(t, email, "Bob", "Test", "VOLUNTEER")
	t.Cleanup(func() {
		testDB.Exec("DELETE FROM magic_links WHERE email = $1", email)
	})

	// First 5 should succeed.
	for i := 0; i < 5; i++ {
		resp := gqlPost(t, "/graphql/auth", "", mutRequestMagicLink, map[string]any{
			"email": email,
		})
		var result struct {
			Success bool `json:"success"`
		}
		unmarshalField(t, resp, "requestMagicLink", &result)
		if !result.Success {
			t.Fatalf("request %d unexpectedly failed", i+1)
		}
	}

	// 6th should be rejected.
	resp := gqlPost(t, "/graphql/auth", "", mutRequestMagicLink, map[string]any{
		"email": email,
	})
	var result struct {
		Success bool `json:"success"`
	}
	unmarshalField(t, resp, "requestMagicLink", &result)
	if result.Success {
		t.Error("expected rate limit to reject 6th request, but got success=true")
	}
}

// ============================================================================
// consumeMagicLink
// ============================================================================

// TestConsumeMagicLink_Valid verifies the full happy path: a valid token
// produces a session token and marks the magic link as used.
func TestConsumeMagicLink_Valid(t *testing.T) {
	email := uniqueEmail(t)
	seedVolunteer(t, email, "Alice", "Test", "VOLUNTEER")

	token := "valid-token-" + email
	seedMagicLink(t, email, token, time.Now().Add(15*time.Minute))

	resp := gqlPost(t, "/graphql/auth", "", mutConsumeMagicLink, map[string]any{
		"token": token,
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GraphQL errors: %v", resp.Errors)
	}

	var result struct {
		Success      bool   `json:"success"`
		Message      string `json:"message"`
		Email        string `json:"email"`
		SessionToken string `json:"sessionToken"`
	}
	unmarshalField(t, resp, "consumeMagicLink", &result)

	if !result.Success {
		t.Fatalf("expected success=true, got false; message: %s", result.Message)
	}
	if result.SessionToken == "" {
		t.Error("expected a session token, got empty string")
	}
	if result.Email != email {
		t.Errorf("expected email=%q, got %q", email, result.Email)
	}

	// The magic link should be marked as used.
	if !magicLinkUsed(t, token) {
		t.Error("expected magic link to be marked used, but it was not")
	}

	// A session should exist in the DB.
	if !sessionExists(t, result.SessionToken) {
		t.Error("expected session to exist in DB, but it does not")
	}
}

// TestConsumeMagicLink_InvalidToken verifies that a made-up token is rejected.
func TestConsumeMagicLink_InvalidToken(t *testing.T) {
	resp := gqlPost(t, "/graphql/auth", "", mutConsumeMagicLink, map[string]any{
		"token": "this-token-does-not-exist",
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GraphQL errors: %v", resp.Errors)
	}

	var result struct {
		Success bool `json:"success"`
	}
	unmarshalField(t, resp, "consumeMagicLink", &result)

	if result.Success {
		t.Error("expected success=false for invalid token, got true")
	}
}

// TestConsumeMagicLink_ExpiredToken verifies that an expired token is rejected.
func TestConsumeMagicLink_ExpiredToken(t *testing.T) {
	email := uniqueEmail(t)
	seedVolunteer(t, email, "Bob", "Test", "VOLUNTEER")

	token := "expired-token-" + email
	seedMagicLink(t, email, token, time.Now().Add(-1*time.Minute)) // already expired

	resp := gqlPost(t, "/graphql/auth", "", mutConsumeMagicLink, map[string]any{
		"token": token,
	})

	var result struct {
		Success bool `json:"success"`
	}
	unmarshalField(t, resp, "consumeMagicLink", &result)

	if result.Success {
		t.Error("expected success=false for expired token, got true")
	}
}

// TestConsumeMagicLink_AlreadyUsed verifies that a token can only be used once.
func TestConsumeMagicLink_AlreadyUsed(t *testing.T) {
	email := uniqueEmail(t)
	seedVolunteer(t, email, "Carol", "Test", "VOLUNTEER")

	token := "reuse-token-" + email
	seedMagicLink(t, email, token, time.Now().Add(15*time.Minute))

	// First use — should succeed.
	resp := gqlPost(t, "/graphql/auth", "", mutConsumeMagicLink, map[string]any{
		"token": token,
	})
	var first struct {
		Success bool `json:"success"`
	}
	unmarshalField(t, resp, "consumeMagicLink", &first)
	if !first.Success {
		t.Fatal("first consume unexpectedly failed")
	}

	// Second use — should fail.
	resp = gqlPost(t, "/graphql/auth", "", mutConsumeMagicLink, map[string]any{
		"token": token,
	})
	var second struct {
		Success bool `json:"success"`
	}
	unmarshalField(t, resp, "consumeMagicLink", &second)
	if second.Success {
		t.Error("expected second consume to fail, but got success=true")
	}
}

// ============================================================================
// logout
// ============================================================================

// TestLogout_Valid verifies that logging out deletes the session.
func TestLogout_Valid(t *testing.T) {
	email := uniqueEmail(t)
	volID := seedVolunteer(t, email, "Dave", "Test", "VOLUNTEER")
	token := seedSession(t, email, volID, "VOLUNTEER", "logout-test-token-"+email)

	resp := gqlPost(t, "/graphql/auth", "", mutLogout, map[string]any{
		"token": token,
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GraphQL errors: %v", resp.Errors)
	}

	var result struct {
		Success bool `json:"success"`
	}
	unmarshalField(t, resp, "logout", &result)

	if !result.Success {
		t.Error("expected logout success=true, got false")
	}

	// Session should no longer exist.
	if sessionExists(t, token) {
		t.Error("expected session to be deleted after logout, but it still exists")
	}
}

// ============================================================================
// requestAccount
// ============================================================================

const mutRequestAccount = `
	mutation RequestAccount($email: String!, $firstName: String!, $lastName: String!) {
		requestAccount(email: $email, firstName: $firstName, lastName: $lastName) {
			success
			message
		}
	}`

// TestRequestAccount_NewVolunteer verifies that an account request for an
// unknown email (no existing record) returns success=true.
func TestRequestAccount_NewVolunteer(t *testing.T) {
	// Seed an admin so the notification email has somewhere to go.
	adminEmail := uniqueEmail(t)
	seedVolunteer(t, adminEmail, "Admin", "User", "ADMINISTRATOR")

	email := uniqueEmail(t)

	resp := gqlPost(t, "/graphql/auth", "", mutRequestAccount, map[string]any{
		"email":     email,
		"firstName": "Jane",
		"lastName":  "Doe",
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GraphQL errors: %v", resp.Errors)
	}

	var result struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	unmarshalField(t, resp, "requestAccount", &result)

	if !result.Success {
		t.Errorf("expected success=true, got false; message: %s", result.Message)
	}
}

// TestRequestAccount_InactiveVolunteer verifies that an account request for
// an email belonging to an inactive volunteer also returns success=true.
// The admin receives a reactivation email rather than a new-account email,
// but from the requester's perspective the result is identical.
func TestRequestAccount_InactiveVolunteer(t *testing.T) {
	// Seed an admin so the notification email has somewhere to go.
	adminEmail := uniqueEmail(t)
	seedVolunteer(t, adminEmail, "Admin", "User", "ADMINISTRATOR")

	email := uniqueEmail(t)
	seedInactiveVolunteer(t, email, "Former", "Volunteer", "VOLUNTEER")

	resp := gqlPost(t, "/graphql/auth", "", mutRequestAccount, map[string]any{
		"email":     email,
		"firstName": "Former",
		"lastName":  "Volunteer",
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GraphQL errors: %v", resp.Errors)
	}

	var result struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	unmarshalField(t, resp, "requestAccount", &result)

	if !result.Success {
		t.Errorf("expected success=true, got false; message: %s", result.Message)
	}
}

// TestRequestAccount_NoAdmins verifies that an account request when no admins
// exist still returns success=true — we don't reveal internal system state.
func TestRequestAccount_NoAdmins(t *testing.T) {
	email := uniqueEmail(t)

	resp := gqlPost(t, "/graphql/auth", "", mutRequestAccount, map[string]any{
		"email":     email,
		"firstName": "No",
		"lastName":  "Admin",
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GraphQL errors: %v", resp.Errors)
	}

	var result struct {
		Success bool `json:"success"`
	}
	unmarshalField(t, resp, "requestAccount", &result)

	if !result.Success {
		t.Error("expected success=true even with no admins, got false")
	}
}

// TestLogout_UnknownToken verifies that logging out with a non-existent token
// does not error — it's a no-op from the user's perspective.
func TestLogout_UnknownToken(t *testing.T) {
	resp := gqlPost(t, "/graphql/auth", "", mutLogout, map[string]any{
		"token": "token-that-never-existed",
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GraphQL errors: %v", resp.Errors)
	}

	var result struct {
		Success bool `json:"success"`
	}
	unmarshalField(t, resp, "logout", &result)

	// Deleting a non-existent session is not an error.
	if !result.Success {
		t.Error("expected logout of unknown token to return success=true, got false")
	}
}
