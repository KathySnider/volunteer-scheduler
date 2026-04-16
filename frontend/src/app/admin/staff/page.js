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
import styles from "./admin-staff.module.css";

/* =========================================================
   GraphQL
   ========================================================= */

const ALL_STAFF = `
  query {
    staff {
      id firstName lastName email phone position
    }
  }
`;

const CREATE_STAFF = `
  mutation CreateStaff($newStaff: NewStaffInput!) {
    createStaff(newStaff: $newStaff) { success message id }
  }
`;

const UPDATE_STAFF = `
  mutation UpdateStaff($staff: UpdateStaffInput!) {
    updateStaff(staff: $staff) { success message }
  }
`;

const DELETE_STAFF = `
  mutation DeleteStaff($staffId: ID!) {
    deleteStaff(staffId: $staffId) { success message }
  }
`;

/* =========================================================
   StaffFormFields
   IMPORTANT: Must remain defined at module level (outside the
   page component) so React does not treat it as a new
   component type on each render, which would unmount/remount
   it on every keystroke and steal input focus.
   ========================================================= */

function StaffFormFields({ form, setForm }) {
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
          <label className={styles.label}>Position</label>
          <input
            className={styles.input}
            value={form.position}
            onChange={(e) => setForm((p) => ({ ...p, position: e.target.value }))}
          />
        </div>
      </div>
    </>
  );
}

/* =========================================================
   Page
   ========================================================= */

const EMPTY_FORM = {
  firstName: "", lastName: "", email: "", phone: "", position: "",
};

export default function AdminStaffPage() {
  const router = useRouter();
  const [gql, setGql]           = useState(null);
  const [userName, setUserName] = useState("");

  const [staff, setStaff]           = useState([]);
  const [loading, setLoading]       = useState(true);
  const [actionMsg, setActionMsg]   = useState(null);
  const [busy, setBusy]             = useState(false);

  /* Search filter */
  const [search, setSearch] = useState("");

  /* Add form state */
  const [addOpen, setAddOpen]   = useState(false);
  const [addForm, setAddForm]   = useState(EMPTY_FORM);

  /* Edit state */
  const [editingId, setEditingId] = useState(null);
  const [editForm, setEditForm]   = useState(EMPTY_FORM);

  /* ----- Auth + load ----- */
  const loadData = useCallback((bound) => {
    setLoading(true);
    bound(ALL_STAFF, null)
      .then((res) => {
        setStaff(res.data?.staff ?? []);
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
    if (!addForm.firstName.trim() || !addForm.lastName.trim() || !addForm.email.trim()) {
      showMsg("error", "First name, last name, and email are required.");
      return;
    }
    await mutate(
      CREATE_STAFF,
      { newStaff: {
        firstName: addForm.firstName.trim(),
        lastName:  addForm.lastName.trim(),
        email:     addForm.email.trim(),
        phone:     addForm.phone.trim() || null,
        position:  addForm.position.trim() || null,
      }},
      "Staff member added.",
      () => {
        setAddOpen(false);
        setAddForm(EMPTY_FORM);
      },
    );
  };

  /* ----- Edit ----- */
  const openEdit = (member) => {
    setEditingId(member.id);
    setEditForm({
      firstName: member.firstName,
      lastName:  member.lastName,
      email:     member.email,
      phone:     member.phone ?? "",
      position:  member.position ?? "",
    });
  };

  const handleSave = async () => {
    if (!editForm.firstName.trim() || !editForm.lastName.trim() || !editForm.email.trim()) {
      showMsg("error", "First name, last name, and email are required.");
      return;
    }
    await mutate(
      UPDATE_STAFF,
      { staff: {
        id:        editingId,
        firstName: editForm.firstName.trim(),
        lastName:  editForm.lastName.trim(),
        email:     editForm.email.trim(),
        phone:     editForm.phone.trim() || null,
        position:  editForm.position.trim() || null,
      }},
      "Staff member updated.",
      () => setEditingId(null),
    );
  };

  /* ----- Delete ----- */
  const handleDelete = async (member) => {
    const name = `${member.firstName} ${member.lastName}`;
    if (!window.confirm(`Delete "${name}"? This cannot be undone.`)) return;
    await mutate(DELETE_STAFF, { staffId: member.id }, "Staff member deleted.");
  };

  /* ----- Client-side filter ----- */
  const lc = search.toLowerCase();
  const filtered = staff.filter((m) =>
    !search ||
    `${m.firstName} ${m.lastName}`.toLowerCase().includes(lc) ||
    m.email.toLowerCase().includes(lc) ||
    (m.position ?? "").toLowerCase().includes(lc)
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
          <h1 className={styles.pageTitle}>Manage Staff</h1>
          <button
            className={styles.btnOutline}
            onClick={() => {
              setAddOpen((v) => !v);
              setAddForm(EMPTY_FORM);
              setActionMsg(null);
            }}
          >
            {addOpen ? "Cancel" : "+ Add Staff Member"}
          </button>
        </div>

        {/* Add form */}
        {addOpen && (
          <div className={styles.addForm}>
            <StaffFormFields form={addForm} setForm={setAddForm} />
            <div className={styles.formActions}>
              <button className={styles.btnPrimary} onClick={handleAdd} disabled={busy}>
                Save Staff Member
              </button>
              <button className={styles.btnSecondary} onClick={() => { setAddOpen(false); setAddForm(EMPTY_FORM); }}>
                Cancel
              </button>
            </div>
          </div>
        )}

        {/* Banners */}
        {actionMsg?.type === "success" && <div className={styles.successBanner}>{actionMsg.text}</div>}
        {actionMsg?.type === "error"   && <div className={styles.errorBanner}>{actionMsg.text}</div>}

        {/* Search */}
        <div className={styles.filterBar}>
          <input
            className={styles.searchInput}
            type="text"
            placeholder="Search by name, email, or position…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>

        {/* Loading */}
        {loading && (
          <div className={styles.stateBox}>
            <div className={styles.spinner} />
            <p>Loading staff…</p>
          </div>
        )}

        {/* Empty */}
        {!loading && filtered.length === 0 && (
          <div className={styles.stateBox}>
            <div className={styles.stateTitle}>
              {search ? "No staff members match your search." : "No staff members yet."}
            </div>
          </div>
        )}

        {/* Staff list */}
        {!loading && filtered.map((member) => {
          const isEditing = editingId === member.id;

          return (
            <div key={member.id} className={styles.staffCard}>
              {/* Card header */}
              <div className={styles.staffHeader}>
                <div className={styles.staffInfo}>
                  <div className={styles.staffName}>
                    {member.firstName} {member.lastName}
                  </div>
                  <div className={styles.staffMeta}>{member.position || "—"}</div>
                  <div className={styles.staffMeta}>{member.email}</div>
                  {member.phone && <div className={styles.staffMeta}>{member.phone}</div>}
                </div>
                <div className={styles.staffActions}>
                  <button
                    className={`${styles.iconBtn} ${styles.iconBtnEdit}`}
                    title="Edit staff member"
                    onClick={() => isEditing ? setEditingId(null) : openEdit(member)}
                  >✏</button>
                  <button
                    className={`${styles.iconBtn} ${styles.iconBtnDelete}`}
                    title="Delete staff member"
                    onClick={() => handleDelete(member)}
                    disabled={busy}
                  >🗑</button>
                </div>
              </div>

              {/* Inline edit form */}
              {isEditing && (
                <div className={styles.editForm}>
                  <StaffFormFields form={editForm} setForm={setEditForm} />
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
