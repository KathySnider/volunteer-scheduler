package integration

import (
	"fmt"
	"testing"
)

// ============================================================================
// Mutation strings
// ============================================================================

const (
	mutCreateVenue = `
		mutation CreateVenue($input: NewVenueInput!) {
			createVenue(newVenue: $input) { success message id }
		}`

	mutUpdateVenue = `
		mutation UpdateVenue($input: UpdateVenueInput!) {
			updateVenue(venue: $input) { success message id }
		}`

	mutDeleteVenue = `
		mutation DeleteVenue($id: ID!) {
			deleteVenue(venueId: $id) { success message id }
		}`

	mutCreateRegion = `
		mutation CreateRegion($input: NewRegionInput!) {
			createRegion(newRegion: $input) { success message id }
		}`

	mutUpdateRegion = `
		mutation UpdateRegion($input: UpdateRegionInput!) {
			updateRegion(region: $input) { success message id }
		}`

	mutDeleteRegion = `
		mutation DeleteRegion($id: Int!) {
			deleteRegion(regionId: $id) { success message id }
		}`

	mutAddVenueRegion = `
		mutation AddVenueRegion($venueId: Int!, $regionId: Int!) {
			addVenueRegion(venueId: $venueId, regionId: $regionId) { success message id }
		}`

	mutRemoveVenueRegion = `
		mutation RemoveVenueRegion($venueId: Int!, $regionId: Int!) {
			removeVenueRegion(venueId: $venueId, regionId: $regionId) { success message id }
		}`
)

// ============================================================================
// createVenue
// ============================================================================

