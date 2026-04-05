package models

// Output types.

type Staff struct {
	ID        string
	FirstName string
	LastName  string
	Email     string
	Phone     *string
	Position  *string
}

// Input types for new elements.

type NewStaffInput struct {
	FirstName string
	LastName  string
	Email     string
	Phone     *string
	Position  *string
}

// Input types for updates.

type UpdateStaffInput struct {
	ID        string
	FirstName string
	LastName  string
	Email     string
	Phone     *string
	Position  *string
}
