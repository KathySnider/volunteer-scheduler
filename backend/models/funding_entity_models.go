package models

type FundingEntity struct {
	ID          int
	Name        string
	Description string
}

type FundingEntityFilter struct {
	ActiveOnly bool
}

type NewFundingEntityInput struct {
	Name        string
	Description *string
}

type UpdateFundingEntityInput struct {
	ID          int
	Name        *string
	Description *string
}
