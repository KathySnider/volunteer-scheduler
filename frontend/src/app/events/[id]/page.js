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

/** Format a DB timestamp string for display in the user's browser timezone. */
function formatDate(dtStr) {
  if (!dtStr) return "";
  const d = new Date(dtStr);
  return d.toLocaleDateString(undefined, {
    weekday: "long",
    month: "long",
    day: "numeric",
    year: "numeric",
  });
}

function formatTime(dtStr) {
  if (!dtStr) return "";
  const d = new Date(dtStr);
  return d.toLocaleTimeString(undefined, { hour: "numeric", minute: "2-digit" });
}

function formatTimeRange(startStr, endStr) {
  return `${formatTime(startStr)} – ${formatTime(endStr)}`;
}

/**
 * Group an array of ShiftView objects by jobName, preserving the
 * order of first appearance, and sorting shifts within each group
 * by startDateTime.
 */
function groupByJob(shifts) {
  const order = [];
  const map = {};
  for (const s of shifts) {
    if (!map[s.jobName]) {
      map[s.jobName] = [];
      order.push(s.jobName);
    }
    map[s.jobName].push(s);
  }
  for (const name of order) {
    map[name].sort((a, b) => (a.startDateTime < b.startDateTime ? -1 : 1));
  }
  return order.map((name) => ({ jobName: name, shifts: map[name] }));
}

/** Sum assigned/max across all shifts in a group. */
function groupTotals(shifts) {
  let assigned = 0;
  let max = 0;
  for (const s of shifts) {
    assigned += s.assignedVolunteers;
    max += s.maxVolunteers ?? 0;
  }
  return { assigned, max };
}

/* ----- Sub-components ----- */

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

function ShiftRow({ shift, isSignedUp, busy, onSignUp, onCancel }) {
  const isFull =
    shift.maxVolunteers !== null &&
    shift.assignedVolunteers >= shift.maxVolunteers;

  return (
    <div className={styles.shiftRow}>
      <div className={styles.shiftTime}>
        <div className={styles.shiftDate}>{formatDate(shift.startDateTime)}</div>
        <div className={styles.shiftTimeRange}>
          {formatTimeRange(shift.startDateTime, shift.endDateTime)}
        </div>
      </div>

      <div className={styles.shiftSpots}>
        <div className={styles.spotsLabel}>Spots</div>
        <div
          className={`${styles.spotsValue} ${
            isFull ? styles.countFull : styles.countOpen
          }`}
        >
          {shift.assignedVolunteers}/{shift.maxVolunteers ?? "∞"}
        </div>
      </div>

      <div className={styles.shiftAction}>
        {isSignedUp ? (
          <button
            className={styles.btnCancel}
            disabled={busy}
            onClick={() => onCancel(shift.id)}
          >
            {busy ? "Cancelling…" : "Cancel Signup"}
          </button>
        ) : isFull ? (
          <button className={styles.btnFull} disabled>
            Full
          </button>
        ) : (
          <button
            className={styles.btnSignUp}
            disabled={busy}
            onClick={() => onSignUp(shift.id)}
          >
            {busy ? "Signing up…" : "Sign Up"}
          </button>
        )}
      </div>
    </div>
  );
}

/* ----- Page ----- */

