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

	mutCreateFundingEntity = `
		mutation CreateFundingEntity($input: NewFundingEntityInput!) {
			createFundingEntity(input: $input) { success message id }
		}`

	mutUpdateFundingEntity = `
		mutation UpdateFundingEntity($input: UpdateFundingEntityInput!) {
			updateFundingEntity(input: $input) { success message id }
		}`

	mutDeleteFundingEntity = `
		mutation DeleteFundingEntity($id: Int!) {
			deleteFundingEntity(id: $id) { success message id }
		}`
)

// ============================================================================
// createVenue
// ============================================================================

// TestCreateVenue verifies that a new venue can be created via the admin
// mutation and that the returned ID maps to a row in the DB with the correct
// fields, including the optional zipCode used by the frontend venue cache.
func TestCreateVenue(t *testing.T) {
	token := makeAdminToken(t)

	resp := gqlPost(t, "/graphql/admin", token, mutCreateVenue, map[string]any{
		"input": map[string]any{
			"name":    "CRUD Test Venue",
			"address": "1221 SW 4th Ave",
			"city":    "Portland",
			"state":   "OR",
			"zipCode": "97204",
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

	// Verify zipCode was persisted — it's a field the frontend cache queries.
	var zip string
	if err := testDB.QueryRow(
		"SELECT COALESCE(zip_code, '') FROM venues WHERE venue_id = $1", venueID,
	).Scan(&zip); err != nil {
		t.Fatalf("querying zip_code: %v", err)
	}
	if zip != "97204" {
		t.Errorf("zip_code: want %q, got %q", "97204", zip)
	}
}

// ============================================================================
// updateVenue
// ============================================================================

// TestUpdateVenue verifies that a venue's name can be changed and the update
// is persisted to the DB.
func TestUpdateVenue(t *testing.T) {
	token := makeAdminToken(t)
	venueID := seedVenue(t, "Pre-Update Venue", "1 Old St", "Salem", "OR")

	resp := gqlPost(t, "/graphql/admin", token, mutUpdateVenue, map[string]any{
		"input": map[string]any{
			"id":      fmt.Sprintf("%d", venueID),
			"name":    "Post-Update Venue",
			"address": "900 Court St NE",
			"city":    "Salem",
			"state":   "OR",
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

// TestUpdateVenue_ZipCode verifies that zipCode can be set and later changed
// via updateVenue — it is an optional field used by the frontend venue cache.
func TestUpdateVenue_ZipCode(t *testing.T) {
	token := makeAdminToken(t)
	// Seed a venue without a zip code.
	venueID := seedVenue(t, "Zip Update Venue", "10 Pine St", "Bend", "OR")

	// First update: set a zip code.
	resp := gqlPost(t, "/graphql/admin", token, mutUpdateVenue, map[string]any{
		"input": map[string]any{
			"id":      fmt.Sprintf("%d", venueID),
			"name":    "Zip Update Venue",
			"address": "10 Pine St",
			"city":    "Bend",
			"state":   "OR",
			"zipCode": "97701",
		},
	})
	if hasGQLErrors(resp) {
		t.Fatalf("set zipCode: unexpected errors: %v", resp.Errors)
	}
	var r1 mutationResult
	unmarshalField(t, resp, "updateVenue", &r1)
	if !r1.Success {
		t.Fatalf("set zipCode: updateVenue returned success=false: %v", r1.Message)
	}

	var zip string
	if err := testDB.QueryRow(
		"SELECT COALESCE(zip_code, '') FROM venues WHERE venue_id = $1", venueID,
	).Scan(&zip); err != nil {
		t.Fatalf("querying zip_code after set: %v", err)
	}
	if zip != "97701" {
		t.Errorf("after set: zip_code want %q, got %q", "97701", zip)
	}

	// Second update: change the zip code.
	resp2 := gqlPost(t, "/graphql/admin", token, mutUpdateVenue, map[string]any{
		"input": map[string]any{
			"id":      fmt.Sprintf("%d", venueID),
			"name":    "Zip Update Venue",
			"address": "10 Pine St",
			"city":    "Bend",
			"state":   "OR",
			"zipCode": "97702",
		},
	})
	if hasGQLErrors(resp2) {
		t.Fatalf("change zipCode: unexpected errors: %v", resp2.Errors)
	}
	var r2 mutationResult
	unmarshalField(t, resp2, "updateVenue", &r2)
	if !r2.Success {
		t.Fatalf("change zipCode: updateVenue returned success=false: %v", r2.Message)
	}

	if err := testDB.QueryRow(
		"SELECT COALESCE(zip_code, '') FROM venues WHERE venue_id = $1", venueID,
	).Scan(&zip); err != nil {
		t.Fatalf("querying zip_code after change: %v", err)
	}
	if zip != "97702" {
		t.Errorf("after change: zip_code want %q, got %q", "97702", zip)
	}
}

// ============================================================================
// deleteVenue
// ============================================================================

// TestDeleteVenue verifies that deleteVenue removes the venue row from the DB.
func TestDeleteVenue(t *testing.T) {
	token := makeAdminToken(t)
	venueID := seedVenue(t, "Venue To Delete", "9 Gone Rd", "Eugene", "OR")

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
// createFundingEntity
// ============================================================================

// TestCreateFundingEntity verifies that a new funding entity can be created
// via the admin mutation and that the returned ID maps to a row in the DB.
func TestCreateFundingEntity(t *testing.T) {
	token := makeAdminToken(t)

	resp := gqlPost(t, "/graphql/admin", token, mutCreateFundingEntity, map[string]any{
		"input": map[string]any{
			"name":        "East Side",
			"description": "East side area offices",
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "createFundingEntity", &result)

	if !result.Success {
		t.Fatalf("createFundingEntity returned success=false: %v", result.Message)
	}
	if result.ID == nil || *result.ID == "" {
		t.Fatal("expected a non-empty id in response")
	}

	id := *result.ID
	t.Cleanup(func() { testDB.Exec("DELETE FROM funding_entities WHERE id = $1", id) })

	if !rowExists(t, "SELECT COUNT(*) FROM funding_entities WHERE id = $1", id) {
		t.Errorf("expected funding_entity row in DB for id=%s", id)
	}
}

// ============================================================================
// updateFundingEntity
// ============================================================================

// TestUpdateFundingEntity verifies that a funding entity's name can be changed
// and the update is persisted to the DB.
func TestUpdateFundingEntity(t *testing.T) {
	token := makeAdminToken(t)
	feID := seedFundingEntity(t, "Pre-Update Area")

	resp := gqlPost(t, "/graphql/admin", token, mutUpdateFundingEntity, map[string]any{
		"input": map[string]any{
			"id":   feID,
			"name": "Post-Update Area",
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "updateFundingEntity", &result)

	if !result.Success {
		t.Fatalf("updateFundingEntity returned success=false: %v", result.Message)
	}

	var name string
	if err := testDB.QueryRow(
		"SELECT name FROM funding_entities WHERE id = $1", feID,
	).Scan(&name); err != nil {
		t.Fatalf("querying updated funding entity: %v", err)
	}
	if name != "Post-Update Area" {
		t.Errorf("expected name='Post-Update Area', got %q", name)
	}
}

// ============================================================================
// deleteFundingEntity
// ============================================================================

// TestDeleteFundingEntity verifies that deleteFundingEntity soft-deletes the
// entity (sets is_active=false) rather than removing the row.
func TestDeleteFundingEntity(t *testing.T) {
	token := makeAdminToken(t)
	feID := seedFundingEntity(t, "Area To Delete")

	resp := gqlPost(t, "/graphql/admin", token, mutDeleteFundingEntity, map[string]any{
		"id": feID,
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "deleteFundingEntity", &result)

	if !result.Success {
		t.Fatalf("deleteFundingEntity returned success=false: %v", result.Message)
	}

	if rowExists(t, "SELECT COUNT(*) FROM funding_entities WHERE id = $1 AND is_active = TRUE", feID) {
		t.Error("expected funding entity to be marked inactive after deleteFundingEntity")
	}
}
