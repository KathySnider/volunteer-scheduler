"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import {
  getAuthToken,
  getAuthRole,
  getAuthName,
  signOut,
  volunteerGql,
  setAuthToken,
} from "../lib/api";
import UserMenu from "../components/UserMenu";
import styles from "./profile.module.css";

/* =========================================================
   GraphQL
   ========================================================= */

const GET_PROFILE = `
  query {
    volunteerProfile {
      firstName
      lastName
      email
      phone
      zipCode
      role
    }
  }
`;

const UPDATE_PROFILE = `
  mutation UpdateOwnProfile($profile: UpdateOwnProfileInput!) {
    updateOwnProfile(profile: $profile) {
      success
      message
    }
  }
`;

/* =========================================================
   Page
   ========================================================= */

const EMPTY_FORM = { firstName: "", lastName: "", email: "", phone: "", zipCode: "" };

export default function ProfilePage() {
  const router = useRouter();
  const [gql, setGql]           = useState(null);
  const [token, setToken]       = useState(null);
  const [userName, setUserName] = useState("");
  const [isAdmin, setIsAdmin]   = useState(false);

  const [form, setForm]         = useState(EMPTY_FORM);
  const [loading, setLoading]   = useState(true);
  const [saving, setSaving]     = useState(false);
  const [actionMsg, setActionMsg] = useState(null); // { type: "success"|"error", text }

  /* ----- Auth + load ----- */
  useEffect(() => {
    const t = getAuthToken();
    if (!t) { router.replace("/login"); return; }
    const role = getAuthRole();
    const bound = (q, v) => volunteerGql(q, v, t);
    setGql(() => bound);
    setToken(t);
    setIsAdmin(role === "ADMINISTRATOR");
    setUserName(getAuthName() ?? "");

    bound(GET_PROFILE, null)
      .then((res) => {
        const p = res.data?.volunteerProfile;
        if (p) {
          setForm({
            firstName: p.firstName ?? "",
            lastName:  p.lastName  ?? "",
            email:     p.email     ?? "",
            phone:     p.phone     ?? "",
            zipCode:   p.zipCode   ?? "",
          });
        }
        if (res.errors) {
          setActionMsg({ type: "error", text: res.errors[0]?.message ?? "Error loading profile." });
        }
      })
      .catch(() => setActionMsg({ type: "error", text: "Unable to reach the server." }))
      .finally(() => setLoading(false));
  }, [router]);

  /* ----- Save ----- */
  const handleSave = async (e) => {
    e.preventDefault();
    if (!form.firstName.trim() || !form.lastName.trim() || !form.email.trim()) {
      setActionMsg({ type: "error", text: "First name, last name, and email are required." });
      return;
    }

    setSaving(true);
    setActionMsg(null);
    try {
      const res = await gql(UPDATE_PROFILE, {
        profile: {
          firstName: form.firstName.trim(),
          lastName:  form.lastName.trim(),
          email:     form.email.trim(),
          phone:     form.phone.trim()   || null,
          zipCode:   form.zipCode.trim() || null,
        },
      });
      const result = res.data?.updateOwnProfile;
      if (res.errors || !result?.success) {
        setActionMsg({ type: "error", text: result?.message ?? res.errors?.[0]?.message ?? "Update failed." });
        return;
      }

      // Keep the display name in localStorage in sync
      const newName = `${form.firstName.trim()} ${form.lastName.trim()}`.trim();
      setAuthToken(token, form.email.trim(), getAuthRole(), newName);
      setUserName(newName);

      setActionMsg({ type: "success", text: "Profile updated." });
    } catch (err) {
      // 401 means the session expired — send the user back to sign in.
      if (err?.message?.includes("401")) {
        router.replace("/login");
        return;
      }
      setActionMsg({ type: "error", text: "Unable to reach the server. Please try again." });
    } finally {
      setSaving(false);
    }
  };

  const handleSignOut = async () => { await signOut(token); router.replace("/login"); };

  if (!gql) return null;

  return (
    <div className={styles.page}>
      {/* Top bar */}
      <div className={styles.topBar}>
        <a href="/events" className={styles.backLink}>← Back to Events</a>
        <UserMenu name={userName} isAdmin={isAdmin} onSignOut={handleSignOut} />
      </div>

      <div className={styles.content}>
        <h1 className={styles.pageTitle}>My Profile</h1>

        {/* Banners */}
        {actionMsg?.type === "success" && (
          <div className={styles.successBanner}>{actionMsg.text}</div>
        )}
        {actionMsg?.type === "error" && (
          <div className={styles.errorBanner}>{actionMsg.text}</div>
        )}

        {loading ? (
          <div className={styles.stateBox}>
            <div className={styles.spinner} />
            <p>Loading profile...</p>
          </div>
        ) : (
          <form className={styles.card} onSubmit={handleSave}>
            <div className={styles.grid2}>
              <div className={styles.field}>
                <label className={styles.label} htmlFor="firstName">
                  First Name <span className={styles.required}>*</span>
                </label>
                <input
                  id="firstName"
                  className={styles.input}
                  value={form.firstName}
                  onChange={(e) => setForm((p) => ({ ...p, firstName: e.target.value }))}
                  required
                />
              </div>

              <div className={styles.field}>
                <label className={styles.label} htmlFor="lastName">
                  Last Name <span className={styles.required}>*</span>
                </label>
                <input
                  id="lastName"
                  className={styles.input}
                  value={form.lastName}
                  onChange={(e) => setForm((p) => ({ ...p, lastName: e.target.value }))}
                  required
                />
              </div>
            </div>

            <div className={styles.field}>
              <label className={styles.label} htmlFor="email">
                Email Address <span className={styles.required}>*</span>
              </label>
              <input
                id="email"
                className={styles.input}
                type="email"
                value={form.email}
                onChange={(e) => setForm((p) => ({ ...p, email: e.target.value }))}
                required
              />
            </div>

            <div className={styles.grid2}>
              <div className={styles.field}>
                <label className={styles.label} htmlFor="phone">
                  Phone
                </label>
                <input
                  id="phone"
                  className={styles.input}
                  type="tel"
                  value={form.phone}
                  placeholder="(555) 555-5555"
                  onFocus={(e) => { const n = e.target.value.length; e.target.setSelectionRange(n, n); }}
                  onChange={(e) => setForm((p) => ({ ...p, phone: e.target.value }))}
                />
              </div>

              <div className={styles.field}>
                <label className={styles.label} htmlFor="zipCode">
                  Zip Code
                </label>
                <input
                  id="zipCode"
                  className={styles.input}
                  value={form.zipCode}
                  placeholder="12345"
                  onChange={(e) => setForm((p) => ({ ...p, zipCode: e.target.value }))}
                />
              </div>
            </div>

            <div className={styles.formActions}>
              <button className={styles.btnPrimary} type="submit" disabled={saving}>
                {saving ? "Saving..." : "Save Changes"}
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  );
}
