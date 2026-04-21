package models

// Output types

// Events

// When showing events to users, get the whole thing all at once
// (venue, dates, etc.)

type ServiceType struct {
	ID   int
	Code string
	Name string
}

type Event struct {
	ID             string
	Name           string
	Description    *string
	EventType      EventType
	Venue          *Venue
	FundingEntity  FundingEntity
	ServiceTypes   []string
	EventDates     []*EventDate
	ShiftSummaries []*EventShiftSummary
}

// EventShiftSummary holds the per-opportunity volunteer counts
// needed to render the event listing cards.
type EventShiftSummary struct {
	JobName            string
	AssignedVolunteers int
	MaxVolunteers      int
}

type EventDate struct {
	ID            string
	StartDateTime string
	EndDateTime   string
	IanaZone      string
}

// Input types for queries (e.g., filters).

type EventFilterInput struct {
	Cities    []string
	EventType *EventType
	Jobs      []int
	TimeFrame *ShiftsTimeFilter
}

//  Input types for new rows.

type NewEventInput struct {
	Name             string
	Description      *string
	EventType        EventType
	VenueId          *string
	FundingEntityID  int
	ServiceTypes     []int
	EventDates       []*NewEventDateInput
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

//  Input types for updates.

type UpdateEventInput struct {
	ID              string
	Name            string
	Description     *string
	EventType       EventType
	VenueId         *string
	FundingEntityID int
	ServiceTypes    []int
}

type UpdateEventDateInput struct {
	ID            string
	StartDateTime string
	EndDateTime   string
	IanaZone      string
}

// Enums
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
