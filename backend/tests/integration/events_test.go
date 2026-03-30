package integration

import (
	"fmt"
	"testing"
	"time"
)

// ============================================================================
// Shared GraphQL query string
// ============================================================================

const queryFilteredEvents = `
	query FilteredEvents($filter: EventFilterInput) {
		filteredEvents(filter: $filter) {
			id
			name
			eventType
		}
	}`

// ============================================================================
// Test fixture
// ============================================================================
//
// Three canonical events, created fresh for every test:
//
//   Event A  –  VIRTUAL     Apr-2026 shift   job=event_support
//   Event B  –  IN_PERSON   Jun-2026 shift   job=advocacy       venue→region1
//   Event C  –  HYBRID      Sep-2026 shift   job=event_support  venue→region2
//
// The fixture also mints a volunteer session token so tests can reach the
// authenticated /graphql/volunteer endpoint.

type eventTestFixture struct {
	token             string
	eventAName        string
	eventBName        string
	eventCName        string
	region1ID         int
	region2ID         int
	jobEventSupportID int
	jobAdvocacyID     int
}

// setupEventFixture seeds regions, venues, events, dates, opportunities, and
// shifts into the test database and returns the fixture. Every seeded row is
// automatically removed by t.Cleanup (LIFO — children before parents).
func setupEventFixture(t *testing.T) eventTestFixture {
	t.Helper()

	// Use a time-based suffix to keep region codes unique if any unique
	// constraint exists on that column.
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	// ── volunteer session ───────────────────────────────────────────────────
	email := uniqueEmail(t)
	volID := seedVolunteer(t, email, "Evelyn", "Filter", "VOLUNTEER")
	token := seedSession(t, email, volID, "VOLUNTEER", "evt-filter-"+suffix)

	// ── lookup IDs (seeded by migrations) ──────────────────────────────────
	jobEventSupportID := getJobTypeID(t, "event_support")
	jobAdvocacyID := getJobTypeID(t, "advocacy")

	// ── regions ─────────────────────────────────────────────────────────────
	// Register region cleanups first so that LIFO ensures they run after
	// all venues and events that reference them have been deleted.
	region1ID := seedRegion(t, "r1-"+suffix, "Test Region 1")
	region2ID := seedRegion(t, "r2-"+suffix, "Test Region 2")

	// ── venues ──────────────────────────────────────────────────────────────
	venue1ID := seedVenue(t, "Seattle Venue", "100 Pike St", "Seattle", "WA", "America/Los_Angeles")
	venue2ID := seedVenue(t, "Spokane Venue", "200 Monroe St", "Spokane", "WA", "America/Los_Angeles")

	// ── venue → region links ─────────────────────────────────────────────────
	seedVenueRegion(t, venue1ID, region1ID)
	seedVenueRegion(t, venue2ID, region2ID)

	// ── events ──────────────────────────────────────────────────────────────
	eventAName := "Virtual Event Apr-" + suffix
	eventBName := "InPerson Event Jun-" + suffix
	eventCName := "Hybrid Event Sep-" + suffix

	eventAID := seedEvent(t, eventAName, true, nil)        // VIRTUAL
	eventBID := seedEvent(t, eventBName, false, &venue1ID) // IN_PERSON
	eventCID := seedEvent(t, eventCName, true, &venue2ID)  // HYBRID

	// ── event dates (ORDER BY earliest.first_date in the query) ─────────────
	seedEventDate(t, eventAID, "2026-04-15T09:00:00Z", "2026-04-15T17:00:00Z")
	seedEventDate(t, eventBID, "2026-06-15T09:00:00Z", "2026-06-15T17:00:00Z")
	seedEventDate(t, eventCID, "2026-09-15T09:00:00Z", "2026-09-15T17:00:00Z")

	// ── opportunities ────────────────────────────────────────────────────────
	oppAID := seedOpportunity(t, eventAID, jobEventSupportID, true)
	oppBID := seedOpportunity(t, eventBID, jobAdvocacyID, false)
	oppCID := seedOpportunity(t, eventCID, jobEventSupportID, true)

	// ── shifts ───────────────────────────────────────────────────────────────
	seedShift(t, oppAID, "2026-04-15T09:00:00Z", "2026-04-15T12:00:00Z", 5)
	seedShift(t, oppBID, "2026-06-15T09:00:00Z", "2026-06-15T12:00:00Z", 5)
	seedShift(t, oppCID, "2026-09-15T09:00:00Z", "2026-09-15T12:00:00Z", 5)

	return eventTestFixture{
		token:             token,
		eventAName:        eventAName,
		eventBName:        eventBName,
		eventCName:        eventCName,
		region1ID:         region1ID,
		region2ID:         region2ID,
		jobEventSupportID: jobEventSupportID,
		jobAdvocacyID:     jobAdvocacyID,
	}
}

