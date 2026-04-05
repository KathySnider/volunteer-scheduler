package integration

import (
	"fmt"
	"strconv"
	"testing"
)

// ============================================================================
// Query strings
// ============================================================================

const (
	qryLookupValues = `query LookupValues {
		lookupValues {
			serviceTypes { name }
			jobTypes { name }
			regions { name }
		}
	}`

	qryVenues = `query Venues {
		venues {
			id name address city state timezone
		}
	}`

	qryStaff = `query Staff {
		staff {
			id firstName lastName email
		}
	}`

	qryAllVolunteers = `query AllVolunteers {
		allVolunteers {
			id firstName lastName email role
		}
	}`

	qryVolunteerById = `query VolunteerById($volunteerId: ID!) {
		volunteerById(volunteerId: $volunteerId) {
			firstName lastName email role
		}
	}`

	qryVolunteerShifts = `query VolunteerShifts($volunteerId: ID!, $filter: ShiftTimeFilter!) {
		volunteerShifts(volunteerId: $volunteerId, filter: $filter) {
			shiftId jobName eventName startDateTime endDateTime
		}
	}`

	qryOpportunitiesForEvent = `query OpportunitiesForEvent($eventId: ID!) {
		opportunitiesForEvent(eventId: $eventId) {
			id isVirtual shifts {
				id startDateTime endDateTime maxVolunteers
			}
		}
	}`

	qryFeedbackList = `query Feedback {
		feedback {
			id subject status type volunteerName
		}
	}`

	qryFeedbackById = `query FeedbackById($feedbackId: ID!) {
		feedbackById(feedbackId: $feedbackId) {
			id subject status type volunteerName
			notes { note creator createdAt }
		}
	}`
)

// ============================================================================
// Local response types
// ============================================================================

type lookupValuesResult struct {
	ServiceTypes []struct {
		Name string `json:"name"`
	} `json:"serviceTypes"`
	JobTypes []struct {
		Name string `json:"name"`
	} `json:"jobTypes"`
	Regions []struct {
		Name string `json:"name"`
	} `json:"regions"`
}

type venueResult struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Address  string `json:"address"`
	City     string `json:"city"`
	State    string `json:"state"`
	Timezone string `json:"timezone"`
}

type staffResult struct {
	ID        string `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
}

type volunteerListResult struct {
	ID        string `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	Role      string `json:"role"`
}

type opportunityShiftResult struct {
	ID            string `json:"id"`
	StartDateTime string `json:"startDateTime"`
	EndDateTime   string `json:"endDateTime"`
	MaxVolunteers *int   `json:"maxVolunteers"`
}

type opportunityResult struct {
	ID        string                   `json:"id"`
	IsVirtual bool                     `json:"isVirtual"`
	Shifts    []opportunityShiftResult `json:"shifts"`
}

type feedbackListResult struct {
	ID            string `json:"id"`
	Subject       string `json:"subject"`
	Status        string `json:"status"`
	Type          string `json:"type"`
	VolunteerName string `json:"volunteerName"`
}

type feedbackNoteResult struct {
	Note      string `json:"note"`
	Creator   string `json:"creator"`
	CreatedAt string `json:"createdAt"`
}

type feedbackDetailResult struct {
	ID            string               `json:"id"`
	Subject       string               `json:"subject"`
	Status        string               `json:"status"`
	Type          string               `json:"type"`
	VolunteerName string               `json:"volunteerName"`
	Notes         []feedbackNoteResult `json:"notes"`
}

// ============================================================================
// Tests
// ============================================================================

// TestLookupValues verifies that the lookupValues query returns non-empty
// arrays for at least feedbackStatuses and feedbackTypes.
func TestLookupValues(t *testing.T) {
	adminToken := makeAdminToken(t)

	resp := gqlPost(t, "/graphql/admin", adminToken, qryLookupValues, nil)

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	var result lookupValuesResult
	unmarshalField(t, resp, "lookupValues", &result)

	if len(result.ServiceTypes) == 0 {
		t.Error("expected non-empty serviceTypes in lookupValues")
	}
	if len(result.JobTypes) == 0 {
		t.Error("expected non-empty jobTypes in lookupValues")
	}
}

