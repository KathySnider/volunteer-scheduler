"use client";

import { useEffect, useState, useRef, useMemo, useCallback } from "react";
import { useRouter } from "next/navigation";
import {
  isAuthenticated,
  getAuthRole,
  getAuthName,
  signOut,
  adminGql,
} from "../../../lib/api";
import AdminTopBar from "../../../components/AdminTopBar";
import FeedbackButton from "../../../components/FeedbackButton";
import styles from "./add-event.module.css";

/* ----- Constants ----- */

const RECURRENCE_DEFAULTS = {
  DAILY: "365", WEEKLY: "52", BIWEEKLY: "26", MONTHLY: "12", YEARLY: "",
};

const US_TIMEZONES = [
  { value: "America/New_York",    label: "Eastern (ET)" },
  { value: "America/Chicago",     label: "Central (CT)" },
  { value: "America/Denver",      label: "Mountain (MT)" },
  { value: "America/Los_Angeles", label: "Pacific (PT)" },
  { value: "America/Anchorage",   label: "Alaska (AKT)" },
  { value: "Pacific/Honolulu",    label: "Hawaii (HT)" },
];

function timezoneOptions(browserZone) {
  const has = US_TIMEZONES.some((z) => z.value === browserZone);
  if (has) return US_TIMEZONES;
  return [{ value: browserZone, label: `Local (${browserZone})` }, ...US_TIMEZONES];
}

/* ----- GraphQL operations ----- */

const VENUES_AND_LOOKUPS = `
  query {
    venues {
      id
      name
      address
      city
      state
      zipCode
    }
    lookupValues {
      fundingEntities { id name }
      serviceTypes { id name }
    }
  }
`;

const CREATE_VENUE = `
  mutation CreateVenue($newVenue: NewVenueInput!) {
    createVenue(newVenue: $newVenue) {
      success
      message
      id
    }
  }
`;

const CREATE_EVENT = `
  mutation CreateEvent($newEvent: NewEventInput!) {
    createEvent(newEvent: $newEvent) {
      success
      message
      id
    }
  }
`

// Used to resolve the group UUID returned by createEvent (recurring) into the
// first occurrence's numeric event ID, so the "Add Opportunities" link works.
const FIRST_IN_GROUP = `
  query {
    filteredEvents {
      id
      recurrenceId
      recurrenceOrder
    }
  }
`

/** Returns true when s is a UUID (36 chars, hyphens at positions 8/13/18/23). */
function isUUID(s) {
  return typeof s === "string" && s.length === 36 &&
    s[8] === "-" && s[13] === "-" && s[18] === "-" && s[23] === "-";
};

/* ----- Helpers ----- */

function normalizeTime(t) {
  if (!t) return "00:00";
  if (t.includes(":")) {
    const [h, m] = t.split(":");
    const hh = h.replace(/\D/g, "").padStart(2, "0").slice(-2);
    const mm = m.replace(/\D/g, "").padStart(2, "0").slice(0, 2);
    return `${hh}:${mm}`;
  }
  const digits = t.replace(/\D/g, "").padStart(4, "0").slice(-4);
  return `${digits.slice(0, 2)}:${digits.slice(2, 4)}`;
}

/** Convert a "YYYY-MM-DDTHH:MM" value to backend format ("2026-05-10 09:00:00"). */
function toBackendDateTime(dtLocal) {
  if (!dtLocal) return "";
  const [d, t] = dtLocal.split("T");
  return `${d} ${normalizeTime(t)}:00`;
}

function splitDT(dtLocal) {
  if (!dtLocal) return { d: "", t: "" };
  const [d, t] = dtLocal.split("T");
  return { d: d ?? "", t: t ?? "" };
}

function joinDT(d, t) {
  if (!d) return "";
  return `${d}T${t || "00:00"}`;
}

/**
 * When an event's start datetime changes (date or time), shift the end
 * datetime by the same delta to preserve the event's duration.
 * Accepts and returns "YYYY-MM-DDTHH:MM" strings.
 * Falls back to oldEnd unchanged if inputs are missing or the existing
 * interval is already negative (already broken — don't make it worse).
 */
