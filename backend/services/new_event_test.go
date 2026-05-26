package services

// Unit tests for the pure date-math helpers in new_event.go.
//
// These tests exercise createDatesForDays, createDatesForMonths,
// createDatesForLastWeekdays, createDatesForYears, eventDatesToTimes,
// and createDatesForPattern.
//
// Reference calendar (all UTC, no DST complications):
//   Jan  1 2026 = Thursday  (weekday 4)
//   Jan  7 2026 = Wednesday (weekday 3) — first Wednesday of January
//   Jan 14 2026 = Wednesday
//   Jan 21 2026 = Wednesday — third Wednesday of January
//   Jan 28 2026 = Wednesday — last  Wednesday of January
//   Jan 29 2026 = Thursday  — last  Thursday  of January
//
//   Feb  1 2026 = Sunday    (weekday 0)
//   Feb  4 2026 = Wednesday — first Wednesday of February
//   Feb 18 2026 = Wednesday — third Wednesday of February
//   Feb 25 2026 = Wednesday — last  Wednesday of February (Feb has 28 days)
//
//   Mar  1 2026 = Sunday    (weekday 0)
//   Mar  4 2026 = Wednesday — first Wednesday of March
//   Mar 18 2026 = Wednesday — third Wednesday of March
//   Mar 25 2026 = Wednesday — last  Wednesday of March

import (
	"testing"
	"time"

	"volunteer-scheduler/models"
)

// ============================================================================
// Test helpers
// ============================================================================

// utcDate creates a time.Time in UTC with seconds and nanoseconds zeroed.
func utcDate(year int, month time.Month, day, hour, min int) time.Time {
	return time.Date(year, month, day, hour, min, 0, 0, time.UTC)
}

// startDateForKey extracts the parsed start time for the given instance key
// from the map returned by createDatesFor* functions.
func startDateForKey(t *testing.T, result *map[int][]*models.NewEventDateInput, key int) time.Time {
	t.Helper()
	dates, ok := (*result)[key]
	if !ok {
		t.Fatalf("key %d not found in dates map", key)
	}
	if len(dates) == 0 {
		t.Fatalf("no dates for key %d", key)
	}
	ts, err := time.ParseInLocation(Layout, dates[0].StartDateTime, time.UTC)
	if err != nil {
		t.Fatalf("startDateForKey parse error: %v", err)
	}
	return ts
}

// endDateForKey extracts the parsed end time for the given instance key.
func endDateForKey(t *testing.T, result *map[int][]*models.NewEventDateInput, key int) time.Time {
	t.Helper()
	dates, ok := (*result)[key]
	if !ok {
		t.Fatalf("key %d not found in dates map", key)
	}
	if len(dates) == 0 {
		t.Fatalf("no dates for key %d", key)
	}
	ts, err := time.ParseInLocation(Layout, dates[0].EndDateTime, time.UTC)
	if err != nil {
		t.Fatalf("endDateForKey parse error: %v", err)
	}
	return ts
}

// singleTuple wraps one start/end pair into a []timeTuple.
func singleTuple(start, end time.Time) []timeTuple {
	return []timeTuple{{start: start, end: end}}
}

// ============================================================================
// createDatesForDays
// ============================================================================

func TestCreateDatesForDays_Daily(t *testing.T) {
	og := singleTuple(utcDate(2026, time.January, 7, 8, 0), utcDate(2026, time.January, 7, 10, 0))
	result := createDatesForDays(og, 1, 3)

	want := []time.Time{
		utcDate(2026, time.January, 7, 8, 0),
		utcDate(2026, time.January, 8, 8, 0),
		utcDate(2026, time.January, 9, 8, 0),
	}
	for i, w := range want {
		if got := startDateForKey(t, result, i+1); !got.Equal(w) {
			t.Errorf("daily instance %d: want %v, got %v", i+1, w, got)
		}
	}
}

func TestCreateDatesForDays_Weekly(t *testing.T) {
	og := singleTuple(utcDate(2026, time.January, 7, 8, 0), utcDate(2026, time.January, 7, 10, 0))
	result := createDatesForDays(og, 7, 3)

	want := []time.Time{
		utcDate(2026, time.January, 7, 8, 0),
		utcDate(2026, time.January, 14, 8, 0),
		utcDate(2026, time.January, 21, 8, 0),
	}
	for i, w := range want {
		if got := startDateForKey(t, result, i+1); !got.Equal(w) {
			t.Errorf("weekly instance %d: want %v, got %v", i+1, w, got)
		}
	}
}

