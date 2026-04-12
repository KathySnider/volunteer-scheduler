"use client";

import { useEffect, useState, useCallback } from "react";
import { useRouter, useParams } from "next/navigation";
import {
  getAuthToken,
  getAuthRole,
  getAuthName,
  clearAuthToken,
  volunteerGql,
  adminGql,
} from "../../lib/api";
import UserMenu from "../../components/UserMenu";
import styles from "./event-detail.module.css";

/* ----- GraphQL operations ----- */

const EVENT_DETAIL = `
  query EventDetail($eventId: ID!) {
    eventById(eventId: $eventId) {
      id
      name
      description
      eventType
      serviceTypes
      venue {
        name
        address
        city
        state
      }
      eventDates {
        startDateTime
        endDateTime
      }
    }
  }
`;

const SHIFTS_FOR_EVENT = `
  query ShiftsForEvent($eventId: ID!) {
    shiftsForEvent(eventId: $eventId) {
      id
      jobName
      startDateTime
      endDateTime
      isVirtual
      maxVolunteers
      assignedVolunteers
    }
  }
`;

const OWN_SHIFTS = `
  query {
    ownShifts(filter: UPCOMING) {
      shiftId
    }
  }
`;

const ASSIGN_SELF = `
  mutation AssignSelf($shiftId: ID!) {
    assignSelfToShift(shiftId: $shiftId) {
      success
      message
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

function formatTimeRange(startStr, endStr) {
  return `${formatTime(startStr)} – ${formatTime(endStr)}`;
}

/** How many open spots for a single shift; null means unlimited. */
function spotsOpen(shift) {
  if (shift.maxVolunteers === null || shift.maxVolunteers === undefined) return null;
  return Math.max(0, shift.maxVolunteers - shift.assignedVolunteers);
}

/** Group an array of shifts by jobName, sorted by first shift start time. */
function groupByJob(shifts) {
  const order = [];
  const map   = {};
  for (const s of shifts) {
    if (!map[s.jobName]) { map[s.jobName] = []; order.push(s.jobName); }
    map[s.jobName].push(s);
  }
  return order.map((name) => ({
    jobName: name,
    shifts:  map[name].slice().sort((a, b) => (a.startDateTime < b.startDateTime ? -1 : 1)),
  }));
}

/** Sum assigned/max across all shifts in a group. */
function groupTotals(shifts) {
  let assigned = 0, max = 0, hasMax = false;
  for (const s of shifts) {
    assigned += s.assignedVolunteers;
    if (s.maxVolunteers != null) { max += s.maxVolunteers; hasMax = true; }
  }
  return { assigned, max: hasMax ? max : null };
}

/* ----- Constants ----- */

const FORMAT_LABELS = {
  VIRTUAL:   "Virtual",
  IN_PERSON: "In-Person",
  HYBRID:    "Hybrid",
};

const FORMAT_BADGE_CLASS = {
  VIRTUAL:   styles.badgeVirtual,
  IN_PERSON: styles.badgeInPerson,
  HYBRID:    styles.badgeHybrid,
};

/* ----- ShiftRow — one row per shift inside a job group card ----- */
function ShiftRow({ shift, isSignedUp, busy, onSignUp, onCancel }) {
  const open   = spotsOpen(shift);
  const isFull = open !== null && open === 0;

  // Spots label — shown as plain text, not on the button
  let spotsLabel = null;
  if (isSignedUp) {
    spotsLabel = <span className={styles.signedUpLabel}>You're signed up</span>;
  } else if (isFull) {
    spotsLabel = <span className={styles.fullLabel}>Full</span>;
  } else if (open !== null) {
    spotsLabel = <span className={styles.spotsLabel}>{open} spot{open === 1 ? "" : "s"} available</span>;
  }

  // Action button — only shown when there's something to do
  let btn = null;
  if (isSignedUp) {
    btn = (
      <button className={styles.btnCancel} disabled={busy} onClick={() => onCancel(shift.id)}>
        {busy ? "Cancelling…" : "Cancel Signup"}
      </button>
    );
  } else if (!isFull) {
    btn = (
      <button className={styles.btnSignUp} disabled={busy} onClick={() => onSignUp(shift.id)}>
        {busy ? "Signing up…" : "Sign Up"}
      </button>
    );
  }

  return (
    <div className={styles.shiftRow}>
      <span className={styles.shiftTime}>
        {formatDate(shift.startDateTime)} · {formatTimeRange(shift.startDateTime, shift.endDateTime)}
      </span>
      <div className={styles.shiftRowRight}>
        {spotsLabel}
        {btn}
      </div>
    </div>
  );
}

/* ----- JobGroupCard — one card per job name, shifts listed inside -----
   IMPORTANT: Defined at module level to prevent React from remounting
   this component on every render (which would reset state and lose focus). */
function JobGroupCard({ jobName, shifts, serviceTypes, signedUpIds, actionBusy, onSignUp, onCancel }) {
  return (
    <div className={styles.jobCard}>
      {/* Header: job name only */}
      <div className={styles.jobCardHeader}>
        <span className={styles.jobName}>{jobName}</span>
      </div>

      {/* Service type badges */}
      {serviceTypes && serviceTypes.length > 0 && (
        <div className={styles.jobBadges}>
          {serviceTypes.map((st) => (
            <span key={st} className={styles.serviceTypeBadge}>{st}</span>
          ))}
        </div>
      )}

      {/* One shift row per shift */}
      {shifts.map((shift) => (
        <ShiftRow
          key={shift.id}
          shift={shift}
          isSignedUp={signedUpIds.has(shift.id)}
          busy={actionBusy === shift.id}
          onSignUp={onSignUp}
          onCancel={onCancel}
        />
      ))}
    </div>
  );
}

/* ----- Page ----- */

export default function EventDetailPage() {
  const router  = useRouter();
  const params  = useParams();
  const eventId = params?.id;

  const [gql,       setGql]       = useState(null);
  const [volGql,    setVolGql]    = useState(null);
  const [userName,  setUserName]  = useState("");
  const [isAdmin,   setIsAdmin]   = useState(false);

  const [event,       setEvent]       = useState(null);
  const [shifts,      setShifts]      = useState([]);
  const [signedUpIds, setSignedUpIds] = useState(new Set());

  const [loading,       setLoading]       = useState(true);
  const [pageError,     setPageError]     = useState("");
  const [actionBusy,    setActionBusy]    = useState(null);
  const [actionMessage, setActionMessage] = useState(null);

  /* Auth check + initial data load */
  useEffect(() => {
    const t = getAuthToken();
    if (!t) { router.replace("/login"); return; }
    const role = getAuthRole();
    const boundGql    = role === "ADMINISTRATOR"
      ? (q, v) => adminGql(q, v, t)
      : (q, v) => volunteerGql(q, v, t);
    const boundVolGql = (q, v) => volunteerGql(q, v, t);

    setGql(() => boundGql);
    setVolGql(() => boundVolGql);
    setUserName(getAuthName() ?? "");
    setIsAdmin(role === "ADMINISTRATOR");

    Promise.all([
      boundGql(EVENT_DETAIL, { eventId }),
      boundVolGql(SHIFTS_FOR_EVENT, { eventId }),
      boundVolGql(OWN_SHIFTS, null),
    ])
      .then(([evRes, shiftRes, ownRes]) => {
        if (evRes.errors) {
          setPageError(evRes.errors[0]?.message ?? "Error loading event.");
          return;
        }
        setEvent(evRes.data?.eventById ?? null);

        // Sort shifts by start time
        const sorted = [...(shiftRes.data?.shiftsForEvent ?? [])];
        sorted.sort((a, b) => (a.startDateTime < b.startDateTime ? -1 : 1));
        setShifts(sorted);

        setSignedUpIds(
          new Set((ownRes.data?.ownShifts ?? []).map((s) => s.shiftId))
        );
      })
      .catch(() => setPageError("Unable to reach the server. Please try again."))
      .finally(() => setLoading(false));
  }, [router, eventId]);

  /* Refresh shifts + own signups after a mutation */
  const refreshShifts = useCallback(async () => {
    if (!volGql) return;
    const [shiftRes, ownRes] = await Promise.all([
      volGql(SHIFTS_FOR_EVENT, { eventId }),
      volGql(OWN_SHIFTS, null),
    ]);
    if (!shiftRes.errors) {
      const sorted = [...(shiftRes.data?.shiftsForEvent ?? [])];
      sorted.sort((a, b) => (a.startDateTime < b.startDateTime ? -1 : 1));
      setShifts(sorted);
    }
    if (!ownRes.errors) {
      setSignedUpIds(new Set((ownRes.data?.ownShifts ?? []).map((s) => s.shiftId)));
    }
  }, [volGql, eventId]);

  const handleSignUp = useCallback(async (shiftId) => {
    if (!volGql) return;
    setActionBusy(shiftId);
    setActionMessage(null);
    try {
      const res    = await volGql(ASSIGN_SELF, { shiftId });
      const result = res.data?.assignSelfToShift;
      if (res.errors || !result?.success) {
        setActionMessage({ type: "error", text: result?.message ?? res.errors?.[0]?.message ?? "Sign-up failed." });
      } else {
        setActionMessage({ type: "success", text: "You're signed up!" });
        await refreshShifts();
      }
    } catch {
      setActionMessage({ type: "error", text: "Unable to reach the server." });
    } finally {
      setActionBusy(null);
    }
  }, [volGql, refreshShifts]);

  const handleCancel = useCallback(async (shiftId) => {
    if (!volGql) return;
    setActionBusy(shiftId);
    setActionMessage(null);
    try {
      const res    = await volGql(CANCEL_OWN, { shiftId });
      const result = res.data?.cancelOwnShift;
      if (res.errors || !result?.success) {
        setActionMessage({ type: "error", text: result?.message ?? res.errors?.[0]?.message ?? "Cancellation failed." });
      } else {
        setActionMessage({ type: "success", text: "Signup cancelled." });
        await refreshShifts();
      }
    } catch {
      setActionMessage({ type: "error", text: "Unable to reach the server." });
    } finally {
      setActionBusy(null);
    }
  }, [volGql, refreshShifts]);

  const handleSignOut = () => { clearAuthToken(); router.replace("/login"); };

  /* ----- Render ----- */
  return (
    <div className={styles.page}>
      {/* Top bar */}
      <div className={styles.topBar}>
        <a href="/events" className={styles.backLink}>← Back to Events</a>
        <UserMenu name={userName} isAdmin={isAdmin} onSignOut={handleSignOut} />
      </div>

      <div className={styles.content}>

        {/* Loading */}
        {loading && (
          <div className={styles.stateBox}>
            <div className={styles.spinner} />
            <p>Loading event…</p>
          </div>
        )}

        {/* Page-level error */}
        {!loading && pageError && (
          <div className={styles.errorBox}>{pageError}</div>
        )}

        {/* Main content */}
        {!loading && event && (
          <>
            {/* Action feedback banner */}
            {actionMessage && (
              <div className={actionMessage.type === "success" ? styles.successBanner : styles.errorBox}>
                {actionMessage.text}
              </div>
            )}

            {/* ── Event info card ── */}
            <div className={styles.eventCard}>
              <h1 className={styles.eventName}>{event.name}</h1>

              <ul className={styles.metaList}>
                {/* Dates */}
                {event.eventDates.map((d, i) => (
                  <li key={i} className={styles.metaItem}>
                    <span className={styles.metaIcon}>📅</span>
                    <span className={styles.metaText}>
                      <strong>{formatDate(d.startDateTime)}</strong>
                    </span>
                  </li>
                ))}
                {/* Time range (from first date) */}
                {event.eventDates.length > 0 && (
                  <li className={styles.metaItem}>
                    <span className={styles.metaIcon}>🕐</span>
                    <span className={styles.metaText}>
                      {formatTimeRange(
                        event.eventDates[0].startDateTime,
                        event.eventDates[event.eventDates.length - 1].endDateTime,
                      )}
                    </span>
                  </li>
                )}

                {/* Location */}
                {event.eventType !== "VIRTUAL" && event.venue && (
                  <>
                    <li className={styles.metaItem}>
                      <span className={styles.metaIcon}>📍</span>
                      <span className={styles.metaText}>
                        {event.venue.city}, {event.venue.state}
                      </span>
                    </li>
                    {event.venue.address && (
                      <li className={styles.metaItem}>
                        <span className={styles.metaIcon}>🏢</span>
                        <span className={`${styles.metaText} ${styles.metaMuted}`}>
                          {event.venue.name ? `${event.venue.name} — ` : ""}
                          {event.venue.address}
                        </span>
                      </li>
                    )}
                  </>
                )}

                {/* Format badge */}
                <li className={styles.metaItem}>
                  <span className={`${styles.formatBadge} ${FORMAT_BADGE_CLASS[event.eventType] ?? ""}`}>
                    {FORMAT_LABELS[event.eventType] ?? event.eventType}
                  </span>
                </li>
              </ul>

              {event.description && (
                <>
                  <div className={styles.descriptionHeading}>About This Event</div>
                  <p className={styles.description}>{event.description}</p>
                </>
              )}
            </div>

            {/* ── Volunteer Opportunities ── */}
            <h2 className={styles.sectionHeading}>Volunteer Opportunities</h2>

            {shifts.length === 0 ? (
              <p className={styles.noShifts}>No shifts have been added to this event yet.</p>
            ) : (
              groupByJob(shifts).map(({ jobName, shifts: jobShifts }) => (
                <JobGroupCard
                  key={jobName}
                  jobName={jobName}
                  shifts={jobShifts}
                  serviceTypes={event.serviceTypes ?? []}
                  signedUpIds={signedUpIds}
                  actionBusy={actionBusy}
                  onSignUp={handleSignUp}
                  onCancel={handleCancel}
                />
              ))
            )}
          </>
        )}
      </div>
    </div>
  );
}
