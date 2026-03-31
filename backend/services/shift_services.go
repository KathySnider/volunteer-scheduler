package services

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"volunteer-scheduler/models"
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
		opp.PreEventInstructions = &instruct.String

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

// Mutations: Create opportunities and shifts.

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

func (s *ShiftService) CreateOpportunity(ctx context.Context, opp models.NewOpportunityInput) (*models.MutationResult, error) {
	var oppInt int

	// Start a transaction, since adding the new
	// opp and it's shifts outght to be atomic.

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		err = fmt.Errorf("error starting transaction to create opportunity: %w", err)
		return nil, err
	}

	// Defer a rollback in case anything fails.
	defer tx.Rollback()

	query := `
		INSERT INTO opportunities (
			event_id, 
			job_type_id, 
			opportunity_is_virtual, 
			pre_event_instructions)
		VALUES ($1, $2, $3, $4)
		RETURNING opportunity_id
	`
	err = tx.QueryRowContext(ctx, query, opp.EventId, opp.JobId, opp.IsVirtual, opp.PreEventInstructions).Scan(&oppInt)

	if err == nil {
		// Opportunity was created. Add shifts.
		err = addNewOpportunityShifts(ctx, opp.Shifts, oppInt, tx)

		if err != nil {
			err = fmt.Errorf("error adding shifts to opportunity: %w", err)
		}
	}

	if err != nil {
		// Return the error now. The defered rollback
		// will happen when we return.
		return &models.MutationResult{
			Success: false,
			Message: ptrString("error creating opportunity"),
			ID:      nil,
		}, err
	}

	// Commit the transaction.
	err = tx.Commit()
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("error committing transaction"),
			ID:      nil,
		}, err
	}

	oppStr := strconv.Itoa(oppInt)
	return &models.MutationResult{
		Success: true,
		Message: ptrString("opportunity sucessfully created"),
		ID:      &oppStr,
	}, nil
}

// CreateShift
// Called from the client when a shift is added to an existing
// opportunity. The parent is already known (included in the
// input structure). No transaction is needed.
func (s *ShiftService) CreateShift(ctx context.Context, shift models.AddShiftInput) (*models.MutationResult, error) {
	var shiftInt int
	var startUTC, endUTC *string
	var staffId, maxVols interface{}

	// sConvert dates, times to UTC.
	startUTC, err := DateTimeToUTC(shift.StartDateTime, shift.IanaZone)
	if err == nil {
		endUTC, err = DateTimeToUTC(shift.EndDateTime, shift.IanaZone)
	}
	if err != nil {
		return nil, err
	}

	// Handle optional values.
	if shift.StaffContactId != nil {
		staffId = *shift.StaffContactId
	}
	if shift.MaxVolunteers != nil {
		maxVols = *shift.MaxVolunteers
	}

	insert := `
		INSERT INTO shifts (
			opportunity_id, 
			shift_start, 
			shift_end, 
			staff_contact_id,
			max_volunteers)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING shift_id
	`
	err = s.DB.QueryRowContext(ctx, insert, shift.OppId, *startUTC, *endUTC, staffId, maxVols).Scan(&shiftInt)

	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("error creating shift"),
			ID:      nil,
		}, err
	}

	shiftStr := strconv.Itoa(shiftInt)
	return &models.MutationResult{
		Success: true,
		Message: ptrString("shift successfully added."),
		ID:      &shiftStr,
	}, nil
}

// Mutations: Updates and assignments.

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
func (s *ShiftService) UpdateOpportunity(ctx context.Context, opp models.UpdateOpportunityInput) (*models.MutationResult, error) {

	oppInt, err := strconv.Atoi(opp.ID)
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Invalid opp.ID."),
			ID:      &opp.ID,
		}, err
	}

	update := `
		UPDATE opportunities 
		SET 
			job_type_id = $1,
			opportunity_is_virtual = $2, 
			pre_event_instructions = $3
		WHERE opportunity_id = $4
	`
	_, err = s.DB.ExecContext(ctx, update, opp.JobId, opp.IsVirtual, opp.PreEventInstructions, oppInt)
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to update opportunity."),
			ID:      &opp.ID,
		}, err
	}

	return &models.MutationResult{
		Success: true,
		Message: ptrString("Opportunity sucessfully updated."),
		ID:      &opp.ID,
	}, nil
}

func (s *ShiftService) UpdateShift(ctx context.Context, shift models.UpdateShiftInput) (*models.MutationResult, error) {
	var startUTC, endUTC *string

	// Convert to datetimes used in the DB.
	startUTC, err := DateTimeToUTC(shift.StartDateTime, shift.IanaZone)
	if err == nil {
		endUTC, err = DateTimeToUTC(shift.EndDateTime, shift.IanaZone)
	}
	if err != nil {
		return nil, err
	}

	update := `
		UPDATE shifts 
		SET 
			shift_start = $1, 
			shift_end = $2, 
			max_volunteers = $3, 
			staff_contact_id = $4
		WHERE shift_id = $5
	`
	_, err = s.DB.ExecContext(ctx, update, startUTC, endUTC, shift.MaxVolunteers, shift.StaffContactId, shift.ID)

	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to update shift."),
			ID:      nil,
		}, err
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

// Mutations: Deletions and cancellations of assignments.

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

// Deleting an opportunity will delete all shifts associated with it.
func (s *ShiftService) DeleteOpportunity(ctx context.Context, oppId string) (*models.MutationResult, error) {
	oppInt, err := strconv.Atoi(oppId)
	if err == nil {
		_, err = s.DB.ExecContext(ctx, "DELETE FROM opportunities WHERE opportunity_id = $1", oppInt)
	}

	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to delete opportunity."),
			ID:      &oppId,
		}, err
	}
	return &models.MutationResult{
		Success: true,
		Message: ptrString("Opportunity successfully deleted."),
		ID:      &oppId,
	}, nil
}

func (s *ShiftService) DeleteShift(ctx context.Context, shiftId string) (*models.MutationResult, error) {

	shiftInt, err := strconv.Atoi(shiftId)
	if err != nil {
		return nil, fmt.Errorf("invalid shiftId, %s: %w", shiftId, err)
	}

	// An opportunity must have at least one shift. Do not delete
	// the shift if this is the last one.
	query := `
    SELECT
        s.opportunity_id,
        (SELECT COUNT(*) FROM shifts WHERE opportunity_id = s.opportunity_id) AS num_shifts
    FROM shifts s
    WHERE s.shift_id = $1
	`
	var oppInt, numShifts int
	err = s.DB.QueryRowContext(ctx, query, shiftInt).Scan(&oppInt, &numShifts)
	if err != nil {
		return nil, fmt.Errorf("unable to find shift with id %d in the DB: %w", shiftInt, err)
	}
	if numShifts < 2 {
		return nil, fmt.Errorf("cannot delete the last shift associated with opportunity %d", oppInt)
	}
	_, err = s.DB.ExecContext(ctx, "DELETE FROM shifts WHERE shift_id = $1", shiftInt)

	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to delete shift."),
			ID:      &shiftId,
		}, err
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
