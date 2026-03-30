package services

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"
	"volunteer-scheduler/models"
)

// ** Handling Strings **
func ptrString(s string) *string {
	return &s
}

// ============================================================================
// Fetching things from the DB
// ============================================================================

// This allows us to get things that shouldn't necessarily be
// exposed through services, as well as reduce duplicate code.

func fetchEmailByVolId(ctx context.Context, DB *sql.DB, volId int) (string, error) {
	var email string
	err := DB.QueryRowContext(ctx,
		"SELECT email FROM volunteers WHERE volunteer_id = $1", volId).Scan(&email)
	if err != nil {
		return "", fmt.Errorf("error fetching volunteer email: %w", err)
	}
	return email, nil
}

func fetchProfile(ctx context.Context, DB *sql.DB, volId int) (*models.VolunteerProfile, error) {
	query := `
		SELECT 
			volunteer_id, 
			first_name, 
			last_name, 
			email, 
			phone, 
			zip_code,
			role
		FROM volunteers 
		WHERE volunteer_id = $1
	`
	var profile models.VolunteerProfile
	var phone, zip sql.NullString

	err := DB.QueryRowContext(ctx, query, volId).Scan(
		&volId,
		&profile.FirstName,
		&profile.LastName,
		&profile.Email,
		&phone,
		&zip,
		&profile.Role)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("volunteer not found")
	}
	if err != nil {
		return nil, fmt.Errorf("error querying volunteer: %w", err)
	}

	if phone.Valid {
		profile.Phone = &phone.String
	} else {
		profile.Phone = nil
	}
	if zip.Valid {
		profile.ZipCode = &zip.String
	} else {
		profile.ZipCode = nil
	}

	return &profile, nil
}

func fetchVolunteerShifts(ctx context.Context, DB *sql.DB, volId int, filter models.ShiftsTimeFilter) ([]*models.VolunteerShift, error) {

	query := `
        SELECT 
			sv.shift_id,
			sv.assigned_at,
			sv.cancelled_at,
			s.shift_start,
			s.shift_end,
			s.max_volunteers,
			jt.name,
			opp.opportunity_is_virtual,
			opp.pre_event_instructions,
            e.event_id,
            e.event_name,
            e.description,
            v.venue_name,
            v.street_address,
            v.city,
            v.state,
            v.zip_code,
			v.timezone,
			vr.region_id
    	FROM volunteer_shifts sv
		LEFT JOIN shifts s ON s.shift_id = sv.shift_id
		LEFT JOIN opportunities opp ON opp.opportunity_id = s.opportunity_id
		LEFT JOIN job_types jt ON jt.job_type_id = opp.job_type_id
		LEFT JOIN events e ON e.event_id = opp.event_id
		LEFT JOIN venues v ON e.venue_id = v.venue_id
		LEFT JOIN venue_regions vr on v.venue_id = vr.venue_id
		WHERE sv.volunteer_id = $1
    `
	switch filter {
	case "UPCOMING":
		query += " AND s.shift_start >= NOW()"
	case "PAST":
		query += " AND s.shift_start < NOW()"
	case "ALL":
		// no filter
	}

	shiftRows, err := DB.QueryContext(ctx, query, volId)
	if err != nil {
		return nil, err
	}
	defer shiftRows.Close()

	shiftsMap := make(map[int]*models.VolunteerShift)

	for shiftRows.Next() {
		var volShift models.VolunteerShift
		var shiftInt, eventInt int
		var cancelledAt, preEventInst, eventDesc sql.NullString
		var venueName, streetAddress, city, state, zip, timezone sql.NullString
		var regionInt sql.NullInt32
		var maxVols sql.NullInt64

		err := shiftRows.Scan(
			&shiftInt,
			&volShift.AssignedAt,
			&cancelledAt,
			&volShift.StartDateTime,
			&volShift.EndDateTime,
			&maxVols,
			&volShift.JobName,
			&volShift.IsVirtual,
			&preEventInst,
			&eventInt,
			&volShift.EventName,
			&eventDesc,
			&venueName,
			&streetAddress,
			&city,
			&state,
			&zip,
			&timezone,
			&regionInt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning shift row: %w", err)
		}

		volShift.ShiftId = strconv.Itoa(shiftInt)
		if cancelledAt.Valid {
			volShift.CancelledAt = &cancelledAt.String
		} else {
			volShift.CancelledAt = nil
		}
		if maxVols.Valid {
			maxVolInt := int(maxVols.Int64)
			volShift.MaxVolunteers = &maxVolInt
		} else {
			volShift.MaxVolunteers = nil
		}
		if preEventInst.Valid {
			volShift.PreEventInstructions = &preEventInst.String
		} else {
			volShift.PreEventInstructions = nil
		}
		if eventDesc.Valid {
			volShift.EventDescription = &eventDesc.String
		} else {
			volShift.EventDescription = nil
		}
		if venueName.Valid {
			volShift.Venue = &models.Venue{
				Name:     &venueName.String,
				Address:  streetAddress.String,
				City:     city.String,
				State:    state.String,
				Timezone: timezone.String,
			}
			if zip.Valid {
				volShift.Venue.ZipCode = &zip.String
			} else {
				volShift.Venue.ZipCode = nil
			}
		} else {
			volShift.Venue = nil
		}

		_, exists := shiftsMap[shiftInt]
		if exists {
			// Duplicate row can be because the venue is in mutiple regions.
			if (shiftsMap[shiftInt].Venue != nil) && (regionInt.Valid) {
				shiftsMap[shiftInt].Venue.Region = append(shiftsMap[shiftInt].Venue.Region, int(regionInt.Int32))
			}
		} else {
			if (volShift.Venue != nil) && (regionInt.Valid) {
				volShift.Venue.Region = append(volShift.Venue.Region, int(regionInt.Int32))
			}
			shiftsMap[shiftInt] = &volShift
		}
	}

	// Convert map to slice
	shifts := make([]*models.VolunteerShift, 0, len(shiftsMap))
	for _, shift := range shiftsMap {
		shifts = append(shifts, shift)
	}

	return shifts, nil
}

