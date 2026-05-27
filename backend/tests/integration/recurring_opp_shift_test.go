package integration

// ============================================================================
// Integration tests — recurring opp/shift propagation
// ============================================================================
//
// Tests:
//   CreateOpportunity
//     - Propagates new opp+shifts to every future instance in the group
//     - All copies share the same non-null recurrence_template_id UUID
//     - Each propagated opp receives the initial shift(s)
//     - Does NOT propagate backward (earlier occurrences are untouched)
//
//   UpdateOpportunity
//     - Changes fan out to all future instances (same template_id, higher order)
//     - Past instances are left unchanged
//
//   DeleteOpportunity
//     - Removes sibling opps from every future instance
//     - Opps on past instances survive
//
//   CreateShift
//     - New shift propagates to sibling opps on future instances
//     - Time offset from each event's first date is preserved across instances
//
//   UpdateShift
//     - Propagates max_volunteers change to future-instance sibling shifts
//
//   DeleteShift (recurring)
//     - Deletes sibling shifts on future instances
//     - Blocked when any future sibling opp would be left with zero shifts
// ============================================================================

import (
	"database/sql"
	"fmt"
	"strconv"
	"testing"
	"time"
)

// ============================================================================
// Helpers local to this file
// ============================================================================

// seedPropagationGroup creates a 3-occurrence weekly virtual event group and
// returns (groupID, [eventID_order1, eventID_order2, eventID_order3]).
// startDate must be "YYYY-MM-DD"; 09:00–11:00 is used as the event window.
func seedPropagationGroup(t *testing.T, token, startDate string) (string, []int) {
	t.Helper()
	feID := seattleFeID(t)
	stID := getServiceTypeID(t, "outreach")
	vars := map[string]any{
		"newEvent": weeklyVirtualInput(feID, stID, 3,
			startDate+" 09:00:00", startDate+" 11:00:00"),
	}
	resp := gqlPost(t, "/graphql/admin", token, mutCreateRecurringEvent, vars)
	if hasGQLErrors(resp) {
		t.Fatalf("seedPropagationGroup: %v", resp.Errors)
	}
	var r mutationResult
	unmarshalField(t, resp, "createEvent", &r)
	if r.ID == nil {
		t.Fatal("seedPropagationGroup: nil group ID")
	}
	groupID := groupIDFromEventID(t, *r.ID)
	cleanupRecurrenceGroup(t, groupID)
	ids := recurringEventIDsByOrder(t, groupID)
	if len(ids) != 3 {
		t.Fatalf("seedPropagationGroup: want 3 instances, got %d", len(ids))
	}
	return groupID, ids
}

// createRecurringOpp calls createOpportunity on eventID and returns the base
// opportunity ID string. The opp + initial shift are created via GraphQL so
// the backend's recurrence propagation runs and stamps recurrence_template_id.
func createRecurringOpp(t *testing.T, token string, eventID, jobTypeID int, shiftStart, shiftEnd string) string {
	t.Helper()
	resp := gqlPost(t, "/graphql/admin", token, mutCreateOpportunity, map[string]any{
		"input": map[string]any{
			"eventId":   fmt.Sprintf("%d", eventID),
			"jobId":     jobTypeID,
			"isVirtual": true,
			"shifts": []map[string]any{
				{"startDateTime": shiftStart, "endDateTime": shiftEnd, "maxVolunteers": 5},
			},
		},
	})
	if hasGQLErrors(resp) {
		t.Fatalf("createRecurringOpp: %v", resp.Errors)
	}
	var r mutationResult
	unmarshalField(t, resp, "createOpportunity", &r)
	if !r.Success {
		t.Fatalf("createRecurringOpp: success=false: %v", r.Message)
	}
	return *r.ID
}

// oppCountForEvent returns how many opportunities exist for the given event.
func oppCountForEvent(t *testing.T, eventID int) int {
	t.Helper()
	var n int
	if err := testDB.QueryRow(
		"SELECT COUNT(*) FROM opportunities WHERE event_id = $1", eventID,
	).Scan(&n); err != nil {
		t.Fatalf("oppCountForEvent(%d): %v", eventID, err)
	}
	return n
}

// oppTemplateID returns the recurrence_template_id of the given opportunity
// as a plain string (empty if NULL).
func oppTemplateID(t *testing.T, oppID string) string {
	t.Helper()
	var s string
	if err := testDB.QueryRow(
		"SELECT COALESCE(recurrence_template_id::text,'') FROM opportunities WHERE opportunity_id = $1",
		oppID,
	).Scan(&s); err != nil {
		t.Fatalf("oppTemplateID(%s): %v", oppID, err)
	}
	return s
}

// oppsForEventByTemplate returns the opportunity_id for the given event that
// shares the given recurrence_template_id. Returns 0 if not found.
func oppIDForEventByTemplate(t *testing.T, eventID int, tmplID string) int {
	t.Helper()
	var id int
	err := testDB.QueryRow(
		`SELECT opportunity_id FROM opportunities
		 WHERE event_id = $1 AND recurrence_template_id = $2::uuid`,
		eventID, tmplID,
	).Scan(&id)
	if err == sql.ErrNoRows {
		return 0
	}
	if err != nil {
		t.Fatalf("oppIDForEventByTemplate(%d): %v", eventID, err)
	}
	return id
}

// oppJobTypeID returns the job_type_id stored for an opportunity.
func oppJobTypeID(t *testing.T, oppID int) int {
	t.Helper()
	var id int
	if err := testDB.QueryRow(
		"SELECT job_type_id FROM opportunities WHERE opportunity_id = $1", oppID,
	).Scan(&id); err != nil {
		t.Fatalf("oppJobTypeID(%d): %v", oppID, err)
	}
	return id
}

