package volunteer

import (
	"volunteer-scheduler/graph/volunteer/generated"
	"volunteer-scheduler/models"
)

// START OF DUPLICATE CODE
// These functions are duplicated in graph/admin/converters.go
// Keep both files in sync when making changes.

// Convert models to generated types, e.g., results
// coming back from services to the API.

func toGenMutationResult(m *models.MutationResult) *generated.MutationResult {
	if m == nil {
		return nil
	}

	return &generated.MutationResult{
		Success: m.Success,
		Message: m.Message,
		ID:      m.ID,
	}
}

func toGenVenue(m *models.Venue) *generated.Venue {
	if m == nil {
		return nil
	}
	return &generated.Venue{
		ID:       m.ID,
		Name:     m.Name,
		Address:  m.Address,
		City:     m.City,
		State:    m.State,
		ZipCode:  m.ZipCode,
		Timezone: m.Timezone,
	}
}

func toGenVolunteerProfile(m *models.VolunteerProfile) *generated.VolunteerProfile {
	if m == nil {
		return nil
	}

	return &generated.VolunteerProfile{
		FirstName: m.FirstName,
		LastName:  m.LastName,
		Email:     m.Email,
		Phone:     m.Phone,
		ZipCode:   m.ZipCode,
		Role:      generated.Role(m.Role),
	}
}

func toGenEvents(ms []*models.Event) []*generated.Event {
	result := make([]*generated.Event, len(ms))
	for i, m := range ms {
		result[i] = toGenEvent(m)
	}
	return result
}

func toGenEvent(m *models.Event) *generated.Event {
	if m == nil {
		return nil
	}

	serviceTypes := make([]generated.ServiceType, len(m.ServiceTypes))
	for i, st := range m.ServiceTypes {
		serviceTypes[i] = generated.ServiceType(st)
	}

	return &generated.Event{
		ID:           m.ID,
		Name:         m.Name,
		Description:  m.Description,
		EventType:    generated.EventType(m.EventType),
		Venue:        toGenVenue(m.Venue),
		EventDates:   toGenEventDates(m.EventDates),
		ServiceTypes: serviceTypes,
	}
}

func toGenEventDates(ms []*models.EventDate) []*generated.EventDate {
	result := make([]*generated.EventDate, len(ms))
	for i, m := range ms {
		result[i] = toGenEventDate(m)
	}
	return result
}

func toGenEventDate(m *models.EventDate) *generated.EventDate {

	return &generated.EventDate{
		ID:            m.ID,
		StartDateTime: m.StartDateTime,
		EndDateTime:   m.EndDateTime,
	}
}

// Convert generated types to models

func toModelEventFilterInput(g *generated.EventFilterInput) *models.EventFilterInput {
	if g == nil {
		return nil
	}

	var eventType *models.EventType
	if g.EventType != nil {
		et := models.EventType(*g.EventType)
		eventType = &et
	}
	jobs := make([]models.Job, len(g.Jobs))
	for i, j := range g.Jobs {
		jobs[i] = models.Job(j)
	}
	return &models.EventFilterInput{
		Cities:         g.Cities,
		EventType:      eventType,
		Jobs:           jobs,
		ShiftStartDate: g.ShiftStartDateTime,
		ShiftEndDate:   g.ShiftEndDateTime,
		IanaZone:       g.IanaZone,
	}
}

// END OF DUPLICATE CODE

func toGenShiftViews(ms []*models.ShiftView) []*generated.ShiftView {
	result := make([]*generated.ShiftView, len(ms))
	for i, m := range ms {
		result[i] = toGenShiftView(m)
	}
	return result
}

func toGenShiftView(m *models.ShiftView) *generated.ShiftView {
	if m == nil {
		return nil
	}
	return &generated.ShiftView{
		ID:                  m.ID,
		Job:                 generated.Job(m.Job),
		OtherJobDescription: m.OtherJobDescription,
		StartDateTime:       m.StartDateTime,
		EndDateTime:         m.EndDateTime,
		IsVirtual:           m.IsVirtual,
		MaxVolunteers:       m.MaxVolunteers,
		AssignedVolunteers:  m.AssignedVolunteers,
	}
}

// Convert generated types to models
func toModelUpdateOwnProfileInput(g *generated.UpdateOwnProfileInput) *models.UpdateOwnProfileInput {
	if g == nil {
		return nil
	}

	return &models.UpdateOwnProfileInput{
		FirstName: g.FirstName,
		LastName:  g.LastName,
		Email:     g.Email,
		Phone:     g.Phone,
		ZipCode:   g.ZipCode,
	}
}

func toGenVolunteerMutationResult(m *models.VolunteerMutationResult) *generated.VolunteerMutationResult {
	if m == nil {
		return nil
	}

	return &generated.VolunteerMutationResult{
		Success: m.Success,
		Message: m.Message,
	}
}
