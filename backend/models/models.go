package models

// Output types

type LookupValues struct {
	FundingEntities []*FundingEntity
	ServiceTypes    []*ServiceType
	JobTypes        []*JobType
	Cities          []string
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
