package services

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"volunteer-scheduler/models"
)

type VolunteerService struct {
	DB        *sql.DB
	mailer    *Mailer
	roleCache map[string]int
}

func NewVolunteerService(db *sql.DB, mailer *Mailer) *VolunteerService {
	return &VolunteerService{
		DB:     db,
		mailer: mailer,
	}
}

// Queries.

// FetchVolunteers retrieves all volunteers with optional filtering
func (s *VolunteerService) FetchAllVolunteers(ctx context.Context, filter *models.VolunteerFilterInput) ([]*models.Volunteer, error) {
	// Get all volunteers for now.
	// We may need a filter here, so it's in the input, but
	// we'll deal with that when we know if we are looking
	// up by email or name or ?? Darrell is working on that.

	query := `
		SELECT 
			volunteer_id, 
			first_name, 
			last_name, 
			email, 
			phone, 
			zip_code,
			role
		FROM volunteers 
	`
	rows, err := s.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error querying volunteers: %w", err)
	}
	defer rows.Close()

	var volunteers []*models.Volunteer
	for rows.Next() {
		var v models.Volunteer
		var volInt int
		var phone, zip sql.NullString

		err := rows.Scan(
			&volInt,
			&v.FirstName,
			&v.LastName,
			&v.Email,
			&phone,
			&zip,
			&v.Role)
		if err != nil {
			return nil, fmt.Errorf("error scanning volunteer: %w", err)
		}
		if phone.Valid {
			v.Phone = &phone.String
		} else {
			v.Phone = nil
		}
		if zip.Valid {
			v.ZipCode = &zip.String
		} else {
			v.ZipCode = nil
		}
		v.ID = strconv.Itoa(volInt)
		volunteers = append(volunteers, &v)
	}

	return volunteers, nil
}

func (s *VolunteerService) FetchOwnProfile(ctx context.Context, volId int) (*models.VolunteerProfile, error) {
	return fetchProfile(ctx, s.DB, volId)
}

func (s *VolunteerService) FetchVolunteerProfileById(ctx context.Context, volId string) (*models.VolunteerProfile, error) {

	volInt, err := strconv.Atoi(volId)
	if err != nil {
		return nil, fmt.Errorf("volunteer id is not valid: %w", err)
	}
	return fetchProfile(ctx, s.DB, volInt)
}

func (s *VolunteerService) UpdateOwnProfile(ctx context.Context, volId int, profile models.UpdateOwnProfileInput) (*models.VolunteerMutationResult, error) {
	query := `
		UPDATE volunteers 
		SET 
			first_name = $1, 
			last_name = $2, 
			email = $3,
			phone = $4,
			zip_code = $5
		WHERE volunteer_id = $6
	`
	_, err := s.DB.ExecContext(ctx, query, profile.FirstName, profile.LastName, profile.Email, profile.Phone, profile.ZipCode, volId)

	if err != nil {
		return &models.VolunteerMutationResult{
			Success: false,
			Message: ptrString("Failed to update volunteer profile."),
		}, err
	}

	return &models.VolunteerMutationResult{
		Success: true,
		Message: ptrString("Volunteer successfully updated."),
	}, nil
}

// Mutations.

// CreateVolunteer is the resolver for the createVolunteer field.
func (s *VolunteerService) CreateVolunteer(ctx context.Context, newVol models.NewVolunteerInput) (*models.MutationResult, error) {
	query := `
		INSERT INTO volunteers (
			first_name, 
			last_name, 
			email, 
			phone, 
			zip_code,
			role,
			created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		RETURNING volunteer_id
	`

	var volInt int
	err := s.DB.QueryRowContext(ctx, query, newVol.FirstName, newVol.LastName, newVol.Email, newVol.Phone, newVol.ZipCode, newVol.Role).Scan(&volInt)
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to create volunteer."),
			ID:      nil,
		}, err
	}

	volStr := strconv.Itoa(volInt)
	return &models.MutationResult{
		Success: true,
		Message: ptrString("Successfullly created volunteer."),
		ID:      &volStr,
	}, nil
}

// UpdateVolunteerProfile
// Volunteers whose identities have been created by an admin
// may update their own profiles as their lives change. Admins
// can also update their profiles for them. Note that only
// admins can change a volunteer's role (via this function).
func (s *VolunteerService) UpdateVolunteerProfile(ctx context.Context, profile models.UpdateVolunteerInput) (*models.MutationResult, error) {

	volInt, err := strconv.Atoi(profile.ID)
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to update profile. Invalid Id."),
			ID:      &profile.ID,
		}, err
	}

	query := `
		UPDATE volunteers 
		SET 
			first_name = $1, 
			last_name = $2, 
			email = $3,
			phone = $4,
			zip_code = $5,
			role = $6,
		WHERE volunteer_id = $7
	`
	_, err = s.DB.ExecContext(ctx, query, profile.FirstName, profile.LastName, profile.Email, profile.Phone, profile.ZipCode, profile.Role, volInt)

	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to update volunteer profile."),
			ID:      &profile.ID,
		}, err
	}

	return &models.MutationResult{
		Success: true,
		Message: ptrString("Volunteer successfully updated."),
		ID:      &profile.ID,
	}, nil
}

// TODO: determine if we should delete volunteers or just mark them
// as inactive or something. Maybe want them for history of events?
// Meanwhile....
func (s *VolunteerService) DeleteVolunteer(ctx context.Context, volId string) (*models.MutationResult, error) {

	volInt, err := strconv.Atoi(volId)
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Invalid volunteer ID."),
			ID:      &volId,
		}, err
	}

	_, err = s.DB.ExecContext(ctx, "DELETE FROM volunteers WHERE volunteer_id = $1", volInt)

	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to delete volunteer."),
			ID:      &volId,
		}, err
	}

	return &models.MutationResult{
		Success: true,
		Message: ptrString("Volunteer successfully deleted."),
		ID:      &volId,
	}, nil
}
