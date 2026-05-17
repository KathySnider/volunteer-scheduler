package integration

// ============================================================================
// Integration tests — distance filter for volunteer event search
// ============================================================================
//
// All tests use real Washington State coordinates so the Haversine math is
// exercised with realistic values:
//
//   Volunteer  Seattle    lat=47.6062  lng=-122.3321
//   Near venue Tacoma     lat=47.2529  lng=-122.4443  (~27 mi — inside 50 mi)
//   Far venue  Spokane    lat=47.6588  lng=-117.4260  (~223 mi — outside 50 mi)
//
// Lat/lng are written directly to the DB after seeding so tests never call
// the external Census Geocoder API.

import (
	"fmt"
	"testing"
	"time"
)

// ── coordinates ──────────────────────────────────────────────────────────────

const (
	seattleLat = 47.6062
	seattleLng = -122.3321
	tacomaLat  = 47.2529
	tacomaLng  = -122.4443
	spokaneLat = 47.6588
	spokaneLng = -117.4260
)

// ── fixture ───────────────────────────────────────────────────────────────────

type distanceTestFixture struct {
	token        string
	volID        int
	nearEventName string // IN_PERSON at Tacoma (~27 mi)
	farEventName  string // IN_PERSON at Spokane (~223 mi)
	virtualName   string // VIRTUAL — no venue
	hybridName    string // HYBRID at Spokane (~223 mi) — always included
}

