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
	DB               *sql.DB
	Mailer           *Mailer
	ShiftService     *ShiftService
	serviceTypeCache map[string]int
}

func NewEventService(db *sql.DB, mailer *Mailer, shiftService *ShiftService) (*EventService, error) {
	s := &EventService{
		DB:           db,
		Mailer:       mailer,
		ShiftService: shiftService,
	}

	// Load cache on initialization
	if err := s.loadServiceTypeCache(); err != nil {
		return nil, err
	}

	return s, nil
}

// We only need to get the service categories at the start, or
// if they change (unlikely).
func (s *EventService) loadServiceTypeCache() error {
	s.serviceTypeCache = make(map[string]int)
	rows, err := s.DB.Query("SELECT service_type_id, code FROM service_types")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var code string
		if err := rows.Scan(&id, &code); err != nil {
			return err
		}
		s.serviceTypeCache[code] = id
	}
	return nil
}

// Queries.

// FetchFilteredEvents
// Retrieve events based on filter criteria.
func (s *EventService) FetchFilteredEvents(ctx context.Context, filter *models.EventFilterInput) ([]*models.Event, error) {

	// Translate all of the filter stuff to a set of events that
	// potentially meet all of the user's criteria. If there are
	// no filters, the call to pass 1 returns all of the events.

	eventsMap, err := FetchFilteredPassOne(ctx, filter, s.DB)
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
		eventIDs := []int{}
		for id := range eventsMap {
			eventIDs = append(eventIDs, id)
		}

		eventsWithShifts, err := FetchFilteredPassTwo(ctx, filter, eventIDs, s.DB)
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

	// Convert map to slice
	events := make([]*models.Event, 0, len(eventsMap))
	for _, event := range eventsMap {
		events = append(events, event)
	}
	return events, nil
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

func (s *EventService) AddServiceTypesToEvent(ctx context.Context, tx *sql.Tx, eventId int, serviceTypes []models.ServiceType) error {

	query := `
		INSERT INTO event_service_types (event_id, service_type_id)
		VALUES ($1, $2)
		`
	for _, serviceType := range serviceTypes {
		serviceTypeId, ok := s.serviceTypeCache[string(serviceType)]
		if !ok {
			return fmt.Errorf("unknown service type: %s", serviceType)
		}

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
	type shiftInfo struct {
		shiftStart string
		shiftEnd   string
	}
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
	shiftMap := make(map[int]*shiftInfo)

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
			var shift shiftInfo

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
			shift.shiftStart = *ss
			shift.shiftEnd = *se

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
	// TODO: Get the proper text for the emails.

	subject := eventName + " has been cancelled"
	volText := "Thank you for signing up to be a volunteer for " + eventName + ". Unfortunately this event has been cancelled ... shift times, etc."
	leadText := "Thank you for signing up to be the lead contact for " + eventName + "...."
	unsent := make([]*string, 0, len(volMap)+len(leadMap))

	for _, emailInfo := range volMap {
		greeting := "Dear " + emailInfo.firstName + " " + emailInfo.lastName + ",\n"
		err = s.Mailer.SendEmail(ctx, emailInfo.email, subject, "", greeting+volText)
		if err != nil {
			// Not being able to send an email is not fatal.
			// Try to notify the others.
			unsent = append(unsent, &emailInfo.email)
			continue
		}
	}
	for _, emailInfo := range leadMap {
		greeting := "Dear " + emailInfo.firstName + " " + emailInfo.lastName + ",\n"
		err = s.Mailer.SendEmail(ctx, emailInfo.email, subject, "", greeting+leadText)
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
	// TODO: Delete the entry.
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
