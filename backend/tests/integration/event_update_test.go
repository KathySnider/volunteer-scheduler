package integration

// ============================================================================
// Integration tests — UpdateEvent mutation
// ============================================================================
//
// Tests:
//   - Non-recurring: name and service types are written to the DB
//   - Non-recurring: old service types are replaced (not appended) on update
//   - Recurring / THIS_ONLY: only the target row is changed; siblings unchanged
//   - Recurring / THIS_ONLY: the updated event stays in the recurrence group
//   - Recurring / THIS_AND_FUTURE: rows with order >= n are updated; earlier rows untouched
//   - Recurring / THIS_AND_FUTURE: service types synced only for affected rows
//   - Invalid event ID returns a GraphQL error

import (
	"fmt"
	"strconv"
	"testing"
	"time"
)

// ============================================================================
// GraphQL operation
// ============================================================================

const mutUpdateEvent = `
	mutation UpdateEvent($event: UpdateEventInput!) {
		updateEvent(event: $event) {
			success
			message
			id
		}
	}`

// ============================================================================
// Local helpers
// ============================================================================

// recurringEventIDsByOrder returns the event_id values for a recurrence group,
// sorted by recurrence_order ascending (i.e. [order=1, order=2, ...]).
func recurringEventIDsByOrder(t *testing.T, groupID string) []int {
	t.Helper()
	rows, err := testDB.Query(
		"SELECT event_id FROM events WHERE recurrence_group_id = $1::uuid ORDER BY recurrence_order",
		groupID,
	)
	if err != nil {
		t.Fatalf("recurringEventIDsByOrder: %v", err)
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("recurringEventIDsByOrder scan: %v", err)
		}
		ids = append(ids, id)
	}
	return ids
}

// eventDBName returns the event_name stored in the DB for the given event_id.
func eventDBName(t *testing.T, eventID int) string {
	t.Helper()
	var name string
	if err := testDB.QueryRow(
		"SELECT event_name FROM events WHERE event_id = $1", eventID,
	).Scan(&name); err != nil {
		t.Fatalf("eventDBName(%d): %v", eventID, err)
	}
	return name
}

// hasServiceType returns true if event_service_types contains (eventID, stID).
func hasServiceType(t *testing.T, eventID, stID int) bool {
	t.Helper()
	return rowExists(t,
		"SELECT COUNT(*) FROM event_service_types WHERE event_id = $1 AND service_type_id = $2",
		eventID, stID,
	)
}

// updateEventInput builds a minimal UpdateEventInput variable map for a
// VIRTUAL event update (no venue required).
func updateEventInput(id string, name string, feID int, stIDs []int, scope *string) map[string]any {
	m := map[string]any{
		"id":              id,
		"name":            name,
		"eventType":       "VIRTUAL",
		"timezone":        "America/Los_Angeles",
		"fundingEntityId": feID,
		"serviceTypes":    stIDs,
	}
	if scope != nil {
		m["recurrenceScope"] = *scope
	}
	return m
}

// ptr returns a pointer to the given string — convenience for scope literals.
func strPtr(s string) *string { return &s }

// ============================================================================
// Tests — non-recurring events
// ============================================================================

func TestUpdateEvent_NonRecurring_UpdatesName(t *testing.T) {
	adminToken, _ := makeAdmin(t)
	feID := seattleFeID(t)
	stID := getServiceTypeID(t, "outreach")

	eventID := seedEvent(t, fmt.Sprintf("OrigName-%d", time.Now().UnixNano()), true, nil)
	newName := fmt.Sprintf("UpdatedName-%d", time.Now().UnixNano())

	vars := map[string]any{
		"event": updateEventInput(strconv.Itoa(eventID), newName, feID, []int{stID}, nil),
	}
	resp := gqlPost(t, "/graphql/admin", adminToken, mutUpdateEvent, vars)
	if hasGQLErrors(resp) {
		t.Fatalf("updateEvent returned errors: %v", resp.Errors)
	}
	var result mutationResult
	unmarshalField(t, resp, "updateEvent", &result)
	if !result.Success {
		msg := ""
		if result.Message != nil {
			msg = *result.Message
		}
		t.Fatalf("updateEvent success=false: %s", msg)
	}

	if got := eventDBName(t, eventID); got != newName {
		t.Errorf("event_name: want %q, got %q", newName, got)
	}
}

