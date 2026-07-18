package services

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"
	"volunteer-scheduler/models"

	"github.com/google/uuid"
)

// These 2 "create" functions are helpers, but they are the workhorses of CreateEvent.

// Create exactly one instance of an event - no uuid, no order, no extra dates.
func (s *EventService) createSingleEvent(ctx context.Context, newEvent models.NewEventInput, contactIdPtr *int, virtualEvent bool, venueIdPtr *int) (*models.MutationResult, error) {
	var query string
	var eventInt int

	// Add the event and it's dates inside a transaction.

	// Get a Tx for making transaction requests.
	var tx *sql.Tx

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		err = fmt.Errorf("error starting transaction: %w", err)
		return nil, err
	}
	// Defer a rollback in case anything fails.
	defer tx.Rollback()

	// The rollback is for insurance. The rollback will occur if we
	// leave the scope of the transaction before it has ended. For
	// good DB practice, DO NOT RETURN while inside of a transaction.

	query = `
		INSERT INTO events (event_name, description, event_is_virtual, staff_contact_id, venue_id, timezone, funding_entity_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING event_id
	`
	err = tx.QueryRowContext(ctx, query, newEvent.Name, newEvent.Description, virtualEvent, contactIdPtr, venueIdPtr, newEvent.Timezone, newEvent.FundingEntityID).Scan(&eventInt)

	if err == nil {
		// Event was inserted. Add the dates.
		err = addNewEventDates(ctx, newEvent.EventDates, eventInt, newEvent.Timezone, tx)
	} else {
		// Save all of the information about what failed.
		err = fmt.Errorf("error inserting the event: %w", err)
	}

	if err == nil {
		err = addServiceTypesToEvent(ctx, tx, eventInt, newEvent.ServiceTypes)
	} else {
		err = fmt.Errorf("error adding dates to the event: %w", err)
	}

	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("transaction failed and was rolled back: %w", err)
	}

	// All good. Commit and return the new event ID.
	err = tx.Commit()
	if err != nil {
		return nil, fmt.Errorf("error committing transaction: %w", err)
	}

	return &models.MutationResult{
		Success: true,
		Message: ptrString("Event successfully created."),
		ID:      ptrString(strconv.Itoa(eventInt)),
	}, nil
}

// Create an instance of a recurring event.
func (s *EventService) createEventRecurrence(ctx context.Context, tx *sql.Tx, ev models.NewEventInput, contactIdPtr *int, virtualEvent bool, venueIdPtr *int, evDates []*models.NewEventDateInput, groupId uuid.UUID, groupOrder int) (*models.MutationResult, error) {
	var query string
	var eventInt int

	// Create the event first. We need the id to continue.
	query = `
		INSERT INTO events (
			event_name,
			description, 
			event_is_virtual, 
			staff_contact_id,
			venue_id,
			timezone, 
			funding_entity_id,
			recurrence_group_id,
			recurrence_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING event_id
	`
	err := tx.QueryRowContext(ctx, query,
		ev.Name,
		ev.Description,
		virtualEvent,
		contactIdPtr,
		venueIdPtr,
		ev.Timezone,
		ev.FundingEntityID,
		groupId.String(),
		groupOrder).Scan(&eventInt)

	if err != nil {
		return nil, friendlyDBError(err)
	}
	err = addNewEventDates(ctx, evDates, eventInt, ev.Timezone, tx)
	if err != nil {
		return nil, fmt.Errorf("error adding event dates: %w", err)
	}
	err = addServiceTypesToEvent(ctx, tx, eventInt, ev.ServiceTypes)
	if err != nil {
		return nil, fmt.Errorf("error adding dates to the event: %w", err)
	}

	return &models.MutationResult{
		Success: true,
		Message: ptrString("Instance successfully created."),
		ID:      ptrString(strconv.Itoa(eventInt)),
	}, nil
}

// These helpers are used to create a new event - whether one-time events or recurring events.

