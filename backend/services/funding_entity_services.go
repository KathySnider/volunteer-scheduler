package services

import (
	"context"
	"database/sql"
)

type FundingEntityService struct {
	DB *sql.DB
}

func NewFundingEntityService(db *sql.DB) *FundingEntityService {
	return &FundingEntityService{DB: db}
}

// Create.

func (s *FundingEntityService) CreateFundingEntity(ctx context.Context, name string, description *string) (int, error) {
	query := `
		INSERT INTO funding_entities (name, description)
		VALUES ($1, $2)
		RETURNING id
	`

	var id int
	err := s.DB.QueryRowContext(ctx, query, name, description).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

// Update.

func (s *FundingEntityService) UpdateFundingEntity(ctx context.Context, id int, name *string, description *string) error {
	update := `
		UPDATE funding_entities
		SET
			name = COALESCE($2, name),
			description = COALESCE($3, description)
		WHERE id = $1
	`

	_, err := s.DB.ExecContext(ctx, update, id, name, description)
	return err
}

// Delete (soft).

func (s *FundingEntityService) DeleteFundingEntity(ctx context.Context, id int) error {
	_, err := s.DB.ExecContext(ctx, "UPDATE funding_entities SET is_active = false WHERE id = $1", id)
	return err
}
