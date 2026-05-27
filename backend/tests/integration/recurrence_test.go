package integration

// ============================================================================
// Integration tests — recurring event creation
// ============================================================================
//
// Tests:
//   - Weekly recurring event creates the correct number of DB rows
//   - All instances share the same recurrence_group_id
//   - recurrence_order values are 1..N  (sequential, no gaps)
//   - Service types are copied to every instance
//   - Virtual event (no venue) does not panic
//   - YEARLY pattern without maxOccurrences returns a GraphQL error
//   - The returned ID on success is the integer event_id of occurrence #1 (not a UUID)

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"
)

// ============================================================================
// GraphQL operations
// ============================================================================

const mutCreateRecurringEvent = `
	mutation CreateEvent($newEvent: NewEventInput!) {
		createEvent(newEvent: $newEvent) {
			success
			message
			id
		}
	}`

// ============================================================================
// Helpers
// ============================================================================

// recurringEventCount returns the number of events in the DB that share the
// given recurrence_group_id (UUID string).
func recurringEventCount(t *testing.T, groupID string) int {
	t.Helper()
	var count int
	err := testDB.QueryRow(
		"SELECT COUNT(*) FROM events WHERE recurrence_group_id = $1::uuid",
		groupID,
	).Scan(&count)
	if err != nil {
		t.Fatalf("recurringEventCount: %v", err)
	}
	return count
}

// recurringEventOrders returns the sorted recurrence_order values for a group.
func recurringEventOrders(t *testing.T, groupID string) []int {
	t.Helper()
	rows, err := testDB.Query(
		"SELECT recurrence_order FROM events WHERE recurrence_group_id = $1::uuid ORDER BY recurrence_order",
		groupID,
	)
	if err != nil {
		t.Fatalf("recurringEventOrders query: %v", err)
	}
	defer rows.Close()
	var orders []int
	for rows.Next() {
		var o int
		if err := rows.Scan(&o); err != nil {
			t.Fatalf("recurringEventOrders scan: %v", err)
		}
		orders = append(orders, o)
	}
	return orders
}

// serviceTypeCountForGroup returns the number of event_service_types rows
// associated with events in the given recurrence group.
func serviceTypeCountForGroup(t *testing.T, groupID string) int {
	t.Helper()
	var count int
	err := testDB.QueryRow(`
		SELECT COUNT(*)
		FROM event_service_types est
		JOIN events e ON e.event_id = est.event_id
		WHERE e.recurrence_group_id = $1::uuid
	`, groupID).Scan(&count)
	if err != nil {
		t.Fatalf("serviceTypeCountForGroup: %v", err)
	}
	return count
}

// cleanupRecurrenceGroup registers a t.Cleanup that deletes all events (and
// their cascaded children) belonging to the given recurrence group, then
// removes the recurrence_groups row itself.
func cleanupRecurrenceGroup(t *testing.T, groupID string) {
	t.Helper()
	t.Cleanup(func() {
		testDB.Exec("DELETE FROM events WHERE recurrence_group_id = $1::uuid", groupID)
		testDB.Exec("DELETE FROM recurrence_groups WHERE id = $1::uuid", groupID)
	})
}

// isUUID returns true if s looks like a UUID (36 chars with hyphens).
func isUUID(s string) bool {
	return len(s) == 36 && s[8] == '-' && s[13] == '-' && s[18] == '-' && s[23] == '-'
}

// groupIDFromEventID resolves the recurrence_group_id UUID for the given
// integer event_id (returned by createRecurringEvent after the API change
// to return the first-occurrence event ID instead of the group UUID).
func groupIDFromEventID(t *testing.T, eventID string) string {
	t.Helper()
	var groupID string
	err := testDB.QueryRow(
		"SELECT recurrence_group_id FROM events WHERE event_id = $1",
		eventID,
	).Scan(&groupID)
	if err != nil {
		t.Fatalf("groupIDFromEventID(%q): %v", eventID, err)
	}
	return groupID
}

