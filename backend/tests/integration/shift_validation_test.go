package integration

// ============================================================================
// Integration tests — shift start/end validation
// ============================================================================
//
// Tests:
//   createShift
//     - End before start → GQL error
//     - End equals start → GQL error
//     - Valid dates      → success, row persisted
//   updateShift
//     - End before start → GQL error, DB row unchanged
//     - End equals start → GQL error
//     - Valid dates      → success

import (
	"fmt"
	"testing"
)

// ============================================================================
// createShift — end before start
// ============================================================================

func TestCreateShift_EndBeforeStart(t *testing.T) {
	token := makeAdminToken(t)
	jobID := getJobTypeID(t, "event_support")
	eventID := seedEvent(t, "Shift Validation EndBefore", true, nil)
	oppID := seedOpportunity(t, eventID, jobID, true)

	resp := gqlPost(t, "/graphql/admin", token, mutCreateShift, map[string]any{
		"input": map[string]any{
			"opportunityId": fmt.Sprintf("%d", oppID),
			"startDateTime": "2028-10-01 14:00:00",
			"endDateTime":   "2028-10-01 09:00:00", // before start
			"maxVolunteers": 5,
		},
	})

	if !hasGQLErrors(resp) {
		t.Error("expected GQL error when shift end is before start, got none")
	}
}

// ============================================================================
// createShift — end equals start
// ============================================================================

func TestCreateShift_EndEqualsStart(t *testing.T) {
	token := makeAdminToken(t)
	jobID := getJobTypeID(t, "event_support")
	eventID := seedEvent(t, "Shift Validation EndEquals", true, nil)
	oppID := seedOpportunity(t, eventID, jobID, true)

	resp := gqlPost(t, "/graphql/admin", token, mutCreateShift, map[string]any{
		"input": map[string]any{
			"opportunityId": fmt.Sprintf("%d", oppID),
			"startDateTime": "2028-10-02 10:00:00",
			"endDateTime":   "2028-10-02 10:00:00", // same as start
			"maxVolunteers": 5,
		},
	})

	if !hasGQLErrors(resp) {
		t.Error("expected GQL error when shift end equals start, got none")
	}
}

// ============================================================================
// createShift — valid dates
// ============================================================================

func TestCreateShift_ValidDates(t *testing.T) {
	token := makeAdminToken(t)
	jobID := getJobTypeID(t, "event_support")
	eventID := seedEvent(t, "Shift Validation Valid Create", true, nil)
	oppID := seedOpportunity(t, eventID, jobID, true)

	resp := gqlPost(t, "/graphql/admin", token, mutCreateShift, map[string]any{
		"input": map[string]any{
			"opportunityId": fmt.Sprintf("%d", oppID),
			"startDateTime": "2028-10-03 09:00:00",
			"endDateTime":   "2028-10-03 17:00:00",
			"maxVolunteers": 5,
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "createShift", &result)

	if !result.Success {
		msg := ""
		if result.Message != nil {
			msg = *result.Message
		}
		t.Fatalf("createShift returned success=false: %s", msg)
	}
	if result.ID == nil || *result.ID == "" {
		t.Fatal("createShift returned no ID on success")
	}

	t.Cleanup(func() { testDB.Exec("DELETE FROM shifts WHERE shift_id = $1", *result.ID) })

	if !rowExists(t, "SELECT COUNT(*) FROM shifts WHERE shift_id = $1", *result.ID) {
		t.Errorf("expected shift row with id=%s in DB after createShift", *result.ID)
	}
}

// ============================================================================
// updateShift — end before start
// ============================================================================

func TestUpdateShift_EndBeforeStart(t *testing.T) {
	token := makeAdminToken(t)
	jobID := getJobTypeID(t, "event_support")
	eventID := seedEvent(t, "Shift Validation Update EndBefore", true, nil)
	oppID := seedOpportunity(t, eventID, jobID, true)
	shiftID := seedShift(t, oppID, "2028-11-01T09:00:00Z", "2028-11-01T17:00:00Z", 5)

	resp := gqlPost(t, "/graphql/admin", token, mutUpdateShift, map[string]any{
		"input": map[string]any{
			"id":            fmt.Sprintf("%d", shiftID),
			"startDateTime": "2028-11-01 14:00:00",
			"endDateTime":   "2028-11-01 09:00:00", // before start
			"maxVolunteers": 5,
		},
	})

	if !hasGQLErrors(resp) {
		t.Error("expected GQL error when shift end is before start, got none")
	}

	// Verify the DB row was not modified — end must still be after start.
	var start, end string
	err := testDB.QueryRow(
		"SELECT shift_start::text, shift_end::text FROM shifts WHERE shift_id = $1",
		shiftID,
	).Scan(&start, &end)
	if err != nil {
		t.Fatalf("could not query shift row %d: %v", shiftID, err)
	}
	if end <= start {
		t.Errorf("DB row was modified by a rejected update: start=%s end=%s", start, end)
	}
}

// ============================================================================
// updateShift — end equals start
// ============================================================================

func TestUpdateShift_EndEqualsStart(t *testing.T) {
	token := makeAdminToken(t)
	jobID := getJobTypeID(t, "event_support")
	eventID := seedEvent(t, "Shift Validation Update EndEquals", true, nil)
	oppID := seedOpportunity(t, eventID, jobID, true)
	shiftID := seedShift(t, oppID, "2028-11-02T09:00:00Z", "2028-11-02T17:00:00Z", 5)

	resp := gqlPost(t, "/graphql/admin", token, mutUpdateShift, map[string]any{
		"input": map[string]any{
			"id":            fmt.Sprintf("%d", shiftID),
			"startDateTime": "2028-11-02 10:00:00",
			"endDateTime":   "2028-11-02 10:00:00", // same as start
			"maxVolunteers": 5,
		},
	})

	if !hasGQLErrors(resp) {
		t.Error("expected GQL error when shift end equals start, got none")
	}
}

// ============================================================================
// updateShift — valid dates
// ============================================================================

func TestUpdateShift_ValidDates(t *testing.T) {
	token := makeAdminToken(t)
	jobID := getJobTypeID(t, "event_support")
	eventID := seedEvent(t, "Shift Validation Update Valid", true, nil)
	oppID := seedOpportunity(t, eventID, jobID, true)
	shiftID := seedShift(t, oppID, "2028-11-03T09:00:00Z", "2028-11-03T17:00:00Z", 5)

	resp := gqlPost(t, "/graphql/admin", token, mutUpdateShift, map[string]any{
		"input": map[string]any{
			"id":            fmt.Sprintf("%d", shiftID),
			"startDateTime": "2028-11-03 10:00:00",
			"endDateTime":   "2028-11-03 18:00:00",
			"maxVolunteers": 8,
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "updateShift", &result)

	if !result.Success {
		msg := ""
		if result.Message != nil {
			msg = *result.Message
		}
		t.Fatalf("updateShift returned success=false: %s", msg)
	}
}
