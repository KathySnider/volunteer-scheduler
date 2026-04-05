package services

// feedback_attachment_service.go
//
// Methods on FeedbackService that handle file attachments stored as bytea in
// the feedback_attachments table. Kept in a separate file to avoid touching
// the existing (bug-prone) feedback_services.go.

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strconv"
	"volunteer-scheduler/models"
)

const maxAttachmentBytes = 5 * 1024 * 1024 // 5 MB

// AttachFileToFeedback stores a binary file in the feedback_attachments table
// and returns a MutationResult with the new attachment_id.
//
// Returns success=false (no error) when:
//   - feedbackID does not exist in the feedback table
//   - the file exceeds the 5 MB limit
//
// Returns a non-nil error only for unexpected DB failures.
func (s *FeedbackService) AttachFileToFeedback(ctx context.Context, feedbackID int, filename string, mimeType string, data []byte) (*models.MutationResult, error) {

	if len(data) > maxAttachmentBytes {
		return &models.MutationResult{
			Success: false,
			Message: ptrString(fmt.Sprintf(
				"File is too large (%d bytes). Maximum allowed size is 5 MB.", len(data),
			)),
		}, nil
	}

	insert := `
		INSERT INTO feedback_attachments (
			feedback_id, 
			filename, 
			mime_type, 
			file_data, 
			file_size, 
			created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING attachment_id
	`

	var id int
	err := s.DB.QueryRowContext(ctx, insert,
		feedbackID, filename, mimeType, data, len(data),
	).Scan(&id)

	if err != nil {
		// A FK violation means the feedbackID doesn't exist.
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to store attachment. Verify the feedback ID is valid."),
		}, fmt.Errorf("AttachFileToFeedback: %w", err)
	}

	idStr := strconv.Itoa(id)
	return &models.MutationResult{
		Success: true,
		Message: ptrString("File attached successfully."),
		ID:      &idStr,
	}, nil
}

// fetchAttachmentsForFeedback returns metadata (no binary data) for all
// attachments belonging to the given feedback item, ordered oldest-first.
// Used to populate the attachments field on a Feedback GraphQL object.
func (s *FeedbackService) fetchAttachmentsForFeedback(ctx context.Context, feedbackID int) ([]*models.FeedbackAttachment, error) {

	query := `
		SELECT attachment_id, 
			filename, 
			mime_type, 
			file_size, 
			created_at
		FROM feedback_attachments
		WHERE feedback_id = $1
		ORDER BY created_at ASC
	`
	rows, err := s.DB.QueryContext(ctx, query, feedbackID)
	if err != nil {
		return nil, fmt.Errorf("FetchAttachmentsForFeedback: %w", err)
	}
	defer rows.Close()

	var attachments []*models.FeedbackAttachment
	for rows.Next() {
		var a models.FeedbackAttachment
		var id int
		if err := rows.Scan(&id, &a.Filename, &a.MimeType, &a.FileSize, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("FetchAttachmentsForFeedback scan: %w", err)
		}
		a.ID = strconv.Itoa(id)
		attachments = append(attachments, &a)
	}
	if attachments == nil {
		attachments = []*models.FeedbackAttachment{} // never return nil for a GQL list
	}
	return attachments, nil
}

// FetchAttachment fetches a single attachment including its binary content,
// returned as a Base64-encoded string so the client can reconstruct the file
// (e.g. display a screenshot inline).
func (s *FeedbackService) FetchAttachment(ctx context.Context, attachmentID int) (*models.AttachmentDownload, error) {

	query := `
		SELECT filename, 
			mime_type, 
			file_data
		FROM feedback_attachments
		WHERE attachment_id = $1
	`

	var filename, mimeType string
	var raw []byte

	err := s.DB.QueryRowContext(ctx, query, attachmentID).Scan(&filename, &mimeType, &raw)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("attachment %d not found", attachmentID)
	}
	if err != nil {
		return nil, fmt.Errorf("GetAttachmentData: %w", err)
	}

	return &models.AttachmentDownload{
		Filename: filename,
		MimeType: mimeType,
		Data:     base64.StdEncoding.EncodeToString(raw),
	}, nil
}
