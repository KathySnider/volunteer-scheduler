package services

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"volunteer-scheduler/models"
)

// services/feedback_service.go

type FeedbackService struct {
	DB     *sql.DB
	Mailer *Mailer
}

func NewFeedbackService(db *sql.DB, mailer *Mailer) *FeedbackService {
	s := &FeedbackService{
		DB:     db,
		Mailer: mailer,
	}
	return s
}

// Queries.

func (s *FeedbackService) FetchOwnFeedback(ctx context.Context, volId int) ([]*models.VolunteerFeedback, error) {
	query := `
        SELECT
            f.feedback_id,
			f.feedback_type,
			f.status,
			f.subject,
			f.app_page_name,
			f.text,
			fn.note,
			fn.note_type,
			f.github_issue_url,
			f.created_at,
			fn.created_at
			from feedback f
			LEFT JOIN feedback_notes fn ON fn.feedback_id = f.feedback_id
		WHERE f.volunteer_id = $1 AND (fn.note_type IS NULL OR fn.note_type != 'ADMIN_NOTE')
		ORDER BY f.created_at, fn.created_at
    `

	rows, err := s.DB.QueryContext(ctx, query, volId)
	if err != nil {
		return nil, fmt.Errorf("error querying feedback: %w", err)
	}
	defer rows.Close()

	feedbackMap := make(map[int]*models.VolunteerFeedback)

	// orderedFB preserves the ORDER BY from the SQL query so the caller
	// can reassemble the slice in the correct order after map operations.
	orderedFB := make([]int, 0)

	for rows.Next() {
		var fb models.VolunteerFeedback
		var fbInt int
		var fbType, fbStatus string
		var note, noteType, noteCreatedAt, url sql.NullString

		err := rows.Scan(
			&fbInt,
			&fbType,
			&fbStatus,
			&fb.Subject,
			&fb.AppPageName,
			&fb.Text,
			&note,
			&noteType,
			&url,
			&fb.CreatedAt,
			&noteCreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning feedback: %w", err)
		}

		// There will be multiple rows for the same ID when there are multiple notes.
		_, exists := feedbackMap[fbInt]
		if !exists {
			// Finish filling out the fields in the NEW entry
			// and add it to the map.
			fb.ID = strconv.Itoa(fbInt)
			fb.Type = models.FeedbackType(fbType)
			fb.Status = models.FeedbackStatus(fbStatus)
			if url.Valid {
				fb.GithubIssueURL = &url.String
			}
			feedbackMap[fbInt] = &fb

			// Since this is the first time we've seen this id, we want to record it
			// to preserve the sort order.
			orderedFB = append(orderedFB, fbInt)
		}

		// Use the pointer from the map directly so that notes accumulate on
		// the heap-allocated struct, not on a throwaway value copy.
		fbPtr := feedbackMap[fbInt]

		if note.Valid {
			// If there is a note, the fields from feedback_notes must also be
			// valid, because they are NOT NULL. We don't need to check each one.
			fbPtr.Notes = append(fbPtr.Notes, &models.VolunteerFeedbackNote{
				Note:      note.String,
				NoteType:  models.FeedbackNoteType(noteType.String),
				CreatedAt: noteCreatedAt.String,
			})
		}

		fbPtr.Attachments, err = s.fetchAttachmentsForFeedback(ctx, fbInt)
		if err != nil {
			return nil, fmt.Errorf("error fetching attachments for feedback with id %d: %w", fbInt, err)
		}
	}

	// Build the feedback slice in the order the DB returned the feedback.
	feedback := make([]*models.VolunteerFeedback, 0, len(feedbackMap))
	for _, id := range orderedFB {
		feedback = append(feedback, feedbackMap[id])
	}

	return feedback, nil
}

