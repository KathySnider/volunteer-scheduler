package services

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"volunteer-scheduler/models"
)

type VenueService struct {
	DB *sql.DB
}

func NewVenueService(db *sql.DB) *VenueService {
	return &VenueService{DB: db}
}

// Queries.

func (s *VenueService) FetchVenues(ctx context.Context) ([]*models.Venue, error) {

	query := `
        SELECT
			venues.venue_id,
            venue_name,
            street_address,
            city,
            state,
            zip_code,
			timezone
        FROM venues
    `

	rows, err := s.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error querying venues: %w", err)
	}
	defer rows.Close()

	venues := make([]*models.Venue, 0)

	for rows.Next() {
		var venue models.Venue
		var venueInt int
		var name, zip sql.NullString

		err := rows.Scan(
			&venueInt,
			&name,
			&venue.Address,
			&venue.City,
			&venue.State,
			&zip,
			&venue.Timezone,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning venue: %w", err)
		}

		venue.ID = strconv.Itoa(venueInt)

		// Name and zip are nullable.
		venue.Name = &name.String
		venue.ZipCode = &zip.String

		venues = append(venues, &venue)
	}

	return venues, nil
}

// Create.

func (s *VenueService) CreateVenue(ctx context.Context, newVenue models.NewVenueInput) (*models.MutationResult, error) {

	query := `
		INSERT INTO venues (
			venue_name,
			street_address,
			city,
			state,
			zip_code,
			timezone)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING venue_id
	`

	var venueInt int
	err := s.DB.QueryRowContext(ctx, query, newVenue.Name, newVenue.Address, newVenue.City, newVenue.State, newVenue.ZipCode, newVenue.IanaZone).Scan(&venueInt)
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to create new venue."),
			ID:      nil,
		}, err
	}

	return &models.MutationResult{
		Success: true,
		Message: ptrString("Venue successfully created."),
		ID:      ptrString(strconv.Itoa(venueInt)),
	}, nil

}

// Update, delete.

func (s *VenueService) UpdateVenue(ctx context.Context, venue models.UpdateVenueInput) (*models.MutationResult, error) {

	venueId, err := strconv.Atoi(venue.ID)
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Invalid venue.ID."),
			ID:      &venue.ID,
		}, err
	}

	update := `
		UPDATE venues
		SET
			venue_name = $1,
			street_address = $2,
			city = $3,
			state = $4,
			zip_code = $5,
			timezone = $6
		WHERE venue_id = $7
	`
	_, err = s.DB.ExecContext(ctx, update, venue.Name, venue.Address, venue.City, venue.State, venue.ZipCode, venue.IanaZone, venueId)

	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to update venue."),
			ID:      &venue.ID,
		}, err
	}

	return &models.MutationResult{
		Success: true,
		Message: ptrString("Venue successfully updated."),
		ID:      &venue.ID,
	}, nil

}

func (s *VenueService) DeleteVenue(ctx context.Context, venueId string) (*models.MutationResult, error) {

	venueInt, err := strconv.Atoi(venueId)
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Invalid venue ID."),
			ID:      &venueId,
		}, err
	}

	_, err = s.DB.ExecContext(ctx, "DELETE FROM venues WHERE venue_id = $1", venueInt)

	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to delete venue."),
			ID:      &venueId,
		}, err
	}

	return &models.MutationResult{
		Success: true,
		Message: ptrString("Venue successfully deleted."),
		ID:      &venueId,
	}, nil
}
