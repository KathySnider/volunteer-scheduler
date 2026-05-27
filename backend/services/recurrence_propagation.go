package services

// ============================================================================
// Recurring-event propagation helpers
//
// When an opportunity or shift is created, updated, or deleted on an event
// that belongs to a recurrence group, the same change is passed on to all
// future instances in that group (recurrence_order > the source event's order).
//
// Linking mechanism
// -----------------
// opportunities.recurrence_template_id  UUID NULL
// shifts.recurrence_template_id         UUID NULL
//
// All copies of an opportunity (or shift) that were created together share
// the same UUID.  Delete / update operations find siblings by this UUID.
//
// Time-offset rule for shifts
// ---------------------------
// A shift's position in time is expressed relative to the source event's
// first scheduled date:
//
//   offset   = shift_start − event.first_date.start_datetime
//   duration = shift_end   − shift_start
//
// For each future peer event m:
//   new_shift_start = peer_m.first_date.start_datetime + offset
//   new_shift_end   = new_shift_start + duration
// ============================================================================

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// peerEvent describes one future instance of a recurring series that a change
// should be propagated to.
type peerEvent struct {
	EventID   int
	FirstDate time.Time // earliest event_date.start_date_time in UTC; zero if none
}

// eventGroupAndOrder returns the recurrence_group_id (as a UUID string) and
// recurrence_order for eventID.  Returns ("", 0, nil) when the event is not
// part of a recurrence group.
func eventGroupAndOrder(ctx context.Context, tx *sql.Tx, eventID int) (string, int, error) {
	var groupID string
	var order int
	err := tx.QueryRowContext(ctx,
		`SELECT COALESCE(recurrence_group_id::text, ''), COALESCE(recurrence_order, 0)
		 FROM events WHERE event_id = $1`,
		eventID,
	).Scan(&groupID, &order)
	return groupID, order, err
}

// eventFirstDate returns the UTC time of the earliest event_date for eventID.
// Returns a zero time.Time when the event has no dates yet.
func eventFirstDate(ctx context.Context, tx *sql.Tx, eventID int) (time.Time, error) {
	var t sql.NullTime
	if err := tx.QueryRowContext(ctx,
		`SELECT MIN(start_date_time) FROM event_dates WHERE event_id = $1`, eventID,
	).Scan(&t); err != nil {
		return time.Time{}, err
	}
	if !t.Valid {
		return time.Time{}, nil
	}
	return t.Time.UTC(), nil
}

// futurePeerEvents returns every event in groupID whose recurrence_order is
// strictly greater than afterOrder, along with that event's first scheduled
// date.  Results are ordered by recurrence_order ascending.
func futurePeerEvents(ctx context.Context, tx *sql.Tx, groupID string, afterOrder int) ([]peerEvent, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT e.event_id,
		       (SELECT MIN(ed.start_date_time)
		        FROM event_dates ed
		        WHERE ed.event_id = e.event_id)
		FROM events e
		WHERE e.recurrence_group_id = $1::uuid
		  AND e.recurrence_order > $2
		ORDER BY e.recurrence_order`,
		groupID, afterOrder,
	)
	if err != nil {
		return nil, fmt.Errorf("futurePeerEvents: %w", err)
	}
	defer rows.Close()

	var peers []peerEvent
	for rows.Next() {
		var p peerEvent
		var fd sql.NullTime
		if err := rows.Scan(&p.EventID, &fd); err != nil {
			return nil, fmt.Errorf("futurePeerEvents scan: %w", err)
		}
		if fd.Valid {
			p.FirstDate = fd.Time.UTC()
		}
		peers = append(peers, p)
	}
	return peers, rows.Err()
}

// adjustTimes returns RFC3339 start and end strings for a target peer event,
// applying the same offset that origStart had from srcFirstDate and preserving
// the shift duration.  When either first-date is zero (event has no dates),
// the original UTC times are returned unchanged.
func adjustTimes(origStart, origEnd time.Time, srcFirstDate, tgtFirstDate time.Time) (string, string) {
	if srcFirstDate.IsZero() || tgtFirstDate.IsZero() {
		return origStart.UTC().Format(time.RFC3339),
			origEnd.UTC().Format(time.RFC3339)
	}
	newStart := tgtFirstDate.Add(origStart.Sub(srcFirstDate))
	newEnd := newStart.Add(origEnd.Sub(origStart))
	return newStart.UTC().Format(time.RFC3339), newEnd.UTC().Format(time.RFC3339)
}
