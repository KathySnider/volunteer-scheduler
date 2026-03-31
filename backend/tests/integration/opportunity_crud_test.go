package integration

import (
	"fmt"
	"testing"
)

// ============================================================================
// Shared mutation strings
// ============================================================================

const (
	mutCreateOpportunity = `
		mutation CreateOpportunity($input: NewOpportunityInput!) {
			createOpportunity(newOpp: $input) { success message id }
		}`

	mutUpdateOpportunity = `
		mutation UpdateOpportunity($input: UpdateOpportunityInput!) {
			updateOpportunity(opp: $input) { success message id }
		}`

	mutDeleteOpportunity = `
		mutation DeleteOpportunity($id: ID!) {
			deleteOpportunity(oppId: $id) { success message id }
		}`
)

// ============================================================================
// createOpportunity
// ============================================================================

// TestCreateOpportunity verifies that a new virtual opportunity (with one
// embedded shift) can be created for an existing event.
func TestCreateOpportunity(t *testing.T) {
	token := makeAdminToken(t)
	jobID := getJobTypeID(t, "event_support")
	eventID := seedEvent(t, "Event For Opp Create", true, nil)

	resp := gqlPost(t, "/graphql/admin", token, mutCreateOpportunity, map[string]any{
		"input": map[string]any{
			"eventId":   fmt.Sprintf("%d", eventID),
			"jobId":     jobID,
			"isVirtual": true,
			"shifts": []map[string]any{
				{
					"startDateTime": "2027-01-10 09:00:00",
					"endDateTime":   "2027-01-10 17:00:00",
					"ianaZone":      "UTC",
					"maxVolunteers": 10,
				},
			},
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "createOpportunity", &result)

	if !result.Success {
		t.Fatalf("createOpportunity returned success=false: %v", result.Message)
	}
	if result.ID == nil || *result.ID == "" {
		t.Fatal("expected a non-empty opportunity ID in response")
	}

	oppID := *result.ID
	// Cascade-delete removes shifts when the opportunity is removed.
	t.Cleanup(func() { testDB.Exec("DELETE FROM opportunities WHERE opportunity_id = $1", oppID) })
}

// TestCreateOpportunity_NoShifts verifies that the service rejects a new
// opportunity that supplies no shifts.
func TestCreateOpportunity_NoShifts(t *testing.T) {
	token := makeAdminToken(t)
	jobID := getJobTypeID(t, "advocacy")
	eventID := seedEvent(t, "Event For Opp NoShifts", true, nil)

	resp := gqlPost(t, "/graphql/admin", token, mutCreateOpportunity, map[string]any{
		"input": map[string]any{
			"eventId":   fmt.Sprintf("%d", eventID),
			"jobId":     jobID,
			"isVirtual": true,
			"shifts":    []map[string]any{},
		},
	})

	if !hasGQLErrors(resp) {
		t.Error("expected a GraphQL error when no shifts are provided for a new opportunity")
	}
}

// ============================================================================
// updateOpportunity
// ============================================================================

// TestUpdateOpportunity verifies that an opportunity's job type and virtual
// flag can be changed, and that the change is persisted to the DB.
func TestUpdateOpportunity(t *testing.T) {
	token := makeAdminToken(t)
	jobID := getJobTypeID(t, "event_support")
	newJobID := getJobTypeID(t, "advocacy")
	eventID := seedEvent(t, "Event For Opp Update", true, nil)
	oppID := seedOpportunity(t, eventID, jobID, true)
	seedShift(t, oppID, "2027-02-01T09:00:00Z", "2027-02-01T17:00:00Z", 5)

	resp := gqlPost(t, "/graphql/admin", token, mutUpdateOpportunity, map[string]any{
		"input": map[string]any{
			"id":        fmt.Sprintf("%d", oppID),
			"jobId":     newJobID,
			"isVirtual": false,
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "updateOpportunity", &result)

	if !result.Success {
		t.Fatalf("updateOpportunity returned success=false: %v", result.Message)
	}

	// Verify the DB reflects the new job type.
	var storedJobID int
	if err := testDB.QueryRow(
		"SELECT job_type_id FROM opportunities WHERE opportunity_id = $1", oppID,
	).Scan(&storedJobID); err != nil {
		t.Fatalf("querying updated opportunity: %v", err)
	}
	if storedJobID != newJobID {
		t.Errorf("expected job_type_id=%d, got %d", newJobID, storedJobID)
	}
}

// ============================================================================
// deleteOpportunity
// ============================================================================

// TestDeleteOpportunity verifies that deleteOpportunity removes the
// opportunity row and cascades to its shifts.
func TestDeleteOpportunity(t *testing.T) {
	token := makeAdminToken(t)
	jobID := getJobTypeID(t, "speaker")
	eventID := seedEvent(t, "Event For Opp Delete", true, nil)
	oppID := seedOpportunity(t, eventID, jobID, true)
	shiftID := seedShift(t, oppID, "2027-03-01T09:00:00Z", "2027-03-01T17:00:00Z", 5)

	resp := gqlPost(t, "/graphql/admin", token, mutDeleteOpportunity, map[string]any{
		"id": fmt.Sprintf("%d", oppID),
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "deleteOpportunity", &result)

	if !result.Success {
		t.Fatalf("deleteOpportunity returned success=false: %v", result.Message)
	}

	if rowExists(t, "SELECT COUNT(*) FROM opportunities WHERE opportunity_id = $1", oppID) {
		t.Error("expected opportunity to be gone after deleteOpportunity")
	}
	// Shifts should be cascade-deleted by the DB.
	if rowExists(t, "SELECT COUNT(*) FROM shifts WHERE shift_id = $1", shiftID) {
		t.Error("expected shift to be cascade-deleted when its parent opportunity was removed")
	}
}