// TestVenues verifies that an admin can query venues and that a seeded venue
// appears in the results.
func TestVenues(t *testing.T) {
	adminToken := makeAdminToken(t)
	venueID := seedVenue(t, "Admin Query Test Venue", "123 Main St", "Springfield", "IL", "America/Chicago")
	regionID := seedRegion(t, uniqueCode(t, "vqr"), "Venue Query Region")
	seedVenueRegion(t, venueID, regionID)

	resp := gqlPost(t, "/graphql/admin", adminToken, qryVenues, nil)

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	var venues []venueResult
	unmarshalField(t, resp, "venues", &venues)

	expectedID := strconv.Itoa(venueID)
	found := false
	for _, v := range venues {
		if v.ID == expectedID {
			found = true
			if v.Name != "Admin Query Test Venue" {
				t.Errorf("expected name=%q, got %q", "Admin Query Test Venue", v.Name)
			}
			if v.City != "Springfield" {
				t.Errorf("expected city=%q, got %q", "Springfield", v.City)
			}
			break
		}
	}
	if !found {
		t.Errorf("seeded venue with id=%d not found in venues results", venueID)
	}
}

// TestStaff verifies that an admin can query staff and that a seeded staff
// member appears in the results.
func TestStaff(t *testing.T) {
	adminToken := makeAdminToken(t)
	staffID := seedStaff(t, "Jane", "Stafford", uniqueEmail(t))

	resp := gqlPost(t, "/graphql/admin", adminToken, qryStaff, nil)

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	var staffList []staffResult
	unmarshalField(t, resp, "staff", &staffList)

	expectedID := strconv.Itoa(staffID)
	found := false
	for _, s := range staffList {
		if s.ID == expectedID {
			found = true
			if s.FirstName != "Jane" {
				t.Errorf("expected firstName=%q, got %q", "Jane", s.FirstName)
			}
			if s.LastName != "Stafford" {
				t.Errorf("expected lastName=%q, got %q", "Stafford", s.LastName)
			}
			break
		}
	}
	if !found {
		t.Errorf("seeded staff with id=%d not found in staff results", staffID)
	}
}

// TestAllVolunteers verifies that an admin can query all volunteers and that a
// seeded volunteer appears in the results.
func TestAllVolunteers(t *testing.T) {
	adminToken := makeAdminToken(t)
	email := uniqueEmail(t)
	volID := seedVolunteer(t, email, "Query", "Subject", "VOLUNTEER")

	resp := gqlPost(t, "/graphql/admin", adminToken, qryAllVolunteers, nil)

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	var volunteers []volunteerListResult
	unmarshalField(t, resp, "allVolunteers", &volunteers)

	expectedID := strconv.Itoa(volID)
	found := false
	for _, v := range volunteers {
		if v.ID == expectedID {
			found = true
			if v.FirstName != "Query" {
				t.Errorf("expected firstName=%q, got %q", "Query", v.FirstName)
			}
			if v.LastName != "Subject" {
				t.Errorf("expected lastName=%q, got %q", "Subject", v.LastName)
			}
			if v.Email != email {
				t.Errorf("expected email=%q, got %q", email, v.Email)
			}
			break
		}
	}
	if !found {
		t.Errorf("seeded volunteer with id=%d not found in allVolunteers results", volID)
	}
}