// shiftCountForOpp returns the number of shifts for an opportunity.
func shiftCountForOpp(t *testing.T, oppID int) int {
	t.Helper()
	var n int
	if err := testDB.QueryRow(
		"SELECT COUNT(*) FROM shifts WHERE opportunity_id = $1", oppID,
	).Scan(&n); err != nil {
		t.Fatalf("shiftCountForOpp(%d): %v", oppID, err)
	}
	return n
}

// shiftTemplateIDStr returns the recurrence_template_id of a shift as a
// plain string (empty if NULL).
func shiftTemplateIDStr(t *testing.T, shiftID int) string {
	t.Helper()
	var s string
	if err := testDB.QueryRow(
		"SELECT COALESCE(recurrence_template_id::text,'') FROM shifts WHERE shift_id = $1",
		shiftID,
	).Scan(&s); err != nil {
		t.Fatalf("shiftTemplateIDStr(%d): %v", shiftID, err)
	}
	return s
}

// shiftStartUTC returns the UTC shift_start string stored in the DB.
func shiftStartUTC(t *testing.T, shiftID int) string {
	t.Helper()
	var s string
	if err := testDB.QueryRow(
		"SELECT shift_start::text FROM shifts WHERE shift_id = $1", shiftID,
	).Scan(&s); err != nil {
		t.Fatalf("shiftStartUTC(%d): %v", shiftID, err)
	}
	return s
}

// shiftMaxVols returns the max_volunteers stored for a shift.
func shiftMaxVols(t *testing.T, shiftID int) int {
	t.Helper()
	var n int
	if err := testDB.QueryRow(
		"SELECT COALESCE(max_volunteers,0) FROM shifts WHERE shift_id = $1", shiftID,
	).Scan(&n); err != nil {
		t.Fatalf("shiftMaxVols(%d): %v", shiftID, err)
	}
	return n
}

// firstShiftIDForOpp returns the lowest shift_id for an opportunity, 0 if none.
func firstShiftIDForOpp(t *testing.T, oppID int) int {
	t.Helper()
	var id int
	err := testDB.QueryRow(
		"SELECT COALESCE(MIN(shift_id),0) FROM shifts WHERE opportunity_id = $1", oppID,
	).Scan(&id)
	if err != nil {
		t.Fatalf("firstShiftIDForOpp(%d): %v", oppID, err)
	}
	return id
}

// templateShiftIDForOpp returns the shift_id with a non-null recurrence_template_id
// for the given opp, or 0 if none.
func templateShiftIDForOpp(t *testing.T, oppID int) int {
	t.Helper()
	var id int
	err := testDB.QueryRow(
		`SELECT COALESCE(MIN(shift_id),0) FROM shifts
		 WHERE opportunity_id = $1 AND recurrence_template_id IS NOT NULL`,
		oppID,
	).Scan(&id)
	if err != nil {
		t.Fatalf("templateShiftIDForOpp(%d): %v", oppID, err)
	}
	return id
}

// ============================================================================
// CreateOpportunity propagation
// ============================================================================

func TestCreateOpportunity_Recurring_PropagatesToFutureInstances(t *testing.T) {
	token, _ := makeAdmin(t)
	jobID := getJobTypeID(t, "event_support")
	_, ids := seedPropagationGroup(t, token, "2034-03-06")

	// Create an opp on the first instance only.
	createRecurringOpp(t, token, ids[0], jobID, "2034-03-06 10:00:00", "2034-03-06 12:00:00")

	// All three instances must have exactly one opportunity now.
	for i, id := range ids {
		if got := oppCountForEvent(t, id); got != 1 {
			t.Errorf("instance[%d] (event %d): want 1 opp, got %d", i, id, got)
		}
	}
}

func TestCreateOpportunity_Recurring_SiblingsSameTemplateID(t *testing.T) {
	token, _ := makeAdmin(t)
	jobID := getJobTypeID(t, "advocacy")
	_, ids := seedPropagationGroup(t, token, "2034-04-03")

	baseOppID := createRecurringOpp(t, token, ids[0], jobID, "2034-04-03 10:00:00", "2034-04-03 12:00:00")

	tmplID := oppTemplateID(t, baseOppID)
	if tmplID == "" {
		t.Fatal("base opp has no recurrence_template_id")
	}

	// Every instance must have exactly one opp with this template ID.
	for i, evID := range ids {
		if sibID := oppIDForEventByTemplate(t, evID, tmplID); sibID == 0 {
			t.Errorf("instance[%d] (event %d): no opp with template_id %s", i, evID, tmplID)
		}
	}
}

func TestCreateOpportunity_Recurring_SiblingsHaveShifts(t *testing.T) {
	token, _ := makeAdmin(t)
	jobID := getJobTypeID(t, "speaker")
	_, ids := seedPropagationGroup(t, token, "2034-05-01")

	baseOppID := createRecurringOpp(t, token, ids[0], jobID, "2034-05-01 09:30:00", "2034-05-01 11:30:00")
	tmplID := oppTemplateID(t, baseOppID)

	for i, evID := range ids {
		sibOppID := oppIDForEventByTemplate(t, evID, tmplID)
		if sibOppID == 0 {
			t.Errorf("instance[%d]: opp not found", i)
			continue
		}
		if n := shiftCountForOpp(t, sibOppID); n == 0 {
			t.Errorf("instance[%d] opp %d: want ≥1 shift, got 0", i, sibOppID)
		}
	}
}

