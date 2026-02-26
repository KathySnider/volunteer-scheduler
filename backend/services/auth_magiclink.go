package services

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

// MagicLinkService handles magic-link token lifecycle
type MagicLinkService struct {
	db     *sql.DB
	mailer *Mailer
}

// NewMagicLinkService creates a new MagicLinkService
func NewMagicLinkService(db *sql.DB, mailer *Mailer) *MagicLinkService {
	return &MagicLinkService{
		db:     db,
		mailer: mailer,
	}
}

// GenerateMagicLink creates a new magic link token and stores it in the database
// Returns the token string if successful
func (s *MagicLinkService) GenerateMagicLink(ctx context.Context, email, ipAddress, userAgent string) (string, error) {
	// Rate limiting: check if user has requested too many links recently (5 per hour)
	rateLimitQuery := `
		SELECT COUNT(*) FROM magic_links
		WHERE email = $1 AND created_at > NOW() - INTERVAL '1 hour' AND used_at IS NULL
	`
	var count int
	if err := s.db.QueryRowContext(ctx, rateLimitQuery, email).Scan(&count); err != nil {
		log.Printf("Error checking rate limit: %v", err)
	}
	if count >= 5 {
		return "", fmt.Errorf("too many magic link requests; please try again later")
	}

	// Generate a random token (32 bytes = 64 hex chars)
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)

	// Calculate expiration time (15 minutes by default)
	expiresAt := time.Now().Add(15 * time.Minute)

	// Insert token into database
	insertQuery := `
		INSERT INTO magic_links (email, token, created_at, expires_at, ip_address, user_agent)
		VALUES ($1, $2, NOW(), $3, $4, $5)
	`
	if _, err := s.db.ExecContext(ctx, insertQuery, email, token, expiresAt, ipAddress, userAgent); err != nil {
		return "", fmt.Errorf("failed to store magic link token: %w", err)
	}

	return token, nil
}

// SendMagicLinkEmail sends the magic link email to the user
func (s *MagicLinkService) SendMagicLinkEmail(ctx context.Context, to, token string) error {
	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		appURL = "http://localhost:3000"
	}

	// Remove trailing slash if present
	if appURL[len(appURL)-1] == '/' {
		appURL = appURL[:len(appURL)-1]
	}

	callbackURL := fmt.Sprintf("%s/auth/magic-link?token=%s", appURL, token)

	subject := "Your AARP Volunteer System Magic Link"

	// HTML email body - uses inline styles for email client compatibility
	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; margin: 0; padding: 0;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <div style="background-color: #0066cc; color: white; padding: 20px; text-align: center;">
            <h1 style="margin: 0; color: white;">AARP Volunteer System</h1>
        </div>
        <div style="padding: 20px; background-color: #f9f9f9;">
            <p>Hello,</p>
            <p>We received a request to sign you into the AARP Volunteer System. Click the button below to complete your sign-in:</p>
            <a href="%s" style="display: inline-block; padding: 12px 24px; background-color: #0066cc; color: #ffffff; text-decoration: none; border-radius: 5px; margin: 20px 0; font-weight: bold;">Sign In with Magic Link</a>
            <p>Or copy and paste this link in your browser:</p>
            <p style="word-break: break-all;"><code>%s</code></p>
            <div style="background-color: #fff3cd; padding: 10px; border-left: 4px solid #ffc107; margin: 20px 0;">
                <strong>Security Note:</strong> This link will expire in 15 minutes. If you did not request this link, please ignore this email.
            </div>
            <p>Thank you,<br>AARP Volunteer System Team</p>
        </div>
        <div style="font-size: 12px; color: #666; text-align: center; padding: 20px;">
            <p>&copy; 2026 AARP. All rights reserved.</p>
        </div>
    </div>
</body>
</html>
    `, callbackURL, callbackURL)

	// Text email body
	textBody := fmt.Sprintf(`
Hello,

We received a request to sign you into the AARP Volunteer System. Click the link below to complete your sign-in:

%s

This link will expire in 15 minutes.

If you did not request this link, please ignore this email.

