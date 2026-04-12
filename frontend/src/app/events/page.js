"use client";

import { useEffect, useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import {
  getAuthToken,
  getAuthRole,
  getAuthName,
  clearAuthToken,
  volunteerGql,
  adminGql,
} from "../lib/api";
import UserMenu from "../components/UserMenu";
import styles from "./events.module.css";

/* ----- GraphQL operations ----- */

const LOOKUP_VALUES = `
  query {
    lookupValues {
      regions {
        id
        name
      }
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

/**
 * Format a date string ("2026-04-15") + time into the "YYYY-MM-DD HH:MM:SS"
 * format that the backend's DateTimeToUTC expects.
 * Returns null when dateStr is empty.
 */
function formatDateForBackend(dateStr, time) {
  if (!dateStr) return null;
  return `${dateStr} ${time}:00`;
}

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
  const [gql, setGql] = useState(null); // volunteerGql or adminGql bound to token
  const [userName, setUserName] = useState("");
  const [isAdmin, setIsAdmin] = useState(false);
  const [ianaZone, setIanaZone] = useState("UTC");

  // Lookup data
  const [regions, setRegions] = useState([]);
  const [jobTypes, setJobTypes] = useState([]);

  // Filter state
  const [selectedRegion, setSelectedRegion] = useState("");
  const [selectedJobType, setSelectedJobType] = useState("");
  const [selectedFormat, setSelectedFormat] = useState("");
  const [startDate, setStartDate] = useState(
    () => new Date().toISOString().slice(0, 10)   // default: today
  );
  const [endDate, setEndDate] = useState("");

  // Results state
  const [events, setEvents] = useState([]);
  const [loading, setLoading] = useState(false);
  const [searchError, setSearchError] = useState("");
  const [hasSearched, setHasSearched] = useState(false);

  /* Auth check and initialisation */
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
    setIanaZone(Intl.DateTimeFormat().resolvedOptions().timeZone);

    // Fetch lookup values, then auto-search with today as the start date.
    const tz = Intl.DateTimeFormat().resolvedOptions().timeZone;
    const todayStr = new Date().toISOString().slice(0, 10);
    boundGql(LOOKUP_VALUES, null)
      .then((res) => {
        const lv = res.data?.lookupValues;
        if (lv) {
          setRegions(lv.regions);
          setJobTypes(lv.jobTypes);
        }
      })
      .catch(() => {
        // Non-fatal; filters will just be empty.
      })
      .finally(() => {
        // Auto-search with default filters (today onwards, nothing else selected).
        doSearch(boundGql, "", "", "", todayStr, "", tz);
      });
  }, [router]);

  /**
   * Core search executor. Takes the gql function and all filter values as
   * explicit arguments so it can be called safely from the initial load
   * (before state settles) and from handleReset (after state is reset).
   * State setters are stable React references so no deps are needed.
   */
  const doSearch = useCallback(async (boundGql, region, jobType, format, start, end, zone) => {
    setLoading(true);
    setSearchError("");
    setHasSearched(true);

    const filter = {};
    if (region)  filter.regions  = [parseInt(region, 10)];
    if (jobType) filter.jobs     = [parseInt(jobType, 10)];
    if (format)  filter.eventType = format;
    if (start)   filter.shiftStartDateTime = formatDateForBackend(start, "00:00");
    if (end)     filter.shiftEndDateTime   = formatDateForBackend(end,   "23:59");
    filter.ianaZone = zone;

    try {
      const res = await boundGql(FILTERED_EVENTS, { filter });
      if (res.errors) {
        setSearchError(res.errors[0]?.message ?? "Error loading events.");
        setEvents([]);
      } else {
        setEvents(res.data?.filteredEvents ?? []);
      }
    } catch {
      setSearchError("Unable to reach the server. Please try again.");
      setEvents([]);
    } finally {
      setLoading(false);
    }
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  /* Search button — passes current state values explicitly */
  const handleSearch = useCallback(() => {
    if (!gql) return;
    doSearch(gql, selectedRegion, selectedJobType, selectedFormat, startDate, endDate, ianaZone);
  }, [gql, doSearch, selectedRegion, selectedJobType, selectedFormat, startDate, endDate, ianaZone]);

  /* Reset — restores defaults and re-runs search immediately */
  const TODAY = new Date().toISOString().slice(0, 10);
  const handleReset = useCallback(() => {
    setSelectedRegion("");
    setSelectedJobType("");
    setSelectedFormat("");
    setStartDate(TODAY);
    setEndDate("");
    setSearchError("");
    if (gql) doSearch(gql, "", "", "", TODAY, "", ianaZone);
  }, [gql, doSearch, ianaZone, TODAY]);

  const handleSignOut = () => {
    clearAuthToken();
    router.replace("/login");
  };

  const handleView = (eventId) => {
    router.push(`/events/${eventId}`);
  };

  if (!token) return null;

  return (
    <div className={styles.page}>
      {/* ---- Top bar ---- */}
      <div className={styles.topBar}>
        <div className={styles.appTitle}>AARP Volunteer Events</div>
        <div className={styles.topBarRight}>
          <a href="/my-shifts" className={styles.myShiftsLink}>My Shifts</a>
          <UserMenu name={userName} isAdmin={isAdmin} onSignOut={handleSignOut} />
        </div>
      </div>

      <div className={styles.body}>
      {/* ---- Left sidebar ---- */}
      <aside className={styles.sidebar}>
        <div className={styles.sidebarTitle}>Find Events</div>

        <div className={styles.filterGroup}>
          <label className={styles.filterLabel} htmlFor="regionFilter">
            Region
          </label>
          <select
            id="regionFilter"
            className={styles.filterSelect}
            value={selectedRegion}
            onChange={(e) => setSelectedRegion(e.target.value)}
          >
            <option value="">All regions</option>
            {regions.map((r) => (
              <option key={r.id} value={r.id}>
                {r.name}
              </option>
            ))}
          </select>
        </div>

        <div className={styles.filterGroup}>
          <label className={styles.filterLabel} htmlFor="jobTypeFilter">
            Job Type
          </label>
          <select
            id="jobTypeFilter"
            className={styles.filterSelect}
            value={selectedJobType}
            onChange={(e) => setSelectedJobType(e.target.value)}
          >
            <option value="">All job types</option>
            {jobTypes.map((j) => (
              <option key={j.id} value={j.id}>
                {j.name}
              </option>
            ))}
          </select>
        </div>

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

        <div className={styles.filterGroup}>
          <label className={styles.filterLabel} htmlFor="startDate">
            From Date
          </label>
          <input
            id="startDate"
            type="date"
            className={styles.filterDate}
            value={startDate}
            onChange={(e) => setStartDate(e.target.value)}
          />
        </div>

        <div className={styles.filterGroup}>
          <label className={styles.filterLabel} htmlFor="endDate">
            To Date
          </label>
          <input
            id="endDate"
            type="date"
            className={styles.filterDate}
            value={endDate}
            onChange={(e) => setEndDate(e.target.value)}
          />
        </div>

        <button
          className={styles.searchButton}
          onClick={handleSearch}
          disabled={loading}
        >
          {loading ? "Searching\u2026" : "Search"}
        </button>

        <button className={styles.resetButton} onClick={handleReset}>
          Reset
        </button>
      </aside>

      {/* ---- Main content ---- */}
      <main className={styles.main}>
        <div className={styles.mainHeader}>
          <h1 className={styles.mainTitle}>Volunteer Events</h1>
          <p className={styles.mainSubtitle}>
            Use the filters to find events that need volunteers.
          </p>
        </div>

        {searchError && <div className={styles.errorBox}>{searchError}</div>}

        {loading && (
          <div className={styles.stateBox}>
            <div className={styles.spinner} />
            <p>Loading events&hellip;</p>
          </div>
        )}

        {!loading && hasSearched && events.length === 0 && !searchError && (
          <div className={styles.stateBox}>
            <div className={styles.stateTitle}>No events found</div>
            <p>Try adjusting your filters.</p>
          </div>
        )}

        {!loading && !hasSearched && (
          <div className={styles.stateBox}>
            <div className={styles.stateTitle}>Ready to search</div>
            <p>Select filters and click Search to see available events.</p>
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
      </div>{/* end .body */}
    </div>
  );
}
