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
import styles from "./admin-regions.module.css";

/* =========================================================
   GraphQL
   ========================================================= */

const LOOKUP_VALUES = `
  query {
    lookupValues {
      regions { id code name isActive }
    }
  }
`;

const CREATE_REGION = `
  mutation CreateRegion($newRegion: NewRegionInput!) {
    createRegion(newRegion: $newRegion) { success message id }
  }
`;

const UPDATE_REGION = `
  mutation UpdateRegion($region: UpdateRegionInput!) {
    updateRegion(region: $region) { success message }
  }
`;

const DELETE_REGION = `
  mutation DeleteRegion($regionId: Int!) {
    deleteRegion(regionId: $regionId) { success message }
  }
`;

/* =========================================================
   RegionFormFields
   IMPORTANT: Must remain defined at module level (outside the
   page component) so React does not treat it as a new
   component type on each render, which would unmount/remount
   it on every keystroke and steal input focus.
   ========================================================= */

function RegionFormFields({ form, setForm }) {
  return (
    <div className={styles.grid2}>
      <div className={styles.field}>
        <label className={styles.label}>
          Code <span className={styles.required}>*</span>
        </label>
        <input
          className={styles.input}
          value={form.code}
          placeholder="e.g. SEA"
          onChange={(e) => setForm((p) => ({ ...p, code: e.target.value }))}
        />
      </div>
      <div className={styles.field}>
        <label className={styles.label}>
          Name <span className={styles.required}>*</span>
        </label>
        <input
          className={styles.input}
          value={form.name}
          placeholder="e.g. Seattle"
          onChange={(e) => setForm((p) => ({ ...p, name: e.target.value }))}
        />
      </div>
    </div>
  );
}

/* =========================================================
   Page
   ========================================================= */

const EMPTY_FORM = { code: "", name: "" };

export default function AdminRegionsPage() {
  const router = useRouter();
  const [gql, setGql]           = useState(null);
  const [userName, setUserName] = useState("");

  const [regions, setRegions]     = useState([]);
  const [loading, setLoading]     = useState(true);
  const [actionMsg, setActionMsg] = useState(null);
  const [busy, setBusy]           = useState(false);

  /* Add form */
  const [showAdd, setShowAdd]   = useState(false);
  const [addForm, setAddForm]   = useState(EMPTY_FORM);

  /* Edit state */
  const [editingId, setEditingId] = useState(null);
  const [editForm, setEditForm]   = useState(EMPTY_FORM);

  /* ----- Auth + load ----- */
  const loadData = useCallback((bound) => {
    setLoading(true);
    bound(LOOKUP_VALUES, null)
      .then((res) => {
        setRegions(res.data?.lookupValues?.regions ?? []);
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

  /* ----- Add ----- */
  const handleAdd = async () => {
    if (!addForm.code.trim() || !addForm.name.trim()) {
      showMsg("error", "Code and name are required.");
      return;
    }
    await mutate(
      CREATE_REGION,
      { newRegion: { code: addForm.code.trim(), name: addForm.name.trim() } },
      "Region created.",
      () => { setShowAdd(false); setAddForm(EMPTY_FORM); },
    );
  };

  /* ----- Edit ----- */
  const openEdit = (region) => {
    setEditingId(region.id);
    setEditForm({ code: region.code, name: region.name });
  };

  const handleSave = async () => {
    if (!editForm.code.trim() || !editForm.name.trim()) {
      showMsg("error", "Code and name are required.");
      return;
    }
    await mutate(
      UPDATE_REGION,
      { region: { id: editingId, code: editForm.code.trim(), name: editForm.name.trim() } },
      "Region updated.",
      () => setEditingId(null),
    );
  };

  /* ----- Delete ----- */
  const handleDelete = async (region) => {
    if (!window.confirm(`Delete region "${region.name}" (${region.code})? This cannot be undone.`)) return;
    await mutate(DELETE_REGION, { regionId: region.id }, "Region deleted.");
  };

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
          <h1 className={styles.pageTitle}>Manage Regions</h1>
          {!showAdd && (
            <button
              className={styles.btnOutline}
              onClick={() => { setShowAdd(true); setAddForm(EMPTY_FORM); setActionMsg(null); }}
            >
              + Add Region
            </button>
          )}
        </div>

        {/* Banners */}
        {actionMsg?.type === "success" && <div className={styles.successBanner}>{actionMsg.text}</div>}
        {actionMsg?.type === "error"   && <div className={styles.errorBanner}>{actionMsg.text}</div>}

        {/* Add form */}
        {showAdd && (
          <div className={styles.addForm}>
            <RegionFormFields form={addForm} setForm={setAddForm} />
            <div className={styles.formActions}>
              <button className={styles.btnPrimary} onClick={handleAdd} disabled={busy}>
                Create Region
              </button>
              <button className={styles.btnSecondary} onClick={() => { setShowAdd(false); setAddForm(EMPTY_FORM); }}>
                Cancel
              </button>
            </div>
          </div>
        )}

        {/* Loading */}
        {loading && (
          <div className={styles.stateBox}>
            <div className={styles.spinner} />
            <p>Loading regions…</p>
          </div>
        )}

        {/* Empty */}
        {!loading && regions.length === 0 && (
          <div className={styles.stateBox}>
            <div className={styles.stateTitle}>No regions yet.</div>
          </div>
        )}

        {/* Region list */}
        {!loading && regions.map((region) => {
          const isEditing = editingId === region.id;

          return (
            <div key={region.id} className={styles.regionCard}>
              {/* Card header */}
              <div className={styles.regionHeader}>
                <div className={styles.regionInfo}>
                  <div className={styles.regionName}>
                    {region.name}
                    <span className={styles.regionCode}>{region.code}</span>
                  </div>
                  <div className={styles.regionMeta}>
                    <span className={region.isActive ? styles.activeBadge : styles.inactiveBadge}>
                      {region.isActive ? "Active" : "Inactive"}
                    </span>
                  </div>
                </div>
                <div className={styles.regionActions}>
                  <button
                    className={`${styles.iconBtn} ${styles.iconBtnEdit}`}
                    title="Edit region"
                    onClick={() => isEditing ? setEditingId(null) : openEdit(region)}
                  >✏</button>
                  <button
                    className={`${styles.iconBtn} ${styles.iconBtnDelete}`}
                    title="Delete region"
                    onClick={() => handleDelete(region)}
                    disabled={busy}
                  >🗑</button>
                </div>
              </div>

              {/* Inline edit form */}
              {isEditing && (
                <div className={styles.editForm}>
                  <RegionFormFields form={editForm} setForm={setEditForm} />
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