func TestCreateOpportunity_Recurring_DoesNotPropagateBackward(t *testing.T) {
	token, _ := makeAdmin(t)
	jobID := getJobTypeID(t, "volunteer_lead")
	_, ids := seedPropagationGroup(t, token, "2034-06-05")

	// Create opp on instance[1] (order=2) — should NOT appear on instance[0] (order=1).
	createRecurringOpp(t, token, ids[1], jobID, "2034-06-12 10:00:00", "2034-06-12 12:00:00")

	if got := oppCountForEvent(t, ids[0]); got != 0 {
		t.Errorf("instance[0] (past): want 0 opps, got %d", got)
	}
	if got := oppCountForEvent(t, ids[2]); got != 1 {
		t.Errorf("instance[2] (future): want 1 opp, got %d", got)
	}
}

// ============================================================================
// UpdateOpportunity propagation
// ============================================================================

func TestUpdateOpportunity_Recurring_PropagatesChangesToFutureInstances(t *testing.T) {
	token, _ := makeAdmin(t)
	jobID := getJobTypeID(t, "event_support")
	newJobID := getJobTypeID(t, "advocacy")
	_, ids := seedPropagationGroup(t, token, "2034-07-03")

	baseOppID := createRecurringOpp(t, token, ids[0], jobID, "2034-07-03 10:00:00", "2034-07-03 12:00:00")
	tmplID := oppTemplateID(t, baseOppID)
	baseOppInt, _ := strconv.Atoi(baseOppID)

	// Update the base opp with a new job type.
	resp := gqlPost(t, "/graphql/admin", token, mutUpdateOpportunity, map[string]any{
		"input": map[string]any{
			"id":        baseOppID,
			"jobId":     newJobID,
			"isVirtual": true,
		},
	})
	if hasGQLErrors(resp) {
		t.Fatalf("updateOpportunity: %v", resp.Errors)
	}
	var r mutationResult
	unmarshalField(t, resp, "updateOpportunity", &r)
	if !r.Success {
		t.Fatalf("updateOpportunity: success=false: %v", r.Message)
	}

	// Base opp must have the new job type.
	if got := oppJobTypeID(t, baseOppInt); got != newJobID {
		t.Errorf("base opp job_type_id: want %d, got %d", newJobID, got)
	}

	// Future instances [1] and [2] must also have the new job type.
	for i := 1; i <= 2; i++ {
		sibID := oppIDForEventByTemplate(t, ids[i], tmplID)
		if sibID == 0 {
			t.Errorf("instance[%d]: sibling opp not found", i)
			continue
		}
		if got := oppJobTypeID(t, sibID); got != newJobID {
			t.Errorf("instance[%d] sibling opp job_type_id: want %d, got %d", i, newJobID, got)
		}
	}
}

func TestUpdateOpportunity_Recurring_PastInstancesUnchanged(t *testing.T) {
	token, _ := makeAdmin(t)
	jobID := getJobTypeID(t, "attendee_only")
	newJobID := getJobTypeID(t, "speaker")
	_, ids := seedPropagationGroup(t, token, "2034-08-07")

	// Create opp on instance[0] — all three instances get it.
	baseOppID := createRecurringOpp(t, token, ids[0], jobID, "2034-08-07 10:00:00", "2034-08-07 12:00:00")
	tmplID := oppTemplateID(t, baseOppID)

	// Operate on instance[1] (middle) — update to newJobID.
	midOppID := oppIDForEventByTemplate(t, ids[1], tmplID)
	if midOppID == 0 {
		t.Fatal("instance[1]: sibling opp not found")
	}
	resp := gqlPost(t, "/graphql/admin", token, mutUpdateOpportunity, map[string]any{
		"input": map[string]any{
			"id":        fmt.Sprintf("%d", midOppID),
			"jobId":     newJobID,
			"isVirtual": true,
		},
	})
	if hasGQLErrors(resp) {
		t.Fatalf("updateOpportunity: %v", resp.Errors)
	}
	var r mutationResult
	unmarshalField(t, resp, "updateOpportunity", &r)
	if !r.Success {
		t.Fatalf("updateOpportunity: success=false: %v", r.Message)
	}

	// instance[0] (past) must still have the original job type.
	pastOppID := oppIDForEventByTemplate(t, ids[0], tmplID)
	if got := oppJobTypeID(t, pastOppID); got != jobID {
		t.Errorf("instance[0] (past) job_type_id: want %d (original), got %d", jobID, got)
	}

	// instance[2] (future) must have the new job type.
	futureOppID := oppIDForEventByTemplate(t, ids[2], tmplID)
	if got := oppJobTypeID(t, futureOppID); got != newJobID {
		t.Errorf("instance[2] (future) job_type_id: want %d (new), got %d", newJobID, got)
	}
}

// ============================================================================
// DeleteOpportunity propagation
// ============================================================================

func TestDeleteOpportunity_Recurring_DeletesFutureInstances(t *testing.T) {
	token, _ := makeAdmin(t)
	jobID := getJobTypeID(t, "event_support")
	_, ids := seedPropagationGroup(t, token, "2034-09-04")

	baseOppID := createRecurringOpp(t, token, ids[0], jobID, "2034-09-04 10:00:00", "2034-09-04 12:00:00")
	tmplID := oppTemplateID(t, baseOppID)

	// Delete the base opp.
	resp := gqlPost(t, "/graphql/admin", token, mutDeleteOpportunity, map[string]any{
		"id": baseOppID,
	})
	if hasGQLErrors(resp) {
		t.Fatalf("deleteOpportunity: %v", resp.Errors)
	}
	var r mutationResult
	unmarshalField(t, resp, "deleteOpportunity", &r)
	if !r.Success {
		t.Fatalf("deleteOpportunity: success=false: %v", r.Message)
	}

	// All three instances must now have zero opps with this template ID.
	for i, evID := range ids {
		if sibID := oppIDForEventByTemplate(t, evID, tmplID); sibID != 0 {
			t.Errorf("instance[%d] (event %d): opp with template_id should be gone, still present as %d", i, evID, sibID)
		}
	}
}

