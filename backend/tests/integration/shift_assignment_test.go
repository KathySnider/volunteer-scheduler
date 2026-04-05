package integration

import (
	"fmt"
	"testing"
)

// ============================================================================
// Mutation / query strings
// ============================================================================

const (
	mutAssignSelfToShift = `mutation AssignSelfToShift($shiftId: ID!) {
		assignSelfToShift(shiftId: $shiftId) { success message id }
	}`

	mutCancelOwnShift = `mutation CancelOwnShift($shiftId: ID!) {
		cancelOwnShift(shiftId: $shiftId) { success message id }
	}`

	qryOwnShifts = `query OwnShifts($filter: ShiftTimeFilter!) {
		ownShifts(filter: $filter) { shiftId jobName eventName startDateTime endDateTime }
	}`

	qryShiftsForEvent = `query ShiftsForEvent($eventId: ID!) {
		shiftsForEvent(eventId: $eventId) {
			id jobName startDateTime endDateTime isVirtual maxVolunteers assignedVolunteers
		}
	}`
)

// ============================================================================
// Local response types
// ============================================================================

type ownShiftResult struct {
	ShiftId       string `json:"shiftId"`
	JobName       string `json:"jobName"`
	EventName     string `json:"eventName"`
	StartDateTime string `json:"startDateTime"`
	EndDateTime   string `json:"endDateTime"`
}

type shiftViewResult struct {
	ID                 string `json:"id"`
	JobName            string `json:"jobName"`
	StartDateTime      string `json:"startDateTime"`
	EndDateTime        string `json:"endDateTime"`
	IsVirtual          bool   `json:"isVirtual"`
	MaxVolunteers      *int   `json:"maxVolunteers"`
	AssignedVolunteers int    `json:"assignedVolunteers"`
}

// ============================================================================
// Shared setup helper
// ============================================================================

// seedEventWithShift seeds the full event → opportunity → shift chain needed
// by assignment tests and returns (eventID, shiftID). All rows are cleaned up
// via t.Cleanup in LIFO order, so the shift is deleted before the opportunity,
// which is deleted before the event, etc.
func seedEventWithShift(t *testing.T, maxVolunteers int) (int, int) {
	t.Helper()
	jobTypeID := seedJobType(t, uniqueCode(t, "jt"), "Test Job")
	eventID := seedEvent(t, "Shift Test Event", true, nil)
	oppID := seedOpportunity(t, eventID, jobTypeID, true)
	shiftID := seedShift(t, oppID, "2027-06-01T09:00:00Z", "2027-06-01T12:00:00Z", maxVolunteers)
	return eventID, shiftID
}

// ============================================================================
// Tests
// ============================================================================

// TestAssignSelfToShift verifies the happy path: a volunteer assigns themselves
// to an open shift and the row appears in volunteer_shifts.
func TestAssignSelfToShift(t *testing.T) {
	token, volID := makeVolunteer(t)
	_, shiftID := seedEventWithShift(t, 2)

	resp := gqlPost(t, "/graphql/volunteer", token, mutAssignSelfToShift, map[string]any{
		"shiftId": fmt.Sprintf("%d", shiftID),
	})

	// Register cleanup for the assignment row the mutation will create.
	// Registered after seedShift's cleanup so it runs first (LIFO), which
	// prevents FK constraint errors when the shift is later deleted.
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
	unmarshalField(t, resp, "assignSelfToShift", &result)

	if !result.Success {
		t.Errorf("expected success=true, got false (message: %v)", result.Message)
	}
	if !rowExists(t, `
		SELECT COUNT(*) FROM volunteer_shifts
		WHERE volunteer_id = $1 AND shift_id = $2 AND cancelled_at IS NULL
	`, volID, shiftID) {
		t.Error("expected an active volunteer_shifts row after assignSelfToShift")
	}
}