func (s *FeedbackService) FetchFeedback(ctx context.Context, filter *models.FeedbackFilterInput) ([]*models.Feedback, error) {

	// In the JOINS below, am using fc as the feedback creator and nc as the
	// notes creator.
	query := `
        SELECT 
            f.feedback_id,
			fc.first_name,
			fc.last_name,
			f.feedback_type,
			f.status,
			f.subject,
			f.app_page_name,
			f.text,
			nc.first_name,
			nc.last_name,
			fn.note,
			fn.created_at,
			f.github_issue_url,
			f.created_at,
			f.last_updated_at,
			f.resolved_at
			from feedback f
			LEFT JOIN volunteers fc ON fc.volunteer_id = f.volunteer_id
			LEFT JOIN feedback_notes fn ON fn.feedback_id = f.feedback_id
			LEFT JOIN volunteers nc ON nc.volunteer_id = fn.volunteer_id
		WHERE 1=1
    `
	if filter != nil {
		if filter.Status != nil {
			dbstatus := strings.ToLower(string(*filter.Status))
			query += " AND f.status = " + dbstatus
		}

		if filter.Type != nil {
			dbtype := strings.ToLower(string(*filter.Type))
			query += " AND f.feedback_type = " + dbtype
		}
	}
	query += " ORDER BY f.created_at, fn.created_at"

	rows, err := s.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error querying feedback: %w", err)
	}
	defer rows.Close()

	feedbackMap := make(map[int]*models.Feedback)

	// orderedFB preserves the ORDER BY from the SQL query so the caller
	// can reassemble the slice in the correct order after map operations.
	orderedFB := make([]int, 0)

	for rows.Next() {
		var fb models.Feedback
		var fbInt int
		var fcFname, fcLname string
		var fbType, fbStatus string
		var ncFname, ncLname, note, noteCreatedAt sql.NullString
		var url, lastUpdateAt, resolvedAt sql.NullString

		err := rows.Scan(
			&fbInt,
			&fcFname,
			&fcLname,
			&fbType,
			&fbStatus,
			&fb.Subject,
			&fb.AppPageName,
			&fb.Text,
			&ncFname,
			&ncLname,
			&note,
			&noteCreatedAt,
			&url,
			&fb.CreatedAt,
			&lastUpdateAt,
			&resolvedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning feedback: %w", err)
		}

		_, exists := feedbackMap[fbInt]
		if !exists {
			// Finish filling out the fields in the NEW entry
			// and add it to the map.
			fb.ID = strconv.Itoa(fbInt)
			fb.VolunteerName = fcFname + " " + fcLname
			fb.Type = models.FeedbackType(fbType)
			fb.Status = models.FeedbackStatus(fbStatus)
			if url.Valid {
				fb.GithubIssueURL = &url.String
			}
			if lastUpdateAt.Valid {
				fb.LastUpdatedAt = &lastUpdateAt.String
			}
			if resolvedAt.Valid {
				fb.ResolvedAt = &resolvedAt.String
			}
			feedbackMap[fbInt] = &fb

			// Since this is the first time we've seen this id, we want to record it
			// to preserve the sort order.
			orderedFB = append(orderedFB, fbInt)
		}

		// Use the pointer from the map directly so that notes accumulate on
		// the heap-allocated struct, not on a throwaway value copy.
		fbPtr := feedbackMap[fbInt]

		if note.Valid {
			// If there is a note, the fields from feedback_notes must also be
			// valid, because they are NOT NULL.
			fbPtr.Notes = append(fbPtr.Notes, &models.FeedbackNote{
				Note:      note.String,
				Creator:   ncFname.String + " " + ncLname.String,
				CreatedAt: noteCreatedAt.String,
			})
		}

		fbPtr.Attachments, err = s.fetchAttachmentsForFeedback(ctx, fbInt)
		if err != nil {
			return nil, fmt.Errorf("error fetching attachments for feedback with id %d: %w", fbInt, err)
		}
	}

	// Build the feedback slice in the order the DB returned the feedback.
	feedback := make([]*models.Feedback, 0, len(feedbackMap))
	for _, id := range orderedFB {
		feedback = append(feedback, feedbackMap[id])
	}

	return feedback, nil
}

