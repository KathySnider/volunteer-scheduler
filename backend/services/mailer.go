package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/mail"
	"net/smtp"
	"os"
	"time"
)

// ============================================================================
// Email Templates
// All templates share a common header and footer defined below.
// ============================================================================

const emailHeader = `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; margin: 0; padding: 0;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <div style="background-color: #0066cc; color: white; padding: 20px; text-align: center;">
            <h1 style="margin: 0; color: white;">Volunteer Scheduler</h1>
        </div>
        <div style="padding: 20px; background-color: #f9f9f9;">`

const emailFooter = `
            <p>Thank you,<br>Volunteer Scheduler</p>
        </div>
        <div style="font-size: 12px; color: #666; text-align: center; padding: 20px;">
            <p>&copy; 2026 Volunteer Scheduler</p>
        </div>
    </div>
</body>
</html>`

const tableOpen = `<table style="width: 100%; border-collapse: collapse; margin: 16px 0;">`
const tableClose = `</table>`
const tdLabel = `style="padding: 6px 12px 6px 0; font-weight: bold; width: 140px; vertical-align: top;"`
const tdValue = `style="padding: 6px 0;"`
const tdValueAlt = `style="padding: 6px 0; background-color: #f0f0f0;"`

// ============================================================================
// New Account Request
// ============================================================================

const newAccountRequestHTMLTmpl = emailHeader + `
            <p>A new volunteer account request has been submitted:</p>
            ` + tableOpen + `
                <tr>
                    <td ` + tdLabel + `>Name</td>
                    <td ` + tdValue + `>{{.FirstName}} {{.LastName}}</td>
                </tr>
                <tr>
                    <td ` + tdLabel + `>Email</td>
                    <td ` + tdValueAlt + `>{{.Email}}</td>
                </tr>
            ` + tableClose + `
            <p>To approve this request, log in to the Volunteer Scheduler and create an account for this volunteer.</p>
            <p>If you do not recognize this person or do not wish to approve their request, no action is needed.</p>
` + emailFooter

const newAccountRequestTextTmpl = `A new volunteer account request has been submitted:

Name:  {{.FirstName}} {{.LastName}}
Email: {{.Email}}

To approve this request, log in to the Volunteer Scheduler and create an account for this volunteer.

If you do not recognize this person or do not wish to approve their request, no action is needed.

Thank you,
Volunteer Scheduler`

type newAccountRequestData struct {
	FirstName string
	LastName  string
	Email     string
}

// ============================================================================
// Request to reactivate an existing inactive account.
// ============================================================================

const activateAccountRequestHTMLTmpl = emailHeader + `
            <p>An account access request has been submitted for an email address that belongs to an <strong>existing inactive account</strong>.</p>
            <p><strong>Requester (as entered in the form):</strong></p>
            ` + tableOpen + `
                <tr>
                    <td ` + tdLabel + `>Name</td>
                    <td ` + tdValue + `>{{.FirstName}} {{.LastName}}</td>
                </tr>
                <tr>
                    <td ` + tdLabel + `>Email</td>
                    <td ` + tdValueAlt + `>{{.Email}}</td>
                </tr>
            ` + tableClose + `
            <p><strong>Existing inactive account:</strong></p>
            ` + tableOpen + `
                <tr>
                    <td ` + tdLabel + `>Name on record</td>
                    <td ` + tdValue + `>{{.ExistingName}}</td>
                </tr>
                <tr>
                    <td ` + tdLabel + `>Volunteer ID</td>
                    <td ` + tdValueAlt + `>{{.ExistingID}}</td>
                </tr>
            ` + tableClose + `
            <p>You have two options:</p>
            <ul>
                <li><strong>Reactivate</strong> the existing account (volunteer ID {{.ExistingID}}) to preserve their shift history.</li>
                <li><strong>Create a new account</strong> if you believe the email address has been reassigned to a different person.</li>
            </ul>
            <p>If you do not recognize this person or do not wish to approve their request, no action is needed.</p>
` + emailFooter

