package models

// Output types

// Volunteers

// Any user can see own profile (sans ID).
type VolunteerProfile struct {
	FirstName string
	LastName  string
	Email     string
	Phone     *string
	ZipCode   *string
	Role      Role
}

type VolunteerShift struct {
	ShiftId              string
	AssignedAt           string
	CancelledAt          *string
	StartDateTime        string
	EndDateTime          string
	MaxVolunteers        *int
	Job                  Job
	OtherJobDescription  *string
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
	Role      Role
}

// Venues (include regions)

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

// Opportunities and Shifts

type Opportunity struct {
	ID                   string
	Job                  Job
	OtherJobDescription  *string
	IsVirtual            bool
	PreEventInstructions *string
	Shifts               []*Shift
}

type ShiftView struct {
	ID                  string
	Job                 Job
	OtherJobDescription *string
	StartDateTime       string
	EndDateTime         string
	IsVirtual           bool
	MaxVolunteers       *int
	AssignedVolunteers  int
}

type Shift struct {
	ID             string
	StartDateTime  string
	EndDateTime    string
	MaxVolunteers  *int
	StaffContactId *string
}

// Events

// When showing events to users, get the whole thing all at once
// (venue, dates, etc.)

type Event struct {
	ID           string
	Name         string
	Description  *string
	EventType    EventType
	Venue        *Venue
	ServiceTypes []ServiceType
	EventDates   []*EventDate
}

type EventDate struct {
	ID            string
	StartDateTime string
	EndDateTime   string
	IanaZone      string
}

// Input types for queries (e.g., filters).

type EventFilterInput struct {
	Regions        []int
	EventType      *EventType
	Jobs           []Job
	ShiftStartDate *string
	ShiftEndDate   *string
	IanaZone       *string
}

type VolunteerFilterInput struct {
	FirstName *string
	LastName  *string
	Email     *string
}

// Input types for new elements.
type NewVolunteerInput struct {
	FirstName string
	LastName  string
	Email     string
	Phone     *string
	ZipCode   *string
	Role      Role
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

type NewRegionInput struct {
	Code string
	Name string
}

type NewEventInput struct {
	Name         string
	Description  *string
	EventType    EventType
	VenueId      *string
	ServiceTypes []ServiceType
	EventDates   []*NewEventDateInput
}

type NewEventDateInput struct {
	StartDateTime string
	EndDateTime   string
	IanaZone      string
}

type AddEventDateInput struct {
	EventID       string
	StartDateTime string
	EndDateTime   string
	IanaZone      string
}

type NewOpportunityInput struct {
	EventId              string
	Job                  Job
	OtherJobDescription  *string
	IsVirtual            bool
	PreEventInstructions *string
	Shifts               []*NewShiftInput
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

// Input types for updating.
type UpdateOwnProfileInput struct {
	FirstName string
	LastName  string
	Email     string
	Phone     *string
	ZipCode   *string
}

type UpdateVolunteerInput struct {
	ID        string
	FirstName string
	LastName  string
	Email     string
	Phone     *string
	ZipCode   *string
	Role      Role
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

type UpdateRegionInput struct {
	ID   int
	Code string
	Name string
}

type UpdateEventInput struct {
	ID           string
	Name         string
	Description  *string
	EventType    EventType
	VenueId      *string
	ServiceTypes []ServiceType
}

type UpdateEventDateInput struct {
	ID            string
	StartDateTime string
	EndDateTime   string
	IanaZone      string
}

type UpdateOpportunityInput struct {
	ID                   string
	Job                  Job
	OtherJobDescription  *string
	IsVirtual            bool
	PreEventInstructions *string
}

type UpdateShiftInput struct {
	ID             string
	StartDateTime  string
	EndDateTime    string
	IanaZone       string
	MaxVolunteers  *int
	StaffContactId *string
}

// Enums
type Role string

const (
	RoleVolunteer     Role = "VOLUNTEER"
	RoleAdministrator Role = "ADMINISTRATOR"
)

type EventType string

const (
	EventTypeVirtual  EventType = "VIRTUAL"
	EventTypeInPerson EventType = "IN_PERSON"
	EventTypeHybrid   EventType = "HYBRID"
)

type ServiceType string

const (
	ServiceTypeOutreach       ServiceType = "OUTREACH"
	ServiceTypeAdvocacy       ServiceType = "ADVOCACY"
	ServiceTypeSpeakersBureau ServiceType = "SPEAKERS_BUREAU"
	ServiceTypeOfficeSupport  ServiceType = "OFFICE_SUPPORT"
	ServiceTypeOther          ServiceType = "OTHER"
)

type Job string

const (
	JobEventSupport  Job = "EVENT_SUPPORT"
	JobAdvocacy      Job = "ADVOCACY"
	JobSpeaker       Job = "SPEAKER"
	JobVolunteerLead Job = "VOLUNTEER_LEAD"
	JobAttendeeOnly  Job = "ATTENDEE_ONLY"
	JobOther         Job = "OTHER"
)

type ShiftsTimeFilter string

const (
	ShiftsFilterUpcoming ShiftsTimeFilter = "UPCOMING"
	ShiftsFilterPast     ShiftsTimeFilter = "PAST"
	ShiftsFilterAll      ShiftsTimeFilter = "ALL"
)

// Result types

type MutationResult struct {
	Success bool
	Message *string
	ID      *string
}

type VolunteerMutationResult struct {
	Success bool
	Message *string
}
