package services

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"volunteer-scheduler/models"
)

type StaffService struct {
	DB *sql.DB
}

func NewStaffService(db *sql.DB) *StaffService {
	return &StaffService{DB: db}
}

// FetchAllStaff returns all staff members ordered by last name then first name.
func (s *StaffService) FetchAllStaff(ctx context.Context) ([]*models.Staff, error) {
	query := `
		SELECT 
			staff_id, 
			first_name, 
			last_name, 
			email, 
			phone, 
			position
		FROM staff
		ORDER BY last_name, first_name
	`
	rows, err := s.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error fetching staff: %w", err)
	}
	defer rows.Close()

	var result []*models.Staff
	for rows.Next() {
		var s models.Staff
		var staffInt int
		var phone, position sql.NullString

		err = rows.Scan(
			&staffInt,
			&s.FirstName,
			&s.LastName,
			&s.Email,
			&phone,
			&position,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning staff row: %w", err)
		}

		s.ID = strconv.Itoa(staffInt)
		if phone.Valid {
			s.Phone = &phone.String
		}
		if position.Valid {
			s.Position = &position.String
		}
		result = append(result, &s)
	}

	return result, rows.Err()
}

// CreateStaff inserts a new staff member.
func (s *StaffService) CreateStaff(ctx context.Context, staff models.NewStaffInput) (*models.MutationResult, error) {

	var staffInt int

	insert := `
		INSERT INTO staff (
			first_name, 
			last_name, 
			email, 
			phone, 
			position)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING staff_id
	`
	err := s.DB.QueryRowContext(ctx, insert, staff.FirstName, staff.LastName, staff.Email, staff.Phone, staff.Position).Scan(&staffInt)
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to created staff member."),
			ID:      nil,
		}, err
	}

	id := strconv.Itoa(staffInt)
	return &models.MutationResult{
		Success: true,
		Message: ptrString("Sucessfully created staff member."),
		ID:      &id,
	}, nil
}

// UpdateStaff updates an existing staff member.
func (s *StaffService) UpdateStaff(ctx context.Context, staff models.UpdateStaffInput) (*models.MutationResult, error) {
	staffInt, err := strconv.Atoi(staff.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid staff id: %w", err)
	}

	update := `
		UPDATE staff
		SET first_name = $1,
		    last_name  = $2,
		    email      = $3,
		    phone      = $4,
		    position   = $5
		WHERE staff_id = $6
	`
	result, err := s.DB.ExecContext(ctx, update, staff.FirstName, staff.LastName, staff.Email, staff.Phone, staff.Position, staffInt)
	if err != nil {
		return nil, fmt.Errorf("error updating staff member: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return nil, fmt.Errorf("staff member %s not found", staff.ID)
	}

	return &models.MutationResult{
		Success: true,
		Message: ptrString("Successfully updated staff member"),
		ID:      &staff.ID,
	}, nil
}

// DeleteStaff hard-deletes a staff member.
// Shifts that referenced this staff member will have staff_contact_id set to NULL
// automatically by the database (ON DELETE SET NULL).
func (s *StaffService) DeleteStaff(ctx context.Context, staffID string) (*models.MutationResult, error) {
	staffInt, err := strconv.Atoi(staffID)
	if err != nil {
		return nil, fmt.Errorf("invalid staff id: %w", err)
	}

	result, err := s.DB.ExecContext(ctx, "DELETE FROM staff WHERE staff_id = $1", staffInt)
	if err != nil {
		return nil, fmt.Errorf("error deleting staff member: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return nil, fmt.Errorf("staff member %s not found", staffID)
	}

	return &models.MutationResult{
		Success: true,
		Message: ptrString("Successfully deleted staff member"),
		ID:      &staffID,
	}, nil
}