// ============================================================================
// Small assertion helpers
// ============================================================================

// eventNamesFromResponse unmarshals the filteredEvents data field and returns
// a map of event name → true for quick membership checks.
func eventNamesFromResponse(t *testing.T, resp gqlResponse) map[string]bool {
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

// assertEventCount fails the test when the response contains a number of
// events other than want.
func assertEventCount(t *testing.T, resp gqlResponse, want int) {
	t.Helper()
	var events []struct {
		Name string `json:"name"`
	}
	unmarshalField(t, resp, "filteredEvents", &events)
	if len(events) != want {
		names := make([]string, len(events))
		for i, e := range events {
			names[i] = e.Name
		}
		t.Errorf("expected %d event(s), got %d: %v", want, len(events), names)
	}
}

// ============================================================================
// Tests: no filter
// ============================================================================

// TestFilteredEvents_NoFilter verifies that omitting the filter returns all
// events (at least the three from the fixture).
func TestFilteredEvents_NoFilter(t *testing.T) {
	fx := setupEventFixture(t)

	resp := gqlPost(t, "/graphql/volunteer", fx.token, queryFilteredEvents, nil)

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	got := eventNamesFromResponse(t, resp)
	for _, name := range []string{fx.eventAName, fx.eventBName, fx.eventCName} {
		if !got[name] {
			t.Errorf("expected event %q in unfiltered results, but it was missing", name)
		}
	}
}

// ============================================================================
// Tests: region filter
// ============================================================================

// TestFilteredEvents_ByRegion1 verifies that filtering by region1 returns
// only the in-person event whose venue is in that region.
func TestFilteredEvents_ByRegion1(t *testing.T) {
	fx := setupEventFixture(t)

	resp := gqlPost(t, "/graphql/volunteer", fx.token, queryFilteredEvents, map[string]any{
		"filter": map[string]any{
			"regions": []int{fx.region1ID},
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	got := eventNamesFromResponse(t, resp)

	if !got[fx.eventBName] {
		t.Errorf("expected in-person event %q in region1 results", fx.eventBName)
	}
	if got[fx.eventAName] {
		t.Errorf("virtual event %q should not appear in region1 filter", fx.eventAName)
	}
	if got[fx.eventCName] {
		t.Errorf("hybrid event %q (region2) should not appear in region1 filter", fx.eventCName)
	}
}

// TestFilteredEvents_ByRegion2 verifies that filtering by region2 returns
// only the hybrid event whose venue is in that region.
func TestFilteredEvents_ByRegion2(t *testing.T) {
	fx := setupEventFixture(t)

	resp := gqlPost(t, "/graphql/volunteer", fx.token, queryFilteredEvents, map[string]any{
		"filter": map[string]any{
			"regions": []int{fx.region2ID},
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	got := eventNamesFromResponse(t, resp)

	if !got[fx.eventCName] {
		t.Errorf("expected hybrid event %q in region2 results", fx.eventCName)
	}
	if got[fx.eventAName] {
		t.Errorf("virtual event %q should not appear in region2 filter", fx.eventAName)
	}
	if got[fx.eventBName] {
		t.Errorf("in-person event %q (region1) should not appear in region2 filter", fx.eventBName)
	}
}

// ============================================================================
// Tests: event-type filter
// ============================================================================

// TestFilteredEvents_ByTypeVirtual verifies that VIRTUAL returns only Event A.
func TestFilteredEvents_ByTypeVirtual(t *testing.T) {
	fx := setupEventFixture(t)

	resp := gqlPost(t, "/graphql/volunteer", fx.token, queryFilteredEvents, map[string]any{
		"filter": map[string]any{
			"eventType": "VIRTUAL",
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	got := eventNamesFromResponse(t, resp)

	if !got[fx.eventAName] {
		t.Errorf("expected virtual event %q in VIRTUAL filter results", fx.eventAName)
	}
	if got[fx.eventBName] {
		t.Errorf("in-person event %q should not appear in VIRTUAL filter", fx.eventBName)
	}
	if got[fx.eventCName] {
		t.Errorf("hybrid event %q should not appear in VIRTUAL filter", fx.eventCName)
	}
}

// TestFilteredEvents_ByTypeInPerson verifies that IN_PERSON returns only Event B.
func TestFilteredEvents_ByTypeInPerson(t *testing.T) {
	fx := setupEventFixture(t)

	resp := gqlPost(t, "/graphql/volunteer", fx.token, queryFilteredEvents, map[string]any{
		"filter": map[string]any{
			"eventType": "IN_PERSON",
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	got := eventNamesFromResponse(t, resp)

	if !got[fx.eventBName] {
		t.Errorf("expected in-person event %q in IN_PERSON filter results", fx.eventBName)
	}
	if got[fx.eventAName] {
		t.Errorf("virtual event %q should not appear in IN_PERSON filter", fx.eventAName)
	}
	if got[fx.eventCName] {
		t.Errorf("hybrid event %q should not appear in IN_PERSON filter", fx.eventCName)
	}
}

// TestFilteredEvents_ByTypeHybrid verifies that HYBRID returns only Event C.
func TestFilteredEvents_ByTypeHybrid(t *testing.T) {
	fx := setupEventFixture(t)

	resp := gqlPost(t, "/graphql/volunteer", fx.token, queryFilteredEvents, map[string]any{
		"filter": map[string]any{
			"eventType": "HYBRID",
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	got := eventNamesFromResponse(t, resp)

	if !got[fx.eventCName] {
		t.Errorf("expected hybrid event %q in HYBRID filter results", fx.eventCName)
	}
	if got[fx.eventAName] {
		t.Errorf("virtual event %q should not appear in HYBRID filter", fx.eventAName)
	}
	if got[fx.eventBName] {
		t.Errorf("in-person event %q should not appear in HYBRID filter", fx.eventBName)
	}
}

// ============================================================================
// Tests: job filter
// ============================================================================

// TestFilteredEvents_ByJobEventSupport verifies that filtering by event_support
// returns Events A and C (both use that job type) but not B.
func TestFilteredEvents_ByJobEventSupport(t *testing.T) {
	fx := setupEventFixture(t)

	resp := gqlPost(t, "/graphql/volunteer", fx.token, queryFilteredEvents, map[string]any{
		"filter": map[string]any{
			"jobs": []int{fx.jobEventSupportID},
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	got := eventNamesFromResponse(t, resp)

	if !got[fx.eventAName] {
		t.Errorf("expected virtual event %q with event_support job in results", fx.eventAName)
	}
	if !got[fx.eventCName] {
		t.Errorf("expected hybrid event %q with event_support job in results", fx.eventCName)
	}
	if got[fx.eventBName] {
		t.Errorf("in-person event %q (advocacy job) should not appear in event_support filter", fx.eventBName)
	}
}

// TestFilteredEvents_ByJobAdvocacy verifies that filtering by advocacy returns
// only Event B.
func TestFilteredEvents_ByJobAdvocacy(t *testing.T) {
	fx := setupEventFixture(t)

	resp := gqlPost(t, "/graphql/volunteer", fx.token, queryFilteredEvents, map[string]any{
		"filter": map[string]any{
			"jobs": []int{fx.jobAdvocacyID},
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	got := eventNamesFromResponse(t, resp)

	if !got[fx.eventBName] {
		t.Errorf("expected in-person event %q with advocacy job in results", fx.eventBName)
	}
	if got[fx.eventAName] {
		t.Errorf("virtual event %q (event_support) should not appear in advocacy filter", fx.eventAName)
	}
	if got[fx.eventCName] {
		t.Errorf("hybrid event %q (event_support) should not appear in advocacy filter", fx.eventCName)
	}
}

// ============================================================================
// Tests: shift date filters
// ============================================================================
//
// Shift dates for reference:
//   Event A  shift_start=2026-04-15  shift_end=2026-04-15T12:00Z
//   Event B  shift_start=2026-06-15  shift_end=2026-06-15T12:00Z
//   Event C  shift_start=2026-09-15  shift_end=2026-09-15T12:00Z

// TestFilteredEvents_ByShiftStartDate verifies that a shiftStartDate set to
// mid-May returns only the events whose shifts start on or after that date
// (Jun and Sep → Events B and C).
func TestFilteredEvents_ByShiftStartDate(t *testing.T) {
	fx := setupEventFixture(t)

	resp := gqlPost(t, "/graphql/volunteer", fx.token, queryFilteredEvents, map[string]any{
		"filter": map[string]any{
			"shiftStartDateTime": "2026-05-01 00:00:00",
			"ianaZone":           "UTC",
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	got := eventNamesFromResponse(t, resp)

	if !got[fx.eventBName] {
		t.Errorf("expected Jun event %q to match shiftStartDateTime >= May 1", fx.eventBName)
	}
	if !got[fx.eventCName] {
		t.Errorf("expected Sep event %q to match shiftStartDateTime >= May 1", fx.eventCName)
	}
	if got[fx.eventAName] {
		t.Errorf("Apr event %q should not match shiftStartDateTime >= May 1", fx.eventAName)
	}
}

// TestFilteredEvents_ByShiftEndDate verifies that a shiftEndDate set to
// August 1 returns only the events whose shifts fall entirely before that
// cutoff (Apr and Jun → Events A and B).
func TestFilteredEvents_ByShiftEndDate(t *testing.T) {
	fx := setupEventFixture(t)

	resp := gqlPost(t, "/graphql/volunteer", fx.token, queryFilteredEvents, map[string]any{
		"filter": map[string]any{
			"shiftEndDateTime": "2026-08-01 00:00:00",
			"ianaZone":         "UTC",
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	got := eventNamesFromResponse(t, resp)

	if !got[fx.eventAName] {
		t.Errorf("expected Apr event %q to match shiftEndDateTime <= Aug 1", fx.eventAName)
	}
	if !got[fx.eventBName] {
		t.Errorf("expected Jun event %q to match shiftEndDateTime <= Aug 1", fx.eventBName)
	}
	if got[fx.eventCName] {
		t.Errorf("Sep event %q should not match shiftEndDateTime <= Aug 1", fx.eventCName)
	}
}

// TestFilteredEvents_ByDateRange verifies that combining shiftStartDateTime and
// shiftEndDateTime to bracket only June returns a single event (Event B).
func TestFilteredEvents_ByDateRange(t *testing.T) {
	fx := setupEventFixture(t)

	resp := gqlPost(t, "/graphql/volunteer", fx.token, queryFilteredEvents, map[string]any{
		"filter": map[string]any{
			"shiftStartDateTime": "2026-05-01 00:00:00",
			"shiftEndDateTime":   "2026-07-31 23:59:59",
			"ianaZone":           "UTC",
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	got := eventNamesFromResponse(t, resp)

	if !got[fx.eventBName] {
		t.Errorf("expected Jun event %q to match May–Jul date range", fx.eventBName)
	}
	if got[fx.eventAName] {
		t.Errorf("Apr event %q should not match May–Jul date range", fx.eventAName)
	}
	if got[fx.eventCName] {
		t.Errorf("Sep event %q should not match May–Jul date range", fx.eventCName)
	}
}

// ============================================================================
// Tests: combined filters
// ============================================================================

// TestFilteredEvents_CombinedRegionAndJob verifies that applying both a
// region filter and a job filter narrows results to the single event that
// satisfies both (Event B: region1 AND advocacy).
func TestFilteredEvents_CombinedRegionAndJob(t *testing.T) {
	fx := setupEventFixture(t)

	resp := gqlPost(t, "/graphql/volunteer", fx.token, queryFilteredEvents, map[string]any{
		"filter": map[string]any{
			"regions": []int{fx.region1ID},
			"jobs":    []int{fx.jobAdvocacyID},
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	got := eventNamesFromResponse(t, resp)

	if !got[fx.eventBName] {
		t.Errorf("expected event %q to match region1+advocacy combined filter", fx.eventBName)
	}
	if got[fx.eventAName] {
		t.Errorf("virtual event %q should not match region1+advocacy filter", fx.eventAName)
	}
	if got[fx.eventCName] {
		t.Errorf("hybrid event %q (region2) should not match region1+advocacy filter", fx.eventCName)
	}
}

// ============================================================================
// Tests: no results
// ============================================================================

// TestFilteredEvents_NoResults verifies that a filter combination that
// matches no events returns an empty list without errors.
// Region1 contains only an in-person event; asking for VIRTUAL within
// region1 yields nothing.
func TestFilteredEvents_NoResults(t *testing.T) {
	fx := setupEventFixture(t)

	resp := gqlPost(t, "/graphql/volunteer", fx.token, queryFilteredEvents, map[string]any{
		"filter": map[string]any{
			"regions":   []int{fx.region1ID},
			"eventType": "VIRTUAL",
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	got := eventNamesFromResponse(t, resp)

	if got[fx.eventAName] || got[fx.eventBName] || got[fx.eventCName] {
		t.Errorf("expected empty result set for region1+VIRTUAL filter, got: %v", got)
	}
}
