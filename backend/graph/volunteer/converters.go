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

func toGenAttachmentDownload(m *models.AttachmentDownload) *generated.AttachmentDownload {
	if m == nil {
		return nil
	}
	return &generated.AttachmentDownload{
		Filename: m.Filename,
		MimeType: m.MimeType,
		Data:     m.Data,
	}
}

func toGenLookupValues(m models.LookupValues) generated.LookupValues {
	return generated.LookupValues{
		Regions:      toGenRegions(m.Regions),
		ServiceTypes: toGenServiceTypes(m.ServiceTypes),
		JobTypes:     toGenJobTypes(m.JobTypes),
	}
}

func toGenRegions(ms []*models.Region) []*generated.Region {
	result := make([]*generated.Region, len(ms))
	for i, m := range ms {
		result[i] = toGenRegion(m)
	}
	return result
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

func toGenServiceType(m *models.ServiceType) *generated.ServiceType {
	if m == nil {
		return nil
	}
	return &generated.ServiceType{
		ID:       m.ID,
		Code:     m.Code,
		Name:     m.Name,
		IsActive: m.IsActive,
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
		JobName:              m.JobName,
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

	return &generated.Event{
		ID:           m.ID,
		Name:         m.Name,
		Description:  m.Description,
		EventType:    generated.EventType(m.EventType),
		Venue:        toGenVenue(m.Venue),
		EventDates:   toGenEventDates(m.EventDates),
		ServiceTypes: m.ServiceTypes,
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
	return &models.EventFilterInput{
		Regions:        g.Regions,
		EventType:      eventType,
		Jobs:           g.Jobs,
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

func toGenFeedbackAttachments(ms []*models.FeedbackAttachment) []*generated.FeedbackAttachment {
	attachments := make([]*generated.FeedbackAttachment, len(ms))
	for i, m := range ms {
		attachments[i] = toGenFeedbackAttachment(m)
	}
	return attachments
}

func toGenFeedbackAttachment(m *models.FeedbackAttachment) *generated.FeedbackAttachment {
	if m == nil {
		return nil
	}
	return &generated.FeedbackAttachment{
		ID:        m.ID,
		Filename:  m.Filename,
		MimeType:  m.MimeType,
		FileSize:  m.FileSize,
		CreatedAt: m.CreatedAt,
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
		ID:                 m.ID,
		JobName:            m.JobName,
		StartDateTime:      m.StartDateTime,
		EndDateTime:        m.EndDateTime,
		IsVirtual:          m.IsVirtual,
		MaxVolunteers:      m.MaxVolunteers,
		AssignedVolunteers: m.AssignedVolunteers,
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
