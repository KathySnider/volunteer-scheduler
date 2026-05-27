package services

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"
	"volunteer-scheduler/models"

	"github.com/google/uuid"
)

type ShiftService struct {
	DB     *sql.DB
	mailer *Mailer
}

func NewShiftService(db *sql.DB, mailer *Mailer) *ShiftService {
	return &ShiftService{
		DB:     db,
		mailer: mailer,
	}
}

// Queries.

func (s *ShiftService) FetchActiveJobs(ctx context.Context) ([]*models.JobType, error) {
	query := `
        SELECT
			job_type_id,
            code,
            name,
			sort_order
        FROM job_types
		WHERE is_active = true
		ORDER BY sort_order
    `
	rows, err := s.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error querying jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*models.JobType
	for rows.Next() {
		var job models.JobType
		err := rows.Scan(
			&job.ID,
			&job.Code,
			&job.Name,
			&job.SortOrder,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning job types: %w", err)
		}

		jobs = append(jobs, &job)
	}

	return jobs, nil
}

// FetchOpportunitiesForEvent
// Fetch opportunities associated with the event.
func (s *ShiftService) FetchOpportunitiesForEvent(ctx context.Context, eventId string) ([]*models.Opportunity, error) {

	eventInt, err := strconv.Atoi(eventId)
	if err != nil {
		return nil, fmt.Errorf("event id is not valid: %s", err)
	}

	query := `
		SELECT
		  opportunity_id,
		  job_type_id,
		  opportunity_is_virtual,
		  pre_event_instructions
		FROM opportunities
		WHERE event_id = $1
	`
	rows, err := s.DB.QueryContext(ctx, query, eventInt)
	if err != nil {
		return nil, fmt.Errorf("error querying opportunities: %w", err)
	}
	defer rows.Close()

	var opportunities []*models.Opportunity

	for rows.Next() {
		var opp models.Opportunity
		var oppInt int
		var instruct sql.NullString

		err := rows.Scan(
			&oppInt,
			&opp.JobId,
			&opp.IsVirtual,
			&instruct)
		if err != nil {
			return nil, fmt.Errorf("error scanning opportunity: %w", err)
		}

		opp.ID = strconv.Itoa(oppInt)
		if instruct.Valid {
			opp.PreEventInstructions = &instruct.String
		} else {
			opp.PreEventInstructions = nil
		}
		// Fetch shifts for this opportunity
		shifts, err := s.FetchShiftsForOpportunity(ctx, opp.ID)
		if err != nil {
			return nil, fmt.Errorf("error fetching shifts for opportunity %d: %w", oppInt, err)
		}
		opp.Shifts = shifts

		opportunities = append(opportunities, &opp)
	}

	return opportunities, nil
}

// FetchShiftsForOpportunity
// Retrieve shifts associated with the opportunity specified. This
// includes all of the fields, so an admin can edit it.
func (s *ShiftService) FetchShiftsForOpportunity(ctx context.Context, oppId string) ([]*models.Shift, error) {

	oppInt, err := strconv.Atoi(oppId)
	if err != nil {
		return nil, fmt.Errorf("opportunity id is not valid: %w", err)
	}

	query := `
		SELECT
		  shift_id,
		  shift_start,
		  shift_end,
		  max_volunteers,
		  staff_contact_id
		FROM shifts
	WHERE opportunity_id = $1
	`

	rows, err := s.DB.QueryContext(ctx, query, oppInt)
	if err != nil {
		return nil, fmt.Errorf("error querying shifts: %w", err)
	}
	defer rows.Close()

	var shifts []*models.Shift
	for rows.Next() {
		var shift models.Shift
		var maxVols, staffId sql.NullInt64

		err := rows.Scan(&shift.ID, &shift.StartDateTime, &shift.EndDateTime, &maxVols, &staffId)
		if err != nil {
			return nil, fmt.Errorf("error scanning shift: %w", err)
		}

		// Max volunteers and staff Id are nullable.
		if maxVols.Valid {
			maxInt := int(maxVols.Int64)
			shift.MaxVolunteers = &maxInt
		} else {
			shift.MaxVolunteers = nil
		}

		if staffId.Valid {
			staffStr := strconv.FormatInt(staffId.Int64, 10) // convert to string since StaffContactId is *string
			shift.StaffContactId = &staffStr
		} else {
			shift.StaffContactId = nil
		}

		shifts = append(shifts, &shift)
	}

	return shifts, nil
}

