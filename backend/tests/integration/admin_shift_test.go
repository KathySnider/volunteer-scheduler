package integration

import (
	"fmt"
	"testing"
)

// ============================================================================
// Mutation strings
// ============================================================================

const (
	mutAssignVolunteerToShift = `
		mutation AssignVolunteerToShift($shiftId: ID!, $volunteerId: ID!) {
			assignVolunteerToShift(shiftId: $shiftId, volunteerId: $volunteerId) {
				success message id
			}
		}`

	mutCancelShift = `
		mutation CancelShift($shiftId: ID!, $volunteerId: ID!) {
			cancelShift(shiftId: $shiftId, volunteerId: $volunteerId) {
				success message id
			}
		}`
)

// ============================================================================
// Tests
// ============================================================================

// TestAdminAssignVolunteerToShift verifies that an admin can assign a specific
// volunteer to a shift and that the volunteer_shifts row is created in the DB.
func TestAdminAssignVolunteerToShift(t *testing.T) {
	adminToken := makeAdminToken(t)
	_, volID := makeVolunteer(t)
	_, shiftID := seedEventWithShift(t, 5)

	resp := gqlPost(t, "/graphql/admin", adminToken, mutAssignVolunteerToShift, map[string]any{
		"shiftId":     fmt.Sprintf("%d", shiftID),
		"volunteerId": fmt.Sprintf("%d", volID),
	})

	// Register cleanup for the assignment row AFTER the mutation, so it runs
	// before the shift cleanup (LIFO), preventing FK constraint errors.
	t.Cleanup(func() {
		testDB.Exec(
			"DELETE FROM volunteer_shifts WHERE volunteer_id = $1 AND shift_id = $2",
			volID, shiftID,
		)
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "assignVolunteerToShift", &result)

	if !result.Success {
		var msg string
		if result.Message != nil {
			msg = *result.Message
		}
		t.Fatalf("assignVolunteerToShift returned success=false: %s", msg)
	}

	if !rowExists(t, `
		SELECT COUNT(*) FROM volunteer_shifts
		WHERE volunteer_id = $1 AND shift_id = $2 AND cancelled_at IS NULL
	`, volID, shiftID) {
		t.Error("expected an active volunteer_shifts row after assignVolunteerToShift")
	}
}

// TestAdminCancelShift verifies that an admin can cancel a volunteer's shift
// assignment by setting cancelled_at in the DB.
//
// Note: the service sets cancelled_at before attempting to send a confirmation
// email, so the DB assertion holds even when the test mailer is used.
func TestAdminCancelShift(t *testing.T) {
	adminToken := makeAdminToken(t)
	_, volID := makeVolunteer(t)
	_, shiftID := seedEventWithShift(t, 5)
	seedVolunteerShift(t, shiftID, volID)

	resp := gqlPost(t, "/graphql/admin", adminToken, mutCancelShift, map[string]any{
		"shiftId":     fmt.Sprintf("%d", shiftID),
		"volunteerId": fmt.Sprintf("%d", volID),
	})

	// The core DB update (cancelled_at) happens before the email send, so this
	// assertion holds regardless of email outcome.
	if !rowExists(t, `
		SELECT COUNT(*) FROM volunteer_shifts
		WHERE volunteer_id = $1 AND shift_id = $2 AND cancelled_at IS NOT NULL
	`, volID, shiftID) {
		t.Error("expected cancelled_at to be set in volunteer_shifts after admin cancelShift")
	}

	// Only check the success flag when no GQL errors (email may fail in tests).
	if !hasGQLErrors(resp) {
		var result mutationResult
		unmarshalField(t, resp, "cancelShift", &result)
		if !result.Success {
			t.Errorf("expected success=true when no GQL errors, got false (message: %v)", result.Message)
		}
	}
}
