package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"
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
		if name.Valid {
			venue.Name = &name.String
		} else {
			venue.Name = nil
		}
		if zip.Valid {
			venue.ZipCode = &zip.String
		} else {
			venue.ZipCode = nil
		}

		venues = append(venues, &venue)
	}

	return venues, nil
}

// Create.

func (s *VenueService) CreateVenue(ctx context.Context, newVenue models.NewVenueInput) (*models.MutationResult, error) {

	zip := ""
	if newVenue.ZipCode != nil {
		zip = *newVenue.ZipCode
	}
	lat, lng, err := GeocodeAddress(newVenue.Address, newVenue.City, newVenue.State, zip)
	if err != nil {
		log.Printf("Unable to get lat/lng for venue address: %v", err)
	}
	query := `
		INSERT INTO venues (
			venue_name,
			street_address,
			city,
			state,
			zip_code,
			latitude,
			longitude,
			timezone)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING venue_id
	`

	var venueInt int
	err = s.DB.QueryRowContext(ctx, query, newVenue.Name, newVenue.Address, newVenue.City, newVenue.State, newVenue.ZipCode, lat, lng, newVenue.IanaZone).Scan(&venueInt)
	if err != nil {
		return nil, friendlyDBError(err)
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
		return nil, fmt.Errorf("invalue venue id %s: %w", venue.ID, err)
	}

	zip := ""
	if venue.ZipCode != nil {
		zip = *venue.ZipCode
	}
	lat, lng, err := GeocodeAddress(venue.Address, venue.City, venue.State, zip)
	if err != nil {
		log.Printf("Unable to get lat/lng for venue address: %v", err)
	}

	update := `
		UPDATE venues
		SET
			venue_name = $1,
			street_address = $2,
			city = $3,
			state = $4,
			zip_code = $5,
			latitude = $6,
			longitude = $7,
			timezone = $8
		WHERE venue_id = $9
	`
	_, err = s.DB.ExecContext(ctx, update, venue.Name, venue.Address, venue.City, venue.State, venue.ZipCode, lat, lng, venue.IanaZone, venueId)

	if err != nil {
		return nil, friendlyDBError(err)
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
		friendly := friendlyDBError(err)
		return &models.MutationResult{
			Success: false,
			Message: ptrString(friendly.Error()),
			ID:      &venueId,
		}, friendly
	}

	return &models.MutationResult{
		Success: true,
		Message: ptrString("Venue successfully deleted."),
		ID:      &venueId,
	}, nil
}
