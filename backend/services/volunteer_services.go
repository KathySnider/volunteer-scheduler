package services

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"volunteer-scheduler/models"
)

type VolunteerService struct {
	DB *sql.DB
}

func NewVolunteerService(db *sql.DB) *VolunteerService {
	return &VolunteerService{DB: db}
}

// Function: AssignVolunteerToShift
// Returns success or failure with error.
// Access: All users.
func (s *VolunteerService) AssignVolunteerToShift(ctx context.Context, shiftID string, volunteerID string) (*models.UpdateResult, error) {

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

func (s *VolunteerService) CreateVolunteer(ctx context.Context, profile *models.NewVolunteerInput) (*models.InsertResult, error) {
	var query string

	query = `
		INSERT INTO volunteers (first_name, last_name, email, created_at)
		VALUES ($1, $2, $3, NOW())
		RETURNING volunteer_id
	`
	var volID int
	err := s.DB.QueryRowContext(ctx, query, profile.FirstName, profile.LastName, profile.Email).Scan(&volID)

	if err != nil {
		return &models.InsertResult{
			Success: false,
			Message: ptrString("Failed to create new volunteer."),
			ID:      nil,
		}, err
	}

	volIDStr := strconv.Itoa(volID)
	return &models.InsertResult{
		Success: true,
		Message: ptrString("Volunteer successfully created."),
		ID:      &volIDStr,
	}, nil

}

// GetAllVolunteers retrieves all volunteers with optional filtering
func (s *VolunteerService) GetAllVolunteers(ctx context.Context, filter *models.VolunteerFilterInput) ([]*models.VolunteerProfile, error) {
	var query string
	var args []interface{}

	// Get all volunteers for now.
	// We may need a filter here, so it's in the input, but
	// we'll deal with that when we know if we are looking
	// up by email or name or ?? Darrell is working on that.
	query = `
		SELECT volunteer_id, first_name, last_name
		FROM volunteers 
	`

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("error querying volunteers: %w", err)
	}
	defer rows.Close()

	var volunteers []*models.VolunteerProfile
	for rows.Next() {
		var v models.VolunteerProfile
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

// Volunteers whose identities have been created by an admin
// may (eventually) update their profiles as their lives change.

// GetVolunteerProfile
// Retrieve the profile of a specific volunteer.
func (s *VolunteerService) GetVolunteerProfile(ctx context.Context, id string) (*models.VolunteerProfile, error) {
	var query string

	query = `
		SELECT 
			v.volunteer_id, 
			v.first_name, 
			v.last_name,
			v.email,
			v.phone,
			v.zip_code
		FROM volunteers v
		WHERE v.volunteer_id = $1
	`
	var volunteer models.VolunteerProfile
	var volID int

	// Fill in these fields for now....
	err := s.DB.QueryRowContext(ctx, query, id).Scan(
		&volID,
		&volunteer.FirstName,
		&volunteer.LastName,
		&volunteer.Email,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("volunteer not found")
	}
	if err != nil {
		return nil, fmt.Errorf("error querying volunteer: %w", err)
	}

	volunteer.ID = fmt.Sprintf("%d", volID) // string for GraphQl

	// TODO: get service types??
	// Still not sure about how those are supposed to be
	// used. Should they be associated with a volunteer
	// at all?

	return &volunteer, nil
}

// UpdateVolunteerProfile
// Push an edited profile back to the DB.
func (s *VolunteerService) UpdateVolunteerProfile(ctx context.Context, profile *models.UpdateVolunteerInput) (*models.UpdateResult, error) {
	// TODO
	return nil, fmt.Errorf("not implemented")
}
