package admin

import (
	"database/sql"
	"volunteer-scheduler/services"
)

// Resolver holds the services needed by GraphQL resolvers
type Resolver struct {
	DB               *sql.DB
	ShiftService     *services.ShiftService
	EventService     *services.EventService
	VolunteerService *services.VolunteerService
	VenueService     *services.VenueService
	FeedbackService  *services.FeedbackService
}
