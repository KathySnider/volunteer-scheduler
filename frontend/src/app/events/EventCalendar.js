"use client";

/**
 * EventCalendar — month-view calendar for volunteer events.
 *
 * Props:
 *   events       {Array}    — filtered eventViews from the parent page
 *   onEventClick {function} — called with event id when an event is clicked
 *
 * Behaviour:
 *   - Opens on the month containing the earliest event in the current results.
 *   - Resets to the new earliest month whenever the events list changes
 *     (i.e. when the user changes a filter).
 *   - Single event on a day: shows name + venue name (or city) + time range.
 *   - Multiple events on a day: shows just the names.
 *   - Today's date is highlighted.
 */

import { useState, useMemo, useEffect, useRef } from "react";
import styles from "./events.module.css";

const DAY_NAMES  = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
const MONTH_NAMES = [
  "January","February","March","April","May","June",
  "July","August","September","October","November","December",
];

/** "YYYY-MM-DD" from an ISO datetime string, in local time. */
function toDateKey(iso) {
  const d = new Date(iso);
  return [
    d.getFullYear(),
    String(d.getMonth() + 1).padStart(2, "0"),
    String(d.getDate()).padStart(2, "0"),
  ].join("-");
}

/** "9:00 AM" from an ISO datetime string. */
function formatTime(iso) {
  if (!iso) return "";
  return new Date(iso).toLocaleTimeString(undefined, {
    hour: "numeric", minute: "2-digit",
  });
}

export default function EventCalendar({ events, onEventClick }) {
  // ---- Build day → event-occurrences map --------------------------------
  const eventsByDay = useMemo(() => {
    const map = {};
    for (const event of events) {
      for (const ed of event.eventDates ?? []) {
        if (!ed.startDateTime) continue;
        const key = toDateKey(ed.startDateTime);
        if (!map[key]) map[key] = [];
        map[key].push({
          event,
          startDateTime: ed.startDateTime,
          endDateTime:   ed.endDateTime,
        });
      }
    }
    return map;
  }, [events]);

  // ---- Starting month = month of the earliest event ---------------------
  const initialMonth = useMemo(() => {
    const keys = Object.keys(eventsByDay).sort();
    if (keys.length === 0) {
      const now = new Date();
      return { year: now.getFullYear(), month: now.getMonth() };
    }
    const [y, m] = keys[0].split("-").map(Number);
    return { year: y, month: m - 1 }; // month is 0-indexed
  }, [eventsByDay]);

  // Restore from sessionStorage on first mount; fall back to initialMonth.
  const [viewYear, setViewYear] = useState(() => {
    try {
      const saved = JSON.parse(sessionStorage.getItem("evtCalMonth") ?? "null");
      if (saved?.year) return saved.year;
    } catch { /* ignore */ }
    return initialMonth.year;
  });
  const [viewMonth, setViewMonth] = useState(() => {
    try {
      const saved = JSON.parse(sessionStorage.getItem("evtCalMonth") ?? "null");
      if (saved?.month != null) return saved.month;
    } catch { /* ignore */ }
    return initialMonth.month;
  });

  // Save month to sessionStorage whenever it changes.
  useEffect(() => {
    try { sessionStorage.setItem("evtCalMonth", JSON.stringify({ year: viewYear, month: viewMonth })); }
    catch { /* ignore */ }
  }, [viewYear, viewMonth]);

  // Reset to the new first-event month when filters change — but skip the
  // very first render so the sessionStorage restore above isn't clobbered.
  const mountedRef = useRef(false);
  useEffect(() => {
    if (!mountedRef.current) { mountedRef.current = true; return; }
    setViewYear(initialMonth.year);
    setViewMonth(initialMonth.month);
  }, [initialMonth.year, initialMonth.month]);

  // ---- Navigation -------------------------------------------------------
  const goBack = () => {
    if (viewMonth === 0) { setViewMonth(11); setViewYear((y) => y - 1); }
    else setViewMonth((m) => m - 1);
  };
  const goForward = () => {
    if (viewMonth === 11) { setViewMonth(0); setViewYear((y) => y + 1); }
    else setViewMonth((m) => m + 1);
  };

  // ---- Grid layout helpers ----------------------------------------------
  const firstDayOfWeek = new Date(viewYear, viewMonth, 1).getDay(); // 0 = Sun
  const daysInMonth    = new Date(viewYear, viewMonth + 1, 0).getDate();

  const todayKey = toDateKey(new Date().toISOString());

  // ---- Render -----------------------------------------------------------
  return (
    <div className={styles.calendarWrapper}>

      {/* Month navigation */}
      <div className={styles.calHeader}>
        <button className={styles.calNavBtn} onClick={goBack} aria-label="Previous month">‹</button>
        <span className={styles.calMonthLabel}>{MONTH_NAMES[viewMonth]} {viewYear}</span>
        <button className={styles.calNavBtn} onClick={goForward} aria-label="Next month">›</button>
      </div>

      {/* 7-column grid */}
      <div className={styles.calGrid}>

        {/* Day-name header row */}
        {DAY_NAMES.map((d) => (
          <div key={d} className={styles.calDayName}>{d}</div>
        ))}

        {/* Leading blank cells */}
        {Array.from({ length: firstDayOfWeek }).map((_, i) => (
          <div key={`blank-${i}`} className={styles.calCell} />
        ))}

        {/* Day cells */}
        {Array.from({ length: daysInMonth }).map((_, i) => {
          const day = i + 1;
          const key = `${viewYear}-${String(viewMonth + 1).padStart(2, "0")}-${String(day).padStart(2, "0")}`;
          const dayEvents = eventsByDay[key] ?? [];
          const isToday   = key === todayKey;

          return (
            <div
              key={key}
              className={`${styles.calCell} ${isToday ? styles.calCellToday : ""}`}
            >
              <div className={`${styles.calDayNum} ${isToday ? styles.calDayNumToday : ""}`}>
                {day}
              </div>

              {dayEvents.length === 1 ? (
                /* Single event — show name + venue + times */
                <button
                  className={styles.calEvent}
                  onClick={() => onEventClick(dayEvents[0].event.id)}
                >
                  <span className={styles.calEventName}>{dayEvents[0].event.name}</span>
                  <span className={styles.calEventMeta}>
                    {dayEvents[0].event.venue?.name || dayEvents[0].event.venue?.city || ""}
                    {dayEvents[0].startDateTime
                      ? ` · ${formatTime(dayEvents[0].startDateTime)}–${formatTime(dayEvents[0].endDateTime)}`
                      : ""}
                  </span>
                </button>
              ) : (
                /* Multiple events — name only */
                dayEvents.map((de, idx) => (
                  <button
                    key={idx}
                    className={styles.calEvent}
                    onClick={() => onEventClick(de.event.id)}
                  >
                    <span className={styles.calEventName}>{de.event.name}</span>
                  </button>
                ))
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