function eventEndDT(oldStart, oldEnd, newStart) {
  if (!newStart || !oldStart || !oldEnd) return oldEnd ?? "";
  const oldMs = new Date(oldStart).getTime();
  const endMs = new Date(oldEnd).getTime();
  const newMs = new Date(newStart).getTime();
  if (isNaN(oldMs) || isNaN(endMs) || isNaN(newMs)) return oldEnd;
  const gap = endMs - oldMs;
  if (gap < 0) return oldEnd;
  const r   = new Date(newMs + gap);
  const pad = (n) => String(n).padStart(2, "0");
  return `${r.getFullYear()}-${pad(r.getMonth() + 1)}-${pad(r.getDate())}T${pad(r.getHours())}:${pad(r.getMinutes())}`;
}

function to12Hour(hhmm) {
  if (!hhmm || !hhmm.includes(":")) return { display: hhmm, period: "AM" };
  let [h, m] = hhmm.split(":");
  h = parseInt(h, 10) || 0;
  const period = h >= 12 ? "PM" : "AM";
  const h12    = h % 12 || 12;
  return { display: `${h12}:${m}`, period };
}

function to24Hour(display, period) {
  const norm = normalizeTime(display);
  let [h, m] = norm.split(":").map(Number);
  if (period === "AM") {
    if (h === 12) h = 0;
  } else {
    if (h !== 12) h += 12;
    if (h >= 24) h = 12;
  }
  return `${String(h).padStart(2, "0")}:${String(m).padStart(2, "0")}`;
}

function venueDisplayName(v) {
  return v.name ? `${v.name} — ${v.city}, ${v.state}` : `${v.address}, ${v.city}, ${v.state}`;
}

/* ----- TimeInput ----- */
/**
 * Free-form time input that lets the user type naturally.
 * Stores raw text locally while focused; on blur it normalizes
 * the value (via to24Hour) and commits it to the parent.
 * Syncs display when value24 changes from outside (e.g. AM/PM toggle).
 */
function TimeInput({ value24, period, onCommit, className }) {
  const [raw, setRaw] = useState(() => to12Hour(value24).display);
  const focusedRef = useRef(false);

  useEffect(() => {
    if (!focusedRef.current) {
      setRaw(to12Hour(value24).display);
    }
  }, [value24]);

  return (
    <input
      type="text"
      placeholder="h:MM"
      className={className}
      value={raw}
      onFocus={(e) => {
        focusedRef.current = true;
        setRaw(to12Hour(value24).display);
        e.target.select();
      }}
      onChange={(e) => setRaw(e.target.value)}
      onBlur={() => {
        focusedRef.current = false;
        const converted = to24Hour(raw, period);
        onCommit(converted);
        setRaw(to12Hour(converted).display);
      }}
    />
  );
}

/* ----- VenueSelector sub-component ----- */