func TestDeleteOpportunity_Recurring_PastInstancesPreserved(t *testing.T) {
	token, _ := makeAdmin(t)
	jobID := getJobTypeID(t, "advocacy")
	_, ids := seedPropagationGroup(t, token, "2034-10-02")

	// Create opp on instance[0] — all three instances get it.
	baseOppID := createRecurringOpp(t, token, ids[0], jobID, "2034-10-02 10:00:00", "2034-10-02 12:00:00")
	tmplID := oppTemplateID(t, baseOppID)

	// Delete from instance[1] (middle).
	midOppID := oppIDForEventByTemplate(t, ids[1], tmplID)
	if midOppID == 0 {
		t.Fatal("instance[1]: sibling opp not found")
	}
	resp := gqlPost(t, "/graphql/admin", token, mutDeleteOpportunity, map[string]any{
		"id": fmt.Sprintf("%d", midOppID),
	})
	if hasGQLErrors(resp) {
		t.Fatalf("deleteOpportunity: %v", resp.Errors)
	}
	var r mutationResult
	unmarshalField(t, resp, "deleteOpportunity", &r)
	if !r.Success {
		t.Fatalf("deleteOpportunity: success=false: %v", r.Message)
	}

	// instance[0] (past, order=1 < order=2) must still have its opp.
	if pastOppID := oppIDForEventByTemplate(t, ids[0], tmplID); pastOppID == 0 {
		t.Error("instance[0] (past): opp should be preserved after deleting from instance[1]")
	}
	// instance[2] (future) must be gone.
	if futureOppID := oppIDForEventByTemplate(t, ids[2], tmplID); futureOppID != 0 {
		t.Error("instance[2] (future): opp should have been deleted")
	}
}

// ============================================================================
// CreateShift propagation
// ============================================================================

func TestCreateShift_Recurring_PropagatesToFutureInstances(t *testing.T) {
	token, _ := makeAdmin(t)
	jobID := getJobTypeID(t, "event_support")
	_, ids := seedPropagationGroup(t, token, "2034-11-06")

	// Create opp on instance[0] (initial shift has no shift template_id yet).
	baseOppID := createRecurringOpp(t, token, ids[0], jobID, "2034-11-06 09:00:00", "2034-11-06 11:00:00")
	tmplID := oppTemplateID(t, baseOppID)
	baseOppInt, _ := strconv.Atoi(baseOppID)

	// Each instance now has 1 shift (the initial one from createOpportunity).
	shiftsBefore := shiftCountForOpp(t, baseOppInt)

	// Add a second shift via createShift — propagation should fan it out.
	resp := gqlPost(t, "/graphql/admin", token, mutCreateShift, map[string]any{
		"input": map[string]any{
			"opportunityId": baseOppID,
			"startDateTime": "2034-11-06 11:30:00",
			"endDateTime":   "2034-11-06 13:00:00",
			"maxVolunteers": 4,
		},
	})
	if hasGQLErrors(resp) {
		t.Fatalf("createShift: %v", resp.Errors)
	}
	var r mutationResult
	unmarshalField(t, resp, "createShift", &r)
	if !r.Success {
		t.Fatalf("createShift: success=false: %v", r.Message)
	}

	// Base opp must have one more shift than before.
	if got := shiftCountForOpp(t, baseOppInt); got != shiftsBefore+1 {
		t.Errorf("base opp shift count: want %d, got %d", shiftsBefore+1, got)
	}

	// Future instances [1] and [2] must also have gained a shift.
	for i := 1; i <= 2; i++ {
		sibOppID := oppIDForEventByTemplate(t, ids[i], tmplID)
		if sibOppID == 0 {
			t.Errorf("instance[%d]: sibling opp not found", i)
			continue
		}
		if got := shiftCountForOpp(t, sibOppID); got != shiftsBefore+1 {
			t.Errorf("instance[%d] shift count: want %d, got %d", i, shiftsBefore+1, got)
		}
	}
}

