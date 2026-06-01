package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"volunteer-scheduler/models"
)

// ** Handling Strings **
func ptrString(s string) *string {
	return &s
}

// ============================================================================
// Fetching things from the DB
// ============================================================================
// These allow us to get things that shouldn't necessarily be
// exposed through services, as well as to reduce duplicate code.

func fetchEmailByVolId(ctx context.Context, DB *sql.DB, volId int) (string, error) {
	var email string
	err := DB.QueryRowContext(ctx,
		"SELECT email FROM volunteers WHERE volunteer_id = $1", volId).Scan(&email)
	if err != nil {
		return "", fmt.Errorf("error fetching volunteer email: %w", err)
	}
	return email, nil
}

// ** Converting event types **
func GetEventType(isVirtual bool, hasVenue bool) models.EventType {
	// Determine if event is virtual, in person, or both.

	if isVirtual && hasVenue {
		// Either.
		return "HYBRID"
	}
	if hasVenue {
		// Not virtual.
		return "IN_PERSON"
	}
	// Virtual only event. No venue.
	return "VIRTUAL"
}

// ** Handling shift assignments **

func assignVolToShift(ctx context.Context, DB *sql.DB, mailer *Mailer, shiftId string, volId int) (*models.MutationResult, error) {
	shiftInt, err := strconv.Atoi(shiftId)
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Invalid shiftId."),
			ID:      &shiftId,
		}, err
	}

	query := `
		SELECT
			s.shift_id,
			COUNT(vs.volunteer_id) FILTER (WHERE vs.cancelled_at IS NULL) as curr_vols,
			s.max_volunteers
		FROM shifts s
		LEFT JOIN volunteer_shifts vs 
			ON s.shift_id = vs.shift_id
		WHERE s.shift_id = $1
		GROUP BY s.shift_id, s.max_volunteers
	`

	var sId, currVols, maxVols int

	err = DB.QueryRowContext(ctx, query, shiftInt).Scan(&sId, &currVols, &maxVols)
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to assign volunteer to shift: unable to query shift assignments."),
			ID:      nil,
		}, err
	}
	if currVols >= maxVols {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to assign volunteer to shift: shift is full."),
			ID:      nil,
		}, nil
	}

	insert := `
		INSERT INTO volunteer_shifts (volunteer_id, shift_id, assigned_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (volunteer_id, shift_id) DO UPDATE
			SET cancelled_at = NULL, assigned_at = NOW()
	`
	_, err = DB.ExecContext(ctx, insert, volId, shiftInt)
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to assign volunteer to shift."),
			ID:      nil,
		}, err
	}

	err = sendAssignmentConfirmation(ctx, DB, mailer, shiftInt, volId)
	if err != nil {
		volStr := strconv.Itoa(volId)
		return &models.MutationResult{
			Success: true,
			Message: ptrString("Successfully assigned shift to vol. Unable to send confirmation email to vol."),
			ID:      &volStr,
		}, err
	}

	return &models.MutationResult{
		Success: true,
		Message: ptrString("Volunteer successfully assigned."),
		ID:      &shiftId,
	}, nil
}

// CancelShiftAssignment
// Cancels a volunteer's shift assignment. A soft delete for the sake of the volunteer's history.
func cancelShiftAssignment(ctx context.Context, DB *sql.DB, mailer *Mailer, shiftId string, volId int) (*models.MutationResult, error) {
	shiftInt, err := strconv.Atoi(shiftId)
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("ShiftId is not valid."),
			ID:      nil,
		}, err
	}

	update := `
		UPDATE volunteer_shifts
		SET cancelled_at = NOW()
		WHERE volunteer_id = $1 AND shift_id = $2
	`
	_, err = DB.ExecContext(ctx, update, volId, shiftInt)
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to cancel shift assignment."),
			ID:      nil,
		}, err
	}

	err = sendCancellationConfirmation(ctx, DB, mailer, shiftInt, volId)
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Cancelled shift assignment; failed to send email to volunteer."),
			ID:      nil,
		}, err
	}

	return &models.MutationResult{
		Success: true,
		Message: ptrString("Successfully deleted shift assignment."),
		ID:      nil,
	}, nil
}

// Currently this is internal, but we might add an interface for this if the
// users request it.
func FetchAssignedVolunteersForShift(ctx context.Context, shiftId int, DB *sql.DB) ([]*models.Volunteer, error) {

	volQuery := `
		SELECT 
			v.volunteer_id, 
			v.first_name, 
			v.last_name
		FROM volunteers v
		JOIN volunteer_shifts vs ON v.volunteer_id = vs.volunteer_id
		WHERE vs.shift_id = $1 AND vs.cancelled_at IS NULL
		ORDER BY v.last_name, v.first_name
	`
	volRows, err := DB.QueryContext(ctx, volQuery, shiftId)
	if err != nil {
		return nil, fmt.Errorf("Error querying assigned volunteers: %w", err)
	}
	defer volRows.Close()

	var assignedVols []*models.Volunteer
	for volRows.Next() {
		var vol models.Volunteer
		var volId int
		err := volRows.Scan(&volId, &vol.FirstName, &vol.LastName)
		if err != nil {
			return nil, fmt.Errorf("Error scanning assigned volunteers: %w", err)
		}

		vol.ID = fmt.Sprintf("%d", volId)
		assignedVols = append(assignedVols, &vol)
	}

	return assignedVols, nil
}

