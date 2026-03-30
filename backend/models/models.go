package models

// Output types

type LookupValues struct {
	Regions      []*Region
	ServiceTypes []*ServiceType
	JobTypes     []*JobType
}

// Staff

type Staff struct {
	ID        string
	FirstName string
	LastName  string
	Email     string
	Phone     *string
	Position  *string
}

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

type JobType struct {
	ID        int
	Code      string
	Name      string
	SortOrder int
	IsActive  bool
}
type Opportunity struct {
	ID                   string
	JobId                int
	IsVirtual            bool
	PreEventInstructions *string
	Shifts               []*Shift
}

type ShiftView struct {
	ID                 string
	JobName            string
	StartDateTime      string
	EndDateTime        string
	IsVirtual          bool
	MaxVolunteers      *int
	AssignedVolunteers int
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

type ServiceType struct {
	ID       int
	Code     string
	Name     string
	IsActive bool
}

type Event struct {
	ID           string
	Name         string
	Description  *string
	EventType    EventType
	Venue        *Venue
	ServiceTypes []string
	EventDates   []*EventDate
}

type EventDate struct {
	ID            string
	StartDateTime string
	EndDateTime   string
	IanaZone      string
}

// Feedback

// Show admins all of the notes complete with who
// wrote each and when.

type FeedbackNote struct {
	ID        string
	Creator   string
	CreatedAt string
	Note      string
}

type Feedback struct {
	ID             string
	VolunteerName  string
	Type           FeedbackType
	Status         FeedbackStatus
	Subject        string
	AppPageName    string
	Text           string
	Notes          []*FeedbackNote
	GithubIssueURL *string
	CreatedAt      string
	LastUpdatedAt  *string
	ResolvedAt     *string
}

// Input types for queries (e.g., filters).

type EventFilterInput struct {
	Regions        []int
	EventType      *EventType
	Jobs           []int
	ShiftStartDate *string
	ShiftEndDate   *string
	IanaZone       *string
}

type VolunteerFilterInput struct {
	FirstName *string
	LastName  *string
	Email     *string
}

type FeedbackFilterInput struct {
	Status *FeedbackStatus
	Type   *FeedbackType
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

type NewStaffInput struct {
	FirstName string
	LastName  string
	Email     string
	Phone     *string
	Position  *string
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
	ServiceTypes []int
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

type NewJobInput struct {
	Code      string
	Name      string
	SortOrder int
}

type NewOpportunityInput struct {
	EventId              string
	JobId                int
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

type NewFeedbackInput struct {
	Type        FeedbackType
	Subject     string
	AppPageName string
	Text        string
}

// Input types for updating.
type UpdateOwnProfileInput struct {
	FirstName string
	LastName  string
	Email     string
	Phone     *string
	ZipCode   *string
}

type UpdateStaffInput struct {
	ID        string
	FirstName string
	LastName  string
	Email     string
	Phone     *string
	Position  *string
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
	ServiceTypes []int
}

type UpdateEventDateInput struct {
	ID            string
	StartDateTime string
	EndDateTime   string
	IanaZone      string
}

type UpdateJobInput struct {
	ID        int
	Code      string
	Name      string
	SortOrder int
}

type UpdateOpportunityInput struct {
	ID                   string
	JobId                int
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

type QuestionFeedbackInput struct {
	ID        string
	EmailText string
	Note      string
}

type UpdateFeedbackInput struct {
	ID             string
	Status         FeedbackStatus
	Note           string
	GithubIssueURL *string
}

type ResolveFeedbackInput struct {
	ID             string
	Status         FeedbackStatus
	Note           string
	GithubIssueURL *string
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

type ShiftsTimeFilter string

const (
	ShiftsFilterUpcoming ShiftsTimeFilter = "UPCOMING"
	ShiftsFilterPast     ShiftsTimeFilter = "PAST"
	ShiftsFilterAll      ShiftsTimeFilter = "ALL"
)

type FeedbackType string

const (
	FeedbackTypeBug         FeedbackType = "BUG"
	FeedbackTypeEnhancement FeedbackType = "ENHANCEMENT"
	FeedbackTypeGeneral     FeedbackType = "GENERAL"
)

type FeedbackStatus string

const (
	FeedbackStatusOpen     FeedbackStatus = "OPEN"
	FeedbackStatusQuestion FeedbackStatus = "QUESTION_SENT"
	FeedbackStatusGithub   FeedbackStatus = "RESOLVED_GITHUB"
	FeedbackStatusRejected FeedbackStatus = "RESOLVED_REJECTED"
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