export default function EventDetailPage() {
  const router = useRouter();
  const params = useParams();
  const eventId = params?.id;

  const [gql, setGql] = useState(null);       // role-appropriate endpoint (for eventById)
  const [volGql, setVolGql] = useState(null); // always volunteer endpoint (shiftsForEvent, ownShifts, sign-up mutations)
  const [userName, setUserName] = useState("");
  const [isAdmin, setIsAdmin] = useState(false);

  // Data
  const [event, setEvent] = useState(null);
  const [groups, setGroups] = useState([]);
  const [signedUpIds, setSignedUpIds] = useState(new Set());

  // UI state
  const [loading, setLoading] = useState(true);
  const [pageError, setPageError] = useState("");
  const [actionBusy, setActionBusy] = useState(null); // shiftId currently being acted on
  const [actionMessage, setActionMessage] = useState(null); // { type: "success"|"error", text }

  /* Auth check + data load */
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
    // shiftsForEvent / ownShifts / sign-up mutations only exist on the
    // volunteer endpoint. Admin tokens are valid there too.
    const boundVolGql = (q, v) => volunteerGql(q, v, t);

    setGql(() => boundGql);
    setVolGql(() => boundVolGql);
    setUserName(getAuthName() ?? "");
    setIsAdmin(role === "ADMINISTRATOR");

    // Fetch all three in parallel
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
        if (shiftRes.errors) {
          setPageError(shiftRes.errors[0]?.message ?? "Error loading shifts.");
          return;
        }
        setEvent(evRes.data?.eventById ?? null);
        setGroups(groupByJob(shiftRes.data?.shiftsForEvent ?? []));

        const ids = new Set(
          (ownRes.data?.ownShifts ?? []).map((s) => s.shiftId)
        );
        setSignedUpIds(ids);
      })
      .catch(() => {
        setPageError("Unable to reach the server. Please try again.");
      })
      .finally(() => setLoading(false));
  }, [router, eventId]);

  /* Refresh shifts + own signups after a mutation */
  const refreshShifts = useCallback(
    async () => {
      if (!volGql) return;
      const [shiftRes, ownRes] = await Promise.all([
        volGql(SHIFTS_FOR_EVENT, { eventId }),
        volGql(OWN_SHIFTS, null),
      ]);
      if (!shiftRes.errors) {
        setGroups(groupByJob(shiftRes.data?.shiftsForEvent ?? []));
      }
      if (!ownRes.errors) {
        setSignedUpIds(
          new Set((ownRes.data?.ownShifts ?? []).map((s) => s.shiftId))
        );
      }
    },
    [volGql, eventId]
  );

  const handleSignUp = useCallback(
    async (shiftId) => {
      if (!volGql) return;
      setActionBusy(shiftId);
      setActionMessage(null);
      try {
        const res = await volGql(ASSIGN_SELF, { shiftId });
        const result = res.data?.assignSelfToShift;
        if (res.errors || !result?.success) {
          setActionMessage({
            type: "error",
            text: result?.message ?? res.errors?.[0]?.message ?? "Sign-up failed.",
          });
        } else {
          setActionMessage({ type: "success", text: "You're signed up!" });
          await refreshShifts();
        }
      } catch {
        setActionMessage({ type: "error", text: "Unable to reach the server." });
      } finally {
        setActionBusy(null);
      }
    },
    [volGql, refreshShifts]
  );

  const handleCancel = useCallback(
    async (shiftId) => {
      if (!volGql) return;
      setActionBusy(shiftId);
      setActionMessage(null);
      try {
        const res = await volGql(CANCEL_OWN, { shiftId });
        const result = res.data?.cancelOwnShift;
        if (res.errors || !result?.success) {
          setActionMessage({
            type: "error",
            text: result?.message ?? res.errors?.[0]?.message ?? "Cancellation failed.",
          });
        } else {
          setActionMessage({ type: "success", text: "Signup cancelled." });
          await refreshShifts();
        }
      } catch {
        setActionMessage({ type: "error", text: "Unable to reach the server." });
      } finally {
        setActionBusy(null);
      }
    },
    [volGql, refreshShifts]
  );

  const handleSignOut = () => {
    clearAuthToken();
    router.replace("/login");
  };

  return (
    <div className={styles.page}>
      {/* Top bar */}
      <div className={styles.topBar}>
        <a href="/events" className={styles.backLink}>
          ← Back to Events
        </a>
        <div className={styles.userArea}>
          <UserMenu
            name={userName}
            isAdmin={isAdmin}
            onSignOut={handleSignOut}
          />
        </div>
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
              <div
                className={
                  actionMessage.type === "success"
                    ? styles.successBanner
                    : styles.errorBox
                }
              >
                {actionMessage.text}
              </div>
            )}

            {/* Event info card */}
            <div className={styles.eventCard}>
              <h1 className={styles.eventName}>{event.name}</h1>

              <ul className={styles.metaList}>
                {event.eventDates.map((d, i) => (
                  <li key={i} className={styles.metaItem}>
                    <span className={styles.metaIcon}>📅</span>
                    <span>
                      {formatDate(d.startDateTime)}
                      <span className={styles.metaMuted}>
                        {" "}
                        &mdash; {formatTimeRange(d.startDateTime, d.endDateTime)}
                      </span>
                    </span>
                  </li>
                ))}

                {event.eventType !== "VIRTUAL" && event.venue && (
                  <>
                    <li className={styles.metaItem}>
                      <span className={styles.metaIcon}>📍</span>
                      <span>
                        {event.venue.city}, {event.venue.state}
                      </span>
                    </li>
                    {event.venue.address && (
                      <li className={styles.metaItem}>
                        <span className={styles.metaIcon}>🏢</span>
                        <span className={styles.metaMuted}>
                          {event.venue.name
                            ? `${event.venue.name} — `
                            : ""}
                          {event.venue.address}
                        </span>
                      </li>
                    )}
                  </>
                )}

                <li className={styles.metaItem}>
                  <FormatBadge eventType={event.eventType} />
                </li>
              </ul>

              {event.description && (
                <>
                  <div className={styles.descriptionHeading}>
                    About This Event
                  </div>
                  <p className={styles.description}>{event.description}</p>
                </>
              )}
            </div>

            {/* Volunteer opportunities */}
            <h2 className={styles.sectionHeading}>Volunteer Opportunities</h2>

            {groups.length === 0 ? (
              <p style={{ color: "var(--color-text-muted)" }}>
                No shifts have been added to this event yet.
              </p>
            ) : (
              groups.map(({ jobName, shifts }) => {
                const { assigned, max } = groupTotals(shifts);
                const isFull = max > 0 && assigned >= max;
                return (
                  <div key={jobName} className={styles.jobGroup}>
                    <div className={styles.jobGroupHeader}>
                      <span className={styles.jobName}>{jobName}</span>
                      <span className={styles.jobCount}>
                        Volunteers&nbsp;
                        <span
                          className={`${styles.jobCountValue} ${
                            isFull ? styles.countFull : styles.countOpen
                          }`}
                        >
                          {assigned}/{max}
                        </span>
                      </span>
                    </div>

                    {shifts.map((shift) => (
                      <ShiftRow
                        key={shift.id}
                        shift={shift}
                        isSignedUp={signedUpIds.has(shift.id)}
                        busy={actionBusy === shift.id}
                        onSignUp={handleSignUp}
                        onCancel={handleCancel}
                      />
                    ))}
                  </div>
                );
              })
            )}
          </>
        )}
      </div>
    </div>
  );
}
