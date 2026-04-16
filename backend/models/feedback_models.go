package models

// Output types.

// Show admins all of the notes complete with who
// wrote each and when.

type FeedbackNote struct {
	ID        string
	Creator   string
	CreatedAt string
	NoteType  FeedbackNoteType
	Note      string
}

type FeedbackAttachment struct {
	ID        string
	Filename  string
	MimeType  string
	FileSize  int
	CreatedAt string
}

type Feedback struct {
	ID             string
	VolunteerName  string
	Type           FeedbackType
	Status         FeedbackStatus
	Subject        string
	AppPageName    string
	Text           string
	Notes          []*FeedbackNote
	GithubIssueURL *string
	CreatedAt      string
	LastUpdatedAt  *string
	ResolvedAt     *string
	Attachments    []*FeedbackAttachment
}

type VolunteerFeedbackNote struct {
	ID        string
	CreatedAt string
	NoteType  FeedbackNoteType
	Note      string
}

type VolunteerFeedback struct {
	ID             string
	Type           FeedbackType
	Status         FeedbackStatus
	Subject        string
	AppPageName    string
	Text           string
	GithubIssueURL *string
	CreatedAt      string
	Notes          []*VolunteerFeedbackNote
	Attachments    []*FeedbackAttachment
}

// Input types for queries (e.g., filters)

type FeedbackFilterInput struct {
	Status *FeedbackStatus
	Type   *FeedbackType
}

// Input types for new elements.

type NewFeedbackInput struct {
	Type        FeedbackType
	Subject     string
	AppPageName string
	Text        string
}

// Input types for updates.
type QuestionFeedbackInput struct {
	ID        string
	EmailText string
	Note      string
}

type UpdateFeedbackInput struct {
	ID             string
	Status         FeedbackStatus
	Note           string
	GithubIssueURL *string
}

type ResolveFeedbackInput struct {
	ID             string
	Status         FeedbackStatus
	Note           string
	GithubIssueURL *string
}

// Result types.

type AttachmentDownload struct {
	Filename string
	MimeType string
	Data     string // Base64-encoded file content
}

// Enums

type FeedbackType string

const (
	FeedbackTypeBug         FeedbackType = "BUG"
	FeedbackTypeEnhancement FeedbackType = "ENHANCEMENT"
	FeedbackTypeGeneral     FeedbackType = "GENERAL"
)

type FeedbackStatus string

const (
	FeedbackStatusOpen     FeedbackStatus = "OPEN"
	FeedbackStatusQuestion FeedbackStatus = "QUESTION_SENT"
	FeedbackStatusGithub   FeedbackStatus = "RESOLVED_GITHUB"
	FeedbackStatusRejected FeedbackStatus = "RESOLVED_REJECTED"
)

type FeedbackNoteType string

const (
	FeedbackNoteTypeAdminNote       FeedbackNoteType = "ADMIN_NOTE"
	FeedbackNoteTypeQuestion        FeedbackNoteType = "QUESTION"
	FeedbackNoteTypeVolunteerReply  FeedbackNoteType = "VOLUNTEER_REPLY"
	FeedbackNoteTypeEmailToVoluneer FeedbackNoteType = "EMAIL_TO_VOLUNTEER"
)