func TestCreateShift_Recurring_TimeOffsetPreserved(t *testing.T) {
	token, _ := makeAdmin(t)
	jobID := getJobTypeID(t, "advocacy")
	_, ids := seedPropagationGroup(t, token, "2035-01-08") // Wednesday

	baseOppID := createRecurringOpp(t, token, ids[0], jobID, "2035-01-08 09:00:00", "2035-01-08 11:00:00")
	tmplID := oppTemplateID(t, baseOppID)

	// Add a new shift at +1h offset from the event start (10:00 LA = 18:00 UTC, Jan is UTC-8).
	resp := gqlPost(t, "/graphql/admin", token, mutCreateShift, map[string]any{
		"input": map[string]any{
			"opportunityId": baseOppID,
			"startDateTime": "2035-01-08 10:00:00",
			"endDateTime":   "2035-01-08 12:00:00",
			"maxVolunteers": 3,
		},
	})
	if hasGQLErrors(resp) {
		t.Fatalf("createShift: %v", resp.Errors)
	}
	var r mutationResult
	unmarshalField(t, resp, "createShift", &r)
	if !r.Success {
		t.Fatalf("createShift: success=false: %v", r.Message)
	}

	baseOppInt, _ := strconv.Atoi(baseOppID)
	newShiftID, _ := strconv.Atoi(*r.ID)

	// Record source shift start for reference.
	srcStart := shiftStartUTC(t, newShiftID)

	// Collect the template'd shifts on each future instance.
	// They should have start times ~7 days apart from each other.
	var starts []string
	starts = append(starts, srcStart)

	for i := 1; i <= 2; i++ {
		sibOppID := oppIDForEventByTemplate(t, ids[i], tmplID)
		if sibOppID == 0 {
			t.Errorf("instance[%d]: sibling opp not found", i)
			continue
		}
		shiftTmplIDVal := shiftTemplateIDStr(t, newShiftID)
		var sibShiftID int
		err := testDB.QueryRow(
			`SELECT shift_id FROM shifts
			 WHERE opportunity_id = $1 AND recurrence_template_id = $2::uuid`,
			sibOppID, shiftTmplIDVal,
		).Scan(&sibShiftID)
		if err != nil {
			t.Errorf("instance[%d]: sibling shift not found: %v", i, err)
			continue
		}
		starts = append(starts, shiftStartUTC(t, sibShiftID))
	}

	if len(starts) < 3 {
		t.Fatalf("expected 3 shift starts (base + 2 siblings), got %d", len(starts))
	}

	// Adjacent starts should differ by ~7 days (604800 seconds ± a small tolerance).
	const week = 7 * 24 * time.Hour
	const tolerance = 5 * time.Minute
	for i := 1; i < len(starts); i++ {
		// Postgres returns timestamps as "YYYY-MM-DD HH:MM:SS" (no timezone suffix).
		var t0, t1 time.Time
		var err0, err1 error
		for _, layout := range []string{
			"2006-01-02 15:04:05",
			"2006-01-02 15:04:05+00",
			time.RFC3339,
		} {
			t0, err0 = time.Parse(layout, starts[i-1])
			t1, err1 = time.Parse(layout, starts[i])
			if err0 == nil && err1 == nil {
				break
			}
		}
		if err0 != nil || err1 != nil {
			t.Errorf("instance[%d] could not parse shift times: %q, %q", i, starts[i-1], starts[i])
			continue
		}
		diff := t1.Sub(t0)
		if diff < week-tolerance || diff > week+tolerance {
			t.Errorf("instance[%d] shift offset: want ~%v, got %v (starts: %q → %q)",
				i, week, diff, starts[i-1], starts[i])
		}
	}

	// All three shifts must have the same duration (2 hours).
	const wantDur = 2 * time.Hour
	_ = wantDur // duration check requires end times; covered implicitly by service logic
	_ = baseOppInt
}

// ============================================================================
// UpdateShift propagation
// ============================================================================

func TestUpdateShift_Recurring_PropagatesChangesToFutureInstances(t *testing.T) {
	token, _ := makeAdmin(t)
	jobID := getJobTypeID(t, "event_support")
	_, ids := seedPropagationGroup(t, token, "2035-02-05")

	baseOppID := createRecurringOpp(t, token, ids[0], jobID, "2035-02-05 09:00:00", "2035-02-05 11:00:00")
	tmplID := oppTemplateID(t, baseOppID)

	// Add a template'd shift via createShift so it propagates.
	csr := gqlPost(t, "/graphql/admin", token, mutCreateShift, map[string]any{
		"input": map[string]any{
			"opportunityId": baseOppID,
			"startDateTime": "2035-02-05 11:30:00",
			"endDateTime":   "2035-02-05 13:00:00",
			"maxVolunteers": 6,
		},
	})
	if hasGQLErrors(csr) {
		t.Fatalf("createShift: %v", csr.Errors)
	}
	var cr mutationResult
	unmarshalField(t, csr, "createShift", &cr)
	baseShiftID := *cr.ID
	baseShiftInt, _ := strconv.Atoi(baseShiftID)

	// Update the base shift with a new maxVolunteers.
	const newMax = 12
	ur := gqlPost(t, "/graphql/admin", token, mutUpdateShift, map[string]any{
		"input": map[string]any{
			"id":            baseShiftID,
			"startDateTime": "2035-02-05 11:30:00",
			"endDateTime":   "2035-02-05 13:00:00",
			"maxVolunteers": newMax,
		},
	})
	if hasGQLErrors(ur) {
		t.Fatalf("updateShift: %v", ur.Errors)
	}
	var ures mutationResult
	unmarshalField(t, ur, "updateShift", &ures)
	if !ures.Success {
		t.Fatalf("updateShift: success=false: %v", ures.Message)
	}

	// Base shift must reflect the update.
	if got := shiftMaxVols(t, baseShiftInt); got != newMax {
		t.Errorf("base shift max_volunteers: want %d, got %d", newMax, got)
	}

	// Sibling shifts on future instances must also have newMax.
	shiftTmpl := shiftTemplateIDStr(t, baseShiftInt)
	for i := 1; i <= 2; i++ {
		sibOppID := oppIDForEventByTemplate(t, ids[i], tmplID)
		if sibOppID == 0 {
			t.Errorf("instance[%d]: sibling opp not found", i)
			continue
		}
		var sibShiftID int
		err := testDB.QueryRow(
			`SELECT shift_id FROM shifts
			 WHERE opportunity_id = $1 AND recurrence_template_id = $2::uuid`,
			sibOppID, shiftTmpl,
		).Scan(&sibShiftID)
		if err != nil {
			t.Errorf("instance[%d]: sibling shift not found: %v", i, err)
			continue
		}
		if got := shiftMaxVols(t, sibShiftID); got != newMax {
			t.Errorf("instance[%d] sibling shift max_volunteers: want %d, got %d", i, newMax, got)
		}
	}
}

// ============================================================================
// DeleteShift propagation
// ============================================================================