// ** Adding shifts **
// These AddNew functions are called internally when we are adding a new opportunity, which *includes*
// shifts. These take the oppId and a transaction as separate parameters since the client won't know
// the oppId until a successful return of the whole transaction.
func addNewOpportunityShifts(ctx context.Context, shifts []*models.NewShiftInput, oppId int, tx *sql.Tx) error {

	// A new opportunity requires at least one shift.
	if len(shifts) == 0 {
		return fmt.Errorf("no shifts found; a new opportunity requires at least one shift")
	}
	for _, shift := range shifts {
		err := addNewOpportunityShift(ctx, shift, oppId, tx)
		if err != nil {
			err = fmt.Errorf("error adding shift: %w", err)
			return err
		}
	}
	// No errors.
	return nil
}

func addNewOpportunityShift(ctx context.Context, shift *models.NewShiftInput, oppId int, tx *sql.Tx) error {
	var shiftId int
	var startUTC, endUTC *string
	var staffId, maxVols interface{}
	var timezone string

	query := `
		SELECT
			e.timezone 
		FROM events e
		JOIN opportunities opp ON opp.event_id = e.event_id
		WHERE opp.opportunity_id = $1
		`
	err := tx.QueryRowContext(ctx, query, oppId).Scan(&timezone)
	if err != nil {
		return friendlyDBError(err)
	}

	// Convert dates, times to UTC.
	startUTC, err = DateTimeToUTC(shift.StartDateTime, timezone)
	if err == nil {
		endUTC, err = DateTimeToUTC(shift.EndDateTime, timezone)
	}
	if err != nil {
		return err
	}

	// Handle optional values.
	if shift.StaffContactId != nil {
		staffId = *shift.StaffContactId
	}
	if shift.MaxVolunteers != nil {
		maxVols = *shift.MaxVolunteers
	}

	insert := `
		INSERT INTO shifts (
			opportunity_id, 
			shift_start, 
			shift_end, 
			staff_contact_id, 
			max_volunteers)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING shift_id
	`
	err = tx.QueryRowContext(ctx, insert, oppId, startUTC, endUTC, staffId, maxVols).Scan(&shiftId)
	if err != nil {
		return fmt.Errorf("error adding shift to new opportunity: %w", err)

	}
	// No errors.
	return nil
}

func addNoteToFeedback(ctx context.Context, DB *sql.DB, feedbackId int, adminId int, note string, noteType string) error {

	insert := `
		INSERT INTO feedback_notes (
			feedback_id,
			volunteer_id,
			note,
			note_type,
			created_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING note_id
	`

	var noteInt int
	err := DB.QueryRowContext(ctx, insert, feedbackId, adminId, note, noteType).Scan(&noteInt)
	if err != nil {
		return fmt.Errorf("error adding feedback note to DB: %w", err)
	}

	_, err = DB.ExecContext(ctx, "UPDATE feedback SET last_updated_at = NOW() WHERE feedback_id = $1", feedbackId)
	if err != nil {
		return fmt.Errorf("error updating feedback table: %w", err)
	}

	// All good.
	return nil
}

// When sending emails about cancelled shifts, there is a "standard" set
// of data we need for volunteers or leads (email address, name, shifts)
// to avoid sending multiple emails to each of them.
type emailInfo struct {
	email     string
	firstName string
	shifts    []int
}

func makeEmailMapForShifts(ctx context.Context, DB *sql.DB, query string, args []any, sMap map[int]*ShiftSummary) (*map[int]*emailInfo, map[int]*ShiftSummary, error) {

	// The key to the shifts map is the shift id.
	// The key to the email map will be the id of the email recipient.
	eMap := map[int]*emailInfo{}

	rows, err := DB.QueryContext(ctx, query, args...)
	if err != nil {
		log.Printf("DB err: %v", err)
		return nil, sMap, friendlyDBError(err)
	}
	defer rows.Close()

	for rows.Next() {
		var eId, sId int
		var email, fname string
		var ss, se string

		err := rows.Scan(
			&eId,
			&sId,
			&email,
			&fname,
			&ss,
			&se,
		)
		if err != nil {
			return nil, sMap, fmt.Errorf("unable to scan rows in delete event: %w", err)
		}

		// If we haven't seen this shift before, add it to the map.
		_, shiftExists := sMap[sId]
		if !shiftExists {
			var summ ShiftSummary
			summ.Start = ss
			summ.End = se

			sMap[sId] = &summ
		}

		_, emailExists := eMap[eId]
		if emailExists {
			// Recipient is already in the map. Just add the shift.
			eMap[eId].shifts = append(eMap[eId].shifts, sId)
		} else {
			// Add the recipient to the map.
			var e emailInfo

			e.email = email
			e.firstName = fname
			e.shifts = append(e.shifts, sId)

			eMap[eId] = &e
		}
	}

	return &eMap, sMap, nil
}

