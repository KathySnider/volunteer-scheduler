package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"

	"volunteer-scheduler/models"
)

// services/event_service.go

type EventService struct {
	DB           *sql.DB
	Mailer       *Mailer
	ShiftService *ShiftService
}

func NewEventService(db *sql.DB, mailer *Mailer, shiftService *ShiftService) (*EventService, error) {
	s := &EventService{
		DB:           db,
		Mailer:       mailer,
		ShiftService: shiftService,
	}

	return s, nil
}

// Queries.
// FetchLookups
// Retrieve regions, service types, job types,... anything
// that uses a lookup table.
func (s *EventService) FetchLookups(ctx context.Context) (*models.LookupValues, error) {

	var lookup models.LookupValues

	// Get regions.
	lookup.Regions = make([]*models.Region, 0)
	rows, err := s.DB.QueryContext(ctx, "Select region_id, code, name, is_active from regions WHERE is_active = true")
	if err != nil {
		return nil, fmt.Errorf("error querying regions: %w", err)
	}

	for rows.Next() {
		var r models.Region

		err = rows.Scan(&r.ID, &r.Code, &r.Name, &r.IsActive)
		if err != nil {
			rows.Close()
			return nil, fmt.Errorf("error scanning regions: %w", err)
		}
		lookup.Regions = append(lookup.Regions, &r)
	}
	rows.Close()

	// Get ServiceTypes.
	lookup.ServiceTypes = make([]*models.ServiceType, 0)
	rows, err = s.DB.QueryContext(ctx, "Select service_type_id, code, name from service_types")
	if err != nil {
		return nil, fmt.Errorf("error querying service types: %w", err)
	}
	for rows.Next() {
		var st models.ServiceType

		err = rows.Scan(&st.ID, &st.Code, &st.Name)
		if err != nil {
			rows.Close()
			return nil, fmt.Errorf("error scanning service types: %w", err)
		}
		lookup.ServiceTypes = append(lookup.ServiceTypes, &st)
	}
	rows.Close()

	// Get JobTypes.
	lookup.JobTypes = make([]*models.JobType, 0)
	rows, err = s.DB.QueryContext(ctx, "Select job_type_id, code, name, is_active from job_types")
	if err != nil {
		return nil, fmt.Errorf("error querying job types: %w", err)
	}
	for rows.Next() {
		var jt models.JobType

		err = rows.Scan(&jt.ID, &jt.Code, &jt.Name, &jt.IsActive)
		if err != nil {
			rows.Close()
			return nil, fmt.Errorf("error scanning job types: %w", err)
		}
		lookup.JobTypes = append(lookup.JobTypes, &jt)
	}
	rows.Close()

	return &lookup, nil
}

// FetchFilteredEvents
// Retrieve events based on filter criteria.
func (s *EventService) FetchFilteredEvents(ctx context.Context, filter *models.EventFilterInput) ([]*models.Event, error) {

	// Translate all of the filter stuff to a set of events that
	// potentially meet all of the user's criteria. If there are
	// no filters, the call to pass 1 returns all of the events.

	// orderedIDs carries the ORDER BY from the SQL query (earliest event date ASC).
	// We must use it when building the final slice — ranging over eventsMap directly
	// would randomise the order because Go maps have no guaranteed iteration order.
	eventsMap, orderedIDs, err := fetchFilteredPassOne(ctx, filter, s.DB)
	if err != nil {
		return nil, fmt.Errorf("error querying events: %w", err)
	}
	if len(eventsMap) == 0 {
		// Return an empty set of events. Nothing matched.
		return []*models.Event{}, nil
	}

	// Skip pass two if there is no filter.
	if filter != nil {
		// Now, for each of the selected events, determine which
		// have shifts that also meet the criteria. Pass 2 just
		// wants the list of ids.
		eventsWithShifts, err := fetchFilteredPassTwo(ctx, filter, orderedIDs, s.DB)
		if err != nil {
			return nil, fmt.Errorf("error querying events: %w", err)
		}

		// Get rid of events that had no shifts in pass 2.
		for id := range eventsMap {
			if !eventsWithShifts[id] {
				delete(eventsMap, id)
			}
		}
	}

	// Build the result slice in the order the DB returned the events.
	// Skipping any IDs that were removed by pass two.
	events := make([]*models.Event, 0, len(eventsMap))
	for _, id := range orderedIDs {
		if event, ok := eventsMap[id]; ok {
			events = append(events, event)
		}
	}
	return events, nil
}

