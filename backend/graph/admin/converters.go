package admin

import (
	"volunteer-scheduler/graph/admin/generated"
	"volunteer-scheduler/models"
)

// DUPLICATE CODE:
// This stuff is in both admin and volunteers, because it has to used the
// correct generated code. Just try to keep it the same.
// Generic - lookups

func toGenLookupValues(m models.LookupValues) generated.LookupValues {
	return generated.LookupValues{
		ServiceTypes: toGenServiceTypes(m.ServiceTypes),
		JobTypes:     toGenJobTypes(m.JobTypes),
		Cities:       m.Cities,
	}
}

func toGenServiceTypes(ms []*models.ServiceType) []*generated.ServiceType {
	result := make([]*generated.ServiceType, len(ms))
	for i, m := range ms {
		result[i] = toGenServiceType(m)
	}
	return result
}

func toGenJobTypes(ms []*models.JobType) []*generated.JobType {
	result := make([]*generated.JobType, len(ms))
	for i, m := range ms {
		result[i] = toGenJobType(m)
	}
	return result
}

func toGenServiceType(m *models.ServiceType) *generated.ServiceType {
	if m == nil {
		return nil
	}
	return &generated.ServiceType{
		ID:   m.ID,
		Code: m.Code,
		Name: m.Name,
	}
}

func toGenJobType(m *models.JobType) *generated.JobType {
	if m == nil {
		return nil
	}

	return &generated.JobType{
		ID:       m.ID,
		Code:     m.Code,
		Name:     m.Name,
		IsActive: m.IsActive,
	}
}

// Events

func toModelShiftTimeFilter(g generated.ShiftTimeFilter) models.ShiftsTimeFilter {
	return models.ShiftsTimeFilter(g)
}

func toGenEventShiftSummaries(ms []*models.EventShiftSummary) []*generated.EventShiftSummary {
	result := make([]*generated.EventShiftSummary, len(ms))
	for i, m := range ms {
		result[i] = toGenEventShiftSummary(m)
	}
	return result
}

func toGenEventShiftSummary(m *models.EventShiftSummary) *generated.EventShiftSummary {
	if m == nil {
		return nil
	}
	return &generated.EventShiftSummary{
		JobName:            m.JobName,
		AssignedVolunteers: m.AssignedVolunteers,
		MaxVolunteers:      m.MaxVolunteers,
	}
}

// Feedback