func TestCreateDatesForDays_Biweekly(t *testing.T) {
	og := singleTuple(utcDate(2026, time.January, 7, 8, 0), utcDate(2026, time.January, 7, 10, 0))
	result := createDatesForDays(og, 14, 3)

	want := []time.Time{
		utcDate(2026, time.January, 7, 8, 0),
		utcDate(2026, time.January, 21, 8, 0),
		utcDate(2026, time.February, 4, 8, 0),
	}
	for i, w := range want {
		if got := startDateForKey(t, result, i+1); !got.Equal(w) {
			t.Errorf("biweekly instance %d: want %v, got %v", i+1, w, got)
		}
	}
}

func TestCreateDatesForDays_KeysAreOneBased(t *testing.T) {
	og := singleTuple(utcDate(2026, time.January, 7, 8, 0), utcDate(2026, time.January, 7, 10, 0))
	result := createDatesForDays(og, 1, 5)

	if len(*result) != 5 {
		t.Fatalf("want 5 entries, got %d", len(*result))
	}
	for k := 1; k <= 5; k++ {
		if _, ok := (*result)[k]; !ok {
			t.Errorf("expected key %d in result map", k)
		}
	}
}

func TestCreateDatesForDays_PreservesTimeOfDay(t *testing.T) {
	og := singleTuple(utcDate(2026, time.January, 7, 14, 30), utcDate(2026, time.January, 7, 16, 0))
	result := createDatesForDays(og, 7, 2)

	got := startDateForKey(t, result, 2)
	if got.Hour() != 14 || got.Minute() != 30 {
		t.Errorf("time of day not preserved: want 14:30, got %02d:%02d", got.Hour(), got.Minute())
	}
}

func TestCreateDatesForDays_ShiftsEndDate(t *testing.T) {
	og := singleTuple(utcDate(2026, time.January, 7, 8, 0), utcDate(2026, time.January, 7, 10, 0))
	result := createDatesForDays(og, 7, 2)

	wantEnd := utcDate(2026, time.January, 14, 10, 0)
	if got := endDateForKey(t, result, 2); !got.Equal(wantEnd) {
		t.Errorf("end date for instance 2: want %v, got %v", wantEnd, got)
	}
}

// ============================================================================
// createDatesForMonths  (nth weekday)
// ============================================================================

func TestCreateDatesForMonths_ThirdWednesday(t *testing.T) {
	// Jan 21, 2026 is the 3rd Wednesday of January.
	og := singleTuple(utcDate(2026, time.January, 21, 8, 0), utcDate(2026, time.January, 21, 10, 0))
	result := createDatesForMonths(og, 2, time.UTC, 3) // weeks=2 → 3rd occurrence

	want := []time.Time{
		utcDate(2026, time.January, 21, 8, 0),
		utcDate(2026, time.February, 18, 8, 0),
		utcDate(2026, time.March, 18, 8, 0),
	}
	for i, w := range want {
		if got := startDateForKey(t, result, i+1); !got.Equal(w) {
			t.Errorf("3rd-Wednesday instance %d: want %v, got %v", i+1, w, got)
		}
	}
}

func TestCreateDatesForMonths_FirstWednesday(t *testing.T) {
	// Jan 7, 2026 is the 1st Wednesday of January.
	og := singleTuple(utcDate(2026, time.January, 7, 8, 0), utcDate(2026, time.January, 7, 10, 0))
	result := createDatesForMonths(og, 0, time.UTC, 3) // weeks=0 → 1st occurrence

	want := []time.Time{
		utcDate(2026, time.January, 7, 8, 0),
		utcDate(2026, time.February, 4, 8, 0),
		utcDate(2026, time.March, 4, 8, 0),
	}
	for i, w := range want {
		if got := startDateForKey(t, result, i+1); !got.Equal(w) {
			t.Errorf("1st-Wednesday instance %d: want %v, got %v", i+1, w, got)
		}
	}
}

