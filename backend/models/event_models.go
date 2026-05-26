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
	EventDates     []*EventDate
	Timezone       string
	ServiceTypes   []string
	ShiftSummaries []*EventShiftSummary
}

type ManagedEvent struct {
	ID                       string
	Name                     string
	Description              *string
	EventType                EventType
	Venue                    *Venue
	EventDates               []*EventDate
	Timezone                 string
	FundingEntity            FundingEntity
	ServiceTypes             []string
	ShiftSummaries           []*EventShiftSummary
	RecurrenceId             string
	RecurrenceOrder          int
	RecurrencePattern        string
	RecurrenceMaxOccurrences *int
	RecurrenceOrdinal        *string
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
}

// Input types for queries (e.g., filters).

type EventFilterInput struct {
	Cities    []string
	Distance  *int
	EventType *EventType
	Jobs      []int
	TimeFrame *ShiftsTimeFilter
}

//  Input types for new rows.

type RecurrenceInput struct {
	Pattern        RecurrencePattern
	MaxOccurrences *int
	WeekdayOrdinal *WeekdayOrdinal
}
type NewEventInput struct {
	Name            string
	Description     *string
	EventType       EventType
	VenueId         *string
	Timezone        string
	FundingEntityID int
	ServiceTypes    []int
	EventDates      []*NewEventDateInput
	Recurrence      *RecurrenceInput
}

type NewEventDateInput struct {
	StartDateTime string
	EndDateTime   string
}

type AddEventDateInput struct {
	EventID       string
	StartDateTime string
	EndDateTime   string
}

//  Input types for updates/deletes.

type UpdateEventInput struct {
	ID              string
	Name            string
	Description     *string
	EventType       EventType
	VenueId         *string
	Timezone        string
	FundingEntityID int
	ServiceTypes    []int
	RecurrenceScope *RecurrenceUpdateScope
}

type UpdateEventDateInput struct {
	ID            string
	StartDateTime string
	EndDateTime   string
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

type RecurrencePattern string

const (
	RecurrencePatternDaily    RecurrencePattern = "DAILY"
	RecurrencePatternWeekly   RecurrencePattern = "WEEKLY"
	RecurrencePatternBiweekly RecurrencePattern = "BIWEEKLY"
	RecurrencePatternMonthly  RecurrencePattern = "MONTHLY"
	RecurrencePatternYearly   RecurrencePattern = "YEARLY"
)

type WeekdayOrdinal string

const (
	WeekdayOrdinalFirst  WeekdayOrdinal = "FIRST"
	WeekdayOrdinalSecond WeekdayOrdinal = "SECOND"
	WeekdayOrdinalThird  WeekdayOrdinal = "THIRD"
	WeekdayOrdinalFourth WeekdayOrdinal = "FOURTH"
	WeekdayOrdinalLast   WeekdayOrdinal = "LAST"
)

type RecurrenceUpdateScope string

const (
	RecurrenceUpdateScopeThisOnly      RecurrenceUpdateScope = "THIS_ONLY"
	RecurrenceUpdateScopeThisAndFuture RecurrenceUpdateScope = "THIS_AND_FUTURE"
)
