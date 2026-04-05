package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strconv"
	"testing"
)

// ============================================================================
// Multipart upload helper
// ============================================================================

// gqlFilePost sends a GraphQL mutation that includes a file upload using the
// GraphQL multipart request spec:
// https://github.com/jaydenseric/graphql-multipart-request-spec
//
// query must use a variable named "file" of type Upload.
// variables should contain all other (non-file) variables; the "file" entry
// is set to null automatically and mapped to the actual multipart part.
//
// filename and mimeType describe the file being uploaded.
// fileContent is the raw bytes of the file.
func gqlFilePost(
	t *testing.T,
	path, token, query string,
	variables map[string]any,
	filename, mimeType string,
	fileContent []byte,
) gqlResponse {
	t.Helper()

	// Clone variables and set file to null (required by the spec).
	vars := make(map[string]any, len(variables)+1)
	for k, v := range variables {
		vars[k] = v
	}
	vars["file"] = nil

	operations, err := json.Marshal(map[string]any{
		"query":     query,
		"variables": vars,
	})
	if err != nil {
		t.Fatalf("gqlFilePost: marshal operations: %v", err)
	}

	// The map field tells the server which part corresponds to which variable.
	mapField, err := json.Marshal(map[string][]string{
		"0": {"variables.file"},
	})
	if err != nil {
		t.Fatalf("gqlFilePost: marshal map: %v", err)
	}

	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	// Part 1: operations
	_ = w.WriteField("operations", string(operations))

	// Part 2: map
	_ = w.WriteField("map", string(mapField))

	// Part 3: the actual file (part key "0" matches the map above)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="0"; filename="%s"`, filename))
	h.Set("Content-Type", mimeType)
	part, err := w.CreatePart(h)
	if err != nil {
		t.Fatalf("gqlFilePost: create file part: %v", err)
	}
	if _, err = part.Write(fileContent); err != nil {
		t.Fatalf("gqlFilePost: write file part: %v", err)
	}
	w.Close()

	req, err := http.NewRequest(http.MethodPost, testServer.URL+path, &body)
	if err != nil {
		t.Fatalf("gqlFilePost: create request: %v", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("gqlFilePost: do request: %v", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("gqlFilePost: read response: %v", err)
	}

	var result gqlResponse
	if err := json.Unmarshal(respBytes, &result); err != nil {
		t.Fatalf("gqlFilePost: unmarshal response: %v\nbody: %s", err, respBytes)
	}
	return result
}

// ============================================================================
// Query / mutation strings
// ============================================================================

const (
	mutAttachFile = `mutation AttachFile($feedbackId: ID!, $file: Upload!) {
		attachFileToFeedback(feedbackId: $feedbackId, file: $file) { success message id }
	}`

	qryGetAttachment = `query GetAttachment($attachmentId: ID!) {
		attachment(attachmentId: $attachmentId) { filename mimeType data }
	}`
)

// ============================================================================
// Local response types
// ============================================================================

type attachmentDownloadResult struct {
	Filename string `json:"filename"`
	MimeType string `json:"mimeType"`
	Data     string `json:"data"` // Base64-encoded file content
}

// ============================================================================
// Tests
// ============================================================================

// seedFeedback inserts a feedback row directly and returns the feedback_id.
// Cleanup removes the row (and cascade-deletes any attachments) when done.
func seedFeedback(t *testing.T, volID int) int {
	t.Helper()
	var id int
	err := testDB.QueryRow(`
		INSERT INTO feedback (volunteer_id, feedback_type, status, subject, app_page_name, text, created_at)
		VALUES ($1, 'BUG', 'OPEN', 'Test subject', 'TestPage', 'Test text', NOW())
		RETURNING feedback_id
	`, volID).Scan(&id)
	if err != nil {
		t.Fatalf("seedFeedback: %v", err)
	}
	t.Cleanup(func() {
		testDB.Exec("DELETE FROM feedback_attachments WHERE feedback_id = $1", id)
		testDB.Exec("DELETE FROM feedback WHERE feedback_id = $1", id)
	})
	return id
}

// TestAttachFileToFeedback verifies that a volunteer can attach a PNG file
// to an existing feedback item and that the attachment row is persisted.
func TestAttachFileToFeedback(t *testing.T) {
	token, volID := makeVolunteer(t)
	feedbackID := seedFeedback(t, volID)

	// Minimal 1×1 white PNG — valid image, well under the 5 MB limit.
	pngBytes := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, // PNG signature
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52, // IHDR chunk length + type
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, // 1×1 px
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, // 8-bit RGB, CRC
		0xde, 0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41, // IDAT chunk
		0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00, // compressed pixel
		0x00, 0x00, 0x02, 0x00, 0x01, 0xe2, 0x21, 0xbc, // CRC
		0x33, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, // IEND chunk
		0x44, 0xae, 0x42, 0x60, 0x82, // IEND CRC
	}

	resp := gqlFilePost(
		t,
		"/graphql/volunteer",
		token,
		mutAttachFile,
		map[string]any{"feedbackId": fmt.Sprintf("%d", feedbackID)},
		"screenshot.png",
		"image/png",
		pngBytes,
	)

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "attachFileToFeedback", &result)

	if !result.Success {
		t.Errorf("expected success=true, got false (message: %v)", result.Message)
	}
	if result.ID == nil {
		t.Fatal("expected attachment id to be set")
	}

	attachmentID, err := strconv.Atoi(*result.ID)
	if err != nil {
		t.Fatalf("expected numeric attachment id, got %q: %v", *result.ID, err)
	}
	if !rowExists(t, "SELECT COUNT(*) FROM feedback_attachments WHERE attachment_id = $1", attachmentID) {
		t.Errorf("expected feedback_attachments row with id=%d", attachmentID)
	}
	if !rowExists(t, `
		SELECT COUNT(*) FROM feedback_attachments
		WHERE attachment_id = $1 AND filename = 'screenshot.png' AND mime_type = 'image/png'
	`, attachmentID) {
		t.Error("expected filename='screenshot.png' and mime_type='image/png' in DB")
	}
}

