"use client";

import { useEffect, useState, useRef, useCallback } from "react";
import dynamic from "next/dynamic";

// Leaflet requires a browser environment — load it only on the client.
const EventMap = dynamic(() => import("./EventMap"), {
  ssr: false,
  loading: () => <div style={{ height: 500, display: "flex", alignItems: "center", justifyContent: "center", color: "var(--color-text-muted)" }}>Loading map…</div>,
});

const EventCalendar = dynamic(() => import("./EventCalendar"), {
  ssr: false,
  loading: () => <div style={{ padding: "2rem", color: "var(--color-text-muted)" }}>Loading calendar…</div>,
});
import { useRouter } from "next/navigation";
import {
  isAuthenticated,
  hasAuthRole,
  Roles,
  getAuthName,
  signOut,
  volunteerGql,
} from "../lib/api";
import AdminTopBar from "../components/AdminTopBar";
import FeedbackButton from "../components/FeedbackButton";
import styles from "./events.module.css";

/* ----- Constants ----- */

const CITY_STORAGE_KEY     = "evtCityFilter";
const DISTANCE_STORAGE_KEY = "evtDistanceFilter";
const DISTANCE_OPTIONS = [10, 25, 50, 100, 200];

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

const GET_VOLUNTEER_PROFILE = `
  query {
    ownProfile {
      zipCode
      distance
    }
  }
`;

