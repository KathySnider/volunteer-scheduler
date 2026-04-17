"use client";

import { useEffect, useState, useRef } from "react";
import { useRouter } from "next/navigation";
import {
  getAuthToken,
  getAuthRole,
  getAuthName,
  signOut,
  adminGql,
} from "../../lib/api";
import UserMenu from "../../components/UserMenu";
import styles from "./admin-events.module.css";

/* ----- GraphQL ----- */

const LOOKUP_VALUES = `
  query {
    lookupValues {
      cities
      jobTypes { id name }
    }
  }
`;

const FILTERED_EVENTS = `
  query FilteredEvents($filter: EventFilterInput) {
    filteredEvents(filter: $filter) {
      id
      name
      eventType
      venue { city state }
      eventDates { startDateTime }
      shiftSummaries { assignedVolunteers maxVolunteers }
    }
  }
`;

const DELETE_EVENT = `
  mutation DeleteEvent($eventId: ID!) {
    deleteEvent(eventId: $eventId) {
      success
      message
    }
  }
`;

/* ----- Helpers ----- */

function formatDate(isoString) {
  if (!isoString) return "—";
  return new Date(isoString).toLocaleDateString(undefined, {
    month: "short", day: "numeric", year: "numeric",
  });
}

function earliestDate(eventDates) {
  if (!eventDates?.length) return null;
  return eventDates.reduce((min, d) =>
    d.startDateTime < min.startDateTime ? d : min
  ).startDateTime;
}

function totalVols(shiftSummaries) {
  let assigned = 0, max = 0;
  for (const s of shiftSummaries ?? []) {
    assigned += s.assignedVolunteers;
    max += s.maxVolunteers;
  }
  return { assigned, max };
}

const FORMAT_LABEL = { VIRTUAL: "Virtual", IN_PERSON: "In Person", HYBRID: "Hybrid" };
const FORMAT_BADGE = {
  VIRTUAL:   styles.badgeVirtual,
  IN_PERSON: styles.badgeInPerson,
  HYBRID:    styles.badgeHybrid,
};

/* ----- Multi-select dropdown ----- */

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

/* ----- Page ----- */

