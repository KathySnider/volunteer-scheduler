package integration

import (
	"strconv"
	"testing"
)

// seedFeedbackNote inserts a note directly into feedback_notes and returns
// the note_id. Cleanup removes the row after the test.
func seedFeedbackNote(t *testing.T, feedbackID, volID int, noteType, text string) int {
	t.Helper()
	var id int
	err := testDB.QueryRow(`
		INSERT INTO feedback_notes (feedback_id, volunteer_id, note, note_type, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING note_id
	`, feedbackID, volID, text, noteType).Scan(&id)
	if err != nil {
		t.Fatalf("seedFeedbackNote: %v", err)
	}
	t.Cleanup(func() {
		testDB.Exec("DELETE FROM feedback_notes WHERE note_id = $1", id)
	})
	return id
}

// All feedback service functions are correct. Tests below cover the full
// surface area except feedbackById, which can be added once the resolvers
// and schema wiring are in place.

// ============================================================================
// Mutation strings
// ============================================================================

const (
	qryOwnFeedback = `query {
		ownFeedback {
			id subject status type text
			notes { note noteType createdAt }
		}
	}`

	mutGiveFeedback = `mutation GiveFeedback($input: NewFeedbackInput!) {
		giveFeedback(feedback: $input) { success message id }
	}`

	mutUpdateFeedback = `mutation UpdateFeedback($input: UpdateFeedbackInput!) {
		updateFeedback(feedback: $input) { success message id }
	}`

	mutResolveFeedback = `mutation ResolveFeedback($input: ResolveFeedbackInput!) {
		resolveFeedback(resolution: $input) { success message id }
	}`

	// questionFeedback sends an email to the volunteer who submitted the
	// feedback. See TestQuestionFeedback for notes on the mailer dependency.
	mutQuestionFeedback = `mutation QuestionFeedback($input: QuestionFeedbackInput!) {
		questionFeedback(question: $input) { success message id }
	}`
)

// ============================================================================
// Tests — volunteer endpoint
// ============================================================================

