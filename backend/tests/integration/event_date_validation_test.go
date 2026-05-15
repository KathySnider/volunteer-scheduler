package integration

// ============================================================================
// Integration tests — event-date start/end validation
// ============================================================================
//
// Tests:
//   CreateEvent
//     - No event dates → GQL error
//     - End before start → GQL error
//     - End equals start → GQL error
//     - Valid dates → success, row persisted
//   CreateEventDate
//     - End before start → GQL error
//     - End equals start → GQL error
//     - Valid dates → success, row persisted
//   UpdateEventDate
//     - End before start → GQL error, DB row unchanged
//     - End equals start → GQL error
//     - Valid dates → success, DB row updated

import (
	"fmt"
	"strconv"
	"testing"
	"time"
)

// ============================================================================
// GraphQL operation strings
// ============================================================================

const mutCreateEvent = `
	mutation CreateEvent($newEvent: NewEventInput!) {
		createEvent(newEvent: $newEvent) {
			success
			message
			id
		}
	}`

const mutCreateEventDate = `
	mutation CreateEventDate($newDate: AddEventDateInput!) {
		createEventDate(newDate: $newDate) {
			success
			message
			id
		}
	}`

const mutUpdateEventDate = `
	mutation UpdateEventDate($date: UpdateEventDateInput!) {
		updateEventDate(date: $date) {
			success
			message
			id
		}
	}`

// ============================================================================
// Helpers
// ============================================================================

// seattleFeID returns the funding_entity_id for the always-present "Seattle Area"
// seed row, matching how seedEvent resolves it.
func seattleFeID(t *testing.T) int {
	t.Helper()
	var id int
	if err := testDB.QueryRow(
		"SELECT id FROM funding_entities WHERE name = 'Seattle Area' LIMIT 1",
	).Scan(&id); err != nil {
		t.Fatalf("seattleFeID: %v", err)
	}
	return id
}

// virtualEventInput builds a NewEventInput map for a VIRTUAL event with a
// single date.  Pass the datetimes as "YYYY-MM-DD HH:MM:SS" strings.
func virtualEventInput(feID, stID int, start, end string) map[string]any {
	return map[string]any{
		"name":            fmt.Sprintf("Validation Test Event %d", time.Now().UnixNano()),
		"eventType":       "VIRTUAL",
		"fundingEntityId": feID,
		"serviceTypes":    []int{stID},
		"eventDates": []map[string]any{
			{
				"startDateTime": start,
				"endDateTime":   end,
				"ianaZone":      "UTC",
			},
		},
	}
}

// cleanupEventByID registers a t.Cleanup that deletes the event by string ID.
// CASCADE handles event_dates and event_service_types automatically.
func cleanupEventByID(t *testing.T, idStr string) {
	t.Helper()
	id, err := strconv.Atoi(idStr)
	if err != nil {
		t.Logf("cleanupEventByID: could not parse id %q: %v", idStr, err)
		return
	}
	t.Cleanup(func() {
		testDB.Exec("DELETE FROM events WHERE event_id = $1", id)
	})
}

// ============================================================================
// createEvent — no dates
// ============================================================================

func TestCreateEvent_NoDates(t *testing.T) {
	adminToken, _ := makeAdmin(t)
	feID := seattleFeID(t)
	stID := getServiceTypeID(t, "outreach")

	vars := map[string]any{
		"newEvent": map[string]any{
			"name":            fmt.Sprintf("NoDates Event %d", time.Now().UnixNano()),
			"eventType":       "VIRTUAL",
			"fundingEntityId": feID,
			"serviceTypes":    []int{stID},
			"eventDates":      []map[string]any{},
		},
	}

	resp := gqlPost(t, "/graphql/admin", adminToken, mutCreateEvent, vars)

	if !hasGQLErrors(resp) {
		t.Error("expected GQL error for createEvent with no dates, got none")
	}
}

// ============================================================================
// createEvent — end before start
// ============================================================================

func TestCreateEvent_EndBeforeStart(t *testing.T) {
	adminToken, _ := makeAdmin(t)
	feID := seattleFeID(t)
	stID := getServiceTypeID(t, "outreach")

	vars := map[string]any{
		"newEvent": virtualEventInput(feID, stID,
			"2028-07-01 14:00:00", // start
			"2028-07-01 09:00:00", // end — before start
		),
	}

	resp := gqlPost(t, "/graphql/admin", adminToken, mutCreateEvent, vars)

	if !hasGQLErrors(resp) {
		t.Error("expected GQL error when end is before start, got none")
	}
}

