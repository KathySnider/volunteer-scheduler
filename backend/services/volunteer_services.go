package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"

	"volunteer-scheduler/models"
)

type VolunteerService struct {
	DB     *sql.DB
	mailer *Mailer
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
			default_distance_miles,
			role
		FROM volunteers
		WHERE is_active = TRUE
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
		var ddm sql.NullInt32

		err := rows.Scan(
			&volInt,
			&v.FirstName,
			&v.LastName,
			&v.Email,
			&phone,
			&zip,
			&ddm,
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
		if ddm.Valid {
			dist := int(ddm.Int32)
			v.Distance = &dist
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

// Mutations.

func (s *VolunteerService) UpdateOwnProfile(ctx context.Context, volId int, profile models.UpdateOwnProfileInput) (*models.VolunteerMutationResult, error) {

	var lat, lng *float64
	var err error
	if profile.ZipCode != nil {
		lat, lng, err = GeocodeZip(*profile.ZipCode)
		if err != nil {
			// Log this, but don't error out - lat and lng will just be nil.
			log.Printf("Unable to get lat/lng for zip %v.", *profile.ZipCode)
		}
	}

	query := `
		UPDATE volunteers 
		SET 
			first_name = $1, 
			last_name = $2, 
			email = $3,
			phone = $4,
			zip_code = $5,
			default_distance_miles = $6,
			latitude = $7,
			longitude = $8
		WHERE volunteer_id = $9
	`
	_, err = s.DB.ExecContext(ctx, query, profile.FirstName, profile.LastName, profile.Email, profile.Phone, profile.ZipCode, profile.Distance, lat, lng, volId)

	if err != nil {
		return nil, fmt.Errorf("unable to update vol profile: %w", err)
	}

	return &models.VolunteerMutationResult{
		Success: true,
		Message: ptrString("Volunteer successfully updated."),
	}, nil
}

// CreateVolunteer is the resolver for the createVolunteer field.
func (s *VolunteerService) CreateVolunteer(ctx context.Context, creatorId int, newVol models.NewVolunteerInput) (*models.MutationResult, error) {

	var lat, lng *float64
	var err error
	if newVol.ZipCode != nil {
		lat, lng, err = GeocodeZip(*newVol.ZipCode)
		if err != nil {
			// Log this, but don't error out - lat and lng will just be nil.
			log.Printf("Unable to get lat/lng for zip %v.", *newVol.ZipCode)
		}
	}

	query := `
		INSERT INTO volunteers (
			first_name, 
			last_name, 
			email, 
			phone, 
			zip_code,
			default_distance_miles,
			latitude,
			longitude,
			role,
			created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		RETURNING volunteer_id
	`

	var volInt int
	err = s.DB.QueryRowContext(ctx, query, newVol.FirstName, newVol.LastName, newVol.Email, newVol.Phone, newVol.ZipCode, newVol.Distance, lat, lng, newVol.Role).Scan(&volInt)
	if err != nil {
		friendly := friendlyDBError(err)
		return &models.MutationResult{
			Success: false,
			Message: ptrString(friendly.Error()),
			ID:      nil,
		}, friendly
	}

	var role string
	if newVol.Role == "VOLUNTEER" {
		role = "volunteer"
	} else {
		role = "administrator"
	}

	// Get the creating admin's email for the notification.
	createdByEmail, err := fetchEmailByVolId(ctx, s.DB, creatorId)
	if err != nil {
		log.Printf("Warning: could not fetch creating admin email: %v", err)
		createdByEmail = "unknown"
	}

	// Welcome email to the new volunteer.
	err = sendAccountCreated(ctx, s.mailer, newVol.FirstName, newVol.LastName, newVol.Email, role)
	if err != nil {
		log.Printf("Warning: failed to send welcome email to %s: %v", newVol.Email, err)
	}

	// Notification to all admins.
	err = sendAccountCreatedAdminNotification(ctx, s.DB, s.mailer, newVol.FirstName, newVol.LastName, newVol.Email, role, createdByEmail)
	if err != nil {
		log.Printf("Warning: failed to send admin notification for %s: %v", newVol.Email, err)
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
		return nil, fmt.Errorf("failed to update profile - invalid id %s: %w", profile.ID, err)
	}

	var lat, lng *float64
	if profile.ZipCode != nil {
		lat, lng, err = GeocodeZip(*profile.ZipCode)
		if err != nil {
			// Log this, but don't error out - lat and lng will just be nil.
			log.Printf("Unable to get lat/lng for zip %v.", *profile.ZipCode)
		}
	}

	query := `
		UPDATE volunteers 
		SET 
			first_name = $1, 
			last_name = $2, 
			email = $3,
			phone = $4,
			zip_code = $5,
			default_distance_miles = $6,
			latitude = $7,
			longitude = $8,
			role = $9
		WHERE volunteer_id = $10
	`
	_, err = s.DB.ExecContext(ctx, query, profile.FirstName, profile.LastName, profile.Email, profile.Phone, profile.ZipCode, profile.Distance, lat, lng, profile.Role, volInt)

	if err != nil {
		return nil, friendlyDBError(err)
	}

	return &models.MutationResult{
		Success: true,
		Message: ptrString("Volunteer successfully updated."),
		ID:      &profile.ID,
	}, nil
}

// This is a "soft deletion". The volunteer will not be deleted from the DB,
// since their history is tied to events. So we will rather mark the volunteer
// as inactive (i.e., is_active = FALSE).
func (s *VolunteerService) DeleteVolunteer(ctx context.Context, volId string) (*models.MutationResult, error) {

	volInt, err := strconv.Atoi(volId)
	if err != nil {
		return nil, fmt.Errorf("unable to delete vol - invalid id %s: %w", volId, err)
	}

	_, err = s.DB.ExecContext(ctx, "UPDATE volunteers SET is_active = FALSE WHERE volunteer_id = $1", volInt)

	if err != nil {
		return nil, fmt.Errorf("unable to deactivate volunteer with id %s: %w", volId, err)
	}

	return &models.MutationResult{
		Success: true,
		Message: ptrString("Volunteer successfully deactivated."),
		ID:      &volId,
	}, nil
}
