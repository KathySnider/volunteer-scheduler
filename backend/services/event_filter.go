package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"volunteer-scheduler/models"

	"github.com/lib/pq"
)

// These filtering functions return 2 sets of data: the map of events, and a slice
// of ordered event ids. The callers must use the ordered slice (the second return
// value) to build the final result — ranging over the map directly would randomise
// the order, because Go maps have no guaranteed iteration order.

// ** Filtering for Managing Events **
func filterEvents(ctx context.Context, filter *models.EventFilterInput, db *sql.DB) (map[int]*models.Event, []int, error) {
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
			e.timezone,
			e.funding_entity_id,
			fe.name,
			e.recurrence_group_id,
			e.recurrence_order,
			rg.pattern,
			rg.max_occurrences,
			rg.weekday_ordinal,
			earliest.first_date
        FROM events e
        LEFT JOIN venues v ON e.venue_id = v.venue_id
        LEFT JOIN recurrence_groups rg ON rg.id = e.recurrence_group_id
		LEFT JOIN funding_entities fe ON e.funding_entity_id = fe.id
        LEFT JOIN opportunities opp ON e.event_id = opp.event_id
		LEFT JOIN job_types jt ON jt.job_type_id = opp.job_type_id
		LEFT JOIN (
			SELECT event_id, MIN(start_date_time) as first_date
			FROM event_dates
			GROUP BY event_id
		) earliest ON e.event_id = earliest.event_id
		WHERE 1=1
    `

	args := []any{}
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
				query += " AND earliest.first_date >= NOW()"
			case "PAST":
				query += " AND earliest.first_date < NOW()"
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
		var eventDesc, venueName, streetAddress, city, state, zip sql.NullString
		var fundingEntityId int
		var fundingEntityName string
		var recurGrpId, recurPattern, recurWdOrd sql.NullString
		var recurOrder, recurMax sql.NullInt32

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
			&e.Timezone,
			&fundingEntityId,
			&fundingEntityName,
			&recurGrpId,
			&recurOrder,
			&recurPattern,
			&recurMax,
			&recurWdOrd,
			&firstDate,
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
			// Since venue is present, the other fields must also be
			// not null - they are NOT NULL in DB. The exceptions are
			// name and zip.
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

		e.FundingEntity = models.FundingEntity{ID: fundingEntityId, Name: fundingEntityName}

		if recurGrpId.Valid {
			rg := &models.RecurrenceGroup{GroupID: recurGrpId.String}
			e.RecurrenceGroup = rg
			if recurOrder.Valid {
				order := int(recurOrder.Int32)
				e.RecurrenceOrder = &order
			} else {
				return nil, nil, fmt.Errorf("recurrence_order is required when recurrence_group_id is not null.")
			}
			if recurPattern.Valid {
				rg.Pattern = recurPattern.String
			}
			if recurMax.Valid {
				max := int(recurMax.Int32)
				rg.MaxOccurrences = &max
			}
			if recurWdOrd.Valid {
				rg.WeekdayOrdinal = &recurWdOrd.String
			}
		}

		stPtrs, err := FetchEventServiceTypes(ctx, db, eventInt)
		if err != nil {
			return nil, nil, fmt.Errorf("error getting event's service types: %w", err)
		}
		e.ServiceTypes = make([]string, len(stPtrs))
		for i := range stPtrs {
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

// ** Filtering Volunteer Events **
// Figuring out all of the event filtering stuff is messy, but worth it. The users can
// get the events they want to see. We currently filter on:
//   - cities OR distance (if the user has provided a zipcode),
//   - event type (virtual, in-person, or hybrid),
//   - jobs, and
//   - timeframe (past, upcoming, all).
//
// We use a 2-pass strategy. This function handles the first pass. This pass
// returns both the map of events (keyed by event_id) and the slice of event
// IDs in the order they came back from the DB (ORDER BY earliest event date ASC).
func fetchFilteredPassOne(ctx context.Context, filter *models.VolunteerEventFilterInput, db *sql.DB, volId int) (map[int]*models.EventView, []int, error) {

	// If distance is used, will need the volunteer's lat/lng.
	// Since there is no way to join a volunteer to any of this,
	// do one quick lookup for lat/lng.
	var volLat, volLng *float64
	var zip sql.NullString

	if filter != nil && filter.Distance != nil {
		var lat, lng sql.NullFloat64

		err := db.QueryRowContext(ctx, "SELECT zip_code, latitude, longitude FROM volunteers where volunteer_id = $1", volId).Scan(&zip, &lat, &lng)
		if err != nil {
			return nil, nil, fmt.Errorf("error getting volunteer's latitude and longitude with distance filter: %w", err)
		}
		if lat.Valid && lng.Valid {
			volLat = &lat.Float64
			volLng = &lng.Float64
		} else {
			if zip.Valid {
				// User has entered a zip, but we weren't able
				// to get geo info. Try again.
				volLat, volLng, err = GeocodeZip(zip.String)
				if err == nil {
					if volLat != nil && volLng != nil {
						// Since we were able to get the geo info, put it in the profile.
						db.ExecContext(ctx, "UPDATE volunteers SET latitude = $1, longitude = $2 WHERE volunteer_id = $3", volLat, volLng, volId)
					}
				} else {
					log.Printf("unable to get geo info for zip %v: %v", zip.String, err)
				}
			}
		}
	}

	// This first assignment to the query var has no filtering, but that will get added below.
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
			v.latitude,
			v.longitude,
			e.timezone,
			earliest.first_date
        FROM events e
        LEFT JOIN venues v ON e.venue_id = v.venue_id
        LEFT JOIN opportunities opp ON e.event_id = opp.event_id
		LEFT JOIN job_types jt ON jt.job_type_id = opp.job_type_id
		LEFT JOIN (
			SELECT event_id, MIN(start_date_time) as first_date
			FROM event_dates
			GROUP BY event_id
		) earliest ON e.event_id = earliest.event_id
		WHERE 1=1
    `

	args := []interface{}{}
	argCount := 1

	// Add the filtering stuff to the query.
	if filter != nil {

		// Filter by cities.
		if filter.Distance == nil && len(filter.Cities) > 0 {

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
				query += " AND earliest.first_date >= NOW()"
			case "PAST":
				query += " AND earliest.first_date < NOW()"
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
	eventsMap := make(map[int]*models.EventView)

	// orderedIDs preserves the ORDER BY from the SQL query so the caller
	// can reassemble the slice in the correct order after map operations.
	orderedIDs := make([]int, 0)

	for rows.Next() {
		var e models.EventView
		var eventInt int
		var venueInt sql.NullInt64
		var isVirtual bool
		var firstDate *time.Time
		var eventDesc, venueName, streetAddress, city, state, zip sql.NullString
		var vLat, vLng sql.NullFloat64

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
			&vLat,
			&vLng,
			&e.Timezone,
			&firstDate,
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
			var venue models.VenueView
			// Since venue ID is present, the other fields must also be
			// not null - they are NOT NULL in DB. The exceptions are
			// name and zip.
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

		// Get distance if user wants it and all values are "go".
		// Note: virtual events are always "within the distance" unless the user has
		// filtered them out above. Also events with no lat/lng information.
		var venLat, venLng *float64

		if !isVirtual && filter != nil && filter.Distance != nil && e.Venue != nil && volLat != nil && volLng != nil {
			// Determine if this venue is within the distance filter. First, make sure
			// we have latitude and longitude for this venue.
			if vLat.Valid && vLng.Valid {
				venLat = &vLat.Float64
				venLng = &vLng.Float64
			} else {
				// We may have failed to get geo info when we created the venue.
				// Try again.
				vzip := ""
				if e.Venue.ZipCode != nil {
					vzip = *e.Venue.ZipCode
				}
				venLat, venLng, err = GeocodeAddress(e.Venue.Address, e.Venue.City, e.Venue.State, vzip)
				if err == nil {
					if venLat != nil && venLng != nil {
						// Since we were now able to get the geo info, put it in the venues table.
						// Don't test for errors on this; if it fails we're no worse off than before.
						db.ExecContext(ctx, "UPDATE venues SET latitude = $1, longitude = $2 WHERE venue_id = $3", venLat, venLng, venueInt.Int64)
					}
				} else {
					log.Printf("unable to get geo info for venue %v: %v", e.Venue.Address, err)
					// Since there was an error, don't count on geocoder: make sure both lat and lng are still nil.
					venLat = nil
					venLng = nil
				}
			}

			if venLat != nil && venLng != nil {
				dist := fetchDistance(*volLat, *volLng, *venLat, *venLng)
				if dist > float64(*filter.Distance) {
					// This event is outside of the specified boundaries. Jump to next row.
					continue
				}
			}
		}

		stPtrs, err := FetchEventServiceTypes(ctx, db, eventInt)
		if err != nil {
			return nil, nil, fmt.Errorf("error getting event's service types: %w", err)
		}
		e.ServiceTypes = make([]string, len(stPtrs))
		for i := range stPtrs {
			e.ServiceTypes[i] = *stPtrs[i]
		}

		dates, err := fetchEventDateViews(ctx, db, eventInt)
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

// In pass 2, we just make sure each event has at least one shift. If there was no filter,
// or if the filter didn't include jobs, pass one might have returned events that have been
// created but do not yet have anything for which a volunteer can sign up.
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
