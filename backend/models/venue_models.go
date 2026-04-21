package models

// Output.

type Venue struct {
	ID       string
	Name     *string
	Address  string
	City     string
	State    string
	ZipCode  *string
	Timezone string
}

// Input for new elements.

type NewVenueInput struct {
	Name     *string
	Address  string
	City     string
	State    string
	ZipCode  *string
	IanaZone string
}

// Input types for updating.

type UpdateVenueInput struct {
	ID       string
	Name     *string
	Address  string
	City     string
	State    string
	ZipCode  *string
	IanaZone string
}
