package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"

	"volunteer-scheduler/models"

	"github.com/google/uuid"
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
// Retrieve cities, service types, job types,... anything
// that uses a lookup table.
func (s *EventService) FetchLookups(ctx context.Context) (*models.LookupValues, error) {

	var lookup models.LookupValues

	// Get FundingEntities.
	lookup.FundingEntities = make([]*models.FundingEntity, 0)
	rows, err := s.DB.QueryContext(ctx, "SELECT id, name, COALESCE(description, '') FROM funding_entities WHERE is_active = true ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("error querying funding entities: %w", err)
	}

	for rows.Next() {
		var fe models.FundingEntity

		err = rows.Scan(&fe.ID, &fe.Name, &fe.Description)
		if err != nil {
			rows.Close()
			return nil, fmt.Errorf("error scanning funding entities: %w", err)
		}
		lookup.FundingEntities = append(lookup.FundingEntities, &fe)
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
	rows, err = s.DB.QueryContext(ctx, "Select job_type_id, code, name, is_active from job_types WHERE is_active = true")
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

	// Get cities from venues.

	lookup.Cities = make([]string, 0)
	rows, err = s.DB.QueryContext(ctx, "SELECT DISTINCT city FROM venues ORDER BY city")
	if err != nil {
		rows.Close()
		return nil, fmt.Errorf("error querying cities from venues: %w", err)
	}
	for rows.Next() {
		var city string
		err = rows.Scan(&city)
		if err != nil {
			rows.Close()
			return nil, fmt.Errorf("error scanning city: %w", err)
		}
		lookup.Cities = append(lookup.Cities, city)
	}

	return &lookup, nil
}

// FetchFilteredEventsWithShifts
// There *are* significant differences between this function and FetchManagedEvents:
//  * This call does a second pass to make sure that it only has events with shifts.
//  * This call returns volunteer events (no recurrence stuff that admins need).
//  * This call must have the id of the volunteer who called it, in case distance is
//    part of the criteria (the volunteers' lat/lng data is stored in their profiles)

func (s *EventService) FetchFilteredEventsWithShifts(ctx context.Context, filter *models.EventFilterInput, volId *int) ([]*models.Event, error) {

	// Translate all of the filter stuff to a set of events that potentially meet
	// all of the user's criteria. If there are no filters, the call to pass 1 just
	// returns all of the events.

	// orderedIDs carries the ORDER BY from the SQL query (earliest event date ASC).
	// We must use it when building the final slice — ranging over eventsMap directly
	// would randomise the order because Go maps have no guaranteed iteration order.
	eventsMap, orderedIDs, err := fetchFilteredPassOne(ctx, filter, s.DB, *volId)
	if err != nil {
		return nil, fmt.Errorf("error querying events: %w", err)
	}
	if len(eventsMap) == 0 {
		// Return an empty set of events. Nothing matched.
		return []*models.Event{}, nil
	}

	// Now, for each of the selected events, determine which
	// have shifts. Pass 2 just wants the list of ids.
	eventsWithShifts, err := fetchFilteredPassTwo(ctx, s.DB, orderedIDs)
	if err != nil {
		return nil, fmt.Errorf("error querying events: %w", err)
	}

	// Build the result slice in the order the DB returned the events.
	// Skipping any IDs that were removed by pass two.
	events := make([]*models.Event, 0, len(eventsMap))
	for _, id := range orderedIDs {
		hasShifts, _ := eventsWithShifts[id]
		if hasShifts {
			event, _ := eventsMap[id]
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
			e.timezone
        FROM events e
        LEFT JOIN venues v ON e.venue_id = v.venue_id
		WHERE e.event_id = $1
    `

	row := s.DB.QueryRowContext(ctx, query, eventInt)

	var e models.Event
	e.ID = eventId

	var isVirtual bool
	var venueInt sql.NullInt64
	var eventDesc, venueName, streetAddress, city, state, zip sql.NullString

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
		&e.Timezone,
	)
	if err != nil {
		return nil, fmt.Errorf("error scanning event: %w", err)
	}

	if eventDesc.Valid {
		e.Description = &eventDesc.String
	} else {
		e.Description = nil
	}
	e.EventType = GetEventType(isVirtual, venueInt.Valid)

	if venueInt.Valid {
		// Since venueInt is valid, most of the strings are valid, because
		// the fields are NOT NULL in the DB. The exceptions are venue name
		// and zip code.
		var venue models.Venue
		venue.ID = strconv.Itoa(int(venueInt.Int64))
		venue.Address = streetAddress.String
		venue.City = city.String
		venue.State = state.String
		if venueName.Valid {
			venue.Name = &venueName.String
		} else {
			venue.Name = nil
		}
		if zip.Valid {
			venue.ZipCode = &zip.String
		} else {
			venue.ZipCode = nil
		}
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

func (s *EventService) FetchManagedEvents(ctx context.Context, filter *models.EventFilterInput) ([]*models.ManagedEvent, error) {

	// Translate all of the filter stuff to a set of events that meet all of the
	// caller's criteria. If there are no filters, return all events.

	eventsMap, orderedIDs, err := filterManagedEvents(ctx, filter, s.DB)
	if err != nil {
		return nil, fmt.Errorf("error querying events: %w", err)
	}
	if len(eventsMap) == 0 {
		// Return an empty set of events. Nothing matched.
		return []*models.ManagedEvent{}, nil
	}

	// orderedIDs carries the ORDER BY from the SQL query (earliest event date ASC).
	// We must use it when building the final slice — ranging over eventsMap directly
	// would randomise the order because Go maps have no guaranteed iteration order.
	// Build the result slice in the order the DB returned the events.

	events := make([]*models.ManagedEvent, 0, len(eventsMap))
	for _, id := range orderedIDs {
		event, _ := eventsMap[id]
		events = append(events, event)
	}
	return events, nil
}

func (s *EventService) FetchManagedEventById(ctx context.Context, eventId string) (*models.ManagedEvent, error) {
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
			e.timezone,
			fe.id,
			fe.name,
			fe.description,
			e.recurrence_group_id,
			e.recurrence_order,
			rg.pattern,
			rg.max_occurrences,
			rg.weekday_ordinal
        FROM events e
        LEFT JOIN venues v ON e.venue_id = v.venue_id
		JOIN funding_entities fe ON fe.id = e.funding_entity_id
		LEFT JOIN recurrence_groups rg ON rg.id = e.recurrence_group_id
		WHERE e.event_id = $1
    `

	row := s.DB.QueryRowContext(ctx, query, eventInt)

	var e models.ManagedEvent
	var fe models.FundingEntity

	e.ID = eventId

	var isVirtual bool
	var venueInt sql.NullInt64
	var eventDesc sql.NullString
	var venueName, streetAddress, city, state, zip sql.NullString
	var feDesc, recurGrpId sql.NullString
	var recurOrder sql.NullInt32
	var recurPattern, recurOrdinal sql.NullString
	var recurMax sql.NullInt32

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
		&e.Timezone,
		&fe.ID,
		&fe.Name,
		&feDesc,
		&recurGrpId,
		&recurOrder,
		&recurPattern,
		&recurMax,
		&recurOrdinal,
	)
	if err != nil {
		return nil, fmt.Errorf("error scanning event: %w", err)
	}

	if eventDesc.Valid {
		e.Description = &eventDesc.String
	} else {
		e.Description = nil
	}
	e.EventType = GetEventType(isVirtual, venueInt.Valid)

	if venueInt.Valid {
		// Since venueInt is valid, most of the strings are valid, because
		// the fields are NOT NULL in the DB. The exceptions are venue name
		// and zip code.
		var venue models.Venue
		venue.ID = strconv.Itoa(int(venueInt.Int64))
		venue.Address = streetAddress.String
		venue.City = city.String
		venue.State = state.String
		if venueName.Valid {
			venue.Name = &venueName.String
		} else {
			venue.Name = nil
		}
		if zip.Valid {
			venue.ZipCode = &zip.String
		} else {
			venue.ZipCode = nil
		}
		e.Venue = &venue
	} else {
		e.Venue = nil
	}

	if feDesc.Valid {
		fe.Description = &feDesc.String
	}
	e.FundingEntity = fe

	if recurGrpId.Valid {
		e.RecurrenceId = recurGrpId.String
		if recurOrder.Valid {
			e.RecurrenceOrder = int(recurOrder.Int32)
		} else {
			return nil, fmt.Errorf("recurrence_order is required when recurrence_group_id is not null.")
		}
		if recurPattern.Valid {
			e.RecurrencePattern = recurPattern.String
		}
		if recurMax.Valid {
			v := int(recurMax.Int32)
			e.RecurrenceMaxOccurrences = &v
		}
		if recurOrdinal.Valid {
			e.RecurrenceOrdinal = &recurOrdinal.String
		}
	}

	stPtrs, err := FetchEventServiceTypes(ctx, s.DB, eventInt)
	if err != nil {
		return nil, fmt.Errorf("error getting event's service types: %w", err)
	}
	// Convert the pointers to the actual strings.
	e.ServiceTypes = make([]string, len(stPtrs))
	for i, strPtr := range stPtrs {
		e.ServiceTypes[i] = *strPtr
	}

	dates, err := FetchEventDates(ctx, s.DB, eventInt)
	if err != nil {
		return nil, fmt.Errorf("error getting event's dates: %w", err)
	}
	e.EventDates = dates

	// All good.
	return &e, nil
}

// Mutations: Create.

// CreateEvent
// Creates the DB entry for the events table, and the entries in
// the associated tables.
func (s *EventService) CreateEvent(ctx context.Context, newEvent models.NewEventInput) (*models.MutationResult, error) {
	var venueIdPtr *int

	// Determine whether or not the event will be virtual.
	// Both virtual and hybrid events have a virtual
	// component, so only in-person events are *not* vitual.
	virtualEvent := true
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

	// The event dates and times need some checking:
	//  * There must be at least one date.
	//  * An event should not end before it starts.
	//  * Each day's end time should not be before it's start time.
	// Do this checking before we start a transaction and have to roll back.
	if len(newEvent.EventDates) == 0 {
		return nil, fmt.Errorf("There must be at least one event date.")
	}
	for i := 0; i < len(newEvent.EventDates); i++ {
		if newEvent.EventDates[i].EndDateTime <= newEvent.EventDates[i].StartDateTime {
			return nil, fmt.Errorf("End time must be after start time for each date.")
		}
	}

	// This function is about to split. We *might* be creating a
	// recurring event, or we might be creating a single event.
	// If single, call createEventInstance and return the result.
	if newEvent.Recurrence == nil {
		// Create a single event.
		return s.createSingleEvent(ctx, newEvent, virtualEvent, venueIdPtr)
	}

	// To create reucurring events, everything starts with the dates.
	evDatesMap, err := createDatesForPattern(newEvent.EventDates, newEvent.Timezone, *newEvent.Recurrence)
	if err != nil {
		return nil, fmt.Errorf("unable to create dates for recurring event: %w", err)
	}
	if evDatesMap == nil {
		return nil, fmt.Errorf("unable to create dates map for recurring event.")
	}

	// Get a UUID for the whole group.
	groupId := uuid.New()

	// Now we're ready to create all of these events.
	// Put this whole thing into a transaction.
	var tx *sql.Tx

	tx, err = s.DB.BeginTx(ctx, nil)
	if err != nil {
		err = fmt.Errorf("error starting transaction: %w", err)
		return nil, err
	}
	// Defer a rollback in case anything fails.
	defer tx.Rollback()

	// Save the recurrence settings so they can be displayed later.
	var ordinalStr *string
	if newEvent.Recurrence.WeekdayOrdinal != nil {
		s := string(*newEvent.Recurrence.WeekdayOrdinal)
		ordinalStr = &s
	}
	_, err = tx.ExecContext(ctx,
		`INSERT INTO recurrence_groups (id, pattern, max_occurrences, weekday_ordinal)
		 VALUES ($1, $2, $3, $4)`,
		groupId.String(),
		string(newEvent.Recurrence.Pattern),
		newEvent.Recurrence.MaxOccurrences,
		ordinalStr,
	)
	if err != nil {
		return nil, fmt.Errorf("error saving recurrence group: %w", err)
	}

	// The map doesn't guarantee in what order the dates will be served.
	// That's fine - we've saved the event's order (within the group) as
	// the key to the map. We don't care which instance is created first
	// or last, just that we know what events to change or delete if asked
	// to do so to all "future" events.
	for key, evDates := range *evDatesMap {

		// Create one instance of this group of events.
		mut, err := s.createEventRecurrence(ctx, tx, newEvent, virtualEvent, venueIdPtr, evDates, groupId, key)
		if err != nil {
			tx.Rollback() // Overkill by an OLD developer - would rollback when I exit this scope.
			return nil, fmt.Errorf("transaction failed to create instance with order %v: %w", key, err)
		}

		log.Printf("created event instance with order %v and id %v", key, *mut.ID)
	}

	// All good. Commit the transaction.
	err = tx.Commit()
	if err != nil {
		return nil, fmt.Errorf("error committing transaction: %w", err)
	}

	// No errors.
	strGroupId := groupId.String()
	return &models.MutationResult{
		Success: true,
		Message: ptrString("Successfully created a group of events."),
		ID:      &strGroupId,
	}, nil
}

// This function is to add a startdate and enddate to an extant event. Event id is known and
// is supplied in the input struct.
func (s *EventService) CreateEventDate(ctx context.Context, dates models.AddEventDateInput) (*models.MutationResult, error) {

	eventInt, err := strconv.Atoi(dates.EventID)
	if err != nil {
		return nil, fmt.Errorf("failed to add dates; invalid event id: %w", err)
	}

	var timezone string
	err = s.DB.QueryRowContext(ctx, "SELECT timezone FROM events WHERE event_id = $1", eventInt).Scan(&timezone)
	if err != nil {
		return nil, fmt.Errorf("unable to get timezone from event: %w", err)
	}

	if dates.EndDateTime <= dates.StartDateTime {
		return nil, fmt.Errorf("end time must be after start time")
	}

	var startUTC, endUTC *string
	startUTC, err = DateTimeToUTC(dates.StartDateTime, timezone)
	if err == nil {
		endUTC, err = DateTimeToUTC(dates.EndDateTime, timezone)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to add dates; invalid datetimes or timezone: %w", err)
	}

	insert := `
		INSERT INTO event_dates (event_id, start_date_time, end_date_time)
		VALUES ($1, $2, $3)
		RETURNING event_date_id
	`
	var eventDateInt int
	err = s.DB.QueryRowContext(ctx, insert, eventInt, startUTC, endUTC).Scan(&eventDateInt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert event date: %w", err)
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
		return nil, fmt.Errorf("failed to update event; invalid event id: %w", err)
	}

	isVirtual := (event.EventType == models.EventTypeVirtual || event.EventType == models.EventTypeHybrid)

	var venueInt *int
	if event.EventType == models.EventTypeVirtual {
		venueInt = nil
	} else {
		// Other types require a venue.
		if event.VenueId == nil {
			return nil, fmt.Errorf("failed to update event; non-virtual event must have a venue id")
		}
		idInt, err := strconv.Atoi(*event.VenueId)
		if err != nil {
			return nil, fmt.Errorf("failed to update event; invalid venue id: %w", err)
		}
		venueInt = &idInt
	}

	query := `
		UPDATE events
		SET
			event_name = $1,
			description = $2,
			event_is_virtual = $3,
			venue_id = $4,
			timezone = $5,
			funding_entity_id = $6
		WHERE event_id = $7
	`
	_, err = s.DB.ExecContext(ctx, query, event.Name, event.Description, isVirtual, venueInt, event.Timezone, event.FundingEntityID, eventInt)

	if err != nil {
		return nil, fmt.Errorf("failed to update event: %w", err)
	}

	return &models.MutationResult{
		Success: true,
		Message: ptrString("Successfully updated event."),
		ID:      &event.ID,
	}, nil
}

func (s *EventService) UpdateEventDate(ctx context.Context, evDate models.UpdateEventDateInput) (*models.MutationResult, error) {

	dateInt, err := strconv.Atoi(evDate.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to update event date; invalid event_date_id: %w", err)
	}

	var eventId int
	var timezone string
	query := `
		SELECT
			ed.event_id,
			e.timezone
		FROM event_dates ed
		JOIN events e ON e.event_id = ed.event_id
		WHERE ed.event_date_id = $1	
		`

	err = s.DB.QueryRowContext(ctx, query, dateInt).Scan(&eventId, &timezone)
	if err != nil {
		return nil, fmt.Errorf("failed to get timezone from events: %w", err)
	}

	if evDate.EndDateTime <= evDate.StartDateTime {
		return nil, fmt.Errorf("end time must be after start time")
	}

	var startUTC, endUTC *string
	startUTC, err = DateTimeToUTC(evDate.StartDateTime, timezone)
	if err == nil {
		endUTC, err = DateTimeToUTC(evDate.EndDateTime, timezone)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update event date; invalid datetimes or timezone: %w", err)
	}

	update := `
		UPDATE event_dates 
		SET 
			start_date_time = $1,
			end_date_time = $2
		WHERE event_date_id = $3
	`
	_, err = s.DB.ExecContext(ctx, update, startUTC, endUTC, dateInt)

	if err != nil {
		return nil, fmt.Errorf("failed to update event date; invalid datetimes or timezone: %w", err)
	}

	return &models.MutationResult{
		Success: true,
		Message: ptrString("Successfully updated event date."),
		ID:      &evDate.ID,
	}, nil
}

// Send emails to all affected volunteers and staff, THEN delete the event.
func (s *EventService) DeleteEvent(ctx context.Context, eventId string, scope *models.RecurrenceUpdateScope) (*models.MutationResult, error) {

	eventInt, err := strconv.Atoi(eventId)
	if err != nil {
		return nil, fmt.Errorf("Failed to delete event; invalid event ID (%s): %w", eventId, err)
	}

	// Get information from the current event that will be the same for all of the
	// emails. We'll need the timezone of the event to format shift datetimes.
	// Note: if this is a "group delete", assume that the name and timezone is the
	// same for all events in the group, even if an admin has modified one or more.
	// We don't want the emails to be overly complex.
	evQuery := `
		SELECT
			event_name,
			timezone,
			recurrence_group_id,
			recurrence_order
		FROM events
		WHERE event_id = $1
	`
	var evName, timezone string
	var recurGrpId sql.NullString
	var recurOrder sql.NullInt32
	err = s.DB.QueryRowContext(ctx, evQuery, eventInt).Scan(&evName, &timezone, &recurGrpId, &recurOrder)
	if err != nil {
		// If we didn't even get this far, we won't be able to delete the event(s).
		log.Printf("DB error: %v", err)
		return nil, friendlyDBError(err)
	}

	// The differences between getting the email info for one event v many is in the queries.
	var volQuery, leadQuery string
	args := []any{}
	if scope == nil || *scope == models.RecurrenceUpdateScopeThisOnly {
		// Only deleting this event.
		volQuery, leadQuery = getQueriesForSingleEvent()
		args = append(args, eventInt)
	} else {
		// If we are deleting recurring events, the UUID and order are required.
		if recurGrpId.Valid && recurOrder.Valid {
			volQuery, leadQuery = getQueriesForRecurringEvent()
			args = append(args, recurGrpId)
			args = append(args, recurOrder)
		} else {
			return nil, fmt.Errorf("RecurrenceID and RecurrenceOrder are required to delete this recurring event.")
		}
	}

	// Start with an empty map of shift times. We will add to it with each query.
	// Note: calling this "dbTimesMap", because the summaries will have datetimes
	// as they appear in the DB (RFC3339 format). Will format later.
	dbTimesMap := map[int]*ShiftSummary{}
	volMap, dbTimesMap, err := makeEmailMapForShifts(ctx, s.DB, volQuery, args, dbTimesMap)
	if err != nil {
		return nil, err
	}
	leadMap, dbTimesMap, err := makeEmailMapForShifts(ctx, s.DB, leadQuery, args, dbTimesMap)
	if err != nil {
		return nil, err
	}

	// Use the timezone (acquired above) to format the DB times.
	// Note - formatShiftTimes doesn't throw any erros. If there is a problem,
	// we get back the DB times, so we can still send an email.
	shiftsMap := formatShiftTimes(dbTimesMap, timezone)

	// Our maps now have a single entry for each volunteer and staff
	// lead contact. We also have the event name, and the formatted
	// dates/times for each shift. SEND the emails.
	sendDeleteEventEmailsForShifts(ctx, s.Mailer, volMap, leadMap, shiftsMap, evName)

	// Finally, delete the event(s) (which will cascade to the opportunities, shifts, and volunteer_shifts).
	if scope == nil || *scope == models.RecurrenceUpdateScopeThisOnly {
		_, err = s.DB.ExecContext(ctx, "DELETE FROM events WHERE event_id = $1", eventInt)
		if err != nil {
			log.Printf("DB error: %v", err)
			return nil, friendlyDBError(err)
		}
	} else {
		_, err = s.DB.ExecContext(ctx, "DELETE FROM events WHERE recurrence_group_id = $1::uuid AND recurrence_order >= $2", recurGrpId, recurOrder)
		if err != nil {
			log.Printf("DB error: %v", err)
			return nil, friendlyDBError(err)
		}
	}
	// Whether or not the scope was THIS_AND_FUTURE, we might not have any events left
	// with the UUID. If that's the case, get rid of the row from the table.
	if recurGrpId.Valid {
		delete := `
			DELETE FROM recurrence_groups
         	WHERE id = $1::uuid
				AND NOT EXISTS (
             		SELECT 1 FROM events WHERE recurrence_group_id = $1::uuid)
		`
		_, err = s.DB.ExecContext(ctx, delete, recurGrpId.String)
		if err != nil {
			// Non-fatal — log it but don't fail the delete
			log.Printf("warning: failed to clean up recurrence_groups row %s: %v", recurGrpId.String, err)
		}
	}

	// All good!

	return &models.MutationResult{
		Success: true,
		Message: ptrString("Successfully deleted event."),
		ID:      &eventId,
	}, nil
}

func (s *EventService) DeleteEventDate(ctx context.Context, evDateId string) (*models.MutationResult, error) {
	dateInt, err := strconv.Atoi(evDateId)
	if err != nil {
		return nil, fmt.Errorf("invalid event date id %s: %w", evDateId, err)
	}

	_, err = s.DB.ExecContext(ctx, "DELETE FROM event_dates WHERE event_date_id = $1", dateInt)

	if err != nil {
		return nil, fmt.Errorf("failed to delete event date: %w", err)
	}

	return &models.MutationResult{
		Success: true,
		Message: ptrString("Successfully deleted event date."),
		ID:      &evDateId,
	}, nil
}
