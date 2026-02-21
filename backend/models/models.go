package models

type Event struct {
	ID            string
	Name          string
	Description   *string
	EventType     EventType
	Venue         *Venue
	Shifts        []*Shift
	Opportunities []*Opportunity
}

type Venue struct {
	Name    *string
	Address string
	City    string
	State   string
	ZipCode *string
}

type Shift struct {
	ID                 string
	Job                Job
	Date               string
	StartTime          string
	EndTime            string
	MaxVolunteers      *int
	AssignedVolunteers []*VolunteerProfile
}

type Opportunity struct {
	ID     string
	Job    Job
	Shifts []*Shift
}

type VolunteerProfile struct {
	ID           string
	FirstName    string
	LastName     string
	Email        string
	ServiceTypes []ServiceType
}

// Input types
type NewVolunteerInput struct {
	FirstName    string
	LastName     string
	Email        string
	Phone        string
	ZipCode      string
	ServiceTypes []ServiceType
}

type UpdateVolunteerInput struct {
	ID           string
	FirstName    *string
	LastName     *string
	Email        *string
	Phone        *string
	ZipCode      *string
	ServiceTypes []ServiceType
}

type EventFilterInput struct {
	Cities    []string
	EventType *EventType
	Jobs      []Job
	StartDate *string
	EndDate   *string
}

type VolunteerFilterInput struct {
	FirstName    *string
	LastName     *string
	ServiceTypes []ServiceType
}

// Input types for creating events
type NewEventInput struct {
	Name          string
	Description   *string
	EventType     EventType
	Venue         *VenueInput
	Opportunities []*NewOpportunityInput
}

type VenueInput struct {
	Name    *string
	Address string
	City    string
	State   string
	ZipCode *string
}

type NewOpportunityInput struct {
	Job    Job
	Shifts []*NewShiftInput
}

type NewShiftInput struct {
	Date          string
	StartTime     string
	EndTime       string
	MaxVolunteers *int
}

// Enums
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

// Result types
type UpdateResult struct {
	Success bool
	Message *string
}

type InsertResult struct {
	Success bool
	Message *string
	ID      *string
}
