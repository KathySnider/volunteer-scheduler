"use client";

import { useEffect, useState, useCallback } from "react";
import { useRouter, useParams } from "next/navigation";
import {
  getAuthToken,
  getAuthRole,
  getAuthName,
  clearAuthToken,
  adminGql,
} from "../../../lib/api";
import UserMenu from "../../../components/UserMenu";
import styles from "./admin-event-detail.module.css";

/* =========================================================
   GraphQL
   ========================================================= */

const ADMIN_EVENT_DETAIL = `
  query AdminEventDetail($eventId: ID!) {
    eventById(eventId: $eventId) {
      id name description eventType
      venue { id name city state timezone region }
      eventDates { id startDateTime endDateTime }
      serviceTypes
    }
    opportunitiesForEvent(eventId: $eventId) {
      id jobId isVirtual preEventInstructions
      shifts { id startDateTime endDateTime maxVolunteers staffContactId }
    }
    lookupValues {
      serviceTypes { id name }
      jobTypes { id name }
    }
    venues { id name city state timezone }
    staff { id firstName lastName position }
  }
`;

const UPDATE_EVENT      = `mutation UpdateEvent($event: UpdateEventInput!) { updateEvent(event: $event) { success message } }`;
const DELETE_EVENT      = `mutation DeleteEvent($eventId: ID!) { deleteEvent(eventId: $eventId) { success message } }`;
const CREATE_EVENT_DATE = `mutation CreateEventDate($newDate: AddEventDateInput!) { createEventDate(newDate: $newDate) { success message id } }`;
const UPDATE_EVENT_DATE = `mutation UpdateEventDate($date: UpdateEventDateInput!) { updateEventDate(date: $date) { success message } }`;
const DELETE_EVENT_DATE = `mutation DeleteEventDate($eventDateId: ID!) { deleteEventDate(eventDateId: $eventDateId) { success message } }`;
const CREATE_OPP        = `mutation CreateOpp($newOpp: NewOpportunityInput!) { createOpportunity(newOpp: $newOpp) { success message id } }`;
const UPDATE_OPP        = `mutation UpdateOpp($opp: UpdateOpportunityInput!) { updateOpportunity(opp: $opp) { success message } }`;
const DELETE_OPP        = `mutation DeleteOpp($oppId: ID!) { deleteOpportunity(oppId: $oppId) { success message } }`;
const CREATE_SHIFT      = `mutation CreateShift($newShift: AddShiftInput!) { createShift(newShift: $newShift) { success message id } }`;
const UPDATE_SHIFT      = `mutation UpdateShift($shift: UpdateShiftInput!) { updateShift(shift: $shift) { success message } }`;
const DELETE_SHIFT      = `mutation DeleteShift($shiftId: ID!) { deleteShift(shiftId: $shiftId) { success message } }`;

/* =========================================================
   Helpers
   ========================================================= */

/**
 * Normalize a user-typed time string to "HH:MM".
 * Strips non-digits, takes the first 4 digits, and formats as HH:MM.
 * e.g. "005:45" → "00:45" is wrong; better: strip to digits "0545" → "05:45"
 * Handles "9:00", "09:00", "0545", "005:45" etc.
 */
function normalizeTime(t) {
  if (!t) return "00:00";
  // If a colon is present, parse h and m separately so "5:45" → "05:45"
  if (t.includes(":")) {
    const [h, m] = t.split(":");
    const hh = h.replace(/\D/g, "").padStart(2, "0").slice(-2);
    const mm = m.replace(/\D/g, "").padStart(2, "0").slice(0, 2);
    return `${hh}:${mm}`;
  }
  // No colon — treat as raw digits, left-pad to 4: "545" → "0545"
  const digits = t.replace(/\D/g, "").padStart(4, "0").slice(-4);
  return `${digits.slice(0, 2)}:${digits.slice(2, 4)}`;
}

/** Convert a "YYYY-MM-DDTHH:MM" value to backend format ("2026-05-10 09:00:00").
 *  Normalizes the time portion so user typos like "005:45" become "05:45". */
function toBackendDateTime(dtLocal) {
  if (!dtLocal) return "";
  const [d, t] = dtLocal.split("T");
  return `${d} ${normalizeTime(t)}:00`;
}

/** Split a "YYYY-MM-DDTHH:MM" string into its date and time parts. */
function splitDT(dtLocal) {
  if (!dtLocal) return { d: "", t: "" };
  const [d, t] = dtLocal.split("T");
  return { d: d ?? "", t: t ?? "" };
}

/** Combine separate date + time strings back into "YYYY-MM-DDTHH:MM". */
function joinDT(d, t) {
  if (!d) return "";
  return `${d}T${t || "00:00"}`;
}

/** Convert UTC ISO string to datetime-local value in a given IANA timezone. */
function toDatetimeLocal(utcString, ianaZone) {
  if (!utcString) return "";
  try {
    const dt = new Date(utcString);
    const parts = new Intl.DateTimeFormat("en-CA", {
      timeZone: ianaZone || "UTC",
      year: "numeric", month: "2-digit", day: "2-digit",
      hour: "2-digit", minute: "2-digit", hour12: false,
    }).formatToParts(dt);
    const get = (type) => parts.find((p) => p.type === type)?.value ?? "00";
    const hh = get("hour") === "24" ? "00" : get("hour");
    return `${get("year")}-${get("month")}-${get("day")}T${hh}:${get("minute")}`;
  } catch {
    return "";
  }
}