func formatShiftTimes(dbShiftsMap map[int]*ShiftSummary, timezone string) map[int]*ShiftSummary {

	sMap := map[int]*ShiftSummary{}

	for id, dbSumm := range dbShiftsMap {
		var summ ShiftSummary
		var start, end *string
		start, err := UTCToTimeZone(dbSumm.Start, timezone)
		if err == nil {
			end, err = UTCToTimeZone(dbSumm.End, timezone)
		}
		if err == nil {
			summ.Start = *start
			summ.End = *end
		} else {
			// It's more important to send the emails than to format them exactly right.
			// We did the best we could. Show the strings in the log; the shift will be gone.
			log.Printf("error formatting shift times (%v and %v): %v", dbSumm.Start, dbSumm.End, err)
			summ.Start = dbSumm.Start
			summ.End = dbSumm.End
		}
		sMap[id] = &summ
	}
	return sMap
}

func sendDeleteEventEmailsForShifts(ctx context.Context, mailer *Mailer, volMap *map[int]*emailInfo, leadMap *map[int]*emailInfo, sMap map[int]*ShiftSummary, evName string) {
	var err error
	unsent := []string{}

	for _, emailInfo := range *volMap {
		// Get all of the shift start and end times for this one email.
		shiftSummaries := []ShiftSummary{}
		for _, shiftKey := range emailInfo.shifts {
			shiftSummaries = append(shiftSummaries, *sMap[shiftKey])
		}
		err = sendEventCancelledToVolunteer(ctx, mailer, emailInfo.firstName, evName, shiftSummaries, emailInfo.email)
		if err != nil {
			// Not being able to send an email is not fatal. Just "note"
			// the email, and try to notify the rest of the list.
			unsent = append(unsent, emailInfo.email)
			continue
		}
	}
	for _, emailInfo := range *leadMap {
		// Get all of the shift start and end times for this email.
		shiftSummaries := []ShiftSummary{}
		for _, shiftKey := range emailInfo.shifts {
			shiftSummaries = append(shiftSummaries, *sMap[shiftKey])
		}
		err = sendEventCancelledToStaff(ctx, mailer, emailInfo.firstName, evName, shiftSummaries, emailInfo.email)
		if err != nil {
			unsent = append(unsent, emailInfo.email)
			continue
		}
	}

	if len(unsent) > 0 {
		log.Println("Unable to send the event cancelled message to the following emails:")
		for _, emailStr := range unsent {
			log.Println(emailStr)
		}
	}
}

func getQueriesForSingleEvent() (string, string) {
	volQuery := `
		SELECT
			v.volunteer_id,
			s.shift_id,
			v.email,
			v.first_name,
    		s.shift_start,
    		s.shift_end
		FROM volunteer_shifts vs
		JOIN volunteers v  ON v.volunteer_id = vs.volunteer_id
		JOIN shifts s ON s.shift_id = vs.shift_id
		JOIN opportunities o ON o.opportunity_id = s.opportunity_id
		JOIN events e ON e.event_id = o.event_id
		WHERE e.event_id = $1 AND vs.cancelled_at IS NULL
	`

	leadQuery := `
		SELECT
			st.staff_id,
			s.shift_id,
			st.email,
			st.first_name,
			s.shift_start,
			s.shift_end
		FROM events e
		JOIN opportunities opp ON opp.event_id = e.event_id
		JOIN shifts s ON s.opportunity_id = opp.opportunity_id
		JOIN staff st ON st.staff_id = s.staff_contact_id
		WHERE e.event_id = $1
	`

	return volQuery, leadQuery
}

func getQueriesForRecurringEvent() (string, string) {
	volQuery := `
		SELECT
			v.volunteer_id,
			s.shift_id,
			v.email,
			v.first_name,
    		s.shift_start,
    		s.shift_end
		FROM volunteer_shifts vs
		JOIN volunteers v ON v.volunteer_id  = vs.volunteer_id
		JOIN shifts s ON s.shift_id = vs.shift_id
		JOIN opportunities o ON o.opportunity_id = s.opportunity_id
		JOIN events e ON e.event_id = o.event_id
		WHERE e.recurrence_group_id = $1::uuid AND e.recurrence_order >= $2
		   AND vs.cancelled_at IS NULL
		ORDER BY v.volunteer_id, s.shift_start
	`
	leadQuery := `
		SELECT
			st.staff_id,
			s.shift_id,
			st.email,
			st.first_name,
			s.shift_start,
			s.shift_end
		FROM events e
		JOIN opportunities opp ON opp.event_id = e.event_id
		JOIN shifts s ON s.opportunity_id = opp.opportunity_id
		JOIN staff st ON st.staff_id = s.staff_contact_id
		WHERE e.recurrence_group_id = $1::uuid AND e.recurrence_order >= $2
	`
	return volQuery, leadQuery
}