// TestCreateVenue verifies that a new venue can be created via the admin
// mutation and that the returned ID maps to a row in the DB.
// The service requires at least one region, so we seed one and pass it.
func TestCreateVenue(t *testing.T) {
	token := makeAdminToken(t)
	regionID := seedRegion(t, uniqueCode(t, "vcr"), "Venue Create Region")

	resp := gqlPost(t, "/graphql/admin", token, mutCreateVenue, map[string]any{
		"input": map[string]any{
			"name":     "CRUD Test Venue",
			"address":  "100 Test Blvd",
			"city":     "Portland",
			"state":    "OR",
			"ianaZone": "America/Los_Angeles",
			"region":   []int{regionID},
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "createVenue", &result)

	if !result.Success {
		var msg string
		if result.Message != nil {
			msg = *result.Message
		}
		t.Fatalf("createVenue returned success=false: %s", msg)
	}
	if result.ID == nil || *result.ID == "" {
		t.Fatal("expected a non-empty venue ID in response")
	}

	venueID := *result.ID
	t.Cleanup(func() { testDB.Exec("DELETE FROM venues WHERE venue_id = $1", venueID) })

	if !rowExists(t, "SELECT COUNT(*) FROM venues WHERE venue_id = $1", venueID) {
		t.Errorf("expected venue row in DB for id=%s", venueID)
	}
}

// ============================================================================
// updateVenue
// ============================================================================

// TestUpdateVenue verifies that a venue's name can be changed and the update
// is persisted to the DB.
func TestUpdateVenue(t *testing.T) {
	token := makeAdminToken(t)
	venueID := seedVenue(t, "Pre-Update Venue", "1 Old St", "Salem", "OR", "America/Los_Angeles")

	resp := gqlPost(t, "/graphql/admin", token, mutUpdateVenue, map[string]any{
		"input": map[string]any{
			"id":       fmt.Sprintf("%d", venueID),
			"name":     "Post-Update Venue",
			"address":  "2 New Ave",
			"city":     "Salem",
			"state":    "OR",
			"ianaZone": "America/Los_Angeles",
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "updateVenue", &result)

	if !result.Success {
		t.Fatalf("updateVenue returned success=false: %v", result.Message)
	}

	var name string
	if err := testDB.QueryRow(
		"SELECT venue_name FROM venues WHERE venue_id = $1", venueID,
	).Scan(&name); err != nil {
		t.Fatalf("querying updated venue: %v", err)
	}
	if name != "Post-Update Venue" {
		t.Errorf("expected venue_name='Post-Update Venue', got %q", name)
	}
}

// ============================================================================
// deleteVenue
// ============================================================================

// TestDeleteVenue verifies that deleteVenue removes the venue row from the DB.
func TestDeleteVenue(t *testing.T) {
	token := makeAdminToken(t)
	venueID := seedVenue(t, "Venue To Delete", "9 Gone Rd", "Eugene", "OR", "America/Los_Angeles")

	resp := gqlPost(t, "/graphql/admin", token, mutDeleteVenue, map[string]any{
		"id": fmt.Sprintf("%d", venueID),
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "deleteVenue", &result)

	if !result.Success {
		t.Fatalf("deleteVenue returned success=false: %v", result.Message)
	}

	if rowExists(t, "SELECT COUNT(*) FROM venues WHERE venue_id = $1", venueID) {
		t.Error("expected venue to be gone from the DB after deleteVenue")
	}
}

// ============================================================================
// createRegion
// ============================================================================

// TestCreateRegion verifies that a new region can be created via the admin
// mutation and that the returned ID maps to a row in the DB.
func TestCreateRegion(t *testing.T) {
	token := makeAdminToken(t)
	code := uniqueCode(t, "rgn")

	resp := gqlPost(t, "/graphql/admin", token, mutCreateRegion, map[string]any{
		"input": map[string]any{
			"code": code,
			"name": "CRUD Test Region",
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "createRegion", &result)

	if !result.Success {
		t.Fatalf("createRegion returned success=false: %v", result.Message)
	}
	if result.ID == nil || *result.ID == "" {
		t.Fatal("expected a non-empty region ID in response")
	}

	regionID := *result.ID
	t.Cleanup(func() { testDB.Exec("DELETE FROM regions WHERE region_id = $1", regionID) })

	if !rowExists(t, "SELECT COUNT(*) FROM regions WHERE region_id = $1", regionID) {
		t.Errorf("expected region row in DB for id=%s", regionID)
	}
}

// ============================================================================
// updateRegion
// ============================================================================

// TestUpdateRegion verifies that a region's name can be changed and the update
// is persisted to the DB.
func TestUpdateRegion(t *testing.T) {
	token := makeAdminToken(t)
	regionID := seedRegion(t, uniqueCode(t, "upd"), "Pre-Update Region")

	resp := gqlPost(t, "/graphql/admin", token, mutUpdateRegion, map[string]any{
		"input": map[string]any{
			"id":   regionID,
			"name": "Post-Update Region",
			"code": uniqueCode(t, "upd2"),
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "updateRegion", &result)

	if !result.Success {
		t.Fatalf("updateRegion returned success=false: %v", result.Message)
	}

	var name string
	if err := testDB.QueryRow(
		"SELECT name FROM regions WHERE region_id = $1", regionID,
	).Scan(&name); err != nil {
		t.Fatalf("querying updated region: %v", err)
	}
	if name != "Post-Update Region" {
		t.Errorf("expected name='Post-Update Region', got %q", name)
	}
}

// ============================================================================
// deleteRegion
// ============================================================================

// TestDeleteRegion verifies that deleteRegion removes the region row from the DB.
func TestDeleteRegion(t *testing.T) {
	token := makeAdminToken(t)
	regionID := seedRegion(t, uniqueCode(t, "del"), "Region To Delete")

	resp := gqlPost(t, "/graphql/admin", token, mutDeleteRegion, map[string]any{
		"id": regionID,
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "deleteRegion", &result)

	if !result.Success {
		t.Fatalf("deleteRegion returned success=false: %v", result.Message)
	}

	if rowExists(t, "SELECT COUNT(*) FROM regions WHERE region_id = $1 AND is_active = TRUE", regionID) {
		t.Error("expected region to be marked inactive after deleteRegion")
	}
}

// ============================================================================
// addVenueRegion / removeVenueRegion
// ============================================================================

// TestAddVenueRegion verifies that an admin can link a region to a venue and
// that the venue_regions join row is created in the DB.
func TestAddVenueRegion(t *testing.T) {
	token := makeAdminToken(t)
	venueID := seedVenue(t, "VR Link Venue", "10 Link Ln", "Bend", "OR", "America/Los_Angeles")
	regionID := seedRegion(t, uniqueCode(t, "vr"), "VR Link Region")

	// Cleanup the join row before the venue/region rows are deleted (LIFO).
	t.Cleanup(func() {
		testDB.Exec(
			"DELETE FROM venue_regions WHERE venue_id = $1 AND region_id = $2",
			venueID, regionID,
		)
	})

	resp := gqlPost(t, "/graphql/admin", token, mutAddVenueRegion, map[string]any{
		"venueId":  venueID,
		"regionId": regionID,
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "addVenueRegion", &result)

	if !result.Success {
		t.Fatalf("addVenueRegion returned success=false: %v", result.Message)
	}

	if !rowExists(t,
		"SELECT COUNT(*) FROM venue_regions WHERE venue_id = $1 AND region_id = $2",
		venueID, regionID,
	) {
		t.Error("expected venue_regions row after addVenueRegion")
	}
}

// TestRemoveVenueRegion verifies that an admin can unlink a region from a
// venue and that the venue_regions join row is removed from the DB.
// The service enforces "venue must have at least one region", so we seed
// two regions and only remove one.
func TestRemoveVenueRegion(t *testing.T) {
	token := makeAdminToken(t)
	venueID := seedVenue(t, "VR Unlink Venue", "11 Unlink Ln", "Bend", "OR", "America/Los_Angeles")
	regionID := seedRegion(t, uniqueCode(t, "vru"), "VR Unlink Region")
	keepRegionID := seedRegion(t, uniqueCode(t, "vrk"), "VR Keep Region")

	// Seed both links. LIFO cleanup order: both venue_regions rows are deleted
	// before either region row, and both region rows before the venue row.
	seedVenueRegion(t, venueID, regionID)
	seedVenueRegion(t, venueID, keepRegionID)

	resp := gqlPost(t, "/graphql/admin", token, mutRemoveVenueRegion, map[string]any{
		"venueId":  venueID,
		"regionId": regionID,
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "removeVenueRegion", &result)

	if !result.Success {
		var msg string
		if result.Message != nil {
			msg = *result.Message
		}
		t.Fatalf("removeVenueRegion returned success=false: %s", msg)
	}

	if rowExists(t,
		"SELECT COUNT(*) FROM venue_regions WHERE venue_id = $1 AND region_id = $2",
		venueID, regionID,
	) {
		t.Error("expected venue_regions row to be gone after removeVenueRegion")
	}
}