// This function gets the shifts in the "flattened" view for the
// volunteers. Each shift includes the job name, even though the
// job is really part of the opportunity that includes these
// shifts.
// Including the name in each shift view makes it easier for
// volunteers to understand what they are signing up for.
func (s *ShiftService) FetchShiftViewsForEvent(ctx context.Context, eventId string) ([]*models.ShiftView, error) {

	eventInt, err := strconv.Atoi(eventId)
	if err != nil {
		return nil, fmt.Errorf("event id is not valid: %w", err)
	}

	query := `
		SELECT
		o.opportunity_id,
		s.shift_id,
		jt.name,
		s.shift_start,
		s.shift_end,
		s.max_volunteers,
		o.opportunity_is_virtual
	FROM shifts s
	LEFT JOIN opportunities o ON s.opportunity_id = o.opportunity_id
	LEFT JOIN job_types jt ON jt.job_type_id = o.job_type_id
	WHERE o.event_id = $1
	ORDER by o.opportunity_id, s.shift_id
	`
	rows, err := s.DB.QueryContext(ctx, query, eventInt)
	if err != nil {
		return nil, fmt.Errorf("error querying opportunities: %w", err)
	}
	defer rows.Close()

	var shifts []*models.ShiftView

	for rows.Next() {
		var shift models.ShiftView
		var shiftInt, oppInt int
		var maxVols sql.NullInt64

		err := rows.Scan(
			&oppInt,
			&shiftInt,
			&shift.JobName,
			&shift.StartDateTime,
			&shift.EndDateTime,
			&maxVols,
			&shift.IsVirtual)
		if err != nil {
			return nil, fmt.Errorf("error scanning opportunity: %w", err)
		}

		shift.ID = strconv.Itoa(shiftInt)
		if maxVols.Valid {
			maxInt := int(maxVols.Int64)
			shift.MaxVolunteers = &maxInt
		} else {
			shift.MaxVolunteers = nil
		}

		assignedVols, err := FetchAssignedVolunteersForShift(ctx, shiftInt, s.DB)
		if err != nil {
			return nil, fmt.Errorf("error getting assigned volunteers for shift: %w", err)
		}
		// Here, we just need the number of volunteers already
		// assigned to the shift.
		shift.AssignedVolunteers = len(assignedVols)

		shifts = append(shifts, &shift)
	}

	return shifts, nil
}

// These 2 functions get *all* of the information for each shift for a volunteer.
// Only qualification is that it might include only upcoming shifts (>= NOW()),
// only past shifts (< NOW()) or all shifts ever.
func (s *ShiftService) FetchOwnShifts(ctx context.Context, volId int, filter models.ShiftsTimeFilter) ([]*models.VolunteerShift, error) {
	return fetchVolunteerShifts(ctx, s.DB, volId, filter)
}

func (s *ShiftService) FetchVolunteerShifts(ctx context.Context, volId string, filter models.ShiftsTimeFilter) ([]*models.VolunteerShift, error) {
	volInt, err := strconv.Atoi(volId)
	if err != nil {
		return nil, fmt.Errorf("volunteer id is not valid: %w", err)
	}
	return fetchVolunteerShifts(ctx, s.DB, volInt, filter)
}

// ============================================================================
// Mutations: Create opportunities and shifts.
// ============================================================================

func (s *ShiftService) CreateJobType(ctx context.Context, newJob models.NewJobTypeInput) (*models.MutationResult, error) {

	query := `
		INSERT INTO job_types (
			code,
			name,
			sort_order)
		VALUES ($1, $2, $3)
		RETURNING job_type_id
	`

	var JobId int
	err := s.DB.QueryRowContext(ctx, query, newJob.Code, newJob.Name, newJob.SortOrder).Scan(&JobId)

	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to create new Job."),
			ID:      nil,
		}, err
	}

	return &models.MutationResult{
		Success: true,
		Message: ptrString("Job successfully created."),
		ID:      ptrString(strconv.Itoa(JobId)),
	}, nil

}

