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
			timezone,
			vr.region_id
        FROM venues
		LEFT JOIN venue_regions vr ON vr.venue_id = venues.venue_id
    `

	rows, err := s.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error querying venues: %w", err)
	}
	defer rows.Close()

	venuesMap := make(map[int]*models.Venue)

	for rows.Next() {
		var venue models.Venue
		var venueInt int
		var regionId sql.NullInt32
		var name, zip sql.NullString

		err := rows.Scan(
			&venueInt,
			&name,
			&venue.Address,
			&venue.City,
			&venue.State,
			&zip,
			&venue.Timezone,
			&regionId,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning venue: %w", err)
		}

		venue.ID = strconv.Itoa(venueInt)

		// Name and zip are nullable.
		venue.Name = &name.String
		venue.ZipCode = &zip.String

		// Duplicate rows will exist when a venue is in multiple regions.
		_, exists := venuesMap[venueInt]
		if !exists {
			venuesMap[venueInt] = &venue
		}
		if regionId.Valid {
			regionInt := int(regionId.Int32)
			venuesMap[venueInt].Region = append(venuesMap[venueInt].Region, regionInt)
		}
	}

	venues := make([]*models.Venue, 0, len(venuesMap))
	for _, v := range venuesMap {
		venues = append(venues, v)
	}
	return venues, nil
}

// Create.

func (s *VenueService) CreateVenue(ctx context.Context, newVenue models.NewVenueInput) (*models.MutationResult, error) {

	// A venue requires at least one region.
	if len(newVenue.Region) == 0 {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("A new venue must be in at least one region."),
			ID:      nil,
		}, nil
	}

	// Add the venue and it's regions inside a transaction.
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		err = fmt.Errorf("error starting transaction: %w", err)
		return nil, err
	}
	// Defer a rollback in case anything fails.
	defer tx.Rollback()

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
	err = tx.QueryRowContext(ctx, query, newVenue.Name, newVenue.Address, newVenue.City, newVenue.State, newVenue.ZipCode, newVenue.IanaZone).Scan(&venueInt)
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to create new venue."),
			ID:      nil,
		}, err
	}

	// Now we have the venue Id; we can create the relationships
	// with the region(s).
	insert := `
		INSERT INTO venue_regions (
			venue_id, 
			region_id) 
		VALUES ($1, $2)
	`
	for _, regionId := range newVenue.Region {
		_, err = tx.ExecContext(ctx, insert, venueInt, regionId)
		if err != nil {
			regStr := strconv.Itoa(regionId)
			return &models.MutationResult{
				Success: false,
				Message: ptrString("Failed to create add venue: could not add region."),
				ID:      &regStr,
			}, err
		}
	}

	// All good. Commit and return the new venue ID.
	err = tx.Commit()
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("error committing transaction"),
			ID:      nil,
		}, err
	}
	return &models.MutationResult{
		Success: true,
		Message: ptrString("Venue successfully created."),
		ID:      ptrString(strconv.Itoa(venueInt)),
	}, nil

}

func (s *VenueService) CreateRegion(ctx context.Context, newRegion models.NewRegionInput) (*models.MutationResult, error) {

	query := `
		INSERT INTO regions (
			code,
			name)
		VALUES ($1, $2)
		RETURNING region_id
	`

	var regionId int
	err := s.DB.QueryRowContext(ctx, query, newRegion.Code, newRegion.Name).Scan(&regionId)

	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to create new region."),
			ID:      nil,
		}, err
	}

	return &models.MutationResult{
		Success: true,
		Message: ptrString("Region successfully created."),
		ID:      ptrString(strconv.Itoa(regionId)),
	}, nil

}

// Update, del

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

func (s *VenueService) UpdateRegion(ctx context.Context, region models.UpdateRegionInput) (*models.MutationResult, error) {
	regStr := strconv.Itoa(region.ID)

	update := `
		UPDATE regions 
		SET 
			code = $1,
			name = $2
		WHERE region_id = $3
	`
	_, err := s.DB.ExecContext(ctx, update, region.Code, region.Name, region.ID)

	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to update region."),
			ID:      &regStr,
		}, err
	}

	return &models.MutationResult{
		Success: true,
		Message: ptrString("Region successfully updated."),
		ID:      &regStr,
	}, nil
}

func (s *VenueService) AddVenueRegion(ctx context.Context, venueId int, regionId int) (*models.MutationResult, error) {
	insert := `
		INSERT INTO venue_regions (venue_id, region_id)
		VALUES ($1, $2)
		`
	_, err := s.DB.ExecContext(ctx, insert, venueId, regionId)
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to add region to venue."),
			ID:      nil,
		}, err
	}

	regStr := strconv.Itoa(regionId)
	return &models.MutationResult{
		Success: true,
		Message: ptrString("Successfully added region to venue."),
		ID:      &regStr,
	}, nil
}

func (s *VenueService) RemoveVenueRegion(ctx context.Context, venueId int, regionId int) (*models.MutationResult, error) {

	// A venue must have at least one region. Make sure this is
	// not the last region for this venue.
	query := `
		SELECT
			venue_id,
			COUNT (region_id) as region_count
		FROM venue_regions
		WHERE venue_id = $1	
		GROUP BY venue_id
	`

	var venId, regCount int
	err := s.DB.QueryRowContext(ctx, query, venueId).Scan(&venId, &regCount)
	if err != nil {
		venStr := strconv.Itoa(venueId)
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to remove region from venue: failed query."),
			ID:      &venStr,
		}, err
	}
	if regCount < 2 {
		venStr := strconv.Itoa(venueId)
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to removed region from venue: venue must have at least one region."),
			ID:      &venStr,
		}, nil
	}

	delete := `
		DELETE FROM venue_regions 
		WHERE venue_id = $1 AND region_id = $2
		`
	_, err = s.DB.ExecContext(ctx, delete, venueId, regionId)
	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to remove region from venue."),
			ID:      nil,
		}, err
	}

	regStr := strconv.Itoa(regionId)
	return &models.MutationResult{
		Success: true,
		Message: ptrString("Successfully removed region from venue."),
		ID:      &regStr,
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

// Soft delete. Preserve region for history.
func (s *VenueService) DeleteRegion(ctx context.Context, regionId int) (*models.MutationResult, error) {
	regStr := strconv.Itoa(regionId)

	_, err := s.DB.ExecContext(ctx, "UPDATE regions SET is_active = false WHERE region_id = $1", regionId)

	if err != nil {
		return &models.MutationResult{
			Success: false,
			Message: ptrString("Failed to delete region."),
			ID:      &regStr,
		}, err
	}

	return &models.MutationResult{
		Success: true,
		Message: ptrString("Successfully deleted region."),
		ID:      &regStr,
	}, nil
}
