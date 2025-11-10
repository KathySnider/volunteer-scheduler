package graph

import (
	"context"
	"fmt"
	"strings"
	"time"

	"vol_sched_api/graph/model"
)

func (r *queryResolver) getOpportunitiesForEvent(ctx context.Context, eventID int) ([]*model.Opportunity, error) {
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

	var opportunities []*model.Opportunity
	for rows.Next() {
		var opp model.Opportunity
		var oppID int
		var role string
		var isVirtual bool

		err := rows.Scan(&oppID, &role, &isVirtual)
		if err != nil {
			return nil, fmt.Errorf("error scanning opportunity: %w", err)
		}

		opp.ID = fmt.Sprintf("%d", oppID)
		opp.Role = model.RoleType(strings.ToUpper(role))

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

func (r *queryResolver) getShiftsForOpportunity(ctx context.Context, opportunityID int) ([]*model.Shift, error) {
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

	var shifts []*model.Shift
	for rows.Next() {
		var shift model.Shift
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
			var assignedVols []*model.Volunteer
			for volRows.Next() {
				var vol model.Volunteer
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
