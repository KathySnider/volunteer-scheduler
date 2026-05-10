package services

// friendly_errors.go
//
// Converts raw PostgreSQL errors into user-safe messages.
// Call friendlyDBError(err) on any error returned from a DB operation
// before returning it from a service method.

import (
	"errors"
	"fmt"

	"github.com/lib/pq"
)

func friendlyDBError(err error) error {
	if err == nil {
		return nil
	}

	var pqErr *pq.Error
	if !errors.As(err, &pqErr) {
		// Not a Postgres error — pass through unchanged.
		return err
	}

	switch pqErr.Code {

	case "23505": // unique_violation
		switch pqErr.Constraint {
		case "volunteers_email_key":
			return fmt.Errorf("a volunteer with that email address already exists")
		case "staff_email_key":
			return fmt.Errorf("a staff member with that email address already exists")
		case "venues_street_address_city_state_key":
			return fmt.Errorf("a venue at that address already exists")
		case "funding_entities_name_key":
			return fmt.Errorf("a region with that name already exists")
		default:
			return fmt.Errorf("a record with those details already exists")
		}

	case "23503": // foreign_key_violation
		switch pqErr.Constraint {
		case "events_venue_id_fkey":
			return fmt.Errorf("this venue cannot be deleted because it is used by one or more events")
		case "events_funding_entity_id_fkey":
			return fmt.Errorf("this region cannot be deleted because it is used by one or more events")
		default:
			return fmt.Errorf("this record cannot be deleted because it is referenced by other records")
		}

	case "23514": // check_violation
		if pqErr.Table == "shifts" {
			return fmt.Errorf("shift end time must be after start time and max volunteers must be greater than zero")
		}
		return fmt.Errorf("the provided values are invalid")

	case "23502": // not_null_violation
		return fmt.Errorf("a required field is missing")
	}

	// Any other Postgres error — don't expose raw constraint/table names.
	return fmt.Errorf("an unexpected database error occurred")
}
