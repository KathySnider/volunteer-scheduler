"use client";

import { useEffect, useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import {
  getAuthToken,
  getAuthRole,
  getAuthName,
  clearAuthToken,
  volunteerGql,
} from "../lib/api";
import UserMenu from "../components/UserMenu";
import FeedbackButton from "../components/FeedbackButton";
import styles from "./my-shifts.module.css";

/* ----- GraphQL ----- */

const OWN_SHIFTS = `
  query OwnShifts($filter: ShiftTimeFilter!) {
    ownShifts(filter: $filter) {
      shiftId
      assignedAt
      startDateTime
      endDateTime
      jobName
      isVirtual
      preEventInstructions
      eventId
      eventName
      eventDescription
      venue {
        name
        address
        city
        state
      }
    }
  }
`;

const CANCEL_OWN = `
  mutation CancelOwn($shiftId: ID!) {
    cancelOwnShift(shiftId: $shiftId) {
      success
      message
    }
  }
`;

/* ----- Helpers ----- */

function formatDate(dtStr) {
  if (!dtStr) return "";
  return new Date(dtStr).toLocaleDateString(undefined, {
    weekday: "long", month: "long", day: "numeric", year: "numeric",
  });
}

function formatTime(dtStr) {
  if (!dtStr) return "";
  return new Date(dtStr).toLocaleTimeString(undefined, {
    hour: "numeric", minute: "2-digit",
  });
}

function formatTimeRange(start, end) {
  return `${formatTime(start)} – ${formatTime(end)}`;
}

/**
 * Group a flat list of VolunteerShift by eventId, preserving the
 * order of first appearance (earliest shift start within each event).
 */
function groupByEvent(shifts) {
  const order  = [];
  const map    = {};
  for (const s of shifts) {
    if (!map[s.eventId]) {
      map[s.eventId] = { eventId: s.eventId, eventName: s.eventName,
        eventDescription: s.eventDescription, venue: s.venue,
        isVirtual: s.isVirtual, shifts: [] };
      order.push(s.eventId);
    }
    map[s.eventId].shifts.push(s);
  }
  // Sort shifts within each event by start time
  for (const id of order) {
    map[id].shifts.sort((a, b) => (a.startDateTime < b.startDateTime ? -1 : 1));
  }
  return order.map((id) => map[id]);
}

/* ----- Page ----- */

export default function MyShiftsPage() {
  const router    = useRouter();
  const [gql,     setGql]     = useState(null);
  const [userName,setUserName]= useState("");
  const [isAdmin, setIsAdmin] = useState(false);

  const [filter,   setFilter]   = useState("UPCOMING");
  const [groups,   setGroups]   = useState([]);
  const [loading,  setLoading]  = useState(true);
  const [pageError,setPageError]= useState("");
  const [busy,     setBusy]     = useState(null);   // shiftId being cancelled
  const [message,  setMessage]  = useState(null);   // { type, text }

  /* ----- Load ----- */
  const loadShifts = useCallback((boundGql, f) => {
    setLoading(true);
    setPageError("");
    boundGql(OWN_SHIFTS, { filter: f })
      .then((res) => {
        if (res.errors) {
          setPageError(res.errors[0]?.message ?? "Error loading shifts.");
          return;
        }
        setGroups(groupByEvent(res.data?.ownShifts ?? []));
      })
      .catch(() => setPageError("Unable to reach the server."))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    const t = getAuthToken();
    if (!t) { router.replace("/login"); return; }
    const role = getAuthRole();
    // ownShifts lives on the volunteer endpoint; admin tokens are accepted there too
    const bound = (q, v) => volunteerGql(q, v, t);
    setGql(() => bound);
    setUserName(getAuthName() ?? "");
    setIsAdmin(role === "ADMINISTRATOR");
    loadShifts(bound, "UPCOMING");
  }, [router, loadShifts]);

  const handleFilterChange = (f) => {
    setFilter(f);
    if (gql) loadShifts(gql, f);
  };

  /* ----- Cancel ----- */
  const handleCancel = useCallback(async (shiftId) => {
    if (!gql) return;
    if (!window.confirm("Cancel your signup for this shift?")) return;
    setBusy(shiftId);
    setMessage(null);
    try {
      const res    = await gql(CANCEL_OWN, { shiftId });
      const result = res.data?.cancelOwnShift;
      if (res.errors || !result?.success) {
        setMessage({ type: "error", text: result?.message ?? res.errors?.[0]?.message ?? "Cancellation failed." });
      } else {
        setMessage({ type: "success", text: "Signup cancelled." });
        loadShifts(gql, filter);
      }
    } catch {
      setMessage({ type: "error", text: "Unable to reach the server." });
    } finally {
      setBusy(null);
    }
  }, [gql, filter, loadShifts]);

  const handleSignOut = () => { clearAuthToken(); router.replace("/login"); };

  if (!gql) return null;

  return (
    <div className={styles.page}>
      {/* Top bar */}
      <div className={styles.topBar}>
        <a href="/events" className={styles.backLink}>← Back to Events</a>
        <UserMenu name={userName} isAdmin={isAdmin} onSignOut={handleSignOut} />
      </div>

      <div className={styles.content}>
        <div className={styles.pageHeader}>
          <h1 className={styles.pageTitle}>My Shifts</h1>

          {/* Upcoming / Past / All toggle */}
          <div className={styles.filterToggle}>
            {["UPCOMING", "PAST", "ALL"].map((f) => (
              <button
                key={f}
                className={`${styles.toggleBtn} ${filter === f ? styles.toggleBtnActive : ""}`}
                onClick={() => handleFilterChange(f)}
              >
                {f.charAt(0) + f.slice(1).toLowerCase()}
              </button>
            ))}
          </div>
        </div>

        {/* Feedback banner */}
        {message && (
          <div className={message.type === "success" ? styles.successBanner : styles.errorBanner}>
            {message.text}
          </div>
        )}

        {/* Page-level error */}
        {pageError && <div className={styles.errorBanner}>{pageError}</div>}

        {/* Loading */}
        {loading && (
          <div className={styles.stateBox}>
            <div className={styles.spinner} />
            <p>Loading shifts…</p>
          </div>
        )}

        {/* Empty */}
        {!loading && !pageError && groups.length === 0 && (
          <div className={styles.stateBox}>
            <div className={styles.stateTitle}>No shifts found</div>
            <p>{filter === "UPCOMING" ? "You have no upcoming shifts." : "No shifts to show."}</p>
          </div>
        )}

        {/* Event groups */}
        {!loading && groups.map((group) => {
          // Earliest start / latest end across all shifts in this event group
          const firstStart = group.shifts[0]?.startDateTime;
          const lastEnd    = group.shifts[group.shifts.length - 1]?.endDateTime;

          const location = group.isVirtual && !group.venue
            ? "Virtual"
            : group.venue
              ? `${group.venue.city}, ${group.venue.state}`
              : null;

          return (
            <div key={group.eventId} className={styles.eventGroup}>
              {/* Event info block */}
              <div className={styles.eventBlock}>
                <h2 className={styles.eventName}>{group.eventName}</h2>
                <ul className={styles.metaList}>
                  {firstStart && (
                    <li className={styles.metaItem}>
                      <span className={styles.metaIcon}>📅</span>
                      <strong>{formatDate(firstStart)}</strong>
                    </li>
                  )}
                  {firstStart && lastEnd && (
                    <li className={styles.metaItem}>
                      <span className={styles.metaIcon}>🕐</span>
                      {formatTimeRange(firstStart, lastEnd)}
                    </li>
                  )}
                  {location && (
                    <li className={styles.metaItem}>
                      <span className={styles.metaIcon}>📍</span>
                      {location}
                    </li>
                  )}
                  {group.venue?.address && (
                    <li className={styles.metaItem}>
                      <span className={styles.metaIcon}>🏢</span>
                      <span className={styles.metaMuted}>
                        {group.venue.name ? `${group.venue.name} — ` : ""}
                        {group.venue.address}
                      </span>
                    </li>
                  )}
                </ul>
                {group.eventDescription && (
                  <p className={styles.eventDescription}>{group.eventDescription}</p>
                )}
              </div>

              {/* Shift rows */}
              <div className={styles.shiftList}>
                {group.shifts.map((shift) => (
                  <div key={shift.shiftId} className={styles.shiftRow}>
                    <div className={styles.shiftInfo}>
                      <span className={styles.shiftJob}>{shift.jobName}</span>
                      <span className={styles.shiftTime}>
                        {formatDate(shift.startDateTime)} · {formatTimeRange(shift.startDateTime, shift.endDateTime)}
                      </span>
                      {shift.preEventInstructions && (
                        <span className={styles.shiftInstructions}>
                          📋 {shift.preEventInstructions}
                        </span>
                      )}
                    </div>
                    {/* Only show cancel on upcoming shifts (filter === UPCOMING or ALL) */}
                    {filter !== "PAST" && (
                      <button
                        className={styles.btnCancel}
                        disabled={busy === shift.shiftId}
                        onClick={() => handleCancel(shift.shiftId)}
                      >
                        {busy === shift.shiftId ? "Cancelling…" : "Cancel Signup"}
                      </button>
                    )}
                  </div>
                ))}
              </div>
            </div>
          );
        })}
      </div>
      <FeedbackButton />
    </div>
  );
}
