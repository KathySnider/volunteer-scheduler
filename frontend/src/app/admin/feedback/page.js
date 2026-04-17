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
import styles from "./admin-feedback.module.css";

/* =========================================================
   GraphQL
   ========================================================= */

const FEEDBACK_QUERY = `
  query Feedback($filter: FeedbackFilterInput) {
    feedback(filter: $filter) {
      id volunteerName type status subject appPageName text
      githubIssueURL createdAt lastUpdatedAt resolvedAt
      notes { id creator noteType note createdAt }
      attachments { id filename mimeType fileSize createdAt }
    }
  }
`;

/* =========================================================
   Helpers / constants
   ========================================================= */

function formatDate(iso) {
  if (!iso) return "—";
  return new Date(iso).toLocaleDateString(undefined, {
    month: "short", day: "numeric", year: "numeric",
  });
}

const STATUS_LABEL = {
  OPEN:               "Open",
  QUESTION_SENT:      "Question Sent",
  RESOLVED_GITHUB:    "Resolved (GitHub)",
  RESOLVED_REJECTED:  "Rejected",
};

const TYPE_LABEL = {
  BUG:         "Bug",
  ENHANCEMENT: "Enhancement",
  GENERAL:     "General",
};

/* =========================================================
   Sub-components (defined at module level)
   ========================================================= */

function TypeBadge({ type }) {
  const cls = {
    BUG:         styles.typeBug,
    ENHANCEMENT: styles.typeEnhancement,
    GENERAL:     styles.typeGeneral,
  }[type] ?? styles.typeGeneral;
  return <span className={`${styles.badge} ${cls}`}>{TYPE_LABEL[type] ?? type}</span>;
}

function StatusBadge({ status }) {
  const cls = {
    OPEN:              styles.statusOpen,
    QUESTION_SENT:     styles.statusQuestion,
    RESOLVED_GITHUB:   styles.statusResolved,
    RESOLVED_REJECTED: styles.statusRejected,
  }[status] ?? styles.statusOpen;
  return <span className={`${styles.badge} ${cls}`}>{STATUS_LABEL[status] ?? status}</span>;
}

/* =========================================================
   Page
   ========================================================= */

export default function AdminFeedbackPage() {
  const router = useRouter();
  const [token, setToken]       = useState(null);
  const [gql, setGql]           = useState(null);
  const [userName, setUserName] = useState("");

  const [allFeedback, setAllFeedback] = useState([]);
  const [loading, setLoading]         = useState(true);
  const [pageError, setPageError]     = useState("");

  // Filters (client-side)
  const [statusFilter, setStatusFilter] = useState("ALL");    // ALL | OPEN | RESOLVED
  const [typeFilter, setTypeFilter]     = useState("");        // "" | BUG | ENHANCEMENT | GENERAL

  /* ----- Load data ----- */
  const loadData = useCallback((bound) => {
    setLoading(true);
    setPageError("");
    bound(FEEDBACK_QUERY, { filter: null })
      .then((res) => {
        setAllFeedback(res.data?.feedback ?? []);
        if (res.errors) setPageError(res.errors[0]?.message ?? "Error loading feedback.");
      })
      .catch(() => setPageError("Unable to reach the server."))
      .finally(() => setLoading(false));
  }, []);

  /* ----- Auth guard ----- */
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

  const handleSignOut = async () => { await signOut(token); router.replace("/login"); };

  /* ----- Client-side filtering ----- */
  const filtered = allFeedback.filter((fb) => {
    const openStatuses     = ["OPEN", "QUESTION_SENT"];
    const resolvedStatuses = ["RESOLVED_GITHUB", "RESOLVED_REJECTED"];

    if (statusFilter === "OPEN"     && !openStatuses.includes(fb.status))     return false;
    if (statusFilter === "RESOLVED" && !resolvedStatuses.includes(fb.status)) return false;
    if (typeFilter && fb.type !== typeFilter) return false;
    return true;
  });

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
          <h1 className={styles.pageTitle}>Manage Feedback</h1>
        </div>

        {/* Error banner */}
        {pageError && <div className={styles.errorBanner}>{pageError}</div>}

        {/* Filter bar */}
        <div className={styles.filterBar}>
          <div className={styles.filterGroup}>
            <label className={styles.filterLabel}>Status</label>
            <div className={styles.segmented}>
              {[
                { value: "ALL",      label: "All" },
                { value: "OPEN",     label: "Open" },
                { value: "RESOLVED", label: "Resolved" },
              ].map(({ value, label }) => (
                <button
                  key={value}
                  className={`${styles.segBtn} ${statusFilter === value ? styles.segBtnActive : ""}`}
                  onClick={() => setStatusFilter(value)}
                >
                  {label}
                </button>
              ))}
            </div>
          </div>

          <div className={styles.filterGroup}>
            <label className={styles.filterLabel}>Type</label>
            <select
              className={styles.filterSelect}
              value={typeFilter}
              onChange={(e) => setTypeFilter(e.target.value)}
            >
              <option value="">All Types</option>
              <option value="BUG">Bug</option>
              <option value="ENHANCEMENT">Enhancement</option>
              <option value="GENERAL">General</option>
            </select>
          </div>
        </div>

        {/* Loading */}
        {loading && (
          <div className={styles.stateBox}>
            <div className={styles.spinner} />
            <p>Loading feedback…</p>
          </div>
        )}

        {/* Empty state */}
        {!loading && filtered.length === 0 && (
          <div className={styles.stateBox}>
            <div className={styles.stateTitle}>No feedback found</div>
            <p>Try adjusting your filters.</p>
          </div>
        )}

        {/* Feedback list */}
        {!loading && filtered.map((fb) => (
          <button
            key={fb.id}
            className={styles.feedbackCard}
            onClick={() => router.push(`/admin/feedback/${fb.id}`)}
          >
            <div className={styles.cardTop}>
              <div className={styles.badges}>
                <TypeBadge type={fb.type} />
                <StatusBadge status={fb.status} />
              </div>
              <span className={styles.cardDate}>{formatDate(fb.createdAt)}</span>
            </div>
            <div className={styles.cardSubject}>{fb.subject || "(no subject)"}</div>
            <div className={styles.cardMeta}>
              <span className={styles.metaVolunteer}>{fb.volunteerName}</span>
              {fb.appPageName && (
                <span className={styles.metaPage}>Page: {fb.appPageName}</span>
              )}
            </div>
          </button>
        ))}
      </div>
    </div>
  );
}