func (s *FeedbackService) FetchFeedbackById(ctx context.Context, feedbackId string) (*models.Feedback, error) {

	fbInt, err := strconv.Atoi(feedbackId)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch feedback; invalid Id %s: %w", feedbackId, err)
	}

	query := `
        SELECT 
			fc.first_name,
			fc.last_name,
			f.feedback_type,
			f.status,
			f.subject,
			f.app_page_name,
			f.text,
			f.github_issue_url,
			f.created_at,
			f.last_updated_at,
			f.resolved_at
			from feedback f
			LEFT JOIN volunteers fc ON fc.volunteer_id = f.volunteer_id
		WHERE f.feedback_id = $1
    `
	row := s.DB.QueryRowContext(ctx, query, fbInt)

	var feedback models.Feedback
	var fcFname, fcLname string
	var fbType, fbStatus string
	var url, lastUpdateAt, resolvedAt sql.NullString

	err = row.Scan(
		&fcFname,
		&fcLname,
		&fbType,
		&fbStatus,
		&feedback.Subject,
		&feedback.AppPageName,
		&feedback.Text,
		&url,
		&feedback.CreatedAt,
		&lastUpdateAt,
		&resolvedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("error scanning feedback: %w", err)
	}

	// Fill out the fields, sans notes.
	feedback.ID = strconv.Itoa(fbInt)
	feedback.VolunteerName = fcFname + " " + fcLname
	feedback.Type = models.FeedbackType(fbType)
	feedback.Status = models.FeedbackStatus(fbStatus)
	if url.Valid {
		feedback.GithubIssueURL = &url.String
	}
	if lastUpdateAt.Valid {
		feedback.LastUpdatedAt = &lastUpdateAt.String
	}
	if resolvedAt.Valid {
		feedback.ResolvedAt = &resolvedAt.String
	}
	feedback.Notes = make([]*models.FeedbackNote, 0, 5) // If there are more than 5 notes (unlikely), the slice will increase by 5 slots.

	// Get all of the notes created for this feedback, along
	// with the creator of each note and the time it was created.
	query = `
        SELECT 
			nc.first_name,
			nc.last_name,
			fn.note,
			fn.created_at
			from feedback_notes fn
			LEFT JOIN volunteers nc ON nc.volunteer_id = fn.volunteer_id
		WHERE fn.feedback_id = $1
		ORDER BY fn.created_at
     `

	rows, err := s.DB.QueryContext(ctx, query, fbInt)
	if err != nil {
		return nil, fmt.Errorf("error querying feedback: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var note models.FeedbackNote
		var ncFname, ncLname string
		err := rows.Scan(
			&ncFname,
			&ncLname,
			&note.Note,
			&note.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning feedback: %w", err)
		}

		note.Creator = ncFname + " " + ncLname

		feedback.Notes = append(feedback.Notes, &note)

	}
	feedback.Attachments, err = s.fetchAttachmentsForFeedback(ctx, fbInt)
	if err != nil {
		return nil, fmt.Errorf("error getting attachments: %w", err)
	}

	return &feedback, nil
}

// Mutations

func (s *FeedbackService) CreateNewFeedback(ctx context.Context, creatorId int, feedback models.NewFeedbackInput) (*models.MutationResult, error) {

	insert := `
		INSERT INTO feedback (
			volunteer_id, 
			feedback_type, 
			subject, 
			app_page_name,
			text,
			created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING feedback_id
	`
	var fbInt int

	err := s.DB.QueryRowContext(ctx, insert, creatorId, feedback.Type, feedback.Subject, feedback.AppPageName, feedback.Text).Scan(&fbInt)
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to create feedback entry in DB."),
			ID:      nil,
		}, err
	}

	fbStr := strconv.Itoa(fbInt)
	return &models.MutationResult{
		Success: true,
		Message: ptrString("Successfully created feedback."),
		ID:      &fbStr,
	}, err
}

