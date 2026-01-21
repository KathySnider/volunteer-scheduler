package services

// This file provides the logic for services in volunteer-scheduler. 
// Some services can be called by both volunteers and admins. Others
// can be called by admins only. The calls come from the respective 
// resolvers.

import (
	"context"
	"volunteer-scheduler/graph/volunteer/generated"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
)

type SchedulerService struct {
	DB *sql.DB
}

// Function: CreateVolunteer 
// Returns the newly created Volunteer.
// Access: Admins only.
func (s *SchedulerService) CreateVolunteer(ctx context.Context, firstName string, lastName string) (*generated.Volunteer, error) {
		
	query := `
		INSERT INTO volunteers (first_name, last_name, created_at)
		VALUES ($1, $2, NOW())
		RETURNING volunteer_id
	`
	var volunteerID int
	err := s.DB.QueryRowContext(ctx, query, firstName, lastName).Scan(&volunteerID)
	if err != nil {
		return nil, fmt.Errorf("error creating volunteer: %w", err)
	}

	return &generated.Volunteer{
		ID:        fmt.Sprintf("%d", volunteerID),
		FirstName: firstName,
		LastName:  lastName,
	}, nil
}

// Function: AssignVolunteerToShift 
// Returns success or failure with error.
// Access: All users.
func (s *SchedulerService) AssignVolunteerToShift(ctx context.Context, shiftID string, volunteerID string) (*generated.AssignmentResult, error) {

	query := `
		INSERT INTO volunteer_shifts (volunteer_id, shift_id, assigned_at, status)
		VALUES ($1, $2, NOW(), 'confirmed')
		ON CONFLICT (volunteer_id, shift_id) DO NOTHING
	`

	_, err := s.DB.ExecContext(ctx, query, volunteerID, shiftID)
	if err != nil {
		return &generated.AssignmentResult{
			Success: false,
			Message: ptrString("Failed to assign volunteer to shift"),
		}, nil
	}

	return &generated.AssignmentResult{
		Success: true,
		Message: ptrString("Volunteer successfully assigned"),
	}, nil
}

// Function: CreateEvent
// Return the newly created Event.
// Access: Admins only.
func (s *SchedulerService) CreateEvent(ctx context.Context, input *generated.CreateEventInput) (*generated.Event, error) {
	panic(fmt.Errorf("not yet implemented: CreateEvent - createEvent"))
}