const activateAccountRequestTextTmpl = `An account access request has been submitted for an email address that belongs to an existing inactive account.

Requester (as entered in the form):
  Name:  {{.FirstName}} {{.LastName}}
  Email: {{.Email}}

Existing inactive account:
  Name on record: {{.ExistingName}}
  Volunteer ID:   {{.ExistingID}}

You have two options:
  1. Reactivate the existing account (volunteer ID {{.ExistingID}}) to preserve their shift history.
  2. Create a new account if you believe the email address has been reassigned to a different person.

If you do not recognize this person or do not wish to approve their request, no action is needed.

Thank you,
Volunteer Scheduler`

type activateAccountRequestData struct {
	FirstName    string
	LastName     string
	Email        string
	ExistingName string
	ExistingID   int
}


// ============================================================================
// Signup Confirmed
// ============================================================================

const signupConfirmedHTMLTmpl = emailHeader + `
            <p>Hello {{.FirstName}},</p>
            <p>You are signed up for the following shift:</p>
            ` + tableOpen + `
                <tr>
                    <td ` + tdLabel + `>Event</td>
                    <td ` + tdValue + `>{{.EventName}}</td>
                </tr>
                <tr>
                    <td ` + tdLabel + `>Start</td>
                    <td ` + tdValueAlt + `>{{.Start}}</td>
                </tr>
                <tr>
                    <td ` + tdLabel + `>End</td>
                    <td ` + tdValue + `>{{.End}}</td>
                </tr>
                {{if .IsVirtual}}
                <tr>
                    <td ` + tdLabel + `>Location</td>
                    <td ` + tdValueAlt + `>This shift is virtual.</td>
                </tr>
                {{else}}
                <tr>
                    <td ` + tdLabel + `>Location</td>
                    <td ` + tdValueAlt + `>{{if .VenueName}}{{.VenueName}}<br>{{end}}{{.Address}}<br>{{.City}}, {{.State}}{{if .Zip}} {{.Zip}}{{end}}</td>
                </tr>
                {{end}}
            ` + tableClose + `
            {{if .Instructions}}
            <div style="background-color: #fff3cd; padding: 10px; border-left: 4px solid #ffc107; margin: 16px 0;">
                <strong>Pre-Event Instructions:</strong> {{.Instructions}}
            </div>
            {{end}}
            <p>To view or manage your shifts, log in to the Volunteer Scheduler.</p>
` + emailFooter

const signupConfirmedTextTmpl = `You are signed up for {{.EventName}}.

Start: {{.Start}}
End:   {{.End}}
{{if .IsVirtual}}
Location: This shift is virtual.
{{else}}
Location:{{if .VenueName}}
  {{.VenueName}}{{end}}
  {{.Address}}
  {{.City}}, {{.State}}{{if .Zip}} {{.Zip}}{{end}}
{{end}}{{if .Instructions}}
Pre-Event Instructions:
{{.Instructions}}
{{end}}
To view or manage your shifts, log in to the Volunteer Scheduler.

Thank you,
Volunteer Scheduler`

type signupConfirmedData struct {
	FirstName    string
	EventName    string
	Start        string
	End          string
	IsVirtual    bool
	VenueName    string
	Address      string
	City         string
	State        string
	Zip          string
	Instructions string
}

// ============================================================================
// Signup Cancelled
// ============================================================================

const signupCancelledHTMLTmpl = emailHeader + `
            <p>Hello {{.FirstName}},</p>
            <p>Your signup for the following shift has been cancelled:</p>
            ` + tableOpen + `
                <tr>
                    <td ` + tdLabel + `>Event</td>
                    <td ` + tdValue + `>{{.EventName}}</td>
                </tr>
                <tr>
                    <td ` + tdLabel + `>Start</td>
                    <td ` + tdValueAlt + `>{{.Start}}</td>
                </tr>
                <tr>
                    <td ` + tdLabel + `>End</td>
                    <td ` + tdValue + `>{{.End}}</td>
                </tr>
            ` + tableClose + `
            <p>If this was a mistake, log in to the Volunteer Scheduler to sign up again.</p>
` + emailFooter

const signupCancelledTextTmpl = `Your signup for the following shift has been cancelled:

Event: {{.EventName}}
Start: {{.Start}}
End:   {{.End}}

If this was a mistake, log in to the Volunteer Scheduler to sign up again.

Thank you,
Volunteer Scheduler`

type signupCancelledData struct {
	FirstName string
	EventName string
	Start     string
	End       string
}

// ============================================================================
// Account Created (welcome to new volunteer)
// ============================================================================

