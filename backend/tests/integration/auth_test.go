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
		}
	}`

// mutLogout sends no token — the server reads it from the session cookie.
const mutLogout = `mutation { logout { success } }`

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

// TestConsumeMagicLink_Valid verifies the full happy path: a valid token sets
// an HttpOnly session cookie and marks the magic link as used.
func TestConsumeMagicLink_Valid(t *testing.T) {
	email := uniqueEmail(t)
	seedVolunteer(t, email, "Alice", "Test", "VOLUNTEER")

	token := "valid-token-" + email
	seedMagicLink(t, email, token, time.Now().Add(15*time.Minute))

	resp, cookies := gqlPostFull(t, "/graphql/auth", "", mutConsumeMagicLink, map[string]any{
		"token": token,
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GraphQL errors: %v", resp.Errors)
	}

	var result struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Email   string `json:"email"`
	}
	unmarshalField(t, resp, "consumeMagicLink", &result)

	if !result.Success {
		t.Fatalf("expected success=true, got false; message: %s", result.Message)
	}
	if result.Email != email {
		t.Errorf("expected email=%q, got %q", email, result.Email)
	}

	// The magic link should be marked as used.
	if !magicLinkUsed(t, token) {
		t.Error("expected magic link to be marked used, but it was not")
	}

	// The response must set a session cookie.
	sessionCookie := findCookie(cookies, "session")
	if sessionCookie == nil {
		t.Fatal("expected a Set-Cookie: session header in the login response")
	}
	if sessionCookie.Value == "" {
		t.Error("session cookie value must not be empty")
	}
	if !sessionCookie.HttpOnly {
		t.Error("session cookie must have HttpOnly attribute")
	}

	// A session should exist in the DB for that cookie value.
	if !sessionExists(t, sessionCookie.Value) {
		t.Error("expected session row in DB matching the cookie value")
	}
}

// TestConsumeMagicLink_SessionExpiry verifies that the session created on
// login has an expires_at matching SESSION_MAX_AGE (86400 s in tests).
func TestConsumeMagicLink_SessionExpiry(t *testing.T) {
	email := uniqueEmail(t)
	seedVolunteer(t, email, "Expiry", "Test", "VOLUNTEER")

	mlToken := "expiry-token-" + email
	seedMagicLink(t, email, mlToken, time.Now().Add(15*time.Minute))

	_, cookies := gqlPostFull(t, "/graphql/auth", "", mutConsumeMagicLink, map[string]any{
		"token": mlToken,
	})

	sessionCookie := findCookie(cookies, "session")
	if sessionCookie == nil {
		t.Fatal("no session cookie set")
	}

	// Check the DB row's expires_at is roughly SESSION_MAX_AGE seconds away.
	// The DB stores the SHA-256 hash of the token, so hash before querying.
	var secondsRemaining float64
	err := testDB.QueryRow(
		`SELECT EXTRACT(EPOCH FROM (expires_at - NOW())) FROM sessions WHERE token = $1`,
		hashSessionToken(sessionCookie.Value),
	).Scan(&secondsRemaining)
	if err != nil {
		t.Fatalf("could not query session expiry: %v", err)
	}

	const sessionMaxAge = 86400 // matches SESSION_MAX_AGE env var in setup_test.go
	if secondsRemaining < float64(sessionMaxAge)-60 || secondsRemaining > float64(sessionMaxAge)+60 {
		t.Errorf("expected session to expire in ~%d seconds, got %.0f", sessionMaxAge, secondsRemaining)
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

// TestLogout_Valid verifies that logging out via cookie deletes the session
// and returns a Set-Cookie that clears the browser cookie.
func TestLogout_Valid(t *testing.T) {
	email := uniqueEmail(t)
	volID := seedVolunteer(t, email, "Dave", "Test", "VOLUNTEER")
	sessionToken := seedSession(t, email, volID, "VOLUNTEER", "logout-test-token-"+email)

	// Send the logout mutation with the session token in the Cookie header.
	resp, cookies := gqlPostFullCookie(t, "/graphql/auth", sessionToken, mutLogout, nil)

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

	// Session should no longer exist in the DB.
	if sessionExists(t, sessionToken) {
		t.Error("expected session to be deleted after logout, but it still exists")
	}

	// The response should clear the session cookie.
	cleared := findCookie(cookies, "session")
	if cleared == nil {
		t.Fatal("expected a Set-Cookie: session header clearing the cookie")
	}
	if cleared.MaxAge >= 0 && cleared.Expires.After(time.Now()) {
		t.Errorf("expected cookie to be cleared (MaxAge<0 or past Expires), got MaxAge=%d Expires=%v",
			cleared.MaxAge, cleared.Expires)
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

// TestLogout_UnknownCookie verifies that logging out with a non-existent
// session cookie value is a no-op — it returns success and clears the cookie
// without erroring.
func TestLogout_UnknownCookie(t *testing.T) {
	resp, cookies := gqlPostFullCookie(t, "/graphql/auth", "cookie-that-never-existed", mutLogout, nil)

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GraphQL errors: %v", resp.Errors)
	}

	var result struct {
		Success bool `json:"success"`
	}
	unmarshalField(t, resp, "logout", &result)

	if !result.Success {
		t.Error("expected logout of unknown cookie to return success=true, got false")
	}

	// Cookie should still be cleared even though the session didn't exist.
	cleared := findCookie(cookies, "session")
	if cleared == nil {
		t.Fatal("expected Set-Cookie clearing header even for unknown session")
	}
}

// TestLogout_NoCookie verifies that calling logout with no cookie at all
// still returns success (best-effort).
func TestLogout_NoCookie(t *testing.T) {
	resp := gqlPost(t, "/graphql/auth", "", mutLogout, nil)

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GraphQL errors: %v", resp.Errors)
	}

	var result struct {
		Success bool `json:"success"`
	}
	unmarshalField(t, resp, "logout", &result)

	if !result.Success {
		t.Error("expected logout with no cookie to return success=true")
	}
}
