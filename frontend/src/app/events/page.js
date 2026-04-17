"use client";

import { useEffect, useState, useRef } from "react";
import { useRouter } from "next/navigation";
import {
  getAuthToken,
  getAuthRole,
  getAuthName,
  signOut,
  volunteerGql,
  adminGql,
} from "../lib/api";
import UserMenu from "../components/UserMenu";
import FeedbackButton from "../components/FeedbackButton";
import styles from "./events.module.css";

/* ----- Constants ----- */

const CITY_STORAGE_KEY = "evtCityFilter";

/* ----- GraphQL operations ----- */

const LOOKUP_VALUES = `
  query {
    lookupValues {
      cities
      jobTypes {
        id
        name
      }
    }
  }
`;

const FILTERED_EVENTS = `
  query FilteredEvents($filter: EventFilterInput) {
    filteredEvents(filter: $filter) {
      id
      name
      description
      eventType
      venue {
        city
        state
      }
      eventDates {
        startDateTime
        endDateTime
      }
      shiftSummaries {
        jobName
        assignedVolunteers
        maxVolunteers
      }
    }
  }
`;

/* ----- Helpers ----- */

/** Format a UTC ISO date string for display in the user's local timezone. */
function formatLocalDateTime(isoString) {
  if (!isoString) return "";
  const date = new Date(isoString);
  return date.toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

/** Return the earliest startDateTime from an eventDates array. */
function earliestDate(eventDates) {
  if (!eventDates || eventDates.length === 0) return null;
  return eventDates.reduce((min, d) =>
    d.startDateTime < min.startDateTime ? d : min
  ).startDateTime;
}

/** Sum assigned and max across all shiftSummaries. */
function totalVolunteers(shiftSummaries) {
  let assigned = 0;
  let max = 0;
  for (const s of shiftSummaries) {
    assigned += s.assignedVolunteers;
    max += s.maxVolunteers;
  }
  return { assigned, max };
}

/* ----- Format badge ----- */

const FORMAT_LABELS = {
  VIRTUAL: "Virtual",
  IN_PERSON: "In Person",
  HYBRID: "Hybrid",
};

const FORMAT_BADGE_CLASS = {
  VIRTUAL: styles.badgeVirtual,
  IN_PERSON: styles.badgeInPerson,
  HYBRID: styles.badgeHybrid,
};

function FormatBadge({ eventType }) {
  return (
    <span className={`${styles.badge} ${FORMAT_BADGE_CLASS[eventType] ?? ""}`}>
      {FORMAT_LABELS[eventType] ?? eventType}
    </span>
  );
}

/* ----- Multi-select dropdown ----- */
// items: array of { value, label }
// selected: array of values (same type as item.value)
// onToggle(value): toggle one item
// onSelectAll / onClearAll: optional, shown as action buttons above the list

function MultiSelectDropdown({
  buttonLabel,
  items,
  selected,
  onToggle,
  onSelectAll,
  onClearAll,
}) {
  const [open, setOpen] = useState(false);
  const containerRef = useRef(null);

  // Close when clicking outside the dropdown
  useEffect(() => {
    if (!open) return;
    function handler(e) {
      if (containerRef.current && !containerRef.current.contains(e.target)) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [open]);

  return (
    <div className={styles.multiSelect} ref={containerRef}>
      <button
        type="button"
        className={`${styles.multiSelectBtn} ${open ? styles.multiSelectBtnOpen : ""}`}
        onClick={() => setOpen((o) => !o)}
        aria-expanded={open}
        aria-haspopup="listbox"
      >
        {buttonLabel}
        <span className={styles.chevron} aria-hidden="true">▾</span>
      </button>

      {open && (
        <div className={styles.checkboxPanel} role="listbox">
          {(onSelectAll || onClearAll) && (
            <div className={styles.panelActions}>
              {onSelectAll && (
                <button type="button" className={styles.panelActionBtn} onClick={onSelectAll}>
                  Select All
                </button>
              )}
              {onClearAll && (
                <button type="button" className={styles.panelActionBtn} onClick={onClearAll}>
                  Clear All
                </button>
              )}
            </div>
          )}
          <div className={styles.checkboxList}>
            {items.map((item) => {
              const checked = selected.includes(item.value);
              return (
                <label key={item.value} className={styles.checkboxItem}>
                  <input
                    type="checkbox"
                    checked={checked}
                    onChange={() => onToggle(item.value)}
                  />
                  {item.label}
                </label>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}

/* ----- Event card ----- */

function EventCard({ event, onView }) {
  const dateStr = formatLocalDateTime(earliestDate(event.eventDates));
  const location =
    event.eventType === "VIRTUAL"
      ? "Virtual"
      : event.venue
      ? `${event.venue.city}, ${event.venue.state}`
      : "Location TBD";

  const { assigned, max } = totalVolunteers(event.shiftSummaries);
  const isFull = assigned >= max && max > 0;

  return (
    <div className={styles.card}>
      <div className={styles.cardBody}>
        <div className={styles.cardName}>{event.name}</div>

        <div className={styles.cardMeta}>
          {dateStr && (
            <span className={styles.cardMetaItem}>
              <span>&#128197;</span> {dateStr}
            </span>
          )}
          <span className={styles.cardMetaItem}>
            <span>&#128205;</span> {location}
          </span>
          <FormatBadge eventType={event.eventType} />
        </div>

        {event.description && (
          <p className={styles.cardDescription}>{event.description}</p>
        )}

        {event.shiftSummaries.length > 0 && (
          <div className={styles.rolesRow}>
            <span className={styles.rolesLabel}>Roles needed:</span>
            {event.shiftSummaries.map((s) => (
              <span key={s.jobName} className={styles.roleChip}>
                {s.jobName} ({s.assignedVolunteers}/{s.maxVolunteers})
              </span>
            ))}
          </div>
        )}
      </div>

      <div className={styles.cardSide}>
        <div className={styles.volunteerCount}>
          <div className={styles.volunteerCountLabel}>Volunteers</div>
          <div
            className={`${styles.volunteerCountValue} ${
              isFull ? styles.countFull : styles.countOpen
            }`}
          >
            {assigned}/{max}
          </div>
          <div
            className={`${styles.volunteerCountStatus} ${
              isFull ? styles.statusFull : styles.statusOpen
            }`}
          >
            {isFull ? "Fully staffed" : "Spots open"}
          </div>
        </div>

        <button className={styles.viewButton} onClick={() => onView(event.id)}>
          View Details
        </button>
      </div>
    </div>
  );
}

/* ----- Page ----- */

export default function EventsPage() {
  const router = useRouter();
  const [token, setToken] = useState(null);
  const [gql, setGql] = useState(null);
  const [userName, setUserName] = useState("");
  const [isAdmin, setIsAdmin] = useState(false);

  // Lookup data
  const [allCities, setAllCities] = useState([]);
  const [jobTypes, setJobTypes] = useState([]);
  const [lookupsReady, setLookupsReady] = useState(false);

  // Filter state
  // Empty array = no filter (show all); non-empty = filter to those values.
  const [selectedCities, setSelectedCities] = useState([]);
  const [selectedJobs, setSelectedJobs] = useState([]);
  const [selectedFormat, setSelectedFormat] = useState("");
  const [selectedTimeFrame, setSelectedTimeFrame] = useState("UPCOMING");

  // Results
  const [events, setEvents] = useState([]);
  const [loading, setLoading] = useState(false);
  const [searchError, setSearchError] = useState("");

  /* ----- Auth check ----- */
  useEffect(() => {
    const t = getAuthToken();
    if (!t) {
      router.replace("/login");
      return;
    }
    const role = getAuthRole();
    const boundGql =
      role === "ADMINISTRATOR"
        ? (q, v) => adminGql(q, v, t)
        : (q, v) => volunteerGql(q, v, t);

    setToken(t);
    setGql(() => boundGql);
    setUserName(getAuthName() ?? "");
    setIsAdmin(role === "ADMINISTRATOR");
  }, [router]);

  /* ----- Fetch lookups once gql is available ----- */
  useEffect(() => {
    if (!gql) return;

    gql(LOOKUP_VALUES, null)
      .then((res) => {
        const lv = res.data?.lookupValues;
        if (lv) {
          // Deduplicate and sort cities alphabetically.
          const unique = [...new Set(lv.cities ?? [])].sort();
          setAllCities(unique);
          setJobTypes(lv.jobTypes ?? []);

          // Restore saved city selection from sessionStorage.
          try {
            const saved = sessionStorage.getItem(CITY_STORAGE_KEY);
            if (saved) {
              const parsed = JSON.parse(saved);
              // Only keep cities that are still in the current list.
              const valid = parsed.filter((c) => unique.includes(c));
              if (valid.length > 0) setSelectedCities(valid);
            }
          } catch {
            // sessionStorage unavailable — ignore.
          }
        }
        // Mark ready inside .then() so all state updates batch together.
        setLookupsReady(true);
      })
      .catch(() => {
        // Non-fatal: filters will be empty but search can still run.
        setLookupsReady(true);
      });
  }, [gql]);

  /* ----- Persist city selection to sessionStorage ----- */
  useEffect(() => {
    if (!lookupsReady) return;
    try {
      sessionStorage.setItem(CITY_STORAGE_KEY, JSON.stringify(selectedCities));
    } catch {
      // ignore
    }
  }, [selectedCities, lookupsReady]);

  /* ----- Auto-search whenever filter state or readiness changes ----- */
  useEffect(() => {
    if (!gql || !lookupsReady) return;

    let cancelled = false;
    setLoading(true);
    setSearchError("");

    const filter = {};
    if (selectedCities.length > 0) filter.cities = selectedCities;
    if (selectedJobs.length > 0)   filter.jobs = selectedJobs;
    if (selectedFormat)            filter.eventType = selectedFormat;
    filter.timeFrame = selectedTimeFrame || "ALL";

    gql(FILTERED_EVENTS, { filter })
      .then((res) => {
        if (cancelled) return;
        if (res.errors) {
          setSearchError(res.errors[0]?.message ?? "Error loading events.");
          setEvents([]);
        } else {
          setEvents(res.data?.filteredEvents ?? []);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setSearchError("Unable to reach the server. Please try again.");
          setEvents([]);
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [gql, lookupsReady, selectedCities, selectedJobs, selectedFormat, selectedTimeFrame]);

  /* ----- Filter callbacks ----- */

  function toggleCity(city) {
    setSelectedCities((prev) =>
      prev.includes(city) ? prev.filter((c) => c !== city) : [...prev, city]
    );
  }

  function toggleJob(jobId) {
    setSelectedJobs((prev) =>
      prev.includes(jobId) ? prev.filter((j) => j !== jobId) : [...prev, jobId]
    );
  }

  function handleReset() {
    setSelectedCities([]);
    setSelectedJobs([]);
    setSelectedFormat("");
    setSelectedTimeFrame("UPCOMING");
    // sessionStorage will be cleared by the persist effect above.
  }

  const handleSignOut = async () => { await signOut(token); router.replace("/login"); };

  const handleView = (eventId) => {
    router.push(`/events/${eventId}`);
  };

  if (!token) return null;

  /* ----- Derived labels for multi-select buttons ----- */
  const cityBtnLabel =
    selectedCities.length === 0
      ? "All Cities"
      : selectedCities.length === 1
      ? selectedCities[0]
      : `Cities: ${selectedCities.length}`;

  const jobBtnLabel =
    selectedJobs.length === 0
      ? "All Jobs"
      : selectedJobs.length === 1
      ? (jobTypes.find((j) => j.id === selectedJobs[0])?.name ?? "1 job")
      : `Jobs: ${selectedJobs.length}`;

  const cityItems = allCities.map((c) => ({ value: c, label: c }));
  const jobItems  = jobTypes.map((j)  => ({ value: j.id, label: j.name }));

  const hasActiveFilters =
    selectedCities.length > 0 ||
    selectedJobs.length > 0 ||
    selectedFormat !== "" ||
    selectedTimeFrame !== "UPCOMING";

  return (
    <div className={styles.page}>

      {/* ---- Top bar ---- */}
      <div className={styles.topBar}>
        <div className={styles.appTitle}>AARP Volunteer Events</div>
        <div className={styles.topBarRight}>
          <a href="/my-shifts" className={styles.topBarLink}>My Shifts</a>
          <a href="/my-feedback" className={styles.topBarLink}>My Feedback</a>
          <UserMenu name={userName} isAdmin={isAdmin} onSignOut={handleSignOut} />
        </div>
      </div>

      {/* ---- Filter bar ---- */}
      <div className={styles.filterBar}>
        <div className={styles.filterCard}>
        <div className={styles.filterBarInner}>

          {/* Cities multi-select */}
          <div className={styles.filterGroup}>
            <span className={styles.filterLabel}>City</span>
            <MultiSelectDropdown
              buttonLabel={cityBtnLabel}
              items={cityItems}
              selected={selectedCities}
              onToggle={toggleCity}
              onSelectAll={() => setSelectedCities([...allCities])}
              onClearAll={() => setSelectedCities([])}
            />
          </div>

          {/* Jobs multi-select */}
          <div className={styles.filterGroup}>
            <span className={styles.filterLabel}>Job</span>
            <MultiSelectDropdown
              buttonLabel={jobBtnLabel}
              items={jobItems}
              selected={selectedJobs}
              onToggle={toggleJob}
            />
          </div>

          {/* Event type */}
          <div className={styles.filterGroup}>
            <label className={styles.filterLabel} htmlFor="formatFilter">
              Format
            </label>
            <select
              id="formatFilter"
              className={styles.filterSelect}
              value={selectedFormat}
              onChange={(e) => setSelectedFormat(e.target.value)}
            >
              <option value="">All formats</option>
              <option value="VIRTUAL">Virtual</option>
              <option value="IN_PERSON">In Person</option>
              <option value="HYBRID">Hybrid</option>
            </select>
          </div>

          {/* Timeframe */}
          <div className={styles.filterGroup}>
            <label className={styles.filterLabel} htmlFor="timeFrameFilter">
              Timeframe
            </label>
            <select
              id="timeFrameFilter"
              className={styles.filterSelect}
              value={selectedTimeFrame}
              onChange={(e) => setSelectedTimeFrame(e.target.value)}
            >
              <option value="UPCOMING">Upcoming</option>
              <option value="PAST">Past</option>
              <option value="ALL">All</option>
            </select>
          </div>

          {/* Reset */}
          {hasActiveFilters && (
            <button
              type="button"
              className={styles.resetButton}
              onClick={handleReset}
            >
              Reset filters
            </button>
          )}
        </div>
        </div>
      </div>

      {/* ---- Main content ---- */}
      <main className={styles.main}>
        <div className={styles.mainHeader}>
          <h1 className={styles.mainTitle}>
            Volunteer Events
            {!loading && events.length > 0 && (
              <span className={styles.eventCount}>({events.length})</span>
            )}
          </h1>
        </div>

        {searchError && <div className={styles.errorBox}>{searchError}</div>}

        {loading && (
          <div className={styles.stateBox}>
            <div className={styles.spinner} />
            <p>Loading events&hellip;</p>
          </div>
        )}

        {!loading && !searchError && events.length === 0 && (
          <div className={styles.stateBox}>
            <div className={styles.stateTitle}>No events found</div>
            <p>Try adjusting your filters.</p>
          </div>
        )}

        {!loading && events.length > 0 && (
          <div className={styles.cardList}>
            {events.map((event) => (
              <EventCard key={event.id} event={event} onView={handleView} />
            ))}
          </div>
        )}
      </main>

      <FeedbackButton />
    </div>
  );
}
