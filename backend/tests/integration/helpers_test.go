package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

// ============================================================================
// GraphQL request helper
// ============================================================================

// gqlResponse is the top-level structure returned by gqlgen.
type gqlResponse struct {
	Data   map[string]json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// gqlPost sends a GraphQL POST request to the given endpoint path (e.g.
// "/graphql/auth") with an optional Bearer token, and returns the parsed
// response. The test is failed immediately if the HTTP request itself fails.
func gqlPost(t *testing.T, path, token, query string, variables map[string]any) gqlResponse {
	t.Helper()

	body := map[string]any{"query": query}
	if variables != nil {
		body["variables"] = variables
	}

	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("gqlPost: marshal request: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, testServer.URL+path, bytes.NewReader(b))
	if err != nil {
		t.Fatalf("gqlPost: create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("gqlPost: do request: %v", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("gqlPost: read response: %v", err)
	}

	var result gqlResponse
	if err := json.Unmarshal(respBytes, &result); err != nil {
		t.Fatalf("gqlPost: unmarshal response: %v\nbody: %s", err, respBytes)
	}
	return result
}

// hasGQLErrors returns true if the response contains any GraphQL errors.
func hasGQLErrors(r gqlResponse) bool {
	return len(r.Errors) > 0
}

// unmarshalField parses a named field from gqlResponse.Data into dest.
func unmarshalField(t *testing.T, r gqlResponse, field string, dest any) {
	t.Helper()
	raw, ok := r.Data[field]
	if !ok {
		t.Fatalf("unmarshalField: field %q not found in response data", field)
	}
	if err := json.Unmarshal(raw, dest); err != nil {
		t.Fatalf("unmarshalField: unmarshal %q: %v", field, err)
	}
}

// ============================================================================
// DB seed helpers
// ============================================================================

// seedVolunteer inserts an active volunteer directly into the DB and returns the ID.
func seedVolunteer(t *testing.T, email, firstName, lastName, role string) int {
	t.Helper()
	var id int
	err := testDB.QueryRow(`
		INSERT INTO volunteers (email, first_name, last_name, role, is_active)
		VALUES ($1, $2, $3, $4, TRUE)
		RETURNING volunteer_id
	`, email, firstName, lastName, role).Scan(&id)
	if err != nil {
		t.Fatalf("seedVolunteer: %v", err)
	}
	t.Cleanup(func() {
		testDB.Exec("DELETE FROM volunteers WHERE volunteer_id = $1", id)
	})
	return id
}

// seedInactiveVolunteer inserts an inactive volunteer directly into the DB and
// returns the ID. Used to test flows where is_active = FALSE.
func seedInactiveVolunteer(t *testing.T, email, firstName, lastName, role string) int {
	t.Helper()
	var id int
	err := testDB.QueryRow(`
		INSERT INTO volunteers (email, first_name, last_name, role, is_active)
		VALUES ($1, $2, $3, $4, FALSE)
		RETURNING volunteer_id
	`, email, firstName, lastName, role).Scan(&id)
	if err != nil {
		t.Fatalf("seedInactiveVolunteer: %v", err)
	}
	t.Cleanup(func() {
		testDB.Exec("DELETE FROM volunteers WHERE volunteer_id = $1", id)
	})
	return id
}

// seedMagicLink inserts a magic link token directly into the DB.
// Use a future expiresAt for a valid token; a past time for an expired one.
// Times are converted to UTC before storage to match the TIMESTAMP column type
// in the database (which stores without timezone and compares against UTC NOW()).
func seedMagicLink(t *testing.T, email, token string, expiresAt time.Time) {
	t.Helper()
	_, err := testDB.Exec(`
		INSERT INTO magic_links (email, token, created_at, expires_at)
		VALUES ($1, $2, NOW(), $3)
	`, email, token, expiresAt.UTC())
	if err != nil {
		t.Fatalf("seedMagicLink: %v", err)
	}
	t.Cleanup(func() {
		testDB.Exec("DELETE FROM magic_links WHERE token = $1", token)
	})
}

// seedSession inserts a session token directly into the DB and returns the token.
// email must match the volunteer's email — the sessions table has a NOT NULL
// unique constraint on email.
func seedSession(t *testing.T, email string, volunteerID int, role, token string) string {
	t.Helper()
	_, err := testDB.Exec(`
		INSERT INTO sessions (email, volunteer_id, role, token, created_at, expires_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW() + INTERVAL '1 day')
	`, email, volunteerID, role, token)
	if err != nil {
		t.Fatalf("seedSession: %v", err)
	}
	t.Cleanup(func() {
		testDB.Exec("DELETE FROM sessions WHERE token = $1", token)
	})
	return token
}

// seedExpiredSession inserts an already-expired session token into the DB.
func seedExpiredSession(t *testing.T, email string, volunteerID int, role, token string) string {
	t.Helper()
	_, err := testDB.Exec(`
		INSERT INTO sessions (email, volunteer_id, role, token, created_at, expires_at)
		VALUES ($1, $2, $3, $4, NOW() - INTERVAL '2 days', NOW() - INTERVAL '1 day')
	`, email, volunteerID, role, token)
	if err != nil {
		t.Fatalf("seedExpiredSession: %v", err)
	}
	t.Cleanup(func() {
		testDB.Exec("DELETE FROM sessions WHERE token = $1", token)
	})
	return token
}

// sessionExists returns true if a session with the given token exists in the DB.
func sessionExists(t *testing.T, token string) bool {
	t.Helper()
	var count int
	err := testDB.QueryRow("SELECT COUNT(*) FROM sessions WHERE token = $1", token).Scan(&count)
	if err != nil {
		t.Fatalf("sessionExists: %v", err)
	}
	return count > 0
}

// magicLinkUsed returns true if the magic link token has been marked as used.
func magicLinkUsed(t *testing.T, token string) bool {
	t.Helper()
	var count int
	err := testDB.QueryRow(
		"SELECT COUNT(*) FROM magic_links WHERE token = $1 AND used_at IS NOT NULL", token,
	).Scan(&count)
	if err != nil {
		t.Fatalf("magicLinkUsed: %v", err)
	}
	return count > 0
}

// uniqueEmail returns a unique email address for use in a single test,
// avoiding collisions between parallel or sequential test runs.
func uniqueEmail(t *testing.T) string {
	return fmt.Sprintf("test-%d@example.com", time.Now().UnixNano())
}

// ============================================================================
// Event seed helpers
// ============================================================================

// getJobTypeID looks up the job_type_id for a seeded job-type code.
func getJobTypeID(t *testing.T, code string) int {
	t.Helper()
	var id int
	err := testDB.QueryRow("SELECT job_type_id FROM job_types WHERE code = $1", code).Scan(&id)
	if err != nil {
		t.Fatalf("getJobTypeID(%q): %v", code, err)
	}
	return id
}

// seedRegion inserts a region and returns its region_id.
func seedRegion(t *testing.T, code, name string) int {
	t.Helper()
	var id int
	err := testDB.QueryRow(`
		INSERT INTO regions (code, name, is_active)
		VALUES ($1, $2, TRUE)
		RETURNING region_id
	`, code, name).Scan(&id)
	if err != nil {
		t.Fatalf("seedRegion: %v", err)
	}
	t.Cleanup(func() {
		testDB.Exec("DELETE FROM regions WHERE region_id = $1", id)
	})
	return id
}

// seedVenue inserts a venue (no zip code) and returns its venue_id.
func seedVenue(t *testing.T, name, address, city, state, timezone string) int {
	t.Helper()
	var id int
	err := testDB.QueryRow(`
		INSERT INTO venues (venue_name, street_address, city, state, timezone)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING venue_id
	`, name, address, city, state, timezone).Scan(&id)
	if err != nil {
		t.Fatalf("seedVenue: %v", err)
	}
	t.Cleanup(func() {
		testDB.Exec("DELETE FROM venues WHERE venue_id = $1", id)
	})
	return id
}

// seedVenueRegion links a venue to a region.
func seedVenueRegion(t *testing.T, venueID, regionID int) {
	t.Helper()
	_, err := testDB.Exec(`
		INSERT INTO venue_regions (venue_id, region_id) VALUES ($1, $2)
	`, venueID, regionID)
	if err != nil {
		t.Fatalf("seedVenueRegion: %v", err)
	}
	t.Cleanup(func() {
		testDB.Exec("DELETE FROM venue_regions WHERE venue_id = $1 AND region_id = $2", venueID, regionID)
	})
}

// seedEvent inserts an event and returns its event_id.
//
//	isVirtual=true,  venueID=nil  → VIRTUAL
//	isVirtual=false, venueID!=nil → IN_PERSON
//	isVirtual=true,  venueID!=nil → HYBRID
func seedEvent(t *testing.T, name string, isVirtual bool, venueID *int) int {
	t.Helper()
	var id int
	var err error
	if venueID == nil {
		err = testDB.QueryRow(`
			INSERT INTO events (event_name, event_is_virtual)
			VALUES ($1, $2)
			RETURNING event_id
		`, name, isVirtual).Scan(&id)
	} else {
		err = testDB.QueryRow(`
			INSERT INTO events (event_name, event_is_virtual, venue_id)
			VALUES ($1, $2, $3)
			RETURNING event_id
		`, name, isVirtual, *venueID).Scan(&id)
	}
	if err != nil {
		t.Fatalf("seedEvent: %v", err)
	}
	t.Cleanup(func() {
		testDB.Exec("DELETE FROM events WHERE event_id = $1", id)
	})
	return id
}

// seedEventDate inserts an event_date row. startUTC and endUTC must be
// RFC3339 strings (e.g. "2026-04-15T09:00:00Z"), matching the format
// used by the production code.
func seedEventDate(t *testing.T, eventID int, startUTC, endUTC string) int {
	t.Helper()
	var id int
	err := testDB.QueryRow(`
		INSERT INTO event_dates (event_id, start_date_time, end_date_time)
		VALUES ($1, $2, $3)
		RETURNING event_date_id
	`, eventID, startUTC, endUTC).Scan(&id)
	if err != nil {
		t.Fatalf("seedEventDate: %v", err)
	}
	t.Cleanup(func() {
		testDB.Exec("DELETE FROM event_dates WHERE event_date_id = $1", id)
	})
	return id
}

// seedOpportunity inserts an opportunity and returns its opportunity_id.
func seedOpportunity(t *testing.T, eventID, jobTypeID int, isVirtual bool) int {
	t.Helper()
	var id int
	err := testDB.QueryRow(`
		INSERT INTO opportunities (event_id, job_type_id, opportunity_is_virtual)
		VALUES ($1, $2, $3)
		RETURNING opportunity_id
	`, eventID, jobTypeID, isVirtual).Scan(&id)
	if err != nil {
		t.Fatalf("seedOpportunity: %v", err)
	}
	t.Cleanup(func() {
		testDB.Exec("DELETE FROM opportunities WHERE opportunity_id = $1", id)
	})
	return id
}

// getServiceTypeID looks up the service_type_id for a seeded service-type code.
func getServiceTypeID(t *testing.T, code string) int {
	t.Helper()
	var id int
	err := testDB.QueryRow("SELECT service_type_id FROM service_types WHERE code = $1", code).Scan(&id)
	if err != nil {
		t.Fatalf("getServiceTypeID(%q): %v", code, err)
	}
	return id
}

// uniqueCode returns a short unique lowercase string suitable for use as a DB
// code column value (e.g. job_types.code, regions.code).
func uniqueCode(t *testing.T, prefix string) string {
	t.Helper()
	return fmt.Sprintf("%s%d", prefix, time.Now().UnixNano())
}

// makeAdminToken creates an ADMINISTRATOR volunteer and session and returns
// the session token. All seeded rows are removed via t.Cleanup.
func makeAdminToken(t *testing.T) string {
	t.Helper()
	email := uniqueEmail(t)
	id := seedVolunteer(t, email, "Admin", "Test", "ADMINISTRATOR")
	return seedSession(t, email, id, "ADMINISTRATOR", "adm-"+email)
}

// mutationResult matches the MutationResult GraphQL type used by all CRUD
// mutations in the admin schema.
type mutationResult struct {
	Success bool    `json:"success"`
	Message *string `json:"message"`
	ID      *string `json:"id"`
}

// rowExists returns true when the given COUNT(*) query returns a positive number.
// Use it to assert whether a row is present or absent after a mutation.
func rowExists(t *testing.T, query string, args ...any) bool {
	t.Helper()
	var count int
	if err := testDB.QueryRow(query, args...).Scan(&count); err != nil {
		t.Fatalf("rowExists: %v", err)
	}
	return count > 0
}

// seedJobType inserts a job type with a placeholder sort_order and returns
// its job_type_id. Use uniqueCode(t, "prefix") for the code argument to
// avoid UNIQUE constraint collisions across test runs.
func seedJobType(t *testing.T, code, name string) int {
	t.Helper()
	var id int
	err := testDB.QueryRow(`
		INSERT INTO job_types (code, name, sort_order)
		VALUES ($1, $2, 0)
		RETURNING job_type_id
	`, code, name).Scan(&id)
	if err != nil {
		t.Fatalf("seedJobType: %v", err)
	}
	t.Cleanup(func() {
		testDB.Exec("DELETE FROM job_types WHERE job_type_id = $1", id)
	})
	return id
}

// seedShift inserts a shift and returns its shift_id. startUTC and endUTC
// must be RFC3339 strings (e.g. "2026-04-15T09:00:00Z").
func seedShift(t *testing.T, opportunityID int, startUTC, endUTC string, maxVolunteers int) int {
	t.Helper()
	var id int
	err := testDB.QueryRow(`
		INSERT INTO shifts (opportunity_id, shift_start, shift_end, max_volunteers)
		VALUES ($1, $2, $3, $4)
		RETURNING shift_id
	`, opportunityID, startUTC, endUTC, maxVolunteers).Scan(&id)
	if err != nil {
		t.Fatalf("seedShift: %v", err)
	}
	t.Cleanup(func() {
		testDB.Exec("DELETE FROM shifts WHERE shift_id = $1", id)
	})
	return id
}

// makeVolunteer creates a VOLUNTEER role volunteer and session and returns
// (sessionToken, volunteerID). All seeded rows are removed via t.Cleanup.
func makeVolunteer(t *testing.T) (string, int) {
	t.Helper()
	email := uniqueEmail(t)
	id := seedVolunteer(t, email, "Vol", "Test", "VOLUNTEER")
	token := seedSession(t, email, id, "VOLUNTEER", "vol-"+email)
	return token, id
}

// makeAdmin creates an ADMINISTRATOR volunteer and session, returning
// (sessionToken, volunteerID). Mirrors makeVolunteer for admin use cases.
func makeAdmin(t *testing.T) (string, int) {
	t.Helper()
	email := uniqueEmail(t)
	id := seedVolunteer(t, email, "Admin", "Test", "ADMINISTRATOR")
	token := seedSession(t, email, id, "ADMINISTRATOR", "adm-"+email)
	return token, id
}

// seedStaff inserts a staff member and returns the staff_id.
func seedStaff(t *testing.T, firstName, lastName, email string) int {
	t.Helper()
	var id int
	err := testDB.QueryRow(`
		INSERT INTO staff (first_name, last_name, email)
		VALUES ($1, $2, $3)
		RETURNING staff_id
	`, firstName, lastName, email).Scan(&id)
	if err != nil {
		t.Fatalf("seedStaff: %v", err)
	}
	t.Cleanup(func() {
		testDB.Exec("DELETE FROM staff WHERE staff_id = $1", id)
	})
	return id
}

// seedVolunteerShift inserts a row into volunteer_shifts assigning volID to
// shiftID. The insert is idempotent (ON CONFLICT DO NOTHING). A Cleanup is
// registered to remove the row when the test ends.
func seedVolunteerShift(t *testing.T, shiftID, volID int) {
	t.Helper()
	_, err := testDB.Exec(`
		INSERT INTO volunteer_shifts (volunteer_id, shift_id, assigned_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (volunteer_id, shift_id) DO NOTHING
	`, volID, shiftID)
	if err != nil {
		t.Fatalf("seedVolunteerShift: %v", err)
	}
	t.Cleanup(func() {
		testDB.Exec("DELETE FROM volunteer_shifts WHERE volunteer_id = $1 AND shift_id = $2", volID, shiftID)
	})
}
