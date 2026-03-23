package services

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"
	"volunteer-scheduler/models"

	"github.com/lib/pq"
)

// ** Handling Strings **
func ptrString(s string) *string {
	return &s
}

// ** Fetching things from the DB **
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
			opp.job,
			opp.other_job_description,
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
		var cancelledAt, otherJobDesc, preEventInst, eventDesc sql.NullString
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
			&volShift.Job,
			&otherJobDesc,
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
		if otherJobDesc.Valid {
			volShift.OtherJobDescription = &otherJobDesc.String
		} else {
			volShift.OtherJobDescription = nil
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
	const Layout = "01-02-2006 15:04 MST"

	loc, err := time.LoadLocation(ianaZone)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone %s: %w", ianaZone, err)
	}

	dateTime, err := time.Parse(time.RFC3339, utcTime)
	if err != nil {
		return nil, fmt.Errorf("error parsing %s: %w", utcTime, err)
	}

	strTime := dateTime.In(loc).Format(Layout)
	return &strTime, nil
}

func UTCToDateTime(utcTime string) (*string, error) {
	const Layout = "01-02-2006 15:04 MST"

	dateTime, err := time.Parse(time.RFC3339, utcTime)
	if err != nil {
		return nil, fmt.Errorf("error parsing %s: %w", utcTime, err)
	}

	strTime := dateTime.Format(Layout)
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

	email, err := fetchEmailByVolId(ctx, DB, volId)
	if err != nil {
		volStr := strconv.Itoa(volId)
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Unable to fetch email for volunteer."),
			ID:      &volStr,
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

	err = mailer.SendEmail(ctx, email, "Shift Assignment", "", "")
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

	email, err := fetchEmailByVolId(ctx, DB, volId)
	if err != nil {
		volStr := strconv.Itoa(volId)
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Unable to fetch email for volunteer."),
			ID:      &volStr,
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
			Message: ptrString("Failed to delete shift assignment."),
			ID:      nil,
		}, err
	}

	// NOTE: in addition to mailing the volunteer, cancelling a shift should send
	// an eamil to the staff lead or to the volunteer coordinators or to SOMEONE.

	err = mailer.SendEmail(ctx, email, "Shift Assignment Cancelled", "We are sorry you are not able to volunteer for this shift...", "")
	if err != nil {
		volStr := strconv.Itoa(volId)
		return &models.MutationResult{
			Success: true,
			Message: ptrString("Successfully canceled assignment. Unable to send email to volunteer"),
			ID:      &volStr,
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
func AddNewOpportunityShifts(ctx context.Context, shifts []*models.NewShiftInput, oppId int, tx *sql.Tx) error {

	for i := 0; i < len(shifts); i++ {
		err := AddNewOpportunityShift(ctx, shifts[i], oppId, tx)
		if err != nil {
			err = fmt.Errorf("error adding shift with index %v: %w", i, err)
			return err
		}
	}
	// No errors.
	return nil
}

func AddNewOpportunityShift(ctx context.Context, shift *models.NewShiftInput, oppId int, tx *sql.Tx) error {
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

// ** Filtering Events **
// Figuring out all of the event filtering stuff is messy, but worth it. The user can
// get the events he wants to see. We use a 2-pass strategy. This function handles the
// first pass.
func FetchFilteredPassOne(ctx context.Context, filter *models.EventFilterInput, db *sql.DB) (map[int]*models.Event, error) {
	query := `
        SELECT DISTINCT
            e.event_id,
            e.event_name,
            e.description,
            e.event_is_virtual,
            e.venue_id,
            v.venue_name,
            v.street_address,
            v.city,
            v.state,
            v.zip_code,
			v.timezone,
			vr.region_id,
			earliest.first_date
        FROM events e
        LEFT JOIN venues v ON e.venue_id = v.venue_id
		LEFT JOIN venue_regions vr ON v.venue_id = vr.venue_id
        LEFT JOIN opportunities opp ON e.event_id = opp.event_id
	    LEFT JOIN shifts s_filter ON opp.opportunity_id = s_filter.opportunity_id 
		LEFT JOIN (
			SELECT event_id, MIN(start_date_time) as first_date
			FROM event_dates
			GROUP BY event_id
		) earliest ON e.event_id = earliest.event_id       
		WHERE 1=1
    `

	args := []interface{}{}
	argCount := 1

	if filter != nil {
		// Filter by regions.
		if len(filter.Regions) > 0 {
			placeholders := []string{}
			for _, region := range filter.Regions {
				placeholders = append(placeholders, fmt.Sprintf("$%d", argCount))
				args = append(args, region)
				argCount++
			}
			query += fmt.Sprintf(" AND vr.region_id IN (%s)", strings.Join(placeholders, ","))
		}

		// Filter by event type.
		if filter.EventType != nil {
			switch *filter.EventType {
			case "VIRTUAL":
				query += " AND e.event_is_virtual = true AND e.venue_id IS NULL"
			case "IN_PERSON":
				query += " AND e.event_is_virtual = false"
			case "HYBRID":
				query += " AND e.event_is_virtual = true AND e.venue_id IS NOT NULL"
			}
		}

		// Filter by Jobs.
		if len(filter.Jobs) > 0 {
			placeholders := []string{}
			for _, job := range filter.Jobs {
				placeholders = append(placeholders, fmt.Sprintf("$%d", argCount))
				dbJob := strings.ToLower(string(job))
				args = append(args, dbJob)
				argCount++
			}
			query += fmt.Sprintf(" AND opp.job IN (%s)", strings.Join(placeholders, ","))
		}

		// Filter by Dates.
		if filter.ShiftStartDate != nil {
			startUTC, err := DateTimeToUTC(*filter.ShiftStartDate, *filter.IanaZone)
			if err != nil {
				return nil, fmt.Errorf("error converting date to UTC using %s %s: %w", *filter.ShiftStartDate, *filter.IanaZone, err)
			}

			query += fmt.Sprintf(" AND s_filter.shift_start >= $%d", argCount)
			args = append(args, startUTC)
			argCount++
		}
		if filter.ShiftEndDate != nil {
			endUTC, err := DateTimeToUTC(*filter.ShiftEndDate, *filter.IanaZone)
			if err != nil {
				return nil, fmt.Errorf("error converting date to UTC using %s %s: %w", *filter.ShiftEndDate, *filter.IanaZone, err)
			}
			query += fmt.Sprintf(" AND s_filter.shift_start <= $%d", argCount)
			args = append(args, endUTC)
			argCount++
		}
	}

	// Get the events in order of start date.
	query += " ORDER BY earliest.first_date ASC NULLS LAST"

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("error querying in pass 1: %w", err)
	}
	defer rows.Close()

	// Each row represents an event that *might* meet the
	// criteria. Turn each row into an event.
	eventsMap := make(map[int]*models.Event)

	for rows.Next() {
		var e models.Event
		var eventInt int
		var venueInt sql.NullInt64
		var isVirtual bool
		var regionInt sql.NullInt32
		var firstDate *time.Time
		var eventDesc, venueName, streetAddress, city, state, zip, timezone sql.NullString

		err := rows.Scan(
			&eventInt,
			&e.Name,
			&eventDesc,
			&isVirtual,
			&venueInt,
			&venueName,
			&streetAddress,
			&city,
			&state,
			&zip,
			&timezone,
			&regionInt,
			&firstDate,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning event: %w", err)
		}

		e.ID = strconv.Itoa(eventInt)
		if eventDesc.Valid {
			e.Description = &eventDesc.String
		}
		e.EventType = GetEventType(isVirtual, venueInt.Valid)

		if venueInt.Valid {
			var venue models.Venue
			venue.ID = strconv.Itoa(int(venueInt.Int64))
			venue.Name = &venueName.String
			venue.Address = streetAddress.String
			venue.City = city.String
			venue.State = state.String
			venue.ZipCode = &zip.String
			venue.Timezone = timezone.String
			e.Venue = &venue
		} else {
			e.Venue = nil
		}

		// Have we already processed this event?
		_, exists := eventsMap[eventInt]
		if exists {
			// Duplicate rows can exist when the venue is in multiple regions.
			if (eventsMap[eventInt].Venue != nil) && (regionInt.Valid) {
				eventsMap[eventInt].Venue.Region = append(eventsMap[eventInt].Venue.Region, int(regionInt.Int32))
			}
		} else {
			if (e.Venue != nil) && (regionInt.Valid) {
				e.Venue.Region = append(e.Venue.Region, int(regionInt.Int32))
			}
			eventsMap[eventInt] = &e
		}
	}

	return eventsMap, nil
}

// The events from pass 1 have at least one shift on one job that matched the filter criteria.
// Now, get just the shifts within each event that match.
// While this appears to be doing the same thing we did in pass, 1 it is not: an event could
// have an opportunity that matched the job filter, but not the dates, while another has a
// shift that matched the dates, but not the job. Rare, but possible.
func FetchFilteredPassTwo(ctx context.Context, filter *models.EventFilterInput, idList []int, DB *sql.DB) (map[int]bool, error) {

	eventsWithShifts := make(map[int]bool, len(idList))

	shiftsQuery := `
		SELECT
			opp.event_id,
			COUNT(*) 
		FROM shifts s
		JOIN opportunities opp ON s.opportunity_id = opp.opportunity_id
		WHERE opp.event_id = ANY($1)
	`

	shiftArgs := []interface{}{pq.Array(idList)}
	argNum := 2

	if filter != nil {
		// Filter shifts by dates.
		if filter.ShiftStartDate != nil {
			startUTC, err := DateTimeToUTC(*filter.ShiftStartDate, *filter.IanaZone)
			if err != nil {
				return nil, fmt.Errorf("error converting date to UTC using %s %s: %w", *filter.ShiftStartDate, *filter.IanaZone, err)
			}
			shiftsQuery += fmt.Sprintf(" AND s.shift_start >= $%d", argNum)
			shiftArgs = append(shiftArgs, startUTC)
			argNum++
		}
		if filter.ShiftEndDate != nil {
			endUTC, err := DateTimeToUTC(*filter.ShiftEndDate, *filter.IanaZone)
			if err != nil {
				return nil, fmt.Errorf("error converting date to UTC using %s %s: %w", *filter.ShiftEndDate, *filter.IanaZone, err)
			}
			shiftsQuery += fmt.Sprintf(" AND s.shift_end <= $%d", argNum)
			shiftArgs = append(shiftArgs, endUTC)
			argNum++
		}

		// Filter by job.
		if len(filter.Jobs) > 0 {
			placeholders := []string{}
			for _, job := range filter.Jobs {
				placeholders = append(placeholders, fmt.Sprintf("$%d", argNum))
				dbJob := strings.ToLower(string(job))
				shiftArgs = append(shiftArgs, dbJob)
				argNum++
			}

			shiftsQuery += fmt.Sprintf(" AND opp.job IN (%s)", strings.Join(placeholders, ","))
		}
	}

	// We really just want to find out if each event has any shift
	// that matches the criteria.
	shiftsQuery += " GROUP BY opp.event_id"

	shiftRows, err := DB.QueryContext(ctx, shiftsQuery, shiftArgs...)
	if err != nil {
		return nil, fmt.Errorf("error querying shifts: %w", err)
	}
	defer shiftRows.Close()

	// If an event is not in the rows, it has no matching shifts.
	// Note: the way GROUP works in this query, we should not
	// expect have any rows with count = 0. The check is just
	// extra safety against future changes to the query.
	for shiftRows.Next() {
		var eInt int
		var count int

		err := shiftRows.Scan(&eInt, &count)
		if err != nil {
			return nil, fmt.Errorf("error scanning shift: %w", err)
		}

		if count > 0 {
			eventsWithShifts[eInt] = true
		}
	}

	return eventsWithShifts, nil
}