func TestUpdateEvent_NonRecurring_ServiceTypes_Written(t *testing.T) {
	adminToken, _ := makeAdmin(t)
	feID := seattleFeID(t)
	stID := getServiceTypeID(t, "advocacy")

	eventID := seedEvent(t, fmt.Sprintf("STWriteEvent-%d", time.Now().UnixNano()), true, nil)

	vars := map[string]any{
		"event": updateEventInput(strconv.Itoa(eventID), "Any Name", feID, []int{stID}, nil),
	}
	resp := gqlPost(t, "/graphql/admin", adminToken, mutUpdateEvent, vars)
	if hasGQLErrors(resp) {
		t.Fatalf("updateEvent errors: %v", resp.Errors)
	}
	var result mutationResult
	unmarshalField(t, resp, "updateEvent", &result)
	if !result.Success {
		t.Fatal("updateEvent success=false")
	}

	if !hasServiceType(t, eventID, stID) {
		t.Errorf("expected service_type %d to be present after update", stID)
	}
}

func TestUpdateEvent_NonRecurring_ServiceTypes_Replaced(t *testing.T) {
	adminToken, _ := makeAdmin(t)
	feID := seattleFeID(t)
	stOld := getServiceTypeID(t, "outreach")
	stNew := getServiceTypeID(t, "advocacy")

	eventID := seedEvent(t, fmt.Sprintf("STReplaceEvent-%d", time.Now().UnixNano()), true, nil)

	// Pre-seed the OLD service type directly in the DB.
	if _, err := testDB.Exec(
		"INSERT INTO event_service_types (event_id, service_type_id) VALUES ($1, $2)",
		eventID, stOld,
	); err != nil {
		t.Fatalf("pre-seed service type: %v", err)
	}

	// Now updateEvent with stNew only — stOld must be removed.
	vars := map[string]any{
		"event": updateEventInput(strconv.Itoa(eventID), "Any Name", feID, []int{stNew}, nil),
	}
	resp := gqlPost(t, "/graphql/admin", adminToken, mutUpdateEvent, vars)
	if hasGQLErrors(resp) {
		t.Fatalf("updateEvent errors: %v", resp.Errors)
	}
	var result mutationResult
	unmarshalField(t, resp, "updateEvent", &result)
	if !result.Success {
		t.Fatal("updateEvent success=false")
	}

	if hasServiceType(t, eventID, stOld) {
		t.Errorf("old service_type %d should have been removed but is still present", stOld)
	}
	if !hasServiceType(t, eventID, stNew) {
		t.Errorf("new service_type %d should be present but is missing", stNew)
	}
}

// ============================================================================
// Tests — recurring events, THIS_ONLY scope
// ============================================================================

func TestUpdateEvent_Recurring_ThisOnly_UpdatesSingleRow(t *testing.T) {
	adminToken, _ := makeAdmin(t)
	feID := seattleFeID(t)
	stID := getServiceTypeID(t, "outreach")

	// Create a 3-instance recurring group.
	createVars := map[string]any{
		"newEvent": weeklyVirtualInput(feID, stID, 3, "2032-03-04 09:00:00", "2032-03-04 11:00:00"),
	}
	createResp := gqlPost(t, "/graphql/admin", adminToken, mutCreateRecurringEvent, createVars)
	if hasGQLErrors(createResp) {
		t.Fatalf("createEvent errors: %v", createResp.Errors)
	}
	var createResult mutationResult
	unmarshalField(t, createResp, "createEvent", &createResult)
	if createResult.ID == nil {
		t.Fatal("createEvent returned nil ID")
	}
	groupID := groupIDFromEventID(t, *createResult.ID)
	cleanupRecurrenceGroup(t, groupID)

	ids := recurringEventIDsByOrder(t, groupID) // [order=1, order=2, order=3]
	if len(ids) != 3 {
		t.Fatalf("expected 3 event IDs, got %d", len(ids))
	}

	// Remember names before update.
	origName0 := eventDBName(t, ids[0])
	origName2 := eventDBName(t, ids[2])
	newName := fmt.Sprintf("ThisOnlyUpdate-%d", time.Now().UnixNano())

	// Update only instance[1] (order=2) with THIS_ONLY.
	scope := "THIS_ONLY"
	updateVars := map[string]any{
		"event": updateEventInput(strconv.Itoa(ids[1]), newName, feID, []int{stID}, &scope),
	}
	resp := gqlPost(t, "/graphql/admin", adminToken, mutUpdateEvent, updateVars)
	if hasGQLErrors(resp) {
		t.Fatalf("updateEvent errors: %v", resp.Errors)
	}
	var result mutationResult
	unmarshalField(t, resp, "updateEvent", &result)
	if !result.Success {
		t.Fatal("updateEvent success=false")
	}

	// instance[1] must have the new name.
	if got := eventDBName(t, ids[1]); got != newName {
		t.Errorf("instance[1] name: want %q, got %q", newName, got)
	}
	// Siblings must be unchanged.
	if got := eventDBName(t, ids[0]); got != origName0 {
		t.Errorf("instance[0] name should not have changed: want %q, got %q", origName0, got)
	}
	if got := eventDBName(t, ids[2]); got != origName2 {
		t.Errorf("instance[2] name should not have changed: want %q, got %q", origName2, got)
	}
}

