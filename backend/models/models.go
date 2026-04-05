package models

// Output types

type LookupValues struct {
	Regions      []*Region
	ServiceTypes []*ServiceType
	JobTypes     []*JobType
}

// Result types (API-wide).

type MutationResult struct {
	Success bool
	Message *string
	ID      *string
}

type VolunteerMutationResult struct {
	Success bool
	Message *string
}

// Enums (system-wide).

type Role string

const (
	RoleVolunteer     Role = "VOLUNTEER"
	RoleAdministrator Role = "ADMINISTRATOR"
)