const accountCreatedHTMLTmpl = emailHeader + `
            <p>Hello {{.FirstName}},</p>
            <p>Welcome to the Volunteer Scheduler! Your account has been
            created with the following details:</p>
            ` + tableOpen + `
                <tr>
                    <td ` + tdLabel + `>Name</td>
                    <td ` + tdValue + `>{{.FirstName}} {{.LastName}}</td>
                </tr>
                <tr>
                    <td ` + tdLabel + `>Email</td>
                    <td ` + tdValueAlt + `>{{.Email}}</td>
                </tr>
                <tr>
                    <td ` + tdLabel + `>Role</td>
                    <td ` + tdValue + `>{{.Role}}</td>
                </tr>
            ` + tableClose + `
            <p>To sign in, go to the Volunteer Scheduler and enter your email address.
            A magic link will be sent to you — no password needed.</p>
` + emailFooter

const accountCreatedTextTmpl = `Hello {{.FirstName}},

Welcome to the Volunteer Scheduler! Your account has been created
with the following details:

Name:  {{.FirstName}} {{.LastName}}
Email: {{.Email}}
Role:  {{.Role}}

To sign in, go to the Volunteer Scheduler and enter your email address.
A magic link will be sent to you — no password needed.

Thank you,
Volunteer Scheduler`

type accountCreatedData struct {
	FirstName string
	LastName  string
	Email     string
	Role      string
}

// ============================================================================
// Account Created — Admin Notification
// ============================================================================

const accountCreatedAdminHTMLTmpl = emailHeader + `
            <p>A new volunteer account has been created:</p>
            ` + tableOpen + `
                <tr>
                    <td ` + tdLabel + `>Name</td>
                    <td ` + tdValue + `>{{.FirstName}} {{.LastName}}</td>
                </tr>
                <tr>
                    <td ` + tdLabel + `>Email</td>
                    <td ` + tdValueAlt + `>{{.Email}}</td>
                </tr>
                <tr>
                    <td ` + tdLabel + `>Role</td>
                    <td ` + tdValue + `>{{.Role}}</td>
                </tr>
                <tr>
                    <td ` + tdLabel + `>Created by</td>
                    <td ` + tdValueAlt + `>{{.CreatedBy}}</td>
                </tr>
            ` + tableClose + `
` + emailFooter

const accountCreatedAdminTextTmpl = `A new volunteer account has been created:

Name:       {{.FirstName}} {{.LastName}}
Email:      {{.Email}}
Role:       {{.Role}}
Created by: {{.CreatedBy}}

Thank you,
Volunteer Scheduler`

type accountCreatedAdminData struct {
	FirstName string
	LastName  string
	Email     string
	Role      string
	CreatedBy string
}

// ============================================================================
// Event Cancelled — Volunteer Notification
// ============================================================================

// ShiftSummary holds the formatted start and end times for a single shift.
// Used in event cancellation emails where a volunteer may have multiple shifts.
type ShiftSummary struct {
	Start string
	End   string
}

const eventCancelledVolunteerHTMLTmpl = emailHeader + `
            <p>Hello {{.FirstName}},</p>
            <p>We regret to inform you that the following event has been cancelled:</p>
            ` + tableOpen + `
                <tr>
                    <td ` + tdLabel + `>Event</td>
                    <td ` + tdValue + `>{{.EventName}}</td>
                </tr>
                {{range $i, $s := .Shifts}}
                <tr>
                    <td ` + tdLabel + `>Shift {{if gt (len $.Shifts) 1}}{{add $i 1}} {{end}}start</td>
                    <td ` + tdValueAlt + `>{{$s.Start}}</td>
                </tr>
                <tr>
                    <td ` + tdLabel + `>Shift {{if gt (len $.Shifts) 1}}{{add $i 1}} {{end}}end</td>
                    <td ` + tdValue + `>{{$s.End}}</td>
                </tr>
                {{end}}
            ` + tableClose + `
            <p>We apologize for any inconvenience. Please log in to Volunteer Scheduler
            to sign up for other upcoming events.</p>
` + emailFooter

