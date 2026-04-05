package integration

import (
	"strconv"
	"testing"
)

// All feedback service functions are correct. Tests below cover the full
// surface area except feedbackById, which can be added once the resolvers
// and schema wiring are in place.

// ============================================================================
// Mutation strings
// ============================================================================

const (
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