function formatDisplay(utcString, ianaZone) {
  if (!utcString) return "—";
  return new Date(utcString).toLocaleString(undefined, {
    timeZone: ianaZone || undefined,
    month: "short", day: "numeric", year: "numeric",
    hour: "numeric", minute: "2-digit",
  });
}

const FORMAT_LABEL = { VIRTUAL: "Virtual", IN_PERSON: "In Person", HYBRID: "Hybrid" };

const US_TIMEZONES = [
  { value: "America/New_York",    label: "Eastern (ET)" },
  { value: "America/Chicago",     label: "Central (CT)" },
  { value: "America/Denver",      label: "Mountain (MT)" },
  { value: "America/Los_Angeles", label: "Pacific (PT)" },
  { value: "America/Anchorage",   label: "Alaska (AKT)" },
  { value: "Pacific/Honolulu",    label: "Hawaii (HT)" },
];

function eventIanaZone(event) {
  return event?.venue?.timezone
    || Intl.DateTimeFormat().resolvedOptions().timeZone;
}

/* =========================================================
   Page
   ========================================================= */

const EMPTY_SHIFT_FORM = {
  start: "", end: "", ianaZone: "", maxVolunteers: "", staffContactId: "",
};

/* =========================================================
   ShiftFormFields
   =========================================================
   IMPORTANT: This component MUST remain defined at the module
   level (outside AdminEventDetailPage). If it is moved inside
   the page component, React will treat it as a new component
   type on every render, unmount/remount it each time, and
   every keystroke will steal focus from the text inputs.
   ========================================================= */

function ShiftFormFields({ form, setForm, staff }) {
  const startParts = splitDT(form.start);
  const endParts   = splitDT(form.end);
  return (
    <div className={styles.grid2}>
      <div className={styles.field}>
        <label className={styles.label}>Start Date <span className={styles.required}>*</span></label>
        <input
          type="date"
          className={styles.input}
          value={startParts.d}
          onChange={(e) => setForm((p) => {
            const newStart = joinDT(e.target.value, splitDT(p.start).t);
            // Default end date to start date if end date not yet set
            const endD = splitDT(p.end).d;
            const newEnd = endD ? p.end : joinDT(e.target.value, splitDT(p.end).t);
            return { ...p, start: newStart, end: newEnd };
          })}
        />
      </div>
      <div className={styles.field}>
        <label className={styles.label}>Start Time <span className={styles.required}>*</span></label>
        <input
          type="text" placeholder="HH:MM"
          className={styles.input}
          value={startParts.t}
          onFocus={(e) => e.target.select()}
          onChange={(e) => setForm((p) => ({ ...p, start: joinDT(splitDT(p.start).d, e.target.value) }))}
        />
      </div>
      <div className={styles.field}>
        <label className={styles.label}>End Date <span className={styles.required}>*</span></label>
        <input
          type="date"
          className={styles.input}
          value={endParts.d}
          onChange={(e) => setForm((p) => ({ ...p, end: joinDT(e.target.value, splitDT(p.end).t) }))}
        />
      </div>
      <div className={styles.field}>
        <label className={styles.label}>End Time <span className={styles.required}>*</span></label>
        <input
          type="text" placeholder="HH:MM"
          className={styles.input}
          value={endParts.t}
          onFocus={(e) => e.target.select()}
          onChange={(e) => setForm((p) => ({ ...p, end: joinDT(splitDT(p.end).d, e.target.value) }))}
        />
      </div>
      <div className={styles.field}>
        <label className={styles.label}>Max Volunteers</label>
        <input
          type="number" min="1"
          className={styles.input}
          value={form.maxVolunteers}
          onChange={(e) => setForm((p) => ({ ...p, maxVolunteers: e.target.value }))}
        />
      </div>
      <div className={styles.field}>
        <label className={styles.label}>Staff Contact</label>
        <select
          className={styles.select}
          value={form.staffContactId}
          onChange={(e) => setForm((p) => ({ ...p, staffContactId: e.target.value }))}
        >
          <option value="">— none —</option>
          {staff.map((s) => (
            <option key={s.id} value={s.id}>
              {s.firstName} {s.lastName}{s.position ? ` (${s.position})` : ""}
            </option>
          ))}
        </select>
      </div>
    </div>
  );
}

