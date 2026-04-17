"use client";

import { useEffect, useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import {
  getAuthToken,
  getAuthRole,
  getAuthName,
  signOut,
  adminGql,
  volunteerGql,
} from "../../lib/api";
import UserMenu from "../../components/UserMenu";
import styles from "./admin-job-types.module.css";

/* =========================================================
   GraphQL
   ========================================================= */

const LOOKUP_VALUES = `
  query {
    lookupValues {
      jobTypes { id code name sortOrder isActive }
    }
  }
`;

const CREATE_JOB_TYPE = `
  mutation CreateJobType($newJob: NewJobTypeInput!) {
    createJobType(newJob: $newJob) { success message }
  }
`;

const UPDATE_JOB_TYPE = `
  mutation UpdateJobType($job: UpdateJobTypeInput!) {
    updateJobType(job: $job) { success message }
  }
`;

const DELETE_JOB_TYPE = `
  mutation DeleteJobType($jobId: Int!) {
    deleteJobType(JobId: $jobId) { success message }
  }
`;

/* =========================================================
   JobTypeFormFields
   Defined at module level to prevent remount on each render
   (which would steal focus on every keystroke).
   ========================================================= */

function JobTypeFormFields({ form, setForm }) {
  return (
    <div className={styles.grid3}>
      <div className={styles.field}>
        <label className={styles.label}>
          Code <span className={styles.required}>*</span>
        </label>
        <input
          className={styles.input}
          value={form.code}
          placeholder="e.g. GRTG"
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
          placeholder="e.g. Greeter"
          onChange={(e) => setForm((p) => ({ ...p, name: e.target.value }))}
        />
      </div>
      <div className={styles.field}>
        <label className={styles.label}>Sort Order</label>
        <input
          className={styles.input}
          type="number"
          min="0"
          value={form.sortOrder}
          placeholder="0"
          onChange={(e) => setForm((p) => ({ ...p, sortOrder: e.target.value }))}
        />
      </div>
    </div>
  );
}

/* =========================================================
   Page
   ========================================================= */

const EMPTY_FORM = { code: "", name: "", sortOrder: "25" };

export default function AdminJobTypesPage() {
  const router = useRouter();
  const [token, setToken]             = useState(null);
  const [adminGqlFn, setAdminGqlFn]   = useState(null);
  const [volGqlFn, setVolGqlFn]       = useState(null);
  const [userName, setUserName]       = useState("");

  const [jobTypes, setJobTypes]       = useState([]);
  const [loading, setLoading]         = useState(true);
  const [actionMsg, setActionMsg]     = useState(null);
  const [busy, setBusy]               = useState(false);

  /* Add form */
  const [showAdd, setShowAdd]   = useState(false);
  const [addForm, setAddForm]   = useState(EMPTY_FORM);

  /* Edit state */
  const [editingId, setEditingId] = useState(null);
  const [editForm, setEditForm]   = useState(EMPTY_FORM);

  /* ----- Auth + load ----- */
  const loadData = useCallback((volFn) => {
    setLoading(true);
    volFn(LOOKUP_VALUES, null)
      .then((res) => {
        const types = res.data?.lookupValues?.jobTypes ?? [];
        // Sort by sortOrder, then name
        types.sort((a, b) => a.sortOrder - b.sortOrder || a.name.localeCompare(b.name));
        setJobTypes(types);
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
    const adminFn = (q, v) => adminGql(q, v, t);
    const volFn   = (q, v) => volunteerGql(q, v, t);
    setToken(t);
    setAdminGqlFn(() => adminFn);
    setVolGqlFn(() => volFn);
    setUserName(getAuthName() ?? "");
    loadData(volFn);
  }, [router, loadData]);

  /* ----- Helpers ----- */
  const showMsg = (type, text) => setActionMsg({ type, text });

  const mutate = async (mutation, variables, successMsg, onSuccess) => {
    setBusy(true);
    setActionMsg(null);
    try {
      const res = await adminGqlFn(mutation, variables);
      const key = Object.keys(res.data ?? {})[0];
      const result = res.data?.[key];
      if (res.errors || !result?.success) {
        showMsg("error", result?.message ?? res.errors?.[0]?.message ?? "Operation failed.");
        return null;
      }
      showMsg("success", successMsg);
      if (onSuccess) onSuccess(result);
      loadData(volGqlFn);
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
      CREATE_JOB_TYPE,
      {
        newJob: {
          code:      addForm.code.trim(),
          name:      addForm.name.trim(),
          sortOrder: parseInt(addForm.sortOrder, 10) || 0,
        },
      },
      "Job type created.",
      () => { setShowAdd(false); setAddForm(EMPTY_FORM); },
    );
  };

  /* ----- Edit ----- */
  const openEdit = (jt) => {
    setEditingId(jt.id);
    setEditForm({ code: jt.code, name: jt.name, sortOrder: String(jt.sortOrder) });
  };

  const handleSave = async () => {
    if (!editForm.code.trim() || !editForm.name.trim()) {
      showMsg("error", "Code and name are required.");
      return;
    }
    await mutate(
      UPDATE_JOB_TYPE,
      {
        job: {
          id:        editingId,
          code:      editForm.code.trim(),
          name:      editForm.name.trim(),
          sortOrder: parseInt(editForm.sortOrder, 10) || 0,
        },
      },
      "Job type updated.",
      () => setEditingId(null),
    );
  };

  /* ----- Delete ----- */
  const handleDelete = async (jt) => {
    if (!window.confirm(`Delete job type "${jt.name}" (${jt.code})? This cannot be undone.`)) return;
    await mutate(DELETE_JOB_TYPE, { jobId: jt.id }, "Job type deleted.");
  };

  const handleSignOut = async () => { await signOut(token); router.replace("/login"); };

  if (!adminGqlFn) return null;

  return (
    <div className={styles.page}>
      {/* Top bar */}
      <div className={styles.topBar}>
        <a href="/events" className={styles.backLink}>← Back to Events</a>
        <UserMenu name={userName} isAdmin={true} onSignOut={handleSignOut} />
      </div>

      <div className={styles.content}>
        <div className={styles.pageHeader}>
          <h1 className={styles.pageTitle}>Manage Job Types</h1>
          {!showAdd && (
            <button
              className={styles.btnOutline}
              onClick={() => { setShowAdd(true); setAddForm(EMPTY_FORM); setActionMsg(null); }}
            >
              + Add Job Type
            </button>
          )}
        </div>

        {/* Banners */}
        {actionMsg?.type === "success" && <div className={styles.successBanner}>{actionMsg.text}</div>}
        {actionMsg?.type === "error"   && <div className={styles.errorBanner}>{actionMsg.text}</div>}

        {/* Add form */}
        {showAdd && (
          <div className={styles.addForm}>
            <JobTypeFormFields form={addForm} setForm={setAddForm} />
            <div className={styles.formActions}>
              <button className={styles.btnPrimary} onClick={handleAdd} disabled={busy}>
                Create Job Type
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
            <p>Loading job types...</p>
          </div>
        )}

        {/* Empty */}
        {!loading && jobTypes.length === 0 && (
          <div className={styles.stateBox}>
            <div className={styles.stateTitle}>No job types yet.</div>
          </div>
        )}

        {/* Job type list */}
        {!loading && jobTypes.map((jt) => {
          const isEditing = editingId === jt.id;

          return (
            <div key={jt.id} className={styles.card}>
              <div className={styles.cardHeader}>
                <div className={styles.cardInfo}>
                  <div className={styles.cardName}>
                    {jt.name}
                    <span className={styles.codeTag}>{jt.code}</span>
                    <span className={styles.orderTag}>#{jt.sortOrder}</span>
                  </div>
                  <div className={styles.cardMeta}>
                    <span className={jt.isActive ? styles.activeBadge : styles.inactiveBadge}>
                      {jt.isActive ? "Active" : "Inactive"}
                    </span>
                  </div>
                </div>
                <div className={styles.cardActions}>
                  <button
                    className={`${styles.iconBtn} ${styles.iconBtnEdit}`}
                    title="Edit job type"
                    onClick={() => isEditing ? setEditingId(null) : openEdit(jt)}
                  >
                    &#9999;
                  </button>
                  <button
                    className={`${styles.iconBtn} ${styles.iconBtnDelete}`}
                    title="Delete job type"
                    onClick={() => handleDelete(jt)}
                    disabled={busy}
                  >
                    &#128465;
                  </button>
                </div>
              </div>

              {/* Inline edit form */}
              {isEditing && (
                <div className={styles.editForm}>
                  <JobTypeFormFields form={editForm} setForm={setEditForm} />
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