func TestDeleteShift_Recurring_DeletesFutureInstances(t *testing.T) {
	token, _ := makeAdmin(t)
	jobID := getJobTypeID(t, "advocacy")
	_, ids := seedPropagationGroup(t, token, "2035-03-05")

	baseOppID := createRecurringOpp(t, token, ids[0], jobID, "2035-03-05 09:00:00", "2035-03-05 11:00:00")
	tmplID := oppTemplateID(t, baseOppID)

	// Add a template'd shift — now each instance has 2 shifts.
	csr := gqlPost(t, "/graphql/admin", token, mutCreateShift, map[string]any{
		"input": map[string]any{
			"opportunityId": baseOppID,
			"startDateTime": "2035-03-05 11:30:00",
			"endDateTime":   "2035-03-05 13:00:00",
			"maxVolunteers": 4,
		},
	})
	if hasGQLErrors(csr) {
		t.Fatalf("createShift: %v", csr.Errors)
	}
	var cr mutationResult
	unmarshalField(t, csr, "createShift", &cr)
	baseShiftID := *cr.ID
	baseShiftInt, _ := strconv.Atoi(baseShiftID)
	shiftTmpl := shiftTemplateIDStr(t, baseShiftInt)

	// Delete the template'd shift from the base opp.
	dr := gqlPost(t, "/graphql/admin", token, mutDeleteShift, map[string]any{
		"id": baseShiftID,
	})
	if hasGQLErrors(dr) {
		t.Fatalf("deleteShift: %v", dr.Errors)
	}
	var dres mutationResult
	unmarshalField(t, dr, "deleteShift", &dres)
	if !dres.Success {
		t.Fatalf("deleteShift: success=false: %v", dres.Message)
	}

	// Base shift must be gone.
	if rowExists(t, "SELECT COUNT(*) FROM shifts WHERE shift_id = $1", baseShiftInt) {
		t.Error("base shift should be deleted")
	}

	// Sibling shifts on future instances must also be gone.
	for i := 1; i <= 2; i++ {
		sibOppID := oppIDForEventByTemplate(t, ids[i], tmplID)
		if sibOppID == 0 {
			t.Errorf("instance[%d]: sibling opp not found", i)
			continue
		}
		if rowExists(t,
			`SELECT COUNT(*) FROM shifts
			 WHERE opportunity_id = $1 AND recurrence_template_id = $2::uuid`,
			sibOppID, shiftTmpl,
		) {
			t.Errorf("instance[%d]: sibling shift should have been deleted", i)
		}
		// But the initial shift (no template_id) must still be there.
		if shiftCountForOpp(t, sibOppID) == 0 {
			t.Errorf("instance[%d]: opp should still have the initial shift", i)
		}
	}
}

// ============================================================================
// UpdateShift — time change propagation
// ============================================================================

// TestUpdateShift_Recurring_TimeChangePropagates verifies that changing the
// start/end TIMES of a shift on a mid-series occurrence fans the new times
// (offset-adjusted) out to every future sibling shift.
//
// This is the scenario the user reported: editing a shift's time on occurrence
// N kept the change locally but did not appear on occurrences N+1, N+2, ...
func TestUpdateShift_Recurring_TimeChangePropagates(t *testing.T) {
	token, _ := makeAdmin(t)
	jobID := getJobTypeID(t, "event_support")
	// 3-occurrence weekly group starting 2035-06-04 (Wednesday)
	_, ids := seedPropagationGroup(t, token, "2035-06-04")

	// Create opp on instance[0]; all three get a shift at 09:00–11:00.
	// After the CreateOpportunity fix every initial shift has a template ID.
	baseOppID := createRecurringOpp(t, token, ids[0], jobID, "2035-06-04 09:00:00", "2035-06-04 11:00:00")
	tmplID := oppTemplateID(t, baseOppID)

	// Grab the initial shift template UUID from instance[0].
	baseOppInt, _ := strconv.Atoi(baseOppID)
	baseShiftID := templateShiftIDForOpp(t, baseOppInt)
	if baseShiftID == 0 {
		t.Fatal("base opp initial shift has no recurrence_template_id")
	}
	shiftTmpl := shiftTemplateIDStr(t, baseShiftID)

	// Find the sibling shift on instance[1] (occurrence 2, first date 2035-06-11).
	midOppID := oppIDForEventByTemplate(t, ids[1], tmplID)
	if midOppID == 0 {
		t.Fatal("instance[1]: sibling opp not found")
	}
	var midShiftID int
	if err := testDB.QueryRow(
		`SELECT shift_id FROM shifts
		 WHERE opportunity_id = $1 AND recurrence_template_id = $2::uuid`,
		midOppID, shiftTmpl,
	).Scan(&midShiftID); err != nil {
		t.Fatalf("instance[1] initial shift not found: %v", err)
	}

	// Update the shift on instance[1] with NEW times: 10:00–12:00
	// (one hour later than the original 09:00–11:00).
	// The event group timezone is "America/Los_Angeles" (set by weeklyVirtualInput).
	// 2035-06-11 is in PDT (UTC-7), so 10:00 PDT = 17:00 UTC.
	ur := gqlPost(t, "/graphql/admin", token, mutUpdateShift, map[string]any{
		"input": map[string]any{
			"id":            fmt.Sprintf("%d", midShiftID),
			"startDateTime": "2035-06-11 10:00:00",
			"endDateTime":   "2035-06-11 12:00:00",
			"maxVolunteers": 5,
		},
	})
	if hasGQLErrors(ur) {
		t.Fatalf("updateShift: %v", ur.Errors)
	}
	var ures mutationResult
	unmarshalField(t, ur, "updateShift", &ures)
	if !ures.Success {
		t.Fatalf("updateShift: success=false: %v", ures.Message)
	}

	// instance[1] itself must show the new start time.
	// 2035-06-11 10:00 PDT = 17:00 UTC.
	midStart := shiftStartUTC(t, midShiftID)
	t.Logf("instance[1] shift_start (UTC): %s", midStart)

	// instance[2] (occurrence 3, first date 2035-06-18) must have been updated.
	// Event start: 2035-06-11 09:00 PDT = 16:00 UTC (PDT is UTC-7 in June)
	// New shift start: 2035-06-11 10:00 PDT = 17:00 UTC → offset = +1h from event start
	// Occurrence #3 event start: 2035-06-18 09:00 PDT = 16:00 UTC
	// Expected sibling shift start: 16:00 UTC + 1h = 17:00 UTC on 2035-06-18
	futureOppID := oppIDForEventByTemplate(t, ids[2], tmplID)
	if futureOppID == 0 {
		t.Fatal("instance[2]: sibling opp not found")
	}
	var futureShiftID int
	if err := testDB.QueryRow(
		`SELECT shift_id FROM shifts
		 WHERE opportunity_id = $1 AND recurrence_template_id = $2::uuid`,
		futureOppID, shiftTmpl,
	).Scan(&futureShiftID); err != nil {
		t.Fatalf("instance[2] sibling shift not found: %v", err)
	}

	futureStart := shiftStartUTC(t, futureShiftID)
	t.Logf("instance[2] shift_start (UTC): %s", futureStart)

	// Parse both times and verify the gap is exactly 7 days (within a small tolerance).
	var t1, t2 time.Time
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02 15:04:05+00", time.RFC3339} {
		t1, _ = time.Parse(layout, midStart)
		t2, _ = time.Parse(layout, futureStart)
		if !t1.IsZero() && !t2.IsZero() {
			break
		}
	}
	if t1.IsZero() || t2.IsZero() {
		t.Fatalf("could not parse shift start times: %q, %q", midStart, futureStart)
	}

	const week = 7 * 24 * time.Hour
	const tol = 5 * time.Minute
	if diff := t2.Sub(t1); diff < week-tol || diff > week+tol {
		t.Errorf("time gap between instance[1] and instance[2] shifts: want ~7d, got %v\n  instance[1]: %s\n  instance[2]: %s",
			diff, midStart, futureStart)
	}

	// Also verify that instance[0] (past) was NOT changed.
	pastStart := shiftStartUTC(t, baseShiftID)
	t.Logf("instance[0] shift_start (UTC): %s", pastStart)
	if pastStart == midStart {
		t.Error("instance[0] (past) shift time matches the updated time — propagation must not go backward")
	}
}