type timeTuple struct {
	start time.Time
	end   time.Time
}

// The createDatesFor... functions create a map of event dates with an
// entry for each event to be created. They all sart with the dates from
// NewEvent. Note that the EventDates (and ogDates) is a slice, and the
// map returned is a map of slices, because there may be 1 or many dates
// for a single event occurrence. We don't make any assumptions about the
// event dates - i.e., they may or may not be continuous, and the start
// and end datetime in each tuple may or may not be on the same date. We
// don't make judgements; we just try to handle the dates we are given.

func createDatesForPattern(evDates []*models.NewEventDateInput, timezone string, recur models.RecurrenceInput) (*map[int][]*models.NewEventDateInput, error) {

	var evDatesMap *map[int][]*models.NewEventDateInput

	// Start by converting the event dates (strings) into tuples of time.Time,
	// so we can use go's time package to manipulate the dates.
	ogDates, err := eventDatesToTimes(evDates, timezone)
	if err != nil {
		return nil, fmt.Errorf("Invalid datetimes in EventDates.")
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("unable to create recurring days; invalid timezone: %w", err)
	}

	var max int

	switch recur.Pattern {
	case models.RecurrencePatternDaily:
		{
			days := 1
			if recur.MaxOccurrences == nil {
				max = 365
			} else {
				max = *recur.MaxOccurrences
			}
			evDatesMap = createDatesForDays(ogDates, days, max)
		}
	case models.RecurrencePatternWeekly:
		{
			days := 7
			if recur.MaxOccurrences == nil {
				max = 52
			} else {
				max = *recur.MaxOccurrences
			}
			evDatesMap = createDatesForDays(ogDates, days, max)
		}
	case models.RecurrencePatternBiweekly:
		{
			days := 14
			if recur.MaxOccurrences == nil {
				max = 26
			} else {
				max = *recur.MaxOccurrences
			}
			evDatesMap = createDatesForDays(ogDates, days, max)
		}
	case models.RecurrencePatternMonthly:
		{
			if recur.MaxOccurrences == nil {
				max = 12
			} else {
				max = *recur.MaxOccurrences
			}

			if recur.WeekdayOrdinal == nil {
				return nil, fmt.Errorf("Weeday recurring events require the weekday ordinal parameter.")
			}
			switch *recur.WeekdayOrdinal {
			case models.WeekdayOrdinalFirst:
				// The second parameter to createDatesForMonths is the number
				// of weeks to add to the first weekday.
				evDatesMap = createDatesForMonths(ogDates, 0, loc, max)
			case models.WeekdayOrdinalSecond:
				evDatesMap = createDatesForMonths(ogDates, 1, loc, max)
			case models.WeekdayOrdinalThird:
				evDatesMap = createDatesForMonths(ogDates, 2, loc, max)
			case models.WeekdayOrdinalFourth:
				evDatesMap = createDatesForMonths(ogDates, 3, loc, max)
			case models.WeekdayOrdinalLast:
				// Treat last weekday as the 5th. Backs up if there are only 4.
				evDatesMap = createDatesForMonths(ogDates, 4, loc, max)
			default:
				return nil, fmt.Errorf("invalid weekday ordinal.")
			}
		}
	case models.RecurrencePatternYearly:
		{
			if recur.MaxOccurrences == nil {
				return nil, fmt.Errorf("Yearly occurrences requires a maximum number.")
			}
			evDatesMap = createDatesForYears(ogDates, *recur.MaxOccurrences)

		}
	default:
		{
			return nil, fmt.Errorf("Invalid recurrence pattern for create event.")
		}
	}
	return evDatesMap, nil
}

