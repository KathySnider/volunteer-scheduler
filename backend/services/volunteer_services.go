package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"

	"volunteer-scheduler/models"

	"github.com/lib/pq"
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
func (s *VolunteerService) FetchVolunteers(ctx context.Context, filter *models.VolunteerFilterInput) ([]*models.Volunteer, error) {
	// Get all volunteers for now.
	// We may need a filter here, so it's in the input, but
	// we'll deal with that when we know if we are looking
	// up by email or name or ?? Darrell is working on that.

	query := `
		SELECT
			v.volunteer_id,
			v.first_name,
			v.last_name,
			v.email,
			v.phone,
			v.zip_code,
			v.default_distance_miles,
			COALESCE(array_agg(r.role_name ORDER BY r.role_name) FILTER (WHERE r.role_name IS NOT NULL), '{}') AS roles
		FROM volunteers v
		LEFT JOIN volunteer_roles vr ON vr.volunteer_id = v.volunteer_id
		LEFT JOIN roles r            ON r.role_id = vr.role_id
		WHERE v.is_active = TRUE
		GROUP BY v.volunteer_id
		ORDER BY v.last_name, v.first_name
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
		var roleNames pq.StringArray

		err := rows.Scan(
			&volInt,
			&v.FirstName,
			&v.LastName,
			&v.Email,
			&phone,
			&zip,
			&ddm,
			&roleNames)
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
		v.Roles = toModelRoles(roleNames)
		volunteers = append(volunteers, &v)
	}

	return volunteers, nil
}

func (s *VolunteerService) FetchOwnProfile(ctx context.Context, volId int) (*models.VolunteerView, error) {
	query := `
		SELECT
			v.first_name,
			v.last_name,
			v.email,
			v.phone,
			v.zip_code,
			v.default_distance_miles,
			COALESCE(array_agg(r.role_name ORDER BY r.role_name) FILTER (WHERE r.role_name IS NOT NULL), '{}') AS roles
		FROM volunteers v
		LEFT JOIN volunteer_roles vr ON vr.volunteer_id = v.volunteer_id
		LEFT JOIN roles r            ON r.role_id = vr.role_id
		WHERE v.volunteer_id = $1
		GROUP BY v.volunteer_id
	`
	var profile models.VolunteerView
	var phone, zip sql.NullString
	var ddm sql.NullInt32
	var roleNames pq.StringArray

	err := s.DB.QueryRowContext(ctx, query, volId).Scan(
		&profile.FirstName,
		&profile.LastName,
		&profile.Email,
		&phone,
		&zip,
		&ddm,
		&roleNames)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("volunteer not found")
	}
	if err != nil {
		return nil, fmt.Errorf("error querying volunteer: %w", err)
	}

	if phone.Valid {
		profile.Phone = &phone.String
	} else {
		profile.Phone = nil
	}
	if zip.Valid {
		profile.ZipCode = &zip.String
	} else {
		profile.ZipCode = nil
	}
	if ddm.Valid {
		dist := int(ddm.Int32)
		profile.Distance = &dist
	}
	profile.Roles = toModelRoles(roleNames)

	return &profile, nil
}

func (s *VolunteerService) FetchVolunteer(ctx context.Context, volId int) (*models.Volunteer, error) {

	query := `
		SELECT
			v.first_name,
			v.last_name,
			v.email,
			v.phone,
			v.zip_code,
			v.default_distance_miles,
			COALESCE(array_agg(r.role_name ORDER BY r.role_name) FILTER (WHERE r.role_name IS NOT NULL), '{}') AS roles
		FROM volunteers v
		LEFT JOIN volunteer_roles vr ON vr.volunteer_id = v.volunteer_id
		LEFT JOIN roles r            ON r.role_id = vr.role_id
		WHERE v.volunteer_id = $1
		GROUP BY v.volunteer_id
	`
	var profile models.Volunteer
	var phone, zip sql.NullString
	var ddm sql.NullInt32
	var roleNames pq.StringArray

	err := s.DB.QueryRowContext(ctx, query, volId).Scan(
		&profile.FirstName,
		&profile.LastName,
		&profile.Email,
		&phone,
		&zip,
		&ddm,
		&roleNames)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("volunteer not found")
	}
	if err != nil {
		return nil, fmt.Errorf("error querying volunteer: %w", err)
	}

	if phone.Valid {
		profile.Phone = &phone.String
	} else {
		profile.Phone = nil
	}
	if zip.Valid {
		profile.ZipCode = &zip.String
	} else {
		profile.ZipCode = nil
	}
	if ddm.Valid {
		dist := int(ddm.Int32)
		profile.Distance = &dist
	}
	profile.Roles = toModelRoles(roleNames)

	return &profile, nil
}

// These 2 functions get *all* of the information for each shift for a volunteer.
// The difference between them:
//  * FetchOwnShifts fetches shifts for a volunteer after confirming the caller is that volunteer.
//  * FetchVolunteerShifts confirms that caller is an admin.
// The only filtering is time:
//  * upcoming (>= NOW()),
//  * past (< NOW()), or
//  * all.

func (s *VolunteerService) FetchOwnShifts(ctx context.Context, volId int, filter models.ShiftsTimeFilter) ([]*models.VolunteerShiftView, error) {

	query := `
        SELECT
			sv.shift_id,
			sv.assigned_at,
			sv.cancelled_at,
			s.shift_start,
			s.shift_end,
			s.max_volunteers,
			jt.name,
			opp.opportunity_is_virtual,
			opp.pre_event_instructions,
            e.event_id,
            e.event_name,
            e.description,
            v.venue_name,
            v.street_address,
            v.city,
            v.state,
            v.zip_code,
			e.timezone
    	FROM volunteer_shifts sv
		JOIN shifts s ON s.shift_id = sv.shift_id
		JOIN opportunities opp ON opp.opportunity_id = s.opportunity_id
		LEFT JOIN job_types jt ON jt.job_type_id = opp.job_type_id
		JOIN events e ON e.event_id = opp.event_id
		LEFT JOIN venues v ON e.venue_id = v.venue_id
		WHERE sv.volunteer_id = $1
		AND sv.cancelled_at IS NULL
    `
	switch filter {
	case "UPCOMING":
		query += " AND s.shift_start >= NOW()"
	case "PAST":
		query += " AND s.shift_start < NOW()"
	case "ALL":
		// no filter
	}
	query += " ORDER BY s.shift_start"

	shiftRows, err := s.DB.QueryContext(ctx, query, volId)
	if err != nil {
		return nil, err
	}
	defer shiftRows.Close()

	shiftsMap := make(map[int]*models.VolunteerShiftView)

	for shiftRows.Next() {
		var volShift models.VolunteerShiftView
		var shiftInt, eventInt int
		var cancelledAt, preEventInst, eventDesc, timezone sql.NullString
		var venueName, streetAddress, city, state, zip sql.NullString
		var maxVols sql.NullInt64

		err := shiftRows.Scan(
			&shiftInt,
			&volShift.AssignedAt,
			&cancelledAt,
			&volShift.StartDateTime,
			&volShift.EndDateTime,
			&maxVols,
			&volShift.JobName,
			&volShift.IsVirtual,
			&preEventInst,
			&eventInt,
			&volShift.EventName,
			&eventDesc,
			&venueName,
			&streetAddress,
			&city,
			&state,
			&zip,
			&timezone,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning shift row: %w", err)
		}

		volShift.ShiftId = strconv.Itoa(shiftInt)
		volShift.EventId = strconv.Itoa(eventInt)
		if cancelledAt.Valid {
			volShift.CancelledAt = &cancelledAt.String
		} else {
			volShift.CancelledAt = nil
		}
		if maxVols.Valid {
			maxVolInt := int(maxVols.Int64)
			volShift.MaxVolunteers = &maxVolInt
		} else {
			volShift.MaxVolunteers = nil
		}
		if preEventInst.Valid {
			volShift.PreEventInstructions = &preEventInst.String
		} else {
			volShift.PreEventInstructions = nil
		}
		if eventDesc.Valid {
			volShift.EventDescription = &eventDesc.String
		} else {
			volShift.EventDescription = nil
		}
		if streetAddress.Valid {
			// If we have one of these, we must have them all.
			volShift.Venue = &models.VenueView{
				Address: streetAddress.String,
				City:    city.String,
				State:   state.String,
			}
			// Name and zip are optional.
			if venueName.Valid {
				volShift.Venue.Name = &venueName.String
			} else {
				volShift.Venue.Name = nil
			}
			if zip.Valid {
				volShift.Venue.ZipCode = &zip.String
			} else {
				volShift.Venue.ZipCode = nil
			}
		} else {
			volShift.Venue = nil
		}

		_, exists := shiftsMap[shiftInt]
		if !exists {
			shiftsMap[shiftInt] = &volShift
		}
	}

	// Convert map to slice
	shifts := make([]*models.VolunteerShiftView, 0, len(shiftsMap))
	for _, shift := range shiftsMap {
		shifts = append(shifts, shift)
	}

	return shifts, nil
}

func (s *VolunteerService) FetchVolunteerShifts(ctx context.Context, volId string, filter models.ShiftsTimeFilter) ([]*models.VolunteerShift, error) {

	volInt, err := strconv.Atoi(volId)
	if err != nil {
		return nil, fmt.Errorf("volunteer id is not valid: %w", err)
	}

	query := `
        SELECT
			sv.shift_id,
			sv.assigned_at,
			sv.cancelled_at,
			s.shift_start,
			s.shift_end,
			s.max_volunteers,
			jt.name,
			opp.opportunity_is_virtual,
			opp.pre_event_instructions,
            e.event_id,
            e.event_name,
            e.description,
            v.venue_name,
            v.street_address,
            v.city,
            v.state,
            v.zip_code,
			e.timezone
    	FROM volunteer_shifts sv
		JOIN shifts s ON s.shift_id = sv.shift_id
		JOIN opportunities opp ON opp.opportunity_id = s.opportunity_id
		LEFT JOIN job_types jt ON jt.job_type_id = opp.job_type_id
		JOIN events e ON e.event_id = opp.event_id
		LEFT JOIN venues v ON e.venue_id = v.venue_id
		WHERE sv.volunteer_id = $1
		AND sv.cancelled_at IS NULL
    `
	switch filter {
	case "UPCOMING":
		query += " AND s.shift_start >= NOW()"
	case "PAST":
		query += " AND s.shift_start < NOW()"
	case "ALL":
		// no filter
	}
	query += " ORDER BY s.shift_start"

	shiftRows, err := s.DB.QueryContext(ctx, query, volInt)
	if err != nil {
		return nil, err
	}
	defer shiftRows.Close()

	shiftsMap := make(map[int]*models.VolunteerShift)

	for shiftRows.Next() {
		var volShift models.VolunteerShift
		var shiftInt, eventInt int
		var cancelledAt, preEventInst, eventDesc, timezone sql.NullString
		var venueName, streetAddress, city, state, zip sql.NullString
		var maxVols sql.NullInt64

		err := shiftRows.Scan(
			&shiftInt,
			&volShift.AssignedAt,
			&cancelledAt,
			&volShift.StartDateTime,
			&volShift.EndDateTime,
			&maxVols,
			&volShift.JobName,
			&volShift.IsVirtual,
			&preEventInst,
			&eventInt,
			&volShift.EventName,
			&eventDesc,
			&venueName,
			&streetAddress,
			&city,
			&state,
			&zip,
			&timezone,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning shift row: %w", err)
		}

		volShift.ShiftId = strconv.Itoa(shiftInt)
		volShift.EventId = strconv.Itoa(eventInt)
		if cancelledAt.Valid {
			volShift.CancelledAt = &cancelledAt.String
		} else {
			volShift.CancelledAt = nil
		}
		if maxVols.Valid {
			maxVolInt := int(maxVols.Int64)
			volShift.MaxVolunteers = &maxVolInt
		} else {
			volShift.MaxVolunteers = nil
		}
		if preEventInst.Valid {
			volShift.PreEventInstructions = &preEventInst.String
		} else {
			volShift.PreEventInstructions = nil
		}
		if eventDesc.Valid {
			volShift.EventDescription = &eventDesc.String
		} else {
			volShift.EventDescription = nil
		}
		if streetAddress.Valid {
			// If we have one of these, we must have them all.
			volShift.Venue = &models.Venue{
				Address: streetAddress.String,
				City:    city.String,
				State:   state.String,
			}
			// Name and zip are optional.
			if venueName.Valid {
				volShift.Venue.Name = &venueName.String
			} else {
				volShift.Venue.Name = nil
			}
			if zip.Valid {
				volShift.Venue.ZipCode = &zip.String
			} else {
				volShift.Venue.ZipCode = nil
			}
		} else {
			volShift.Venue = nil
		}

		_, exists := shiftsMap[shiftInt]
		if !exists {
			shiftsMap[shiftInt] = &volShift
		}
	}

	// Convert map to slice
	shifts := make([]*models.VolunteerShift, 0, len(shiftsMap))
	for _, shift := range shiftsMap {
		shifts = append(shifts, shift)
	}

	return shifts, nil
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
			created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		RETURNING volunteer_id
	`

	var volInt int
	err = s.DB.QueryRowContext(ctx, query, newVol.FirstName, newVol.LastName, newVol.Email, newVol.Phone, newVol.ZipCode, newVol.Distance, lat, lng).Scan(&volInt)
	if err != nil {
		friendly := friendlyDBError(err)
		return &models.MutationResult{
			Success: false,
			Message: ptrString(friendly.Error()),
			ID:      nil,
		}, friendly
	}

	// Insert roles into the junction table.
	// ADMINISTRATOR always implies VOLUNTEER (business rule).
	rolesToInsert := []models.Role{models.RoleVolunteer}
	if newVol.Role == models.RoleAdministrator {
		rolesToInsert = append(rolesToInsert, models.RoleAdministrator)
	}
	for _, r := range rolesToInsert {
		_, err = s.DB.ExecContext(ctx, `
			INSERT INTO volunteer_roles (volunteer_id, role_id)
			SELECT $1, role_id FROM roles WHERE role_name = $2
		`, volInt, string(r))
		if err != nil {
			log.Printf("Warning: could not assign role %s to volunteer %d: %v", r, volInt, err)
		}
	}

	var role string
	if newVol.Role == models.RoleVolunteer {
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

	updateQuery := `
		UPDATE volunteers
		SET
			first_name             = $1,
			last_name              = $2,
			email                  = $3,
			phone                  = $4,
			zip_code               = $5,
			default_distance_miles = $6,
			latitude               = $7,
			longitude              = $8
		WHERE volunteer_id = $9
	`
	_, err = s.DB.ExecContext(ctx, updateQuery, profile.FirstName, profile.LastName, profile.Email, profile.Phone, profile.ZipCode, profile.Distance, lat, lng, volInt)
	if err != nil {
		return nil, friendlyDBError(err)
	}

	// Replace the volunteer's roles in the junction table.
	// Delete all current roles, then re-insert the requested set.
	// ADMINISTRATOR always implies VOLUNTEER.
	_, err = s.DB.ExecContext(ctx, "DELETE FROM volunteer_roles WHERE volunteer_id = $1", volInt)
	if err != nil {
		return nil, fmt.Errorf("could not clear volunteer roles: %w", err)
	}
	rolesToInsert := []models.Role{models.RoleVolunteer}
	if profile.Role == models.RoleAdministrator {
		rolesToInsert = append(rolesToInsert, models.RoleAdministrator)
	}
	for _, r := range rolesToInsert {
		_, err = s.DB.ExecContext(ctx, `
			INSERT INTO volunteer_roles (volunteer_id, role_id)
			SELECT $1, role_id FROM roles WHERE role_name = $2
		`, volInt, string(r))
		if err != nil {
			return nil, fmt.Errorf("could not assign role %s: %w", r, err)
		}
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

// toModelRoles converts a pq.StringArray of role name strings into a
// []models.Role slice.
func toModelRoles(names pq.StringArray) []models.Role {
	roles := make([]models.Role, len(names))
	for i, n := range names {
		roles[i] = models.Role(n)
	}
	return roles
}
