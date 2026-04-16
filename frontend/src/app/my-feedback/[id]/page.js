"use client";

import { useEffect, useState, useCallback } from "react";
import { useRouter, useParams } from "next/navigation";
import {
  getAuthToken,
  getAuthName,
  clearAuthToken,
  volunteerGql,
} from "../../lib/api";
import UserMenu from "../../components/UserMenu";
import styles from "./my-feedback-detail.module.css";

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

function formatDateTime(isoString) {
  if (!isoString) return "";
  return new Date(isoString).toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
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

/* ----- Note item ----- */

const NOTE_CONFIG = {
  QUESTION: {
    label: "Admin asked:",
    cls: "noteQuestion",
  },
  VOLUNTEER_REPLY: {
    label: "Your reply:",
    cls: "noteReply",
  },
  EMAIL_TO_VOLUNTEER: {
    label: "Message from admin:",
    cls: "noteEmail",
  },
};

function NoteItem({ note }) {
  const config = NOTE_CONFIG[note.noteType] ?? { label: note.noteType, cls: "noteEmail" };
  return (
    <div className={`${styles.noteItem} ${styles[config.cls]}`}>
      <div className={styles.noteLabel}>{config.label}</div>
      <div className={styles.noteText}>{note.note}</div>
      <div className={styles.noteDate}>{formatDateTime(note.createdAt)}</div>
    </div>
  );
}

/* ----- Page ----- */

export default function FeedbackDetailPage() {
  const router = useRouter();
  const { id } = useParams();
  const [gql, setGql] = useState(null);
  const [userName, setUserName] = useState("");
  const [item, setItem] = useState(null);
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
        const all = res.data?.ownFeedback ?? [];
        const found = all.find((f) => String(f.id) === String(id));
        if (found) {
          setItem(found);
        } else {
          setError("Feedback item not found.");
        }
      }
    } catch {
      setError("Unable to reach the server. Please try again.");
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    const t = getAuthToken();
    if (!t) { router.replace("/login"); return; }
    const bound = (q, v) => volunteerGql(q, v, t);
    setGql(() => bound);
    setUserName(getAuthName() ?? "");
    loadData(bound);
  }, [router, loadData]);

  const handleSignOut = () => {
    clearAuthToken();
    router.replace("/login");
  };

  if (!gql) return null;

  return (
    <div className={styles.page}>
      {/* Top bar */}
      <div className={styles.topBar}>
        <div className={styles.topBarLeft}>
          <a href="/my-feedback" className={styles.backLink}>&#8592; My Feedback</a>
        </div>
        <div className={styles.topBarTitle}>Feedback Detail</div>
        <div className={styles.topBarRight}>
          <UserMenu name={userName} isAdmin={false} onSignOut={handleSignOut} />
        </div>
      </div>

      <main className={styles.main}>
        {error && <div className={styles.errorBox}>{error}</div>}

        {loading && (
          <div className={styles.stateBox}>
            <div className={styles.spinner} />
            <p>Loading&hellip;</p>
          </div>
        )}

        {!loading && item && (
          <>
            {/* Main feedback card */}
            <div className={styles.card}>
              <div className={styles.cardHeader}>
                <div className={styles.cardBadges}>
                  <TypeBadge type={item.type} />
                  <StatusBadge status={item.status} />
                </div>
                <span className={styles.cardDate}>{formatDate(item.createdAt)}</span>
              </div>

              <h1 className={styles.cardSubject}>{item.subject}</h1>

              <div className={styles.cardMeta}>
                Submitted from: <span className={styles.cardMetaValue}>{item.appPageName}</span>
              </div>

              <div className={styles.cardText}>{item.text}</div>

              {item.githubIssueURL && (
                <div className={styles.githubLink}>
                  <a
                    href={item.githubIssueURL}
                    target="_blank"
                    rel="noopener noreferrer"
                    className={styles.githubAnchor}
                  >
                    &#128279; View GitHub Issue
                  </a>
                </div>
              )}
            </div>

            {/* Question-sent notice */}
            {item.status === "QUESTION_SENT" && (
              <div className={styles.questionNotice}>
                The admin has a question &mdash; a reply feature is coming soon.
              </div>
            )}

            {/* Notes thread */}
            {item.notes && item.notes.length > 0 && (
              <div className={styles.notesSection}>
                <h2 className={styles.notesTitle}>Thread</h2>
                <div className={styles.notesList}>
                  {item.notes.map((note) => (
                    <NoteItem key={note.id} note={note} />
                  ))}
                </div>
              </div>
            )}
          </>
        )}
      </main>
    </div>
  );
}
