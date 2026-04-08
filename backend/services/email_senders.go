package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"
)

// ============================================================================
// Send functions — called from the service layer
// ============================================================================

func sendAssignmentConfirmation(ctx context.Context, DB *sql.DB, mailer *Mailer, shiftId int, volId int) error {
	email, err := fetchEmailByVolId(ctx, DB, volId)
	if err != nil {
		return fmt.Errorf("unable to get email address: %w", err)
	}

	query := `
		SELECT
			vol.first_name,
			e.event_name,
			opp.opportunity_is_virtual,
			opp.pre_event_instructions,
			s.shift_start,
			s.shift_end,
			v.venue_name,
			v.street_address,
			v.city,
			v.state,
			v.zip_code,
			v.timezone
		FROM shifts s
		LEFT JOIN opportunities opp ON opp.opportunity_id = s.opportunity_id
		LEFT JOIN events e ON e.event_id = opp.event_id
		LEFT JOIN venues v ON v.venue_id = e.venue_id
		LEFT JOIN volunteers vol ON vol.volunteer_id = $2
		WHERE s.shift_id = $1
	`

	var firstName, eventName, shiftStart, shiftEnd string
	var venueName, address, city, state, zip, timezone, instruct sql.NullString
	var isVirtual bool

	err = DB.QueryRowContext(ctx, query, shiftId, volId).Scan(
		&firstName,
		&eventName,
		&isVirtual,
		&instruct,
		&shiftStart,
		&shiftEnd,
		&venueName,
		&address,
		&city,
		&state,
		&zip,
		&timezone,
	)
	if err != nil {
		return fmt.Errorf("error scanning shift information: %w", err)
	}

	if !isVirtual && !address.Valid {
		return fmt.Errorf("non-virtual shift has no venue address")
	}

	fmtStart, fmtEnd, err := formatStartEnd(ctx, shiftStart, shiftEnd, timezone)
	if err != nil {
		return fmt.Errorf("error formatting shift times: %w", err)
	}

	data := signupConfirmedData{
		FirstName:    firstName,
		EventName:    eventName,
		Start:        *fmtStart,
		End:          *fmtEnd,
		IsVirtual:    isVirtual,
		VenueName:    venueName.String,
		Address:      address.String,
		City:         city.String,
		State:        state.String,
		Zip:          zip.String,
		Instructions: instruct.String,
	}

	subject := "Signup Confirmed: " + eventName
	htmlBody, err := renderTemplate(signupConfirmedHTMLTmpl, data)
	if err != nil {
		return err
	}
	textBody, err := renderTemplate(signupConfirmedTextTmpl, data)
	if err != nil {
		return err
	}

	return mailer.SendEmail(ctx, email, subject, htmlBody, textBody)
}

func sendCancellationConfirmation(ctx context.Context, DB *sql.DB, mailer *Mailer, shiftId int, volId int) error {
	email, err := fetchEmailByVolId(ctx, DB, volId)
	if err != nil {
		return fmt.Errorf("unable to get email address: %w", err)
	}

	query := `
		SELECT
			vol.first_name,
			e.event_name,
			s.shift_start,
			s.shift_end,
			v.timezone
		FROM shifts s
		LEFT JOIN opportunities opp ON opp.opportunity_id = s.opportunity_id
		LEFT JOIN events e ON e.event_id = opp.event_id
		LEFT JOIN venues v ON v.venue_id = e.venue_id
		LEFT JOIN volunteers vol ON vol.volunteer_id = $2
		WHERE s.shift_id = $1
	`

	var firstName, eventName, shiftStart, shiftEnd string
	var timezone sql.NullString

	err = DB.QueryRowContext(ctx, query, shiftId, volId).Scan(
		&firstName, &eventName, &shiftStart, &shiftEnd, &timezone,
	)
	if err != nil {
		return fmt.Errorf("error scanning shift information: %w", err)
	}

	fmtStart, fmtEnd, err := formatStartEnd(ctx, shiftStart, shiftEnd, timezone)
	if err != nil {
		return fmt.Errorf("error formatting shift times: %w", err)
	}

	data := signupCancelledData{
		FirstName: firstName,
		EventName: eventName,
		Start:     *fmtStart,
		End:       *fmtEnd,
	}

	subject := "Signup Cancelled: " + eventName
	htmlBody, err := renderTemplate(signupCancelledHTMLTmpl, data)
	if err != nil {
		return err
	}
	textBody, err := renderTemplate(signupCancelledTextTmpl, data)
	if err != nil {
		return err
	}

	return mailer.SendEmail(ctx, email, subject, htmlBody, textBody)
}
func sendAccountCreated(ctx context.Context, mailer *Mailer, firstName, lastName, email, role string) error {
	data := accountCreatedData{
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
		Role:      role,
	}

	subject := "Your AARP Washington Volunteer System Account Has Been Created"
	htmlBody, err := renderTemplate(accountCreatedHTMLTmpl, data)
	if err != nil {
		return err
	}
	textBody, err := renderTemplate(accountCreatedTextTmpl, data)
	if err != nil {
		return err
	}

	return mailer.SendEmail(ctx, email, subject, htmlBody, textBody)
}

