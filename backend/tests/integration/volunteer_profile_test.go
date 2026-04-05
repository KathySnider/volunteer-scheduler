package integration

import (
	"fmt"
	"testing"
)

// ============================================================================
// Query / mutation strings
// ============================================================================

const (
	qryVolunteerProfile = `query {
		volunteerProfile { firstName lastName email role }
	}`

	// updateOwnProfile returns VolunteerMutationResult { success message }
	// — there is no id field on that type.
	mutUpdateOwnProfile = `mutation UpdateOwnProfile($input: UpdateOwnProfileInput!) {
		updateOwnProfile(profile: $input) { success message }
	}`

	qryEventById = `query EventById($id: ID!) {
		eventById(eventId: $id) { id name eventType }
	}`
)

// ============================================================================
// Local response types
// ============================================================================

type volunteerProfileResult struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	Role      string `json:"role"`
}

// volunteerMutationResult mirrors VolunteerMutationResult { success message }.
// Unlike the admin MutationResult, this type has no id field.
// We reuse the mutationResult struct (id field stays nil) which is fine.
type eventByIdResult struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	EventType string `json:"eventType"`
}

// ============================================================================
// Tests
// ============================================================================

// TestVolunteerProfile verifies that a logged-in volunteer can fetch their
// own profile and that the returned fields match what was seeded.
func TestVolunteerProfile(t *testing.T) {
	token, _ := makeVolunteer(t)

	resp := gqlPost(t, "/graphql/volunteer", token, qryVolunteerProfile, nil)

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	var profile volunteerProfileResult
	unmarshalField(t, resp, "volunteerProfile", &profile)

	if profile.FirstName != "Vol" {
		t.Errorf("expected firstName=%q, got %q", "Vol", profile.FirstName)
	}
	if profile.LastName != "Test" {
		t.Errorf("expected lastName=%q, got %q", "Test", profile.LastName)
	}
	if profile.Role != "VOLUNTEER" {
		t.Errorf("expected role=VOLUNTEER, got %q", profile.Role)
	}
}

// TestUpdateOwnProfile verifies that a volunteer can update their own profile
// via the volunteer endpoint and that the change is reflected in the DB.
func TestUpdateOwnProfile(t *testing.T) {
	token, volID := makeVolunteer(t)
	newEmail := uniqueEmail(t)

	resp := gqlPost(t, "/graphql/volunteer", token, mutUpdateOwnProfile, map[string]any{
		"input": map[string]any{
			"firstName": "Updated",
			"lastName":  "Profile",
			"email":     newEmail,
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	// updateOwnProfile returns VolunteerMutationResult (no id).
	// mutationResult works because the id field simply stays nil.
	var result mutationResult
	unmarshalField(t, resp, "updateOwnProfile", &result)

	if !result.Success {
		t.Errorf("expected success=true, got false (message: %v)", result.Message)
	}
	if !rowExists(t, "SELECT COUNT(*) FROM volunteers WHERE volunteer_id = $1 AND first_name = 'Updated'", volID) {
		t.Error("expected first_name='Updated' in DB after updateOwnProfile")
	}
}

// TestEventById verifies that a volunteer can query an event by ID and receive
// the correct name and event type.
func TestEventById(t *testing.T) {
	token, _ := makeVolunteer(t)
	eventID := seedEvent(t, "Profile Test Event", true, nil)

	resp := gqlPost(t, "/graphql/volunteer", token, qryEventById, map[string]any{
		"id": fmt.Sprintf("%d", eventID),
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	var event eventByIdResult
	unmarshalField(t, resp, "eventById", &event)

	if event.Name != "Profile Test Event" {
		t.Errorf("expected event name %q, got %q", "Profile Test Event", event.Name)
	}
	if event.EventType != "VIRTUAL" {
		t.Errorf("expected eventType=VIRTUAL, got %q", event.EventType)
	}
}
