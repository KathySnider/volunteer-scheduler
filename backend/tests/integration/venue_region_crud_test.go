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
// mutation and that the returned ID maps to a row in the DB.
func TestCreateVenue(t *testing.T) {
	token := makeAdminToken(t)

	resp := gqlPost(t, "/graphql/admin", token, mutCreateVenue, map[string]any{
		"input": map[string]any{
			"name":     "CRUD Test Venue",
			"address":  "100 Test Blvd",
			"city":     "Portland",
			"state":    "OR",
			"ianaZone": "America/Los_Angeles",
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
