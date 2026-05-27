package services

// ============================================================================
// Unit tests for recurrence_propagation.go helpers.
//
// adjustTimes is a pure function — no database or network calls needed.
//
// Rules under test:
//   1. Normal case: start offset from srcFirstDate is preserved on target;
//      shift duration is preserved.
//   2. Zero srcFirstDate → original UTC times returned unchanged.
//   3. Zero tgtFirstDate → original UTC times returned unchanged.
//   4. Negative offset (shift starts before its event) is handled correctly.
//   5. Output strings are valid RFC3339.
// ============================================================================

import (
	"testing"
	"time"
)

// ============================================================================
// adjustTimes
// ============================================================================

// utcTime is a small helper that builds a UTC time.Time from parts.
func utcTime(year int, month time.Month, day, hour, min int) time.Time {
	return time.Date(year, month, day, hour, min, 0, 0, time.UTC)
}

func TestAdjustTimes_OffsetPreserved(t *testing.T) {
	// Source event first date: 2035-01-08 09:00 UTC
	// Shift starts 1h after event → 10:00 UTC, ends 2h later → 12:00 UTC
	srcFirst := utcTime(2035, time.January, 8, 9, 0)
	origStart := utcTime(2035, time.January, 8, 10, 0)
	origEnd := utcTime(2035, time.January, 8, 12, 0)

	// Target event first date: 7 days later → 2035-01-15 09:00 UTC
	tgtFirst := utcTime(2035, time.January, 15, 9, 0)

	gotStart, gotEnd := adjustTimes(origStart, origEnd, srcFirst, tgtFirst)

	wantStart := "2035-01-15T10:00:00Z"
	wantEnd := "2035-01-15T12:00:00Z"

	if gotStart != wantStart {
		t.Errorf("start: want %s, got %s", wantStart, gotStart)
	}
	if gotEnd != wantEnd {
		t.Errorf("end: want %s, got %s", wantEnd, gotEnd)
	}
}

func TestAdjustTimes_DurationPreserved(t *testing.T) {
	// 90-minute shift; offset from event = +30 min
	srcFirst := utcTime(2035, time.February, 5, 9, 0)
	origStart := utcTime(2035, time.February, 5, 9, 30)
	origEnd := utcTime(2035, time.February, 5, 11, 0)

	tgtFirst := utcTime(2035, time.February, 12, 9, 0)

	gotStart, gotEnd := adjustTimes(origStart, origEnd, srcFirst, tgtFirst)

	// Parse back and check the duration.
	ts, err := time.Parse(time.RFC3339, gotStart)
	if err != nil {
		t.Fatalf("parsing gotStart: %v", err)
	}
	te, err := time.Parse(time.RFC3339, gotEnd)
	if err != nil {
		t.Fatalf("parsing gotEnd: %v", err)
	}

	const wantDur = 90 * time.Minute
	if got := te.Sub(ts); got != wantDur {
		t.Errorf("duration: want %v, got %v", wantDur, got)
	}
}

func TestAdjustTimes_ZeroSrcFirstDate_ReturnOriginal(t *testing.T) {
	origStart := utcTime(2035, time.March, 5, 10, 0)
	origEnd := utcTime(2035, time.March, 5, 12, 0)
	tgtFirst := utcTime(2035, time.March, 12, 10, 0)

	gotStart, gotEnd := adjustTimes(origStart, origEnd, time.Time{}, tgtFirst)

	wantStart := origStart.UTC().Format(time.RFC3339)
	wantEnd := origEnd.UTC().Format(time.RFC3339)

	if gotStart != wantStart {
		t.Errorf("start: want %s, got %s", wantStart, gotStart)
	}
	if gotEnd != wantEnd {
		t.Errorf("end: want %s, got %s", wantEnd, gotEnd)
	}
}

func TestAdjustTimes_ZeroTgtFirstDate_ReturnOriginal(t *testing.T) {
	srcFirst := utcTime(2035, time.March, 5, 9, 0)
	origStart := utcTime(2035, time.March, 5, 10, 0)
	origEnd := utcTime(2035, time.March, 5, 12, 0)

	gotStart, gotEnd := adjustTimes(origStart, origEnd, srcFirst, time.Time{})

	wantStart := origStart.UTC().Format(time.RFC3339)
	wantEnd := origEnd.UTC().Format(time.RFC3339)

	if gotStart != wantStart {
		t.Errorf("start: want %s, got %s", wantStart, gotStart)
	}
	if gotEnd != wantEnd {
		t.Errorf("end: want %s, got %s", wantEnd, gotEnd)
	}
}

func TestAdjustTimes_NegativeOffset(t *testing.T) {
	// Shift begins 30 minutes BEFORE the event's first scheduled date (offset = −30m).
	// This is unusual but the math should still work.
	srcFirst := utcTime(2035, time.April, 2, 9, 0)
	origStart := utcTime(2035, time.April, 2, 8, 30) // −30m offset
	origEnd := utcTime(2035, time.April, 2, 9, 0)    // exactly at event start

	tgtFirst := utcTime(2035, time.April, 9, 9, 0)

	gotStart, gotEnd := adjustTimes(origStart, origEnd, srcFirst, tgtFirst)

	wantStart := "2035-04-09T08:30:00Z"
	wantEnd := "2035-04-09T09:00:00Z"

	if gotStart != wantStart {
		t.Errorf("start: want %s, got %s", wantStart, gotStart)
	}
	if gotEnd != wantEnd {
		t.Errorf("end: want %s, got %s", wantEnd, gotEnd)
	}
}

func TestAdjustTimes_OutputIsRFC3339(t *testing.T) {
	srcFirst := utcTime(2035, time.May, 7, 9, 0)
	origStart := utcTime(2035, time.May, 7, 10, 0)
	origEnd := utcTime(2035, time.May, 7, 12, 0)
	tgtFirst := utcTime(2035, time.May, 14, 9, 0)

	gotStart, gotEnd := adjustTimes(origStart, origEnd, srcFirst, tgtFirst)

	if _, err := time.Parse(time.RFC3339, gotStart); err != nil {
		t.Errorf("gotStart is not RFC3339: %q — %v", gotStart, err)
	}
	if _, err := time.Parse(time.RFC3339, gotEnd); err != nil {
		t.Errorf("gotEnd is not RFC3339: %q — %v", gotEnd, err)
	}
}

func TestAdjustTimes_MultiWeekOffset(t *testing.T) {
	// Simulate a monthly series: target first date is ~4 weeks later.
	srcFirst := utcTime(2035, time.June, 4, 9, 0)
	origStart := utcTime(2035, time.June, 4, 11, 0) // +2h offset
	origEnd := utcTime(2035, time.June, 4, 13, 0)

	tgtFirst := utcTime(2035, time.July, 2, 9, 0) // ~28 days later

	gotStart, gotEnd := adjustTimes(origStart, origEnd, srcFirst, tgtFirst)

	wantStart := "2035-07-02T11:00:00Z"
	wantEnd := "2035-07-02T13:00:00Z"

	if gotStart != wantStart {
		t.Errorf("start: want %s, got %s", wantStart, gotStart)
	}
	if gotEnd != wantEnd {
		t.Errorf("end: want %s, got %s", wantEnd, gotEnd)
	}
}
