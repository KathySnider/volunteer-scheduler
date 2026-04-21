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

// services/event_filter.go

// ** Filtering Events **
// Figuring out all of the event filtering stuff is messy, but worth it. The users can
// get the events they want to see. We currently filter on cities, event types (virtual,
// in-person, or hybrid), jobs, and timeframe (past, upcoming, all).
// We use a 2-pass strategy. This function handles the first pass.
// fetchFilteredPassOne returns both a map of events (keyed by event_id) and a
// slice of event IDs in the order they came back from the DB (ORDER BY earliest
// event date ASC). The caller must use the ordered slice to build the final
// result — ranging over the map directly would randomise the order because Go
// maps have no guaranteed iteration order.
func fetchFilteredPassOne(ctx context.Context, filter *models.EventFilterInput, db *sql.DB) (map[int]*models.Event, []int, error) {
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
			earliest.first_date,
			e.funding_entity_id,
			fe.name
        FROM events e
        LEFT JOIN venues v ON e.venue_id = v.venue_id
        LEFT JOIN funding_entities fe ON e.funding_entity_id = fe.id
        LEFT JOIN opportunities opp ON e.event_id = opp.event_id
		LEFT JOIN job_types jt ON jt.job_type_id = opp.job_type_id
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
			for _, jobId := range filter.Jobs {
				placeholders = append(placeholders, fmt.Sprintf("$%d", argCount))
				args = append(args, jobId)
				argCount++
			}
			query += fmt.Sprintf(" AND opp.job_type_id IN (%s)", strings.Join(placeholders, ","))
		}

		// Filter by TimeFrame.
		if filter.TimeFrame != nil {
			switch *filter.TimeFrame {
			case "UPCOMING":
				query += " AND s_filter.shift_start >= NOW()"
			case "PAST":
				query += " AND s_filter.shift_start < NOW()"
			case "ALL":
				// NO filter needed
			}
		}
	}

	// Get the events in order of start date.
	query += " ORDER BY earliest.first_date ASC NULLS LAST"

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("error querying in pass 1: %w", err)
	}
	defer rows.Close()

	// Each row represents an event that *might* meet the
	// criteria. Turn each row into an event.
	eventsMap := make(map[int]*models.Event)
	// orderedIDs preserves the ORDER BY from the SQL query so the caller
	// can reassemble the slice in the correct order after map operations.
	orderedIDs := make([]int, 0)

	for rows.Next() {
		var e models.Event
		var eventInt int
		var venueInt sql.NullInt64
		var isVirtual bool
		var firstDate *time.Time
		var eventDesc, venueName, streetAddress, city, state, zip, timezone sql.NullString
		var fundingEntityId int
		var fundingEntityName string

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
			&fundingEntityId,
			&fundingEntityName,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("error scanning event: %w", err)
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

		e.FundingEntity = models.FundingEntity{ID: fundingEntityId, Name: fundingEntityName}

		stPtrs, err := FetchEventServiceTypes(ctx, db, eventInt)
		if err != nil {
			return nil, nil, fmt.Errorf("error getting event's service types: %w", err)
		}
		e.ServiceTypes = make([]string, len(stPtrs))
		for i := 0; i < len(stPtrs); i++ {
			e.ServiceTypes[i] = *stPtrs[i]
		}

		dates, err := FetchEventDates(ctx, db, eventInt)
		if err != nil {
			return nil, nil, fmt.Errorf("error getting event's dates: %w", err)
		}
		e.EventDates = dates

		summaries, err := FetchEventShiftSummaries(ctx, db, eventInt)
		if err != nil {
			return nil, nil, fmt.Errorf("error getting event's shift summaries: %w", err)
		}
		e.ShiftSummaries = summaries

		// Have we already processed this event?
		_, exists := eventsMap[eventInt]
		if !exists {
			eventsMap[eventInt] = &e
			// Record this ID the first time we see it to preserve the DB's ORDER BY.
			orderedIDs = append(orderedIDs, eventInt)
		}
	}

	return eventsMap, orderedIDs, nil
}

// FetchEventShiftSummaries returns per-opportunity volunteer counts for a single event.
// Each row represents one opportunity (job type) and aggregates across all of its shifts.
func FetchEventShiftSummaries(ctx context.Context, db *sql.DB, eventID int) ([]*models.EventShiftSummary, error) {
	query := `
		SELECT
			jt.name,
			COALESCE(SUM(
				(SELECT COUNT(*) FROM volunteer_shifts vs WHERE vs.shift_id = s.shift_id AND vs.cancelled_at IS NULL)
			), 0) AS assigned,
			COALESCE(SUM(s.max_volunteers), 0) AS max_vol
		FROM opportunities opp
		JOIN job_types jt ON jt.job_type_id = opp.job_type_id
		LEFT JOIN shifts s ON s.opportunity_id = opp.opportunity_id
		WHERE opp.event_id = $1
		GROUP BY opp.opportunity_id, jt.name
		ORDER BY jt.name
	`

	rows, err := db.QueryContext(ctx, query, eventID)
	if err != nil {
		return nil, fmt.Errorf("error querying shift summaries: %w", err)
	}
	defer rows.Close()

	summaries := make([]*models.EventShiftSummary, 0)
	for rows.Next() {
		var s models.EventShiftSummary
		if err := rows.Scan(&s.JobName, &s.AssignedVolunteers, &s.MaxVolunteers); err != nil {
			return nil, fmt.Errorf("error scanning shift summary: %w", err)
		}
		summaries = append(summaries, &s)
	}
	return summaries, nil
}

// Now, just make sure each event has at least one shift. If there was no filter,
// or if the filter didn't include jobs, pass one might have returned events that
// have been created but do not yet have anything for which a volunteer can sign up.
func fetchFilteredPassTwo(ctx context.Context, DB *sql.DB, idList []int) (map[int]bool, error) {

	eventsWithShifts := make(map[int]bool, len(idList))

	shiftsQuery := `
		SELECT
			opp.event_id,
			COUNT(*) 
		FROM shifts s
		JOIN opportunities opp ON s.opportunity_id = opp.opportunity_id
		WHERE opp.event_id = ANY($1)
		GROUP BY opp.event_id
	`

	shiftArgs := []interface{}{pq.Array(idList)}

	shiftRows, err := DB.QueryContext(ctx, shiftsQuery, shiftArgs...)
	if err != nil {
		return nil, fmt.Errorf("error querying shifts: %w", err)
	}
	defer shiftRows.Close()

	for shiftRows.Next() {
		var eInt int
		var count int

		err := shiftRows.Scan(&eInt, &count)
		if err != nil {
			return nil, fmt.Errorf("error scanning shift: %w", err)
		}

		// With the query as is, we don't expect rows with 0, but will check
		// in case the query ever changes.
		if count > 0 {
			eventsWithShifts[eInt] = true
		}
	}

	return eventsWithShifts, nil
}
