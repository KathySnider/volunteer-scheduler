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
import styles from "./admin-volunteers.module.css";

/* =========================================================
   GraphQL
   ========================================================= */

const ALL_VOLUNTEERS = `
  query {
    allVolunteers {
      id firstName lastName email phone zipCode role
    }
  }
`;

const VOLUNTEER_SHIFTS = `
  query VolunteerShifts($volunteerId: ID!, $filter: ShiftTimeFilter!) {
    volunteerShifts(volunteerId: $volunteerId, filter: $filter) {
      shiftId startDateTime endDateTime jobName eventName eventId
    }
  }
`;

const UPDATE_VOLUNTEER = `
  mutation UpdateVolunteer($profile: UpdateVolunteerInput!) {
    updateVolunteer(profile: $profile) { success message }
  }
`;

const DELETE_VOLUNTEER = `
  mutation DeleteVolunteer($volunteerId: ID!) {
    deleteVolunteer(volunteerId: $volunteerId) { success message }
  }
`;

/* =========================================================
   Helpers
   ========================================================= */

function formatDisplay(utcString) {
  if (!utcString) return "—";
  return new Date(utcString).toLocaleString(undefined, {
    month: "short", day: "numeric", year: "numeric",
    hour: "numeric", minute: "2-digit",
  });
}

/* =========================================================
   VolunteerFormFields
   IMPORTANT: Must remain defined at module level (outside the
   page component) so React does not treat it as a new
   component type on each render, which would unmount/remount
   it on every keystroke and steal input focus.
   ========================================================= */

function VolunteerFormFields({ form, setForm }) {
  return (
    <>
      <div className={styles.grid2}>
        <div className={styles.field}>
          <label className={styles.label}>
            First Name <span className={styles.required}>*</span>
          </label>
          <input
            className={styles.input}
            value={form.firstName}
            onChange={(e) => setForm((p) => ({ ...p, firstName: e.target.value }))}
          />
        </div>
        <div className={styles.field}>
          <label className={styles.label}>
            Last Name <span className={styles.required}>*</span>
          </label>
          <input
            className={styles.input}
            value={form.lastName}
            onChange={(e) => setForm((p) => ({ ...p, lastName: e.target.value }))}
          />
        </div>
        <div className={styles.field}>
          <label className={styles.label}>
            Email <span className={styles.required}>*</span>
          </label>
          <input
            className={styles.input}
            type="email"
            value={form.email}
            onChange={(e) => setForm((p) => ({ ...p, email: e.target.value }))}
          />
        </div>
        <div className={styles.field}>
          <label className={styles.label}>Phone</label>
          <input
            className={styles.input}
            value={form.phone}
            onChange={(e) => setForm((p) => ({ ...p, phone: e.target.value }))}
          />
        </div>
        <div className={styles.field}>
          <label className={styles.label}>Zip Code</label>
          <input
            className={styles.input}
            value={form.zipCode}
            onChange={(e) => setForm((p) => ({ ...p, zipCode: e.target.value }))}
          />
        </div>
        <div className={styles.field}>
          <label className={styles.label}>
            Role <span className={styles.required}>*</span>
          </label>
          <select
            className={styles.select}
            value={form.role}
            onChange={(e) => setForm((p) => ({ ...p, role: e.target.value }))}
          >
            <option value="VOLUNTEER">Volunteer</option>
            <option value="ADMINISTRATOR">Administrator</option>
          </select>
        </div>
      </div>
    </>
  );
}

/* =========================================================
   Page
   ========================================================= */

const EMPTY_FORM = {
  firstName: "", lastName: "", email: "", phone: "", zipCode: "", role: "VOLUNTEER",
};