func sendAccountCreatedAdminNotification(ctx context.Context, DB *sql.DB, mailer *Mailer, firstName, lastName, email, role, createdBy string) error {
	data := accountCreatedAdminData{
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
		Role:      role,
		CreatedBy: createdBy,
	}

	subject := fmt.Sprintf("New Account Created: %s %s", firstName, lastName)
	htmlBody, err := renderTemplate(accountCreatedAdminHTMLTmpl, data)
	if err != nil {
		return err
	}
	textBody, err := renderTemplate(accountCreatedAdminTextTmpl, data)
	if err != nil {
		return err
	}

	rows, err := DB.QueryContext(ctx, "SELECT email FROM volunteers WHERE role = 'ADMINISTRATOR'")
	if err != nil {
		return fmt.Errorf("error fetching admin emails: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var adminEmail string
		if err := rows.Scan(&adminEmail); err != nil {
			log.Printf("Warning: error scanning admin email: %v", err)
			continue
		}
		if err := mailer.SendEmail(ctx, adminEmail, subject, htmlBody, textBody); err != nil {
			log.Printf("Warning: failed to send account creation notice to %s: %v", adminEmail, err)
		}
	}

	return nil
}

func sendNewAccountRequest(ctx context.Context, mailer *Mailer, adminEmails []string, firstName, lastName, email string) error {
	data := newAccountRequestData{
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
	}

	subject := fmt.Sprintf("New Volunteer Account Request — %s %s", firstName, lastName)
	htmlBody, err := renderTemplate(newAccountRequestHTMLTmpl, data)
	if err != nil {
		return err
	}
	textBody, err := renderTemplate(newAccountRequestTextTmpl, data)
	if err != nil {
		return err
	}

	for _, adminEmail := range adminEmails {
		if err := mailer.SendEmail(ctx, adminEmail, subject, htmlBody, textBody); err != nil {
			log.Printf("Warning: failed to send account request notification to %s: %v", adminEmail, err)
		}
	}
	return nil
}

func sendActivateAccountRequest(ctx context.Context, mailer *Mailer, adminEmails []string, firstName, lastName, email, existingName string, existingID int) error {
	data := activateAccountRequestData{
		FirstName:    firstName,
		LastName:     lastName,
		Email:        email,
		ExistingName: existingName,
		ExistingID:   existingID,
	}

	subject := fmt.Sprintf("Account Reactivation Request — %s %s (existing ID %d)", firstName, lastName, existingID)
	htmlBody, err := renderTemplate(activateAccountRequestHTMLTmpl, data)
	if err != nil {
		return err
	}
	textBody, err := renderTemplate(activateAccountRequestTextTmpl, data)
	if err != nil {
		return err
	}

	for _, adminEmail := range adminEmails {
		if err := mailer.SendEmail(ctx, adminEmail, subject, htmlBody, textBody); err != nil {
			log.Printf("Warning: failed to send account reactivation notification to %s: %v", adminEmail, err)
		}
	}
	return nil
}

// sendEventCancelledToVolunteer and sendEventCancelledToStaff are called from
// DeleteEvent in event_services.go, which has already fetched and formatted
// the shift times, so we accept pre-formatted strings here.

func sendEventCancelledToVolunteer(ctx context.Context, mailer *Mailer, firstName, eventName string, shifts []ShiftSummary, email string) error {
	data := eventCancelledVolunteerData{
		FirstName: firstName,
		EventName: eventName,
		Shifts:    shifts,
	}

	subject := eventName + " Has Been Cancelled"
	htmlBody, err := renderTemplate(eventCancelledVolunteerHTMLTmpl, data)
	if err != nil {
		return err
	}
	textBody, err := renderTemplate(eventCancelledVolunteerTextTmpl, data)
	if err != nil {
		return err
	}

	return mailer.SendEmail(ctx, email, subject, htmlBody, textBody)
}

func sendEventCancelledToStaff(ctx context.Context, mailer *Mailer, firstName, eventName string, shifts []ShiftSummary, email string) error {
	data := eventCancelledStaffData{
		FirstName: firstName,
		EventName: eventName,
		Shifts:    shifts,
	}

	subject := eventName + " Has Been Cancelled"
	htmlBody, err := renderTemplate(eventCancelledStaffHTMLTmpl, data)
	if err != nil {
		return err
	}
	textBody, err := renderTemplate(eventCancelledStaffTextTmpl, data)
	if err != nil {
		return err
	}

	return mailer.SendEmail(ctx, email, subject, htmlBody, textBody)
}
