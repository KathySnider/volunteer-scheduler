"use client";

import { useEffect, useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import {
  getAuthToken,
  getAuthRole,
  getAuthName,
  signOut,
  adminGql,
} from "../../lib/api";
import UserMenu from "../../components/UserMenu";
import styles from "./admin-venues.module.css";

/* ----- GraphQL ----- */

const VENUES_QUERY = `
  query {
    venues {
      id name address city state zipCode timezone
    }
  }
`;

const UPDATE_VENUE = `
  mutation UpdateVenue($venue: UpdateVenueInput!) {
    updateVenue(venue: $venue) { success message }
  }
`;

const DELETE_VENUE = `
  mutation DeleteVenue($venueId: ID!) {
    deleteVenue(venueId: $venueId) { success message }
  }
`;

const CREATE_VENUE = `
  mutation CreateVenue($newVenue: NewVenueInput!) {
    createVenue(newVenue: $newVenue) { success message id }
  }
`;

/* ----- Constants ----- */

const US_TIMEZONES = [
  { value: "America/New_York",    label: "Eastern (ET)" },
  { value: "America/Chicago",     label: "Central (CT)" },
  { value: "America/Denver",      label: "Mountain (MT)" },
  { value: "America/Los_Angeles", label: "Pacific (PT)" },
  { value: "America/Anchorage",   label: "Alaska (AKT)" },
  { value: "Pacific/Honolulu",    label: "Hawaii (HT)" },
];

const EMPTY_VENUE_FORM = {
  name: "", address: "", city: "", state: "WA",
  zipCode: "", ianaZone: "America/Los_Angeles",
};

/* ----- VenueFormFields -----
   IMPORTANT: This component MUST remain defined at the module level
   (outside AdminVenuesPage). If it is moved inside the page component,
   React will treat it as a new component type on every render,
   unmount/remount it each keystroke, and focus will be lost after
   every character typed in a text input. */
function VenueFormFields({ form, setForm }) {
  return (
    <div className={styles.grid2}>
      <div className={styles.field}>
        <label className={styles.label}>Venue Name</label>
        <input className={styles.input} value={form.name}
          onChange={(e) => setForm((p) => ({ ...p, name: e.target.value }))}
          placeholder="e.g. Central Library" />
      </div>
      <div className={styles.field}>
        <label className={styles.label}>Address <span className={styles.required}>*</span></label>
        <input className={styles.input} value={form.address}
          onChange={(e) => setForm((p) => ({ ...p, address: e.target.value }))} />
      </div>
      <div className={styles.field}>
        <label className={styles.label}>City <span className={styles.required}>*</span></label>
        <input className={styles.input} value={form.city}
          onChange={(e) => setForm((p) => ({ ...p, city: e.target.value }))} />
      </div>
      <div className={styles.field}>
        <label className={styles.label}>State <span className={styles.required}>*</span></label>
        <input className={styles.input} value={form.state}
          onChange={(e) => setForm((p) => ({ ...p, state: e.target.value }))} />
      </div>
      <div className={styles.field}>
        <label className={styles.label}>Zip Code</label>
        <input className={styles.input} value={form.zipCode}
          onChange={(e) => setForm((p) => ({ ...p, zipCode: e.target.value }))} />
      </div>
      <div className={styles.field}>
        <label className={styles.label}>Timezone <span className={styles.required}>*</span></label>
        <select className={styles.select} value={form.ianaZone}
          onChange={(e) => setForm((p) => ({ ...p, ianaZone: e.target.value }))}>
          {US_TIMEZONES.map((tz) => (
            <option key={tz.value} value={tz.value}>{tz.label}</option>
          ))}
        </select>
      </div>
    </div>
  );
}

/* ----- Page ----- */

export default function AdminVenuesPage() {
  const router = useRouter();
  const [token, setToken]       = useState(null);
  const [gql, setGql]           = useState(null);
  const [userName, setUserName] = useState("");

  const [venues, setVenues]   = useState([]);
  const [loading, setLoading] = useState(true);
  const [actionMsg, setActionMsg] = useState(null);
  const [busy, setBusy]       = useState(false);

  /* Edit state */
  const [editingId, setEditingId] = useState(null);
  const [editForm, setEditForm]   = useState(EMPTY_VENUE_FORM);

  /* Add state */
  const [adding, setAdding]     = useState(false);
  const [addForm, setAddForm]   = useState(EMPTY_VENUE_FORM);

  /* ----- Auth + load ----- */
  const loadData = useCallback((bound) => {
    setLoading(true);
    bound(VENUES_QUERY, null)
      .then((res) => {
        setVenues(res.data?.venues ?? []);
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
    setToken(t);
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
  const openEdit = (venue) => {
    setEditingId(venue.id);
    setEditForm({
      name:     venue.name ?? "",
      address:  venue.address,
      city:     venue.city,
      state:    venue.state,
      zipCode:  venue.zipCode ?? "",
      ianaZone: venue.timezone,
    });
    setAdding(false);
  };

  const handleSave = async () => {
    if (!editForm.address || !editForm.city || !editForm.state) {
      showMsg("error", "Address, city, and state are required."); return;
    }
    await mutate(
      UPDATE_VENUE,
      { venue: {
        id:       editingId,
        name:     editForm.name.trim() || null,
        address:  editForm.address.trim(),
        city:     editForm.city.trim(),
        state:    editForm.state.trim(),
        zipCode:  editForm.zipCode.trim() || null,
        ianaZone: editForm.ianaZone,
      }},
      "Venue updated.",
      () => setEditingId(null),
    );
  };

  /* ----- Delete ----- */
  const handleDelete = async (venue) => {
    if (!window.confirm(`Delete "${venue.name || venue.address}"? This cannot be undone.`)) return;
    await mutate(DELETE_VENUE, { venueId: venue.id }, "Venue deleted.");
  };

  /* ----- Add ----- */
  const handleAdd = async () => {
    if (!addForm.address || !addForm.city || !addForm.state) {
      showMsg("error", "Address, city, and state are required."); return;
    }
    await mutate(
      CREATE_VENUE,
      { newVenue: {
        name:     addForm.name.trim() || null,
        address:  addForm.address.trim(),
        city:     addForm.city.trim(),
        state:    addForm.state.trim(),
        zipCode:  addForm.zipCode.trim() || null,
        ianaZone: addForm.ianaZone,
      }},
      "Venue created.",
      () => { setAdding(false); setAddForm(EMPTY_VENUE_FORM); },
    );
  };

  const handleSignOut = async () => { await signOut(token); router.replace("/login"); };

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
          <h1 className={styles.pageTitle}>Manage Venues</h1>
          {!adding && (
            <button className={styles.createBtn} onClick={() => { setAdding(true); setEditingId(null); setAddForm(EMPTY_VENUE_FORM); }}>
              + Add Venue
            </button>
          )}
        </div>

        {/* Banners */}
        {actionMsg?.type === "success" && <div className={styles.successBanner}>{actionMsg.text}</div>}
        {actionMsg?.type === "error"   && <div className={styles.errorBanner}>{actionMsg.text}</div>}

        {/* Add venue form */}
        {adding && (
          <div className={styles.formCard}>
            <div className={styles.formCardTitle}>New Venue</div>
            <VenueFormFields form={addForm} setForm={setAddForm} />
            <div className={styles.formActions}>
              <button className={styles.btnPrimary} onClick={handleAdd} disabled={busy}>Create Venue</button>
              <button className={styles.btnSecondary} onClick={() => setAdding(false)}>Cancel</button>
            </div>
          </div>
        )}

        {/* Loading */}
        {loading && (
          <div className={styles.stateBox}>
            <div className={styles.spinner} />
            <p>Loading venues…</p>
          </div>
        )}

        {/* Venue list */}
        {!loading && venues.length === 0 && (
          <div className={styles.stateBox}>
            <div className={styles.stateTitle}>No venues yet</div>
            <p>Add a venue above.</p>
          </div>
        )}

        {!loading && venues.map((venue) => {
          const isEditing = editingId === venue.id;
          return (
            <div key={venue.id} className={styles.venueCard}>
              <div className={styles.venueHeader}>
                <div>
                  {venue.name && <div className={styles.venueName}>{venue.name}</div>}
                  <div className={styles.venueAddress}>
                    {venue.address}, {venue.city}, {venue.state}{venue.zipCode ? ` ${venue.zipCode}` : ""}
                  </div>
                  <div className={styles.venueMeta}>
                    {venue.timezone}
                  </div>
                </div>
                <div className={styles.venueActions}>
                  <button
                    className={`${styles.iconBtn} ${styles.iconBtnEdit}`}
                    title="Edit venue"
                    onClick={() => isEditing ? setEditingId(null) : openEdit(venue)}
                  >✏</button>
                  <button
                    className={`${styles.iconBtn} ${styles.iconBtnDelete}`}
                    title="Delete venue"
                    onClick={() => handleDelete(venue)}
                    disabled={busy}
                  >🗑</button>
                </div>
              </div>

              {/* Inline edit form */}
              {isEditing && (
                <div className={styles.editForm}>
                  <VenueFormFields form={editForm} setForm={setEditForm} />
                  <div className={styles.formActions}>
                    <button className={styles.btnPrimary} onClick={handleSave} disabled={busy}>Save Changes</button>
                    <button className={styles.btnSecondary} onClick={() => setEditingId(null)}>Cancel</button>
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