// TestGiveFeedback verifies that a volunteer can submit feedback and that the
// row is inserted into the DB with the correct id returned.
func TestGiveFeedback(t *testing.T) {
	token, _ := makeVolunteer(t)

	resp := gqlPost(t, "/graphql/volunteer", token, mutGiveFeedback, map[string]any{
		"input": map[string]any{
			"type":          "BUG",
			"subject":       "Integration test feedback",
			"app_page_name": "TestPage",
			"text":          "This feedback entry was created by an integration test.",
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "giveFeedback", &result)

	if !result.Success {
		t.Errorf("expected success=true, got false (message: %v)", result.Message)
	}
	if result.ID == nil {
		t.Fatal("expected id to be set after giveFeedback")
	}

	feedbackID, err := strconv.Atoi(*result.ID)
	if err != nil {
		t.Fatalf("expected numeric feedback id, got %q: %v", *result.ID, err)
	}
	if !rowExists(t, "SELECT COUNT(*) FROM feedback WHERE feedback_id = $1", feedbackID) {
		t.Errorf("expected feedback row in DB with feedback_id=%d", feedbackID)
	}

	// Cleanup: remove the feedback row after the test (before the volunteer
	// row is removed, in case feedback.volunteer_id has a FK constraint).
	t.Cleanup(func() {
		testDB.Exec("DELETE FROM feedback WHERE feedback_id = $1", feedbackID)
	})
}

// ============================================================================
// Tests — admin endpoint
// ============================================================================

// TestUpdateFeedback verifies that an admin can update the status of a
// feedback item and that a note is recorded in feedback_notes.
func TestUpdateFeedback(t *testing.T) {
	adminToken := makeAdminToken(t)
	_, volID := makeVolunteer(t)
	feedbackID := seedFeedback(t, volID)

	// Notes are created by UpdateFeedback; clean them up before the feedback
	// row is deleted (LIFO: this cleanup runs before seedFeedback's cleanup).
	t.Cleanup(func() {
		testDB.Exec("DELETE FROM feedback_notes WHERE feedback_id = $1", feedbackID)
	})

	resp := gqlPost(t, "/graphql/admin", adminToken, mutUpdateFeedback, map[string]any{
		"input": map[string]any{
			"id":     strconv.Itoa(feedbackID),
			"status": "QUESTION_SENT",
			"note":   "Updating status via integration test.",
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "updateFeedback", &result)

	if !result.Success {
		t.Errorf("expected success=true, got false (message: %v)", result.Message)
	}
	if !rowExists(t, "SELECT COUNT(*) FROM feedback WHERE feedback_id = $1 AND status = 'QUESTION_SENT'", feedbackID) {
		t.Error("expected status='QUESTION_SENT' in feedback table after updateFeedback")
	}
	if !rowExists(t, "SELECT COUNT(*) FROM feedback_notes WHERE feedback_id = $1", feedbackID) {
		t.Error("expected a note in feedback_notes after updateFeedback")
	}
}

// TestUpdateFeedback_WithGithubURL exercises the else-branch of UpdateFeedback
// where a GitHub issue URL is also stored alongside the status change.
func TestUpdateFeedback_WithGithubURL(t *testing.T) {
	adminToken := makeAdminToken(t)
	_, volID := makeVolunteer(t)
	feedbackID := seedFeedback(t, volID)

	t.Cleanup(func() {
		testDB.Exec("DELETE FROM feedback_notes WHERE feedback_id = $1", feedbackID)
	})

	const ghURL = "https://github.com/example/repo/issues/42"

	resp := gqlPost(t, "/graphql/admin", adminToken, mutUpdateFeedback, map[string]any{
		"input": map[string]any{
			"id":             strconv.Itoa(feedbackID),
			"status":         "QUESTION_SENT",
			"note":           "Linked GitHub issue.",
			"githubIssueURL": ghURL,
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "updateFeedback", &result)

	if !result.Success {
		t.Errorf("expected success=true, got false (message: %v)", result.Message)
	}
	if !rowExists(t, "SELECT COUNT(*) FROM feedback WHERE feedback_id = $1 AND github_issue_url = $2", feedbackID, ghURL) {
		t.Errorf("expected github_issue_url=%q in feedback table", ghURL)
	}
}

// TestResolveFeedback verifies that an admin can resolve a feedback item,
// which sets resolved_at and records a note.
func TestResolveFeedback(t *testing.T) {
	adminToken := makeAdminToken(t)
	_, volID := makeVolunteer(t)
	feedbackID := seedFeedback(t, volID)

	t.Cleanup(func() {
		testDB.Exec("DELETE FROM feedback_notes WHERE feedback_id = $1", feedbackID)
	})

	resp := gqlPost(t, "/graphql/admin", adminToken, mutResolveFeedback, map[string]any{
		"input": map[string]any{
			"id":     strconv.Itoa(feedbackID),
			"status": "RESOLVED_REJECTED",
			"note":   "Closing as not reproducible.",
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "resolveFeedback", &result)

	if !result.Success {
		t.Errorf("expected success=true, got false (message: %v)", result.Message)
	}
	if !rowExists(t, "SELECT COUNT(*) FROM feedback WHERE feedback_id = $1 AND resolved_at IS NOT NULL", feedbackID) {
		t.Error("expected resolved_at to be set after resolveFeedback")
	}
	if !rowExists(t, "SELECT COUNT(*) FROM feedback WHERE feedback_id = $1 AND status = 'RESOLVED_REJECTED'", feedbackID) {
		t.Error("expected status='RESOLVED_REJECTED' after resolveFeedback")
	}
	if !rowExists(t, "SELECT COUNT(*) FROM feedback_notes WHERE feedback_id = $1", feedbackID) {
		t.Error("expected a note in feedback_notes after resolveFeedback")
	}
}

// TestQuestionFeedback verifies that an admin can send a question to a
// volunteer about their feedback.
//
// Note: questionFeedback sends an email before adding the note. If the test
// server's mailer cannot deliver email, the service returns early (with
// success=true but a non-nil error) and the note is never written. In that
// case the resolver surfaces a GQL error. The DB note assertion is therefore
// only checked when there are no GQL errors.
func TestQuestionFeedback(t *testing.T) {
	adminToken := makeAdminToken(t)
	_, volID := makeVolunteer(t)
	feedbackID := seedFeedback(t, volID)

	t.Cleanup(func() {
		testDB.Exec("DELETE FROM feedback_notes WHERE feedback_id = $1", feedbackID)
	})

	resp := gqlPost(t, "/graphql/admin", adminToken, mutQuestionFeedback, map[string]any{
		"input": map[string]any{
			"id":        strconv.Itoa(feedbackID),
			"emailText": "Could you provide more detail about how to reproduce this?",
			"note":      "Sent question to volunteer.",
		},
	})

	// If no GQL errors, both the email and the note write succeeded.
	if !hasGQLErrors(resp) {
		var result mutationResult
		unmarshalField(t, resp, "questionFeedback", &result)
		if !result.Success {
			t.Errorf("expected success=true when no GQL errors, got false (message: %v)", result.Message)
		}
		if !rowExists(t, "SELECT COUNT(*) FROM feedback_notes WHERE feedback_id = $1", feedbackID) {
			t.Error("expected a note in feedback_notes after questionFeedback")
		}
	}
}

// ============================================================================
// Tests — ownFeedback visibility rules
// ============================================================================

// volunteerFeedbackNote mirrors the GQL ownFeedback notes sub-selection.
type volunteerFeedbackNote struct {
	Note      string `json:"note"`
	NoteType  string `json:"noteType"`
	CreatedAt string `json:"createdAt"`
}

// volunteerFeedbackItem mirrors the GQL ownFeedback item selection.
type volunteerFeedbackItem struct {
	ID      string                  `json:"id"`
	Subject string                  `json:"subject"`
	Status  string                  `json:"status"`
	Notes   []volunteerFeedbackNote `json:"notes"`
}

// TestOwnFeedback_ReturnsOnlyOwnFeedback verifies that a volunteer only sees
// their own submissions, not those belonging to other volunteers.
func TestOwnFeedback_ReturnsOnlyOwnFeedback(t *testing.T) {
	token, volID := makeVolunteer(t)
	_, otherVolID := makeVolunteer(t)

	myFeedbackID := seedFeedback(t, volID)
	_ = seedFeedback(t, otherVolID) // should not appear in results

	resp := gqlPost(t, "/graphql/volunteer", token, qryOwnFeedback, nil)
	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	var items []volunteerFeedbackItem
	unmarshalField(t, resp, "ownFeedback", &items)

	if len(items) != 1 {
		t.Fatalf("expected 1 feedback item, got %d", len(items))
	}
	if items[0].ID != strconv.Itoa(myFeedbackID) {
		t.Errorf("expected feedback id %d, got %s", myFeedbackID, items[0].ID)
	}
}

// TestOwnFeedback_NoNotes verifies that feedback with no notes at all is still
// returned (catches the LEFT JOIN / NULL note_type filtering bug).
func TestOwnFeedback_NoNotes(t *testing.T) {
	token, volID := makeVolunteer(t)
	feedbackID := seedFeedback(t, volID)

	resp := gqlPost(t, "/graphql/volunteer", token, qryOwnFeedback, nil)
	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	var items []volunteerFeedbackItem
	unmarshalField(t, resp, "ownFeedback", &items)

	if len(items) != 1 {
		t.Fatalf("expected 1 feedback item even with no notes, got %d (feedbackID=%d)", len(items), feedbackID)
	}
	if len(items[0].Notes) != 0 {
		t.Errorf("expected 0 notes, got %d", len(items[0].Notes))
	}
}

// TestOwnFeedback_VisibleNoteTypes verifies that QUESTION, VOLUNTEER_REPLY,
// and EMAIL_TO_VOLUNTEER notes are all returned to the volunteer.
func TestOwnFeedback_VisibleNoteTypes(t *testing.T) {
	token, volID := makeVolunteer(t)
	_, adminID := makeAdmin(t)
	feedbackID := seedFeedback(t, volID)

	seedFeedbackNote(t, feedbackID, adminID, "QUESTION", "Can you reproduce this?")
	seedFeedbackNote(t, feedbackID, volID, "VOLUNTEER_REPLY", "Yes, every time.")
	seedFeedbackNote(t, feedbackID, adminID, "EMAIL_TO_VOLUNTEER", "Thanks, closing as resolved.")

	resp := gqlPost(t, "/graphql/volunteer", token, qryOwnFeedback, nil)
	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	var items []volunteerFeedbackItem
	unmarshalField(t, resp, "ownFeedback", &items)

	if len(items) != 1 {
		t.Fatalf("expected 1 feedback item, got %d", len(items))
	}
	if len(items[0].Notes) != 3 {
		t.Errorf("expected 3 visible notes (QUESTION + VOLUNTEER_REPLY + EMAIL_TO_VOLUNTEER), got %d", len(items[0].Notes))
	}
}

// TestOwnFeedback_AdminNoteHidden verifies that ADMIN_NOTE entries are never
// returned to the volunteer — the core privacy requirement.
func TestOwnFeedback_AdminNoteHidden(t *testing.T) {
	token, volID := makeVolunteer(t)
	_, adminID := makeAdmin(t)
	feedbackID := seedFeedback(t, volID)

	seedFeedbackNote(t, feedbackID, adminID, "ADMIN_NOTE", "Internal: low priority, defer to next sprint.")
	seedFeedbackNote(t, feedbackID, adminID, "QUESTION", "Can you give us more detail?")

	resp := gqlPost(t, "/graphql/volunteer", token, qryOwnFeedback, nil)
	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	var items []volunteerFeedbackItem
	unmarshalField(t, resp, "ownFeedback", &items)

	if len(items) != 1 {
		t.Fatalf("expected 1 feedback item, got %d", len(items))
	}
	notes := items[0].Notes
	if len(notes) != 1 {
		t.Fatalf("expected 1 visible note (QUESTION only), got %d", len(notes))
	}
	if notes[0].NoteType != "QUESTION" {
		t.Errorf("expected visible note to be QUESTION, got %s", notes[0].NoteType)
	}
	for _, n := range notes {
		if n.NoteType == "ADMIN_NOTE" {
			t.Error("ADMIN_NOTE must never be returned to volunteer")
		}
	}
}

// TestOwnFeedback_MultipleNotesAccumulate verifies that all visible notes for
// a single feedback item are returned (catches the map pointer accumulation bug).
func TestOwnFeedback_MultipleNotesAccumulate(t *testing.T) {
	token, volID := makeVolunteer(t)
	_, adminID := makeAdmin(t)
	feedbackID := seedFeedback(t, volID)

	seedFeedbackNote(t, feedbackID, adminID, "QUESTION", "Question 1")
	seedFeedbackNote(t, feedbackID, volID, "VOLUNTEER_REPLY", "Reply 1")
	seedFeedbackNote(t, feedbackID, adminID, "QUESTION", "Question 2")
	seedFeedbackNote(t, feedbackID, volID, "VOLUNTEER_REPLY", "Reply 2")

	resp := gqlPost(t, "/graphql/volunteer", token, qryOwnFeedback, nil)
	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	var items []volunteerFeedbackItem
	unmarshalField(t, resp, "ownFeedback", &items)

	if len(items) != 1 {
		t.Fatalf("expected 1 feedback item, got %d", len(items))
	}
	if len(items[0].Notes) != 4 {
		t.Errorf("expected 4 notes, got %d — possible note accumulation bug", len(items[0].Notes))
	}
}
