"use client";

import { useEffect, useState, useCallback } from "react";
import { useRouter, useParams } from "next/navigation";
import {
  isAuthenticated,
  hasAuthRole,
  Roles,
  getAuthName,
  signOut,
  volunteerGql,
  getOwnShifts,
  setOwnShiftsCache,
} from "../../lib/api";
import AdminTopBar from "../../components/AdminTopBar";
import FeedbackButton from "../../components/FeedbackButton";
import styles from "./event-detail.module.css";

/* ----- GraphQL operations ----- */

const EVENT_DETAIL = `
  query EventDetail($eventId: ID!) {
    eventView(eventId: $eventId) {
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
    eventShiftViews(eventId: $eventId) {
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

// Direct own-shifts fetch used by refreshShifts — bypasses the module-level
// cache so signed-up status is always authoritative after a mutation.
const OWN_SHIFTS_FRESH = `
  query {
    ownShifts(filter: UPCOMING) {
      shiftId
      startDateTime
      endDateTime
      eventName
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

/**
 * Return the subset of the volunteer's own upcoming shifts that overlap with
 * the given candidate shift.  Excludes the shift itself (already signed up).
 * Uses a half-open interval: overlap when start1 < end2 AND end1 > start2.
 */
function findConflicts(shift, ownShifts) {
  const s1 = new Date(shift.startDateTime).getTime();
  const e1 = new Date(shift.endDateTime).getTime();
  return ownShifts.filter((own) => {
    if (own.shiftId === String(shift.id)) return false; // same shift — already signed up
    const s2 = new Date(own.startDateTime).getTime();
    const e2 = new Date(own.endDateTime).getTime();
    return s1 < e2 && e1 > s2;
  });
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
function ShiftRow({ shift, isSignedUp, busy, onSignUp, onCancel, conflictingShifts }) {
  const open   = spotsOpen(shift);
  const isFull = open !== null && open === 0;
  const hasConflict = !isSignedUp && conflictingShifts && conflictingShifts.length > 0;

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
      <button
        className={hasConflict ? styles.btnSignUpAnyway : styles.btnSignUp}
        disabled={busy}
        onClick={() => onSignUp(shift.id)}
      >
        {busy ? "Signing up…" : hasConflict ? "Sign Up Anyway" : "Sign Up"}
      </button>
    );
  }

  // Conflict warning text — names up to 2 conflicting events
  let conflictNote = null;
  if (hasConflict) {
    const names = conflictingShifts.map((c) => c.eventName).filter(Boolean);
    const label = names.length === 1
      ? names[0]
      : names.length === 2
        ? `${names[0]} and ${names[1]}`
        : `${names[0]} and ${names.length - 1} others`;
    conflictNote = (
      <div className={styles.conflictWarning}>
        ⚠️ Overlaps with your shift: <strong>{label}</strong>
      </div>
    );
  }

  return (
    <div className={styles.shiftRow}>
      <div className={styles.shiftLeft}>
        <span className={styles.shiftTime}>
          {formatDate(shift.startDateTime)} · {formatTimeRange(shift.startDateTime, shift.endDateTime)}
        </span>
        {conflictNote}
      </div>
      <div className={styles.shiftRowRight}>
        {spotsLabel}
        {btn}
      </div>
    </div>
  );
}

/* ----- JobGroupCard — one card per job name, shifts listed inside -----
   IMPORTANT: Defined at module level to prevent React from remounting
   this component on every render (which would reset state and lose focus).

   accordion={true}  → collapsible; starts open if the volunteer already has
                        a signed-up shift in this group, otherwise starts closed.
   accordion={false} → always expanded (used when there is only one job type). */
function JobGroupCard({ jobName, shifts, serviceTypes, signedUpIds, ownShifts, actionBusy, onSignUp, onCancel, accordion }) {
  const [isOpen, setIsOpen] = useState(() => {
    if (!accordion) return true;
    // Auto-open any group the volunteer is already signed up for.
    return shifts.some((s) => signedUpIds.has(s.id));
  });

  /* The badges + shift rows that are toggled in accordion mode */
  const shiftContent = (
    <>
      {serviceTypes && serviceTypes.length > 0 && (
        <div className={styles.jobBadges}>
          {serviceTypes.map((st) => (
            <span key={st} className={styles.serviceTypeBadge}>{st}</span>
          ))}
        </div>
      )}
      {shifts.map((shift) => (
        <ShiftRow
          key={shift.id}
          shift={shift}
          isSignedUp={signedUpIds.has(shift.id)}
          busy={actionBusy === shift.id}
          onSignUp={onSignUp}
          onCancel={onCancel}
          conflictingShifts={findConflicts(shift, ownShifts)}
        />
      ))}
    </>
  );

  return (
    <div className={styles.jobCard}>
      {/* Header — clickable button in accordion mode, plain div otherwise */}
      {accordion ? (
        <button
          type="button"
          className={styles.jobCardHeaderBtn}
          onClick={() => setIsOpen((o) => !o)}
          aria-expanded={isOpen}
        >
          <span className={styles.jobName}>{jobName}</span>
          <span className={`${styles.chevron} ${isOpen ? styles.chevronOpen : ""}`}>▾</span>
        </button>
      ) : (
        <div className={styles.jobCardHeader}>
          <span className={styles.jobName}>{jobName}</span>
        </div>
      )}

      {/* Content — animated collapse when in accordion mode */}
      {accordion ? (
        <div className={`${styles.shiftList} ${isOpen ? styles.shiftListOpen : ""}`}>
          <div className={styles.shiftListInner}>{shiftContent}</div>
        </div>
      ) : (
        shiftContent
      )}
    </div>
  );
}

/* ----- Page ----- */

export default function EventDetailPage() {
  const router  = useRouter();
  const params  = useParams();
  const eventId = params?.id;
  const [volGql,    setVolGql]    = useState(null);
  const [userName,  setUserName]  = useState("");
  const [isAdmin,      setIsAdmin]      = useState(false);
  const [feedbackOpen, setFeedbackOpen] = useState(false);

  const [event,       setEvent]       = useState(null);
  const [shifts,      setShifts]      = useState([]);
  const [ownShifts,   setOwnShifts]   = useState([]);  // cached upcoming shifts for conflict detection
  const [signedUpIds, setSignedUpIds] = useState(new Set());

  const [loading,       setLoading]       = useState(true);
  const [pageError,     setPageError]     = useState("");
  const [actionBusy,    setActionBusy]    = useState(null);
  const [actionMessage, setActionMessage] = useState(null);

  /* Auth check + initial data load */
  useEffect(() => {
    if (!isAuthenticated()) { router.replace("/login"); return; }
    setVolGql(() => volunteerGql);
    setUserName(getAuthName() ?? "");
    setIsAdmin(hasAuthRole(Roles.ADMINISTRATOR));

    Promise.all([
      volunteerGql(EVENT_DETAIL, { eventId }),
      volunteerGql(SHIFTS_FOR_EVENT, { eventId }),
      getOwnShifts().catch(() => []),
    ])
      .then(([evRes, shiftRes, cached]) => {
        if (evRes.errors) {
          setPageError(evRes.errors[0]?.message ?? "Error loading event.");
          return;
        }
        setEvent(evRes.data?.eventView ?? null);

        // Sort shifts by start time
        const sorted = [...(shiftRes.data?.eventShiftViews ?? [])];
        sorted.sort((a, b) => (a.startDateTime < b.startDateTime ? -1 : 1));
        setShifts(sorted);

        // Own shifts from cache — used for both signed-up status and conflict detection
        setOwnShifts(cached);
        setSignedUpIds(new Set(cached.map((s) => s.shiftId)));
      })
      .catch(() => setPageError("Unable to reach the server. Please try again."))
      .finally(() => setLoading(false));
  }, [router, eventId]);

  /* Refresh event shifts after a sign-up or cancel mutation.
     Queries the server directly — never reads from the module-level cache —
     so signedUpIds and shift counts are always authoritative.
     After the fetch, the cache is repopulated so conflict detection on other
     event pages stays accurate. */
  const refreshShifts = useCallback(async () => {
    if (!volGql) return;
    const [shiftRes, ownRes] = await Promise.all([
      volGql(SHIFTS_FOR_EVENT, { eventId }),
      volunteerGql(OWN_SHIFTS_FRESH).catch(() => null),
    ]);
    if (shiftRes?.data?.eventShiftViews) {
      const sorted = [...shiftRes.data.eventShiftViews];
      sorted.sort((a, b) => (a.startDateTime < b.startDateTime ? -1 : 1));
      setShifts(sorted);
    }
    const ownList = ownRes?.data?.ownShifts ?? null;
    if (ownList !== null) {
      // Keep the module-level cache fresh for conflict detection elsewhere.
      setOwnShiftsCache(ownList);
      setOwnShifts(ownList);
      setSignedUpIds(new Set(ownList.map((s) => s.shiftId)));
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

  const handleSignOut = async () => { await signOut(); router.replace("/login"); };

  /* ----- Derived render data ----- */
  const jobs         = shifts.length > 0 ? groupByJob(shifts) : [];
  const useAccordion = jobs.length > 1;

  /* ----- Render ----- */
  return (
    <div className={styles.page}>
      <AdminTopBar userName={userName} isAdmin={isAdmin} onSignOut={handleSignOut} onFeedbackOpen={() => setFeedbackOpen(true)} />

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
              jobs.map(({ jobName, shifts: jobShifts }) => (
                <JobGroupCard
                  key={jobName}
                  jobName={jobName}
                  shifts={jobShifts}
                  serviceTypes={event.serviceTypes ?? []}
                  signedUpIds={signedUpIds}
                  ownShifts={ownShifts}
                  actionBusy={actionBusy}
                  onSignUp={handleSignUp}
                  onCancel={handleCancel}
                  accordion={useAccordion}
                />
              ))
            )}
          </>
        )}
      </div>
      <FeedbackButton open={feedbackOpen} onClose={() => setFeedbackOpen(false)} />
    </div>
  );
}
