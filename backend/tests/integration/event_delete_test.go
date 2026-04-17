package integration

// ============================================================================
// Integration tests — DeleteEvent mutation and city/timeFrame filters
// ============================================================================
//
// DeleteEvent tests:
//   - Happy path: event with shifts+volunteers is deleted; rows cascade
//   - Event with no opportunities or shifts can be deleted (bug fix)
//   - Invalid ID returns success=false, no server error
//
// filteredEventsWithShifts — cities filter:
//   - Filtering by one city returns only events in that city
//   - Filtering by multiple cities returns events from either city
//   - City filter is case-sensitive (matches DB value)
//
// filteredEventsWithShifts — timeFrame filter:
//   - UPCOMING returns future shifts, not past
//   - PAST returns past shifts, not future
//   - ALL returns both past and future
//   - Combined city + timeFrame narrows correctly

import (
	"fmt"
	"testing"
	"time"
)

// ============================================================================
// GraphQL operation strings
// ============================================================================

const mutationDeleteEvent = `
	mutation DeleteEvent($eventId: ID!) {
		deleteEvent(eventId: $eventId) {
			success
			message
		}
	}`

// ============================================================================
// DeleteEvent — happy path (event has shifts and a signed-up volunteer)
// ============================================================================

func TestDeleteEvent_WithShiftsAndVolunteer(t *testing.T) {
	adminToken, _ := makeAdmin(t)
	volToken, volID := makeVolunteer(t)
	_ = volToken

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	venueID := seedVenue(t, "Delete Venue "+suffix, "1 Main St", "Portland", "OR", "America/Los_Angeles")
	jobID := getJobTypeID(t, "event_support")

	eventName := "DeleteMe-" + suffix
	eventID := seedEvent(t, eventName, false, &venueID)
	seedEventDate(t, eventID, "2027-03-01T09:00:00Z", "2027-03-01T17:00:00Z")
	oppID := seedOpportunity(t, eventID, jobID, false)
	shiftID := seedShift(t, oppID, "2027-03-01T09:00:00Z", "2027-03-01T12:00:00Z", 5)
	seedVolunteerShift(t, shiftID, volID)

	// Confirm the event exists before deletion.
	if !rowExists(t, "SELECT COUNT(*) FROM events WHERE event_id = $1", eventID) {
		t.Fatal("expected event to exist before delete")
	}

	resp := gqlPost(t, "/graphql/admin", adminToken, mutationDeleteEvent, map[string]any{
		"eventId": fmt.Sprintf("%d", eventID),
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GraphQL errors: %v", resp.Errors)
	}

	var result struct {
		DeleteEvent mutationResult `json:"deleteEvent"`
	}
	unmarshalField(t, resp, "deleteEvent", &result.DeleteEvent)

	if !result.DeleteEvent.Success {
		msg := ""
		if result.DeleteEvent.Message != nil {
			msg = *result.DeleteEvent.Message
		}
		t.Fatalf("deleteEvent returned success=false: %s", msg)
	}

	// The event row should be gone (cascade removes opportunities, shifts,
	// volunteer_shifts). The cleanup registered by seedEvent/seedShift etc.
	// will no-op gracefully on missing rows.
	if rowExists(t, "SELECT COUNT(*) FROM events WHERE event_id = $1", eventID) {
		t.Error("event row should have been deleted but still exists")
	}
	if rowExists(t, "SELECT COUNT(*) FROM shifts WHERE shift_id = $1", shiftID) {
		t.Error("shift row should have been cascade-deleted but still exists")
	}
	if rowExists(t,
		"SELECT COUNT(*) FROM volunteer_shifts WHERE shift_id = $1 AND volunteer_id = $2",
		shiftID, volID,
	) {
		t.Error("volunteer_shifts row should have been cascade-deleted but still exists")
	}
}

// ============================================================================
// DeleteEvent — event with no opportunities or shifts (the bug we fixed)
// ============================================================================

func TestDeleteEvent_NoOpportunitiesOrShifts(t *testing.T) {
	adminToken, _ := makeAdmin(t)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	venueID := seedVenue(t, "Empty Venue "+suffix, "2 Empty Rd", "Boise", "ID", "America/Boise")

	eventName := "EmptyEvent-" + suffix
	eventID := seedEvent(t, eventName, false, &venueID)
	seedEventDate(t, eventID, "2027-04-01T09:00:00Z", "2027-04-01T17:00:00Z")
	// Deliberately NO opportunities or shifts.

	resp := gqlPost(t, "/graphql/admin", adminToken, mutationDeleteEvent, map[string]any{
		"eventId": fmt.Sprintf("%d", eventID),
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GraphQL errors: %v", resp.Errors)
	}

	var result struct {
		DeleteEvent mutationResult `json:"deleteEvent"`
	}
	unmarshalField(t, resp, "deleteEvent", &result.DeleteEvent)

	if !result.DeleteEvent.Success {
		msg := ""
		if result.DeleteEvent.Message != nil {
			msg = *result.DeleteEvent.Message
		}
		t.Fatalf("deleteEvent (no shifts) returned success=false: %s", msg)
	}

	if rowExists(t, "SELECT COUNT(*) FROM events WHERE event_id = $1", eventID) {
		t.Error("event row should have been deleted but still exists")
	}
}

// ============================================================================
// DeleteEvent — invalid ID
// ============================================================================

func TestDeleteEvent_InvalidID(t *testing.T) {
	adminToken, _ := makeAdmin(t)

	resp := gqlPost(t, "/graphql/admin", adminToken, mutationDeleteEvent, map[string]any{
		"eventId": "not-a-number",
	})

	// Should return a resolver-level error or success=false, not a 500.
	var result struct {
		DeleteEvent mutationResult `json:"deleteEvent"`
	}

	// Either GraphQL errors or success=false is acceptable — just not a panic/500.
	if !hasGQLErrors(resp) {
		unmarshalField(t, resp, "deleteEvent", &result.DeleteEvent)
		if result.DeleteEvent.Success {
			t.Error("expected success=false for invalid event ID, got true")
		}
	}
}

// ============================================================================
// filteredEventsWithShifts — cities filter
// ============================================================================

type cityFilterFixture struct {
	token        string
	seattleName  string
	spokaneName  string
	portlandName string
}

func setupCityFilterFixture(t *testing.T) cityFilterFixture {
	t.Helper()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	email := uniqueEmail(t)
	volID := seedVolunteer(t, email, "City", "Tester", "VOLUNTEER")
	token := seedSession(t, email, volID, "VOLUNTEER", "city-"+suffix)

	jobID := getJobTypeID(t, "event_support")

	seattleVenueID  := seedVenue(t, "Seattle Lib "+suffix, "100 4th Ave", "Seattle", "WA", "America/Los_Angeles")
	spokaneVenueID  := seedVenue(t, "Spokane Hall "+suffix, "200 Monroe", "Spokane", "WA", "America/Los_Angeles")
	portlandVenueID := seedVenue(t, "Portland Ctr "+suffix, "300 SW 5th", "Portland", "OR", "America/Los_Angeles")

	seattleName  := "SeattleEvent-"  + suffix
	spokaneName  := "SpokaneEvent-"  + suffix
	portlandName := "PortlandEvent-" + suffix

	seattleID  := seedEvent(t, seattleName,  false, &seattleVenueID)
	spokaneID  := seedEvent(t, spokaneName,  false, &spokaneVenueID)
	portlandID := seedEvent(t, portlandName, false, &portlandVenueID)

	for _, eid := range []int{seattleID, spokaneID, portlandID} {
		seedEventDate(t, eid, "2027-06-01T09:00:00Z", "2027-06-01T17:00:00Z")
		oppID := seedOpportunity(t, eid, jobID, false)
		seedShift(t, oppID, "2027-06-01T09:00:00Z", "2027-06-01T12:00:00Z", 5)
	}

	return cityFilterFixture{
		token:        token,
		seattleName:  seattleName,
		spokaneName:  spokaneName,
		portlandName: portlandName,
	}
}

func TestFilteredEvents_BySingleCity(t *testing.T) {
	fx := setupCityFilterFixture(t)

	resp := gqlPost(t, "/graphql/volunteer", fx.token, queryFilteredEvents, map[string]any{
		"filter": map[string]any{
			"cities":    []string{"Seattle"},
			"timeFrame": "ALL",
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	got := eventNamesFromResponse(t, resp)

	if !got[fx.seattleName] {
		t.Errorf("expected Seattle event %q in results", fx.seattleName)
	}
	if got[fx.spokaneName] {
		t.Errorf("Spokane event %q should not appear in Seattle filter", fx.spokaneName)
	}
	if got[fx.portlandName] {
		t.Errorf("Portland event %q should not appear in Seattle filter", fx.portlandName)
	}
}

func TestFilteredEvents_ByMultipleCities(t *testing.T) {
	fx := setupCityFilterFixture(t)

	resp := gqlPost(t, "/graphql/volunteer", fx.token, queryFilteredEvents, map[string]any{
		"filter": map[string]any{
			"cities":    []string{"Seattle", "Portland"},
			"timeFrame": "ALL",
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	got := eventNamesFromResponse(t, resp)

	if !got[fx.seattleName] {
		t.Errorf("expected Seattle event %q in multi-city results", fx.seattleName)
	}
	if !got[fx.portlandName] {
		t.Errorf("expected Portland event %q in multi-city results", fx.portlandName)
	}
	if got[fx.spokaneName] {
		t.Errorf("Spokane event %q should not appear in Seattle+Portland filter", fx.spokaneName)
	}
}

func TestFilteredEvents_CityFilterNoMatch(t *testing.T) {
	fx := setupCityFilterFixture(t)

	resp := gqlPost(t, "/graphql/volunteer", fx.token, queryFilteredEvents, map[string]any{
		"filter": map[string]any{
			"cities":    []string{"Atlantis"},
			"timeFrame": "ALL",
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	got := eventNamesFromResponse(t, resp)

	for _, name := range []string{fx.seattleName, fx.spokaneName, fx.portlandName} {
		if got[name] {
			t.Errorf("event %q should not appear when filtering by non-existent city", name)
		}
	}
}

// ============================================================================
// filteredEventsWithShifts — timeFrame filter
// ============================================================================

type timeFrameFixture struct {
	token        string
	upcomingName string
	pastName     string
}

func setupTimeFrameFixture(t *testing.T) timeFrameFixture {
	t.Helper()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	email := uniqueEmail(t)
	volID := seedVolunteer(t, email, "Time", "Tester", "VOLUNTEER")
	token := seedSession(t, email, volID, "VOLUNTEER", "tf-"+suffix)

	jobID := getJobTypeID(t, "event_support")
	venueID := seedVenue(t, "TF Venue "+suffix, "1 Clock St", "Reno", "NV", "America/Los_Angeles")

	upcomingName := "UpcomingEvent-" + suffix
	pastName     := "PastEvent-"     + suffix

	upcomingID := seedEvent(t, upcomingName, false, &venueID)
	pastID     := seedEvent(t, pastName,     false, &venueID)

	seedEventDate(t, upcomingID, "2028-01-01T09:00:00Z", "2028-01-01T17:00:00Z")
	seedEventDate(t, pastID,     "2020-01-01T09:00:00Z", "2020-01-01T17:00:00Z")

	upOppID  := seedOpportunity(t, upcomingID, jobID, false)
	pastOppID := seedOpportunity(t, pastID, jobID, false)

	seedShift(t, upOppID,  "2028-01-01T09:00:00Z", "2028-01-01T12:00:00Z", 3)
	seedShift(t, pastOppID, "2020-01-01T09:00:00Z", "2020-01-01T12:00:00Z", 3)

	return timeFrameFixture{
		token:        token,
		upcomingName: upcomingName,
		pastName:     pastName,
	}
}

func TestFilteredEvents_TimeFrameUpcoming(t *testing.T) {
	fx := setupTimeFrameFixture(t)

	resp := gqlPost(t, "/graphql/volunteer", fx.token, queryFilteredEvents, map[string]any{
		"filter": map[string]any{
			"timeFrame": "UPCOMING",
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	got := eventNamesFromResponse(t, resp)

	if !got[fx.upcomingName] {
		t.Errorf("expected upcoming event %q in UPCOMING results", fx.upcomingName)
	}
	if got[fx.pastName] {
		t.Errorf("past event %q should not appear in UPCOMING results", fx.pastName)
	}
}

func TestFilteredEvents_TimeFramePast(t *testing.T) {
	fx := setupTimeFrameFixture(t)

	resp := gqlPost(t, "/graphql/volunteer", fx.token, queryFilteredEvents, map[string]any{
		"filter": map[string]any{
			"timeFrame": "PAST",
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	got := eventNamesFromResponse(t, resp)

	if !got[fx.pastName] {
		t.Errorf("expected past event %q in PAST results", fx.pastName)
	}
	if got[fx.upcomingName] {
		t.Errorf("upcoming event %q should not appear in PAST results", fx.upcomingName)
	}
}

func TestFilteredEvents_TimeFrameAll(t *testing.T) {
	fx := setupTimeFrameFixture(t)

	resp := gqlPost(t, "/graphql/volunteer", fx.token, queryFilteredEvents, map[string]any{
		"filter": map[string]any{
			"timeFrame": "ALL",
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	got := eventNamesFromResponse(t, resp)

	if !got[fx.upcomingName] {
		t.Errorf("expected upcoming event %q in ALL results", fx.upcomingName)
	}
	if !got[fx.pastName] {
		t.Errorf("expected past event %q in ALL results", fx.pastName)
	}
}

func TestFilteredEvents_CombinedCityAndTimeFrame(t *testing.T) {
	fx := setupTimeFrameFixture(t)

	// Reno is the city used by the timeFrame fixture venues.
	resp := gqlPost(t, "/graphql/volunteer", fx.token, queryFilteredEvents, map[string]any{
		"filter": map[string]any{
			"cities":    []string{"Reno"},
			"timeFrame": "UPCOMING",
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	got := eventNamesFromResponse(t, resp)

	if !got[fx.upcomingName] {
		t.Errorf("expected upcoming event %q in Reno+UPCOMING results", fx.upcomingName)
	}
	if got[fx.pastName] {
		t.Errorf("past event %q should not appear in Reno+UPCOMING results", fx.pastName)
	}
}
