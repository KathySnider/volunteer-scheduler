package integration

// ============================================================================
// Integration tests — admin filteredEvents query
// ============================================================================
//
// The admin endpoint exposes filteredEvents (not filteredEventsWithShifts).
// Key behavioural difference from the volunteer endpoint: events that have no
// shifts are included in admin results so that admins can see and manage
// incomplete events.
//
// Tests:
//   - Events with no shifts appear in admin results
//   - Events with no shifts do NOT appear in volunteer results (contrast test)
//   - City filter works on the admin endpoint
//   - TimeFrame filter works on the admin endpoint

import (
	"fmt"
	"testing"
	"time"
)

// ============================================================================
// GraphQL operation strings
// ============================================================================

const queryAdminFilteredEvents = `
	query FilteredEvents($filter: EventFilterInput) {
		filteredEvents(filter: $filter) {
			id
			name
			eventType
		}
	}`

// ============================================================================
// Helpers
// ============================================================================

// adminEventNamesFromResponse unmarshals the admin filteredEvents field and
// returns a map of event name → true for quick membership checks.
func adminEventNamesFromResponse(t *testing.T, resp gqlResponse) map[string]bool {
	t.Helper()
	var events []struct {
		Name string `json:"name"`
	}
	unmarshalField(t, resp, "filteredEvents", &events)
	m := make(map[string]bool, len(events))
	for _, e := range events {
		m[e.Name] = true
	}
	return m
}

// ============================================================================
// No-shifts visibility — the core admin vs volunteer distinction
// ============================================================================

// TestAdminFilteredEvents_IncludesNoShiftEvents verifies that an event with no
// shifts appears in admin results but not in volunteer results.
func TestAdminFilteredEvents_IncludesNoShiftEvents(t *testing.T) {
	adminToken, _ := makeAdmin(t)
	volToken, _ := makeVolunteer(t)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	venueID := seedVenue(t, "Admin Test Venue "+suffix, "1 Admin St", "Tacoma", "WA", "America/Los_Angeles")
	jobID := getJobTypeID(t, "event_support")

	// Event WITH shifts — should appear for both admin and volunteer.
	withShiftsName := "WithShifts-" + suffix
	withShiftsID := seedEvent(t, withShiftsName, false, &venueID)
	seedEventDate(t, withShiftsID, "2028-05-01T09:00:00Z", "2028-05-01T17:00:00Z")
	oppID := seedOpportunity(t, withShiftsID, jobID, false)
	seedShift(t, oppID, "2028-05-01T09:00:00Z", "2028-05-01T12:00:00Z", 5)

	// Event WITHOUT shifts — should appear for admin only.
	noShiftsName := "NoShifts-" + suffix
	noShiftsID := seedEvent(t, noShiftsName, false, &venueID)
	seedEventDate(t, noShiftsID, "2028-05-02T09:00:00Z", "2028-05-02T17:00:00Z")
	// Deliberately no opportunity or shifts.

	filter := map[string]any{
		"filter": map[string]any{
			"cities":    []string{"Tacoma"},
			"timeFrame": "ALL",
		},
	}

	// ── Admin sees both ──────────────────────────────────────────────────────
	adminResp := gqlPost(t, "/graphql/admin", adminToken, queryAdminFilteredEvents, filter)
	if hasGQLErrors(adminResp) {
		t.Fatalf("admin: unexpected GraphQL errors: %v", adminResp.Errors)
	}
	adminGot := adminEventNamesFromResponse(t, adminResp)

	if !adminGot[withShiftsName] {
		t.Errorf("admin: expected event with shifts %q in results", withShiftsName)
	}
	if !adminGot[noShiftsName] {
		t.Errorf("admin: expected no-shifts event %q in results", noShiftsName)
	}

	// ── Volunteer sees only the event that has shifts ────────────────────────
	volResp := gqlPost(t, "/graphql/volunteer", volToken, queryFilteredEvents, filter)
	if hasGQLErrors(volResp) {
		t.Fatalf("volunteer: unexpected GraphQL errors: %v", volResp.Errors)
	}
	volGot := eventNamesFromResponse(t, volResp)

	if !volGot[withShiftsName] {
		t.Errorf("volunteer: expected event with shifts %q in results", withShiftsName)
	}
	if volGot[noShiftsName] {
		t.Errorf("volunteer: no-shifts event %q should not appear in volunteer results", noShiftsName)
	}
}

// ============================================================================
// Admin — city filter
// ============================================================================