// weeklyVirtualInput builds a NewEventInput map for a WEEKLY virtual recurring
// event. start/end are in "YYYY-MM-DD HH:MM:SS" format.
func weeklyVirtualInput(feID, stID, occurrences int, start, end string) map[string]any {
	return map[string]any{
		"name":            fmt.Sprintf("Weekly Recur %d", time.Now().UnixNano()),
		"eventType":       "VIRTUAL",
		"fundingEntityId": feID,
		"timezone":        "America/Los_Angeles",
		"serviceTypes":    []int{stID},
		"eventDates": []map[string]any{
			{"startDateTime": start, "endDateTime": end},
		},
		"recurrence": map[string]any{
			"pattern":        "WEEKLY",
			"maxOccurrences": occurrences,
		},
	}
}

// ============================================================================
// Tests
// ============================================================================

func TestCreateRecurringEvent_Weekly_CorrectCount(t *testing.T) {
	adminToken, _ := makeAdmin(t)
	feID := seattleFeID(t)
	stID := getServiceTypeID(t, "outreach")

	const wantInstances = 4
	vars := map[string]any{
		"newEvent": weeklyVirtualInput(feID, stID, wantInstances,
			"2030-06-04 08:00:00", "2030-06-04 10:00:00"),
	}

	resp := gqlPost(t, "/graphql/admin", adminToken, mutCreateRecurringEvent, vars)
	if hasGQLErrors(resp) {
		t.Fatalf("createEvent errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "createEvent", &result)
	if !result.Success {
		msg := ""
		if result.Message != nil {
			msg = *result.Message
		}
		t.Fatalf("createEvent success=false: %s", msg)
	}
	if result.ID == nil {
		t.Fatal("createEvent returned nil ID")
	}

	groupID := groupIDFromEventID(t, *result.ID)
	cleanupRecurrenceGroup(t, groupID)

	if got := recurringEventCount(t, groupID); got != wantInstances {
		t.Errorf("want %d events in group, got %d", wantInstances, got)
	}
}

func TestCreateRecurringEvent_Weekly_SharedGroupID(t *testing.T) {
	adminToken, _ := makeAdmin(t)
	feID := seattleFeID(t)
	stID := getServiceTypeID(t, "outreach")

	vars := map[string]any{
		"newEvent": weeklyVirtualInput(feID, stID, 3,
			"2030-07-02 08:00:00", "2030-07-02 10:00:00"),
	}

	resp := gqlPost(t, "/graphql/admin", adminToken, mutCreateRecurringEvent, vars)
	if hasGQLErrors(resp) {
		t.Fatalf("createEvent errors: %v", resp.Errors)
	}
	var result mutationResult
	unmarshalField(t, resp, "createEvent", &result)
	if result.ID == nil {
		t.Fatal("no group ID returned")
	}

	groupID := groupIDFromEventID(t, *result.ID)
	cleanupRecurrenceGroup(t, groupID)

	// Every event in the group should have recurrence_group_id = groupID.
	var count int
	err := testDB.QueryRow(`
		SELECT COUNT(*)
		FROM events
		WHERE recurrence_group_id = $1::uuid
		  AND recurrence_group_id IS NOT NULL
	`, groupID).Scan(&count)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 3 {
		t.Errorf("want 3 events with group_id set, got %d", count)
	}
}

func TestCreateRecurringEvent_Weekly_SequentialOrder(t *testing.T) {
	adminToken, _ := makeAdmin(t)
	feID := seattleFeID(t)
	stID := getServiceTypeID(t, "outreach")

	const n = 4
	vars := map[string]any{
		"newEvent": weeklyVirtualInput(feID, stID, n,
			"2030-08-06 09:00:00", "2030-08-06 11:00:00"),
	}

	resp := gqlPost(t, "/graphql/admin", adminToken, mutCreateRecurringEvent, vars)
	if hasGQLErrors(resp) {
		t.Fatalf("createEvent errors: %v", resp.Errors)
	}
	var result mutationResult
	unmarshalField(t, resp, "createEvent", &result)
	if result.ID == nil {
		t.Fatal("no group ID returned")
	}

	groupID := groupIDFromEventID(t, *result.ID)
	cleanupRecurrenceGroup(t, groupID)

	orders := recurringEventOrders(t, groupID)
	if len(orders) != n {
		t.Fatalf("want %d orders, got %d", n, len(orders))
	}
	for i, o := range orders {
		if o != i+1 {
			t.Errorf("orders[%d]: want %d, got %d", i, i+1, o)
		}
	}
}

func TestCreateRecurringEvent_ServiceTypesCopied(t *testing.T) {
	adminToken, _ := makeAdmin(t)
	feID := seattleFeID(t)
	stID := getServiceTypeID(t, "outreach")

	const n = 3
	vars := map[string]any{
		"newEvent": weeklyVirtualInput(feID, stID, n,
			"2030-09-03 08:00:00", "2030-09-03 10:00:00"),
	}

	resp := gqlPost(t, "/graphql/admin", adminToken, mutCreateRecurringEvent, vars)
	if hasGQLErrors(resp) {
		t.Fatalf("createEvent errors: %v", resp.Errors)
	}
	var result mutationResult
	unmarshalField(t, resp, "createEvent", &result)
	if result.ID == nil {
		t.Fatal("no group ID returned")
	}

	groupID := groupIDFromEventID(t, *result.ID)
	cleanupRecurrenceGroup(t, groupID)

	// Each instance should have exactly 1 service_type row (we passed 1 stID).
	got := serviceTypeCountForGroup(t, groupID)
	if got != n {
		t.Errorf("want %d service_type rows (1 per instance), got %d", n, got)
	}
}

func TestCreateRecurringEvent_Virtual_Succeeds(t *testing.T) {
	// Virtual events have no venue — verify no nil-pointer panic.
	adminToken, _ := makeAdmin(t)
	feID := seattleFeID(t)
	stID := getServiceTypeID(t, "outreach")

	vars := map[string]any{
		"newEvent": weeklyVirtualInput(feID, stID, 2,
			"2030-10-01 08:00:00", "2030-10-01 10:00:00"),
	}

	resp := gqlPost(t, "/graphql/admin", adminToken, mutCreateRecurringEvent, vars)
	if hasGQLErrors(resp) {
		t.Fatalf("virtual recurring event failed: %v", resp.Errors)
	}
	var result mutationResult
	unmarshalField(t, resp, "createEvent", &result)
	if !result.Success {
		t.Fatalf("expected success for virtual recurring event")
	}
	if result.ID != nil {
		cleanupRecurrenceGroup(t, groupIDFromEventID(t, *result.ID))
	}
}

func TestCreateRecurringEvent_Yearly_NoMax_Fails(t *testing.T) {
	adminToken, _ := makeAdmin(t)
	feID := seattleFeID(t)
	stID := getServiceTypeID(t, "outreach")

	vars := map[string]any{
		"newEvent": map[string]any{
			"name":            fmt.Sprintf("Yearly No Max %d", time.Now().UnixNano()),
			"eventType":       "VIRTUAL",
			"fundingEntityId": feID,
			"timezone":        "America/Los_Angeles",
			"serviceTypes":    []int{stID},
			"eventDates": []map[string]any{
				{"startDateTime": "2030-01-07 08:00:00", "endDateTime": "2030-01-07 10:00:00"},
			},
			"recurrence": map[string]any{
				"pattern": "YEARLY",
				// maxOccurrences intentionally omitted
			},
		},
	}

	resp := gqlPost(t, "/graphql/admin", adminToken, mutCreateRecurringEvent, vars)
	if !hasGQLErrors(resp) {
		t.Error("expected GQL error for YEARLY with no maxOccurrences, got none")
	}
}

func TestCreateRecurringEvent_ReturnedID_IsFirstOccurrenceEventID(t *testing.T) {
	adminToken, _ := makeAdmin(t)
	feID := seattleFeID(t)
	stID := getServiceTypeID(t, "outreach")

	vars := map[string]any{
		"newEvent": weeklyVirtualInput(feID, stID, 2,
			"2030-11-05 08:00:00", "2030-11-05 10:00:00"),
	}

	resp := gqlPost(t, "/graphql/admin", adminToken, mutCreateRecurringEvent, vars)
	if hasGQLErrors(resp) {
		t.Fatalf("createEvent errors: %v", resp.Errors)
	}
	var result mutationResult
	unmarshalField(t, resp, "createEvent", &result)
	if result.ID == nil {
		t.Fatal("no ID returned")
	}

	groupID := groupIDFromEventID(t, *result.ID)
	cleanupRecurrenceGroup(t, groupID)

	// The returned ID must be parseable as an integer (it is the event_id of the
	// first occurrence, not a UUID).
	if _, err := strconv.Atoi(*result.ID); err != nil {
		t.Errorf("expected integer event_id for recurring event ID, got %q", *result.ID)
	}

	// That event must have recurrence_order = 1 in the DB.
	var order int
	if err := testDB.QueryRow(
		"SELECT recurrence_order FROM events WHERE event_id = $1",
		*result.ID,
	).Scan(&order); err != nil {
		t.Fatalf("could not look up recurrence_order: %v", err)
	}
	if order != 1 {
		t.Errorf("expected recurrence_order=1 for first-occurrence event, got %d", order)
	}
}

func TestCreateNonRecurringEvent_ReturnedID_IsInteger(t *testing.T) {
	// Sanity check: single (non-recurring) events still return a plain integer ID.
	adminToken, _ := makeAdmin(t)
	feID := seattleFeID(t)
	stID := getServiceTypeID(t, "outreach")

	vars := map[string]any{
		"newEvent": map[string]any{
			"name":            fmt.Sprintf("Single Event %d", time.Now().UnixNano()),
			"eventType":       "VIRTUAL",
			"fundingEntityId": feID,
			"timezone":        "America/Los_Angeles",
			"serviceTypes":    []int{stID},
			"eventDates": []map[string]any{
				{"startDateTime": "2030-12-03 08:00:00", "endDateTime": "2030-12-03 10:00:00"},
			},
			// no recurrence field
		},
	}

	resp := gqlPost(t, "/graphql/admin", adminToken, mutCreateRecurringEvent, vars)
	if hasGQLErrors(resp) {
		t.Fatalf("createEvent errors: %v", resp.Errors)
	}
	var result mutationResult
	unmarshalField(t, resp, "createEvent", &result)
	if result.ID == nil {
		t.Fatal("no ID returned")
	}

	// The ID should be parseable as an integer for single events.
	if _, err := strconv.Atoi(*result.ID); err != nil {
		t.Errorf("single event ID should be an integer, got %q", *result.ID)
	}

	// Cleanup.
	id, _ := strconv.Atoi(*result.ID)
	t.Cleanup(func() { testDB.Exec("DELETE FROM events WHERE event_id = $1", id) })
}

// TestCreateRecurringEvent_RecurrenceGroupSaved verifies that creating a
// recurring event inserts a row into recurrence_groups with the correct
// pattern, max_occurrences, and weekday_ordinal values.
func TestCreateRecurringEvent_RecurrenceGroupSaved(t *testing.T) {
	adminToken, _ := makeAdmin(t)
	feID := seattleFeID(t)
	stID := getServiceTypeID(t, "outreach")

	const wantPattern = "WEEKLY"
	const wantMax = 3

	vars := map[string]any{
		"newEvent": weeklyVirtualInput(feID, stID, wantMax,
			"2031-01-07 09:00:00", "2031-01-07 11:00:00"),
	}

	resp := gqlPost(t, "/graphql/admin", adminToken, mutCreateRecurringEvent, vars)
	if hasGQLErrors(resp) {
		t.Fatalf("createEvent errors: %v", resp.Errors)
	}
	var result mutationResult
	unmarshalField(t, resp, "createEvent", &result)
	if result.ID == nil {
		t.Fatal("createEvent returned nil ID")
	}

	groupID := groupIDFromEventID(t, *result.ID)
	cleanupRecurrenceGroup(t, groupID)

	// Verify the recurrence_groups row was created.
	var gotPattern string
	var gotMax int
	var gotOrdinal *string
	err := testDB.QueryRow(
		"SELECT pattern, max_occurrences, weekday_ordinal FROM recurrence_groups WHERE id = $1::uuid",
		groupID,
	).Scan(&gotPattern, &gotMax, &gotOrdinal)
	if err != nil {
		t.Fatalf("recurrence_groups row not found: %v", err)
	}
	if gotPattern != wantPattern {
		t.Errorf("pattern: want %q, got %q", wantPattern, gotPattern)
	}
	if gotMax != wantMax {
		t.Errorf("max_occurrences: want %d, got %d", wantMax, gotMax)
	}
	if gotOrdinal != nil {
		t.Errorf("weekday_ordinal: want nil for weekly, got %q", *gotOrdinal)
	}
}

// Keep the compiler happy if json is only used in some build tags.
var _ = json.Marshal
