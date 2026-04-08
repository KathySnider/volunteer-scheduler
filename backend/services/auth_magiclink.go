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
	DB     *sql.DB
	mailer *Mailer
}

// NewMagicLinkService creates a new MagicLinkService
func NewMagicLinkService(DB *sql.DB, mailer *Mailer) *MagicLinkService {
	return &MagicLinkService{
		DB:     DB,
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
	err := s.DB.QueryRowContext(ctx, rateLimitQuery, email).Scan(&count)
	if err != nil {
		log.Printf("Error checking rate limit: %v", err)
	}
	if count >= 5 {
		return "", fmt.Errorf("too many magic link requests; please try again later")
	}

	// Make sure this email is in the DB and the user is active.
	row := s.DB.QueryRowContext(ctx, "SELECT is_active FROM volunteers WHERE email = $1", email)
	if row == nil {
		return "", fmt.Errorf("no volunteer account found for this email")
	}
	var isActive bool
	err = row.Scan(&isActive) 
	if err != nil {
		return "", fmt.Errorf("no volunteer account found for this email")
	}
	if !isActive {
		return "", fmt.Errorf("volunteer account found for this email is inactive")
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
	if _, err := s.DB.ExecContext(ctx, insertQuery, email, token, expiresAt, ipAddress, userAgent); err != nil {
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

	callbackPath := os.Getenv("MAGIC_LINK_CALLBACK_PATH")
	if callbackPath == "" {
		callbackPath = "/auth/magic-link"
	}

	callbackURL := fmt.Sprintf("%s%s?token=%s", appURL, callbackPath, token)

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
	if err := s.DB.QueryRowContext(ctx, query, token).Scan(&email); err != nil {
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
	if _, err := s.DB.ExecContext(ctx, updateQuery, token); err != nil {
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
	result, err := s.DB.ExecContext(ctx, query)
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

// CreateSessionToken creates a session token for the authenticated user,
// storing both volunteer ID and role for fast context population on each request.
func (s *MagicLinkService) CreateSessionToken(ctx context.Context, email string) (string, error) {

	// Look up volunteer ID and role — never exposed to caller.
	volunteerId, role, err := fetchVolunteerIdAndRoleByEmail(ctx, s.DB, email)
	if err != nil {
		return "", err
	}

	// Generate session token.
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate session token: %w", err)
	}
	sessionToken := hex.EncodeToString(tokenBytes)

	// Get session max age from environment (default 30 days).
	sessionMaxAgeStr := os.Getenv("SESSION_MAX_AGE")
	sessionMaxAge := 2592000
	if val, err := strconv.Atoi(sessionMaxAgeStr); err == nil {
		sessionMaxAge = val
	}
	expiresAt := time.Now().Add(time.Duration(sessionMaxAge) * time.Second)

	insertQuery := `
        INSERT INTO sessions (email, token, created_at, expires_at, volunteer_id, role)
        VALUES ($1, $2, NOW(), $3, $4, $5)
        ON CONFLICT (email) DO UPDATE SET
            token = EXCLUDED.token,
            created_at = NOW(),
            expires_at = EXCLUDED.expires_at,
            volunteer_id = EXCLUDED.volunteer_id,
            role = EXCLUDED.role
    `
	if _, err := s.DB.ExecContext(ctx, insertQuery, email, sessionToken, expiresAt, volunteerId, role); err != nil {
		return "", fmt.Errorf("failed to store session: %w", err)
	}

	return sessionToken, nil
}

// ValidateSessionToken validates the session token and returns the volunteer ID
// and role stored at login time, avoiding a DB lookup on every request.
func (s *MagicLinkService) ValidateSessionToken(ctx context.Context, token string) (int, string, error) {
	query := `
        SELECT volunteer_id, role FROM sessions
        WHERE token = $1 AND expires_at > NOW()
        LIMIT 1
    `
	var volunteerId int
	var role string
	if err := s.DB.QueryRowContext(ctx, query, token).Scan(&volunteerId, &role); err != nil {
		if err == sql.ErrNoRows {
			return 0, "", fmt.Errorf("invalid or expired session token")
		}
		return 0, "", fmt.Errorf("error validating session: %w", err)
	}

	// Update last activity.
	s.DB.ExecContext(ctx, "UPDATE sessions SET last_activity_at = NOW() WHERE token = $1", token)

	return volunteerId, role, nil
}

// fetchVolunteerIdAndRoleByEmail looks up both the volunteer ID and role in one query.
func fetchVolunteerIdAndRoleByEmail(ctx context.Context, DB *sql.DB, email string) (int, string, error) {
	var volunteerId int
	var role string
	err := DB.QueryRowContext(ctx,
		"SELECT volunteer_id, role FROM volunteers WHERE email = $1", email).Scan(&volunteerId, &role)
	if err == sql.ErrNoRows {
		return 0, "", fmt.Errorf("no volunteer account found for this email")
	}
	if err != nil {
		return 0, "", fmt.Errorf("error looking up volunteer: %w", err)
	}
	return volunteerId, role, nil
}

// RequestAccount sends an account request email to all admins.
// No DB record is created — the admin reviews the request and
// creates or reactivates the volunteer manually if approved.
// If the email belongs to an existing inactive volunteer, the admin
// receives a reactivation request with the existing account details instead.
func (s *MagicLinkService) RequestAccount(ctx context.Context, email, firstName, lastName string) error {

	// Fetch all admin email addresses.
	rows, err := s.DB.QueryContext(ctx,
		"SELECT email FROM volunteers WHERE role = 'ADMINISTRATOR'")
	if err != nil {
		return fmt.Errorf("error fetching admin emails: %w", err)
	}
	defer rows.Close()

	var adminEmails []string
	for rows.Next() {
		var adminEmail string
		if err := rows.Scan(&adminEmail); err != nil {
			return fmt.Errorf("error scanning admin email: %w", err)
		}
		adminEmails = append(adminEmails, adminEmail)
	}

	if len(adminEmails) == 0 {
		// No admins found — log it but still return success so we
		// don't reveal anything about the system to the requester.
		log.Printf("Warning: account request from %s %s <%s> but no admins found to notify", firstName, lastName, email)
		return nil
	}

	// Check whether this email belongs to an existing inactive volunteer.
	var existingID int
	var existingFirst, existingLast string
	err = s.DB.QueryRowContext(ctx,
		"SELECT volunteer_id, first_name, last_name FROM volunteers WHERE email = $1 AND is_active = false",
		email).Scan(&existingID, &existingFirst, &existingLast)

	if err == nil {
		// Inactive account found — send a reactivation request.
		existingName := existingFirst + " " + existingLast
		return sendActivateAccountRequest(ctx, s.mailer, adminEmails, firstName, lastName, email, existingName, existingID)
	}

	// No inactive account — send a standard new account request.
	return sendNewAccountRequest(ctx, s.mailer, adminEmails, firstName, lastName, email)
}

func (s *MagicLinkService) Logout(ctx context.Context, token string) error {
	_, err := s.DB.ExecContext(ctx, "DELETE FROM sessions where token = $1", token)
	return err
}