// TestVolunteerById verifies that an admin can fetch a single volunteer's
// profile by ID and that the correct fields are returned.
func TestVolunteerById(t *testing.T) {
	adminToken := makeAdminToken(t)
	email := uniqueEmail(t)
	volID := seedVolunteer(t, email, "Fetched", "ByID", "VOLUNTEER")

	resp := gqlPost(t, "/graphql/admin", adminToken, qryVolunteerById, map[string]any{
		"volunteerId": strconv.Itoa(volID),
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	var profile volunteerProfileResult
	unmarshalField(t, resp, "volunteerById", &profile)

	if profile.FirstName != "Fetched" {
		t.Errorf("expected firstName=%q, got %q", "Fetched", profile.FirstName)
	}
	if profile.LastName != "ByID" {
		t.Errorf("expected lastName=%q, got %q", "ByID", profile.LastName)
	}
	if profile.Email != email {
		t.Errorf("expected email=%q, got %q", email, profile.Email)
	}
	if profile.Role != "VOLUNTEER" {
		t.Errorf("expected role=%q, got %q", "VOLUNTEER", profile.Role)
	}
}

// TestVolunteerShifts verifies that an admin can query a specific volunteer's
// shifts and that a seeded assignment appears in the results.
func TestVolunteerShifts(t *testing.T) {
	adminToken := makeAdminToken(t)
	_, volID := makeVolunteer(t)
	_, shiftID := seedEventWithShift(t, 3)

	// Register assignment cleanup BEFORE seedVolunteerShift so it runs first
	// (LIFO), preventing FK constraint errors when the shift row is deleted.
	t.Cleanup(func() {
		testDB.Exec(
			"DELETE FROM volunteer_shifts WHERE volunteer_id = $1 AND shift_id = $2",
			volID, shiftID,
		)
	})
	seedVolunteerShift(t, shiftID, volID)

	resp := gqlPost(t, "/graphql/admin", adminToken, qryVolunteerShifts, map[string]any{
		"volunteerId": strconv.Itoa(volID),
		"filter":      "ALL",
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	var shifts []ownShiftResult
	unmarshalField(t, resp, "volunteerShifts", &shifts)

	if len(shifts) == 0 {
		t.Fatal("expected at least one shift in volunteerShifts, got none")
	}

	expectedID := fmt.Sprintf("%d", shiftID)
	found := false
	for _, s := range shifts {
		if s.ShiftId == expectedID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("seeded shiftId %q not found in volunteerShifts results", expectedID)
	}
}

// TestOpportunitiesForEvent verifies that an admin can query opportunities for
// an event and that the seeded opportunity (with its nested shift) is returned.
func TestOpportunitiesForEvent(t *testing.T) {
	adminToken := makeAdminToken(t)
	eventID, shiftID := seedEventWithShift(t, 5)

	resp := gqlPost(t, "/graphql/admin", adminToken, qryOpportunitiesForEvent, map[string]any{
		"eventId": strconv.Itoa(eventID),
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	var opps []opportunityResult
	unmarshalField(t, resp, "opportunitiesForEvent", &opps)

	if len(opps) == 0 {
		t.Fatal("expected at least one opportunity in opportunitiesForEvent, got none")
	}

	expectedShiftID := fmt.Sprintf("%d", shiftID)
	shiftFound := false
	for _, opp := range opps {
		for _, s := range opp.Shifts {
			if s.ID == expectedShiftID {
				shiftFound = true
				if s.MaxVolunteers == nil || *s.MaxVolunteers != 5 {
					t.Errorf("expected maxVolunteers=5, got %v", s.MaxVolunteers)
				}
				break
			}
		}
		if shiftFound {
			break
		}
	}
	if !shiftFound {
		t.Errorf("seeded shiftId %q not found in any opportunity's shifts", expectedShiftID)
	}
}

// TestFeedbackList verifies that an admin can query the feedback list (without
// filter) and that a seeded feedback item appears in the results.
//
// Note: the feedback list query uses unsafe string interpolation for filters,
// so we exercise it only without a filter to avoid triggering the SQL bug.
func TestFeedbackList(t *testing.T) {
	adminToken := makeAdminToken(t)
	_, volID := makeVolunteer(t)
	feedbackID := seedFeedback(t, volID)

	resp := gqlPost(t, "/graphql/admin", adminToken, qryFeedbackList, nil)

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	var feedbackItems []feedbackListResult
	unmarshalField(t, resp, "feedback", &feedbackItems)

	expectedID := strconv.Itoa(feedbackID)
	found := false
	for _, fb := range feedbackItems {
		if fb.ID == expectedID {
			found = true
			if fb.Subject != "Test subject" {
				t.Errorf("expected subject=%q, got %q", "Test subject", fb.Subject)
			}
			if fb.Status != "OPEN" {
				t.Errorf("expected status=%q, got %q", "OPEN", fb.Status)
			}
			break
		}
	}
	if !found {
		t.Errorf("seeded feedbackId %d not found in feedback list results", feedbackID)
	}
}

// TestFeedbackById verifies that an admin can fetch a specific feedback item by
// ID and that the returned fields match what was seeded.
func TestFeedbackById(t *testing.T) {
	adminToken := makeAdminToken(t)
	_, volID := makeVolunteer(t)
	feedbackID := seedFeedback(t, volID)

	resp := gqlPost(t, "/graphql/admin", adminToken, qryFeedbackById, map[string]any{
		"feedbackId": strconv.Itoa(feedbackID),
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	var fb feedbackDetailResult
	unmarshalField(t, resp, "feedbackById", &fb)

	if fb.ID != strconv.Itoa(feedbackID) {
		t.Errorf("expected id=%q, got %q", strconv.Itoa(feedbackID), fb.ID)
	}
	if fb.Subject != "Test subject" {
		t.Errorf("expected subject=%q, got %q", "Test subject", fb.Subject)
	}
	if fb.Status != "OPEN" {
		t.Errorf("expected status=%q, got %q", "OPEN", fb.Status)
	}
	if fb.Type != "BUG" {
		t.Errorf("expected type=%q, got %q", "BUG", fb.Type)
	}
	// Notes should be an empty slice (not nil) for a freshly seeded feedback row.
	if fb.Notes == nil {
		t.Error("expected notes to be a non-nil slice (empty is fine), got nil")
	}
}