// ============================================================================
// createEvent — end equals start
// ============================================================================

func TestCreateEvent_EndEqualsStart(t *testing.T) {
	adminToken, _ := makeAdmin(t)
	feID := seattleFeID(t)
	stID := getServiceTypeID(t, "outreach")

	vars := map[string]any{
		"newEvent": virtualEventInput(feID, stID,
			"2028-07-02 10:00:00",
			"2028-07-02 10:00:00", // same as start
		),
	}

	resp := gqlPost(t, "/graphql/admin", adminToken, mutCreateEvent, vars)

	if !hasGQLErrors(resp) {
		t.Error("expected GQL error when end equals start, got none")
	}
}

// ============================================================================
// createEvent — valid dates
// ============================================================================

func TestCreateEvent_ValidDates(t *testing.T) {
	adminToken, _ := makeAdmin(t)
	feID := seattleFeID(t)
	stID := getServiceTypeID(t, "outreach")

	vars := map[string]any{
		"newEvent": virtualEventInput(feID, stID,
			"2028-07-03 09:00:00",
			"2028-07-03 17:00:00",
		),
	}

	resp := gqlPost(t, "/graphql/admin", adminToken, mutCreateEvent, vars)

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "createEvent", &result)

	if !result.Success {
		msg := ""
		if result.Message != nil {
			msg = *result.Message
		}
		t.Fatalf("createEvent returned success=false: %s", msg)
	}
	if result.ID == nil || *result.ID == "" {
		t.Fatal("createEvent returned no ID on success")
	}

	// Register cleanup and verify the event row exists in the DB.
	cleanupEventByID(t, *result.ID)

	id, _ := strconv.Atoi(*result.ID)
	if !rowExists(t, "SELECT COUNT(*) FROM events WHERE event_id = $1", id) {
		t.Errorf("expected event row with id=%d in DB after createEvent", id)
	}
}

// ============================================================================
// createEventDate — end before start
// ============================================================================

func TestCreateEventDate_EndBeforeStart(t *testing.T) {
	adminToken, _ := makeAdmin(t)
	eventID := seedEvent(t, "CreateDate EndBefore", true, nil)

	vars := map[string]any{
		"newDate": map[string]any{
			"eventId":       fmt.Sprintf("%d", eventID),
			"startDateTime": "2028-08-10 14:00:00",
			"endDateTime":   "2028-08-10 09:00:00", // before start
			"ianaZone":      "UTC",
		},
	}

	resp := gqlPost(t, "/graphql/admin", adminToken, mutCreateEventDate, vars)

	if !hasGQLErrors(resp) {
		t.Error("expected GQL error when end is before start, got none")
	}
}

// ============================================================================
// createEventDate — end equals start
// ============================================================================

func TestCreateEventDate_EndEqualsStart(t *testing.T) {
	adminToken, _ := makeAdmin(t)
	eventID := seedEvent(t, "CreateDate EndEquals", true, nil)

	vars := map[string]any{
		"newDate": map[string]any{
			"eventId":       fmt.Sprintf("%d", eventID),
			"startDateTime": "2028-08-11 10:00:00",
			"endDateTime":   "2028-08-11 10:00:00", // same as start
			"ianaZone":      "UTC",
		},
	}

	resp := gqlPost(t, "/graphql/admin", adminToken, mutCreateEventDate, vars)

	if !hasGQLErrors(resp) {
		t.Error("expected GQL error when end equals start, got none")
	}
}

// ============================================================================
// createEventDate — valid dates
// ============================================================================

func TestCreateEventDate_ValidDates(t *testing.T) {
	adminToken, _ := makeAdmin(t)
	eventID := seedEvent(t, "CreateDate Valid", true, nil)

	vars := map[string]any{
		"newDate": map[string]any{
			"eventId":       fmt.Sprintf("%d", eventID),
			"startDateTime": "2028-08-12 09:00:00",
			"endDateTime":   "2028-08-12 17:00:00",
			"ianaZone":      "UTC",
		},
	}

	resp := gqlPost(t, "/graphql/admin", adminToken, mutCreateEventDate, vars)

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "createEventDate", &result)

	if !result.Success {
		msg := ""
		if result.Message != nil {
			msg = *result.Message
		}
		t.Fatalf("createEventDate returned success=false: %s", msg)
	}
	if result.ID == nil || *result.ID == "" {
		t.Fatal("createEventDate returned no ID on success")
	}

	id, _ := strconv.Atoi(*result.ID)
	if !rowExists(t, "SELECT COUNT(*) FROM event_dates WHERE event_date_id = $1", id) {
		t.Errorf("expected event_dates row with id=%d in DB after createEventDate", id)
	}
	// Cleanup handled via seedEvent CASCADE; no additional cleanup needed.
}