func toGenFeedbackMetaAttachments(ms []*models.FeedbackMetaAttachment) []*generated.FeedbackMetaAttachment {
	result := make([]*generated.FeedbackMetaAttachment, len(ms))
	for i, m := range ms {
		result[i] = toGenFeedbackMetaAttachment(m)
	}

	return result
}
func toGenFeedbackMetaAttachment(m *models.FeedbackMetaAttachment) *generated.FeedbackMetaAttachment {
	if m == nil {
		return nil
	}
	return &generated.FeedbackMetaAttachment{
		ID:        m.ID,
		Filename:  m.Filename,
		MimeType:  m.MimeType,
		FileSize:  m.FileSize,
		CreatedAt: m.CreatedAt,
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

// Results

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

// Roles

func toGenRoles(ms []models.Role) []generated.Role {
	result := make([]generated.Role, len(ms))
	for i, r := range ms {
		result[i] = generated.Role(r)
	}
	return result
}

// END OF DUPLICATE CODE.

// Convert models to generated (graphql) types. (Output from services to the API.)

// Generic - lookup types.

func toGenFundingEntities(ms []*models.FundingEntity) []*generated.FundingEntity {
	result := make([]*generated.FundingEntity, len(ms))
	for i, m := range ms {
		result[i] = toGenFundingEntity(m)
	}
	return result
}

func toGenFundingEntity(m *models.FundingEntity) *generated.FundingEntity {
	if m == nil {
		return nil
	}
	return &generated.FundingEntity{
		ID:          m.ID,
		Name:        m.Name,
		Description: m.Description,
	}
}

// Events, Opportunities, Shifts

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

	return &generated.Event{
		ID:              m.ID,
		Name:            m.Name,
		Description:     m.Description,
		EventType:       generated.EventType(m.EventType),
		StaffContactID:  m.StaffContactId,
		Venue:           toGenVenue(m.Venue),
		EventDates:      toGenEventDates(m.EventDates),
		Timezone:        m.Timezone,
		FundingEntity:   toGenFundingEntity(&m.FundingEntity),
		ServiceTypes:    m.ServiceTypes,
		ShiftSummaries:  toGenEventShiftSummaries(m.ShiftSummaries),
		RecurrenceGroup: toGenRecurrenceGroup(m.RecurrenceGroup),
		RecurrenceOrder: m.RecurrenceOrder,
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

func toGenRecurrenceGroup(m *models.RecurrenceGroup) *generated.RecurrenceGroup {
	if m == nil {
		return nil
	}

	return &generated.RecurrenceGroup{
		GroupID:        m.GroupID,
		Pattern:        generated.RecurrencePattern(m.Pattern),
		MaxOccurrences: m.MaxOccurrences,
		WeekdayOrdinal: m.WeekdayOrdinal,
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
		JobID:                m.JobId,
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
		ID:            m.ID,
		StartDateTime: m.StartDateTime,
		EndDateTime:   m.EndDateTime,
		MaxVolunteers: m.MaxVolunteers,
	}
}

// Feedback

func toGenFeedbackAttachment(m *models.FeedbackAttachment) *generated.FeedbackAttachment {
	if m == nil {
		return nil
	}
	return &generated.FeedbackAttachment{
		Filename: m.Filename,
		MimeType: m.MimeType,
		Data:     m.Data,
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
	return &generated.FeedbackNote{
		ID:        m.ID,
		Creator:   m.Creator,
		Note:      m.Note,
		NoteType:  generated.FeedbackNoteType(m.NoteType),
		CreatedAt: m.CreatedAt,
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
		ID:            m.ID,
		VolunteerName: m.VolunteerName,
		Type:          generated.FeedbackType(m.Type),
		Status:        generated.FeedbackStatus(m.Status),
		Subject:       m.Subject,
		AppPageName:   m.AppPageName,
		Text:          m.Text,
		Notes:         toGenFeedbackNotes(m.Notes),
		CreatedAt:     m.CreatedAt,
		LastUpdatedAt: m.LastUpdatedAt,
		ResolvedAt:    m.ResolvedAt,
		Attachments:   toGenFeedbackMetaAttachments(m.Attachments),
	}
}

// Staff

func toGenAllStaff(ms []*models.Staff) []*generated.Staff {
	result := make([]*generated.Staff, len(ms))
	for i, m := range ms {
		result[i] = toGenStaff(m)
	}
	return result
}

func toGenStaff(m *models.Staff) *generated.Staff {
	if m == nil {
		return nil
	}
	return &generated.Staff{
		ID:        m.ID,
		FirstName: m.FirstName,
		LastName:  m.LastName,
		Email:     m.Email,
		Phone:     m.Phone,
		Position:  m.Position,
	}
}

// Venues
func toGenVenues(ms []*models.Venue) []*generated.Venue {
	result := make([]*generated.Venue, len(ms))
	for i, m := range ms {
		result[i] = toGenVenue(m)
	}
	return result
}

func toGenVenue(m *models.Venue) *generated.Venue {
	if m == nil {
		return nil
	}
	return &generated.Venue{
		ID:      m.ID,
		Name:    m.Name,
		Address: m.Address,
		City:    m.City,
		State:   m.State,
		ZipCode: m.ZipCode,
	}
}

// Volunteers

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
		Distance:  m.Distance,
		Roles:     toGenRoles(m.Roles),
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
		JobName:              m.JobName,
		IsVirtual:            m.IsVirtual,
		PreEventInstructions: m.PreEventInstructions,
		EventID:              m.EventId,
		EventName:            m.EventName,
		EventDescription:     m.EventDescription,
		Venue:                toGenVenue(m.Venue),
	}
}

// Convert generated (graphql) types to models. (Input from API to services.)

// Generic

func toModelNewJobTypeInput(g generated.NewJobTypeInput) models.NewJobTypeInput {
	return models.NewJobTypeInput{
		Code: g.Code,
		Name: g.Name,
	}
}

func toModelUpdateJobTypeInput(g generated.UpdateJobTypeInput) models.UpdateJobTypeInput {
	return models.UpdateJobTypeInput{
		ID:        g.ID,
		Code:      g.Code,
		Name:      g.Name,
		SortOrder: g.SortOrder,
	}
}

func toModelNewFundingEntityInput(g generated.NewFundingEntityInput) models.NewFundingEntityInput {
	return models.NewFundingEntityInput{
		Name:        g.Name,
		Description: g.Description,
	}
}

func toModelUpdateFundingEntityInput(g generated.UpdateFundingEntityInput) models.UpdateFundingEntityInput {
	return models.UpdateFundingEntityInput{
		ID:          g.ID,
		Name:        g.Name,
		Description: g.Description,
	}
}

// Events

func toModelEventFilterInput(g *generated.EventFilterInput) *models.EventFilterInput {
	if g == nil {
		return nil
	}

	var eventType *models.EventType
	if g.EventType != nil {
		et := models.EventType(*g.EventType)
		eventType = &et
	}
	var timeframe *models.ShiftsTimeFilter
	if g.TimeFrame != nil {
		tf := models.ShiftsTimeFilter(*g.TimeFrame)
		timeframe = &tf
	}
	return &models.EventFilterInput{
		Cities:    g.Cities,
		EventType: eventType,
		Jobs:      g.Jobs,
		TimeFrame: timeframe,
	}
}

func toModelNewEventInput(g generated.NewEventInput) models.NewEventInput {

	eventType := models.EventType(g.EventType)

	return models.NewEventInput{
		Name:            g.Name,
		Description:     g.Description,
		EventType:       eventType,
		StaffContactId:  g.StaffContactID,
		VenueId:         g.VenueID,
		Timezone:        g.Timezone,
		FundingEntityID: g.FundingEntityID,
		ServiceTypes:    g.ServiceTypes,
		EventDates:      toModelNewEventDates(g.EventDates),
		Recurrence:      toModelRecurrenceInput(g.Recurrence),
	}
}

func toModelRecurrenceInput(g *generated.RecurrenceInput) *models.RecurrenceInput {
	if g == nil {
		return nil
	}
	r := &models.RecurrenceInput{
		Pattern:        models.RecurrencePattern(g.Pattern),
		MaxOccurrences: g.MaxOccurrences,
	}
	if g.WeekdayOrdinal != nil {
		wo := models.WeekdayOrdinal(*g.WeekdayOrdinal)
		r.WeekdayOrdinal = &wo
	}
	return r
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
	}
}

func toModelAddEventDate(g generated.AddEventDateInput) models.AddEventDateInput {

	return models.AddEventDateInput{
		EventID:       g.EventID,
		StartDateTime: g.StartDateTime,
		EndDateTime:   g.EndDateTime,
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
		JobId:                g.JobID,
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
		StartDateTime: g.StartDateTime,
		EndDateTime:   g.EndDateTime,
		MaxVolunteers: g.MaxVolunteers,
	}
}

func toModelAddShiftInput(g generated.AddShiftInput) models.AddShiftInput {

	return models.AddShiftInput{
		OppId:         g.OpportunityID,
		StartDateTime: g.StartDateTime,
		EndDateTime:   g.EndDateTime,
		MaxVolunteers: g.MaxVolunteers,
	}
}

// Scope is for updating/deleting recurring events.
func toModelScope(s *generated.RecurrenceUpdateScope) *models.RecurrenceUpdateScope {
	if s == nil {
		return nil
	}
	ms := models.RecurrenceUpdateScope(*s)
	return &ms
}

func toModelUpdateEventInput(g generated.UpdateEventInput) models.UpdateEventInput {

	return models.UpdateEventInput{
		ID:              g.ID,
		Name:            g.Name,
		Description:     g.Description,
		EventType:       models.EventType(g.EventType),
		StaffContactId:  g.StaffContactID,
		VenueId:         g.VenueID,
		Timezone:        g.Timezone,
		FundingEntityID: g.FundingEntityID,
		ServiceTypes:    g.ServiceTypes,
		RecurrenceScope: toModelScope(g.RecurrenceScope),
	}
}

func toModelUpdateEventDateInput(g generated.UpdateEventDateInput) models.UpdateEventDateInput {
	return models.UpdateEventDateInput{
		ID:            g.ID,
		StartDateTime: g.StartDateTime,
		EndDateTime:   g.EndDateTime,
	}
}

func toModelUpdateOpportunity(g generated.UpdateOpportunityInput) models.UpdateOpportunityInput {
	return models.UpdateOpportunityInput{
		ID:                   g.ID,
		JobId:                g.JobID,
		IsVirtual:            g.IsVirtual,
		PreEventInstructions: g.PreEventInstructions,
	}
}

func toModelUpdateShift(g generated.UpdateShiftInput) models.UpdateShiftInput {
	return models.UpdateShiftInput{
		ID:            g.ID,
		StartDateTime: g.StartDateTime,
		EndDateTime:   g.EndDateTime,
		MaxVolunteers: g.MaxVolunteers,
	}
}

// Feedback

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

func toModelFeedbackStatusUpdateInput(g generated.FeedbackStatusUpdateInput) models.FeedbackStatusUpdateInput {
	return models.FeedbackStatusUpdateInput{
		FeedbackID: g.FeedbackID,
		Status:     models.FeedbackStatus(g.Status),
		Note:       g.Note,
	}
}

func toModelFeedbackNoteInput(g generated.FeedbackNoteInput) models.FeedbackNoteInput {
	return models.FeedbackNoteInput{
		FeedbackID: g.FeedbackID,
		Note:       g.Note,
	}
}

func toModelFeedbackEmailInput(g generated.FeedbackEmailInput) models.FeedbackEmailInput {
	return models.FeedbackEmailInput{
		FeedbackID:   g.FeedbackID,
		EmailText:    g.EmailText,
		RequireReply: g.RequireReply,
	}
}

// Staff

func toModelNewStaffInput(g generated.NewStaffInput) models.NewStaffInput {
	return models.NewStaffInput{
		FirstName: g.FirstName,
		LastName:  g.LastName,
		Email:     g.Email,
		Phone:     g.Phone,
		Position:  g.Position,
	}
}

func toModelUpdateStaffInput(g generated.UpdateStaffInput) models.UpdateStaffInput {
	return models.UpdateStaffInput{
		ID:        g.ID,
		FirstName: g.FirstName,
		LastName:  g.LastName,
		Email:     g.Email,
		Phone:     g.Phone,
		Position:  g.Position,
	}
}

// Venues

func toModelNewVenueInput(g generated.NewVenueInput) models.NewVenueInput {

	return models.NewVenueInput{
		Name:    g.Name,
		Address: g.Address,
		City:    g.City,
		State:   g.State,
		ZipCode: g.ZipCode,
	}
}

func toModelUpdateVenue(g generated.UpdateVenueInput) models.UpdateVenueInput {
	return models.UpdateVenueInput{
		ID:      g.ID,
		Name:    g.Name,
		Address: g.Address,
		City:    g.City,
		State:   g.State,
		ZipCode: g.ZipCode,
	}
}

// Volunteers

func toModelVolunteeFilterInput(g *generated.VolunteerFilterInput) *models.VolunteerFilterInput {
	if g == nil {
		return nil
	}
	return &models.VolunteerFilterInput{
		FirstName: g.FirstName,
		LastName:  g.LastName,
		Email:     g.Email,
	}
}

func toModelNewVolunteerInput(g generated.NewVolunteerInput) models.NewVolunteerInput {

	return models.NewVolunteerInput{
		FirstName: g.FirstName,
		LastName:  g.LastName,
		Email:     g.Email,
		Phone:     g.Phone,
		ZipCode:   g.ZipCode,
		Distance:  g.Distance,
		Role:      models.Role(g.Role),
	}
}

func toModelUpdateVolunteerInput(g generated.UpdateVolunteerInput) models.UpdateVolunteerInput {
	return models.UpdateVolunteerInput{
		ID:        g.ID,
		FirstName: g.FirstName,
		LastName:  g.LastName,
		Email:     g.Email,
		Phone:     g.Phone,
		ZipCode:   g.ZipCode,
		Distance:  g.Distance,
		Role:      models.Role(g.Role),
	}
}
