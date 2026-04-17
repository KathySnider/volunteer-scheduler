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
		filteredEventsWithShifts(filter: $filter) {
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

	// ── venues ──────────────────────────────────────────────────────────────
	venue1ID := seedVenue(t, "Seattle Venue", "100 Pike St", "Seattle", "WA", "America/Los_Angeles")
	venue2ID := seedVenue(t, "Spokane Venue", "200 Monroe St", "Spokane", "WA", "America/Los_Angeles")

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
		jobEventSupportID: jobEventSupportID,
		jobAdvocacyID:     jobAdvocacyID,
	}
}

// ============================================================================
// Small assertion helpers
// ============================================================================

// eventNamesFromResponse unmarshals the filteredEventsWithShifts data field and returns
// a map of event name → true for quick membership checks.
func eventNamesFromResponse(t *testing.T, resp gqlResponse) map[string]bool {
	t.Helper()
	var events []struct {
		Name string `json:"name"`
	}
	unmarshalField(t, resp, "filteredEventsWithShifts", &events)
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
	unmarshalField(t, resp, "filteredEventsWithShifts", &events)
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

