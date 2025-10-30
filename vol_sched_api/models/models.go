package models

import "time"

type Event struct {
	ID          int
	Name        string
	Description *string
	IsVirtual   bool
	LocationID  *int
	Location    *Location
	Shifts      []*Shift
}

type Location struct {
	ID      int
	Name    *string
	Address string
	City    string
	State   string
	ZipCode *string
}

type Shift struct {
	ID            int
	Role          string
	StartTime     time.Time
	EndTime       time.Time
	OpportunityID int
}

type EventFilter struct {
	Cities    []string
	EventType *string
	Roles     []string
	StartDate *string
	EndDate   *string
}