export default function AdminEventsPage() {
  const router = useRouter();
  const [token, setToken] = useState(null);
  const [gql, setGql] = useState(null);
  const [userName, setUserName] = useState("");

  // Lookup data
  const [allCities, setAllCities] = useState([]);
  const [jobTypes, setJobTypes] = useState([]);
  const [lookupsReady, setLookupsReady] = useState(false);

  // Filter state
  const [selectedCities, setSelectedCities] = useState([]);
  const [selectedJobs, setSelectedJobs] = useState([]);
  const [selectedFormat, setSelectedFormat] = useState("");
  const [selectedTimeFrame, setSelectedTimeFrame] = useState("ALL");

  // Events
  const [events, setEvents] = useState([]);
  const [loading, setLoading] = useState(false);
  const [pageError, setPageError] = useState("");
  const [actionMsg, setActionMsg] = useState(null);

  /* ----- Auth check ----- */
  useEffect(() => {
    const t = getAuthToken();
    if (!t) { router.replace("/login"); return; }
    const role = getAuthRole();
    if (role !== "ADMINISTRATOR") { router.replace("/events"); return; }

    const bound = (q, v) => adminGql(q, v, t);
    setToken(t);
    setGql(() => bound);
    setUserName(getAuthName() ?? "");
  }, [router]);

  /* ----- Fetch lookups once gql is available ----- */
  useEffect(() => {
    if (!gql) return;

    gql(LOOKUP_VALUES, null)
      .then((res) => {
        const lv = res.data?.lookupValues;
        if (lv) {
          const unique = [...new Set(lv.cities ?? [])].sort();
          setAllCities(unique);
          setJobTypes(lv.jobTypes ?? []);
        }
        setLookupsReady(true);
      })
      .catch(() => {
        setLookupsReady(true);
      });
  }, [gql]);

  /* ----- Auto-search whenever filter state or readiness changes ----- */
  useEffect(() => {
    if (!gql || !lookupsReady) return;

    let cancelled = false;
    setLoading(true);
    setPageError("");

    const filter = {};
    if (selectedCities.length > 0) filter.cities    = selectedCities;
    if (selectedJobs.length > 0)   filter.jobs      = selectedJobs;
    if (selectedFormat)            filter.eventType = selectedFormat;
    filter.timeFrame = selectedTimeFrame || "ALL";

    gql(FILTERED_EVENTS, { filter })
      .then((res) => {
        if (cancelled) return;
        if (res.errors) {
          setPageError(res.errors[0]?.message ?? "Error loading events.");
          setEvents([]);
        } else {
          setEvents(res.data?.filteredEvents ?? []);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setPageError("Unable to reach the server.");
          setEvents([]);
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => { cancelled = true; };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [gql, lookupsReady, selectedCities, selectedJobs, selectedFormat, selectedTimeFrame]);

  /* ----- Filter helpers ----- */

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
    setSelectedTimeFrame("ALL");
    setActionMsg(null);
  }

  /* ----- Delete ----- */
  const handleDelete = async (event) => {
    if (!window.confirm(`Delete "${event.name}"? This cannot be undone.`)) return;
    setActionMsg(null);
    try {
      const res = await gql(DELETE_EVENT, { eventId: event.id });
      const result = res.data?.deleteEvent;
      if (result?.success) {
        setEvents((prev) => prev.filter((e) => e.id !== event.id));
        setActionMsg({ type: "success", text: `"${event.name}" was deleted.` });
      } else {
        setActionMsg({ type: "error", text: result?.message ?? "Failed to delete event." });
      }
    } catch {
      setActionMsg({ type: "error", text: "Unable to reach the server." });
    }
  };

  const handleSignOut = async () => { await signOut(token); router.replace("/login"); };

  if (!token) return null;

  /* ----- Derived labels ----- */
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
    selectedTimeFrame !== "ALL";

  return (
    <div className={styles.page}>
      {/* Top bar */}
      <div className={styles.topBar}>
        <a href="/events" className={styles.backLink}>← Back to Events</a>
        <UserMenu name={userName} isAdmin={true} onSignOut={handleSignOut} />
      </div>

      <div className={styles.content}>
        {/* Header row */}
        <div className={styles.pageHeader}>
          <h1 className={styles.pageTitle}>
            Manage Events
            {!loading && events.length > 0 && (
              <span className={styles.eventCount}>({events.length})</span>
            )}
          </h1>
          <a href="/admin/events/new" className={styles.createBtn}>+ Create New Event</a>
        </div>

        {/* Banners */}
        {actionMsg?.type === "success" && (
          <div className={styles.successBanner}>{actionMsg.text}</div>
        )}
        {(actionMsg?.type === "error" || pageError) && (
          <div className={styles.errorBanner}>{actionMsg?.text ?? pageError}</div>
        )}

        {/* Filter bar */}
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

              {/* Format */}
              <div className={styles.filterGroup}>
                <label className={styles.filterLabel} htmlFor="adminFormatFilter">
                  Format
                </label>
                <select
                  id="adminFormatFilter"
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
                <label className={styles.filterLabel} htmlFor="adminTimeFrameFilter">
                  Timeframe
                </label>
                <select
                  id="adminTimeFrameFilter"
                  className={styles.filterSelect}
                  value={selectedTimeFrame}
                  onChange={(e) => setSelectedTimeFrame(e.target.value)}
                >
                  <option value="ALL">All</option>
                  <option value="UPCOMING">Upcoming</option>
                  <option value="PAST">Past</option>
                </select>
              </div>

              {/* Conditional reset */}
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

        {/* Loading */}
        {loading && (
          <div className={styles.stateBox}>
            <div className={styles.spinner} />
            <p>Loading events…</p>
          </div>
        )}

        {/* Empty */}
        {!loading && events.length === 0 && !pageError && (
          <div className={styles.stateBox}>
            <div className={styles.stateTitle}>No events found</div>
            <p>Try adjusting your filters, or create a new event.</p>
          </div>
        )}

        {/* Table */}
        {!loading && events.length > 0 && (
          <div className={styles.tableWrap}>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th>Event Name</th>
                  <th>Date</th>
                  <th>Location</th>
                  <th>Format</th>
                  <th className={styles.right}>Volunteers</th>
                  <th className={styles.right}>Actions</th>
                </tr>
              </thead>
              <tbody>
                {events.map((event) => {
                  const { assigned, max } = totalVols(event.shiftSummaries);
                  const isFull = assigned >= max && max > 0;
                  const location = event.eventType === "VIRTUAL"
                    ? "Virtual"
                    : event.venue
                    ? `${event.venue.city}, ${event.venue.state}`
                    : "TBD";

                  return (
                    <tr key={event.id}>
                      <td>
                        <div className={styles.eventName}>{event.name}</div>
                      </td>
                      <td>{formatDate(earliestDate(event.eventDates))}</td>
                      <td>{location}</td>
                      <td>
                        <span className={`${styles.badge} ${FORMAT_BADGE[event.eventType] ?? ""}`}>
                          {FORMAT_LABEL[event.eventType] ?? event.eventType}
                        </span>
                      </td>
                      <td className={styles.right}>
                        {max > 0 ? (
                          <span className={isFull ? styles.countFull : styles.countOpen}>
                            {assigned}/{max}
                          </span>
                        ) : (
                          <span className={styles.textMuted}>—</span>
                        )}
                      </td>
                      <td className={styles.right}>
                        <div className={styles.actions}>
                          <button
                            className={`${styles.iconBtn} ${styles.iconBtnEdit}`}
                            title="Edit event"
                            onClick={() => router.push(`/admin/events/${event.id}`)}
                          >
                            ✏
                          </button>
                          <button
                            className={`${styles.iconBtn} ${styles.iconBtnDelete}`}
                            title="Delete event"
                            onClick={() => handleDelete(event)}
                          >
                            🗑
                          </button>
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
