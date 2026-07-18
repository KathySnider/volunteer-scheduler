package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
)

type ReminderScheduler struct {
	DB     *sql.DB
	mailer *Mailer
}

func NewReminderScheduler(db *sql.DB, mailer *Mailer) *ReminderScheduler {
	return &ReminderScheduler{
		DB:     db,
		mailer: mailer,
	}
}

func (s *ReminderScheduler) RunReminderScheduler(ctx context.Context) {
	if err := s.SendPendingReminders(ctx); err != nil {
		log.Printf("Reminder scheduler error: %v", err)
	}
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := s.SendPendingReminders(ctx); err != nil {
				log.Printf("Reminder scheduler error: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *ReminderScheduler) SendPendingReminders(ctx context.Context) error {

	query := `
		SELECT
			vs.volunteer_id,
			vs.shift_id,
			v.email,
		    v.first_name,
			e.event_name,
			s.shift_start,
			s.shift_end,
			opp.opportunity_is_virtual,
			ven.venue_name,
			ven.street_address,
			ven.city,
			ven.state,
			ven.zip_code,
			e.timezone,
			opp.pre_event_instructions,
			sc.first_name,
			sc.last_name,
			sc.position
		FROM volunteer_shifts vs
		JOIN volunteers v ON v.volunteer_id = vs.volunteer_id
		JOIN shifts s ON s.shift_id = vs.shift_id
		JOIN opportunities opp ON opp.opportunity_id = s.opportunity_id
		JOIN events e ON e.event_id = opp.event_id
		LEFT JOIN venues ven on ven.venue_id = e.venue_id
		LEFT JOIN staff sc ON sc.staff_id = e.staff_contact_id
		WHERE vs.cancelled_at IS NULL
					AND s.shift_start BETWEEN now() + interval '23 hours'
                        AND now() + interval '25 hours'
					AND vs.reminder_sent_at IS NULL
					AND v.is_active = true

		`
	rows, err := s.DB.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("error querying reminders data: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var volInt, shiftInt int
		var email, firstName, eventName, start, end sql.NullString
		var isVirtual bool
		var venName, address, city, state, zip sql.NullString
		var timezone string
		var instruct sql.NullString
		var scFirst, scLast, scPosition sql.NullString
		var fmtStart, fmtEnd *string

		err = rows.Scan(
			&volInt,
			&shiftInt,
			&email,
			&firstName,
			&eventName,
			&start,
			&end,
			&isVirtual,
			&venName,
			&address,
			&city,
			&state,
			&zip,
			&timezone,
			&instruct,
			&scFirst,
			&scLast,
			&scPosition)

		if err != nil {
			return fmt.Errorf("error scanning reminders data: %w", err)
		}

		if start.Valid && end.Valid {
			fmtStart, fmtEnd = formatStartEnd(start.String, end.String, timezone)
		} else {
			return fmt.Errorf("Shift start/end returned NULL from DB for shift: %d.", shiftInt)
		}

		staffContact := ""
		if scFirst.Valid && scLast.Valid {
			staffContact = scFirst.String + " " + scLast.String
			if scPosition.Valid && scPosition.String != "" {
				staffContact += " (" + scPosition.String + ")"
			}
		}

		reminder := &shiftReminderData{
			FirstName:    firstName.String,
			EventName:    eventName.String,
			Start:        *fmtStart,
			End:          *fmtEnd,
			IsVirtual:    isVirtual,
			VenueName:    venName.String,
			Address:      address.String,
			City:         city.String,
			State:        state.String,
			Zip:          zip.String,
			Instructions: instruct.String,
			StaffContact: staffContact,
		}

		err = SendShiftReminder(ctx, s.mailer, *reminder, email.String)
		if err != nil {
			log.Printf("Failed to send reminder email to %s: %v", email.String, err)
			continue
		}

		// Mark volunteer_shifts entry so we know this reminder has
		// been sent.
		update := `
			UPDATE volunteer_shifts
			SET reminder_sent_at = NOW()
			WHERE volunteer_id = $1 AND shift_id = $2
		`
		_, err = s.DB.ExecContext(ctx, update, volInt, shiftInt)
		if err != nil {
			log.Printf("Failed to update volunteer_shifts after email was sent; volId = %d; shiftId = %d. Error: %v", volInt, shiftInt, err)
			continue
		}
	}

	err = rows.Err()
	if err != nil {
		return fmt.Errorf("error iterating reminder rows: %w", err)
	}

	// No errors.
	return nil
}
