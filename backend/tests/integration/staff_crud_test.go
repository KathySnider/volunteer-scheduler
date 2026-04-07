package integration

import (
	"fmt"
	"testing"
)

// ============================================================================
// Mutation strings
// ============================================================================

const (
	mutCreateStaff = `
		mutation CreateStaff($input: NewStaffInput!) {
			createStaff(newStaff: $input) { success message id }
		}`

	mutUpdateStaff = `
		mutation UpdateStaff($input: UpdateStaffInput!) {
			updateStaff(staff: $input) { success message id }
		}`

	mutDeleteStaff = `
		mutation DeleteStaff($id: ID!) {
			deleteStaff(staffId: $id) { success message id }
		}`
)

// ============================================================================
// createStaff
// ============================================================================

// TestCreateStaff verifies that a new staff member can be created via the
// admin mutation and that the returned ID maps to a row in the DB.
func TestCreateStaff(t *testing.T) {
	token := makeAdminToken(t)
	email := uniqueEmail(t)

	resp := gqlPost(t, "/graphql/admin", token, mutCreateStaff, map[string]any{
		"input": map[string]any{
			"firstName": "New",
			"lastName":  "Staffer",
			"email":     email,
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "createStaff", &result)

	if !result.Success {
		var msg string
		if result.Message != nil {
			msg = *result.Message
		}
		t.Fatalf("createStaff returned success=false: %s", msg)
	}
	if result.ID == nil || *result.ID == "" {
		t.Fatal("expected a non-empty staff ID in response")
	}

	staffID := *result.ID
	t.Cleanup(func() { testDB.Exec("DELETE FROM staff WHERE staff_id = $1", staffID) })

	if !rowExists(t, "SELECT COUNT(*) FROM staff WHERE staff_id = $1", staffID) {
		t.Errorf("expected staff row in DB for id=%s", staffID)
	}
}

// ============================================================================
// updateStaff
// ============================================================================

// TestUpdateStaff verifies that a staff member's name can be changed and the
// update is persisted to the DB.
func TestUpdateStaff(t *testing.T) {
	token := makeAdminToken(t)
	email := uniqueEmail(t)
	staffID := seedStaff(t, "Before", "Update", email)

	resp := gqlPost(t, "/graphql/admin", token, mutUpdateStaff, map[string]any{
		"input": map[string]any{
			"id":        fmt.Sprintf("%d", staffID),
			"firstName": "After",
			"lastName":  "Update",
			"email":     email,
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "updateStaff", &result)

	if !result.Success {
		var msg string
		if result.Message != nil {
			msg = *result.Message
		}
		t.Fatalf("updateStaff returned success=false: %s", msg)
	}

	var firstName string
	if err := testDB.QueryRow(
		"SELECT first_name FROM staff WHERE staff_id = $1", staffID,
	).Scan(&firstName); err != nil {
		t.Fatalf("querying updated staff: %v", err)
	}
	if firstName != "After" {
		t.Errorf("expected first_name='After', got %q", firstName)
	}
}

// ============================================================================
// deleteStaff
// ============================================================================

// TestDeleteStaff verifies that deleteStaff removes the staff row from the DB.
func TestDeleteStaff(t *testing.T) {
	token := makeAdminToken(t)
	staffID := seedStaff(t, "To", "Delete", uniqueEmail(t))

	resp := gqlPost(t, "/graphql/admin", token, mutDeleteStaff, map[string]any{
		"id": fmt.Sprintf("%d", staffID),
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "deleteStaff", &result)

	if !result.Success {
		var msg string
		if result.Message != nil {
			msg = *result.Message
		}
		t.Fatalf("deleteStaff returned success=false: %s", msg)
	}

	if rowExists(t, "SELECT COUNT(*) FROM staff WHERE staff_id = $1", staffID) {
		t.Error("expected staff row to be gone from the DB after deleteStaff")
	}
}
