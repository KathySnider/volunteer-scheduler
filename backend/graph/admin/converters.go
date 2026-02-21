package admin

import (
	"volunteer-scheduler/graph/admin/generated"
	"volunteer-scheduler/models"
)

// Convert models to generated types

func toGenUpdateResult(m *models.UpdateResult) *generated.UpdateResult {
	if m == nil {
		return nil
	}
	return &generated.UpdateResult{
		Success: m.Success,
		Message: m.Message,
	}
}

func toGenInsertResult(m *models.InsertResult) *generated.InsertResult {
	if m == nil {
		return nil
	}
	return &generated.InsertResult{
		ID:      m.ID,
		Success: m.Success,
		Message: m.Message,
	}
}

func toGenEvent(m *models.Event) *generated.Event {
	if m == nil {
		return nil
	}
	return &generated.Event{
		ID:            m.ID,
		Name:          m.Name,
		Description:   m.Description,
		EventType:     generated.EventType(m.EventType),
		Venue:         toGenVenue(m.Venue),
		Shifts:        toGenShifts(m.Shifts),
		Opportunities: toGenOpportunities(m.Opportunities),
	}
}

func toGenEvents(ms []*models.Event) []*generated.Event {
	result := make([]*generated.Event, len(ms))
	for i, m := range ms {
		result[i] = toGenEvent(m)
	}
	return result
}

func toGenVenue(m *models.Venue) *generated.Venue {
	if m == nil {
		return nil
	}
	return &generated.Venue{
		Name:    m.Name,
		Address: m.Address,
		City:    m.City,
		State:   m.State,
		ZipCode: m.ZipCode,
	}
}

func toGenShift(m *models.Shift) *generated.Shift {
	if m == nil {
		return nil
	}
	return &generated.Shift{
		ID:                 m.ID,
		Date:               m.Date,
		StartTime:          m.StartTime,
		EndTime:            m.EndTime,
		MaxVolunteers:      m.MaxVolunteers,
		AssignedVolunteers: toGenVolunteerProfiles(m.AssignedVolunteers),
	}
}

func toGenShifts(ms []*models.Shift) []*generated.Shift {
	result := make([]*generated.Shift, len(ms))
	for i, m := range ms {
		result[i] = toGenShift(m)
	}
	return result
}

func toGenOpportunity(m *models.Opportunity) *generated.Opportunity {
	if m == nil {
		return nil
	}
	return &generated.Opportunity{
		ID:     m.ID,
		Job:    generated.Job(m.Job),
		Shifts: toGenShifts(m.Shifts),
	}
}

func toGenOpportunities(ms []*models.Opportunity) []*generated.Opportunity {
	result := make([]*generated.Opportunity, len(ms))
	for i, m := range ms {
		result[i] = toGenOpportunity(m)
	}
	return result
}

func toGenVolunteerProfile(m *models.VolunteerProfile) *generated.VolunteerProfile {
	if m == nil {
		return nil
	}
	serviceTypes := make([]generated.ServiceType, len(m.ServiceTypes))
	for i, st := range m.ServiceTypes {
		serviceTypes[i] = generated.ServiceType(st)
	}
	return &generated.VolunteerProfile{
		ID:           m.ID,
		FirstName:    m.FirstName,
		LastName:     m.LastName,
		Email:        m.Email,
		ServiceTypes: serviceTypes,
	}
}

func toGenVolunteerProfiles(ms []*models.VolunteerProfile) []*generated.VolunteerProfile {
	result := make([]*generated.VolunteerProfile, len(ms))
	for i, m := range ms {
		result[i] = toGenVolunteerProfile(m)
	}
	return result
}

// Convert generated types to models

func toModelNewEventInput(g *generated.NewEventInput) *models.NewEventInput {
	if g == nil {
		return nil
	}
	// TODO
	return nil
}

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
		Cities:    g.Cities,
		EventType: eventType,
		Jobs:      jobs,
		StartDate: g.StartDate,
		EndDate:   g.EndDate,
	}
}

func toModelVolunteerFilterInput(g *generated.VolunteerFilterInput) *models.VolunteerFilterInput {
	if g == nil {
		return nil
	}
	serviceTypes := make([]models.ServiceType, len(g.ServiceTypes))
	for i, st := range g.ServiceTypes {
		serviceTypes[i] = models.ServiceType(st)
	}
	return &models.VolunteerFilterInput{
		FirstName:    g.FirstName,
		LastName:     g.LastName,
		ServiceTypes: serviceTypes,
	}
}

func toModelNewVolunteerInput(g *generated.NewVolunteerInput) *models.NewVolunteerInput {
	serviceTypes := make([]models.ServiceType, len(g.ServiceTypes))
	for i, st := range g.ServiceTypes {
		serviceTypes[i] = models.ServiceType(st)
	}
	return &models.NewVolunteerInput{
		FirstName:    g.FirstName,
		LastName:     g.LastName,
		Email:        g.Email,
		Phone:        g.Phone,
		ZipCode:      g.ZipCode,
		ServiceTypes: serviceTypes,
	}
}

func toModelUpdateVolunteerInput(g *generated.UpdateVolunteerInput) *models.UpdateVolunteerInput {
	serviceTypes := make([]models.ServiceType, len(g.ServiceTypes))
	for i, st := range g.ServiceTypes {
		serviceTypes[i] = models.ServiceType(st)
	}
	return &models.UpdateVolunteerInput{
		ID:           g.ID,
		FirstName:    g.FirstName,
		LastName:     g.LastName,
		Email:        g.Email,
		Phone:        g.Phone,
		ZipCode:      g.ZipCode,
		ServiceTypes: serviceTypes,
	}
}
