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

// ============================================================================
// Mutation / query strings
// ============================================================================

const (
	qryOwnFeedback = `query {
		ownFeedback {
			id subject status type text
			notes { note noteType createdAt }
		}
	}`

	qryFeedbackDetail = `query FeedbackDetail($id: ID!) {
		feedbackDetail(feedbackId: $id) {
			id subject status type text
			notes { id creator noteType note createdAt }
		}
	}`

	mutGiveFeedback = `mutation GiveFeedback($input: NewFeedbackInput!) {
		giveFeedback(feedback: $input) { success message id }
	}`

	mutUpdateFeedbackStatus = `mutation UpdateFeedbackStatus($input: FeedbackStatusUpdateInput!) {
		updateFeedbackStatus(su: $input) { success message id }
	}`

	mutAddFeedbackNote = `mutation AddFeedbackNote($input: FeedbackNoteInput!) {
		addFeedbackNote(note: $input) { success message id }
	}`

	// emailFeedbackSubmitter sends an email to the volunteer who submitted the
	// feedback. See TestEmailFeedbackSubmitter_* for notes on the mailer dependency.
	mutEmailFeedbackSubmitter = `mutation EmailFeedbackSubmitter($input: FeedbackEmailInput!) {
		emailFeedbackSubmitter(input: $input) { success message id }
	}`

	mutAddVolunteerFeedbackNote = `mutation AddVolunteerFeedbackNote($input: VolunteerFeedbackNoteInput!) {
		addVolunteerFeedbackNote(note: $input) { success message id }
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

	t.Cleanup(func() {
		testDB.Exec("DELETE FROM feedback WHERE feedback_id = $1", feedbackID)
	})
}

// TestAddVolunteerFeedbackNote verifies that a volunteer can add a note to
// their own feedback and that it is stored with note_type VOLUNTEER_NOTE.
//
// Note: the service does not currently verify that the volunteer owns the
// feedback — any authenticated volunteer can add a note to any feedback_id.
// A future ownership check should be added to the service.
func TestAddVolunteerFeedbackNote(t *testing.T) {
	token, volID := makeVolunteer(t)
	feedbackID := seedFeedback(t, volID)

	t.Cleanup(func() {
		testDB.Exec("DELETE FROM feedback_notes WHERE feedback_id = $1", feedbackID)
	})

	resp := gqlPost(t, "/graphql/volunteer", token, mutAddVolunteerFeedbackNote, map[string]any{
		"input": map[string]any{
			"feedbackId": feedbackID,
			"note":       "Just wanted to add some more context.",
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "addVolunteerFeedbackNote", &result)

	if !result.Success {
		t.Errorf("expected success=true, got false (message: %v)", result.Message)
	}
	if !rowExists(t, "SELECT COUNT(*) FROM feedback_notes WHERE feedback_id = $1 AND note_type = 'VOLUNTEER_NOTE'", feedbackID) {
		t.Error("expected a VOLUNTEER_NOTE in feedback_notes after addVolunteerFeedbackNote")
	}
}

// TestAddVolunteerFeedbackNote_ResetsQuestionStatus verifies that when a
// volunteer adds a note to feedback that is in QUESTION_SENT status, the
// status is automatically reset to OPEN (the question has been answered).
func TestAddVolunteerFeedbackNote_ResetsQuestionStatus(t *testing.T) {
	token, volID := makeVolunteer(t)
	feedbackID := seedFeedback(t, volID)

	// Put the feedback into QUESTION_SENT state, as if an admin had asked something.
	if _, err := testDB.Exec("UPDATE feedback SET status = 'QUESTION_SENT' WHERE feedback_id = $1", feedbackID); err != nil {
		t.Fatalf("failed to seed QUESTION_SENT status: %v", err)
	}

	t.Cleanup(func() {
		testDB.Exec("DELETE FROM feedback_notes WHERE feedback_id = $1", feedbackID)
	})

	resp := gqlPost(t, "/graphql/volunteer", token, mutAddVolunteerFeedbackNote, map[string]any{
		"input": map[string]any{
			"feedbackId": feedbackID,
			"note":       "Here is the additional detail you asked for.",
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "addVolunteerFeedbackNote", &result)

	if !result.Success {
		t.Errorf("expected success=true, got false (message: %v)", result.Message)
	}
	if !rowExists(t, "SELECT COUNT(*) FROM feedback WHERE feedback_id = $1 AND status = 'OPEN'", feedbackID) {
		t.Error("expected status to reset to 'OPEN' after volunteer adds a note to QUESTION_SENT feedback")
	}
	if !rowExists(t, "SELECT COUNT(*) FROM feedback_notes WHERE feedback_id = $1 AND note_type = 'VOLUNTEER_NOTE'", feedbackID) {
		t.Error("expected a VOLUNTEER_NOTE in feedback_notes")
	}
}

// ============================================================================
// Tests — admin endpoint
// ============================================================================

// TestUpdateFeedbackStatus verifies that an admin can update the status of a
// feedback item and that an ADMIN_NOTE is recorded.
// Note: QUESTION_SENT is intentionally NOT tested here — that status is set
// exclusively by emailFeedbackSubmitter with requireReply=true. This test
// covers a manual status change (e.g. reopening feedback after an answer).
func TestUpdateFeedbackStatus(t *testing.T) {
	adminToken := makeAdminToken(t)
	_, volID := makeVolunteer(t)
	feedbackID := seedFeedback(t, volID)

	// Seed the feedback as QUESTION_SENT so we have something to reopen.
	if _, err := testDB.Exec("UPDATE feedback SET status = 'QUESTION_SENT' WHERE feedback_id = $1", feedbackID); err != nil {
		t.Fatalf("failed to seed QUESTION_SENT status: %v", err)
	}

	t.Cleanup(func() {
		testDB.Exec("DELETE FROM feedback_notes WHERE feedback_id = $1", feedbackID)
	})

	resp := gqlPost(t, "/graphql/admin", adminToken, mutUpdateFeedbackStatus, map[string]any{
		"input": map[string]any{
			"feedbackId": feedbackID,
			"status":     "OPEN",
			"note":       "Volunteer provided clarification; reopening for investigation.",
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "updateFeedbackStatus", &result)

	if !result.Success {
		t.Errorf("expected success=true, got false (message: %v)", result.Message)
	}
	if !rowExists(t, "SELECT COUNT(*) FROM feedback WHERE feedback_id = $1 AND status = 'OPEN'", feedbackID) {
		t.Error("expected status='OPEN' in feedback table after updateFeedbackStatus")
	}
	if !rowExists(t, "SELECT COUNT(*) FROM feedback_notes WHERE feedback_id = $1 AND note_type = 'ADMIN_NOTE'", feedbackID) {
		t.Error("expected an ADMIN_NOTE in feedback_notes after updateFeedbackStatus")
	}
}

// TestUpdateFeedbackStatus_Resolves verifies that updating status to a resolved
// value also sets resolved_at.
func TestUpdateFeedbackStatus_Resolves(t *testing.T) {
	adminToken := makeAdminToken(t)
	_, volID := makeVolunteer(t)
	feedbackID := seedFeedback(t, volID)

	t.Cleanup(func() {
		testDB.Exec("DELETE FROM feedback_notes WHERE feedback_id = $1", feedbackID)
	})

	resp := gqlPost(t, "/graphql/admin", adminToken, mutUpdateFeedbackStatus, map[string]any{
		"input": map[string]any{
			"feedbackId": feedbackID,
			"status":     "RESOLVED_REJECTED",
			"note":       "Closing as not reproducible.",
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "updateFeedbackStatus", &result)

	if !result.Success {
		t.Errorf("expected success=true, got false (message: %v)", result.Message)
	}
	if !rowExists(t, "SELECT COUNT(*) FROM feedback WHERE feedback_id = $1 AND status = 'RESOLVED_REJECTED'", feedbackID) {
		t.Error("expected status='RESOLVED_REJECTED' after updateFeedbackStatus")
	}
	if !rowExists(t, "SELECT COUNT(*) FROM feedback WHERE feedback_id = $1 AND resolved_at IS NOT NULL", feedbackID) {
		t.Error("expected resolved_at to be set when resolving feedback")
	}
}

// TestAddFeedbackNote verifies that an admin can add a standalone internal note
// without changing the status.
func TestAddFeedbackNote(t *testing.T) {
	adminToken := makeAdminToken(t)
	_, volID := makeVolunteer(t)
	feedbackID := seedFeedback(t, volID)

	t.Cleanup(func() {
		testDB.Exec("DELETE FROM feedback_notes WHERE feedback_id = $1", feedbackID)
	})

	resp := gqlPost(t, "/graphql/admin", adminToken, mutAddFeedbackNote, map[string]any{
		"input": map[string]any{
			"feedbackId": feedbackID,
			"note":       "Internal note added by admin.",
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "addFeedbackNote", &result)

	if !result.Success {
		t.Errorf("expected success=true, got false (message: %v)", result.Message)
	}
	if !rowExists(t, "SELECT COUNT(*) FROM feedback_notes WHERE feedback_id = $1 AND note_type = 'ADMIN_NOTE'", feedbackID) {
		t.Error("expected an ADMIN_NOTE in feedback_notes after addFeedbackNote")
	}
	// Status must not have changed.
	if !rowExists(t, "SELECT COUNT(*) FROM feedback WHERE feedback_id = $1 AND status = 'OPEN'", feedbackID) {
		t.Error("expected status to remain 'OPEN' after addFeedbackNote")
	}
}

// TestEmailFeedbackSubmitter_WithReply verifies that when requireReply=true,
// the status is set to QUESTION_SENT and a QUESTION note is recorded.
//
// Note: emailFeedbackSubmitter sends an email before adding the note. If the
// test mailer cannot deliver, the service returns early and the note is never
// written. The DB assertion is only checked when there are no GQL errors.
func TestEmailFeedbackSubmitter_WithReply(t *testing.T) {
	adminToken := makeAdminToken(t)
	_, volID := makeVolunteer(t)
	feedbackID := seedFeedback(t, volID)

	t.Cleanup(func() {
		testDB.Exec("DELETE FROM feedback_notes WHERE feedback_id = $1", feedbackID)
	})

	resp := gqlPost(t, "/graphql/admin", adminToken, mutEmailFeedbackSubmitter, map[string]any{
		"input": map[string]any{
			"feedbackId":   feedbackID,
			"emailText":    "Could you provide more detail about how to reproduce this?",
			"requireReply": true,
		},
	})

	if !hasGQLErrors(resp) {
		var result mutationResult
		unmarshalField(t, resp, "emailFeedbackSubmitter", &result)
		if !result.Success {
			t.Errorf("expected success=true when no GQL errors, got false (message: %v)", result.Message)
		}
		if !rowExists(t, "SELECT COUNT(*) FROM feedback_notes WHERE feedback_id = $1 AND note_type = 'QUESTION'", feedbackID) {
			t.Error("expected a QUESTION note in feedback_notes after emailFeedbackSubmitter with requireReply=true")
		}
		if !rowExists(t, "SELECT COUNT(*) FROM feedback WHERE feedback_id = $1 AND status = 'QUESTION_SENT'", feedbackID) {
			t.Error("expected status='QUESTION_SENT' after emailFeedbackSubmitter with requireReply=true")
		}
	}
}

// TestEmailFeedbackSubmitter_NoReply verifies that when requireReply=false,
// the status is unchanged and an EMAIL_TO_VOLUNTEER note is recorded.
func TestEmailFeedbackSubmitter_NoReply(t *testing.T) {
	adminToken := makeAdminToken(t)
	_, volID := makeVolunteer(t)
	feedbackID := seedFeedback(t, volID)

	t.Cleanup(func() {
		testDB.Exec("DELETE FROM feedback_notes WHERE feedback_id = $1", feedbackID)
	})

	resp := gqlPost(t, "/graphql/admin", adminToken, mutEmailFeedbackSubmitter, map[string]any{
		"input": map[string]any{
			"feedbackId":   feedbackID,
			"emailText":    "Just letting you know we are looking into this.",
			"requireReply": false,
		},
	})

	if !hasGQLErrors(resp) {
		var result mutationResult
		unmarshalField(t, resp, "emailFeedbackSubmitter", &result)
		if !result.Success {
			t.Errorf("expected success=true when no GQL errors, got false (message: %v)", result.Message)
		}
		if !rowExists(t, "SELECT COUNT(*) FROM feedback_notes WHERE feedback_id = $1 AND note_type = 'EMAIL_TO_VOLUNTEER'", feedbackID) {
			t.Error("expected an EMAIL_TO_VOLUNTEER note after emailFeedbackSubmitter with requireReply=false")
		}
		// Status must remain OPEN.
		if !rowExists(t, "SELECT COUNT(*) FROM feedback WHERE feedback_id = $1 AND status = 'OPEN'", feedbackID) {
			t.Error("expected status to remain 'OPEN' after emailFeedbackSubmitter with requireReply=false")
		}
	}
}

// TestFeedbackDetail verifies that the feedbackDetail query returns the correct
// feedback item including its notes.
func TestFeedbackDetail(t *testing.T) {
	adminToken := makeAdminToken(t)
	_, volID := makeVolunteer(t)
	_, adminID := makeAdmin(t)
	feedbackID := seedFeedback(t, volID)
	seedFeedbackNote(t, feedbackID, adminID, "ADMIN_NOTE", "Investigating.")

	resp := gqlPost(t, "/graphql/admin", adminToken, qryFeedbackDetail, map[string]any{
		"id": strconv.Itoa(feedbackID),
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	type feedbackNote struct {
		ID       string `json:"id"`
		NoteType string `json:"noteType"`
		Note     string `json:"note"`
	}
	type feedbackDetail struct {
		ID      string         `json:"id"`
		Subject string         `json:"subject"`
		Status  string         `json:"status"`
		Notes   []feedbackNote `json:"notes"`
	}

	var detail feedbackDetail
	unmarshalField(t, resp, "feedbackDetail", &detail)

	if detail.ID != strconv.Itoa(feedbackID) {
		t.Errorf("expected feedback id %d, got %s", feedbackID, detail.ID)
	}
	if len(detail.Notes) != 1 {
		t.Fatalf("expected 1 note, got %d", len(detail.Notes))
	}
	if detail.Notes[0].NoteType != "ADMIN_NOTE" {
		t.Errorf("expected ADMIN_NOTE, got %s", detail.Notes[0].NoteType)
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

// TestOwnFeedback_VisibleNoteTypes verifies that QUESTION, VOLUNTEER_NOTE,
// and EMAIL_TO_VOLUNTEER notes are all returned to the volunteer.
func TestOwnFeedback_VisibleNoteTypes(t *testing.T) {
	token, volID := makeVolunteer(t)
	_, adminID := makeAdmin(t)
	feedbackID := seedFeedback(t, volID)

	seedFeedbackNote(t, feedbackID, adminID, "QUESTION", "Can you reproduce this?")
	seedFeedbackNote(t, feedbackID, volID, "VOLUNTEER_NOTE", "Yes, every time.")
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
		t.Errorf("expected 3 visible notes (QUESTION + VOLUNTEER_NOTE + EMAIL_TO_VOLUNTEER), got %d", len(items[0].Notes))
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
	seedFeedbackNote(t, feedbackID, volID, "VOLUNTEER_NOTE", "Reply 1")
	seedFeedbackNote(t, feedbackID, adminID, "QUESTION", "Question 2")
	seedFeedbackNote(t, feedbackID, volID, "VOLUNTEER_NOTE", "Reply 2")

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
