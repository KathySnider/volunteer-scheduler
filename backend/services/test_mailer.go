package services

import "context"

// noOpTransport silently discards all emails.
type noOpTransport struct{}

func (n *noOpTransport) SendEmail(_ context.Context, _, _, _, _ string) error {
	return nil
}

// NewTestMailer returns a Mailer that discards all email sends.
// For use in tests only — do not call in production code.
func NewTestMailer() *Mailer {
	return &Mailer{
		transport: &noOpTransport{},
		fromEmail: "test@example.com",
		fromName:  "Test",
	}
}
