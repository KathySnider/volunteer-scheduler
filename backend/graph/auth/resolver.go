package auth

import "volunteer-scheduler/services"

type Resolver struct {
	MagicLinkService *services.MagicLinkService
}