func TestCreateDatesForMonths_PreservesTimeOfDay(t *testing.T) {
	og := singleTuple(utcDate(2026, time.January, 21, 14, 30), utcDate(2026, time.January, 21, 16, 0))
	result := createDatesForMonths(og, 2, time.UTC, 2)

	got := startDateForKey(t, result, 2)
	if got.Hour() != 14 || got.Minute() != 30 {
		t.Errorf("time of day not preserved: want 14:30, got %02d:%02d", got.Hour(), got.Minute())
	}
}

// ============================================================================
// createDatesForLastWeekdays
// ============================================================================

func TestCreateDatesForLastWeekdays_LastWednesday(t *testing.T) {
	// Jan 28, 2026 is the last Wednesday of January.
	// Note: Jan 29 would overflow when adding 1 month — see OverflowBug test.
	og := singleTuple(utcDate(2026, time.January, 28, 8, 0), utcDate(2026, time.January, 28, 10, 0))
	result := createDatesForMonths(og, 4, time.UTC, 3)

	want := []time.Time{
		utcDate(2026, time.January, 28, 8, 0),
		utcDate(2026, time.February, 25, 8, 0),
		utcDate(2026, time.March, 25, 8, 0),
	}
	for i, w := range want {
		if got := startDateForKey(t, result, i+1); !got.Equal(w) {
			t.Errorf("last-Wednesday instance %d: want %v, got %v", i+1, w, got)
		}
	}
}

// TestCreateDatesForLastWeekdays_ShortMonth verifies that "last weekday" works
// correctly when the start date falls on day 29–31 of a long month and the
// following month is shorter (the classic Feb overflow case).
//
// Jan 29, 2026 is the last Thursday of January.  Adding one month naively
// would give Feb 29, which doesn't exist — the implementation must land on
// Feb 26 (the last Thursday of February 2026).
func TestCreateDatesForLastWeekdays_ShortMonth(t *testing.T) {
	// Jan 29, 2026 is the last Thursday of January.
	og := singleTuple(utcDate(2026, time.January, 29, 8, 0), utcDate(2026, time.January, 29, 10, 0))
	result := createDatesForMonths(og, 4, time.UTC, 2)

	// Feb 2026 last Thursday should be Feb 26.
	want := utcDate(2026, time.February, 26, 8, 0)
	if got := startDateForKey(t, result, 2); !got.Equal(want) {
		t.Errorf("want %v, got %v", want, got)
	}
}

// ============================================================================
// createDatesForYears
// ============================================================================

func TestCreateDatesForYears_Basic(t *testing.T) {
	og := singleTuple(utcDate(2026, time.January, 7, 8, 0), utcDate(2026, time.January, 7, 10, 0))
	result := createDatesForYears(og, 3)

	want := []time.Time{
		utcDate(2026, time.January, 7, 8, 0),
		utcDate(2027, time.January, 7, 8, 0),
		utcDate(2028, time.January, 7, 8, 0),
	}
	for i, w := range want {
		if got := startDateForKey(t, result, i+1); !got.Equal(w) {
			t.Errorf("yearly instance %d: want %v, got %v", i+1, w, got)
		}
	}
}

func TestCreateDatesForYears_Count(t *testing.T) {
	og := singleTuple(utcDate(2026, time.March, 15, 9, 0), utcDate(2026, time.March, 15, 11, 0))
	result := createDatesForYears(og, 5)
	if len(*result) != 5 {
		t.Fatalf("want 5 yearly instances, got %d", len(*result))
	}
}

// ============================================================================
// eventDatesToTimes
// ============================================================================

func TestEventDatesToTimes_Basic(t *testing.T) {
	input := []*models.NewEventDateInput{
		{StartDateTime: "2026-01-07 08:00:00", EndDateTime: "2026-01-07 10:00:00"},
	}
	tuples, err := eventDatesToTimes(input, "America/Los_Angeles")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tuples) != 1 {
		t.Fatalf("want 1 tuple, got %d", len(tuples))
	}

	loc, _ := time.LoadLocation("America/Los_Angeles")
	wantStart := time.Date(2026, time.January, 7, 8, 0, 0, 0, loc)
	wantEnd := time.Date(2026, time.January, 7, 10, 0, 0, 0, loc)

	if !tuples[0].start.Equal(wantStart) {
		t.Errorf("start: want %v, got %v", wantStart, tuples[0].start)
	}
	if !tuples[0].end.Equal(wantEnd) {
		t.Errorf("end: want %v, got %v", wantEnd, tuples[0].end)
	}
}