func (s *EventService) FetchEventById(ctx context.Context, eventId string) (*models.Event, error) {
	eventInt, err := strconv.Atoi(eventId)
	if err != nil {
		return nil, fmt.Errorf("invalid event id: %w", err)
	}

	query := `
        SELECT 
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
			vr.region_id
        FROM events e
        LEFT JOIN venues v ON e.venue_id = v.venue_id
		LEFT JOIN venue_regions vr ON v.venue_id = vr.venue_id
		WHERE e.event_id = $1
    `

	row := s.DB.QueryRowContext(ctx, query, eventInt)

	var e models.Event
	e.ID = eventId

	var isVirtual bool
	var venueInt sql.NullInt64
	var eventDesc, venueName, streetAddress, city, state, zip, timezone sql.NullString
	var regionInt sql.NullInt32

	err = row.Scan(
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
	)
	if err != nil {
		return nil, fmt.Errorf("error scanning event: %w", err)
	}

	if eventDesc.Valid {
		e.Description = &eventDesc.String
	}
	e.EventType = GetEventType(isVirtual, venueInt.Valid)

	if venueInt.Valid {
		// Since venueInt is valid, most of the strings are valid, because
		// the fields are NOT NULL in the DB. The exceptions are venue name
		// and zip code; since we use pointers in those cases, we're fine.
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

	stPtrs, err := FetchEventServiceTypes(ctx, s.DB, eventInt)
	if err != nil {
		return nil, fmt.Errorf("error getting event's service types: %w", err)
	}
	// Convert the pointers to the actual strings.
	e.ServiceTypes = make([]string, len(stPtrs))
	for i := 0; i < len(stPtrs); i++ {
		e.ServiceTypes[i] = *stPtrs[i]
	}

	dates, err := FetchEventDates(ctx, s.DB, eventInt)
	if err != nil {
		return nil, fmt.Errorf("error getting event's dates: %w", err)
	}
	e.EventDates = dates

	// All good.
	return &e, nil
}

// Mutations: create.

// CreateEvent
// Creates the DB entry for the events table.
func (s *EventService) CreateEvent(ctx context.Context, newEvent models.NewEventInput) (*models.MutationResult, error) {
	var query string
	var virtualEvent bool
	var venueIdPtr *int
	var eventInt int

	// Determine whether or not the event will be virtual.
	// Both virtual and hybrid events have a virtual
	// component, so only in-person events are *not* vitual.
	virtualEvent = true
	if newEvent.EventType == models.EventTypeInPerson {
		virtualEvent = false
	}

	// Both in-person and hybrid events require a venue.
	if newEvent.EventType == models.EventTypeVirtual {
		// There s/b no venue for a virtual event.
		venueIdPtr = nil

	} else {
		// A venue is required.
		if newEvent.VenueId == nil {
			return nil, fmt.Errorf("A venue is required for in-person and hybrid events.")
		}

		venueInt, err := strconv.Atoi(*newEvent.VenueId)
		if err != nil {
			return nil, fmt.Errorf("The value at VenueId was not a Valid ID. %w", err)
		}

		venueIdPtr = &venueInt
	}

	// Add the event and it's dates inside a transaction.

	// Get a Tx for making transaction requests.
	var tx *sql.Tx

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		err = fmt.Errorf("error starting transaction: %w", err)
		return nil, err
	}
	// Defer a rollback in case anything fails.
	defer tx.Rollback()

	// The rollback is for insurance. The rollback will occur if we
	// leave the scope of the transaction before it has ended. For
	// good DB practice, DO NOT RETURN while inside of a transaction.

	// Create the event first. We need the id to continue.
	if venueIdPtr == nil {
		query = `
		INSERT INTO events (event_name, description, event_is_virtual)
		VALUES ($1, $2, $3)
		RETURNING event_id
	`
		err = tx.QueryRowContext(ctx, query, newEvent.Name, newEvent.Description, virtualEvent).Scan(&eventInt)

	} else {
		query = `
		INSERT INTO events (event_name, description, event_is_virtual, venue_id)
		VALUES ($1, $2, $3, $4)
		RETURNING event_id
	`
		err = tx.QueryRowContext(ctx, query, newEvent.Name, newEvent.Description, virtualEvent, *venueIdPtr).Scan(&eventInt)
	}

	if err == nil {
		// Event was inserted. Add the dates.
		err = AddNewEventDates(ctx, newEvent.EventDates, eventInt, tx)
	} else {
		// Save all of the information about what failed.
		err = fmt.Errorf("error inserting the event: %w", err)
	}

	if err == nil {
		err = s.AddServiceTypesToEvent(ctx, tx, eventInt, newEvent.ServiceTypes)
	} else {
		err = fmt.Errorf("error adding dates to the event: %w", err)
	}

	if err != nil {
		tx.Rollback()

		// NOW return an error ...
		return &models.MutationResult{
			Success: false,
			Message: ptrString("transaction failed and was rolled back."),
			ID:      nil,
		}, err
	}

	// All good. Commit and return the new event ID.
	err = tx.Commit()
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("error committing transaction"),
			ID:      nil,
		}, err
	}
	return &models.MutationResult{
		Success: true,
		Message: ptrString("Event successfully created."),
		ID:      ptrString(strconv.Itoa(eventInt)),
	}, nil
}