// A user has created feedback. An admin wants to ask a question.
func (s *FeedbackService) QuestionFeedback(ctx context.Context, adminId int, question models.QuestionFeedbackInput) (*models.MutationResult, error) {

	fbInt, err := strconv.Atoi(question.ID)
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to send question; invalid feedback Id."),
			ID:      &question.ID,
		}, err
	}

	// Get the information about the user who created the feedback.
	query := `
		SELECT 
			fc.email, 
			f.subject
		FROM feedback f
		LEFT JOIN volunteers fc ON fc.volunteer_id = f.volunteer_id
		WHERE f.feedback_id = $1
	`
	row := s.DB.QueryRowContext(ctx, query, fbInt)

	var email, subject string
	err = row.Scan(
		&email,
		&subject,
	)
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to send question."),
			ID:      nil,
		}, fmt.Errorf("error scanning feedback row: %w", err)
	}

	subject = "re: " + subject
	err = s.Mailer.SendEmail(ctx, email, subject, "", question.EmailText)
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to send email to volunteer."),
			ID:      &question.ID,
		}, err
	}

	// Store the email text as a QUESTION note — visible to the volunteer.
	err = addNoteToFeedback(ctx, s.DB, fbInt, adminId, question.EmailText, "QUESTION")
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Sent email. Failed to add question note to feedback."),
			ID:      &question.ID,
		}, err
	}

	// Store the internal admin note (not visible to volunteer).
	err = addNoteToFeedback(ctx, s.DB, fbInt, adminId, question.Note, "ADMIN_NOTE")
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Sent email. Failed to add internal note to feedback."),
			ID:      &question.ID,
		}, err
	}

	// Update the feedback status to QUESTION_SENT.
	_, err = s.DB.ExecContext(ctx, "UPDATE feedback SET status = 'QUESTION_SENT', last_updated_at = NOW() WHERE feedback_id = $1", fbInt)
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Sent question. Failed to update feedback status."),
			ID:      &question.ID,
		}, err
	}

	return &models.MutationResult{
		Success: true,
		Message: ptrString("Successfully sent question to feedback's creator."),
		ID:      &question.ID,
	}, err
}

func (s *FeedbackService) UpdateFeedback(ctx context.Context, adminId int, update models.UpdateFeedbackInput) (*models.MutationResult, error) {

	// Convert the id of the feedback.
	fbInt, err := strconv.Atoi(update.ID)
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to update DB; invalid feedback Id."),
			ID:      &update.ID,
		}, err
	}
	err = addNoteToFeedback(ctx, s.DB, fbInt, adminId, update.Note, "ADMIN_NOTE")
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to update DB."),
			ID:      &update.ID,
		}, err
	}

	if update.GithubIssueURL == nil {

		_, err = s.DB.ExecContext(ctx, "UPDATE feedback SET status = $1, last_updated_at = NOW() WHERE feedback_id = $2", update.Status, fbInt)
		if err != nil {
			return &models.MutationResult{
				Success: false,
				Message: ptrString("Failed to update DB."),
				ID:      &update.ID,
			}, err
		}
	} else {

		_, err = s.DB.ExecContext(ctx, "UPDATE feedback SET status = $1, github_issue_url = $2, last_updated_at = NOW() WHERE feedback_id = $3", update.Status, update.GithubIssueURL, fbInt)
		if err != nil {
			return &models.MutationResult{
				Success: false,
				Message: ptrString("Failed to update DB."),
				ID:      &update.ID,
			}, err
		}
	}

	return &models.MutationResult{
		Success: true,
		Message: ptrString("Successfully updated feedback."),
		ID:      &update.ID,
	}, err
}

func (s *FeedbackService) ResolveFeedback(ctx context.Context, adminId int, resolution models.ResolveFeedbackInput) (*models.MutationResult, error) {
	// Convert the id of the feedback.
	fbInt, err := strconv.Atoi(resolution.ID)
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to resolve feedback; invalid feedback Id."),
			ID:      &resolution.ID,
		}, err
	}

	err = addNoteToFeedback(ctx, s.DB, fbInt, adminId, resolution.Note, "ADMIN_NOTE")
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to resolve feedback."),
			ID:      &resolution.ID,
		}, err
	}

	if resolution.GithubIssueURL == nil {
		_, err = s.DB.ExecContext(ctx, "UPDATE feedback SET status = $1, resolved_at = NOW() WHERE feedback_id = $2", resolution.Status, fbInt)
		if err != nil {
			return &models.MutationResult{
				Success: false,
				Message: ptrString("Failed to resolve feedback."),
				ID:      &resolution.ID,
			}, err
		}
	} else {
		_, err = s.DB.ExecContext(ctx, "UPDATE feedback SET status = $1, github_issue_url = $2, resolved_at = NOW() WHERE feedback_id = $3", resolution.Status, resolution.GithubIssueURL, fbInt)
		if err != nil {
			return &models.MutationResult{
				Success: false,
				Message: ptrString("Failed to resolve feedback."),
				ID:      &resolution.ID,
			}, err
		}
	}

	return &models.MutationResult{
		Success: true,
		Message: ptrString("Successfully resolved feedback."),
		ID:      &resolution.ID,
	}, err

}
