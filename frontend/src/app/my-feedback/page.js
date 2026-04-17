"use client";

import { useEffect, useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import {
  getAuthToken,
  getAuthName,
  signOut,
  volunteerGql,
} from "../lib/api";
import UserMenu from "../components/UserMenu";
import styles from "./my-feedback.module.css";

/* ----- GraphQL ----- */

const OWN_FEEDBACK = `
  query {
    ownFeedback {
      id
      type
      status
      subject
      appPageName
      text
      githubIssueURL
      createdAt
      notes { id noteType note createdAt }
      attachments { id filename mimeType fileSize createdAt }
    }
  }
`;

/* ----- Helpers ----- */

function formatDate(isoString) {
  if (!isoString) return "";
  return new Date(isoString).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

/* ----- Badges ----- */

const TYPE_LABELS = {
  BUG: "Bug Report",
  ENHANCEMENT: "Enhancement",
  GENERAL: "General",
};

const STATUS_LABELS = {
  OPEN: "Open",
  QUESTION_SENT: "Question Sent",
  RESOLVED_GITHUB: "Resolved",
  RESOLVED_REJECTED: "Closed",
};

function TypeBadge({ type }) {
  const cls = {
    BUG: styles.typeBug,
    ENHANCEMENT: styles.typeEnhancement,
    GENERAL: styles.typeGeneral,
  }[type] ?? "";
  return (
    <span className={`${styles.badge} ${cls}`}>
      {TYPE_LABELS[type] ?? type}
    </span>
  );
}

function StatusBadge({ status }) {
  const cls = {
    OPEN: styles.statusOpen,
    QUESTION_SENT: styles.statusQuestion,
    RESOLVED_GITHUB: styles.statusResolved,
    RESOLVED_REJECTED: styles.statusRejected,
  }[status] ?? "";
  return (
    <span className={`${styles.badge} ${cls}`}>
      {STATUS_LABELS[status] ?? status}
    </span>
  );
}

/* ----- Feedback card ----- */

function FeedbackCard({ item, onClick }) {
  return (
    <button className={styles.card} onClick={onClick}>
      <div className={styles.cardTop}>
        <div className={styles.cardBadges}>
          <TypeBadge type={item.type} />
          <StatusBadge status={item.status} />
        </div>
        <span className={styles.cardDate}>{formatDate(item.createdAt)}</span>
      </div>
      <div className={styles.cardSubject}>{item.subject}</div>
      <div className={styles.cardPage}>{item.appPageName}</div>
    </button>
  );
}

/* ----- Page ----- */

export default function MyFeedbackPage() {
  const router = useRouter();
  const [token, setToken] = useState(null);
  const [gql, setGql] = useState(null);
  const [userName, setUserName] = useState("");
  const [feedback, setFeedback] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const loadData = useCallback(async (boundGql) => {
    setLoading(true);
    setError("");
    try {
      const res = await boundGql(OWN_FEEDBACK, null);
      if (res.errors) {
        setError(res.errors[0]?.message ?? "Error loading feedback.");
      } else {
        const items = res.data?.ownFeedback ?? [];
        items.sort((a, b) => (b.createdAt > a.createdAt ? 1 : -1));
        setFeedback(items);
      }
    } catch {
      setError("Unable to reach the server. Please try again.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    const t = getAuthToken();
    if (!t) { router.replace("/login"); return; }
    const bound = (q, v) => volunteerGql(q, v, t);
    setToken(t);
    setGql(() => bound);
    setUserName(getAuthName() ?? "");
    loadData(bound);
  }, [router, loadData]);

  const handleSignOut = async () => { await signOut(token); router.replace("/login"); };

  if (!gql) return null;

  return (
    <div className={styles.page}>
      {/* Top bar */}
      <div className={styles.topBar}>
        <div className={styles.topBarLeft}>
          <a href="/events" className={styles.backLink}>&#8592; Back to Events</a>
        </div>
        <div className={styles.topBarTitle}>My Feedback</div>
        <div className={styles.topBarRight}>
          <UserMenu name={userName} isAdmin={false} onSignOut={handleSignOut} />
        </div>
      </div>

      <main className={styles.main}>
        <h1 className={styles.pageTitle}>My Feedback</h1>

        {error && <div className={styles.errorBox}>{error}</div>}

        {loading && (
          <div className={styles.stateBox}>
            <div className={styles.spinner} />
            <p>Loading feedback&hellip;</p>
          </div>
        )}

        {!loading && !error && feedback.length === 0 && (
          <div className={styles.emptyState}>
            <div className={styles.emptyIcon}>&#128172;</div>
            <div className={styles.emptyTitle}>No feedback submitted yet.</div>
            <p className={styles.emptyText}>
              Use the feedback button on any page to share your thoughts.
            </p>
          </div>
        )}

        {!loading && feedback.length > 0 && (
          <div className={styles.cardList}>
            {feedback.map((item) => (
              <FeedbackCard
                key={item.id}
                item={item}
                onClick={() => router.push(`/my-feedback/${item.id}`)}
              />
            ))}
          </div>
        )}
      </main>
    </div>
  );
}