func TestUpdateEvent_Recurring_ThisOnly_EventStaysInGroup(t *testing.T) {
	adminToken, _ := makeAdmin(t)
	feID := seattleFeID(t)
	stID := getServiceTypeID(t, "outreach")

	createVars := map[string]any{
		"newEvent": weeklyVirtualInput(feID, stID, 3, "2032-04-01 09:00:00", "2032-04-01 11:00:00"),
	}
	createResp := gqlPost(t, "/graphql/admin", adminToken, mutCreateRecurringEvent, createVars)
	if hasGQLErrors(createResp) {
		t.Fatalf("createEvent errors: %v", createResp.Errors)
	}
	var createResult mutationResult
	unmarshalField(t, createResp, "createEvent", &createResult)
	if createResult.ID == nil {
		t.Fatal("no group ID returned")
	}
	groupID := groupIDFromEventID(t, *createResult.ID)
	cleanupRecurrenceGroup(t, groupID)

	ids := recurringEventIDsByOrder(t, groupID)
	if len(ids) < 2 {
		t.Fatalf("expected at least 2 IDs, got %d", len(ids))
	}

	// Update instance[1] with THIS_ONLY.
	scope := "THIS_ONLY"
	updateVars := map[string]any{
		"event": updateEventInput(
			strconv.Itoa(ids[1]),
			fmt.Sprintf("StillInGroup-%d", time.Now().UnixNano()),
			feID, []int{stID}, &scope,
		),
	}
	resp := gqlPost(t, "/graphql/admin", adminToken, mutUpdateEvent, updateVars)
	if hasGQLErrors(resp) {
		t.Fatalf("updateEvent errors: %v", resp.Errors)
	}
	var result mutationResult
	unmarshalField(t, resp, "updateEvent", &result)
	if !result.Success {
		t.Fatal("updateEvent success=false")
	}

	// The updated event must still belong to the same recurrence group.
	if !rowExists(t,
		"SELECT COUNT(*) FROM events WHERE event_id = $1 AND recurrence_group_id = $2::uuid",
		ids[1], groupID,
	) {
		t.Errorf("instance[1] should still belong to group %s after THIS_ONLY update", groupID)
	}
}

// ============================================================================
// Tests — recurring events, THIS_AND_FUTURE scope
// ============================================================================

func TestUpdateEvent_Recurring_ThisAndFuture_UpdatesTargetAndLater(t *testing.T) {
	adminToken, _ := makeAdmin(t)
	feID := seattleFeID(t)
	stID := getServiceTypeID(t, "outreach")

	// Create a 4-instance recurring group.
	createVars := map[string]any{
		"newEvent": weeklyVirtualInput(feID, stID, 4, "2032-05-06 09:00:00", "2032-05-06 11:00:00"),
	}
	createResp := gqlPost(t, "/graphql/admin", adminToken, mutCreateRecurringEvent, createVars)
	if hasGQLErrors(createResp) {
		t.Fatalf("createEvent errors: %v", createResp.Errors)
	}
	var createResult mutationResult
	unmarshalField(t, createResp, "createEvent", &createResult)
	if createResult.ID == nil {
		t.Fatal("no group ID returned")
	}
	groupID := groupIDFromEventID(t, *createResult.ID)
	cleanupRecurrenceGroup(t, groupID)

	ids := recurringEventIDsByOrder(t, groupID) // [order=1, order=2, order=3, order=4]
	if len(ids) != 4 {
		t.Fatalf("expected 4 event IDs, got %d", len(ids))
	}

	origName0 := eventDBName(t, ids[0]) // order=1 — must NOT be updated
	newName := fmt.Sprintf("FutureUpdate-%d", time.Now().UnixNano())

	// Update from instance[1] (order=2) forward.
	scope := "THIS_AND_FUTURE"
	updateVars := map[string]any{
		"event": updateEventInput(strconv.Itoa(ids[1]), newName, feID, []int{stID}, &scope),
	}
	resp := gqlPost(t, "/graphql/admin", adminToken, mutUpdateEvent, updateVars)
	if hasGQLErrors(resp) {
		t.Fatalf("updateEvent errors: %v", resp.Errors)
	}
	var result mutationResult
	unmarshalField(t, resp, "updateEvent", &result)
	if !result.Success {
		t.Fatal("updateEvent success=false")
	}

	// instance[0] (order=1) must be untouched.
	if got := eventDBName(t, ids[0]); got != origName0 {
		t.Errorf("instance[0] (order=1) should be unchanged: want %q, got %q", origName0, got)
	}
	// Instances [1], [2], [3] (order=2,3,4) must all have the new name.
	for i := 1; i < 4; i++ {
		if got := eventDBName(t, ids[i]); got != newName {
			t.Errorf("instance[%d] name: want %q, got %q", i, newName, got)
		}
	}
}