const FILTERED_EVENTS = `
  query FilteredEvents($filter: VolunteerEventFilterInput) {
    eventViews(filter: $filter) {
      id
      name
      description
      eventType
      venue {
        name
        city
        state
        latitude
        longitude
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
  const [ready, setReady] = useState(false);
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

  // Distance mode — active when the volunteer has a zip code on their profile.
  const [hasZip, setHasZip] = useState(false);
  const [profileZip, setProfileZip] = useState("");
  const [selectedDistance, setSelectedDistance] = useState("");

  // Results
  const [events, setEvents] = useState([]);
  const [loading, setLoading] = useState(false);
  const [searchError, setSearchError] = useState("");
  const [feedbackOpen, setFeedbackOpen] = useState(false);
  const [viewMode, setViewMode] = useState(() => {
    try { return sessionStorage.getItem("evtViewMode") || "list"; } catch { return "list"; }
  });

  /* ----- Auth check ----- */
  useEffect(() => {
    if (!isAuthenticated()) {
      router.replace("/login");
      return;
    }
    setReady(true);
    setGql(() => volunteerGql);
    setUserName(getAuthName() ?? "");
    setIsAdmin(hasAuthRole(Roles.ADMINISTRATOR));
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

  /* ----- Fetch volunteer profile for distance mode (all roles) ----- */
  useEffect(() => {
    if (!gql) return;
    volunteerGql(GET_VOLUNTEER_PROFILE, null)
      .then((res) => {
        const p = res.data?.ownProfile;
        if (p?.zipCode) {
          setHasZip(true);
          setProfileZip(p.zipCode);
          // Restore from sessionStorage first; fall back to profile default.
          try {
            const saved = sessionStorage.getItem(DISTANCE_STORAGE_KEY);
            if (saved !== null) {
              setSelectedDistance(saved);
              return;
            }
          } catch { /* sessionStorage unavailable */ }
          if (p.distance != null) {
            setSelectedDistance(String(p.distance));
          }
        }
      })
      .catch(() => {
        // Non-fatal: distance mode simply won't activate.
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

  /* ----- Persist distance selection to sessionStorage ----- */
  useEffect(() => {
    if (!hasZip) return;
    try {
      if (selectedDistance !== "") {
        sessionStorage.setItem(DISTANCE_STORAGE_KEY, selectedDistance);
      } else {
        // Remove the key so the next load falls back to the profile default.
        sessionStorage.removeItem(DISTANCE_STORAGE_KEY);
      }
    } catch {
      // ignore
    }
  }, [selectedDistance, hasZip]);

  /* ----- Persist view mode to sessionStorage ----- */
  useEffect(() => {
    try { sessionStorage.setItem("evtViewMode", viewMode); } catch { /* ignore */ }
  }, [viewMode]);

  /* ----- Auto-search whenever filter state or readiness changes ----- */
  useEffect(() => {
    if (!gql || !lookupsReady) return;

    let cancelled = false;
    setLoading(true);
    setSearchError("");

    const filter = {};
    if (hasZip && selectedDistance) {
      filter.distance = parseInt(selectedDistance, 10);
    } else if (!hasZip && selectedCities.length > 0) {
      filter.cities = selectedCities;
    }
    if (selectedJobs.length > 0) filter.jobs = selectedJobs;
    if (selectedFormat)          filter.eventType = selectedFormat;
    filter.timeFrame = selectedTimeFrame || "ALL";

    gql(FILTERED_EVENTS, { filter })
      .then((res) => {
        if (cancelled) return;
        if (res.errors) {
          setSearchError(res.errors[0]?.message ?? "Error loading events.");
          setEvents([]);
        } else {
          setEvents(res.data?.eventViews ?? []);
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
  }, [gql, lookupsReady, hasZip, selectedDistance, selectedCities, selectedJobs, selectedFormat, selectedTimeFrame]);

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
    if (hasZip) {
      setSelectedDistance("");
      // sessionStorage will be cleared by the persist effect above.
    } else {
      setSelectedCities([]);
      // sessionStorage will be cleared by the persist effect above.
    }
    setSelectedJobs([]);
    setSelectedFormat("");
    setSelectedTimeFrame("UPCOMING");
  }

  const handleSignOut = async () => { await signOut(); router.replace("/login"); };

  const handleView = (eventId) => {
    router.push(`/events/${eventId}`);
  };

  if (!ready) return null;

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
    (hasZip ? selectedDistance !== "" : selectedCities.length > 0) ||
    selectedJobs.length > 0 ||
    selectedFormat !== "" ||
    selectedTimeFrame !== "UPCOMING";

  return (
    <div className={styles.page}>

      <AdminTopBar userName={userName} isAdmin={isAdmin} onSignOut={handleSignOut} onFeedbackOpen={() => setFeedbackOpen(true)} />

      <h1 className={styles.pageTitle}>
        Volunteer Events
        {!loading && <span className={styles.eventCount}>({events.length})</span>}
      </h1>

      {/* ---- Filter bar ---- */}
      <div className={styles.filterBar}>
        <div className={styles.filterCard}>
        <div className={styles.filterBarInner}>

          {/* Distance select (when volunteer has zip) or city multi-select */}
          {hasZip ? (
            <div className={styles.filterGroup}>
              <label className={styles.filterLabel} htmlFor="distanceFilter">Within</label>
              <span className={styles.filterHint}>miles from zip {profileZip}</span>
              <select
                id="distanceFilter"
                className={styles.filterSelect}
                value={selectedDistance}
                onChange={(e) => setSelectedDistance(e.target.value)}
              >
                <option value="">Any distance</option>
                {DISTANCE_OPTIONS.map((mi) => (
                  <option key={mi} value={String(mi)}>{mi} miles</option>
                ))}
              </select>
            </div>
          ) : (
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
          )}

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

      {/* ---- List / Map / Calendar toggle ---- */}
      {!loading && events.length > 0 && (
        <div className={styles.viewToggle}>
          <button
            className={`${styles.viewToggleBtn} ${viewMode === "list" ? styles.viewToggleBtnActive : ""}`}
            onClick={() => setViewMode("list")}
          >
            ☰ List
          </button>
          <button
            className={`${styles.viewToggleBtn} ${viewMode === "map" ? styles.viewToggleBtnActive : ""}`}
            onClick={() => setViewMode("map")}
          >
            🗺 Map
          </button>
          <button
            className={`${styles.viewToggleBtn} ${viewMode === "calendar" ? styles.viewToggleBtnActive : ""}`}
            onClick={() => setViewMode("calendar")}
          >
            📅 Calendar
          </button>
        </div>
      )}

      {/* ---- Main content ---- */}
      <main className={styles.main}>
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

        {!loading && events.length > 0 && viewMode === "list" && (
          <div className={styles.cardList}>
            {events.map((event) => (
              <EventCard key={event.id} event={event} onView={handleView} />
            ))}
          </div>
        )}

        {!loading && events.length > 0 && viewMode === "map" && (
          <EventMap events={events} onEventClick={handleView} />
        )}

        {!loading && events.length > 0 && viewMode === "calendar" && (
          <EventCalendar events={events} onEventClick={handleView} />
        )}
      </main>

      <FeedbackButton
        open={feedbackOpen}
        onClose={() => setFeedbackOpen(false)}
      />
    </div>
  );
}
