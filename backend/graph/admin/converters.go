package admin

import (
	"volunteer-scheduler/graph/admin/generated"
	"volunteer-scheduler/models"
)

// START OF DUPLICATE CODE
// These functions are duplicated in graph/volunteers/converters.go
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

func toGenRegions(ms []*models.Region) []*generated.Region {
	result := make([]*generated.Region, len(ms))
	for i, m := range ms {
		result[i] = toGenRegion(m)
	}
	return result

}
func toGenRegion(m *models.Region) *generated.Region {
	if m == nil {
		return nil
	}
	return &generated.Region{
		ID:       m.ID,
		Code:     m.Code,
		Name:     m.Name,
		IsActive: m.IsActive,
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
		Region:   m.Region,
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

func toGenVolunteerShifts(ms []*models.VolunteerShift) []*generated.VolunteerShift {
	result := make([]*generated.VolunteerShift, len(ms))
	for i, m := range ms {
		result[i] = toGenVolunteerShift(m)
	}
	return result
}

func toGenVolunteerShift(m *models.VolunteerShift) *generated.VolunteerShift {
	if m == nil {
		return nil
	}
	return &generated.VolunteerShift{
		ShiftID:              m.ShiftId,
		AssignedAt:           m.AssignedAt,
		CancelledAt:          m.CancelledAt,
		StartDateTime:        m.StartDateTime,
		EndDateTime:          m.EndDateTime,
		MaxVolunteers:        m.MaxVolunteers,
		Job:                  generated.Job(m.Job),
		OtherJobDescription:  m.OtherJobDescription,
		IsVirtual:            m.IsVirtual,
		PreEventInstructions: m.PreEventInstructions,
		EventID:              m.EventId,
		EventName:            m.EventName,
		EventDescription:     m.EventDescription,
		Venue:                toGenVenue(m.Venue),
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

func toModelShiftTimeFilter(g generated.ShiftTimeFilter) models.ShiftsTimeFilter {
	return models.ShiftsTimeFilter(g)
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
		Regions:        g.Regions,
		EventType:      eventType,
		Jobs:           jobs,
		ShiftStartDate: g.ShiftStartDateTime,
		ShiftEndDate:   g.ShiftEndDateTime,
		IanaZone:       g.IanaZone,
	}
}

func toModelNewFeedbackInput(g generated.NewFeedbackInput) models.NewFeedbackInput {
	return models.NewFeedbackInput{
		Type:        models.FeedbackType(g.Type),
		Subject:     g.Subject,
		AppPageName: g.AppPageName,
		Text:        g.Text,
	}
}

// END OF DUPLICATE CODE

// Everything below this is unique to admins only.
// Don't worry, if you accidentally put any of this stuff
// in volunteer files, the gqlgenerate step will fail.

// Convert models to generated types
// Elements coming back from services to the API.

func toGenVenues(ms []*models.Venue) []*generated.Venue {
	result := make([]*generated.Venue, len(ms))
	for i, m := range ms {
		result[i] = toGenVenue(m)
	}
	return result
}

func toGenVolunteers(ms []*models.Volunteer) []*generated.Volunteer {
	result := make([]*generated.Volunteer, len(ms))
	for i, m := range ms {
		result[i] = toGenVolunteer(m)
	}
	return result
}

func toGenVolunteer(m *models.Volunteer) *generated.Volunteer {
	if m == nil {
		return nil
	}
	return &generated.Volunteer{
		ID:        m.ID,
		FirstName: m.FirstName,
		LastName:  m.LastName,
		Email:     m.Email,
		Phone:     m.Phone,
		ZipCode:   m.ZipCode,
		Role:      generated.Role(m.Role),
	}
}

func toGenOpportunities(ms []*models.Opportunity) []*generated.Opportunity {
	result := make([]*generated.Opportunity, len(ms))
	for i, m := range ms {
		result[i] = toGenOpportunity(m)
	}
	return result
}

func toGenOpportunity(m *models.Opportunity) *generated.Opportunity {
	if m == nil {
		return nil
	}

	return &generated.Opportunity{
		ID:                   m.ID,
		Job:                  generated.Job(m.Job),
		OtherJobDescription:  m.OtherJobDescription,
		IsVirtual:            m.IsVirtual,
		PreEventInstructions: m.PreEventInstructions,
		Shifts:               toGenShifts(m.Shifts),
	}
}

func toGenShifts(ms []*models.Shift) []*generated.Shift {
	result := make([]*generated.Shift, len(ms))
	for i, m := range ms {
		result[i] = toGenShift(m)
	}
	return result
}

func toGenShift(m *models.Shift) *generated.Shift {
	if m == nil {
		return nil
	}
	return &generated.Shift{
		ID:             m.ID,
		StartDateTime:  m.StartDateTime,
		EndDateTime:    m.EndDateTime,
		MaxVolunteers:  m.MaxVolunteers,
		StaffContactID: m.StaffContactId,
	}
}

func toGenFeedbacks(ms []*models.Feedback) []*generated.Feedback {
	result := make([]*generated.Feedback, len(ms))
	for i, m := range ms {
		result[i] = toGenFeedback(m)
	}
	return result
}

func toGenFeedback(m *models.Feedback) *generated.Feedback {
	if m == nil {
		return nil
	}

	return &generated.Feedback{
		ID:             m.ID,
		VolunteerName:  m.VolunteerName,
		Type:           generated.FeedbackType(m.Type),
		Status:         generated.FeedbackStatus(m.Status),
		Subject:        m.Subject,
		AppPageName:    m.AppPageName,
		Text:           m.Text,
		Notes:          toGenFeedbackNotes(m.Notes),
		GithubIssueURL: m.GithubIssueURL,
		CreatedAt:      m.CreatedAt,
		LastUpdatedAt:  m.LastUpdatedAt,
		ResolvedAt:     m.ResolvedAt,
	}
}

func toGenFeedbackNotes(ms []*models.FeedbackNote) []*generated.FeedbackNote {
	notes := make([]*generated.FeedbackNote, len(ms))
	for i, m := range ms {
		notes[i] = toGenFeedbackNote(m)
	}
	return notes
}

func toGenFeedbackNote(m *models.FeedbackNote) *generated.FeedbackNote {
	if m == nil {
		return nil
	}
	return &generated.FeedbackNote{}

}

// Convert generated types to models.

// Filters from the API to the services.

func toModelVolunteerFilterInput(g *generated.VolunteerFilterInput) *models.VolunteerFilterInput {
	if g == nil {
		return nil
	}

	return &models.VolunteerFilterInput{
		FirstName: g.FirstName,
		LastName:  g.LastName,
		Email:     g.Email,
	}
}

/*
	if g.EventType != nil {
			et := models.EventType(*g.EventType)
			eventType = &et
		}
*/
func toModelFeedbackFilterInput(g *generated.FeedbackFilterInput) *models.FeedbackFilterInput {
	if g == nil {
		return nil
	}

	var fs models.FeedbackStatus
	if g.Status != nil {
		fs = models.FeedbackStatus(*g.Status)
	}
	var ft models.FeedbackType
	if g.Type != nil {
		ft = models.FeedbackType(*g.Type)
	}

	return &models.FeedbackFilterInput{
		Status: &fs,
		Type:   &ft,
	}
}

// New elements from the API to the services.

func toModelNewVolunteerInput(g generated.NewVolunteerInput) models.NewVolunteerInput {

	return models.NewVolunteerInput{
		FirstName: g.FirstName,
		LastName:  g.LastName,
		Email:     g.Email,
		Phone:     g.Phone,
		ZipCode:   g.ZipCode,
		Role:      models.Role(g.Role),
	}
}

func toModelNewVenueInput(g generated.NewVenueInput) models.NewVenueInput {

	return models.NewVenueInput{
		Name:     g.Name,
		Address:  g.Address,
		City:     g.City,
		State:    g.State,
		ZipCode:  g.ZipCode,
		IanaZone: g.IanaZone,
		Region:   g.Region,
	}
}

func toModelNewRegionInput(g generated.NewRegionInput) models.NewRegionInput {
	return models.NewRegionInput{
		Code: g.Code,
		Name: g.Name,
	}
}

func toModelNewEventInput(g generated.NewEventInput) models.NewEventInput {

	eventType := models.EventType(g.EventType)

	serviceTypes := make([]models.ServiceType, len(g.ServiceTypes))
	for i, st := range g.ServiceTypes {
		serviceTypes[i] = models.ServiceType(st)
	}

	return models.NewEventInput{
		Name:         g.Name,
		Description:  g.Description,
		EventType:    eventType,
		VenueId:      g.VenueID,
		ServiceTypes: serviceTypes,
		EventDates:   toModelNewEventDates(g.EventDates),
	}
}

func toModelNewEventDates(gs []*generated.NewEventDateInput) []*models.NewEventDateInput {
	result := make([]*models.NewEventDateInput, len(gs))
	for i, g := range gs {
		result[i] = toModelNewEventDate(g)
	}

	return result
}

func toModelNewEventDate(g *generated.NewEventDateInput) *models.NewEventDateInput {

	return &models.NewEventDateInput{
		StartDateTime: g.StartDateTime,
		EndDateTime:   g.EndDateTime,
		IanaZone:      g.IanaZone,
	}
}

func toModelAddEventDate(g generated.AddEventDateInput) models.AddEventDateInput {

	return models.AddEventDateInput{
		EventID:       g.EventID,
		StartDateTime: g.StartDateTime,
		EndDateTime:   g.EndDateTime,
		IanaZone:      g.IanaZone,
	}
}

func toModelNewOpportunities(gs []generated.NewOpportunityInput) []models.NewOpportunityInput {
	result := make([]models.NewOpportunityInput, len(gs))

	for i, g := range gs {
		result[i] = toModelNewOpportunity(g)
	}
	return result
}

func toModelNewOpportunity(g generated.NewOpportunityInput) models.NewOpportunityInput {

	return models.NewOpportunityInput{
		EventId:              g.EventID,
		Job:                  models.Job(g.Job),
		OtherJobDescription:  g.OtherJobDescription,
		IsVirtual:            g.IsVirtual,
		PreEventInstructions: g.PreEventInstructions,
		Shifts:               toModelNewShifts(g.Shifts),
	}
}

func toModelNewShifts(gs []*generated.NewShiftInput) []*models.NewShiftInput {
	result := make([]*models.NewShiftInput, len(gs))
	for i, g := range gs {
		result[i] = toModelNewShiftInput(g)
	}
	return result
}

func toModelNewShiftInput(g *generated.NewShiftInput) *models.NewShiftInput {
	if g == nil {
		return nil
	}
	return &models.NewShiftInput{
		StartDateTime:  g.StartDateTime,
		EndDateTime:    g.EndDateTime,
		IanaZone:       g.IanaZone,
		MaxVolunteers:  g.MaxVolunteers,
		StaffContactId: g.StaffContactID,
	}
}

func toModelAddShiftInput(g generated.AddShiftInput) models.AddShiftInput {

	return models.AddShiftInput{
		OppId:          g.OpportunityID,
		StartDateTime:  g.StartDateTime,
		EndDateTime:    g.EndDateTime,
		IanaZone:       g.IanaZone,
		MaxVolunteers:  g.MaxVolunteers,
		StaffContactId: g.StaffContactID,
	}
}

// Updates from the API to the services.

func toModelUpdateVolunteerInput(g generated.UpdateVolunteerInput) models.UpdateVolunteerInput {
	return models.UpdateVolunteerInput{
		ID:        g.ID,
		FirstName: g.FirstName,
		LastName:  g.LastName,
		Email:     g.Email,
		Phone:     g.Phone,
		ZipCode:   g.ZipCode,
	}
}

func toModelUpdateVenue(g generated.UpdateVenueInput) models.UpdateVenueInput {
	return models.UpdateVenueInput{
		ID:       g.ID,
		Name:     g.Name,
		Address:  g.Address,
		City:     g.City,
		State:    g.State,
		ZipCode:  g.ZipCode,
		IanaZone: g.IanaZone,
	}
}

func toModelUpdateRegionInput(g generated.UpdateRegionInput) models.UpdateRegionInput {
	return models.UpdateRegionInput{
		ID:   g.ID,
		Code: g.Code,
		Name: g.Name,
	}
}

func toModelUpdateEventInput(g generated.UpdateEventInput) models.UpdateEventInput {

	serviceTypes := make([]models.ServiceType, len(g.ServiceTypes))
	for i, st := range g.ServiceTypes {
		serviceTypes[i] = models.ServiceType(st)
	}

	return models.UpdateEventInput{
		ID:           g.ID,
		Name:         g.Name,
		Description:  g.Description,
		EventType:    models.EventType(g.EventType),
		VenueId:      g.VenueID,
		ServiceTypes: serviceTypes,
	}
}

func toModelUpdateEventDateInput(g generated.UpdateEventDateInput) models.UpdateEventDateInput {
	return models.UpdateEventDateInput{
		ID:            g.ID,
		StartDateTime: g.StartDateTime,
		EndDateTime:   g.EndDateTime,
		IanaZone:      g.IanaZone,
	}
}

func toModelUpdateOpportunity(g generated.UpdateOpportunityInput) models.UpdateOpportunityInput {
	return models.UpdateOpportunityInput{
		ID:                   g.ID,
		Job:                  (models.Job)(g.Job),
		OtherJobDescription:  g.OtherJobDescription,
		IsVirtual:            g.IsVirtual,
		PreEventInstructions: g.PreEventInstructions,
	}
}

func toModelUpdateShift(g generated.UpdateShiftInput) models.UpdateShiftInput {
	return models.UpdateShiftInput{
		ID:             g.ID,
		StartDateTime:  g.StartDateTime,
		EndDateTime:    g.EndDateTime,
		IanaZone:       g.IanaZone,
		MaxVolunteers:  g.MaxVolunteers,
		StaffContactId: g.StaffContactID,
	}
}

func toModelQuestionFreedbackInput(g generated.QuestionFeedbackInput) models.QuestionFeedbackInput {
	return models.QuestionFeedbackInput{
		ID:        g.ID,
		EmailText: g.EmailText,
		Note:      g.Note,
	}
}

func toModelUpdateFeedbackInput(g generated.UpdateFeedbackInput) models.UpdateFeedbackInput {
	return models.UpdateFeedbackInput{
		ID:             g.ID,
		Status:         models.FeedbackStatus(g.Status),
		Note:           g.Note,
		GithubIssueURL: g.GithubIssueURL,
	}
}

func toModelResolveFeedbackInput(g generated.ResolveFeedbackInput) models.ResolveFeedbackInput {
	return models.ResolveFeedbackInput{
		ID:             g.ID,
		Status:         models.FeedbackStatus(g.Status),
		Note:           g.Note,
		GithubIssueURL: g.GithubIssueURL,
	}
}