func TestUpdateEvent_Recurring_ThisAndFuture_ServiceTypes_Scoped(t *testing.T) {
	adminToken, _ := makeAdmin(t)
	feID := seattleFeID(t)
	stOld := getServiceTypeID(t, "outreach")  // initial ST for all instances
	stNew := getServiceTypeID(t, "advocacy")  // replacement ST for order>=2

	// Create a 3-instance group with stOld.
	createVars := map[string]any{
		"newEvent": weeklyVirtualInput(feID, stOld, 3, "2032-06-03 09:00:00", "2032-06-03 11:00:00"),
	}
	createResp := gqlPost(t, "/graphql/admin", adminToken, mutCreateRecurringEvent, createVars)
	if hasGQLErrors(createResp) {
		t.Fatalf("createEvent errors: %v", createResp.Errors)
	}
	var createResult mutationResult
	unmarshalField(t, createResp, "createEvent", &createResult)
	if createResult.ID == nil {
		t.Fatal("no group ID returned")
	}
	groupID := groupIDFromEventID(t, *createResult.ID)
	cleanupRecurrenceGroup(t, groupID)

	ids := recurringEventIDsByOrder(t, groupID) // [order=1, order=2, order=3]
	if len(ids) != 3 {
		t.Fatalf("expected 3 event IDs, got %d", len(ids))
	}

	// Update from instance[1] (order=2) forward, replacing stOld with stNew.
	scope := "THIS_AND_FUTURE"
	updateVars := map[string]any{
		"event": updateEventInput(
			strconv.Itoa(ids[1]),
			fmt.Sprintf("STScopedUpdate-%d", time.Now().UnixNano()),
			feID, []int{stNew}, &scope,
		),
	}
	resp := gqlPost(t, "/graphql/admin", adminToken, mutUpdateEvent, updateVars)
	if hasGQLErrors(resp) {
		t.Fatalf("updateEvent errors: %v", resp.Errors)
	}
	var result mutationResult
	unmarshalField(t, resp, "updateEvent", &result)
	if !result.Success {
		t.Fatal("updateEvent success=false")
	}

	// instance[0] (order=1) keeps stOld and must not gain stNew.
	if !hasServiceType(t, ids[0], stOld) {
		t.Errorf("instance[0] should still have stOld=%d", stOld)
	}
	if hasServiceType(t, ids[0], stNew) {
		t.Errorf("instance[0] should NOT have stNew=%d", stNew)
	}

	// instance[1] and instance[2] (order=2,3) must have stNew and not stOld.
	for i := 1; i <= 2; i++ {
		if !hasServiceType(t, ids[i], stNew) {
			t.Errorf("instance[%d] should have stNew=%d", i, stNew)
		}
		if hasServiceType(t, ids[i], stOld) {
			t.Errorf("instance[%d] should NOT have stOld=%d after update", i, stOld)
		}
	}
}

// ============================================================================
// Tests — error cases
// ============================================================================

func TestUpdateEvent_InvalidID_ReturnsError(t *testing.T) {
	adminToken, _ := makeAdmin(t)
	feID := seattleFeID(t)
	stID := getServiceTypeID(t, "outreach")

	vars := map[string]any{
		"event": updateEventInput("not-a-number", "Some Name", feID, []int{stID}, nil),
	}
	resp := gqlPost(t, "/graphql/admin", adminToken, mutUpdateEvent, vars)
	if !hasGQLErrors(resp) {
		t.Error("expected a GraphQL error for an invalid event ID, got none")
	}
}