const eventCancelledVolunteerTextTmpl = `Hello {{.FirstName}},

We regret to inform you that the following event has been cancelled:

Event: {{.EventName}}
{{range $i, $s := .Shifts}}
Shift {{if gt (len $.Shifts) 1}}{{add $i 1}} {{end}}start: {{$s.Start}}
Shift {{if gt (len $.Shifts) 1}}{{add $i 1}} {{end}}end:   {{$s.End}}
{{end}}
We apologize for any inconvenience. Please log in to the Volunteer Scheduler
to sign up for other upcoming events.

Thank you,
Volunteer Scheduler`

type eventCancelledVolunteerData struct {
	FirstName string
	EventName string
	Shifts    []ShiftSummary
}

// ============================================================================
// Event Cancelled — Staff Lead Notification
// ============================================================================

const eventCancelledStaffHTMLTmpl = emailHeader + `
            <p>Hello {{.FirstName}},</p>
            <p>The following event has been cancelled. You were listed as the staff
            contact for one or more shifts:</p>
            ` + tableOpen + `
                <tr>
                    <td ` + tdLabel + `>Event</td>
                    <td ` + tdValue + `>{{.EventName}}</td>
                </tr>
                {{range $i, $s := .Shifts}}
                <tr>
                    <td ` + tdLabel + `>Shift {{if gt (len $.Shifts) 1}}{{add $i 1}} {{end}}start</td>
                    <td ` + tdValueAlt + `>{{$s.Start}}</td>
                </tr>
                <tr>
                    <td ` + tdLabel + `>Shift {{if gt (len $.Shifts) 1}}{{add $i 1}} {{end}}end</td>
                    <td ` + tdValue + `>{{$s.End}}</td>
                </tr>
                {{end}}
            ` + tableClose + `
            <p>Volunteers assigned to your shift(s) have been notified separately.</p>
` + emailFooter

const eventCancelledStaffTextTmpl = `Hello {{.FirstName}},

The following event has been cancelled. You were listed as the staff contact
for one or more shifts:

Event: {{.EventName}}
{{range $i, $s := .Shifts}}
Shift {{if gt (len $.Shifts) 1}}{{add $i 1}} {{end}}start: {{$s.Start}}
Shift {{if gt (len $.Shifts) 1}}{{add $i 1}} {{end}}end:   {{$s.End}}
{{end}}
Volunteers assigned to your shift(s) have been notified separately.

Thank you,
Volunteer Scheduler`

type eventCancelledStaffData struct {
	FirstName string
	EventName string
	Shifts    []ShiftSummary
}

// ============================================================================
// Template rendering helper
// ============================================================================

func renderTemplate(tmplStr string, data any) (string, error) {
	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}
	tmpl, err := template.New("email").Funcs(funcMap).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("error parsing email template: %w", err)
	}
	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("error rendering email template: %w", err)
	}
	return buf.String(), nil
}

// EmailTransport defines the interface for sending emails
type EmailTransport interface {
	SendEmail(ctx context.Context, to, subject, htmlBody, textBody string) error
}

// Mailer wraps the email transport and configuration
type Mailer struct {
	transport EmailTransport
	fromEmail string
	fromName  string
	apiKey    string
}

// NewMailer creates a new Mailer instance based on environment configuration
func NewMailer(apiKey string) (*Mailer, error) {

	fromEmail := os.Getenv("MAIL_FROM")
	if fromEmail == "" {
		return nil, fmt.Errorf("MAIL_FROM environment variable is required")
	}

	// Parse email to extract name if present
	fromName := "Volunteer Scheduler"
	if addr, err := mail.ParseAddress(fromEmail); err == nil && addr.Name != "" {
		fromName = addr.Name
		fromEmail = addr.Address
	}

	log.Printf("Email sender address: %s (display name: %s)", fromEmail, fromName)

	var transport EmailTransport
	var err error

	if useResend() {
		transport, err = NewResendTransport(apiKey)
		if err != nil {
			return nil, err
		}
		log.Println("Email transport: Resend API")
	} else {
		transport, err = NewMailhogTransport()
		if err != nil {
			return nil, err
		}
		log.Println("Email transport: Mailhog (SMTP)")
	}

	return &Mailer{
		transport: transport,
		fromEmail: fromEmail,
		fromName:  fromName,
	}, nil
}

// useResend determines which email transport to use
func useResend() bool {
	if os.Getenv("USE_RESEND") == "true" {
		return true
	}
	if os.Getenv("APP_ENV") == "production" {
		return true
	}
	return false
}