// TestAssignSelfToShift_Full verifies that when a shift is already at capacity
// the service returns success=false with no GQL-level error.
func TestAssignSelfToShift_Full(t *testing.T) {
	token, _ := makeVolunteer(t)
	_, otherID := makeVolunteer(t) // fills the shift
	_, shiftID := seedEventWithShift(t, 1)

	// Fill the one available slot.
	seedVolunteerShift(t, shiftID, otherID)

	resp := gqlPost(t, "/graphql/volunteer", token, mutAssignSelfToShift, map[string]any{
		"shiftId": fmt.Sprintf("%d", shiftID),
	})

	// Full-shift returns success=false with nil error — no GQL error.
	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors for full-shift assignment: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "assignSelfToShift", &result)

	if result.Success {
		t.Error("expected success=false when shift is full, got true")
	}
}

// TestCancelOwnShift verifies that a volunteer can cancel their own shift
// assignment. The service sets cancelled_at in the DB before attempting to
// send a confirmation email.
//
// Note: if the test server's mailer cannot deliver email, cancelOwnShift
// returns success=false and the resolver surfaces a GQL error. The DB update
// (setting cancelled_at) still occurs before the email attempt, so the DB
// assertion below passes regardless of email outcome.
func TestCancelOwnShift(t *testing.T) {
	token, volID := makeVolunteer(t)
	_, shiftID := seedEventWithShift(t, 2)
	seedVolunteerShift(t, shiftID, volID)

	resp := gqlPost(t, "/graphql/volunteer", token, mutCancelOwnShift, map[string]any{
		"shiftId": fmt.Sprintf("%d", shiftID),
	})

	// The core DB update happens before the email send, so this assertion holds
	// even when the mailer fails.
	if !rowExists(t, `
		SELECT COUNT(*) FROM volunteer_shifts
		WHERE volunteer_id = $1 AND shift_id = $2 AND cancelled_at IS NOT NULL
	`, volID, shiftID) {
		t.Error("expected cancelled_at to be set in volunteer_shifts after cancelOwnShift")
	}

	// When email succeeds (no GQL errors), also verify the success flag.
	if !hasGQLErrors(resp) {
		var result mutationResult
		unmarshalField(t, resp, "cancelOwnShift", &result)
		if !result.Success {
			t.Errorf("expected success=true when no GQL errors, got false (message: %v)", result.Message)
		}
	}
}

// TestOwnShifts verifies that a volunteer's assigned shifts appear in the
// ownShifts(ALL) query result.
func TestOwnShifts(t *testing.T) {
	token, volID := makeVolunteer(t)
	_, shiftID := seedEventWithShift(t, 2)
	seedVolunteerShift(t, shiftID, volID)

	resp := gqlPost(t, "/graphql/volunteer", token, qryOwnShifts, map[string]any{
		"filter": "ALL",
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	var shifts []ownShiftResult
	unmarshalField(t, resp, "ownShifts", &shifts)

	if len(shifts) == 0 {
		t.Fatal("expected at least one shift in ownShifts(ALL), got none")
	}

	expectedID := fmt.Sprintf("%d", shiftID)
	found := false
	for _, s := range shifts {
		if s.ShiftId == expectedID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("seeded shiftId %q not found in ownShifts results", expectedID)
	}
}

// TestShiftsForEvent verifies that shiftsForEvent returns the shifts seeded
// for a given event, including the correct assignedVolunteers count.
func TestShiftsForEvent(t *testing.T) {
	token, _ := makeVolunteer(t)
	eventID, shiftID := seedEventWithShift(t, 3)

	resp := gqlPost(t, "/graphql/volunteer", token, qryShiftsForEvent, map[string]any{
		"eventId": fmt.Sprintf("%d", eventID),
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	var shifts []shiftViewResult
	unmarshalField(t, resp, "shiftsForEvent", &shifts)

	if len(shifts) == 0 {
		t.Fatal("expected at least one shift in shiftsForEvent, got none")
	}

	expectedID := fmt.Sprintf("%d", shiftID)
	found := false
	for _, s := range shifts {
		if s.ID == expectedID {
			found = true
			if s.AssignedVolunteers != 0 {
				t.Errorf("expected assignedVolunteers=0 for unseeded shift, got %d", s.AssignedVolunteers)
			}
			break
		}
	}
	if !found {
		t.Errorf("seeded shiftId %q not found in shiftsForEvent results", expectedID)
	}
}