func TestAdminFilteredEvents_CityFilter(t *testing.T) {
	adminToken, _ := makeAdmin(t)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	jobID := getJobTypeID(t, "event_support")

	olyVenueID := seedVenue(t, "Olympia Venue "+suffix, "1 Capitol Way", "Olympia", "WA", "America/Los_Angeles")
	belVenueID := seedVenue(t, "Bellingham Venue "+suffix, "2 State St", "Bellingham", "WA", "America/Los_Angeles")

	olyName := "OlympiaEvent-" + suffix
	belName := "BellinghamEvent-" + suffix

	olyID := seedEvent(t, olyName, false, &olyVenueID)
	belID := seedEvent(t, belName, false, &belVenueID)

	for _, eid := range []int{olyID, belID} {
		seedEventDate(t, eid, "2028-07-01T09:00:00Z", "2028-07-01T17:00:00Z")
		opp := seedOpportunity(t, eid, jobID, false)
		seedShift(t, opp, "2028-07-01T09:00:00Z", "2028-07-01T12:00:00Z", 3)
	}

	resp := gqlPost(t, "/graphql/admin", adminToken, queryAdminFilteredEvents, map[string]any{
		"filter": map[string]any{
			"cities":    []string{"Olympia"},
			"timeFrame": "ALL",
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GraphQL errors: %v", resp.Errors)
	}

	got := adminEventNamesFromResponse(t, resp)

	if !got[olyName] {
		t.Errorf("expected Olympia event %q in results", olyName)
	}
	if got[belName] {
		t.Errorf("Bellingham event %q should not appear in Olympia-only filter", belName)
	}
}

// ============================================================================
// Admin — timeFrame filter
// ============================================================================

func TestAdminFilteredEvents_TimeFrameUpcoming(t *testing.T) {
	adminToken, _ := makeAdmin(t)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	jobID := getJobTypeID(t, "event_support")
	venueID := seedVenue(t, "TF Admin Venue "+suffix, "3 Time St", "Yakima", "WA", "America/Los_Angeles")

	futureName := "AdminFuture-" + suffix
	pastName := "AdminPast-" + suffix

	futureID := seedEvent(t, futureName, false, &venueID)
	pastID := seedEvent(t, pastName, false, &venueID)

	seedEventDate(t, futureID, "2029-01-01T09:00:00Z", "2029-01-01T17:00:00Z")
	seedEventDate(t, pastID, "2021-01-01T09:00:00Z", "2021-01-01T17:00:00Z")

	futureOpp := seedOpportunity(t, futureID, jobID, false)
	pastOpp := seedOpportunity(t, pastID, jobID, false)
	seedShift(t, futureOpp, "2029-01-01T09:00:00Z", "2029-01-01T12:00:00Z", 3)
	seedShift(t, pastOpp, "2021-01-01T09:00:00Z", "2021-01-01T12:00:00Z", 3)

	resp := gqlPost(t, "/graphql/admin", adminToken, queryAdminFilteredEvents, map[string]any{
		"filter": map[string]any{
			"cities":    []string{"Yakima"},
			"timeFrame": "UPCOMING",
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GraphQL errors: %v", resp.Errors)
	}

	got := adminEventNamesFromResponse(t, resp)

	if !got[futureName] {
		t.Errorf("expected future event %q in UPCOMING results", futureName)
	}
	if got[pastName] {
		t.Errorf("past event %q should not appear in UPCOMING results", pastName)
	}
}

func TestAdminFilteredEvents_TimeFrameAll(t *testing.T) {
	adminToken, _ := makeAdmin(t)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	jobID := getJobTypeID(t, "event_support")
	venueID := seedVenue(t, "TF Admin ALL Venue "+suffix, "4 All St", "Walla Walla", "WA", "America/Los_Angeles")

	futureName := "AdminAllFuture-" + suffix
	pastName := "AdminAllPast-" + suffix

	futureID := seedEvent(t, futureName, false, &venueID)
	pastID := seedEvent(t, pastName, false, &venueID)

	seedEventDate(t, futureID, "2029-02-01T09:00:00Z", "2029-02-01T17:00:00Z")
	seedEventDate(t, pastID, "2021-02-01T09:00:00Z", "2021-02-01T17:00:00Z")

	futureOpp := seedOpportunity(t, futureID, jobID, false)
	pastOpp := seedOpportunity(t, pastID, jobID, false)
	seedShift(t, futureOpp, "2029-02-01T09:00:00Z", "2029-02-01T12:00:00Z", 3)
	seedShift(t, pastOpp, "2021-02-01T09:00:00Z", "2021-02-01T12:00:00Z", 3)

	resp := gqlPost(t, "/graphql/admin", adminToken, queryAdminFilteredEvents, map[string]any{
		"filter": map[string]any{
			"cities":    []string{"Walla Walla"},
			"timeFrame": "ALL",
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GraphQL errors: %v", resp.Errors)
	}

	got := adminEventNamesFromResponse(t, resp)

	if !got[futureName] {
		t.Errorf("expected future event %q in ALL results", futureName)
	}
	if !got[pastName] {
		t.Errorf("expected past event %q in ALL results", pastName)
	}
}
