package volunteer

import (
	"volunteer-scheduler/graph/volunteer/generated"
	"volunteer-scheduler/models"
)

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

func toGenVenueView(m *models.VenueView) *generated.VenueView {
	if m == nil {
		return nil
	}
	return &generated.VenueView{
		Name:    m.Name,
		Address: m.Address,
		City:    m.City,
		State:   m.State,
		ZipCode: m.ZipCode,
	}
}

func toGenVolunteerView(m *models.VolunteerView) *generated.VolunteerView {
	if m == nil {
		return nil
	}

	return &generated.VolunteerView{
		FirstName: m.FirstName,
		LastName:  m.LastName,
		Email:     m.Email,
		Phone:     m.Phone,
		ZipCode:   m.ZipCode,
		Distance:  m.Distance,
		Roles:     toGenRoles(m.Roles),
	}
}

func toGenRoles(ms []models.Role) []generated.Role {
	result := make([]generated.Role, len(ms))
	for i, r := range ms {
		result[i] = generated.Role(r)
	}
	return result
}

func toGenEventViews(ms []*models.EventView) []*generated.EventView {
	result := make([]*generated.EventView, len(ms))
	for i, m := range ms {
		result[i] = toGenEventView(m)
	}
	return result
}

func toGenEventView(m *models.EventView) *generated.EventView {
	if m == nil {
		return nil
	}

	return &generated.EventView{
		ID:             m.ID,
		Name:           m.Name,
		Description:    m.Description,
		EventType:      generated.EventType(m.EventType),
		Venue:          toGenVenueView(m.Venue),
		EventDates:     toGenEventDateViews(m.EventDates),
		ServiceTypes:   m.ServiceTypes,
		ShiftSummaries: toGenEventShiftSummaries(m.ShiftSummaries),
	}
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

func toGenEventDateViews(ms []*models.EventDateView) []*generated.EventDateView {
	result := make([]*generated.EventDateView, len(ms))
	for i, m := range ms {
		result[i] = toGenEventDateView(m)
	}
	return result
}

func toGenEventDateView(m *models.EventDateView) *generated.EventDateView {

	return &generated.EventDateView{
		StartDateTime: m.StartDateTime,
		EndDateTime:   m.EndDateTime,
	}
}

// Convert generated types to models

func toModelShiftTimeFilter(g generated.ShiftTimeFilter) models.ShiftsTimeFilter {
	return models.ShiftsTimeFilter(g)
}

func toModelVolunteerEventFilterInput(g *generated.VolunteerEventFilterInput) *models.VolunteerEventFilterInput {
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
	return &models.VolunteerEventFilterInput{
		Cities:    g.Cities,
		Distance:  g.Distance,
		EventType: eventType,
		Jobs:      g.Jobs,
		TimeFrame: timeframe,
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

func toGenFeedbackAttachmentViews(ms []*models.FeedbackAttachmentView) []*generated.FeedbackAttachmentView {
	attachments := make([]*generated.FeedbackAttachmentView, len(ms))
	for i, m := range ms {
		attachments[i] = toGenFeedbackAttachmentView(m)
	}
	return attachments
}

func toGenFeedbackAttachmentView(m *models.FeedbackAttachmentView) *generated.FeedbackAttachmentView {
	if m == nil {
		return nil
	}
	return &generated.FeedbackAttachmentView{
		Filename: m.Filename,
		MimeType: m.MimeType,
		Data:     m.Data,
	}
}

// END OF DUPLICATE CODE

func toGenEventShiftViews(ms []*models.EventShiftView) []*generated.EventShiftView {
	result := make([]*generated.EventShiftView, len(ms))
	for i, m := range ms {
		result[i] = toGenEventShiftView(m)
	}
	return result
}

func toGenEventShiftView(m *models.EventShiftView) *generated.EventShiftView {
	if m == nil {
		return nil
	}
	return &generated.EventShiftView{
		ID:                 m.ID,
		JobName:            m.JobName,
		StartDateTime:      m.StartDateTime,
		EndDateTime:        m.EndDateTime,
		IsVirtual:          m.IsVirtual,
		MaxVolunteers:      m.MaxVolunteers,
		AssignedVolunteers: m.AssignedVolunteers,
	}
}

func toGenFeedbackNoteViews(ms []*models.FeedbackNoteView) []*generated.FeedbackNoteView {
	notes := make([]*generated.FeedbackNoteView, len(ms))
	for i, m := range ms {
		notes[i] = toGenFeedbackNoteView(m)
	}
	return notes
}

func toGenFeedbackNoteView(m *models.FeedbackNoteView) *generated.FeedbackNoteView {
	if m == nil {
		return nil
	}
	return &generated.FeedbackNoteView{
		ID:        m.ID,
		Note:      m.Note,
		NoteType:  generated.FeedbackNoteType(m.NoteType),
		CreatedAt: m.CreatedAt,
	}
}

func toGenFeedbackViews(ms []*models.FeedbackView) []*generated.FeedbackView {
	result := make([]*generated.FeedbackView, len(ms))
	for i, m := range ms {
		result[i] = toGenFeedbackView(m)
	}
	return result
}

func toGenFeedbackView(m *models.FeedbackView) *generated.FeedbackView {
	if m == nil {
		return nil
	}
	return &generated.FeedbackView{
		ID:             m.ID,
		Type:           generated.FeedbackType(m.Type),
		Status:         generated.FeedbackStatus(m.Status),
		Subject:        m.Subject,
		AppPageName:    m.AppPageName,
		Text:           m.Text,
		Notes:          toGenFeedbackNoteViews(m.Notes),
		GithubIssueURL: m.GithubIssueURL,
		Attachments:    toGenFeedbackMetaAttachments(m.Attachments),
	}
}
func toGenVolunteerShiftViews(ms []*models.VolunteerShiftView) []*generated.VolunteerShiftView {
	result := make([]*generated.VolunteerShiftView, len(ms))
	for i, m := range ms {
		result[i] = toGenVolunteerShiftView(m)
	}
	return result
}

func toGenVolunteerShiftView(m *models.VolunteerShiftView) *generated.VolunteerShiftView {
	if m == nil {
		return nil
	}
	return &generated.VolunteerShiftView{
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
		Venue:                toGenVenueView(m.Venue),
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
		Distance:  g.Distance,
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