func FetchEventServiceTypes(ctx context.Context, DB *sql.DB, eventId int) ([]*string, error) {
	query := `
        SELECT 
			st.name
		FROM event_service_types est
		LEFT JOIN service_types st ON st.service_type_id = est.service_type_id
    	WHERE est.event_id = $1
    `
	rows, err := DB.QueryContext(ctx, query, eventId)
	if err != nil {
		return nil, fmt.Errorf("error querying service types: %w", err)
	}

	serviceTypes := make([]*string, 0)

	for rows.Next() {
		var name string

		err = rows.Scan(&name)
		if err != nil {
			return nil, fmt.Errorf("error scanning service types: %w", err)
		}

		serviceTypes = append(serviceTypes, &name)
	}

	return serviceTypes, nil
}

func FetchEventDates(ctx context.Context, DB *sql.DB, eventId int) ([]*models.EventDate, error) {
	query := `
        SELECT 
			event_date_id,
            start_date_time,
            end_date_time
        FROM event_dates
    	WHERE event_id = $1
    `

	rows, err := DB.QueryContext(ctx, query, eventId)
	if err != nil {
		return nil, fmt.Errorf("error querying dates %w", err)
	}

	dates := make([]*models.EventDate, 0)

	for rows.Next() {
		var date models.EventDate
		date.IanaZone = "UTC"

		err = rows.Scan(
			&date.ID,
			&date.StartDateTime,
			&date.EndDateTime,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning dates: %w", err)
		}

		dates = append(dates, &date)
	}

	// Got 'em
	return dates, nil
}

// ** Handling datetimes **
// We store all dates and times in the DB as UTC with RFC-3339 format.
func DateTimeToUTC(dateTimeStr string, ianaZone string) (*string, error) {
	loc, err := time.LoadLocation(ianaZone)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone %s: %w", ianaZone, err)
	}

	const Layout = "2006-01-02 15:04:05"
	datetime, err := time.ParseInLocation(Layout, dateTimeStr, loc)
	if err != nil {
		return nil, fmt.Errorf("error parsing %s: %w", dateTimeStr, err)
	}

	rfc := datetime.UTC().Format(time.RFC3339)
	return &rfc, nil
}

func UTCToTimeZone(utcTime string, ianaZone string) (*string, error) {

	dateTime, err := time.Parse(time.RFC3339, utcTime)
	if err != nil {
		return nil, fmt.Errorf("error parsing %s: %w", utcTime, err)
	}

	loc, err := time.LoadLocation(ianaZone)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone %s: %w", ianaZone, err)
	}
	strTime := dateTime.In(loc).Format("01-02-2006 15:04 MST")

	return &strTime, nil
}

