package models

// Output types.

type JobType struct {
	ID        int
	Code      string
	Name      string
	SortOrder int
	IsActive  bool
}

type Shift struct {
	ID             string
	StartDateTime  string
	EndDateTime    string
	MaxVolunteers  *int
	StaffContactId *string
}

// Flattened view for volunteers; combines
// opportunity and shifts.

type ShiftView struct {
	ID                 string
	JobName            string
	StartDateTime      string
	EndDateTime        string
	IsVirtual          bool
	MaxVolunteers      *int
	AssignedVolunteers int
}

type Opportunity struct {
	ID                   string
	JobId                int
	IsVirtual            bool
	PreEventInstructions *string
	Shifts               []*Shift
}

// Input types for new elements.

type NewJobTypeInput struct {
	Code      string
	Name      string
	SortOrder int
}

type NewShiftInput struct {
	StartDateTime  string
	EndDateTime    string
	IanaZone       string
	MaxVolunteers  *int
	StaffContactId *string
}

type AddShiftInput struct {
	OppId          string
	StartDateTime  string
	EndDateTime    string
	IanaZone       string
	MaxVolunteers  *int
	StaffContactId *string
}

type NewOpportunityInput struct {
	EventId              string
	JobId                int
	IsVirtual            bool
	PreEventInstructions *string
	Shifts               []*NewShiftInput
}

// Input types for updates.

type UpdateJobTypeInput struct {
	ID        int
	Code      string
	Name      string
	SortOrder int
}

type UpdateShiftInput struct {
	ID             string
	StartDateTime  string
	EndDateTime    string
	IanaZone       string
	MaxVolunteers  *int
	StaffContactId *string
}

type UpdateOpportunityInput struct {
	ID                   string
	JobId                int
	IsVirtual            bool
	PreEventInstructions *string
}