// CreateOpportunity inserts a new opportunity (with its initial shifts) for an
// event.  If the event belongs to a recurrence group the opportunity and its
// shifts are also propagated to every future instance in the group; all copies
// share a recurrence_template_id UUID so later edits/deletes also propagate.
func (s *ShiftService) CreateOpportunity(ctx context.Context, opp models.NewOpportunityInput) (*models.MutationResult, error) {
	eventInt, err := strconv.Atoi(opp.EventId)
	if err != nil {
		return nil, fmt.Errorf("invalid event id: %w", err)
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert the base opportunity.
	var oppInt int
	err = tx.QueryRowContext(ctx, `
		INSERT INTO opportunities (event_id, job_type_id, opportunity_is_virtual, pre_event_instructions)
		VALUES ($1, $2, $3, $4)
		RETURNING opportunity_id`,
		opp.EventId, opp.JobId, opp.IsVirtual, opp.PreEventInstructions,
	).Scan(&oppInt)
	if err != nil {
		return &models.MutationResult{Success: false, Message: ptrString("error creating opportunity")}, err
	}

	// Add initial shifts to the base opportunity.
	if err = addNewOpportunityShifts(ctx, opp.Shifts, oppInt, tx); err != nil {
		return &models.MutationResult{Success: false, Message: ptrString("error adding shifts to opportunity")}, err
	}

	// Check whether the event belongs to a recurrence group.
	groupID, order, err := eventGroupAndOrder(ctx, tx, eventInt)
	if err != nil {
		return nil, fmt.Errorf("error checking recurrence group: %w", err)
	}

	if groupID != "" {
		// Generate a shared template UUID and stamp the base opportunity.
		tmplID := uuid.New().String()
		if _, err = tx.ExecContext(ctx,
			`UPDATE opportunities SET recurrence_template_id = $1::uuid WHERE opportunity_id = $2`,
			tmplID, oppInt,
		); err != nil {
			return nil, fmt.Errorf("error setting opp template id: %w", err)
		}

		// Stamp each initial shift with its own recurrence_template_id so that
		// future edits to those shifts also fan out to sibling events.
		// Fetch the IDs in insertion order (shift_id is a serial, so ORDER BY shift_id
		// matches the order addNewOpportunityShifts inserted them).
		shiftIDRows, err := tx.QueryContext(ctx,
			`SELECT shift_id FROM shifts WHERE opportunity_id = $1 ORDER BY shift_id`, oppInt)
		if err != nil {
			return nil, fmt.Errorf("fetching base shift ids: %w", err)
		}
		var baseShiftIDs []int
		for shiftIDRows.Next() {
			var id int
			if err := shiftIDRows.Scan(&id); err != nil {
				shiftIDRows.Close()
				return nil, fmt.Errorf("scanning base shift id: %w", err)
			}
			baseShiftIDs = append(baseShiftIDs, id)
		}
		shiftIDRows.Close()
		if err := shiftIDRows.Err(); err != nil {
			return nil, fmt.Errorf("iterating base shift ids: %w", err)
		}

		// One template UUID per initial shift.
		shiftTmplIDs := make([]string, len(baseShiftIDs))
		for i, shiftID := range baseShiftIDs {
			shiftTmplIDs[i] = uuid.New().String()
			if _, err = tx.ExecContext(ctx,
				`UPDATE shifts SET recurrence_template_id = $1::uuid WHERE shift_id = $2`,
				shiftTmplIDs[i], shiftID,
			); err != nil {
				return nil, fmt.Errorf("stamping base shift %d: %w", shiftID, err)
			}
		}

		// Fetch the event timezone so we can convert the input shift times to UTC.
		var timezone string
		if err = tx.QueryRowContext(ctx,
			`SELECT COALESCE(timezone, '') FROM events WHERE event_id = $1`, eventInt,
		).Scan(&timezone); err != nil {
			return nil, fmt.Errorf("error fetching event timezone: %w", err)
		}

		// Get the source event's first date for offset calculations.
		srcFirstDate, err := eventFirstDate(ctx, tx, eventInt)
		if err != nil {
			return nil, fmt.Errorf("error fetching source event first date: %w", err)
		}

		// Convert each input shift's local times to UTC time.Time values.
		type shiftUTC struct {
			start, end    time.Time
			staffID, maxV interface{}
		}
		var utcShifts []shiftUTC
		for _, sh := range opp.Shifts {
			startStr, err := DateTimeToUTC(sh.StartDateTime, timezone)
			if err != nil {
				return nil, fmt.Errorf("error converting shift start time: %w", err)
			}
			endStr, err := DateTimeToUTC(sh.EndDateTime, timezone)
			if err != nil {
				return nil, fmt.Errorf("error converting shift end time: %w", err)
			}
			start, _ := time.Parse(time.RFC3339, *startStr)
			end, _ := time.Parse(time.RFC3339, *endStr)
			var staffID, maxV interface{}
			if sh.StaffContactId != nil {
				staffID = *sh.StaffContactId
			}
			if sh.MaxVolunteers != nil {
				maxV = *sh.MaxVolunteers
			}
			utcShifts = append(utcShifts, shiftUTC{start, end, staffID, maxV})
		}

		// Get future peer events.
		peers, err := futurePeerEvents(ctx, tx, groupID, order)
		if err != nil {
			return nil, fmt.Errorf("error fetching peer events: %w", err)
		}

		// Propagate the opportunity and its shifts to each future instance.
		// Sibling shifts are inserted with the same per-shift template UUIDs so
		// edits and deletes on any occurrence propagate correctly.
		for _, peer := range peers {
			var sibOppID int
			if err = tx.QueryRowContext(ctx, `
				INSERT INTO opportunities
				  (event_id, job_type_id, opportunity_is_virtual, pre_event_instructions, recurrence_template_id)
				VALUES ($1, $2, $3, $4, $5::uuid)
				RETURNING opportunity_id`,
				peer.EventID, opp.JobId, opp.IsVirtual, opp.PreEventInstructions, tmplID,
			).Scan(&sibOppID); err != nil {
				return nil, fmt.Errorf("error creating sibling opp for event %d: %w", peer.EventID, err)
			}

			for i, sh := range utcShifts {
				newStart, newEnd := adjustTimes(sh.start, sh.end, srcFirstDate, peer.FirstDate)
				if _, err = tx.ExecContext(ctx, `
					INSERT INTO shifts
					  (opportunity_id, shift_start, shift_end, staff_contact_id, max_volunteers, recurrence_template_id)
					VALUES ($1, $2, $3, $4, $5, $6::uuid)`,
					sibOppID, newStart, newEnd, sh.staffID, sh.maxV, shiftTmplIDs[i],
				); err != nil {
					return nil, fmt.Errorf("error creating sibling shift for event %d: %w", peer.EventID, err)
				}
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return &models.MutationResult{Success: false, Message: ptrString("error committing transaction")}, err
	}

	oppStr := strconv.Itoa(oppInt)
	return &models.MutationResult{
		Success: true,
		Message: ptrString("opportunity successfully created"),
		ID:      &oppStr,
	}, nil
}

// CreateShift adds a standalone shift to an existing opportunity.
// If the opportunity belongs to a recurrence group the shift is also
// propagated to the matching opportunity on every future instance, with
// start/end times adjusted to preserve the same offset from each event's
// first scheduled date.
func (s *ShiftService) CreateShift(ctx context.Context, shift models.AddShiftInput) (*models.MutationResult, error) {
	oppInt, err := strconv.Atoi(shift.OppId)
	if err != nil {
		return nil, fmt.Errorf("invalid opportunity id: %w", err)
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Fetch the timezone for UTC conversion.
	var timezone string
	err = tx.QueryRowContext(ctx, `
		SELECT e.timezone
		FROM events e
		JOIN opportunities opp ON opp.event_id = e.event_id
		WHERE opp.opportunity_id = $1`,
		oppInt,
	).Scan(&timezone)
	if err != nil {
		return nil, friendlyDBError(err)
	}

	var startUTC, endUTC *string
	startUTC, err = DateTimeToUTC(shift.StartDateTime, timezone)
	if err == nil {
		endUTC, err = DateTimeToUTC(shift.EndDateTime, timezone)
	}
	if err != nil {
		return nil, err
	}
	if *endUTC <= *startUTC {
		return nil, fmt.Errorf("A shift must end after it starts.")
	}

	var staffId, maxVols interface{}
	if shift.StaffContactId != nil {
		staffId = *shift.StaffContactId
	}
	if shift.MaxVolunteers != nil {
		maxVols = *shift.MaxVolunteers
	}

	// Insert the base shift.
	var shiftInt int
	err = tx.QueryRowContext(ctx, `
		INSERT INTO shifts (opportunity_id, shift_start, shift_end, staff_contact_id, max_volunteers)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING shift_id`,
		oppInt, *startUTC, *endUTC, staffId, maxVols,
	).Scan(&shiftInt)
	if err != nil {
		return nil, friendlyDBError(err)
	}

	// Propagate to sibling opportunities in the recurrence group, if any.
	// Look up the opp's recurrence_template_id and the event's group info.
	var oppTmplID, groupID string
	var order, eventID int
	err = tx.QueryRowContext(ctx, `
		SELECT COALESCE(o.recurrence_template_id::text, ''),
		       COALESCE(e.recurrence_group_id::text, ''),
		       COALESCE(e.recurrence_order, 0),
		       e.event_id
		FROM opportunities o
		JOIN events e ON e.event_id = o.event_id
		WHERE o.opportunity_id = $1`,
		oppInt,
	).Scan(&oppTmplID, &groupID, &order, &eventID)
	if err != nil {
		return nil, fmt.Errorf("error looking up opp recurrence info: %w", err)
	}

	if oppTmplID != "" && groupID != "" {
		// Generate a shift template UUID and stamp the base shift.
		shiftTmplID := uuid.New().String()
		if _, err = tx.ExecContext(ctx,
			`UPDATE shifts SET recurrence_template_id = $1::uuid WHERE shift_id = $2`,
			shiftTmplID, shiftInt,
		); err != nil {
			return nil, fmt.Errorf("error setting shift template id: %w", err)
		}

		srcFirstDate, err := eventFirstDate(ctx, tx, eventID)
		if err != nil {
			return nil, fmt.Errorf("error fetching source event first date: %w", err)
		}
		shiftStart, _ := time.Parse(time.RFC3339, *startUTC)
		shiftEnd, _ := time.Parse(time.RFC3339, *endUTC)

		peers, err := futurePeerEvents(ctx, tx, groupID, order)
		if err != nil {
			return nil, fmt.Errorf("error fetching peer events: %w", err)
		}

		for _, peer := range peers {
			// Find the sibling opportunity on this peer event.
			var sibOppID int
			err = tx.QueryRowContext(ctx,
				`SELECT opportunity_id FROM opportunities
				 WHERE event_id = $1 AND recurrence_template_id = $2::uuid`,
				peer.EventID, oppTmplID,
			).Scan(&sibOppID)
			if err == sql.ErrNoRows {
				continue // sibling missing — skip rather than fail
			}
			if err != nil {
				return nil, fmt.Errorf("finding sibling opp for event %d: %w", peer.EventID, err)
			}

			newStart, newEnd := adjustTimes(shiftStart, shiftEnd, srcFirstDate, peer.FirstDate)
			if _, err = tx.ExecContext(ctx, `
				INSERT INTO shifts
				  (opportunity_id, shift_start, shift_end, staff_contact_id, max_volunteers, recurrence_template_id)
				VALUES ($1, $2, $3, $4, $5, $6::uuid)`,
				sibOppID, newStart, newEnd, staffId, maxVols, shiftTmplID,
			); err != nil {
				return nil, fmt.Errorf("inserting sibling shift for event %d: %w", peer.EventID, err)
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	shiftStr := strconv.Itoa(shiftInt)
	return &models.MutationResult{
		Success: true,
		Message: ptrString("shift successfully added."),
		ID:      &shiftStr,
	}, nil
}

// ============================================================================
// Mutations: Updates and assignments.
// ============================================================================

func (s *ShiftService) UpdateJobType(ctx context.Context, job models.UpdateJobTypeInput) (*models.MutationResult, error) {
	jobStr := strconv.Itoa(job.ID)

	update := `
		UPDATE job_types
		SET
			code = $1,
			name = $2,
			sort_order = $3
		WHERE job_type_id = $4
	`
	_, err := s.DB.ExecContext(ctx, update, job.Code, job.Name, job.SortOrder, job.ID)

	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to update job type."),
			ID:      &jobStr,
		}, err
	}

	return &models.MutationResult{
		Success: true,
		Message: ptrString("Job type successfully updated."),
		ID:      &jobStr,
	}, nil

}

// UpdateOpportunity updates an opportunity's fields.  If the opportunity is
// part of a recurrence group its siblings on all future instances are also
// updated with the same values.
func (s *ShiftService) UpdateOpportunity(ctx context.Context, opp models.UpdateOpportunityInput) (*models.MutationResult, error) {
	oppInt, err := strconv.Atoi(opp.ID)
	if err != nil {
		return &models.MutationResult{Success: false, Message: ptrString("Invalid opp.ID."), ID: &opp.ID}, err
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Update the base opportunity.
	if _, err = tx.ExecContext(ctx, `
		UPDATE opportunities
		SET job_type_id = $1, opportunity_is_virtual = $2, pre_event_instructions = $3
		WHERE opportunity_id = $4`,
		opp.JobId, opp.IsVirtual, opp.PreEventInstructions, oppInt,
	); err != nil {
		return &models.MutationResult{Success: false, Message: ptrString("Failed to update opportunity."), ID: &opp.ID}, err
	}

	// Look up recurrence info to decide whether to propagate.
	var tmplID, groupID string
	var order int
	err = tx.QueryRowContext(ctx, `
		SELECT COALESCE(o.recurrence_template_id::text, ''),
		       COALESCE(e.recurrence_group_id::text, ''),
		       COALESCE(e.recurrence_order, 0)
		FROM opportunities o
		JOIN events e ON e.event_id = o.event_id
		WHERE o.opportunity_id = $1`,
		oppInt,
	).Scan(&tmplID, &groupID, &order)
	if err != nil {
		return nil, fmt.Errorf("error looking up opp recurrence info: %w", err)
	}

	if tmplID != "" && groupID != "" {
		// Propagate the same field values to all sibling opps on future events.
		if _, err = tx.ExecContext(ctx, `
			UPDATE opportunities o
			SET job_type_id = $1, opportunity_is_virtual = $2, pre_event_instructions = $3
			FROM events e
			WHERE o.event_id = e.event_id
			  AND o.recurrence_template_id = $4::uuid
			  AND e.recurrence_group_id = $5::uuid
			  AND e.recurrence_order > $6`,
			opp.JobId, opp.IsVirtual, opp.PreEventInstructions, tmplID, groupID, order,
		); err != nil {
			return nil, fmt.Errorf("error propagating opp update: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return &models.MutationResult{
		Success: true,
		Message: ptrString("Opportunity successfully updated."),
		ID:      &opp.ID,
	}, nil
}

// UpdateShift updates a shift's time window, max volunteers, and staff contact.
// If the shift belongs to a recurrence group its siblings on all future instances
// are also updated; their times are recalculated to preserve the same offset
// from each event's first scheduled date.
func (s *ShiftService) UpdateShift(ctx context.Context, shift models.UpdateShiftInput) (*models.MutationResult, error) {
	shiftInt, err := strconv.Atoi(shift.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid shift id: %w", err)
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Fetch timezone, recurrence info, and shift template in one query.
	var timezone, shiftTmplID, groupID string
	var order, eventID int
	err = tx.QueryRowContext(ctx, `
		SELECT e.timezone,
		       COALESCE(s.recurrence_template_id::text, ''),
		       COALESCE(e.recurrence_group_id::text, ''),
		       COALESCE(e.recurrence_order, 0),
		       e.event_id
		FROM events e
		JOIN opportunities opp ON opp.event_id = e.event_id
		JOIN shifts s ON s.opportunity_id = opp.opportunity_id
		WHERE s.shift_id = $1`,
		shiftInt,
	).Scan(&timezone, &shiftTmplID, &groupID, &order, &eventID)
	if err != nil {
		return nil, friendlyDBError(err)
	}

	var startUTC, endUTC *string
	startUTC, err = DateTimeToUTC(shift.StartDateTime, timezone)
	if err == nil {
		endUTC, err = DateTimeToUTC(shift.EndDateTime, timezone)
	}
	if err != nil {
		return nil, err
	}
	if *endUTC <= *startUTC {
		return nil, fmt.Errorf("A shift must end after it starts.")
	}

	// Update the base shift.
	if _, err = tx.ExecContext(ctx, `
		UPDATE shifts
		SET shift_start = $1, shift_end = $2, max_volunteers = $3, staff_contact_id = $4
		WHERE shift_id = $5`,
		startUTC, endUTC, shift.MaxVolunteers, shift.StaffContactId, shiftInt,
	); err != nil {
		return nil, friendlyDBError(err)
	}

	// Propagate to sibling shifts on future instances, with time offsets recalculated
	// relative to each event's first scheduled date.
	if shiftTmplID != "" && groupID != "" {
		srcFirstDate, err := eventFirstDate(ctx, tx, eventID)
		if err != nil {
			return nil, fmt.Errorf("error fetching source event first date: %w", err)
		}
		shiftStart, _ := time.Parse(time.RFC3339, *startUTC)
		shiftEnd, _ := time.Parse(time.RFC3339, *endUTC)

		peers, err := futurePeerEvents(ctx, tx, groupID, order)
		if err != nil {
			return nil, fmt.Errorf("error fetching peer events: %w", err)
		}
		for _, peer := range peers {
			newStart, newEnd := adjustTimes(shiftStart, shiftEnd, srcFirstDate, peer.FirstDate)
			if _, err = tx.ExecContext(ctx, `
				UPDATE shifts s
				SET shift_start = $1, shift_end = $2, max_volunteers = $3, staff_contact_id = $4
				FROM opportunities o, events e
				WHERE s.opportunity_id = o.opportunity_id
				  AND o.event_id = e.event_id
				  AND s.recurrence_template_id = $5::uuid
				  AND e.event_id = $6`,
				newStart, newEnd, shift.MaxVolunteers, shift.StaffContactId, shiftTmplID, peer.EventID,
			); err != nil {
				return nil, fmt.Errorf("error updating sibling shift for event %d: %w", peer.EventID, err)
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return &models.MutationResult{
		Success: true,
		Message: ptrString("Shift successfully updated."),
		ID:      &shift.ID,
	}, nil
}

func (s *ShiftService) AssignSelfToShift(ctx context.Context, shiftId string, volId int) (*models.MutationResult, error) {
	return assignVolToShift(ctx, s.DB, s.mailer, shiftId, volId)
}

func (s *ShiftService) AssignVolunteerToShift(ctx context.Context, shiftId string, volunteerId string) (*models.MutationResult, error) {

	volInt, err := strconv.Atoi(volunteerId)
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Invalid volunteerId."),
			ID:      &volunteerId,
		}, err
	}

	return assignVolToShift(ctx, s.DB, s.mailer, shiftId, volInt)
}

// ============================================================================
// Mutations: Deletions and cancellations of assignments.
// ============================================================================

// Soft delete since jobs are part of event history.
func (s *ShiftService) DeleteJobType(ctx context.Context, jobTypeId int) (*models.MutationResult, error) {

	update := `
		UPDATE job_types
		SET
			is_active = false,
			sort_order = 0
		WHERE job_type_id = $1
	`
	_, err := s.DB.ExecContext(ctx, update, jobTypeId)
	if err != nil {
		return nil, fmt.Errorf("unable to delete job type: %w", err)
	}

	jobTypeStr := strconv.Itoa(jobTypeId)
	return &models.MutationResult{
		Success: true,
		Message: ptrString("Successfully deleted job type."),
		ID:      &jobTypeStr,
	}, nil
}

// DeleteOpportunity removes an opportunity and its shifts.  If the opportunity
// is part of a recurrence group its siblings on all future instances are also
// deleted (the DB cascades their shifts automatically).
func (s *ShiftService) DeleteOpportunity(ctx context.Context, oppId string) (*models.MutationResult, error) {
	oppInt, err := strconv.Atoi(oppId)
	if err != nil {
		return &models.MutationResult{Success: false, Message: ptrString("Failed to delete opportunity."), ID: &oppId}, err
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Look up recurrence info before deleting so we can propagate.
	var tmplID, groupID string
	var order int
	err = tx.QueryRowContext(ctx, `
		SELECT COALESCE(o.recurrence_template_id::text, ''),
		       COALESCE(e.recurrence_group_id::text, ''),
		       COALESCE(e.recurrence_order, 0)
		FROM opportunities o
		JOIN events e ON e.event_id = o.event_id
		WHERE o.opportunity_id = $1`,
		oppInt,
	).Scan(&tmplID, &groupID, &order)
	if err != nil {
		return &models.MutationResult{Success: false, Message: ptrString("Failed to delete opportunity."), ID: &oppId}, err
	}

	// Delete the base opportunity (DB cascades to its shifts).
	if _, err = tx.ExecContext(ctx, `DELETE FROM opportunities WHERE opportunity_id = $1`, oppInt); err != nil {
		return &models.MutationResult{Success: false, Message: ptrString("Failed to delete opportunity."), ID: &oppId}, err
	}

	// Propagate deletion to sibling opps on future instances.
	if tmplID != "" && groupID != "" {
		if _, err = tx.ExecContext(ctx, `
			DELETE FROM opportunities o
			USING events e
			WHERE o.event_id = e.event_id
			  AND o.recurrence_template_id = $1::uuid
			  AND e.recurrence_group_id = $2::uuid
			  AND e.recurrence_order > $3`,
			tmplID, groupID, order,
		); err != nil {
			return nil, fmt.Errorf("error deleting sibling opps: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return &models.MutationResult{
		Success: true,
		Message: ptrString("Opportunity successfully deleted."),
		ID:      &oppId,
	}, nil
}

// DeleteShift removes a shift.  An opportunity must always have at least one
// shift, so the delete is blocked if:
//   - the base opportunity would be left with zero shifts, or
//   - any sibling opportunity (on a future recurring instance) would be left
//     with zero shifts after its copy of the shift is also deleted.
//
// If the shift is part of a recurrence group its siblings on future instances
// are deleted as well.
func (s *ShiftService) DeleteShift(ctx context.Context, shiftId string) (*models.MutationResult, error) {
	shiftInt, err := strconv.Atoi(shiftId)
	if err != nil {
		return nil, fmt.Errorf("invalid shiftId, %s: %w", shiftId, err)
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Fetch opportunity, shift count, and recurrence info in one query.
	var oppInt, numShifts, order, eventID int
	var shiftTmplID, groupID string
	err = tx.QueryRowContext(ctx, `
		SELECT s.opportunity_id,
		       (SELECT COUNT(*) FROM shifts WHERE opportunity_id = s.opportunity_id),
		       COALESCE(s.recurrence_template_id::text, ''),
		       COALESCE(e.recurrence_group_id::text, ''),
		       COALESCE(e.recurrence_order, 0),
		       e.event_id
		FROM shifts s
		JOIN opportunities o ON o.opportunity_id = s.opportunity_id
		JOIN events e ON e.event_id = o.event_id
		WHERE s.shift_id = $1`,
		shiftInt,
	).Scan(&oppInt, &numShifts, &shiftTmplID, &groupID, &order, &eventID)
	if err != nil {
		return nil, fmt.Errorf("unable to find shift %d: %w", shiftInt, err)
	}

	// The base opportunity must keep at least one shift.
	if numShifts < 2 {
		return nil, fmt.Errorf("cannot delete the last shift associated with opportunity %d", oppInt)
	}

	// For recurring shifts: ensure no sibling opp would be left with zero shifts.
	if shiftTmplID != "" && groupID != "" {
		var wouldEmpty int
		err = tx.QueryRowContext(ctx, `
			SELECT COUNT(*)
			FROM opportunities o
			JOIN events e ON e.event_id = o.event_id
			WHERE o.recurrence_template_id = (
			        SELECT recurrence_template_id FROM opportunities WHERE opportunity_id = $1
			      )
			  AND e.recurrence_group_id = $2::uuid
			  AND e.recurrence_order > $3
			  AND (SELECT COUNT(*) FROM shifts WHERE opportunity_id = o.opportunity_id) = 1`,
			oppInt, groupID, order,
		).Scan(&wouldEmpty)
		if err != nil {
			return nil, fmt.Errorf("error checking sibling shift counts: %w", err)
		}
		if wouldEmpty > 0 {
			return nil, fmt.Errorf(
				"cannot delete: %d future event(s) would have no shifts remaining on this opportunity",
				wouldEmpty,
			)
		}
	}

	// Delete the base shift.
	if _, err = tx.ExecContext(ctx, `DELETE FROM shifts WHERE shift_id = $1`, shiftInt); err != nil {
		return &models.MutationResult{Success: false, Message: ptrString("Failed to delete shift."), ID: &shiftId}, err
	}

	// Propagate deletion to sibling shifts on future instances.
	if shiftTmplID != "" && groupID != "" {
		if _, err = tx.ExecContext(ctx, `
			DELETE FROM shifts s
			USING opportunities o, events e
			WHERE s.opportunity_id = o.opportunity_id
			  AND o.event_id = e.event_id
			  AND s.recurrence_template_id = $1::uuid
			  AND e.recurrence_group_id = $2::uuid
			  AND e.recurrence_order > $3`,
			shiftTmplID, groupID, order,
		); err != nil {
			return nil, fmt.Errorf("error deleting sibling shifts: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return &models.MutationResult{
		Success: true,
		Message: ptrString("Shift successfully deleted."),
		ID:      &shiftId,
	}, nil
}

func (s *ShiftService) CancelOwnShift(ctx context.Context, shiftId string, volId int) (*models.MutationResult, error) {
	return cancelShiftAssignment(ctx, s.DB, s.mailer, shiftId, volId)
}

func (s *ShiftService) CancelShiftAssignment(ctx context.Context, shiftId string, volId string) (*models.MutationResult, error) {

	volInt, err := strconv.Atoi(volId)
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("VolunteerId is not valid."),
			ID:      nil,
		}, err
	}

	return cancelShiftAssignment(ctx, s.DB, s.mailer, shiftId, volInt)
}