export default function AdminEventDetailPage() {
  const router = useRouter();
  const params = useParams();
  const eventId = params?.id;

  const [gql, setGql]           = useState(null);
  const [userName, setUserName] = useState("");

  /* Page data */
  const [event, setEvent]       = useState(null);
  const [opps, setOpps]         = useState([]);
  const [jobTypes, setJobTypes] = useState([]);
  const [svcTypes, setSvcTypes] = useState([]);
  const [venues, setVenues]     = useState([]);
  const [staff, setStaff]       = useState([]);

  /* UI state */
  const [loading, setLoading]     = useState(true);
  const [pageError, setPageError] = useState("");
  const [actionMsg, setActionMsg] = useState(null); // { type, text }
  const [busy, setBusy]           = useState(false);

  /* --- Edit event form --- */
  const [editingEvent, setEditingEvent] = useState(false);
  const [evForm, setEvForm] = useState({
    name: "", description: "", eventType: "IN_PERSON",
    venueId: "", serviceTypes: [],
  });

  /* --- Event dates (within event section) --- */
  const [editingDateId, setEditingDateId]     = useState(null);
  const [editDateForm, setEditDateForm]       = useState({ start: "", end: "" });
  const [addingDate, setAddingDate]           = useState(false);
  const [addDateForm, setAddDateForm]         = useState({ start: "", end: "" });

  /* --- Add / Edit opportunity --- */
  const [addingOpp, setAddingOpp]     = useState(false);
  const [oppForm, setOppForm] = useState({
    jobId: "", isVirtual: false, preEventInstructions: "",
    shiftStart: "", shiftEnd: "", shiftMaxVols: "", shiftStaffId: "",
  });
  const [editingOppId, setEditingOppId]   = useState(null);
  const [editOppForm, setEditOppForm]     = useState({
    jobId: "", isVirtual: false, preEventInstructions: "",
  });

  /* --- Add / Edit shift --- */
  const [addingShiftOppId, setAddingShiftOppId] = useState(null);
  const [addShiftForm, setAddShiftForm]         = useState(EMPTY_SHIFT_FORM);
  const [editingShiftId, setEditingShiftId]     = useState(null);
  const [editShiftForm, setEditShiftForm]       = useState(EMPTY_SHIFT_FORM);

  /* ---- Auth + data load ---- */
  const loadPage = useCallback((bound, eid) => {
    setLoading(true);
    bound(ADMIN_EVENT_DETAIL, { eventId: eid })
      .then((res) => {
        if (res.errors) { setPageError(res.errors[0]?.message ?? "Error loading data."); return; }
        setEvent(res.data?.eventById ?? null);
        setOpps(res.data?.opportunitiesForEvent ?? []);
        setJobTypes(res.data?.lookupValues?.jobTypes ?? []);
        setSvcTypes(res.data?.lookupValues?.serviceTypes ?? []);
        setVenues(res.data?.venues ?? []);
        setStaff(res.data?.staff ?? []);
      })
      .catch(() => setPageError("Unable to reach the server."))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    const t = getAuthToken();
    if (!t) { router.replace("/login"); return; }
    const role = getAuthRole();
    if (role !== "ADMINISTRATOR") { router.replace("/events"); return; }
    const bound = (q, v) => adminGql(q, v, t);
    setGql(() => bound);
    setUserName(getAuthName() ?? "");
    loadPage(bound, eventId);
  }, [router, eventId, loadPage]);

  /* ---- Helpers ---- */
  const tz           = eventIanaZone(event);
  const staffMap     = Object.fromEntries(staff.map((s) => [s.id, `${s.firstName} ${s.lastName}`]));
  const jobMap       = Object.fromEntries(jobTypes.map((j) => [String(j.id), j.name]));

  const showMsg = (type, text) => {
    setActionMsg({ type, text });
    window.scrollTo({ top: 0, behavior: "smooth" });
  };

  const mutate = async (mutation, variables, successMsg, onSuccess) => {
    setBusy(true);
    setActionMsg(null);
    try {
      const res = await gql(mutation, variables);
      const key = Object.keys(res.data ?? {})[0];
      const result = res.data?.[key];
      if (res.errors || !result?.success) {
        showMsg("error", result?.message ?? res.errors?.[0]?.message ?? "Operation failed.");
        return null;
      }
      showMsg("success", successMsg);
      if (onSuccess) onSuccess(result);
      loadPage(gql, eventId);
      return result;
    } catch {
      showMsg("error", "Unable to reach the server.");
      return null;
    } finally {
      setBusy(false);
    }
  };

  /* =========================================================
     Event handlers
     ========================================================= */

  /* --- Edit Event --- */
  const openEditEvent = () => {
    const svcIds = svcTypes
      .filter((st) => (event.serviceTypes ?? []).includes(st.name))
      .map((st) => st.id);
    setEvForm({
      name: event.name,
      description: event.description ?? "",
      eventType: event.eventType,
      venueId: event.venue?.id ?? "",
      serviceTypes: svcIds,
    });
    setEditingEvent(true);
    setAddingOpp(false);
  };

  const handleSaveEvent = async () => {
    if (!evForm.name.trim()) { showMsg("error", "Event name is required."); return; }
    await mutate(
      UPDATE_EVENT,
      { event: {
        id: eventId,
        name: evForm.name.trim(),
        description: evForm.description.trim() || null,
        eventType: evForm.eventType,
        venueId: evForm.venueId || null,
        serviceTypes: evForm.serviceTypes.map(Number),
      }},
      "Event updated.",
      () => setEditingEvent(false),
    );
  };

  const handleDeleteEvent = async () => {
    if (!window.confirm(`Delete "${event?.name}"? This cannot be undone.`)) return;
    await mutate(DELETE_EVENT, { eventId }, "Event deleted.", () => {
      router.replace("/admin/events");
    });
  };

  /* --- Event Dates --- */
  const openEditDate = (date) => {
    setEditingDateId(date.id);
    setEditDateForm({
      start: toDatetimeLocal(date.startDateTime, tz),
      end:   toDatetimeLocal(date.endDateTime,   tz),
    });
  };

  const handleSaveDate = async () => {
    await mutate(
      UPDATE_EVENT_DATE,
      { date: {
        id: editingDateId,
        startDateTime: toBackendDateTime(editDateForm.start),
        endDateTime:   toBackendDateTime(editDateForm.end),
        ianaZone: tz,
      }},
      "Date updated.",
      () => setEditingDateId(null),
    );
  };

  const handleDeleteDate = async (dateId) => {
    if (!window.confirm("Remove this event date?")) return;
    await mutate(DELETE_EVENT_DATE, { eventDateId: dateId }, "Date removed.");
  };

  const handleAddDate = async () => {
    await mutate(
      CREATE_EVENT_DATE,
      { newDate: {
        eventId,
        startDateTime: toBackendDateTime(addDateForm.start),
        endDateTime:   toBackendDateTime(addDateForm.end),
        ianaZone: tz,
      }},
      "Date added.",
      () => { setAddingDate(false); setAddDateForm({ start: "", end: "" }); },
    );
  };

  /* --- Opportunities --- */
  const openAddOpp = () => {
    setOppForm({
      jobId: jobTypes[0]?.id ? String(jobTypes[0].id) : "",
      isVirtual: event?.eventType === "VIRTUAL",
      preEventInstructions: "",
      shiftStart: "", shiftEnd: "",
      shiftMaxVols: "", shiftStaffId: "",
    });
    setAddingOpp(true);
    setEditingEvent(false);
  };

  const handleSaveOpp = async () => {
    if (!oppForm.jobId) { showMsg("error", "Please select a job type."); return; }
    if (!oppForm.shiftStart || !oppForm.shiftEnd) {
      showMsg("error", "First shift start and end are required.");
      return;
    }
    await mutate(
      CREATE_OPP,
      { newOpp: {
        eventId,
        jobId: parseInt(oppForm.jobId, 10),
        isVirtual: oppForm.isVirtual,
        preEventInstructions: oppForm.preEventInstructions.trim() || null,
        shifts: [{
          startDateTime: toBackendDateTime(oppForm.shiftStart),
          endDateTime:   toBackendDateTime(oppForm.shiftEnd),
          ianaZone: tz,
          maxVolunteers: oppForm.shiftMaxVols ? parseInt(oppForm.shiftMaxVols, 10) : null,
          staffContactId: oppForm.shiftStaffId || null,
        }],
      }},
      "Opportunity added.",
      () => setAddingOpp(false),
    );
  };

  const openEditOpp = (opp) => {
    setEditingOppId(opp.id);
    setEditOppForm({
      jobId: String(opp.jobId),
      isVirtual: opp.isVirtual,
      preEventInstructions: opp.preEventInstructions ?? "",
    });
    setEditingShiftId(null);
    setAddingShiftOppId(null);
  };

  const handleSaveEditOpp = async () => {
    await mutate(
      UPDATE_OPP,
      { opp: {
        id: editingOppId,
        jobId: parseInt(editOppForm.jobId, 10),
        isVirtual: editOppForm.isVirtual,
        preEventInstructions: editOppForm.preEventInstructions.trim() || null,
      }},
      "Opportunity updated.",
      () => setEditingOppId(null),
    );
  };

  const handleDeleteOpp = async (opp) => {
    if (!window.confirm(`Delete the "${jobMap[String(opp.jobId)] ?? "this"}" opportunity and all its shifts?`)) return;
    await mutate(DELETE_OPP, { oppId: opp.id }, "Opportunity deleted.");
  };

  /* --- Shifts --- */
  const openAddShift = (oppId) => {
    setAddingShiftOppId(oppId);
    setAddShiftForm({ ...EMPTY_SHIFT_FORM, ianaZone: tz });
    setEditingShiftId(null);
  };

  const handleSaveAddShift = async () => {
    if (!addShiftForm.start || !addShiftForm.end) {
      showMsg("error", "Shift start and end are required.");
      return;
    }
    await mutate(
      CREATE_SHIFT,
      { newShift: {
        opportunityId: addingShiftOppId,
        startDateTime: toBackendDateTime(addShiftForm.start),
        endDateTime:   toBackendDateTime(addShiftForm.end),
        ianaZone: tz,
        maxVolunteers: addShiftForm.maxVolunteers ? parseInt(addShiftForm.maxVolunteers, 10) : null,
        staffContactId: addShiftForm.staffContactId || null,
      }},
      "Shift added.",
      () => setAddingShiftOppId(null),
    );
  };

  const openEditShift = (shift) => {
    setEditingShiftId(shift.id);
    setEditShiftForm({
      start:          toDatetimeLocal(shift.startDateTime, tz),
      end:            toDatetimeLocal(shift.endDateTime,   tz),
      ianaZone:       tz,
      maxVolunteers:  shift.maxVolunteers != null ? String(shift.maxVolunteers) : "",
      staffContactId: shift.staffContactId ?? "",
    });
    setAddingShiftOppId(null);
  };

  const handleSaveEditShift = async () => {
    await mutate(
      UPDATE_SHIFT,
      { shift: {
        id: editingShiftId,
        startDateTime: toBackendDateTime(editShiftForm.start),
        endDateTime:   toBackendDateTime(editShiftForm.end),
        ianaZone: tz,
        maxVolunteers: editShiftForm.maxVolunteers ? parseInt(editShiftForm.maxVolunteers, 10) : null,
        staffContactId: editShiftForm.staffContactId || null,
      }},
      "Shift updated.",
      () => setEditingShiftId(null),
    );
  };

  const handleDeleteShift = async (shift) => {
    if (!window.confirm("Delete this shift?")) return;
    await mutate(DELETE_SHIFT, { shiftId: shift.id }, "Shift deleted.");
  };

  const handleSignOut = () => { clearAuthToken(); router.replace("/login"); };

  /* =========================================================
     Loading / error state
     ========================================================= */

  if (loading) {
    return (
      <div className={styles.page}>
        <div className={styles.topBar}>
          <a href="/admin/events" className={styles.backLink}>← Back to Manage Events</a>
        </div>
        <div className={styles.content}>
          <div className={styles.stateBox}>
            <div className={styles.spinner} />
            <p>Loading event…</p>
          </div>
        </div>
      </div>
    );
  }

  if (!event) {
    return (
      <div className={styles.page}>
        <div className={styles.topBar}>
          <a href="/admin/events" className={styles.backLink}>← Back to Manage Events</a>
        </div>
        <div className={styles.content}>
          <div className={styles.stateBox}>
            <div className={styles.stateTitle}>Event not found</div>
          </div>
        </div>
      </div>
    );
  }

  /* =========================================================
     Main render
     ========================================================= */

  return (
    <div className={styles.page}>
      {/* Top bar */}
      <div className={styles.topBar}>
        <a href="/admin/events" className={styles.backLink}>← Back to Manage Events</a>
        <UserMenu name={userName} isAdmin={true} onSignOut={handleSignOut} />
      </div>

      <div className={styles.content}>
        {/* Banners */}
        {actionMsg?.type === "success" && (
          <div className={styles.successBanner}>{actionMsg.text}</div>
        )}
        {actionMsg?.type === "error" && (
          <div className={styles.errorBanner}>{actionMsg.text}</div>
        )}
        {pageError && <div className={styles.errorBanner}>{pageError}</div>}

        {/* ---- Event details section ---- */}
        <div className={styles.section}>
          <div className={styles.sectionHeader}>
            <div className={styles.sectionTitle}>Event Details</div>
            <div className={styles.oppActions}>
              <button
                className={`${styles.iconBtn} ${styles.iconBtnEdit}`}
                title="Edit event"
                onClick={openEditEvent}
              >✏</button>
              <button
                className={`${styles.iconBtn} ${styles.iconBtnDelete}`}
                title="Delete event"
                onClick={handleDeleteEvent}
                disabled={busy}
              >🗑</button>
            </div>
          </div>

          {/* Read-only display */}
          {!editingEvent && (
            <div className={styles.metaGrid}>
              <span className={styles.metaLabel}>Name</span>
              <span className={styles.metaValue}>{event.name}</span>

              {event.description && (
                <>
                  <span className={styles.metaLabel}>Description</span>
                  <span className={styles.metaValue}>{event.description}</span>
                </>
              )}

              <span className={styles.metaLabel}>Format</span>
              <span className={styles.metaValue}>{FORMAT_LABEL[event.eventType] ?? event.eventType}</span>

              <span className={styles.metaLabel}>Venue</span>
              <span className={styles.metaValue}>
                {event.venue ? `${event.venue.name ?? event.venue.city}, ${event.venue.state}` : "— none (virtual)"}
              </span>

              {(event.serviceTypes?.length > 0) && (
                <>
                  <span className={styles.metaLabel}>Service Types</span>
                  <span className={styles.metaValue}>{event.serviceTypes.join(", ")}</span>
                </>
              )}

              {/* Event Dates */}
              <span className={styles.metaLabel}>Event Dates</span>
              <div className={styles.metaValue}>
                {event.eventDates?.length === 0 && <em className={styles.emptyMsg}>No dates set</em>}
                {event.eventDates?.map((date) => (
                  <div key={date.id} className={styles.shiftRow}>
                    <div>
                      <div className={styles.shiftInfo}>{formatDisplay(date.startDateTime, tz)}</div>
                      <div className={styles.shiftSub}>to {formatDisplay(date.endDateTime, tz)}</div>
                    </div>
                    <div className={styles.oppActions}>
                      <button
                        className={`${styles.iconBtn} ${styles.iconBtnEdit}`}
                        title="Edit date"
                        onClick={() => openEditDate(date)}
                      >✏</button>
                      <button
                        className={`${styles.iconBtn} ${styles.iconBtnDelete}`}
                        title="Remove date"
                        onClick={() => handleDeleteDate(date.id)}
                        disabled={busy}
                      >🗑</button>
                    </div>
                  </div>
                ))}

                {/* Edit date inline form */}
                {editingDateId && (
                  <div className={styles.inlineForm}>
                    <div className={styles.inlineFormTitle}>Edit Date</div>
                    <div className={styles.grid2}>
                      <div className={styles.field}>
                        <label className={styles.label}>Start Date</label>
                        <input type="date" className={styles.input}
                          value={splitDT(editDateForm.start).d}
                          onChange={(e) => setEditDateForm((p) => ({ ...p, start: joinDT(e.target.value, splitDT(p.start).t) }))} />
                      </div>
                      <div className={styles.field}>
                        <label className={styles.label}>Start Time</label>
                        <input type="time" step="60" className={styles.input}
                          value={splitDT(editDateForm.start).t}
                          onFocus={(e) => e.target.select()}
                          onChange={(e) => setEditDateForm((p) => ({ ...p, start: joinDT(splitDT(p.start).d, e.target.value) }))} />
                      </div>
                      <div className={styles.field}>
                        <label className={styles.label}>End Date</label>
                        <input type="date" className={styles.input}
                          value={splitDT(editDateForm.end).d}
                          onChange={(e) => setEditDateForm((p) => ({ ...p, end: joinDT(e.target.value, splitDT(p.end).t) }))} />
                      </div>
                      <div className={styles.field}>
                        <label className={styles.label}>End Time</label>
                        <input type="time" step="60" className={styles.input}
                          value={splitDT(editDateForm.end).t}
                          onFocus={(e) => e.target.select()}
                          onChange={(e) => setEditDateForm((p) => ({ ...p, end: joinDT(splitDT(p.end).d, e.target.value) }))} />
                      </div>
                    </div>
                    <div className={styles.formActions}>
                      <button className={styles.btnPrimary} onClick={handleSaveDate} disabled={busy}>Save</button>
                      <button className={styles.btnSecondary} onClick={() => setEditingDateId(null)}>Cancel</button>
                    </div>
                  </div>
                )}

                {/* Add date */}
                {!addingDate && !editingDateId && (
                  <button className={styles.btnOutline} style={{ marginTop: "0.5rem" }} onClick={() => setAddingDate(true)}>
                    + Add Date
                  </button>
                )}
                {addingDate && (
                  <div className={styles.inlineForm}>
                    <div className={styles.inlineFormTitle}>Add Date</div>
                    <div className={styles.grid2}>
                      <div className={styles.field}>
                        <label className={styles.label}>Start Date</label>
                        <input type="date" className={styles.input}
                          value={splitDT(addDateForm.start).d}
                          onChange={(e) => setAddDateForm((p) => ({ ...p, start: joinDT(e.target.value, splitDT(p.start).t) }))} />
                      </div>
                      <div className={styles.field}>
                        <label className={styles.label}>Start Time</label>
                        <input type="time" step="60" className={styles.input}
                          value={splitDT(addDateForm.start).t}
                          onFocus={(e) => e.target.select()}
                          onChange={(e) => setAddDateForm((p) => ({ ...p, start: joinDT(splitDT(p.start).d, e.target.value) }))} />
                      </div>
                      <div className={styles.field}>
                        <label className={styles.label}>End Date</label>
                        <input type="date" className={styles.input}
                          value={splitDT(addDateForm.end).d}
                          onChange={(e) => setAddDateForm((p) => ({ ...p, end: joinDT(e.target.value, splitDT(p.end).t) }))} />
                      </div>
                      <div className={styles.field}>
                        <label className={styles.label}>End Time</label>
                        <input type="time" step="60" className={styles.input}
                          value={splitDT(addDateForm.end).t}
                          onFocus={(e) => e.target.select()}
                          onChange={(e) => setAddDateForm((p) => ({ ...p, end: joinDT(splitDT(p.end).d, e.target.value) }))} />
                      </div>
                    </div>
                    <div className={styles.formActions}>
                      <button className={styles.btnPrimary} onClick={handleAddDate} disabled={busy}>Add Date</button>
                      <button className={styles.btnSecondary} onClick={() => setAddingDate(false)}>Cancel</button>
                    </div>
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Edit event inline form */}
          {editingEvent && (
            <div className={styles.inlineForm}>
              <div className={styles.field}>
                <label className={styles.label}>Event Name <span className={styles.required}>*</span></label>
                <input className={styles.input} value={evForm.name}
                  onChange={(e) => setEvForm((p) => ({ ...p, name: e.target.value }))} />
              </div>
              <div className={styles.field}>
                <label className={styles.label}>Description</label>
                <textarea className={styles.textarea} value={evForm.description}
                  onChange={(e) => setEvForm((p) => ({ ...p, description: e.target.value }))} />
              </div>
              <div className={styles.field}>
                <label className={styles.label}>Format</label>
                <div className={styles.radioGroup}>
                  {["IN_PERSON", "VIRTUAL", "HYBRID"].map((fmt) => (
                    <label key={fmt} className={styles.radioLabel}>
                      <input type="radio" name="editFormat" value={fmt}
                        checked={evForm.eventType === fmt}
                        onChange={() => setEvForm((p) => ({ ...p, eventType: fmt, venueId: fmt === "VIRTUAL" ? "" : p.venueId }))} />
                      {FORMAT_LABEL[fmt]}
                    </label>
                  ))}
                </div>
              </div>
              {evForm.eventType !== "VIRTUAL" && (
                <div className={styles.field}>
                  <label className={styles.label}>Venue</label>
                  <select className={styles.select} value={evForm.venueId}
                    onChange={(e) => setEvForm((p) => ({ ...p, venueId: e.target.value }))}>
                    <option value="">— none —</option>
                    {venues.map((v) => (
                      <option key={v.id} value={v.id}>
                        {v.name ? `${v.name} — ` : ""}{v.city}, {v.state}
                      </option>
                    ))}
                  </select>
                </div>
              )}
              <div className={styles.field}>
                <label className={styles.label}>Service Types</label>
                <div className={styles.checkboxGroup}>
                  {svcTypes.map((st) => (
                    <label key={st.id} className={styles.checkboxLabel}>
                      <input type="checkbox"
                        checked={evForm.serviceTypes.includes(st.id)}
                        onChange={() => setEvForm((p) => ({
                          ...p,
                          serviceTypes: p.serviceTypes.includes(st.id)
                            ? p.serviceTypes.filter((id) => id !== st.id)
                            : [...p.serviceTypes, st.id],
                        }))} />
                      {st.name}
                    </label>
                  ))}
                </div>
              </div>
              <div className={styles.formActions}>
                <button className={styles.btnPrimary} onClick={handleSaveEvent} disabled={busy}>Save Changes</button>
                <button className={styles.btnSecondary} onClick={() => setEditingEvent(false)}>Cancel</button>
              </div>
            </div>
          )}
        </div>

        {/* ---- Opportunities section ---- */}
        <div className={styles.section}>
          <div className={styles.sectionHeader}>
            <div className={styles.sectionTitle}>Volunteer Opportunities</div>
            <button className={styles.btnOutline} onClick={openAddOpp}>+ Add Opportunity</button>
          </div>

          {/* Add opportunity form */}
          {addingOpp && (
            <div className={styles.inlineForm}>
              <div className={styles.inlineFormTitle}>New Opportunity</div>
              <div className={styles.grid2}>
                <div className={styles.field}>
                  <label className={styles.label}>Job Type <span className={styles.required}>*</span></label>
                  <select className={styles.select} value={oppForm.jobId}
                    onChange={(e) => setOppForm((p) => ({ ...p, jobId: e.target.value }))}>
                    <option value="">— select —</option>
                    {jobTypes.map((j) => (
                      <option key={j.id} value={j.id}>{j.name}</option>
                    ))}
                  </select>
                </div>
                <div className={styles.field}>
                  <label className={styles.checkboxLabel} style={{ marginTop: "1.75rem" }}>
                    <input type="checkbox" checked={oppForm.isVirtual}
                      onChange={(e) => setOppForm((p) => ({ ...p, isVirtual: e.target.checked }))} />
                    Virtual opportunity
                  </label>
                </div>
              </div>
              <div className={styles.field}>
                <label className={styles.label}>Pre-Event Instructions</label>
                <textarea className={styles.textarea} value={oppForm.preEventInstructions}
                  onChange={(e) => setOppForm((p) => ({ ...p, preEventInstructions: e.target.value }))} />
              </div>

              <div className={styles.shiftDivider}>First Shift (required)</div>
              <ShiftFormFields
                staff={staff}
                form={{ start: oppForm.shiftStart, end: oppForm.shiftEnd, maxVolunteers: oppForm.shiftMaxVols, staffContactId: oppForm.shiftStaffId }}
                setForm={(updater) => setOppForm((p) => {
                  const updated = typeof updater === "function"
                    ? updater({ start: p.shiftStart, end: p.shiftEnd, maxVolunteers: p.shiftMaxVols, staffContactId: p.shiftStaffId })
                    : updater;
                  return { ...p, shiftStart: updated.start, shiftEnd: updated.end, shiftMaxVols: updated.maxVolunteers, shiftStaffId: updated.staffContactId };
                })}
              />

              <div className={styles.formActions}>
                <button className={styles.btnPrimary} onClick={handleSaveOpp} disabled={busy}>Add Opportunity</button>
                <button className={styles.btnSecondary} onClick={() => setAddingOpp(false)}>Cancel</button>
              </div>
            </div>
          )}

          {/* Opportunities list */}
          {opps.length === 0 && !addingOpp && (
            <div style={{ padding: "1rem 1.75rem" }}>
              <em className={styles.emptyMsg}>No opportunities yet. Add one above.</em>
            </div>
          )}

          {opps.map((opp) => {
            const jobName = jobMap[String(opp.jobId)] ?? `Job #${opp.jobId}`;
            const isEditingThisOpp = editingOppId === opp.id;

            return (
              <div key={opp.id} className={styles.oppCard}>
                {/* Opportunity header */}
                <div className={styles.oppHeader}>
                  <div>
                    <div className={styles.oppTitle}>{jobName}</div>
                    <div className={styles.oppMeta}>
                      {opp.isVirtual ? "Virtual" : "In Person"}
                      {opp.preEventInstructions && ` · ${opp.preEventInstructions}`}
                    </div>
                  </div>
                  <div className={styles.oppActions}>
                    <button
                      className={`${styles.iconBtn} ${styles.iconBtnEdit}`}
                      title="Edit opportunity"
                      onClick={() => isEditingThisOpp ? setEditingOppId(null) : openEditOpp(opp)}
                    >✏</button>
                    <button
                      className={`${styles.iconBtn} ${styles.iconBtnDelete}`}
                      title="Delete opportunity"
                      onClick={() => handleDeleteOpp(opp)}
                      disabled={busy}
                    >🗑</button>
                  </div>
                </div>

                {/* Edit opportunity inline form */}
                {isEditingThisOpp && (
                  <div className={styles.inlineForm}>
                    <div className={styles.grid2}>
                      <div className={styles.field}>
                        <label className={styles.label}>Job Type</label>
                        <select className={styles.select} value={editOppForm.jobId}
                          onChange={(e) => setEditOppForm((p) => ({ ...p, jobId: e.target.value }))}>
                          {jobTypes.map((j) => (
                            <option key={j.id} value={j.id}>{j.name}</option>
                          ))}
                        </select>
                      </div>
                      <div className={styles.field}>
                        <label className={styles.checkboxLabel} style={{ marginTop: "1.75rem" }}>
                          <input type="checkbox" checked={editOppForm.isVirtual}
                            onChange={(e) => setEditOppForm((p) => ({ ...p, isVirtual: e.target.checked }))} />
                          Virtual opportunity
                        </label>
                      </div>
                    </div>
                    <div className={styles.field}>
                      <label className={styles.label}>Pre-Event Instructions</label>
                      <textarea className={styles.textarea} value={editOppForm.preEventInstructions}
                        onChange={(e) => setEditOppForm((p) => ({ ...p, preEventInstructions: e.target.value }))} />
                    </div>
                    <div className={styles.formActions}>
                      <button className={styles.btnPrimary} onClick={handleSaveEditOpp} disabled={busy}>Save</button>
                      <button className={styles.btnSecondary} onClick={() => setEditingOppId(null)}>Cancel</button>
                    </div>
                  </div>
                )}

                {/* Shifts list */}
                <div className={styles.shiftList}>
                  {opp.shifts.length === 0 && (
                    <div className={styles.emptyMsg}>No shifts — add one below.</div>
                  )}

                  {opp.shifts.map((shift) => {
                    const isEditingThisShift = editingShiftId === shift.id;
                    return (
                      <div key={shift.id}>
                        <div className={styles.shiftRow}>
                          <div>
                            <div className={styles.shiftInfo}>
                              {formatDisplay(shift.startDateTime, tz)}
                            </div>
                            <div className={styles.shiftSub}>
                              to {formatDisplay(shift.endDateTime, tz)}
                              {shift.maxVolunteers != null && ` · Max ${shift.maxVolunteers}`}
                              {shift.staffContactId && staffMap[shift.staffContactId] && ` · ${staffMap[shift.staffContactId]}`}
                            </div>
                          </div>
                          <div className={styles.oppActions}>
                            <button
                              className={`${styles.iconBtn} ${styles.iconBtnEdit}`}
                              title="Edit shift"
                              onClick={() => isEditingThisShift ? setEditingShiftId(null) : openEditShift(shift)}
                            >✏</button>
                            <button
                              className={`${styles.iconBtn} ${styles.iconBtnDelete}`}
                              title="Delete shift"
                              onClick={() => handleDeleteShift(shift)}
                              disabled={busy}
                            >🗑</button>
                          </div>
                        </div>

                        {/* Edit shift inline form */}
                        {isEditingThisShift && (
                          <div className={styles.inlineForm}>
                            <div className={styles.inlineFormTitle}>Edit Shift</div>
                            <ShiftFormFields staff={staff} form={editShiftForm} setForm={setEditShiftForm} />
                            <div className={styles.formActions}>
                              <button className={styles.btnPrimary} onClick={handleSaveEditShift} disabled={busy}>Save</button>
                              <button className={styles.btnSecondary} onClick={() => setEditingShiftId(null)}>Cancel</button>
                            </div>
                          </div>
                        )}
                      </div>
                    );
                  })}

                  {/* Add shift */}
                  {addingShiftOppId === opp.id ? (
                    <div className={styles.inlineForm}>
                      <div className={styles.inlineFormTitle}>Add Shift</div>
                      <ShiftFormFields staff={staff} form={addShiftForm} setForm={setAddShiftForm} />
                      <div className={styles.formActions}>
                        <button className={styles.btnPrimary} onClick={handleSaveAddShift} disabled={busy}>Add Shift</button>
                        <button className={styles.btnSecondary} onClick={() => setAddingShiftOppId(null)}>Cancel</button>
                      </div>
                    </div>
                  ) : (
                    <button
                      className={styles.btnOutline}
                      style={{ marginTop: "0.5rem", fontSize: "0.8rem" }}
                      onClick={() => openAddShift(opp.id)}
                    >
                      + Add Shift
                    </button>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