function VenueSelector({ venues, selectedVenue, onSelect, onClear, gql }) {
  const [search, setSearch] = useState("");
  const [open, setOpen] = useState(false);
  const [showNewForm, setShowNewForm] = useState(false);
  const [newVenue, setNewVenue] = useState({
    name: "", address: "", city: "", state: "WA", zipCode: "",
  });
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState("");
  const wrapperRef = useRef(null);

  // Close dropdown on outside click
  useEffect(() => {
    if (!open) return;
    const handler = (e) => {
      if (wrapperRef.current && !wrapperRef.current.contains(e.target)) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [open]);

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return venues.slice(0, 8);
    return venues
      .filter((v) =>
        (v.name && v.name.toLowerCase().includes(q)) ||
        v.city.toLowerCase().includes(q) ||
        v.address.toLowerCase().includes(q)
      )
      .slice(0, 8);
  }, [venues, search]);

  const handleSelect = (v) => {
    onSelect(v);
    setSearch("");
    setOpen(false);
    setShowNewForm(false);
  };

  const handleAddNew = () => {
    setNewVenue((prev) => ({ ...prev, name: search }));
    setOpen(false);
    setShowNewForm(true);
  };

  const handleCreateVenue = async () => {
    setCreateError("");
    if (!newVenue.address || !newVenue.city || !newVenue.state) {
      setCreateError("Address, city, and state are required.");
      return;
    }
    setCreating(true);
    try {
      const res = await gql(CREATE_VENUE, {
        newVenue: {
          name:    newVenue.name    || null,
          address: newVenue.address,
          city:    newVenue.city,
          state:   newVenue.state,
          zipCode: newVenue.zipCode || null,
        },
      });
      const result = res.data?.createVenue;
      if (!result?.success || !result?.id) {
        setCreateError(result?.message ?? "Failed to create venue.");
        return;
      }
      // Auto-select the newly created venue
      onSelect({
        id:      result.id,
        name:    newVenue.name,
        address: newVenue.address,
        city:    newVenue.city,
        state:   newVenue.state,
      });
      setShowNewForm(false);
      setSearch("");
    } catch {
      setCreateError("Unable to reach the server.");
    } finally {
      setCreating(false);
    }
  };

  // --- If a venue is already selected, show it as a chip ---
  if (selectedVenue) {
    return (
      <div className={styles.venueSelected}>
        <div>
          <div className={styles.venueSelectedName}>
            {selectedVenue.name || selectedVenue.address}
          </div>
          <div className={styles.venueSelectedSub}>
            {selectedVenue.city}, {selectedVenue.state}
          </div>
        </div>
        <button className={styles.venueClearBtn} onClick={onClear}>
          Change
        </button>
      </div>
    );
  }

  return (
    <div ref={wrapperRef} className={styles.venueWrapper}>
      {!showNewForm && (
        <div className={styles.venueInputRow}>
          <input
            className={`${styles.input} ${styles.venueInput}`}
            placeholder="Search venues by name or city…"
            value={search}
            onChange={(e) => { setSearch(e.target.value); setOpen(true); }}
            onFocus={() => setOpen(true)}
            autoComplete="off"
          />
        </div>
      )}

      {open && !showNewForm && (
        <div className={styles.venueDropdown}>
          {filtered.length === 0 && (
            <div className={styles.venueOption} style={{ color: "var(--color-text-muted)" }}>
              No matches
            </div>
          )}
          {filtered.map((v) => (
            <div key={v.id} className={styles.venueOption} onClick={() => handleSelect(v)}>
              <div>{v.name || v.address}</div>
              <div className={styles.venueOptionSub}>{v.city}, {v.state}</div>
            </div>
          ))}
          <div className={styles.venueAddOption} onClick={handleAddNew}>
            ＋ Add {search ? `"${search}" as` : ""} new venue
          </div>
        </div>
      )}

      {showNewForm && (
        <div className={styles.newVenueForm}>
          <div className={styles.newVenueTitle}>Add New Venue</div>

          <div className={styles.grid2}>
            <div className={styles.field}>
              <label className={styles.label}>Venue Name</label>
              <input
                className={styles.input}
                placeholder="e.g. Cascade Park Library"
                value={newVenue.name}
                onChange={(e) => setNewVenue((p) => ({ ...p, name: e.target.value }))}
              />
            </div>
            <div className={styles.field}>
              <label className={styles.label}>
                Address <span className={styles.required}>*</span>
              </label>
              <input
                className={styles.input}
                value={newVenue.address}
                onChange={(e) => setNewVenue((p) => ({ ...p, address: e.target.value }))}
              />
            </div>
            <div className={styles.field}>
              <label className={styles.label}>
                City <span className={styles.required}>*</span>
              </label>
              <input
                className={styles.input}
                value={newVenue.city}
                onChange={(e) => setNewVenue((p) => ({ ...p, city: e.target.value }))}
              />
            </div>
            <div className={styles.field}>
              <label className={styles.label}>
                State <span className={styles.required}>*</span>
              </label>
              <input
                className={styles.input}
                value={newVenue.state}
                onChange={(e) => setNewVenue((p) => ({ ...p, state: e.target.value }))}
              />
            </div>
            <div className={styles.field}>
              <label className={styles.label}>Zip Code</label>
              <input
                className={styles.input}
                value={newVenue.zipCode}
                onChange={(e) => setNewVenue((p) => ({ ...p, zipCode: e.target.value }))}
              />
            </div>
          </div>

          {createError && <div className={styles.fieldError}>{createError}</div>}

          <div className={styles.newVenueActions}>
            <button
              className={styles.btnPrimary}
              onClick={handleCreateVenue}
              disabled={creating}
            >
              {creating ? "Creating…" : "Create Venue"}
            </button>
            <button
              className={styles.btnLink}
              onClick={() => { setShowNewForm(false); setCreateError(""); }}
            >
              Cancel
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

/* ----- Page ----- */

const EMPTY_DATE = () => ({
  id: Date.now(),
  startDate: "", startTime: "00:00",
  endDate:   "", endTime:   "00:00",
});

export default function AddEventPage() {
  const router = useRouter();
  const [gql, setGql] = useState(null);
  const [userName, setUserName] = useState("");
  const [isAdmin, setIsAdmin] = useState(false);
  const [feedbackOpen, setFeedbackOpen] = useState(false);
  const browserZone = useRef(Intl.DateTimeFormat().resolvedOptions().timeZone);

  // Lookup data
  const [venues, setVenues] = useState([]);
  const [fundingEntities, setFundingEntities] = useState([]);
  const [serviceTypes, setServiceTypes] = useState([]);
  const [loadError, setLoadError] = useState("");

  // Region field
  const [fundingEntityId, setFundingEntityId] = useState("");

  // Form state
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [eventType, setEventType] = useState("IN_PERSON");
  const [selectedVenue, setSelectedVenue] = useState(null);
  const [selectedServiceTypes, setSelectedServiceTypes] = useState([]);
  const [ianaZone, setIanaZone] = useState(browserZone.current);
  const [recurring,         setRecurring]         = useState(false);
  const [recurrencePattern, setRecurrencePattern] = useState("WEEKLY");
  const [recurrenceMax,     setRecurrenceMax]     = useState(RECURRENCE_DEFAULTS.WEEKLY);
  const [recurrenceOrdinal, setRecurrenceOrdinal] = useState("FIRST");
  const [eventDates, setEventDates] = useState([EMPTY_DATE()]);

  // Validation errors
  const [errors, setErrors] = useState({});

  // Submit state
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState("");
  const [createdEvent, setCreatedEvent] = useState(null); // { id, name }

  /* Auth check + load data */
  useEffect(() => {
    if (!isAuthenticated()) { router.replace("/login"); return; }
    const role = getAuthRole();
    if (role !== "ADMINISTRATOR") { router.replace("/events"); return; }

    const boundGql = adminGql;
    setGql(() => boundGql);
    setUserName(getAuthName() ?? "");
    setIsAdmin(true);

    boundGql(VENUES_AND_LOOKUPS, null)
      .then((res) => {
        // Use whatever data came back even if one field errored.
        if (res.data?.venues)                             setVenues(res.data.venues);
        if (res.data?.lookupValues?.fundingEntities) {
          setFundingEntities(res.data.lookupValues.fundingEntities);
          if (res.data.lookupValues.fundingEntities.length > 0 && !fundingEntityId) {
            setFundingEntityId(String(res.data.lookupValues.fundingEntities[0].id));
          }
        }
        if (res.data?.lookupValues?.serviceTypes)         setServiceTypes(res.data.lookupValues.serviceTypes);
        if (res.errors) setLoadError(res.errors[0]?.message ?? "Error loading some data.");
      })
      .catch(() => setLoadError("Unable to reach the server."));
  }, [router]);

  /* When event type changes to VIRTUAL, clear venue */
  useEffect(() => {
    if (eventType === "VIRTUAL") {
      setSelectedVenue(null);
      setIanaZone(browserZone.current);
    }
  }, [eventType]);

  const handleVenueSelect = useCallback((v) => {
    setSelectedVenue(v);
    setErrors((prev) => ({ ...prev, venue: undefined }));
  }, []);

  const handleVenueClear = () => {
    setSelectedVenue(null);
    setIanaZone(browserZone.current);
  };

  /* Service type toggles */
  const toggleServiceType = (id) => {
    setSelectedServiceTypes((prev) =>
      prev.includes(id) ? prev.filter((s) => s !== id) : [...prev, id]
    );
  };

  /* Event date management */
  const addDate = () => setEventDates((prev) => [...prev, EMPTY_DATE()]);

  const removeDate = (id) =>
    setEventDates((prev) => prev.filter((d) => d.id !== id));

  const updateDate = (id, field, value) =>
    setEventDates((prev) =>
      prev.map((d) => (d.id === id ? { ...d, [field]: value } : d))
    );

  /** Update startTime and shift endDate/endTime to preserve the duration. */
  const updateDateStartTime = (id, dateObj, t24) => {
    const newEnd = splitDT(eventEndDT(
      joinDT(dateObj.startDate, dateObj.startTime),
      joinDT(dateObj.endDate,   dateObj.endTime),
      joinDT(dateObj.startDate, t24),
    ));
    setEventDates((prev) =>
      prev.map((d) =>
        d.id === id
          ? { ...d, startTime: t24, endDate: newEnd.d || d.endDate, endTime: newEnd.t || t24 }
          : d
      )
    );
  };

  /* Validation */
  const validate = () => {
    const errs = {};
    if (!name.trim()) errs.name = "Event name is required.";
    if (!fundingEntityId) errs.fundingEntityId = "Region is required.";
    if (eventType !== "VIRTUAL" && !selectedVenue) {
      errs.venue = "Please select or add a venue.";
    }
    const dateErrs = eventDates.map((d) => {
      if (!d.startDate) return "Start date is required.";
      if (!d.endDate)   return "End date is required.";
      const startDT = `${d.startDate}T${d.startTime}`;
      const endDT   = `${d.endDate}T${d.endTime}`;
      if (startDT >= endDT) return "End must be after start.";
      return null;
    });
    if (dateErrs.some(Boolean)) errs.dates = dateErrs;
    if (recurring && recurrencePattern === "YEARLY" && !recurrenceMax.trim()) {
      errs.recurrenceMax = "Number of occurrences is required for yearly events.";
    }
    return errs;
  };

  /* Submit */
  const handleSubmit = async () => {
    const errs = validate();
    setErrors(errs);
    if (Object.keys(errs).length > 0) return;

    setSubmitting(true);
    setSubmitError("");

    const newEvent = {
      name:            name.trim(),
      description:     description.trim() || null,
      eventType,
      venueId:         selectedVenue?.id ?? null,
      fundingEntityId: parseInt(fundingEntityId, 10),
      serviceTypes:    selectedServiceTypes.map(Number),
      timezone:        ianaZone,
      eventDates:      eventDates.map((d) => ({
        startDateTime: `${d.startDate} ${normalizeTime(d.startTime)}:00`,
        endDateTime:   `${d.endDate} ${normalizeTime(d.endTime)}:00`,
      })),
      recurrence: recurring ? {
        pattern:        recurrencePattern,
        maxOccurrences: recurrenceMax.trim() ? parseInt(recurrenceMax, 10) : null,
        ...(recurrencePattern === "MONTHLY" ? { weekdayOrdinal: recurrenceOrdinal } : {}),
      } : undefined,
    };

    try {
      const res = await gql(CREATE_EVENT, { newEvent });
      const result = res.data?.createEvent;
      if (res.errors || !result?.success || !result?.id) {
        setSubmitError(result?.message ?? res.errors?.[0]?.message ?? "Failed to create event.");
        return;
      }

      // For recurring events, createEvent returns the recurrence group UUID.
      // Resolve it to the first occurrence's numeric ID for the "Add Opportunities" link.
      let targetId = result.id;
      if (isUUID(result.id)) {
        try {
          const listRes = await gql(FIRST_IN_GROUP, null);
          const events = listRes.data?.filteredEvents ?? [];
          const first = events
            .filter((e) => e.recurrenceId === result.id)
            .sort((a, b) => (a.recurrenceOrder ?? 0) - (b.recurrenceOrder ?? 0))[0];
          if (first?.id) targetId = first.id;
        } catch {
          // leave targetId as the UUID; the link will gracefully degrade
        }
      }

      setCreatedEvent({ id: targetId, name: name.trim() });
    } catch {
      setSubmitError("Unable to reach the server. Please try again.");
    } finally {
      setSubmitting(false);
    }
  };

  /* Reset form for another entry */
  const handleCreateAnother = () => {
    setName("");
    setDescription("");
    setEventType("IN_PERSON");
    setSelectedVenue(null);
    setSelectedServiceTypes([]);
    setFundingEntityId(fundingEntities[0] ? String(fundingEntities[0].id) : "");
    setIanaZone(browserZone.current);
    setRecurring(false);
    setRecurrencePattern("WEEKLY");
    setRecurrenceMax(RECURRENCE_DEFAULTS.WEEKLY);
    setRecurrenceOrdinal("FIRST");
    setEventDates([EMPTY_DATE()]);
    setErrors({});
    setSubmitError("");
    setCreatedEvent(null);
  };

  const handleSignOut = async () => { await signOut(); router.replace("/login"); };

  const tzOptions = timezoneOptions(browserZone.current);

  if (!gql) return null;

  return (
    <div className={styles.page}>
      {/* Top bar */}
      <AdminTopBar userName={userName} isAdmin={true} onSignOut={handleSignOut} onFeedbackOpen={() => setFeedbackOpen(true)} />

      <div className={styles.content}>
        <h1 className={styles.pageTitle}>Add Event</h1>

        {loadError && <div className={styles.submitError}>{loadError}</div>}

        {/* Success state */}
        {createdEvent && (
          <div className={styles.successCard}>
            <div className={styles.successIcon}>✓</div>
            <div className={styles.successTitle}>Event Created!</div>
            <div className={styles.successName}>{createdEvent.name}</div>
            <div className={styles.successActions}>
              <a href={`/admin/events/${createdEvent.id}`} className={styles.btnPrimary}
                style={{ textDecoration: "none", display: "inline-block" }}>
                Add Volunteer Opportunities
              </a>
              <button className={styles.btnSecondary} onClick={handleCreateAnother}>
                Create Another Event
              </button>
            </div>
          </div>
        )}

        {/* Form — hidden after success */}
        {!createdEvent && (
          <>
            {/* Event details */}
            <div className={styles.section}>
              <div className={styles.sectionTitle}>Event Details</div>

              <div className={styles.field}>
                <label className={styles.label}>
                  Event Name <span className={styles.required}>*</span>
                </label>
                <input
                  className={`${styles.input} ${errors.name ? styles.error : ""}`}
                  value={name}
                  onChange={(e) => { setName(e.target.value); setErrors((p) => ({ ...p, name: undefined })); }}
                  placeholder="e.g. Medicare Q&A Workshop"
                />
                {errors.name && <div className={styles.fieldError}>{errors.name}</div>}
              </div>

              <div className={styles.field}>
                <label className={styles.label}>Description</label>
                <textarea
                  className={styles.textarea}
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Briefly describe the event for volunteers…"
                />
              </div>

              <div className={styles.field}>
                <label className={styles.label}>
                  Region <span className={styles.required}>*</span>
                </label>
                <select
                  className={`${styles.select}${errors.fundingEntityId ? ` ${styles.error}` : ""}`}
                  value={fundingEntityId}
                  onChange={(e) => {
                    setFundingEntityId(e.target.value);
                    setErrors((p) => ({ ...p, fundingEntityId: undefined }));
                  }}
                >
                  <option value="">Select a region…</option>
                  {fundingEntities.map((fe) => (
                    <option key={fe.id} value={fe.id}>{fe.name}</option>
                  ))}
                </select>
                {errors.fundingEntityId && (
                  <div className={styles.fieldError}>{errors.fundingEntityId}</div>
                )}
              </div>

              <div className={styles.field}>
                <label className={styles.label}>Format</label>
                <div className={styles.radioGroup}>
                  {[
                    { value: "IN_PERSON", label: "In Person" },
                    { value: "VIRTUAL",   label: "Virtual" },
                    { value: "HYBRID",    label: "Hybrid" },
                  ].map((opt) => (
                    <label key={opt.value} className={styles.radioLabel}>
                      <input
                        type="radio"
                        name="eventType"
                        value={opt.value}
                        checked={eventType === opt.value}
                        onChange={() => setEventType(opt.value)}
                      />
                      {opt.label}
                    </label>
                  ))}
                </div>
              </div>
            </div>

            {/* Venue */}
            {eventType !== "VIRTUAL" && (
              <div className={styles.section}>
                <div className={styles.sectionTitle}>Venue</div>
                <div className={styles.field}>
                  <label className={styles.label}>
                    Venue <span className={styles.required}>*</span>
                  </label>
                  <VenueSelector
                    venues={venues}
                    selectedVenue={selectedVenue}
                    onSelect={handleVenueSelect}
                    onClear={handleVenueClear}
                    gql={gql}
                  />
                  {errors.venue && <div className={styles.fieldError}>{errors.venue}</div>}
                </div>
              </div>
            )}

            {/* Service types */}
            <div className={styles.section}>
              <div className={styles.sectionTitle}>Service Types</div>
              <div className={styles.checkboxGroup}>
                {serviceTypes.map((st) => (
                  <label key={st.id} className={styles.checkboxLabel}>
                    <input
                      type="checkbox"
                      checked={selectedServiceTypes.includes(st.id)}
                      onChange={() => toggleServiceType(st.id)}
                    />
                    {st.name}
                  </label>
                ))}
              </div>
            </div>

            {/* Timezone */}
            <div className={styles.section}>
              <div className={styles.sectionTitle}>Timezone</div>
              <div className={styles.timezoneRow}>
                <select
                  className={styles.select}
                  style={{ width: "auto" }}
                  value={ianaZone}
                  onChange={(e) => setIanaZone(e.target.value)}
                >
                  {tzOptions.map((tz) => (
                    <option key={tz.value} value={tz.value}>{tz.label}</option>
                  ))}
                </select>
              </div>
            </div>

            {/* Recurrence */}
            <div className={styles.section}>
              <div className={styles.sectionTitle}>Recurrence</div>
              <div className={styles.field}>
                <label className={styles.checkboxLabel}>
                  <input
                    type="checkbox"
                    checked={recurring}
                    onChange={(e) => {
                      setRecurring(e.target.checked);
                      setErrors((p) => ({ ...p, recurrenceMax: undefined }));
                    }}
                  />
                  Repeat this event
                </label>
              </div>

              <div className={`${styles.recurFields} ${!recurring ? styles.recurDisabled : ""}`}>
                <div className={styles.grid2}>
                  <div className={styles.field}>
                    <label htmlFor="recurrencePattern" className={styles.label}>Pattern</label>
                    <select
                      id="recurrencePattern"
                      className={styles.select}
                      value={recurrencePattern}
                      disabled={!recurring}
                      onChange={(e) => {
                        const p = e.target.value;
                        setRecurrencePattern(p);
                        setRecurrenceMax(RECURRENCE_DEFAULTS[p]);
                      }}
                    >
                      <option value="DAILY">Daily</option>
                      <option value="WEEKLY">Weekly</option>
                      <option value="BIWEEKLY">Every 2 Weeks</option>
                      <option value="MONTHLY">Monthly</option>
                      <option value="YEARLY">Yearly</option>
                    </select>
                  </div>

                  <div className={styles.field}>
                    <label htmlFor="recurrenceMax" className={styles.label}>
                      Occurrences
                      {recurrencePattern === "YEARLY" && <span className={styles.required}>*</span>}
                    </label>
                    <input
                      id="recurrenceMax"
                      type="number"
                      min="1"
                      className={`${styles.input}${errors.recurrenceMax ? ` ${styles.error}` : ""}`}
                      value={recurrenceMax}
                      disabled={!recurring}
                      onChange={(e) => {
                        setRecurrenceMax(e.target.value);
                        setErrors((p) => ({ ...p, recurrenceMax: undefined }));
                      }}
                      placeholder={recurrencePattern === "YEARLY" ? "Required" : "Default"}
                    />
                    {errors.recurrenceMax && (
                      <div className={styles.fieldError}>{errors.recurrenceMax}</div>
                    )}
                  </div>
                </div>

                {recurrencePattern === "MONTHLY" && (
                  <div className={styles.field}>
                    <label className={styles.label}>Week of Month</label>
                    <div className={styles.timezoneRow}>
                      <select
                        className={styles.select}
                        style={{ width: "auto" }}
                        value={recurrenceOrdinal}
                        disabled={!recurring}
                        onChange={(e) => setRecurrenceOrdinal(e.target.value)}
                      >
                        <option value="FIRST">1st</option>
                        <option value="SECOND">2nd</option>
                        <option value="THIRD">3rd</option>
                        <option value="FOURTH">4th</option>
                        <option value="LAST">Last</option>
                      </select>
                      <span className={styles.recurNote}>
                        Derived from first occurrence date — adjust if needed.
                      </span>
                    </div>
                  </div>
                )}
              </div>
            </div>

            {/* Event dates */}
            <div className={styles.section}>
              <div className={styles.sectionTitle}>
                {recurring ? "First Occurrence Dates" : "Event Dates"}
              </div>
              {recurring && (
                <p className={styles.recurNote}>
                  Enter the dates for the first occurrence; the system will generate the rest.
                </p>
              )}

              {eventDates.map((d, i) => (
                <div key={d.id}>
                  <div className={styles.dateRow}>
                    <div className={styles.dateRowField}>
                      <label className={styles.dateRowLabel}>Start Date</label>
                      <input
                        type="date"
                        className={`${styles.input} ${errors.dates?.[i] ? styles.error : ""}`}
                        value={d.startDate}
                        onChange={(e) => updateDate(d.id, "startDate", e.target.value)}
                        onBlur={(e) => {
                          const newDate = e.target.value;
                          if (newDate && (!d.endDate || d.endDate < newDate)) {
                            updateDate(d.id, "endDate", newDate);
                          }
                        }}
                      />
                    </div>
                    <div className={styles.dateRowField}>
                      <label className={styles.dateRowLabel}>Start Time</label>
                      <div className={styles.timeRow}>
                        <TimeInput
                          value24={d.startTime}
                          period={to12Hour(d.startTime).period}
                          className={`${styles.input} ${errors.dates?.[i] ? styles.error : ""}`}
                          onCommit={(t24) => updateDateStartTime(d.id, d, t24)}
                        />
                        <select
                          className={styles.ampmSelect}
                          value={to12Hour(d.startTime).period}
                          onChange={(e) => {
                            updateDateStartTime(d.id, d, to24Hour(to12Hour(d.startTime).display, e.target.value));
                          }}
                        >
                          <option>AM</option>
                          <option>PM</option>
                        </select>
                      </div>
                    </div>
                    <div className={styles.dateRowField}>
                      <label className={styles.dateRowLabel}>End Date</label>
                      <input
                        type="date"
                        className={`${styles.input} ${errors.dates?.[i] ? styles.error : ""}`}
                        value={d.endDate}
                        onChange={(e) => updateDate(d.id, "endDate", e.target.value)}
                      />
                    </div>
                    <div className={styles.dateRowField}>
                      <label className={styles.dateRowLabel}>End Time</label>
                      <div className={styles.timeRow}>
                        <TimeInput
                          value24={d.endTime}
                          period={to12Hour(d.endTime).period}
                          className={`${styles.input} ${errors.dates?.[i] ? styles.error : ""}`}
                          onCommit={(t24) => updateDate(d.id, "endTime", t24)}
                        />
                        <select
                          className={styles.ampmSelect}
                          value={to12Hour(d.endTime).period}
                          onChange={(e) => {
                            updateDate(d.id, "endTime", to24Hour(to12Hour(d.endTime).display, e.target.value));
                          }}
                        >
                          <option>AM</option>
                          <option>PM</option>
                        </select>
                      </div>
                    </div>
                    <button
                      className={styles.removeDateBtn}
                      onClick={() => removeDate(d.id)}
                      disabled={eventDates.length === 1}
                      title="Remove this date"
                    >
                      ✕
                    </button>
                  </div>
                  {errors.dates?.[i] && (
                    <div className={styles.fieldError}>{errors.dates[i]}</div>
                  )}
                </div>
              ))}

              <button className={styles.addDateBtn} onClick={addDate}>
                ＋ Add another date
              </button>
            </div>

            {/* Submit */}
            {submitError && <div className={styles.submitError}>{submitError}</div>}
            <div className={styles.formFooter}>
              <button
                className={styles.btnPrimary}
                onClick={handleSubmit}
                disabled={submitting}
              >
                {submitting ? "Creating Event…" : "Create Event"}
              </button>
              <button className={styles.btnSecondary} onClick={() => router.push("/events")}>
                Cancel
              </button>
            </div>
          </>
        )}
      </div>
      <FeedbackButton open={feedbackOpen} onClose={() => setFeedbackOpen(false)} />
    </div>
  );
}