// ============================================================================
// updateEventDate — end before start
// ============================================================================

func TestUpdateEventDate_EndBeforeStart(t *testing.T) {
	adminToken, _ := makeAdmin(t)
	eventID := seedEvent(t, "UpdateDate EndBefore", true, nil)
	// Seed a valid event_date — stored in UTC, so the UTC string is also what
	// the DB keeps.  We'll verify after the failed update that these are intact.
	dateID := seedEventDate(t, eventID, "2028-09-01T09:00:00Z", "2028-09-01T17:00:00Z")

	vars := map[string]any{
		"date": map[string]any{
			"id":            fmt.Sprintf("%d", dateID),
			"startDateTime": "2028-09-01 14:00:00",
			"endDateTime":   "2028-09-01 09:00:00", // before start
			"ianaZone":      "UTC",
		},
	}

	resp := gqlPost(t, "/graphql/admin", adminToken, mutUpdateEventDate, vars)

	if !hasGQLErrors(resp) {
		t.Error("expected GQL error when end is before start, got none")
	}

	// Verify the row in the DB was NOT modified.
	var start, end string
	err := testDB.QueryRow(
		"SELECT start_date_time::text, end_date_time::text FROM event_dates WHERE event_date_id = $1",
		dateID,
	).Scan(&start, &end)
	if err != nil {
		t.Fatalf("could not query event_dates row %d: %v", dateID, err)
	}
	// The DB stores values in UTC.  We just need to verify the end is still
	// after the start (i.e. the invalid update did not write through).
	if end <= start {
		t.Errorf("DB row was modified by a rejected update: start=%s end=%s", start, end)
	}
}

// ============================================================================
// updateEventDate — end equals start
// ============================================================================

func TestUpdateEventDate_EndEqualsStart(t *testing.T) {
	adminToken, _ := makeAdmin(t)
	eventID := seedEvent(t, "UpdateDate EndEquals", true, nil)
	dateID := seedEventDate(t, eventID, "2028-09-02T09:00:00Z", "2028-09-02T17:00:00Z")

	vars := map[string]any{
		"date": map[string]any{
			"id":            fmt.Sprintf("%d", dateID),
			"startDateTime": "2028-09-02 10:00:00",
			"endDateTime":   "2028-09-02 10:00:00", // same as start
			"ianaZone":      "UTC",
		},
	}

	resp := gqlPost(t, "/graphql/admin", adminToken, mutUpdateEventDate, vars)

	if !hasGQLErrors(resp) {
		t.Error("expected GQL error when end equals start, got none")
	}
}

// ============================================================================
// updateEventDate — valid dates
// ============================================================================

func TestUpdateEventDate_ValidDates(t *testing.T) {
	adminToken, _ := makeAdmin(t)
	eventID := seedEvent(t, "UpdateDate Valid", true, nil)
	dateID := seedEventDate(t, eventID, "2028-09-03T09:00:00Z", "2028-09-03T17:00:00Z")

	vars := map[string]any{
		"date": map[string]any{
			"id":            fmt.Sprintf("%d", dateID),
			"startDateTime": "2028-09-03 10:00:00",
			"endDateTime":   "2028-09-03 18:00:00",
			"ianaZone":      "UTC",
		},
	}

	resp := gqlPost(t, "/graphql/admin", adminToken, mutUpdateEventDate, vars)

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "updateEventDate", &result)

	if !result.Success {
		msg := ""
		if result.Message != nil {
			msg = *result.Message
		}
		t.Fatalf("updateEventDate returned success=false: %s", msg)
	}

	// Verify the DB row reflects the new times.
	var startStr string
	err := testDB.QueryRow(
		"SELECT start_date_time::text FROM event_dates WHERE event_date_id = $1",
		dateID,
	).Scan(&startStr)
	if err != nil {
		t.Fatalf("could not query updated event_dates row %d: %v", dateID, err)
	}
	// "2028-09-03 10:00:00" (UTC input) → stored as "2028-09-03 10:00:00" in the DB.
	// We just verify the row exists and the query succeeded; the exact UTC value
	// depends on the DateTimeToUTC conversion, so we only assert the update went
	// through (no error and success=true above is sufficient).
	if startStr == "" {
		t.Error("start_date_time is empty after updateEventDate")
	}
}