func TestEventDatesToTimes_MultipleInputs(t *testing.T) {
	input := []*models.NewEventDateInput{
		{StartDateTime: "2026-01-07 08:00:00", EndDateTime: "2026-01-07 10:00:00"},
		{StartDateTime: "2026-01-08 09:00:00", EndDateTime: "2026-01-08 11:00:00"},
	}
	tuples, err := eventDatesToTimes(input, "UTC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tuples) != 2 {
		t.Fatalf("want 2 tuples, got %d — check for the make(len) append bug", len(tuples))
	}
}

func TestEventDatesToTimes_BadTimezone(t *testing.T) {
	input := []*models.NewEventDateInput{
		{StartDateTime: "2026-01-07 08:00:00", EndDateTime: "2026-01-07 10:00:00"},
	}
	_, err := eventDatesToTimes(input, "Not/ATimezone")
	if err == nil {
		t.Error("want error for invalid timezone, got nil")
	}
}

// ============================================================================
// createDatesForPattern  (dispatch)
// ============================================================================

func TestCreateDatesForPattern_Daily_Default(t *testing.T) {
	input := []*models.NewEventDateInput{
		{StartDateTime: "2026-01-07 08:00:00", EndDateTime: "2026-01-07 10:00:00"},
	}
	max := 3
	recur := models.RecurrenceInput{Pattern: models.RecurrencePatternDaily, MaxOccurrences: &max}

	result, err := createDatesForPattern(input, "UTC", recur)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(*result) != 3 {
		t.Fatalf("want 3 instances, got %d", len(*result))
	}
}

func TestCreateDatesForPattern_Weekly_DefaultMax(t *testing.T) {
	// Omitting maxOccurrences should default to 52 for weekly.
	input := []*models.NewEventDateInput{
		{StartDateTime: "2026-01-07 08:00:00", EndDateTime: "2026-01-07 10:00:00"},
	}
	recur := models.RecurrenceInput{Pattern: models.RecurrencePatternWeekly}

	result, err := createDatesForPattern(input, "UTC", recur)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(*result) != 52 {
		t.Fatalf("want 52 weekly instances (1 year default), got %d", len(*result))
	}
}

func TestCreateDatesForPattern_Yearly_RequiresMaxOccurrences(t *testing.T) {
	input := []*models.NewEventDateInput{
		{StartDateTime: "2026-01-07 08:00:00", EndDateTime: "2026-01-07 10:00:00"},
	}
	recur := models.RecurrenceInput{Pattern: models.RecurrencePatternYearly} // no max

	_, err := createDatesForPattern(input, "UTC", recur)
	if err == nil {
		t.Error("want error for YEARLY with no maxOccurrences, got nil")
	}
}

// TestCreateDatesForPattern_Monthly_NilOrdinal verifies that passing a nil
// WeekdayOrdinal for a MONTHLY pattern returns an error rather than panicking.
func TestCreateDatesForPattern_Monthly_NilOrdinal(t *testing.T) {

	input := []*models.NewEventDateInput{
		{StartDateTime: "2026-01-21 08:00:00", EndDateTime: "2026-01-21 10:00:00"},
	}
	recur := models.RecurrenceInput{
		Pattern:        models.RecurrencePatternMonthly,
		WeekdayOrdinal: nil,
	}
	_, err := createDatesForPattern(input, "UTC", recur)
	if err == nil {
		t.Error("want error for MONTHLY with nil weekdayOrdinal, got nil")
	}
}

func TestCreateDatesForPattern_InvalidPattern(t *testing.T) {
	input := []*models.NewEventDateInput{
		{StartDateTime: "2026-01-07 08:00:00", EndDateTime: "2026-01-07 10:00:00"},
	}
	recur := models.RecurrenceInput{Pattern: models.RecurrencePattern("FORTNIGHTLY")}

	_, err := createDatesForPattern(input, "UTC", recur)
	if err == nil {
		t.Error("want error for unknown pattern, got nil")
	}
}