// setupDistanceFixture seeds a volunteer (with Seattle coordinates), two
// in-person events (near and far), one virtual event, and one hybrid event
// (at the far venue). All rows are cleaned up via t.Cleanup.
func setupDistanceFixture(t *testing.T) distanceTestFixture {
	t.Helper()

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	// ── volunteer ────────────────────────────────────────────────────────────
	email := uniqueEmail(t)
	volID := seedVolunteer(t, email, "Dist", "Tester", "VOLUNTEER")
	token := seedSession(t, email, volID, "VOLUNTEER", "dist-"+suffix)

	// Set volunteer coordinates directly — no geocoding API call.
	_, err := testDB.Exec(
		"UPDATE volunteers SET zip_code = '98101', latitude = $1, longitude = $2 WHERE volunteer_id = $3",
		seattleLat, seattleLng, volID,
	)
	if err != nil {
		t.Fatalf("setupDistanceFixture: set volunteer coords: %v", err)
	}

	// ── venues ───────────────────────────────────────────────────────────────
	nearVenueID := seedVenue(t, "Tacoma Venue "+suffix, "1 Commerce St", "Tacoma", "WA", "America/Los_Angeles")
	farVenueID  := seedVenue(t, "Spokane Venue "+suffix, "1 Riverside Ave", "Spokane", "WA", "America/Los_Angeles")

	_, err = testDB.Exec(
		"UPDATE venues SET latitude = $1, longitude = $2 WHERE venue_id = $3",
		tacomaLat, tacomaLng, nearVenueID,
	)
	if err != nil {
		t.Fatalf("setupDistanceFixture: set near venue coords: %v", err)
	}
	_, err = testDB.Exec(
		"UPDATE venues SET latitude = $1, longitude = $2 WHERE venue_id = $3",
		spokaneLat, spokaneLng, farVenueID,
	)
	if err != nil {
		t.Fatalf("setupDistanceFixture: set far venue coords: %v", err)
	}

	// ── events ───────────────────────────────────────────────────────────────
	jobID := getJobTypeID(t, "event_support")

	nearName    := "NearEvent-" + suffix
	farName     := "FarEvent-" + suffix
	virtualName := "VirtualEvent-" + suffix
	hybridName  := "HybridEvent-" + suffix

	nearID    := seedEvent(t, nearName,    false, &nearVenueID) // IN_PERSON Tacoma
	farID     := seedEvent(t, farName,     false, &farVenueID)  // IN_PERSON Spokane
	virtualID := seedEvent(t, virtualName, true,  nil)          // VIRTUAL
	hybridID  := seedEvent(t, hybridName,  true,  &farVenueID)  // HYBRID Spokane

	// ── event dates ──────────────────────────────────────────────────────────
	seedEventDate(t, nearID,    "2028-08-01T09:00:00Z", "2028-08-01T17:00:00Z")
	seedEventDate(t, farID,     "2028-08-02T09:00:00Z", "2028-08-02T17:00:00Z")
	seedEventDate(t, virtualID, "2028-08-03T09:00:00Z", "2028-08-03T17:00:00Z")
	seedEventDate(t, hybridID,  "2028-08-04T09:00:00Z", "2028-08-04T17:00:00Z")

	// ── opportunities + shifts (required for volunteer view) ─────────────────
	seedShift(t, seedOpportunity(t, nearID,    jobID, false), "2028-08-01T09:00:00Z", "2028-08-01T12:00:00Z", 5)
	seedShift(t, seedOpportunity(t, farID,     jobID, false), "2028-08-02T09:00:00Z", "2028-08-02T12:00:00Z", 5)
	seedShift(t, seedOpportunity(t, virtualID, jobID, true),  "2028-08-03T09:00:00Z", "2028-08-03T12:00:00Z", 5)
	seedShift(t, seedOpportunity(t, hybridID,  jobID, true),  "2028-08-04T09:00:00Z", "2028-08-04T12:00:00Z", 5)

	return distanceTestFixture{
		token:         token,
		volID:         volID,
		nearEventName: nearName,
		farEventName:  farName,
		virtualName:   virtualName,
		hybridName:    hybridName,
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func distanceFilter(miles int) map[string]any {
	return map[string]any{
		"filter": map[string]any{
			"distance": miles,
		},
	}
}

// ============================================================================
// TestDistanceFilter_NearbyInPersonIncluded
// ============================================================================

// An in-person event at a venue within the radius must appear in results.
func TestDistanceFilter_NearbyInPersonIncluded(t *testing.T) {
	fx := setupDistanceFixture(t)

	resp := gqlPost(t, "/graphql/volunteer", fx.token, queryFilteredEvents, distanceFilter(50))

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	got := eventNamesFromResponse(t, resp)
	if !got[fx.nearEventName] {
		t.Errorf("expected near in-person event %q within 50 mi to be included", fx.nearEventName)
	}
}

// ============================================================================
// TestDistanceFilter_FarInPersonExcluded
// ============================================================================

// An in-person event at a venue outside the radius must be excluded.
func TestDistanceFilter_FarInPersonExcluded(t *testing.T) {
	fx := setupDistanceFixture(t)

	resp := gqlPost(t, "/graphql/volunteer", fx.token, queryFilteredEvents, distanceFilter(50))

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	got := eventNamesFromResponse(t, resp)
	if got[fx.farEventName] {
		t.Errorf("far in-person event %q (~223 mi) should be excluded from 50 mi radius", fx.farEventName)
	}
}

// ============================================================================
// TestDistanceFilter_VirtualAlwaysIncluded
// ============================================================================

// Virtual events have no venue so they must always pass the distance filter.
func TestDistanceFilter_VirtualAlwaysIncluded(t *testing.T) {
	fx := setupDistanceFixture(t)

	resp := gqlPost(t, "/graphql/volunteer", fx.token, queryFilteredEvents, distanceFilter(50))

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	got := eventNamesFromResponse(t, resp)
	if !got[fx.virtualName] {
		t.Errorf("virtual event %q should always be included regardless of distance filter", fx.virtualName)
	}
}

// ============================================================================
// TestDistanceFilter_HybridAlwaysIncluded
// ============================================================================

// Hybrid events are always included — volunteers can attend virtually even if
// the venue is far away.  This event is at the far Spokane venue (~223 mi).
func TestDistanceFilter_HybridAlwaysIncluded(t *testing.T) {
	fx := setupDistanceFixture(t)

	resp := gqlPost(t, "/graphql/volunteer", fx.token, queryFilteredEvents, distanceFilter(50))

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	got := eventNamesFromResponse(t, resp)
	if !got[fx.hybridName] {
		t.Errorf("hybrid event %q should always be included regardless of distance filter", fx.hybridName)
	}
}

// ============================================================================
// TestDistanceFilter_NoVolunteerCoords_FilterIgnored
// ============================================================================

// If the volunteer has no coordinates, the distance filter is silently ignored
// and all events are returned.
func TestDistanceFilter_NoVolunteerCoords_FilterIgnored(t *testing.T) {
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	// Seed a volunteer with NO lat/lng.
	email := uniqueEmail(t)
	volID := seedVolunteer(t, email, "NoCoord", "Vol", "VOLUNTEER")
	token := seedSession(t, email, volID, "VOLUNTEER", "nocoord-"+suffix)

	jobID := getJobTypeID(t, "event_support")

	nearVenueID := seedVenue(t, "NoCoord Near "+suffix, "1 Main St", "Tacoma", "WA", "America/Los_Angeles")
	farVenueID  := seedVenue(t, "NoCoord Far "+suffix,  "2 Main St", "Spokane", "WA", "America/Los_Angeles")

	// Set venue coords so the distance calc would work if it ran.
	testDB.Exec("UPDATE venues SET latitude = $1, longitude = $2 WHERE venue_id = $3", tacomaLat, tacomaLng, nearVenueID)
	testDB.Exec("UPDATE venues SET latitude = $1, longitude = $2 WHERE venue_id = $3", spokaneLat, spokaneLng, farVenueID)

	nearName := "NoCoordNear-" + suffix
	farName  := "NoCoordFar-" + suffix

	nearID := seedEvent(t, nearName, false, &nearVenueID)
	farID  := seedEvent(t, farName,  false, &farVenueID)

	seedEventDate(t, nearID, "2028-09-01T09:00:00Z", "2028-09-01T17:00:00Z")
	seedEventDate(t, farID,  "2028-09-02T09:00:00Z", "2028-09-02T17:00:00Z")

	seedShift(t, seedOpportunity(t, nearID, jobID, false), "2028-09-01T09:00:00Z", "2028-09-01T12:00:00Z", 5)
	seedShift(t, seedOpportunity(t, farID,  jobID, false), "2028-09-02T09:00:00Z", "2028-09-02T12:00:00Z", 5)

	resp := gqlPost(t, "/graphql/volunteer", token, queryFilteredEvents, distanceFilter(50))

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	got := eventNamesFromResponse(t, resp)
	if !got[nearName] {
		t.Errorf("near event %q should be included when volunteer has no coordinates", nearName)
	}
	if !got[farName] {
		t.Errorf("far event %q should be included when volunteer has no coordinates (filter ignored)", farName)
	}
}

// ============================================================================
// TestDistanceFilter_CityFilterSkippedWhenDistanceSet
// ============================================================================

// When a distance filter is active, the city filter must be ignored — otherwise
// a volunteer in Seattle filtering by distance would get no results if the city
// list happened to not include their city.
func TestDistanceFilter_CityFilterSkippedWhenDistanceSet(t *testing.T) {
	fx := setupDistanceFixture(t)

	// Pass both distance and a city filter for a city that does NOT match any
	// seeded venue.  The city filter should be ignored and the near event should
	// still appear.
	resp := gqlPost(t, "/graphql/volunteer", fx.token, queryFilteredEvents, map[string]any{
		"filter": map[string]any{
			"distance": 50,
			"cities":   []string{"Olympia"},
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	got := eventNamesFromResponse(t, resp)
	if !got[fx.nearEventName] {
		t.Errorf("near event %q should appear when distance is set even though city filter would exclude it", fx.nearEventName)
	}
}