// Function: GetFilteredEvents 
// Return all events matching the criteria.
// Access: All users.
func (s *SchedulerService) GetFilteredEvents(ctx context.Context, filter *generated.EventFilter) ([]*generated.Event, error) {

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
    // Convert the filter from GraphQl fields to DB fields.
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

	// Filter by roles.
	if filter != nil && len(filter.Roles) > 0 {
		placeholders := []string{}
		for _, role := range filter.Roles {
			placeholders = append(placeholders, fmt.Sprintf("$%d", argCount))
			dbRole := strings.ToLower(string(role))
			args = append(args, dbRole)
			argCount++
		}
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
	// criteria. Process each event to see if there are volunteer
	// opportunities with open shifts that meet their criteria.
	eventsMap := make(map[string]*generated.Event)

	for rows.Next() {
		var e generated.Event
		var loc generated.Location
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
				e.Location = &loc
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

		// Filter by role.
		if filter != nil && len(filter.Roles) > 0 {
			placeholders := []string{}
			for _, role := range filter.Roles {
				placeholders = append(placeholders, fmt.Sprintf("$%d", argNum))
				dbRole := strings.ToLower(string(role))
				shiftArgs = append(shiftArgs, dbRole)
				argNum++
			}
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
			var shift generated.Shift
			var eventInt int
			var eventStr string
			var startTime, endTime time.Time
			var role string
			var shiftInt int

			err := shiftRows.Scan(
				&shiftInt,
				&startTime,
				&endTime,
				&role,
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

			// Convert role string to RoleType enum.
			shift.Role = generated.RoleType(strings.ToUpper(role))

			if event, exists := eventsMap[eventStr]; exists {
				event.Shifts = append(event.Shifts, &shift)
			}
		}
	}

	// Convert map to slice
	events := make([]*generated.Event, 0, len(eventsMap))
	for _, event := range eventsMap {
		events = append(events, event)
	}

	return events, nil
}

// Function: GetEventByID
// Returns the Event requested by the id.
// Access: All users.
func (s *SchedulerService) GetEventByID(ctx context.Context, id string) (*generated.Event, error) {

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

	var event generated.Event
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

	// Add location if it exists.
	if locationID != nil && address != nil && city != nil && state != nil {
		loc := &generated.Location{
			Name:    locationName,
			Address: *address,
			City:    *city,
			State:   *state,
			ZipCode: zipCode,
		}
		event.Location = loc
	}

	// Fetch opportunities for this event
	opportunities, err := r.getOpportunitiesForEvent(ctx, eventID)
	if err != nil {
		return nil, err
	}
	event.Opportunities = opportunities

	return &event, nil
}

// Function: GetQualifiedVolunteers 
// Returns volunteers that have the requested qualifications in their profile.
// Access: All users.
func (s *SchedulerService) GetQualifiedVolunteers(ctx context.Context, qualifications []string) ([]*generated.Volunteer, error) {
	var query string
	var args []interface{}

	if len(qualifications) > 0 {
		// Filter by qualifications
		query = `
			SELECT DISTINCT v.volunteer_id, v.first_name, v.last_name
			FROM volunteers v
			JOIN volunteer_qualifications vq ON v.volunteer_id = vq.volunteer_id
			WHERE vq.qualification = ANY($1)
		`
		args = append(args, pq.Array(qualifications))
	} else {
		// Get all volunteers
		query = `
			SELECT volunteer_id, first_name, last_name
			FROM volunteers
		`
	}

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("error querying volunteers: %w", err)
	}
	defer rows.Close()

	var volunteers []*generated.Volunteer
	for rows.Next() {
		var v generated.Volunteer
		var volunteerID int

		err := rows.Scan(&volunteerID, &v.FirstName, &v.LastName)
		if err != nil {
			return nil, fmt.Errorf("error scanning volunteer: %w", err)
		}

		v.ID = fmt.Sprintf("%d", volunteerID)
		volunteers = append(volunteers, &v)
	}

	return volunteers, nil
}

// Function: getOpportunitiesForEvent 
// Returns Opportunities associated with the Event (indicated by ID).
// Access: All users.
func (s *SchedulerService) getOpportunitiesForEvent(ctx context.Context, eventID int) ([]*generated.Opportunity, error) {

	oppQuery := `
		SELECT opportunity_id, role, opportunity_is_virtual
		FROM opportunities
		WHERE event_id = $1
	`

	rows, err := r.DB.QueryContext(ctx, oppQuery, eventID)
	if err != nil {
		return nil, fmt.Errorf("error querying opportunities: %w", err)
	}
	defer rows.Close()

	var opportunities []*generated.Opportunity
	for rows.Next() {
		var opp generated.Opportunity
		var oppID int
		var role string
		var isVirtual bool

		err := rows.Scan(&oppID, &role, &isVirtual)
		if err != nil {
			return nil, fmt.Errorf("error scanning opportunity: %w", err)
		}

		opp.ID = fmt.Sprintf("%d", oppID)
		opp.Role = generated.RoleType(strings.ToUpper(role))

		// Get required qualifications
		qualQuery := `
			SELECT required_qualification
			FROM opportunity_requirements
			WHERE opportunity_id = $1
		`
		qualRows, err := r.DB.QueryContext(ctx, qualQuery, oppID)
		if err == nil {
			var quals []string
			for qualRows.Next() {
				var qual string
				if err := qualRows.Scan(&qual); err == nil {
					quals = append(quals, qual)
				}
			}
			qualRows.Close()
			opp.RequiresQualifications = quals
		}

		// Get shifts for this opportunity
		shifts, err := r.getShiftsForOpportunity(ctx, oppID)
		if err != nil {
			return nil, err
		}
		opp.Shifts = shifts

		opportunities = append(opportunities, &opp)
	}

	return opportunities, nil
}

// Function: getShiftsForOpportunity 
// Returns Shifts associated with the Opportunity (indicated by ID).
// Access: All users.
func (s *SchedulerService) getShiftsForOpportunity(ctx context.Context, opportunityID int) ([]*generated.Shift, error) {

	shiftQuery := `
		SELECT shift_id, shift_start, shift_end, max_volunteers
		FROM shifts
		WHERE opportunity_id = $1
	`

	rows, err := r.DB.QueryContext(ctx, shiftQuery, opportunityID)
	if err != nil {
		return nil, fmt.Errorf("error querying shifts: %w", err)
	}
	defer rows.Close()

	var shifts []*generated.Shift
	for rows.Next() {
		var shift generated.Shift
		var shiftID int
		var startTime, endTime time.Time
		var maxVols *int

		err := rows.Scan(&shiftID, &startTime, &endTime, &maxVols)
		if err != nil {
			return nil, fmt.Errorf("error scanning shift: %w", err)
		}

		shift.ID = fmt.Sprintf("%d", shiftID)
		shift.Date = startTime.Format("2006-01-02")
		shift.StartTime = startTime.Format("15:04:05")
		shift.EndTime = endTime.Format("15:04:05")
		if maxVols != nil {
			shift.MaxVolunteers = maxVols
		}

		// Get assigned volunteers
		volQuery := `
			SELECT v.volunteer_id, v.first_name, v.last_name
			FROM volunteers v
			JOIN volunteer_shifts vs ON v.volunteer_id = vs.volunteer_id
			WHERE vs.shift_id = $1
		`
		volRows, err := r.DB.QueryContext(ctx, volQuery, shiftID)
		if err == nil {
			var assignedVols []*generated.Volunteer
			for volRows.Next() {
				var vol generated.Volunteer
				var volID int
				if err := volRows.Scan(&volID, &vol.FirstName, &vol.LastName); err == nil {
					vol.ID = fmt.Sprintf("%d", volID)
					assignedVols = append(assignedVols, &vol)
				}
			}
			volRows.Close()
			shift.AssignedVolunteers = assignedVols
		}

		shifts = append(shifts, &shift)
	}

	return shifts, nil
}

// Function: ptrString 
// Returns a pointer to a string (used for messages).
// Access: All users.
func ptrString(s string) *string {
	return &s
}