func (s *EventService) AddServiceTypesToEvent(ctx context.Context, tx *sql.Tx, eventId int, serviceTypes []int) error {

	query := `
		INSERT INTO event_service_types (event_id, service_type_id)
		VALUES ($1, $2)
		`
	for _, serviceTypeId := range serviceTypes {
		_, err := tx.ExecContext(ctx, query, eventId, serviceTypeId)

		if err != nil {
			return fmt.Errorf("error adding service type to event: %w", err)
		}
	}

	// No errors.
	return nil
}

// This function is to add a startdate and enddate to an extant event.
func (s *EventService) CreateEventDate(ctx context.Context, dates models.AddEventDateInput) (*models.MutationResult, error) {
	var startUTC, endUTC *string
	startUTC, err := DateTimeToUTC(dates.StartDateTime, dates.IanaZone)
	if err == nil {
		endUTC, err = DateTimeToUTC(dates.EndDateTime, dates.IanaZone)
	}
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to create event date. Invalid datetimes or IANA zone."),
			ID:      nil,
		}, err
	}

	eventInt, err := strconv.Atoi(dates.EventID)
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to create event date. Invalid event Id."),
			ID:      nil,
		}, err
	}

	create := `
		INSERT INTO event_dates (event_id, start_date_time, end_date_time)
		VALUES ($1, $2, $3)
		RETURNING event_date_id
	`
	var eventDateInt int
	err = s.DB.QueryRowContext(ctx, create, eventInt, startUTC, endUTC).Scan(&eventDateInt)
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to insert event date."),
			ID:      nil,
		}, err
	}
	return &models.MutationResult{
		Success: true,
		Message: ptrString("Successfully created event date."),
		ID:      ptrString(strconv.Itoa(eventDateInt)),
	}, nil
}

// Mutations: Update, delete.

func (s *EventService) UpdateEvent(ctx context.Context, event models.UpdateEventInput) (*models.MutationResult, error) {

	eventInt, err := strconv.Atoi(event.ID)
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to update event; invalid event Id."),
			ID:      &event.ID,
		}, err
	}

	isVirtual := (event.EventType == models.EventTypeVirtual || event.EventType == models.EventTypeHybrid)

	var venueInt *int
	if event.EventType == models.EventTypeVirtual {
		venueInt = nil
	} else {
		// Other types require a venue.
		if event.VenueId == nil {
			return &models.MutationResult{
				Success: false,
				Message: ptrString("Failed to update event; event must have a venue id."),
				ID:      &event.ID,
			}, err
		}
		idInt, err := strconv.Atoi(*event.VenueId)
		if err != nil {
			return &models.MutationResult{
				Success: false,
				Message: ptrString("Failed to update event; event must have a valid venue id."),
				ID:      &event.ID,
			}, err
		}
		venueInt = &idInt
	}

	query := `
		UPDATE events 
		SET 
			event_name = $1,
			description = $2, 
			event_is_virtual = $3,
			venue_id = $4
		WHERE event_id = $5
	`
	_, err = s.DB.ExecContext(ctx, query, event.Name, event.Description, isVirtual, venueInt, eventInt)

	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to update event."),
			ID:      &event.ID,
		}, err
	}

	return &models.MutationResult{
		Success: true,
		Message: ptrString("Successfully updated event."),
		ID:      &event.ID,
	}, nil
}

