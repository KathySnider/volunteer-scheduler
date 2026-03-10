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

func fetchVolunteerIdByEmail(ctx context.Context, DB *sql.DB, email string) (int, error) {
	var volunteerId int
	err := DB.QueryRowContext(ctx,
		"SELECT volunteer_id FROM volunteers WHERE email = $1", email).Scan(&volunteerId)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("no volunteer account found for this email")
	}
	if err != nil {
		return 0, fmt.Errorf("error looking up volunteer: %w", err)
	}
	return volunteerId, nil
}
func fetchProfile(ctx context.Context, DB *sql.DB, volId int) (*models.VolunteerProfile, error) {
	query := `
		SELECT 
			volunteer_id, 
			first_name, 
			last_name, 
			email, 
			phone, 
			zip_code
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
		&zip)

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

func assignVolToShift(ctx context.Context, DB *sql.DB, shiftId string, volId int) (*models.MutationResult, error) {
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
			COUNT(vs.volunteer_id) as curr_vols,
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
			Message: ptrString("Failed to assign volunteer to shift."),
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

	return &models.MutationResult{
		Success: true,
		Message: ptrString("Volunteer successfully assigned."),
		ID:      &shiftId,
	}, nil
}

func cancelShiftAssignment(ctx context.Context, DB *sql.DB, shiftId string, volId int) (*models.MutationResult, error) {
	shiftInt, err := strconv.Atoi(shiftId)
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("ShiftId is not valid."),
			ID:      nil,
		}, err
	}

	delete := `
		DELETE FROM volunteer_shifts
		WHERE volunteer_id = $1 AND shift_id = $2
	`
	_, err = DB.ExecContext(ctx, delete, volId, shiftInt)

	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to delete shift assignment."),
			ID:      nil,
		}, err
	}

	// No errors.

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
		WHERE vs.shift_id = $1
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
			earliest.first_date
        FROM events e
        LEFT JOIN venues v ON e.venue_id = v.venue_id
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
		// Filter by cities.
		if len(filter.Cities) > 0 {
			placeholders := []string{}
			for _, city := range filter.Cities {
				placeholders = append(placeholders, fmt.Sprintf("$%d", argCount))
				args = append(args, city)
				argCount++
			}
			query += fmt.Sprintf(" AND v.city IN (%s)", strings.Join(placeholders, ","))
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
		if !exists {
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
