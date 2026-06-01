package models

// Output types.

// Any user can see own profile (sans ID).
type VolunteerView struct {
	FirstName string
	LastName  string
	Email     string
	Phone     *string
	ZipCode   *string
	Distance  *int
	Roles     []Role
}

type VolunteerShift struct {
	ShiftId              string
	AssignedAt           string
	CancelledAt          *string
	StartDateTime        string
	EndDateTime          string
	MaxVolunteers        *int
	JobName              string
	IsVirtual            bool
	PreEventInstructions *string
	EventId              string
	EventName            string
	EventDescription     *string
	Venue                *Venue
}

// Admins can see/use ID.
type Volunteer struct {
	ID        string
	FirstName string
	LastName  string
	Email     string
	Phone     *string
	ZipCode   *string
	Distance  *int
	Roles     []Role
}

// Input types for queries (e.g., filters).

type VolunteerFilterInput struct {
	FirstName *string
	LastName  *string
	Email     *string
}

type VolunteerShiftView struct {
	ShiftId              string
	AssignedAt           string
	CancelledAt          *string
	StartDateTime        string
	EndDateTime          string
	MaxVolunteers        *int
	JobName              string
	IsVirtual            bool
	PreEventInstructions *string
	EventId              string
	EventName            string
	EventDescription     *string
	Venue                *VenueView
}

// Input for new elements.

type NewVolunteerInput struct {
	FirstName string
	LastName  string
	Email     string
	Phone     *string
	ZipCode   *string
	Distance  *int
	Role      Role
}

// Input for updates.

type UpdateVolunteerInput struct {
	ID        string
	FirstName string
	LastName  string
	Email     string
	Phone     *string
	ZipCode   *string
	Distance  *int
	Role      Role
}

type UpdateOwnProfileInput struct {
	FirstName string
	LastName  string
	Email     string
	Phone     *string
	ZipCode   *string
	Distance  *int
}
