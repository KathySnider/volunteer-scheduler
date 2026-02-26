package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/mail"
	"net/smtp"
	"os"
	"time"
)

// EmailTransport defines the interface for sending emails
type EmailTransport interface {
	SendEmail(ctx context.Context, to, subject, htmlBody, textBody string) error
}

// Mailer wraps the email transport and configuration
type Mailer struct {
	transport EmailTransport
	fromEmail string
	fromName  string
}

// NewMailer creates a new Mailer instance based on environment configuration
func NewMailer() (*Mailer, error) {
	fromEmail := os.Getenv("EMAIL_FROM")
	if fromEmail == "" {
		return nil, fmt.Errorf("EMAIL_FROM environment variable is required")
	}

	// Parse email to extract name if present
	fromName := "AARP Volunteer System"
	if addr, err := mail.ParseAddress(fromEmail); err == nil && addr.Name != "" {
		fromName = addr.Name
		fromEmail = addr.Address
	}

	var transport EmailTransport
	var err error

	if useResend() {
		transport, err = NewResendTransport()
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
	if os.Getenv("NODE_ENV") == "production" {
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
func NewResendTransport() (*ResendTransport, error) {
	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("RESEND_API_KEY environment variable is required for Resend transport")
	}
	return &ResendTransport{apiKey: apiKey}, nil
}

// ResendRequest represents a request to the Resend API
type ResendRequest struct {
	From    string `json:"from"`
	To      string `json:"to"`
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
	fromEmail := os.Getenv("EMAIL_FROM")
	if fromEmail == "" {
		return fmt.Errorf("EMAIL_FROM not set")
	}

	// Parse email address to extract name if present
	if addr, err := mail.ParseAddress(fromEmail); err == nil && addr.Name != "" {
		fromEmail = addr.Address
	}

	request := ResendRequest{
		From:    fromEmail,
		To:      to,
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
	fromEmail := os.Getenv("EMAIL_FROM")
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