func UTCToDateTime(utcTime string) (*string, error) {

	dateTime, err := time.Parse(time.RFC3339, utcTime)
	if err != nil {
		return nil, fmt.Errorf("error parsing %s: %w", utcTime, err)
	}

	strTime := dateTime.Format("01-02-2006 15:04 MST")
	return &strTime, nil
}

// These "AddNew" functions are called internally, when creating a new event. The dates must be added
// within a transaction, and the event id must be provided, since, when the date elements were populated,
// the client didn't know the event Id.
func AddNewEventDates(ctx context.Context, dates []*models.NewEventDateInput, eventId int, tx *sql.Tx) error {
	for i := 0; i < len(dates); i++ {
		err := AddNewEventDate(ctx, dates[i], eventId, tx)
		if err != nil {
			return fmt.Errorf("error inserting date with index %d: %w", i, err)
		}
	}

	// No errors.
	return nil
}

func formatStartEnd(ctx context.Context, start string, end string, timezone sql.NullString) (*string, *string, error) {
	var fmtStart, fmtEnd *string
	var err error

	if timezone.Valid {
		fmtStart, err = UTCToTimeZone(start, timezone.String)
		if err == nil {
			fmtEnd, err = UTCToTimeZone(end, timezone.String)
		}
		if err != nil {
			return nil, nil, fmt.Errorf("unable to format shift times: %w", err)
		}
	} else {
		fmtStart, err = UTCToDateTime(start)
		if err == nil {
			fmtEnd, err = UTCToDateTime(end)
		}
		if err != nil {
			return nil, nil, fmt.Errorf("unable to format shift times: %w", err)
		}
	}

	return fmtStart, fmtEnd, nil
}

// ** Converting event types **
func GetEventType(isVirtual bool, hasVenue bool) models.EventType {
	// Determine if event is virtual, in person, or both.

	if isVirtual && hasVenue {
		// Either.
		return "HYBRID"
	}
	if isVirtual {
		// Virtual only event. No venue.
		return "VIRTUAL"
	}
	// Not virtual.
	return "IN_PERSON"
}

func AddNewEventDate(ctx context.Context, dates *models.NewEventDateInput, eventId int, tx *sql.Tx) error {
	var startUTC, endUTC *string
	startUTC, err := DateTimeToUTC(dates.StartDateTime, dates.IanaZone)
	if err == nil {
		endUTC, err = DateTimeToUTC(dates.EndDateTime, dates.IanaZone)
	}
	if err != nil {
		return err
	}

	insert := `
		INSERT INTO event_dates (event_id, start_date_time, end_date_time)
		VALUES ($1, $2, $3)
		RETURNING event_date_id
	`

	var eventDateInt int
	err = tx.QueryRowContext(ctx, insert, eventId, startUTC, endUTC).Scan(&eventDateInt)
	if err != nil {
		return fmt.Errorf("error inserting datetimes: %w", err)
	}

	// No errors.
	return nil
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
		ON CONFLICT (volunteer_id, shift_id) DO NOTHING
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
	`
	volRows, err := DB.QueryContext(ctx, volQuery, shiftId)
	if err != nil {
		return nil, fmt.Errorf("Error querying assigned volunteers: %w", err)
	}
	defer volRows.Close()

	var assignedVols []*models.Volunteer
	for volRows.Next() {
		var vol models.Volunteer
		var volID int
		err := volRows.Scan(&volID, &vol.FirstName, &vol.LastName)
		if err != nil {
			return nil, fmt.Errorf("Error scanning assigned volunteers: %w", err)
		}

		vol.ID = fmt.Sprintf("%d", volID)
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

	// Convert dates, times to UTC.
	startUTC, err := DateTimeToUTC(shift.StartDateTime, shift.IanaZone)
	if err == nil {
		endUTC, err = DateTimeToUTC(shift.EndDateTime, shift.IanaZone)
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

func AddNoteToFeedback(ctx context.Context, DB *sql.DB, feedbackId int, adminId int, note string) error {

	insert := `
		INSERT INTO feedback_notes (
			feedback_id,
			volunteer_id, 
			note,
			createdAt)
		VALUES ($1, $2, $3, NOW())
		RETURNING note_id
	`

	var noteInt int
	err := DB.QueryRowContext(ctx, insert, feedbackId, adminId, note).Scan(&noteInt)
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