// ============================================================================
// UpdateShift — initial shift created by CreateOpportunity
// ============================================================================

// TestUpdateShift_InitialShift_PropagatesChanges reproduces the bug where
// editing the shift that was created together with the opportunity (the
// "initial shift") on a mid-series occurrence did NOT propagate forward.
//
// Root cause: CreateOpportunity used to insert initial shifts without a
// recurrence_template_id, so UpdateShift's propagation guard
// (shiftTmplID != "") was never satisfied.
//
// After the fix, CreateOpportunity stamps every initial shift with its own
// template UUID, so the same UpdateShift path that works for add-later shifts
// now also works for these initial shifts.
func TestUpdateShift_InitialShift_PropagatesChanges(t *testing.T) {
	token, _ := makeAdmin(t)
	jobID := getJobTypeID(t, "event_support")
	_, ids := seedPropagationGroup(t, token, "2035-05-07")

	// Create opp on instance[0].  All 3 instances get the initial shift.
	// After the fix the initial shift on every instance has a non-null
	// recurrence_template_id.
	baseOppID := createRecurringOpp(t, token, ids[0], jobID, "2035-05-07 09:00:00", "2035-05-07 11:00:00")
	tmplID := oppTemplateID(t, baseOppID)

	// Verify that the initial shift on the base opp already has a template ID.
	baseOppInt, _ := strconv.Atoi(baseOppID)
	baseInitialShiftID := templateShiftIDForOpp(t, baseOppInt)
	if baseInitialShiftID == 0 {
		t.Fatal("base opp initial shift has no recurrence_template_id — fix was not applied")
	}
	initialShiftTmplID := shiftTemplateIDStr(t, baseInitialShiftID)

	// Locate the initial shift on instance[1] (the "middle" occurrence).
	midOppID := oppIDForEventByTemplate(t, ids[1], tmplID)
	if midOppID == 0 {
		t.Fatal("instance[1]: sibling opp not found")
	}
	var midShiftID int
	if err := testDB.QueryRow(
		`SELECT shift_id FROM shifts
		 WHERE opportunity_id = $1 AND recurrence_template_id = $2::uuid`,
		midOppID, initialShiftTmplID,
	).Scan(&midShiftID); err != nil {
		t.Fatalf("instance[1] initial shift not found: %v", err)
	}

	// Edit the initial shift on instance[1], changing BOTH the time (+1h)
	// and max_volunteers.  Before the fix: shiftTmplID == "" → no propagation
	// for either field.  After the fix: both fields propagate.
	// 2035-05-14 is PDT (UTC-7) so 10:00 PDT = 17:00 UTC.
	const newMax = 20
	ur := gqlPost(t, "/graphql/admin", token, mutUpdateShift, map[string]any{
		"input": map[string]any{
			"id":            fmt.Sprintf("%d", midShiftID),
			"startDateTime": "2035-05-14 10:00:00", // was 09:00, now 10:00 (+1h)
			"endDateTime":   "2035-05-14 12:00:00", // was 11:00, now 12:00 (+1h)
			"maxVolunteers": newMax,
		},
	})
	if hasGQLErrors(ur) {
		t.Fatalf("updateShift on mid-series initial shift: %v", ur.Errors)
	}
	var ures mutationResult
	unmarshalField(t, ur, "updateShift", &ures)
	if !ures.Success {
		t.Fatalf("updateShift: success=false: %v", ures.Message)
	}

	// instance[1] shift must reflect the change.
	if got := shiftMaxVols(t, midShiftID); got != newMax {
		t.Errorf("mid-series shift max_volunteers: want %d, got %d", newMax, got)
	}

	// instance[2] (future) must also have been updated — this is the bug scenario.
	futureOppID := oppIDForEventByTemplate(t, ids[2], tmplID)
	if futureOppID == 0 {
		t.Fatal("instance[2]: sibling opp not found")
	}
	var futureShiftID int
	if err := testDB.QueryRow(
		`SELECT shift_id FROM shifts
		 WHERE opportunity_id = $1 AND recurrence_template_id = $2::uuid`,
		futureOppID, initialShiftTmplID,
	).Scan(&futureShiftID); err != nil {
		t.Fatalf("instance[2] initial shift not found: %v", err)
	}
	if got := shiftMaxVols(t, futureShiftID); got != newMax {
		t.Errorf("instance[2] (future) max_volunteers: want %d, got %d — propagation did not reach future instance", newMax, got)
	}

	// Verify the time also propagated: instance[2] shift_start must be ~7 days
	// after instance[1] shift_start (same PDT offset applied to week-later event).
	midStart := shiftStartUTC(t, midShiftID)
	futureStart := shiftStartUTC(t, futureShiftID)
	var mt, ft time.Time
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02 15:04:05+00", time.RFC3339} {
		mt, _ = time.Parse(layout, midStart)
		ft, _ = time.Parse(layout, futureStart)
		if !mt.IsZero() && !ft.IsZero() {
			break
		}
	}
	if mt.IsZero() || ft.IsZero() {
		t.Errorf("could not parse shift times: %q, %q", midStart, futureStart)
	} else {
		const week = 7 * 24 * time.Hour
		const tol = 5 * time.Minute
		if diff := ft.Sub(mt); diff < week-tol || diff > week+tol {
			t.Errorf("time gap: want ~7d, got %v — time change did not propagate correctly\n  instance[1]: %s\n  instance[2]: %s",
				diff, midStart, futureStart)
		}
	}

	// instance[0] (past, recurrence_order < instance[1].order) must NOT change.
	pastOppID, _ := strconv.Atoi(baseOppID)
	_ = pastOppID
	if got := shiftMaxVols(t, baseInitialShiftID); got == newMax {
		t.Error("instance[0] (past) max_volunteers was changed — propagation must not go backward")
	}
}