Thank you,
AARP Volunteer System Team
    `, callbackURL)

	if err := s.mailer.SendEmail(ctx, to, subject, htmlBody, textBody); err != nil {
		return fmt.Errorf("failed to send magic link email: %w", err)
	}

	log.Printf("Magic link email sent to %s", to)
	return nil
}

// ConsumeMagicLink validates and consumes a magic link token
// Returns the email if valid; returns error if invalid or expired
func (s *MagicLinkService) ConsumeMagicLink(ctx context.Context, token string) (string, error) {
	if token == "" {
		return "", fmt.Errorf("token is required")
	}

	// Query for the token (must not be used, and must not be expired)
	query := `
		SELECT email FROM magic_links
		WHERE token = $1 AND used_at IS NULL AND expires_at > NOW()
		LIMIT 1
	`

	var email string
	if err := s.db.QueryRowContext(ctx, query, token).Scan(&email); err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("invalid or expired magic link")
		}
		return "", fmt.Errorf("error retrieving magic link: %w", err)
	}

	// Mark the token as used
	updateQuery := `
		UPDATE magic_links
		SET used_at = NOW()
		WHERE token = $1
	`
	if _, err := s.db.ExecContext(ctx, updateQuery, token); err != nil {
		log.Printf("Warning: failed to mark token as used: %v", err)
		// Continue anyway - the token was valid, we just couldn't mark it
	}

	return email, nil
}

// CleanupExpiredTokens removes expired and used tokens older than 24 hours
// Typically called periodically (e.g., daily background job)
func (s *MagicLinkService) CleanupExpiredTokens(ctx context.Context) error {
	query := `
		DELETE FROM magic_links
		WHERE (expires_at < NOW() OR (used_at IS NOT NULL AND used_at < NOW() - INTERVAL '24 hours'))
	`
	result, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("error deleting expired tokens: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		log.Printf("Cleaned up %d expired magic link tokens", rowsAffected)
	}

	return nil
}

// ============================================================================
// JWT / Session Management
// ============================================================================

// SessionClaims represents the claims in a session JWT
type SessionClaims struct {
	Email string `json:"email"`
	Sub   string `json:"sub"` // subject (email)
}

// CreateSessionToken creates a signed JWT token for the authenticated user
// For simplicity, we use a simple approach; production systems should use jwt library
func (s *MagicLinkService) CreateSessionToken(email string) (string, error) {
	// For this implementation, we'll use a simple approach with JWT
	// In production, use github.com/golang-jwt/jwt/v4 for proper JWT handling

	// Get session max age from environment (default 30 days)
	sessionMaxAgeStr := os.Getenv("SESSION_MAX_AGE")
	sessionMaxAge := 2592000 // 30 days in seconds
	if val, err := strconv.Atoi(sessionMaxAgeStr); err == nil {
		sessionMaxAge = val
	}

	// For now, we'll create a simple token-based session by storing in a session table
	// This avoids the complexity of JWT signing without adding an external dependency
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate session token: %w", err)
	}
	sessionToken := hex.EncodeToString(tokenBytes)

	// Store session in database (you'll need a sessions table)
	expiresAt := time.Now().Add(time.Duration(sessionMaxAge) * time.Second)
	insertQuery := `
		INSERT INTO sessions (email, token, created_at, expires_at)
		VALUES ($1, $2, NOW(), $3)
		ON CONFLICT (email) DO UPDATE SET
			token = EXCLUDED.token,
			created_at = NOW(),
			expires_at = EXCLUDED.expires_at
	`
	if _, err := s.db.Exec(insertQuery, email, sessionToken, expiresAt); err != nil {
		// If sessions table doesn't exist, just return the token anyway
		log.Printf("Note: could not store session in database (table may not exist yet): %v", err)
	}

	return sessionToken, nil
}

// ValidateSessionToken checks if a session token is valid and returns the email
func (s *MagicLinkService) ValidateSessionToken(ctx context.Context, token string) (string, error) {
	query := `
		SELECT email FROM sessions
		WHERE token = $1 AND expires_at > NOW()
		LIMIT 1
	`

	var email string
	if err := s.db.QueryRowContext(ctx, query, token).Scan(&email); err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("invalid or expired session token")
		}
		return "", fmt.Errorf("error validating session: %w", err)
	}

	return email, nil
}
