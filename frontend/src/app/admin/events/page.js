"use client";

import { useEffect, useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import {
  getAuthToken,
  getAuthRole,
  getAuthName,
  clearAuthToken,
  adminGql,
} from "../../lib/api";
import UserMenu from "../../components/UserMenu";
import styles from "./admin-events.module.css";

/* ----- GraphQL ----- */

const ADMIN_EVENTS_QUERY = `
  query AdminEvents($filter: EventFilterInput) {
    filteredEvents(filter: $filter) {
      id
      name
      eventType
      venue { city state }
      eventDates { startDateTime }
      shiftSummaries { assignedVolunteers maxVolunteers }
    }
    lookupValues {
      regions { id name }
      jobTypes { id name }
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

/* ----- Page ----- */

export default function AdminEventsPage() {
  const router = useRouter();
  const [gql, setGql] = useState(null);
  const [userName, setUserName] = useState("");

  // Lookup data for filters
  const [regions, setRegions] = useState([]);
  const [jobTypes, setJobTypes] = useState([]);

  // Filter state
  const [selectedRegion, setSelectedRegion] = useState("");
  const [selectedJobType, setSelectedJobType] = useState("");
  const [selectedFormat, setSelectedFormat] = useState("");

  // Events
  const [events, setEvents] = useState([]);
  const [loading, setLoading] = useState(true);
  const [pageError, setPageError] = useState("");
  const [actionMsg, setActionMsg] = useState(null);

  /* Auth check + initial load (no filter → all events) */
  useEffect(() => {
    const t = getAuthToken();
    if (!t) { router.replace("/login"); return; }
    const role = getAuthRole();
    if (role !== "ADMINISTRATOR") { router.replace("/events"); return; }

    const bound = (q, v) => adminGql(q, v, t);
    setGql(() => bound);
    setUserName(getAuthName() ?? "");

    // Pass null filter to get ALL events (including those without shifts)
    bound(ADMIN_EVENTS_QUERY, { filter: null })
      .then((res) => {
        setEvents(res.data?.filteredEvents ?? []);
        setRegions(res.data?.lookupValues?.regions ?? []);
        setJobTypes(res.data?.lookupValues?.jobTypes ?? []);
        if (res.errors) setPageError(res.errors[0]?.message ?? "Error loading data.");
      })
      .catch(() => setPageError("Unable to reach the server."))
      .finally(() => setLoading(false));
  }, [router]);

  /* Filtered search */
  const handleSearch = useCallback(() => {
    if (!gql) return;
    setLoading(true);
    setPageError("");
    setActionMsg(null);

    const filter = {};
    if (selectedRegion)  filter.regions   = [parseInt(selectedRegion, 10)];
    if (selectedJobType) filter.jobs      = [parseInt(selectedJobType, 10)];
    if (selectedFormat)  filter.eventType = selectedFormat;

    // If no filter values selected, pass null to get all events
    const filterArg = Object.keys(filter).length ? filter : null;

    gql(ADMIN_EVENTS_QUERY, { filter: filterArg })
      .then((res) => {
        setEvents(res.data?.filteredEvents ?? []);
        if (res.errors) setPageError(res.errors[0]?.message ?? "Error loading data.");
      })
      .catch(() => setPageError("Unable to reach the server."))
      .finally(() => setLoading(false));
  }, [gql, selectedRegion, selectedJobType, selectedFormat]);

  /* Reset filters + reload all events */
  const handleReset = useCallback(() => {
    setSelectedRegion("");
    setSelectedJobType("");
    setSelectedFormat("");
    setActionMsg(null);
    if (!gql) return;
    setLoading(true);
    gql(ADMIN_EVENTS_QUERY, { filter: null })
      .then((res) => setEvents(res.data?.filteredEvents ?? []))
      .catch(() => setPageError("Unable to reach the server."))
      .finally(() => setLoading(false));
  }, [gql]);

  /* Delete */
  const handleDelete = useCallback(async (event) => {
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
  }, [gql]);

  const handleSignOut = () => { clearAuthToken(); router.replace("/login"); };

  if (!gql) return null;

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
          <h1 className={styles.pageTitle}>Manage Events</h1>
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
          <div className={styles.filterGroup}>
            <label className={styles.filterLabel}>Region</label>
            <select
              className={styles.filterSelect}
              value={selectedRegion}
              onChange={(e) => setSelectedRegion(e.target.value)}
            >
              <option value="">All regions</option>
              {regions.map((r) => (
                <option key={r.id} value={r.id}>{r.name}</option>
              ))}
            </select>
          </div>

          <div className={styles.filterGroup}>
            <label className={styles.filterLabel}>Job Type</label>
            <select
              className={styles.filterSelect}
              value={selectedJobType}
              onChange={(e) => setSelectedJobType(e.target.value)}
            >
              <option value="">All job types</option>
              {jobTypes.map((j) => (
                <option key={j.id} value={j.id}>{j.name}</option>
              ))}
            </select>
          </div>

          <div className={styles.filterGroup}>
            <label className={styles.filterLabel}>Format</label>
            <select
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

          <div className={styles.filterActions}>
            <button className={styles.btnSearch} onClick={handleSearch} disabled={loading}>
              Search
            </button>
            <button className={styles.btnReset} onClick={handleReset}>
              Reset
            </button>
          </div>
        </div>

        {/* Loading */}
        {loading && (
          <div className={styles.stateBox}>
            <div className={styles.spinner} />
            <p>Loading events…</p>
          </div>
        )}

        {/* Table */}
        {!loading && events.length === 0 && (
          <div className={styles.stateBox}>
            <div className={styles.stateTitle}>No events found</div>
            <p>Try adjusting your filters, or create a new event.</p>
          </div>
        )}

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