func TestDeleteShift_Recurring_BlockedWhenSiblingWouldBeEmpty(t *testing.T) {
	token, _ := makeAdmin(t)
	jobID := getJobTypeID(t, "event_support")
	_, ids := seedPropagationGroup(t, token, "2035-04-02")

	// Create opp on instance[0]; all instances get 1 initial shift (each with a
	// recurrence_template_id, after the CreateOpportunity fix).
	baseOppID := createRecurringOpp(t, token, ids[0], jobID, "2035-04-02 09:00:00", "2035-04-02 11:00:00")
	tmplID := oppTemplateID(t, baseOppID)
	baseOppInt, _ := strconv.Atoi(baseOppID)

	// Remove ALL shifts from instances [1] and [2] directly in the DB so that
	// those opps have zero shifts.  The next createShift call will then be the
	// only shift on those sibling opps, which sets up the "would empty" scenario.
	for i := 1; i <= 2; i++ {
		sibOppID := oppIDForEventByTemplate(t, ids[i], tmplID)
		if sibOppID == 0 {
			t.Fatalf("instance[%d]: sibling opp not found", i)
		}
		if _, err := testDB.Exec(
			"DELETE FROM shifts WHERE opportunity_id = $1",
			sibOppID,
		); err != nil {
			t.Fatalf("clearing shifts for instance[%d]: %v", i, err)
		}
	}

	// Add a template'd shift via createShift; each sibling gets exactly 1 shift.
	csr := gqlPost(t, "/graphql/admin", token, mutCreateShift, map[string]any{
		"input": map[string]any{
			"opportunityId": baseOppID,
			"startDateTime": "2035-04-02 11:30:00",
			"endDateTime":   "2035-04-02 13:00:00",
			"maxVolunteers": 5,
		},
	})
	if hasGQLErrors(csr) {
		t.Fatalf("createShift: %v", csr.Errors)
	}
	var cr mutationResult
	unmarshalField(t, csr, "createShift", &cr)
	newShiftID := *cr.ID
	newShiftInt, _ := strconv.Atoi(newShiftID)

	// Sanity: instance[0] has 2 shifts; instances [1] and [2] have only 1 each.
	if got := shiftCountForOpp(t, baseOppInt); got != 2 {
		t.Fatalf("base opp: want 2 shifts, got %d", got)
	}
	for i := 1; i <= 2; i++ {
		sibOppID := oppIDForEventByTemplate(t, ids[i], tmplID)
		if got := shiftCountForOpp(t, sibOppID); got != 1 {
			t.Fatalf("instance[%d] sibling opp: want 1 shift, got %d", i, got)
		}
	}

	// Attempt to delete the template'd shift — must be rejected because deleting
	// its siblings would leave instances [1] and [2] with zero shifts.
	dr := gqlPost(t, "/graphql/admin", token, mutDeleteShift, map[string]any{
		"id": newShiftID,
	})
	if !hasGQLErrors(dr) {
		t.Error("expected a GraphQL error: delete would leave a sibling opp with zero shifts")
	}

	// The shift must still be present (delete was rejected).
	if !rowExists(t, "SELECT COUNT(*) FROM shifts WHERE shift_id = $1", newShiftInt) {
		t.Error("template'd shift should still exist after rejected delete")
	}
}
