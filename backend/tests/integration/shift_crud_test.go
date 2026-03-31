package integration

import (
	"fmt"
	"testing"
)

// ============================================================================
// Shared mutation strings
// ============================================================================

const (
	mutCreateShift = `
		mutation CreateShift($input: AddShiftInput!) {
			createShift(newShift: $input) { success message id }
		}`

	mutUpdateShift = `
		mutation UpdateShift($input: UpdateShiftInput!) {
			updateShift(shift: $input) { success message id }
		}`

	mutDeleteShift = `
		mutation DeleteShift($id: ID!) {
			deleteShift(shiftId: $id) { success message id }
		}`
)

// ============================================================================
// createShift
// ============================================================================

// TestCreateShift verifies that a shift can be added to an existing
// opportunity via the createShift mutation, and that the mutation returns
// a non-empty ID.
func TestCreateShift(t *testing.T) {
	token := makeAdminToken(t)
	jobID := getJobTypeID(t, "event_support")
	eventID := seedEvent(t, "Event For Shift Create", true, nil)
	oppID := seedOpportunity(t, eventID, jobID, true)
	// A pre-existing shift satisfies the "at least one shift per opportunity"
	// invariant; the mutation will add a second.
	seedShift(t, oppID, "2027-04-01T09:00:00Z", "2027-04-01T17:00:00Z", 5)

	resp := gqlPost(t, "/graphql/admin", token, mutCreateShift, map[string]any{
		"input": map[string]any{
			"opportunityId": fmt.Sprintf("%d", oppID),
			"startDateTime": "2027-04-15 09:00:00",
			"endDateTime":   "2027-04-15 17:00:00",
			"ianaZone":      "UTC",
			"maxVolunteers": 8,
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "createShift", &result)

	if !result.Success {
		t.Fatalf("createShift returned success=false: %v", result.Message)
	}
	if result.ID == nil || *result.ID == "" {
		t.Fatal("expected a non-empty shift ID in response")
	}

	shiftID := *result.ID
	t.Cleanup(func() { testDB.Exec("DELETE FROM shifts WHERE shift_id = $1", shiftID) })
}

// ============================================================================
// updateShift
// ============================================================================

// TestUpdateShift verifies that a shift's start and end times can be changed
// via the updateShift mutation, and that the change is persisted.
func TestUpdateShift(t *testing.T) {
	token := makeAdminToken(t)
	jobID := getJobTypeID(t, "advocacy")
	eventID := seedEvent(t, "Event For Shift Update", true, nil)
	oppID := seedOpportunity(t, eventID, jobID, true)
	shiftID := seedShift(t, oppID, "2027-05-01T09:00:00Z", "2027-05-01T17:00:00Z", 5)

	resp := gqlPost(t, "/graphql/admin", token, mutUpdateShift, map[string]any{
		"input": map[string]any{
			"id":            fmt.Sprintf("%d", shiftID),
			"startDateTime": "2027-05-10 10:00:00",
			"endDateTime":   "2027-05-10 18:00:00",
			"ianaZone":      "UTC",
			"maxVolunteers": 12,
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "updateShift", &result)

	if !result.Success {
		t.Fatalf("updateShift returned success=false: %v", result.Message)
	}
}

// ============================================================================
// deleteShift
// ============================================================================

// TestDeleteShift_Success verifies that a shift can be deleted when its
// parent opportunity has at least one other shift remaining.
func TestDeleteShift_Success(t *testing.T) {
	token := makeAdminToken(t)
	jobID := getJobTypeID(t, "volunteer_lead")
	eventID := seedEvent(t, "Event For Shift Delete", true, nil)
	oppID := seedOpportunity(t, eventID, jobID, true)

	// Seed two shifts so that deleting one leaves the opportunity intact.
	keepShiftID := seedShift(t, oppID, "2027-06-01T09:00:00Z", "2027-06-01T13:00:00Z", 5)
	deleteShiftID := seedShift(t, oppID, "2027-06-01T14:00:00Z", "2027-06-01T18:00:00Z", 5)

	resp := gqlPost(t, "/graphql/admin", token, mutDeleteShift, map[string]any{
		"id": fmt.Sprintf("%d", deleteShiftID),
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "deleteShift", &result)

	if !result.Success {
		t.Fatalf("deleteShift returned success=false: %v", result.Message)
	}

	if rowExists(t, "SELECT COUNT(*) FROM shifts WHERE shift_id = $1", deleteShiftID) {
		t.Error("expected deleted shift to be gone from the DB")
	}
	// The sibling shift must be unaffected.
	if !rowExists(t, "SELECT COUNT(*) FROM shifts WHERE shift_id = $1", keepShiftID) {
		t.Error("sibling shift should not be removed when a different shift is deleted")
	}
}

// TestDeleteShift_LastShift verifies that the service rejects a deletion
// request when it would leave an opportunity with zero shifts.
func TestDeleteShift_LastShift(t *testing.T) {
	token := makeAdminToken(t)
	jobID := getJobTypeID(t, "attendee_only")
	eventID := seedEvent(t, "Event For Last Shift Delete", true, nil)
	oppID := seedOpportunity(t, eventID, jobID, true)
	shiftID := seedShift(t, oppID, "2027-07-01T09:00:00Z", "2027-07-01T17:00:00Z", 5)

	resp := gqlPost(t, "/graphql/admin", token, mutDeleteShift, map[string]any{
		"id": fmt.Sprintf("%d", shiftID),
	})

	if !hasGQLErrors(resp) {
		t.Error("expected a GraphQL error when attempting to delete the last shift of an opportunity")
	}

	// The shift must still be in the DB — the deletion was rejected.
	if !rowExists(t, "SELECT COUNT(*) FROM shifts WHERE shift_id = $1", shiftID) {
		t.Error("last shift should remain in the DB after a rejected delete")
	}
}
