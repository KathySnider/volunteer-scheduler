package volunteer

import (
	"database/sql"
	"volunteer-scheduler/services"
)

// Resolver holds the services needed by GraphQL resolvers
type Resolver struct {
	DB               *sql.DB
	EventService     *services.EventService
	VolunteerService *services.VolunteerService
	ShiftService     *services.ShiftService
	VenueService     *services.VenueService
}
