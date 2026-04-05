package models

// Output.

type Region struct {
	ID       int
	Code     string
	Name     string
	IsActive bool
}

type Venue struct {
	ID       string
	Name     *string
	Address  string
	City     string
	State    string
	ZipCode  *string
	Timezone string
	Region   []int
}

// Input for new elements.

type NewRegionInput struct {
	Code string
	Name string
}

type NewVenueInput struct {
	Name     *string
	Address  string
	City     string
	State    string
	ZipCode  *string
	IanaZone string
	Region   []int
}

// Input types for updating.

type UpdateRegionInput struct {
	ID   int
	Code string
	Name string
}

type UpdateVenueInput struct {
	ID       string
	Name     *string
	Address  string
	City     string
	State    string
	ZipCode  *string
	IanaZone string
}
