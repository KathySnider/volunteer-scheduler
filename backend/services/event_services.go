package services

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"

	"volunteer-scheduler/models"
)

type EventService struct {
	DB           *sql.DB
	ShiftService *ShiftService
}

func NewEventService(db *sql.DB, shiftService *ShiftService) *EventService {
	return &EventService{
		DB:           db,
		ShiftService: shiftService,
	}
}

// GetEventByID retrieves a specific event
func (s *EventService) GetEventByID(ctx context.Context, id string) (*models.Event, error) {
	query := `
		SELECT 
			e.event_id,
			e.event_name,
			e.description,
			e.event_is_virtual,
			e.location_id,
			l.location_name,
			l.street_address,
			l.city,
			l.state,
			l.zip_code
		FROM events e
		LEFT JOIN locations l ON e.location_id = l.location_id
		WHERE e.event_id = $1
	`

	var event models.Event
	var eventID int
	var isVirtual bool
	var locationID *int
	var locationName, address, city, state, zipCode *string

	err := s.DB.QueryRowContext(ctx, query, id).Scan(
		&eventID,
		&event.Name,
		&event.Description,
		&isVirtual,
		&locationID,
		&locationName,
		&address,
		&city,
		&state,
		&zipCode,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("event not found")
	}
	if err != nil {
		return nil, fmt.Errorf("error querying event: %w", err)
	}

	event.ID = fmt.Sprintf("%d", eventID) // string for GraphQl

	// Determine event's type.
	if isVirtual && locationID != nil {
		event.EventType = "HYBRID"
	} else if isVirtual {
		event.EventType = "VIRTUAL"
	} else {
		event.EventType = "IN_PERSON"
	}

	// Add venue if it exists.
	if locationID != nil && address != nil && city != nil && state != nil {
		venue := &models.Venue{
			Name:    locationName,
			Address: *address,
			City:    *city,
			State:   *state,
			ZipCode: zipCode,
		}
		event.Venue = venue
	}

	// Fetch opportunities for this event.
	opportunities, err := s.GetOpportunitiesForEvent(ctx, eventID)
	if err != nil {
		return nil, err
	}
	event.Opportunities = opportunities

	return &event, nil
}

