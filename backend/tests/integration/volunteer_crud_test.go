package integration

import (
	"fmt"
	"testing"
)

// ============================================================================
// Mutation strings
// ============================================================================

const (
	mutCreateVolunteer = `mutation CreateVolunteer($input: NewVolunteerInput!) {
		createVolunteer(newVol: $input) { success message id }
	}`

	mutUpdateVolunteer = `mutation UpdateVolunteer($input: UpdateVolunteerInput!) {
		updateVolunteer(profile: $input) { success message id }
	}`

	mutDeleteVolunteer = `mutation DeleteVolunteer($id: ID!) {
		deleteVolunteer(volunteerId: $id) { success message id }
	}`
)

// ============================================================================
// Tests
// ============================================================================

func TestCreateVolunteer(t *testing.T) {
	token := makeAdminToken(t)
	email := uniqueEmail(t)

	// Register cleanup upfront so the row created by the mutation is removed
	// even if the test fails partway through.
	t.Cleanup(func() {
		testDB.Exec("DELETE FROM volunteers WHERE email = $1", email)
	})

	resp := gqlPost(t, "/graphql/admin", token, mutCreateVolunteer, map[string]any{
		"input": map[string]any{
			"firstName": "Jane",
			"lastName":  "Doe",
			"email":     email,
			"role":      "VOLUNTEER",
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "createVolunteer", &result)

	if !result.Success {
		t.Errorf("expected success=true, got false (message: %v)", result.Message)
	}
	if result.ID == nil {
		t.Error("expected id to be set in createVolunteer result")
	}
	if !rowExists(t, "SELECT COUNT(*) FROM volunteers WHERE email = $1 AND is_active = TRUE", email) {
		t.Error("expected active volunteer row in DB after createVolunteer")
	}
}

func TestCreateVolunteer_DuplicateEmail(t *testing.T) {
	token := makeAdminToken(t)
	email := uniqueEmail(t)

	// Seed a volunteer with this email first.
	seedVolunteer(t, email, "First", "Vol", "VOLUNTEER")

	resp := gqlPost(t, "/graphql/admin", token, mutCreateVolunteer, map[string]any{
		"input": map[string]any{
			"firstName": "Second",
			"lastName":  "Vol",
			"email":     email,
			"role":      "VOLUNTEER",
		},
	})

	// A duplicate email must fail — either via GQL errors or success=false.
	if !hasGQLErrors(resp) {
		var result mutationResult
		unmarshalField(t, resp, "createVolunteer", &result)
		if result.Success {
			t.Error("expected success=false for duplicate email, got true")
		}
	}
}

func TestUpdateVolunteer(t *testing.T) {
	token := makeAdminToken(t)
	email := uniqueEmail(t)
	volID := seedVolunteer(t, email, "Old", "Name", "VOLUNTEER")

	newEmail := uniqueEmail(t)

	resp := gqlPost(t, "/graphql/admin", token, mutUpdateVolunteer, map[string]any{
		"input": map[string]any{
			"id":        fmt.Sprintf("%d", volID),
			"firstName": "Updated",
			"lastName":  "Name",
			"email":     newEmail,
			"role":      "VOLUNTEER",
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "updateVolunteer", &result)

	if !result.Success {
		t.Errorf("expected success=true, got false (message: %v)", result.Message)
	}
	if !rowExists(t, "SELECT COUNT(*) FROM volunteers WHERE volunteer_id = $1 AND first_name = 'Updated'", volID) {
		t.Error("expected first_name='Updated' in DB after updateVolunteer")
	}
}

// TestDeleteVolunteer verifies that deleteVolunteer is a soft delete:
// the row remains in the DB but is_active is set to false.
func TestDeleteVolunteer(t *testing.T) {
	token := makeAdminToken(t)
	email := uniqueEmail(t)
	volID := seedVolunteer(t, email, "Temp", "Vol", "VOLUNTEER")

	resp := gqlPost(t, "/graphql/admin", token, mutDeleteVolunteer, map[string]any{
		"id": fmt.Sprintf("%d", volID),
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "deleteVolunteer", &result)

	if !result.Success {
		t.Errorf("expected success=true, got false (message: %v)", result.Message)
	}

	// Soft delete: the row must still exist.
	if !rowExists(t, "SELECT COUNT(*) FROM volunteers WHERE volunteer_id = $1", volID) {
		t.Error("volunteer row should still exist after soft delete")
	}
	// But is_active must be false.
	if rowExists(t, "SELECT COUNT(*) FROM volunteers WHERE volunteer_id = $1 AND is_active = TRUE", volID) {
		t.Error("expected is_active=false after deleteVolunteer")
	}
}