func (s *EventService) UpdateEventDate(ctx context.Context, evDate models.UpdateEventDateInput) (*models.MutationResult, error) {
	var startUTC, endUTC *string
	startUTC, err := DateTimeToUTC(evDate.StartDateTime, evDate.IanaZone)
	if err == nil {
		endUTC, err = DateTimeToUTC(evDate.EndDateTime, evDate.IanaZone)
	}
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to update event date; invalid datetimes or IANA zone."),
			ID:      nil,
		}, err
	}

	dateInt, err := strconv.Atoi(evDate.ID)
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to update event date; invalid event_date_id."),
			ID:      &evDate.ID,
		}, err
	}

	query := `
		UPDATE event_dates 
		SET 
			start_date_time = $1,
			end_date_time = $2
		WHERE event_date_id = $3
	`
	_, err = s.DB.ExecContext(ctx, query, startUTC, endUTC, dateInt)

	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to update event date."),
			ID:      &evDate.ID,
		}, err
	}

	return &models.MutationResult{
		Success: true,
		Message: ptrString("Successfully updated event date."),
		ID:      &evDate.ID,
	}, nil
}

// Send emails to all affected volunteers and staff, THEN delete the event.
func (s *EventService) DeleteEvent(ctx context.Context, eventId string) (*models.MutationResult, error) {
	eventInt, err := strconv.Atoi(eventId)
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to delete event; invalid event ID."),
			ID:      &eventId,
		}, err
	}

	// Get all of the information out of the DB that we'll need
	// to send coherent emails.
	query := `
		SELECT
		  e.event_name,
		  ven.timezone,
		  vs.shift_id,
		  s.shift_start,
		  s.shift_end,
		  vol.volunteer_id,
		  vol.email,
		  vol.first_name,
		  vol.last_name,
		  staff.staff_id,
		  staff.email,
		  staff.first_name,
		  staff.last_name
		FROM events e
		LEFT JOIN opportunities opp ON opp.event_id = e.event_id
		LEFT JOIN shifts s ON s.opportunity_id = opp.opportunity_id
		LEFT JOIN volunteer_shifts vs ON vs.shift_id = s.shift_id AND vs.cancelled_at IS NULL
		LEFT JOIN volunteers vol ON vol.volunteer_id = vs.volunteer_id
		LEFT JOIN staff ON staff.staff_id = s.staff_contact_id
		LEFT JOIN venues ven ON ven.venue_id = e.venue_id
	WHERE e.event_id = $1
	`
	rows, err := s.DB.QueryContext(ctx, query, eventInt)
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to delete event; unable to query DB."),
			ID:      &eventId,
		}, err
	}
	defer rows.Close()

	// Create some structures to handle the information needed for an email
	// for each user affected by the cancellation.
	type emailInfo struct {
		email     string
		firstName string
		lastName  string
		shiftsMap map[int]bool
	}
	// Make a map of volunteers to email, and a map for staff
	// leads. We'll gather all of the information from the
	// subsequent calls, and, finally send all of the emails.
	volMap := make(map[int]*emailInfo)
	leadMap := make(map[int]*emailInfo)
	shiftMap := make(map[int]*ShiftSummary)

	var eventName string
	var venueZone sql.NullString

	for rows.Next() {
		// This is a little tricky, since we could have a row for a
		// shift that has a staff lead, but no volunteers have yet
		// signed up, or a row with volunteers, but no staff lead.
		// So a lot of this information may be NULLs coming from
		// the DB.
		// Also, since there is only one event Id, and
		// either 0 or 1 venues, the eventName and venueZone s/b
		// the same in every row. I don't bother to check for that.

		var shiftId int
		var shiftStart, shiftEnd string
		var volId sql.NullInt32
		var volEmail, volFirst, volLast sql.NullString
		var leadId sql.NullInt32
		var leadEmail, leadFirst, leadLast sql.NullString

		err := rows.Scan(
			&eventName,
			&venueZone,
			&shiftId,
			&shiftStart,
			&shiftEnd,
			&volId,
			&volEmail,
			&volFirst,
			&volLast,
			&leadId,
			&leadEmail,
			&leadFirst,
			&leadLast,
		)
		if err != nil {
			return &models.MutationResult{
				Success: false,
				Message: ptrString("Failed to delete event; unable to scan rows."),
				ID:      &eventId,
			}, err
		}

		_, shiftExists := shiftMap[shiftId]
		if !shiftExists {
			var ss, se *string
			var shift ShiftSummary

			// Convert the dates and times once and save them.
			if venueZone.Valid {
				ss, err = UTCToTimeZone(shiftStart, venueZone.String)
				if err == nil {
					se, err = UTCToTimeZone(shiftEnd, venueZone.String)
				}
			} else {
				ss, err = UTCToDateTime(shiftStart)
				if err == nil {
					se, err = UTCToDateTime(shiftEnd)
				}
			}
			if err != nil {
				return &models.MutationResult{
					Success: false,
					Message: ptrString("Failed to delete event; unable to convert datetimes."),
					ID:      &eventId,
				}, err
			}
			shift.Start = *ss
			shift.End = *se

			shiftMap[shiftId] = &shift
		}
		if volId.Valid {
			volInt := int(volId.Int32)
			_, volExists := volMap[volInt]
			if volExists {
				// Is this a new shift for this vol?
				_, shiftExists := volMap[volInt].shiftsMap[shiftId]
				if shiftExists {
					// Due to multiple-multiple joins, and possible
					// situations, don't worry about this.
				} else {
					volMap[volInt].shiftsMap[shiftId] = true
				}
			} else {
				// Haven't seen this volunteer yet. Since volId is
				// not NULL, the volunteer's e
				// mail, first- and last-
				// names are also not NULL.
				var vol emailInfo
				vol.shiftsMap = make(map[int]bool)

				vol.email = volEmail.String
				vol.firstName = volFirst.String
				vol.lastName = volLast.String
				vol.shiftsMap[shiftId] = true
				volMap[volInt] = &vol
			}
		}
		if leadId.Valid {
			leadInt := int(leadId.Int32)
			_, leadExists := leadMap[leadInt]
			if leadExists {
				_, shiftExists := leadMap[leadInt].shiftsMap[shiftId]
				if shiftExists {
					// Do nothing
				} else {
					leadMap[leadInt].shiftsMap[shiftId] = true
				}
			} else {
				// Haven't processed this staff lead.
				var lead emailInfo
				lead.shiftsMap = make(map[int]bool)

				lead.email = leadEmail.String
				lead.firstName = leadFirst.String
				lead.lastName = leadLast.String
				lead.shiftsMap[shiftId] = true
				leadMap[leadInt] = &lead
			}
		}
	}

	// Our maps now have a single entry for each volunteer and staff
	// lead contact. We also have the event name, and the formatted
	// dates/times for each shift.

	unsent := make([]*string, 0, len(volMap)+len(leadMap))

	for _, emailInfo := range volMap {
		// Get all of the shift start and end times for this email.
		shiftSummaries := make([]ShiftSummary, 0, len(emailInfo.shiftsMap))
		for shiftKey := range emailInfo.shiftsMap {
			shiftSummaries = append(shiftSummaries, *shiftMap[shiftKey])
		}
		err = sendEventCancelledToVolunteer(ctx, s.Mailer, emailInfo.firstName, eventName, shiftSummaries, emailInfo.email)
		if err != nil {
			// Not being able to send an email is not fatal.
			// Try to notify the others.
			unsent = append(unsent, &emailInfo.email)
			continue
		}
	}
	for _, emailInfo := range leadMap {
		// Get all of the shift start and end times for this email.
		shiftSummaries := make([]ShiftSummary, 0, len(emailInfo.shiftsMap))
		for shiftKey := range emailInfo.shiftsMap {
			shiftSummaries = append(shiftSummaries, *shiftMap[shiftKey])
		}
		err = sendEventCancelledToStaff(ctx, s.Mailer, emailInfo.firstName, eventName, shiftSummaries, emailInfo.email)
		if err != nil {
			unsent = append(unsent, &emailInfo.email)
			continue
		}
	}

	if len(unsent) > 0 {
		log.Println("Unable to send the event (" + eventName + ") cancelled message to the following emails:")
		for _, emailStr := range unsent {
			log.Println(emailStr)
		}
	}

	// Finally, delete the event (which will cascade to the opportunities, shifts, and volunteer_shifts).
	_, err = s.DB.ExecContext(ctx, "DELETE FROM events WHERE event_id = $1", eventInt)
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to delete event from DB."),
			ID:      &eventId,
		}, err
	}

	return &models.MutationResult{
		Success: true,
		Message: ptrString("Successfully deleted event."),
		ID:      &eventId,
	}, nil
}

func (s *EventService) DeleteEventDate(ctx context.Context, evDateId string) (*models.MutationResult, error) {
	dateInt, err := strconv.Atoi(evDateId)
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to delete event date; invalid event_date_id."),
			ID:      &evDateId,
		}, err
	}

	_, err = s.DB.ExecContext(ctx, "DELETE FROM event_dates WHERE event_date_id = $1", dateInt)

	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to delete event date."),
			ID:      &evDateId,
		}, err
	}
	return &models.MutationResult{
		Success: true,
		Message: ptrString("Successfully deleted event date."),
		ID:      &evDateId,
	}, nil
}
