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
// get the events they want to see. We currently filter on region, event type (virtual,
// in-person, or hybrid), jobs, and dates.
// We use a 2-pass strategy. This function handles the first pass.
func fetchFilteredPassOne(ctx context.Context, filter *models.EventFilterInput, db *sql.DB) (map[int]*models.Event, error) {
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
			for _, jobId := range filter.Jobs {
				placeholders = append(placeholders, fmt.Sprintf("$%d", argCount))
				args = append(args, jobId)
				argCount++
			}
			query += fmt.Sprintf(" AND opp.job_type_id IN (%s)", strings.Join(placeholders, ","))
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

		stPtrs, err := FetchEventServiceTypes(ctx, db, eventInt)
		if err != nil {
			return nil, fmt.Errorf("error getting event's service types: %w", err)
		}
		e.ServiceTypes = make([]string, len(stPtrs))
		for i := 0; i < len(stPtrs); i++ {
			e.ServiceTypes[i] = *stPtrs[i]
		}

		dates, err := FetchEventDates(ctx, db, eventInt)
		if err != nil {
			return nil, fmt.Errorf("error getting event's dates: %w", err)
		}
		e.EventDates = dates

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
// have an opportunity that matched the job filter, but its shift crosses the filter's end date.
// Rare, but possible.
func fetchFilteredPassTwo(ctx context.Context, filter *models.EventFilterInput, idList []int, DB *sql.DB) (map[int]bool, error) {

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
			for _, jobId := range filter.Jobs {
				placeholders = append(placeholders, fmt.Sprintf("$%d", argNum))
				shiftArgs = append(shiftArgs, jobId)
				argNum++
			}

			shiftsQuery += fmt.Sprintf(" AND opp.job_type_id IN (%s)", strings.Join(placeholders, ","))
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
