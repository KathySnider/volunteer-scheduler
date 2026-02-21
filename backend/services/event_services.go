package services

import (
	"context"
	"database/sql"
	"fmt"
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

// CreateEvent allows user to create a new event,
// complete with opportunities and shifts.
func (s *EventService) CreateEvent(ctx context.Context, newEvent *models.NewEventInput) (*models.InsertResult, error) {

	return &models.InsertResult{
		Success: false,
		Message: ptrString("Not yet Implemented."),
		ID:      nil,
	}, nil
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

	// Determine if the event is virtual, in person, or either (hybrid).
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