// GetFilteredEvents
// Retrieve events based on filter criteria.
func (s *EventService) GetFilteredEvents(ctx context.Context, filter *models.EventFilterInput) ([]*models.Event, error) {
	query := `
        SELECT DISTINCT
            e.event_id,
            e.event_name,
            e.description,
            e.event_is_virtual,
            e.location_id,
            l.location_name,
            l.street_address,
            l.city,
            l.state,
            l.zip_code
        FROM events e
        LEFT JOIN locations l ON e.location_id = l.location_id
        LEFT JOIN opportunities opp ON e.event_id = opp.event_id
        WHERE 1=1
    `

	args := []interface{}{}
	argCount := 1

	// Filter by cities.
	if filter != nil && len(filter.Cities) > 0 {
		placeholders := []string{}
		for _, city := range filter.Cities {
			placeholders = append(placeholders, fmt.Sprintf("$%d", argCount))
			args = append(args, city)
			argCount++
		}
		query += fmt.Sprintf(" AND l.city IN (%s)", strings.Join(placeholders, ","))
	}

	// Filter by event type.
	if filter != nil && filter.EventType != nil {
		switch *filter.EventType {
		case "VIRTUAL":
			query += " AND e.event_is_virtual = true AND e.location_id IS NULL"
		case "IN_PERSON":
			query += " AND e.event_is_virtual = false"
		case "HYBRID":
			query += " AND e.event_is_virtual = true AND e.location_id IS NOT NULL"
		}
	}

	// Filter by Jobs.
	if filter != nil && len(filter.Jobs) > 0 {
		placeholders := []string{}
		for _, job := range filter.Jobs {
			placeholders = append(placeholders, fmt.Sprintf("$%d", argCount))
			dbJob := strings.ToLower(string(job))
			args = append(args, dbJob)
			argCount++
		}
		// TODO: currently, in the DB, the job is called role. Fix name.
		query += fmt.Sprintf(" AND opp.role IN (%s)", strings.Join(placeholders, ","))
	}

	// Filter by date range.
	if filter != nil && (filter.StartDate != nil || filter.EndDate != nil) {
		query = strings.Replace(query, "WHERE 1=1",
			"LEFT JOIN opportunities opp2 ON e.event_id = opp2.event_id "+
				"LEFT JOIN shifts s_filter ON opp2.opportunity_id = s_filter.opportunity_id "+
				"WHERE 1=1", 1)

		if filter.StartDate != nil {
			query += fmt.Sprintf(" AND s_filter.shift_start >= $%d", argCount)
			args = append(args, *filter.StartDate)
			argCount++
		}
		if filter.EndDate != nil {
			query += fmt.Sprintf(" AND s_filter.shift_start <= $%d", argCount)
			args = append(args, *filter.EndDate)
			argCount++
		}
	}

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("error querying events: %w", err)
	}
	defer rows.Close()

	// Now we have selected the events that meet all of the user's
	// criteria. Process each one to see if there are volunteer
	// opportunities with open shifts that also meet their criteria.
	eventsMap := make(map[string]*models.Event)

	for rows.Next() {
		var e models.Event
		var loc models.Venue
		var locationID *int
		var locationName, address, city, state, zipCode *string

		var eventInt int
		var eventStr string
		var isVirtual bool // Temporary variable for database value

		err := rows.Scan(
			&eventInt,
			&e.Name,
			&e.Description,
			&isVirtual,
			&locationID,
			&locationName,
			&address,
			&city,
			&state,
			&zipCode,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning event: %w", err)
		}

		eventStr = fmt.Sprintf("%d", eventInt) // string for GraphQL.
		e.ID = eventStr

		// Determine if event is virtual, in person, or both.
		if isVirtual && locationID != nil {
			e.EventType = "HYBRID"
		} else if isVirtual {
			e.EventType = "VIRTUAL"
		} else {
			e.EventType = "IN_PERSON"
		}

		// Check if we've already processed this event.
		if _, exists := eventsMap[eventStr]; !exists {
			// Add location if it exists.
			if locationID != nil && address != nil && city != nil && state != nil {
				loc.Name = locationName
				loc.Address = *address
				loc.City = *city
				loc.State = *state
				loc.ZipCode = zipCode
				e.Venue = &loc
			}

			eventsMap[eventStr] = &e
		}
	}

	// Now fetch shifts for these events.
	if len(eventsMap) > 0 {
		eventIDs := []string{}
		for id := range eventsMap {
			eventIDs = append(eventIDs, id)
		}

		shiftsQuery := `
            SELECT
                s.shift_id,
                s.shift_start,
                s.shift_end,
                opp.role,
                opp.event_id
            FROM shifts s
            JOIN opportunities opp ON s.opportunity_id = opp.opportunity_id
            WHERE opp.event_id = ANY($1)
        `

		shiftArgs := []interface{}{pq.Array(eventIDs)}
		argNum := 2

		// Filter shifts by dates.
		if filter != nil && filter.StartDate != nil {
			shiftsQuery += fmt.Sprintf(" AND s.shift_start >= $%d", argNum)
			shiftArgs = append(shiftArgs, *filter.StartDate)
			argNum++
		}
		if filter != nil && filter.EndDate != nil {
			shiftsQuery += fmt.Sprintf(" AND s.shift_start <= $%d", argNum)
			shiftArgs = append(shiftArgs, *filter.EndDate)
			argNum++
		}

		// Filter by job.
		if filter != nil && len(filter.Jobs) > 0 {
			placeholders := []string{}
			for _, job := range filter.Jobs {
				placeholders = append(placeholders, fmt.Sprintf("$%d", argNum))
				dbJob := strings.ToLower(string(job))
				shiftArgs = append(shiftArgs, dbJob)
				argNum++
			}
			// TODO: change name of role!
			shiftsQuery += fmt.Sprintf(" AND opp.role IN (%s)", strings.Join(placeholders, ","))
		}

		shiftRows, err := s.DB.QueryContext(ctx, shiftsQuery, shiftArgs...)
		if err != nil {
			return nil, fmt.Errorf("error querying shifts: %w", err)
		}
		defer shiftRows.Close()

		// We are left with events that have open shifts that match all
		// criteria. Do some formatting.
		for shiftRows.Next() {
			var shift models.Shift
			var eventInt int
			var eventStr string
			var startTime, endTime time.Time
			var job string
			var shiftInt int

			err := shiftRows.Scan(
				&shiftInt,
				&startTime,
				&endTime,
				&job,
				&eventInt,
			)
			if err != nil {
				return nil, fmt.Errorf("error scanning shift: %w", err)
			}

			eventStr = fmt.Sprintf("%d", eventInt) // string for GraphQL.

			// Format the timestamps.
			shift.Date = startTime.Format("2006-01-02")
			shift.StartTime = startTime.Format("15:04:05")
			shift.EndTime = endTime.Format("15:04:05")

			// Convert role string to Job enum.
			shift.Job = models.Job(strings.ToUpper(job))

			if event, exists := eventsMap[eventStr]; exists {
				event.Shifts = append(event.Shifts, &shift)
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

// GetOpportunitiesForEvent
// Retrieve opportunities associated with the event.
func (s *EventService) GetOpportunitiesForEvent(ctx context.Context, eventID int) ([]*models.Opportunity, error) {
	oppQuery := `
		SELECT opportunity_id, role, opportunity_is_virtual
		FROM opportunities
		WHERE event_id = $1
	`
	rows, err := s.DB.QueryContext(ctx, oppQuery, eventID)
	if err != nil {
		return nil, fmt.Errorf("error querying opportunities: %w", err)
	}
	defer rows.Close()

	var opportunities []*models.Opportunity
	for rows.Next() {
		var opp models.Opportunity
		var oppID int
		var job string
		var isVirtual bool

		err := rows.Scan(&oppID, &job, &isVirtual)
		if err != nil {
			return nil, fmt.Errorf("error scanning opportunity: %w", err)
		}

		opp.ID = fmt.Sprintf("%d", oppID)

		// Get shifts for this opportunity
		shifts, err := s.ShiftService.GetShiftsForOpportunity(ctx, oppID)
		if err != nil {
			return nil, err
		}
		opp.Shifts = shifts

		opportunities = append(opportunities, &opp)
	}

	return opportunities, nil
}

// CreateEvent allows user to create a new event,
// complete with opportunities and shifts.
func (s *EventService) CreateEvent(ctx context.Context, newEvent *models.NewEventInput) (*models.InsertResult, error) {
	var query string
	var virtualEvent bool
	var venueIdPtr *int
	var eventId int
	var eventIdStr string

	// Determine whether or not the event will be virtual.
	// Both virtual and hybrid events have a virtual
	// component, so onlu in-person events are *not* vitual.
	virtualEvent = true
	if newEvent.EventType == models.EventTypeInPerson {
		virtualEvent = false
	}

	// Both in-person and hybrid events require a venue.
	if newEvent.EventType == models.EventTypeVirtual {
		// No venue for a virtual event.
		venueIdPtr = nil

	} else {
		// A venue is required. May already exist, else
		// create it.
		intPtr, err := s.UpsertVenue(ctx, newEvent.Venue)

		if err != nil {
			return nil, fmt.Errorf("error upserting venue: %w", err)
		}

		venueIdPtr = intPtr
	}

	// Create the new event, opportunities, and shifts inside
	// of a transaction. We don't want a partial event in the
	// DB, nor do we want volunteers to see incomplete events.

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

	if venueIdPtr == nil {
		query = `
		INSERT INTO events (name, description, is_virtual)
		VALUES ($1, $2, $3)
		RETURNING event_id
	`
		err = tx.QueryRowContext(ctx, query, newEvent.Name, newEvent.Description, virtualEvent).Scan(&eventId)

	} else {
		query = `
		INSERT INTO events (name, description, is_virtual, location_id)
		VALUES ($1, $2, $3, $4)
		RETURNING event_id
	`
		err = tx.QueryRowContext(ctx, query, newEvent.Name, newEvent.Description, virtualEvent, *venueIdPtr).Scan(&eventId)

	}

	if err != nil {
		// Save all of the information about what failed.
		err = fmt.Errorf("error inserting the event: %w", err)

	} else {
		// Event was inserted. Add the opportunites.
		err = s.AddOpportunitiesToEvent(ctx, tx, eventId, newEvent.Opportunities)
	}

	if err != nil {
		tx.Rollback()

		// NOW return an error ...
		return &models.InsertResult{
			Success: false,
			Message: ptrString("transaction failed and was rolled back."),
			ID:      nil,
		}, err
	}

	// All good. Commit and return the new event ID.
	tx.Commit()

	eventIdStr = strconv.Itoa(eventId)
	return &models.InsertResult{
		Success: true,
		Message: ptrString("Volunteer successfully created."),
		ID:      &eventIdStr,
	}, nil
}

// UpsertVenue
// Determines if the venue exists (as specified) in the DB.
// If so, returns the ID of the existing venue.
// Else, inserts the new venue and returns the new ID.
func (s *EventService) UpsertVenue(ctx context.Context, venue *models.VenueInput) (*int, error) {
	var query string
	var venueId int

	if venue == nil || venue.Name == nil {
		// Return an error. venue is required.
		err := fmt.Errorf("A venue is required for this type of event.")
		return nil, err
	}

	// (This is temporary. We'll actually have the ID
	// in the call if the location exists.) For now,
	// if the user indicated an existing venue, only
	// the name will have been supplied. That's how
	// we'll look it up.
	query = `
	SELECT 
		l.location_id, 
	FROM locations l
	WHERE l.location_name = $1 
	`

	err := s.DB.QueryRowContext(ctx, query, venue.Name).Scan(&venueId)

	if err == sql.ErrNoRows {
		// The venue is new. Create it now.
		venueIdPtr, err := s.CreateVenue(ctx, venue)

		if err != nil {
			return nil, fmt.Errorf("error creating location: %w", err)
		}

		venueId = *venueIdPtr

	} else if err != nil {
		return nil, fmt.Errorf("error querying location: %w", err)
	}

	// Success.
	return &venueId, nil
}

func (s *EventService) CreateVenue(ctx context.Context, venue *models.VenueInput) (*int, error) {
	var query string
	var locId int

	query = `
		INSERT INTO locations (location_name, address, city, state, zip_code)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING location_id
	`
	err := s.DB.QueryRowContext(ctx, query, venue.Name, venue.Address, venue.City, venue.State, venue.ZipCode).Scan(&locId)

	return &locId, err
}

func (s *EventService) AddOpportunitiesToEvent(ctx context.Context, tx *sql.Tx, eventId int, opps []*models.NewOpportunityInput) error {
	var i int

	for i = 0; i < len(opps); i++ {
		err := s.CreateOpportunity(ctx, tx, eventId, opps[i])

		if err != nil {
			err = fmt.Errorf("error inserting opp with index %v: %w", i, err)
			return err
		}
	}

	// No errors.
	return nil
}

func (s *EventService) CreateOpportunity(ctx context.Context, tx *sql.Tx, eventId int, opp *models.NewOpportunityInput) error {
	var query string
	var oppId int

	query = `
		INSERT INTO opportunities (event_id, role, other_role_description, opportunity_is_virtual, pre_event_instructions)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING opportunity_id
	`
	err := tx.QueryRowContext(ctx, query, eventId, opp.Job, nil, false, nil).Scan(&oppId)

	if err != nil {
		return err
	}

	// Opportunity was created. Add shifts.
	return s.AddShiftsToOpportunity(ctx, tx, oppId, opp.Shifts)

}

func (s *EventService) AddShiftsToOpportunity(ctx context.Context, tx *sql.Tx, oppId int, shifts []*models.NewShiftInput) error {
	var i int

	for i = 0; i < len(shifts); i++ {
		err := s.CreateShift(ctx, tx, oppId, shifts[i])

		if err != nil {
			err = fmt.Errorf("error inserting shift with index %v: %w", i, err)
			return err
		}
	}

	// No errors.
	return nil
}

func (s *EventService) CreateShift(ctx context.Context, tx *sql.Tx, oppId int, shift *models.NewShiftInput) error {
	var query string
	var shiftId int

	query = `
		INSERT INTO shifts (opportunity_id, shift_start, shift_end, staff_lead_id, max_volunteers)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING shift_id
	`
	err := tx.QueryRowContext(ctx, query, oppId, shift.StartTime, shift.EndTime, nil, shift.MaxVolunteers).Scan(&shiftId)

	return err
}
