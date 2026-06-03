package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"
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

func (s *FeedbackService) FetchOwnFeedback(ctx context.Context, volId int) ([]*models.FeedbackView, error) {
	query := `
        SELECT
            f.feedback_id,
			f.feedback_type,
			f.status,
			f.subject,
			f.app_page_name,
			f.text,
			fn.note_id,
			fn.note,
			fn.note_type,
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

	feedbackMap := make(map[int]*models.FeedbackView)

	// orderedFB preserves the ORDER BY from the SQL query so the caller
	// can reassemble the slice in the correct order after map operations.
	orderedFB := make([]int, 0)

	for rows.Next() {
		var fb models.FeedbackView
		var fbInt int
		var fbType, fbStatus string
		var noteId sql.NullInt64
		var note, noteType, noteCreatedAt sql.NullString

		err := rows.Scan(
			&fbInt,
			&fbType,
			&fbStatus,
			&fb.Subject,
			&fb.AppPageName,
			&fb.Text,
			&noteId,
			&note,
			&noteType,
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
			fbPtr.Notes = append(fbPtr.Notes, &models.FeedbackNoteView{
				ID:        strconv.FormatInt(noteId.Int64, 10),
				Note:      note.String,
				NoteType:  models.FeedbackNoteType(noteType.String),
				CreatedAt: noteCreatedAt.String,
			})
		}

		fbPtr.Attachments, err = s.fetchMetaAttachments(ctx, fbInt)
		if err != nil {
			return nil, fmt.Errorf("error fetching attachments for feedback with id %d: %w", fbInt, err)
		}
	}

	// Build the feedback slice in the order the DB returned the feedback.
	feedback := make([]*models.FeedbackView, 0, len(feedbackMap))
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
			f.created_at,
			f.last_updated_at,
			f.resolved_at
			from feedback f
			LEFT JOIN volunteers fc ON fc.volunteer_id = f.volunteer_id
			LEFT JOIN feedback_notes fn ON fn.feedback_id = f.feedback_id
			LEFT JOIN volunteers nc ON nc.volunteer_id = fn.volunteer_id
		WHERE 1=1
    `

	args := []any{}
	argcount := 1

	if filter != nil {
		if filter.Status != nil {
			dbstatus := strings.ToLower(string(*filter.Status))
			query += " AND f.status = $" + strconv.Itoa(argcount)
			args = append(args, dbstatus)
			argcount++
		}

		if filter.Type != nil {
			dbtype := strings.ToLower(string(*filter.Type))
			query += " AND f.feedback_type = $" + strconv.Itoa(argcount)
			args = append(args, dbtype)
			argcount++
		}
	}
	query += " ORDER BY f.created_at, fn.created_at"

	rows, err := s.DB.QueryContext(ctx, query, args...)
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
		var lastUpdateAt, resolvedAt sql.NullString

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
			if lastUpdateAt.Valid {
				fb.LastUpdatedAt = &lastUpdateAt.String
			} else {
				fb.LastUpdatedAt = nil
			}
			if resolvedAt.Valid {
				fb.ResolvedAt = &resolvedAt.String
			} else {
				fb.ResolvedAt = nil
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

		fbPtr.Attachments, err = s.fetchMetaAttachments(ctx, fbInt)
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

func (s *FeedbackService) FetchFeedbackDetail(ctx context.Context, feedbackId string) (*models.Feedback, error) {

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
	var lastUpdateAt, resolvedAt sql.NullString

	err = row.Scan(
		&fcFname,
		&fcLname,
		&fbType,
		&fbStatus,
		&feedback.Subject,
		&feedback.AppPageName,
		&feedback.Text,
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
	if lastUpdateAt.Valid {
		feedback.LastUpdatedAt = &lastUpdateAt.String
	} else {
		feedback.LastUpdatedAt = nil
	}
	if resolvedAt.Valid {
		feedback.ResolvedAt = &resolvedAt.String
	} else {
		feedback.ResolvedAt = nil
	}
	feedback.Notes = make([]*models.FeedbackNote, 0, 5) // If there are more than 5 notes (unlikely), the slice will increase by 5 slots.

	// Get all of the notes created for this feedback, along
	// with the creator of each note and the time it was created.
	query = `
        SELECT
			nc.first_name,
			nc.last_name,
			fn.note_type,
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
		var ncFname, ncLname, noteType string
		err := rows.Scan(
			&ncFname,
			&ncLname,
			&noteType,
			&note.Note,
			&note.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning feedback: %w", err)
		}

		note.Creator = ncFname + " " + ncLname
		note.NoteType = models.FeedbackNoteType(noteType)

		feedback.Notes = append(feedback.Notes, &note)

	}
	feedback.Attachments, err = s.fetchMetaAttachments(ctx, fbInt)
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
		return nil, friendlyDBError(err)
	}

	fbStr := strconv.Itoa(fbInt)
	return &models.MutationResult{
		Success: true,
		Message: ptrString("Successfully created feedback."),
		ID:      &fbStr,
	}, err
}

// We use this string a lot.
const insertNote = `
		INSERT INTO feedback_notes (
			feedback_id,
			volunteer_id,
			note,
			note_type,
			created_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING note_id
	`

// A user has created feedback. An admin wants to ask a question or send information.
func (s *FeedbackService) EmailFeedbackSubmitter(ctx context.Context, adminId int, input models.FeedbackEmailInput) (*models.MutationResult, error) {

	// Get information about the feedback's creator.
	query := `
		SELECT 
			fc.email, 
			f.subject
		FROM feedback f
		LEFT JOIN volunteers fc ON fc.volunteer_id = f.volunteer_id
		WHERE f.feedback_id = $1
	`
	var email, subject string
	err := s.DB.QueryRowContext(ctx, query, input.FeedbackID).Scan(&email, &subject)
	if err != nil {
		return nil, friendlyDBError(err)
	}

	subject = "re: " + subject
	err = s.Mailer.SendEmail(ctx, email, subject, "", input.EmailText)
	if err != nil {
		return nil, fmt.Errorf("failed to send email to volunteer: %w", err)
	}

	// If a reply is needed....
	var noteInt int
	if input.RequireReply {
		// Call this a "question" and change the feedback's status.
		err = s.DB.QueryRowContext(ctx, insertNote, input.FeedbackID, adminId, input.EmailText, string(models.FeedbackNoteTypeQuestion)).Scan(&noteInt)
		if err != nil {
			return nil, friendlyDBError(err)
		}
		_, err = s.DB.ExecContext(ctx, "UPDATE feedback SET status = $1, last_updated_at = NOW() WHERE feedback_id = $2", string(models.FeedbackStatusQuestion), input.FeedbackID)
		if err != nil {
			return nil, friendlyDBError(err)
		}
	} else {
		// Just add the email to the thread. No change in status.
		err = s.DB.QueryRowContext(ctx, insertNote, input.FeedbackID, adminId, input.EmailText, string(models.FeedbackNoteTypeEmailToVoluneer)).Scan(&noteInt)
		if err != nil {
			return nil, friendlyDBError(err)
		}
		_, err = s.DB.ExecContext(ctx, "UPDATE feedback SET last_updated_at = NOW() WHERE feedback_id = $1", input.FeedbackID)
		if err != nil {
			return nil, friendlyDBError(err)
		}

	}

	noteId := strconv.Itoa(noteInt)
	return &models.MutationResult{
		Success: true,
		Message: ptrString("Successfully sent email to feedback's creator."),
		ID:      &noteId,
	}, err
}

func (s *FeedbackService) AddVolunteerFeedbackNote(ctx context.Context, volId int, note models.VolunteerFeedbackNoteInput) (*models.MutationResult, error) {

	// Get information about feedback's submitter, and the current feedback status.
	query := `
		SELECT
			volunteer_id,
			status
		FROM feedback
		WHERE feedback_id = $1
		`

	var creatorId int
	var status string
	err := s.DB.QueryRowContext(ctx, query, note.FeedbackId).Scan(&creatorId, &status)
	if err != nil {
		return nil, friendlyDBError(err)
	}
	if volId != creatorId {
		return nil, fmt.Errorf("volunteers may only add notes to their own feedback.")
	}

	var noteInt int
	err = s.DB.QueryRowContext(ctx, insertNote, note.FeedbackId, volId, note.Note, string(models.FeedbackNoteTypeVolunteerNote)).Scan(&noteInt)
	if err != nil {
		return nil, friendlyDBError(err)
	}

	if status == string(models.FeedbackStatusQuestion) {
		_, err = s.DB.ExecContext(ctx, "UPDATE feedback SET status = $1, last_updated_at = NOW() WHERE feedback_id = $2", string(models.FeedbackStatusOpen), note.FeedbackId)
		if err != nil {
			return nil, friendlyDBError(err)
		}
		// Notify the admin who asked the question that the volunteer has replied.
		s.notifyAdminOfVolunteerReply(ctx, note.FeedbackId)
	} else {
		_, err = s.DB.ExecContext(ctx, "UPDATE feedback SET last_updated_at = NOW() WHERE feedback_id = $1", note.FeedbackId)
		if err != nil {
			return nil, friendlyDBError(err)
		}
	}

	noteId := strconv.Itoa(noteInt)
	return &models.MutationResult{
		Success: true,
		Message: ptrString("Successfully added note to feedback."),
		ID:      &noteId,
	}, nil
}

func (s *FeedbackService) AddFeedbackNote(ctx context.Context, adminId int, note models.FeedbackNoteInput) (*models.MutationResult, error) {

	var noteInt int
	err := s.DB.QueryRowContext(ctx, insertNote, note.FeedbackID, adminId, note.Note, string(models.FeedbackNoteTypeAdminNote)).Scan(&noteInt)
	if err != nil {
		return nil, friendlyDBError(err)
	}

	_, err = s.DB.ExecContext(ctx, "UPDATE feedback SET last_updated_at = NOW() WHERE feedback_id = $1", note.FeedbackID)
	if err != nil {
		return nil, friendlyDBError(err)
	}

	noteId := strconv.Itoa(noteInt)
	return &models.MutationResult{
		Success: true,
		Message: ptrString("Successfully added note to feedback."),
		ID:      &noteId,
	}, nil
}

func (s *FeedbackService) UpdateFeedbackStatus(ctx context.Context, adminId int, su models.FeedbackStatusUpdateInput) (*models.MutationResult, error) {

	// Add the note. This gives us the context of the status change, so is required.
	var noteInt int
	err := s.DB.QueryRowContext(ctx, insertNote, su.FeedbackID, adminId, su.Note, string(models.FeedbackNoteTypeAdminNote)).Scan(&noteInt)
	if err != nil {
		return nil, friendlyDBError(err)
	}

	// Update the status and the last_updated_at fields in the feedback table; if this is a resolution, also update the resolved_at field.
	if su.Status == models.FeedbackStatusImplemented || su.Status == models.FeedbackStatusRejected {
		_, err = s.DB.ExecContext(ctx, "UPDATE feedback SET status = $1, last_updated_at = NOW(), resolved_at = NOW() WHERE feedback_id = $2", su.Status, su.FeedbackID)
	} else {
		_, err = s.DB.ExecContext(ctx, "UPDATE feedback SET status = $1, last_updated_at = NOW() WHERE feedback_id = $2", su.Status, su.FeedbackID)
	}
	if err != nil {
		return nil, friendlyDBError(err)
	}

	// If resolving, notify the submitter by email.
	if su.Status == models.FeedbackStatusImplemented || su.Status == models.FeedbackStatusRejected {
		s.notifyVolunteerOfResolution(ctx, su.FeedbackID, su.Status)
	}

	strId := strconv.Itoa(su.FeedbackID)
	return &models.MutationResult{
		Success: true,
		Message: ptrString("Successfully updated feedback."),
		ID:      &strId,
	}, nil

}

// notifyAdminOfVolunteerReply emails the admin who most recently asked a question
// on the feedback to let them know the volunteer has replied. Logs on failure
// rather than returning an error — the DB operation has already succeeded.
func (s *FeedbackService) notifyAdminOfVolunteerReply(ctx context.Context, feedbackID int) {
	var email, subject string
	err := s.DB.QueryRowContext(ctx, `
		SELECT v.email, f.subject
		FROM feedback_notes fn
		JOIN volunteers v ON v.volunteer_id = fn.volunteer_id
		JOIN feedback f ON f.feedback_id = fn.feedback_id
		WHERE fn.feedback_id = $1 AND fn.note_type = 'QUESTION'
		ORDER BY fn.created_at DESC
		LIMIT 1
	`, feedbackID).Scan(&email, &subject)
	if err != nil {
		log.Printf("notifyAdminOfVolunteerReply: could not find question author for feedback %d: %v", feedbackID, err)
		return
	}
	body := fmt.Sprintf("A volunteer has replied to your question on feedback \"%s\".\n\nLog in to review their response.", subject)
	if err := s.Mailer.SendEmail(ctx, email, "re: "+subject, "", body); err != nil {
		log.Printf("notifyAdminOfVolunteerReply: failed to send email for feedback %d: %v", feedbackID, err)
	}
}

// notifyVolunteerOfResolution emails the volunteer who submitted the feedback
// when it is marked as resolved or rejected. Logs on failure rather than
// returning an error — the DB operation has already succeeded.
func (s *FeedbackService) notifyVolunteerOfResolution(ctx context.Context, feedbackID int, status models.FeedbackStatus) {
	var email, firstName, subject string
	err := s.DB.QueryRowContext(ctx, `
		SELECT v.email, v.first_name, f.subject
		FROM feedback f
		JOIN volunteers v ON v.volunteer_id = f.volunteer_id
		WHERE f.feedback_id = $1
	`, feedbackID).Scan(&email, &firstName, &subject)
	if err != nil {
		log.Printf("notifyVolunteerOfResolution: could not find submitter for feedback %d: %v", feedbackID, err)
		return
	}
	var body string
	if status == models.FeedbackStatusImplemented {
		body = fmt.Sprintf("Hi %s,\n\nYour feedback \"%s\" has been reviewed and implemented. Thank you for taking the time to share it with us.", firstName, subject)
	} else {
		body = fmt.Sprintf("Hi %s,\n\nYour feedback \"%s\" has been reviewed. We appreciate you sharing it, though we're unable to take action on it at this time.", firstName, subject)
	}
	if err := s.Mailer.SendEmail(ctx, email, "Your feedback has been reviewed: "+subject, "", body); err != nil {
		log.Printf("notifyVolunteerOfResolution: failed to send email for feedback %d: %v", feedbackID, err)
	}
}