// Create dates for instances with a fixed number of days between them.
func createDatesForDays(ogDates []timeTuple, days int, max int) *map[int][]*models.NewEventDateInput {

	allEventDates := map[int][]*models.NewEventDateInput{}

	// The slice of time tuples (ogDates) represents the dates of the first
	// instance of the event. Loop over the number of events to be created.
	for instance := range max {

		currDates := addDays(ogDates, instance*days)

		// Convert datetimes back to strings.
		evDates := []*models.NewEventDateInput{}
		for _, currDate := range currDates {
			var evDate models.NewEventDateInput

			evDate.StartDateTime = currDate.start.Format(Layout)
			evDate.EndDateTime = currDate.end.Format(Layout)

			// Add the formatted dates to the slice for this occurrence.
			evDates = append(evDates, &evDate)
		}

		// Add the slice of dates (in NewEventDateInput form) to the map.
		// key will be used for the group order, so needs to start with 1.
		key := instance + 1
		allEventDates[key] = evDates
	}

	return &allEventDates
}

func createDatesForMonths(ogDates []timeTuple, weeks int, loc *time.Location, max int) *map[int][]*models.NewEventDateInput {

	allEventDates := map[int][]*models.NewEventDateInput{}

	for instance := range max {

		currDates := addMonthsToWeekdays(ogDates, weeks, instance, loc)

		// Convert datetimes back to strings.
		evDates := []*models.NewEventDateInput{}
		for _, currDate := range currDates {
			var evDate models.NewEventDateInput

			evDate.StartDateTime = currDate.start.Format(Layout)
			evDate.EndDateTime = currDate.end.Format(Layout)

			// Add the formatted dates to the slice for this
			// occurrence.
			evDates = append(evDates, &evDate)
		}

		key := instance + 1
		allEventDates[key] = evDates
	}

	return &allEventDates
}

func createDatesForYears(ogDates []timeTuple, max int) *map[int][]*models.NewEventDateInput {
	allEventDates := map[int][]*models.NewEventDateInput{}

	// We now have a slice of time tuples (ogDates) that represents the dates
	// of the first event. Loop over the number of events to be created (max).
	for instance := range max {

		currDates := addYears(ogDates, instance)

		// Convert datetimes back to strings.
		evDates := []*models.NewEventDateInput{}
		for _, currDate := range currDates {
			var evDate models.NewEventDateInput

			evDate.StartDateTime = currDate.start.Format(Layout)
			evDate.EndDateTime = currDate.end.Format(Layout)

			// Add the formatted dates to the slice for this occurrence.
			evDates = append(evDates, &evDate)
		}

		key := instance + 1
		allEventDates[key] = evDates
	}

	return &allEventDates

}

func addDays(inDates []timeTuple, days int) []timeTuple {
	outDates := []timeTuple{}

	for _, inDate := range inDates {
		var outDate timeTuple

		outDate.start = inDate.start.AddDate(0, 0, days)
		outDate.end = inDate.end.AddDate(0, 0, days)
		outDates = append(outDates, outDate)
	}
	return outDates
}

func addMonthsToWeekdays(inDates []timeTuple, weeks int, mos int, loc *time.Location) []timeTuple {
	outDates := []timeTuple{}

	for _, inDate := range inDates {
		var outDate timeTuple

		outDate.start = addMonthsToWeekday(inDate.start, mos, weeks, loc)
		outDate.end = addMonthsToWeekday(inDate.end, mos, weeks, loc)
		outDates = append(outDates, outDate)
	}
	return outDates
}

