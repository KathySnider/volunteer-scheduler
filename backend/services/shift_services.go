package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	"volunteer-scheduler/models"
)

type ShiftService struct {
	DB *sql.DB
}

func NewShiftService(db *sql.DB) *ShiftService {
	return &ShiftService{DB: db}
}

// AssignVolunteerToShift
// Assigns the volunteer to the shift.
func (s *ShiftService) AssignVolunteerToShift(ctx context.Context, shiftID string, volunteerID string) (*models.UpdateResult, error) {
	query := `
		INSERT INTO volunteer_shifts (volunteer_id, shift_id, assigned_at, status)
		VALUES ($1, $2, NOW(), 'confirmed')
		ON CONFLICT (volunteer_id, shift_id) DO NOTHING
	`

	_, err := s.DB.ExecContext(ctx, query, volunteerID, shiftID)
	if err != nil {
		return &models.UpdateResult{
			Success: false,
			Message: ptrString("Failed to assign volunteer to shift"),
		}, nil
	}

	return &models.UpdateResult{
		Success: true,
		Message: ptrString("Volunteer successfully assigned"),
	}, nil
}

// CancelShiftAssignment
// Cancels a volunteer's shift assignment.
// NOTE: in addition to taking the assignment out of the DB,
// the code should send an email to the volunteer lead? or
// to the volunteer coordinators? or someone?
func (s *ShiftService) CancelShiftAssignment(ctx context.Context, shiftID string, volunteerID string) (*models.UpdateResult, error) {
	// TODO
	return &models.UpdateResult{
		Success: false,
		Message: ptrString("not implemented"),
	}, nil
}

// GetShiftsForOpportunity
// Retrieve shifts associated with the opportunity.
func (s *ShiftService) GetShiftsForOpportunity(ctx context.Context, opportunityID int) ([]*models.Shift, error) {
	shiftQuery := `
		SELECT shift_id, shift_start, shift_end, max_volunteers
		FROM shifts
		WHERE opportunity_id = $1
	`

	rows, err := s.DB.QueryContext(ctx, shiftQuery, opportunityID)
	if err != nil {
		return nil, fmt.Errorf("error querying shifts: %w", err)
	}
	defer rows.Close()

	var shifts []*models.Shift
	for rows.Next() {
		var shift models.Shift
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
		volRows, err := s.DB.QueryContext(ctx, volQuery, shiftID)
		if err == nil {
			var assignedVols []*models.VolunteerProfile
			for volRows.Next() {
				var vol models.VolunteerProfile
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

// Helper function to create string pointer
func ptrString(s string) *string {
	return &s
}