// TestAttachFileToFeedback_TooLarge verifies that the service rejects a file
// that exceeds the 5 MB limit, returning success=false with no GQL error.
func TestAttachFileToFeedback_TooLarge(t *testing.T) {
	token, volID := makeVolunteer(t)
	feedbackID := seedFeedback(t, volID)

	oversize := make([]byte, 5*1024*1024+1) // 5 MB + 1 byte

	resp := gqlFilePost(
		t,
		"/graphql/volunteer",
		token,
		mutAttachFile,
		map[string]any{"feedbackId": fmt.Sprintf("%d", feedbackID)},
		"toobig.png",
		"image/png",
		oversize,
	)

	if hasGQLErrors(resp) {
		t.Fatalf("unexpected GQL errors: %v", resp.Errors)
	}

	var result mutationResult
	unmarshalField(t, resp, "attachFileToFeedback", &result)

	if result.Success {
		t.Error("expected success=false for oversized file, got true")
	}
}

// TestGetAttachment verifies that an attached file can be retrieved as a
// Base64-encoded string via the getAttachment query.
func TestGetAttachment(t *testing.T) {
	token, volID := makeVolunteer(t)
	feedbackID := seedFeedback(t, volID)

	content := []byte("hello attachment")

	// Attach the file first.
	attachResp := gqlFilePost(
		t,
		"/graphql/volunteer",
		token,
		mutAttachFile,
		map[string]any{"feedbackId": fmt.Sprintf("%d", feedbackID)},
		"note.txt",
		"text/plain",
		content,
	)
	if hasGQLErrors(attachResp) {
		t.Fatalf("attach: unexpected GQL errors: %v", attachResp.Errors)
	}
	var attachResult mutationResult
	unmarshalField(t, attachResp, "attachFileToFeedback", &attachResult)
	if !attachResult.Success || attachResult.ID == nil {
		t.Fatalf("attach failed: success=%v message=%v", attachResult.Success, attachResult.Message)
	}

	// Now retrieve it.
	getResp := gqlPost(t, "/graphql/volunteer", token, qryGetAttachment, map[string]any{
		"attachmentId": *attachResult.ID,
	})
	if hasGQLErrors(getResp) {
		t.Fatalf("getAttachment: unexpected GQL errors: %v", getResp.Errors)
	}

	var dl attachmentDownloadResult
	unmarshalField(t, getResp, "attachment", &dl)

	if dl.Filename != "note.txt" {
		t.Errorf("expected filename=%q, got %q", "note.txt", dl.Filename)
	}
	if dl.MimeType != "text/plain" {
		t.Errorf("expected mimeType=%q, got %q", "text/plain", dl.MimeType)
	}
	if dl.Data == "" {
		t.Error("expected non-empty Base64 data in getAttachment response")
	}
}