export default function AdminVolunteersPage() {
  const router = useRouter();
  const [gql, setGql]           = useState(null);
  const [userName, setUserName] = useState("");

  const [volunteers, setVolunteers] = useState([]);
  const [loading, setLoading]       = useState(true);
  const [actionMsg, setActionMsg]   = useState(null);
  const [busy, setBusy]             = useState(false);

  /* Search filter */
  const [search, setSearch] = useState("");

  /* Edit state */
  const [editingId, setEditingId] = useState(null);
  const [editForm, setEditForm]   = useState(EMPTY_FORM);

  /* Shifts panel state — keyed by volunteer id */
  const [openShiftsId, setOpenShiftsId]     = useState(null);
  const [shiftFilter, setShiftFilter]       = useState("UPCOMING"); // "UPCOMING" | "ALL"
  const [shiftsData, setShiftsData]         = useState(null);  // null = not yet loaded
  const [shiftsLoading, setShiftsLoading]   = useState(false);

  /* ----- Auth + load ----- */
  const loadData = useCallback((bound) => {
    setLoading(true);
    bound(ALL_VOLUNTEERS, null)
      .then((res) => {
        setVolunteers(res.data?.allVolunteers ?? []);
        if (res.errors) setActionMsg({ type: "error", text: res.errors[0]?.message ?? "Error loading data." });
      })
      .catch(() => setActionMsg({ type: "error", text: "Unable to reach the server." }))
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
    loadData(bound);
  }, [router, loadData]);

  /* ----- Helpers ----- */
  const showMsg = (type, text) => setActionMsg({ type, text });

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
      loadData(gql);
      return result;
    } catch {
      showMsg("error", "Unable to reach the server.");
      return null;
    } finally {
      setBusy(false);
    }
  };

  /* ----- Edit ----- */
  const openEdit = (vol) => {
    setEditingId(vol.id);
    setEditForm({
      firstName: vol.firstName,
      lastName:  vol.lastName,
      email:     vol.email,
      phone:     vol.phone ?? "",
      zipCode:   vol.zipCode ?? "",
      role:      vol.role,
    });
    setOpenShiftsId(null);
  };

  const handleSave = async () => {
    if (!editForm.firstName.trim() || !editForm.lastName.trim() || !editForm.email.trim()) {
      showMsg("error", "First name, last name, and email are required.");
      return;
    }
    await mutate(
      UPDATE_VOLUNTEER,
      { profile: {
        id:        editingId,
        firstName: editForm.firstName.trim(),
        lastName:  editForm.lastName.trim(),
        email:     editForm.email.trim(),
        phone:     editForm.phone.trim() || null,
        zipCode:   editForm.zipCode.trim() || null,
        role:      editForm.role,
      }},
      "Volunteer updated.",
      () => setEditingId(null),
    );
  };

  /* ----- Delete ----- */
  const handleDelete = async (vol) => {
    const name = `${vol.firstName} ${vol.lastName}`;
    if (!window.confirm(`Delete "${name}"? This cannot be undone.`)) return;
    await mutate(DELETE_VOLUNTEER, { volunteerId: vol.id }, "Volunteer deleted.");
  };

  /* ----- Shifts panel ----- */
  const loadShifts = useCallback((bound, volunteerId, filter) => {
    setShiftsLoading(true);
    setShiftsData(null);
    bound(VOLUNTEER_SHIFTS, { volunteerId, filter })
      .then((res) => {
        setShiftsData(res.data?.volunteerShifts ?? []);
      })
      .catch(() => setShiftsData([]))
      .finally(() => setShiftsLoading(false));
  }, []);

  const toggleShiftsPanel = (vol) => {
    if (openShiftsId === vol.id) {
      setOpenShiftsId(null);
      setShiftsData(null);
      return;
    }
    setOpenShiftsId(vol.id);
    setShiftFilter("UPCOMING");
    setEditingId(null);
    loadShifts(gql, vol.id, "UPCOMING");
  };

  const handleFilterChange = (newFilter) => {
    setShiftFilter(newFilter);
    setShiftsData(null);
    loadShifts(gql, openShiftsId, newFilter);
  };

  /* Group shifts by event name */
  const groupedShifts = (shiftsData ?? []).reduce((acc, s) => {
    const key = s.eventName;
    if (!acc[key]) acc[key] = [];
    acc[key].push(s);
    return acc;
  }, {});

  /* ----- Client-side filter ----- */
  const lc = search.toLowerCase();
  const filtered = volunteers.filter((v) =>
    !search ||
    `${v.firstName} ${v.lastName}`.toLowerCase().includes(lc) ||
    v.email.toLowerCase().includes(lc)
  );

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
        <div className={styles.pageHeader}>
          <h1 className={styles.pageTitle}>Manage Volunteers</h1>
        </div>

        {/* Banners */}
        {actionMsg?.type === "success" && <div className={styles.successBanner}>{actionMsg.text}</div>}
        {actionMsg?.type === "error"   && <div className={styles.errorBanner}>{actionMsg.text}</div>}

        {/* Search */}
        <div className={styles.filterBar}>
          <input
            className={styles.searchInput}
            type="text"
            placeholder="Search by name or email…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>

        {/* Loading */}
        {loading && (
          <div className={styles.stateBox}>
            <div className={styles.spinner} />
            <p>Loading volunteers…</p>
          </div>
        )}

        {/* Empty */}
        {!loading && filtered.length === 0 && (
          <div className={styles.stateBox}>
            <div className={styles.stateTitle}>
              {search ? "No volunteers match your search." : "No volunteers yet."}
            </div>
          </div>
        )}

        {/* Volunteer list */}
        {!loading && filtered.map((vol) => {
          const isEditing      = editingId === vol.id;
          const shiftsOpen     = openShiftsId === vol.id;
          const isAdmin        = vol.role === "ADMINISTRATOR";

          return (
            <div key={vol.id} className={styles.volCard}>
              {/* Card header */}
              <div className={styles.volHeader}>
                <div className={styles.volInfo}>
                  <div className={styles.volName}>
                    {vol.firstName} {vol.lastName}
                    <span className={`${styles.roleBadge} ${isAdmin ? styles.roleBadgeAdmin : styles.roleBadgeVolunteer}`}>
                      {isAdmin ? "Administrator" : "Volunteer"}
                    </span>
                  </div>
                  <div className={styles.volMeta}>{vol.email}</div>
                  {vol.phone  && <div className={styles.volMeta}>{vol.phone}</div>}
                  {vol.zipCode && <div className={styles.volMeta}>Zip: {vol.zipCode}</div>}
                </div>
                <div className={styles.volActions}>
                  <button
                    className={`${styles.iconBtn} ${styles.iconBtnShifts} ${shiftsOpen ? styles.iconBtnShiftsActive : ""}`}
                    title="View shifts"
                    onClick={() => toggleShiftsPanel(vol)}
                  >📋</button>
                  <button
                    className={`${styles.iconBtn} ${styles.iconBtnEdit}`}
                    title="Edit volunteer"
                    onClick={() => isEditing ? setEditingId(null) : openEdit(vol)}
                  >✏</button>
                  <button
                    className={`${styles.iconBtn} ${styles.iconBtnDelete}`}
                    title="Delete volunteer"
                    onClick={() => handleDelete(vol)}
                    disabled={busy}
                  >🗑</button>
                </div>
              </div>

              {/* Shifts panel */}
              {shiftsOpen && (
                <div className={styles.shiftsPanel}>
                  <div className={styles.shiftsPanelHeader}>
                    <span className={styles.shiftsPanelTitle}>Shifts</span>
                    <div className={styles.shiftFilterToggle}>
                      <button
                        className={`${styles.toggleBtn} ${shiftFilter === "UPCOMING" ? styles.toggleBtnActive : ""}`}
                        onClick={() => shiftFilter !== "UPCOMING" && handleFilterChange("UPCOMING")}
                      >
                        Upcoming
                      </button>
                      <button
                        className={`${styles.toggleBtn} ${shiftFilter === "ALL" ? styles.toggleBtnActive : ""}`}
                        onClick={() => shiftFilter !== "ALL" && handleFilterChange("ALL")}
                      >
                        All
                      </button>
                    </div>
                  </div>

                  {shiftsLoading && (
                    <div className={styles.shiftsLoading}>Loading shifts…</div>
                  )}

                  {!shiftsLoading && shiftsData !== null && (
                    <div className={styles.shiftsList}>
                      {shiftsData.length === 0 ? (
                        <div className={styles.emptyMsg}>No shifts found.</div>
                      ) : (
                        Object.entries(groupedShifts).map(([eventName, shifts]) => (
                          <div key={eventName} className={styles.shiftGroup}>
                            <div className={styles.shiftGroupTitle}>{eventName}</div>
                            {shifts.map((s) => (
                              <div key={s.shiftId} className={styles.shiftItem}>
                                <span className={styles.shiftTime}>
                                  {formatDisplay(s.startDateTime)}
                                </span>
                                <span className={styles.shiftMeta}>
                                  to {formatDisplay(s.endDateTime)} · {s.jobName}
                                </span>
                              </div>
                            ))}
                          </div>
                        ))
                      )}
                    </div>
                  )}
                </div>
              )}

              {/* Inline edit form */}
              {isEditing && (
                <div className={styles.editForm}>
                  <VolunteerFormFields form={editForm} setForm={setEditForm} />
                  <div className={styles.formActions}>
                    <button className={styles.btnPrimary} onClick={handleSave} disabled={busy}>
                      Save Changes
                    </button>
                    <button className={styles.btnSecondary} onClick={() => setEditingId(null)}>
                      Cancel
                    </button>
                  </div>
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