func addMonthsToWeekday(inDate time.Time, mos int, weeks int, loc *time.Location) time.Time {

	// Find out what weekday we want from the date passed in.
	wd := int(inDate.Weekday())

	// Advance by months from the 1st of the current month to avoid day
	// overflow (e.g. Jan 29 + 1 month would give Mar 1, not Feb 28).
	firstOfMonth := time.Date(inDate.Year(), inDate.Month(), 1, inDate.Hour(), inDate.Minute(), 0, 0, loc)
	target := firstOfMonth.AddDate(0, mos, 0)

	// Save the target month.
	tm := target.Month()

	// Get the first day of the target month.
	first := time.Date(target.Year(), tm, 1, target.Hour(), target.Minute(), 0, 0, loc)

	// Now get the first weekday of the month that matches our weekday.
	daysToWd := (wd - int(first.Weekday()) + 7) % 7
	firstWd := first.AddDate(0, 0, daysToWd)

	// Add 7 days for each week passed in.
	outDate := firstWd.AddDate(0, 0, weeks*7)

	// We may have overflowed into the following month. This can happen
	// if we want the last weekday of the month, so we add 4 weeks to the
	// first weekday. If there are 5 instances of our weekday in that
	// month, we are golden, but, if there are only 4, we'll be 1 week
	// past our target.
	// If so, back up 1 week.
	if outDate.Month() != tm {
		outDate = outDate.AddDate(0, 0, -7)
	}

	return outDate
}

func addYears(inDates []timeTuple, years int) []timeTuple {
	outDates := []timeTuple{}

	for _, inDate := range inDates {
		var outDate timeTuple

		outDate.start = inDate.start.AddDate(years, 0, 0)
		outDate.end = inDate.end.AddDate(years, 0, 0)

		outDates = append(outDates, outDate)
	}
	return outDates
}

// These "AddNew" functions are called internally, when creating a new event. The dates must be added
// within a transaction, and the event id must be provided, since, when the date elements were populated,
// the client didn't know the event Id.
func addNewEventDates(ctx context.Context, dates []*models.NewEventDateInput, eventId int, timezone string, tx *sql.Tx) error {
	for i := 0; i < len(dates); i++ {
		err := addNewEventDate(ctx, dates[i], eventId, timezone, tx)
		if err != nil {
			return fmt.Errorf("error inserting date with index %d: %w", i, err)
		}
	}

	// No errors.
	return nil
}

func addNewEventDate(ctx context.Context, dates *models.NewEventDateInput, eventId int, timezone string, tx *sql.Tx) error {
	var startUTC, endUTC *string
	startUTC, err := DateTimeToUTC(dates.StartDateTime, timezone)
	if err == nil {
		endUTC, err = DateTimeToUTC(dates.EndDateTime, timezone)
	}
	if err != nil {
		return err
	}

	insert := `
		INSERT INTO event_dates (event_id, start_date_time, end_date_time)
		VALUES ($1, $2, $3)
		RETURNING event_date_id
	`

	var eventDateInt int
	err = tx.QueryRowContext(ctx, insert, eventId, startUTC, endUTC).Scan(&eventDateInt)
	if err != nil {
		return fmt.Errorf("error inserting datetimes: %w", err)
	}

	// No errors.
	return nil
}

func addServiceTypesToEvent(ctx context.Context, tx *sql.Tx, eventId int, serviceTypes []int) error {

	query := `
		INSERT INTO event_service_types (event_id, service_type_id)
		VALUES ($1, $2)
		`
	for _, serviceTypeId := range serviceTypes {
		_, err := tx.ExecContext(ctx, query, eventId, serviceTypeId)

		if err != nil {
			return fmt.Errorf("error adding service type to event: %w", err)
		}
	}

	// No errors.
	return nil
}

// Turn event datetimes (strings) into time.Time (using the timezone in the event), so we can
// manipulate them using go's time package.
func eventDatesToTimes(evDates []*models.NewEventDateInput, timezone string) ([]timeTuple, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("bad timezone in new event: %w", err)
	}
	ogDates := []timeTuple{}
	for _, evDate := range evDates {
		var ogDate timeTuple
		ogDate.start, err = time.ParseInLocation(Layout, evDate.StartDateTime, loc)
		if err == nil {
			ogDate.end, err = time.ParseInLocation(Layout, evDate.EndDateTime, loc)
		}
		if err != nil {
			return nil, fmt.Errorf("bad event dates in new event: %w", err)
		}
		ogDates = append(ogDates, ogDate)
	}

	return ogDates, nil
}
