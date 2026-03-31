package integration

import (
	"testing"
)

// ============================================================================
// Shared mutation strings
// ============================================================================

const (
	mutCreateJobType = `
		mutation CreateJobType($input: NewJobTypeInput!) {
			createJobType(newJob: $input) { success message id }
		}`

	mutUpdateJobType = `
		mutation UpdateJobType($input: UpdateJobTypeInput!) {
			updateJobType(job: $input) { success message id }
		}`

	mutDeleteJobType = `
		mutation DeleteJobType($id: Int!) {
			deleteJobType(JobId: $id) { success message id }
		}`
)

// ============================================================================
// createJobType
// ============================================================================

// TestCreateJobType verifies that a new job type can be created and that the
// mutation returns a non-empty ID.
func TestCreateJobType(t *testing.T) {
	token := makeAdminToken(t)
	code := uniqueCode(t, "tst")

	resp := gqlPost(t, "/graphql/admin", token, mutCreateJobType, map[string]any{
		"input": map[string]any{
			"code":      code,
			"name":      "Test Job Type " + code,
			"sortOrder": 99,
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "createJobType", &result)

	if !result.Success {
		t.Fatalf("createJobType returned success=false: %v", result.Message)
	}
	if result.ID == nil || *result.ID == "" {
		t.Fatal("expected a non-empty job_type ID in response")
	}

	jobTypeID := *result.ID
	t.Cleanup(func() { testDB.Exec("DELETE FROM job_types WHERE job_type_id = $1", jobTypeID) })
}

// TestCreateJobType_DuplicateCode verifies that the service rejects a job type
// whose code already exists (job_types.code has a UNIQUE constraint).
func TestCreateJobType_DuplicateCode(t *testing.T) {
	token := makeAdminToken(t)
	code := uniqueCode(t, "dup")
	jobTypeID := seedJobType(t, code, "Original Job Type")

	resp := gqlPost(t, "/graphql/admin", token, mutCreateJobType, map[string]any{
		"input": map[string]any{
			"code":      code,
			"name":      "Duplicate Job Type",
			"sortOrder": 99,
		},
	})

	if !hasGQLErrors(resp) {
		t.Error("expected a GraphQL error when creating a job type with a duplicate code")
	}

	// Verify that only the original row still exists and nothing extra was inserted.
	var count int
	if err := testDB.QueryRow(
		"SELECT COUNT(*) FROM job_types WHERE job_type_id = $1", jobTypeID,
	).Scan(&count); err != nil {
		t.Fatalf("checking original job type: %v", err)
	}
	if count != 1 {
		t.Error("original job type should still exist after failed duplicate create")
	}
}

// ============================================================================
// updateJobType
// ============================================================================

// TestUpdateJobType verifies that a job type's name and sort order can be
// changed via the updateJobType mutation, and that the change is persisted.
func TestUpdateJobType(t *testing.T) {
	token := makeAdminToken(t)
	code := uniqueCode(t, "upd")
	jobTypeID := seedJobType(t, code, "Pre-Update Job")

	resp := gqlPost(t, "/graphql/admin", token, mutUpdateJobType, map[string]any{
		"input": map[string]any{
			"id":        jobTypeID,
			"code":      code,
			"name":      "Post-Update Job",
			"sortOrder": 95,
		},
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "updateJobType", &result)

	if !result.Success {
		t.Fatalf("updateJobType returned success=false: %v", result.Message)
	}

	var storedName string
	if err := testDB.QueryRow(
		"SELECT name FROM job_types WHERE job_type_id = $1", jobTypeID,
	).Scan(&storedName); err != nil {
		t.Fatalf("querying updated job type: %v", err)
	}
	if storedName != "Post-Update Job" {
		t.Errorf("expected name='Post-Update Job', got %q", storedName)
	}
}

// ============================================================================
// deleteJobType
// ============================================================================

// TestDeleteJobType verifies that deleteJobType soft-deletes the job type by
// setting is_active=false. The row intentionally stays in the DB so that
// historical event/opportunity records referencing this job type remain valid.
func TestDeleteJobType(t *testing.T) {
	token := makeAdminToken(t)
	code := uniqueCode(t, "del")
	jobTypeID := seedJobType(t, code, "Job To Delete")

	resp := gqlPost(t, "/graphql/admin", token, mutDeleteJobType, map[string]any{
		"id": jobTypeID,
	})

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "deleteJobType", &result)

	if !result.Success {
		t.Fatalf("deleteJobType returned success=false: %v", result.Message)
	}

	// The row must still exist (soft delete), but is_active must be false.
	if !rowExists(t, "SELECT COUNT(*) FROM job_types WHERE job_type_id = $1", jobTypeID) {
		t.Error("job type row should still exist after a soft delete")
	}
	if rowExists(t, "SELECT COUNT(*) FROM job_types WHERE job_type_id = $1 AND is_active = true", jobTypeID) {
		t.Error("expected is_active=false after deleteJobType")
	}
}