// SendEmail sends an email via the configured transport
func (m *Mailer) SendEmail(ctx context.Context, to, subject, htmlBody, textBody string) error {
	return m.transport.SendEmail(ctx, to, subject, htmlBody, textBody)
}

// ============================================================================
// ResendTransport - Uses Resend.com API
// ============================================================================

type ResendTransport struct {
	apiKey string
}

// NewResendTransport creates a new Resend transport
func NewResendTransport(apiKey string) (*ResendTransport, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("RESEND_API_KEY is required for Resend transport")
	}
	return &ResendTransport{apiKey: apiKey}, nil
}

// ResendRequest represents a request to the Resend API
type ResendRequest struct {
	From    string `json:"from"`
	To      string `json:"to"`
	ReplyTo string `json:"reply_to,omitempty"`
	Subject string `json:"subject"`
	HTML    string `json:"html"`
	Text    string `json:"text,omitempty"`
}

// ResendResponse represents a response from the Resend API
type ResendResponse struct {
	ID    string `json:"id"`
	From  string `json:"from"`
	To    string `json:"to"`
	Error string `json:"error,omitempty"`
}

// SendEmail sends an email via the Resend API
func (r *ResendTransport) SendEmail(ctx context.Context, to, subject, htmlBody, textBody string) error {
	fromEmail := os.Getenv("MAIL_FROM")
	if fromEmail == "" {
		return fmt.Errorf("EMAIL_FROM not set")
	}

	// Parse email address to extract name if present
	if addr, err := mail.ParseAddress(fromEmail); err == nil && addr.Name != "" {
		fromEmail = addr.Address
	}

	replyTo := os.Getenv("MAIL_REPLY_TO")

	request := ResendRequest{
		From:    fromEmail,
		To:      to,
		ReplyTo: replyTo,
		Subject: subject,
		HTML:    htmlBody,
		Text:    textBody,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal Resend request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create Resend request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+r.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send email via Resend: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		log.Printf("Resend API error (%d) sending to %s from %s — raw body: %s", resp.StatusCode, to, fromEmail, string(respBody))
		var errResp ResendResponse
		json.Unmarshal(respBody, &errResp)
		return fmt.Errorf("Resend API error (%d): %s", resp.StatusCode, errResp.Error)
	}

	var successResp ResendResponse
	if err := json.Unmarshal(respBody, &successResp); err == nil {
		log.Printf("Email sent via Resend to %s (id: %s)", to, successResp.ID)
	}

	return nil
}

// ============================================================================
// MailhogTransport - Uses local Mailhog SMTP for development
// ============================================================================

type MailhogTransport struct {
	host string
	port string
}

// NewMailhogTransport creates a new Mailhog transport
func NewMailhogTransport() (*MailhogTransport, error) {
	host := os.Getenv("EMAIL_SERVER_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("EMAIL_SERVER_PORT")
	if port == "" {
		port = "1025"
	}
	return &MailhogTransport{host: host, port: port}, nil
}

// SendEmail sends an email via Mailhog SMTP
func (m *MailhogTransport) SendEmail(ctx context.Context, to, subject, htmlBody, textBody string) error {
	fromEmail := os.Getenv("MAIL_FROM")
	if fromEmail == "" {
		return fmt.Errorf("EMAIL_FROM not set")
	}

	// Parse email address to extract name if present
	if addr, err := mail.ParseAddress(fromEmail); err == nil && addr.Name != "" {
		fromEmail = addr.Address
	}

	// Build email message as MIME format
	var buf bytes.Buffer

	// Write headers
	buf.WriteString(fmt.Sprintf("From: %s\r\n", fromEmail))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", to))
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: multipart/alternative; boundary=boundary123\r\n\r\n")

	// Text part
	if textBody != "" {
		buf.WriteString("--boundary123\r\n")
		buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		buf.WriteString(textBody)
		buf.WriteString("\r\n")
	}

	// HTML part
	buf.WriteString("--boundary123\r\n")
	buf.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
	buf.WriteString(htmlBody)
	buf.WriteString("\r\n--boundary123--\r\n")

	// Connect to Mailhog SMTP and send
	addr := fmt.Sprintf("%s:%s", m.host, m.port)
	return smtp.SendMail(addr, nil, fromEmail, []string{to}, buf.Bytes())
}
