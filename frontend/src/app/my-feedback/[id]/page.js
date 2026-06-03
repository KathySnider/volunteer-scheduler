"use client";

import { useEffect, useState, useCallback } from "react";
import { useRouter, useParams } from "next/navigation";
import {
  isAuthenticated,
  getAuthName,
  hasAuthRole,
  Roles,
  signOut,
  volunteerGql,
  downloadAttachment,
} from "../../lib/api";
import AdminTopBar from "../../components/AdminTopBar";
import FeedbackButton from "../../components/FeedbackButton";
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
      createdAt
      notes { id noteType note createdAt }
      attachments { id filename mimeType fileSize createdAt }
    }
  }
`;

const ADD_VOLUNTEER_NOTE = `
  mutation AddVolunteerFeedbackNote($input: VolunteerFeedbackNoteInput!) {
    addVolunteerFeedbackNote(note: $input) { success message }
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
  BUG:         "Bug Report",
  ENHANCEMENT: "Enhancement",
  GENERAL:     "General",
};

const STATUS_LABELS = {
  OPEN:                 "Open",
  QUESTION_SENT:        "Question Sent",
  RESOLVED_IMPLEMENTED: "Resolved",
  RESOLVED_REJECTED:    "Closed",
};

function TypeBadge({ type }) {
  const cls = {
    BUG:         styles.typeBug,
    ENHANCEMENT: styles.typeEnhancement,
    GENERAL:     styles.typeGeneral,
  }[type] ?? "";
  return (
    <span className={`${styles.badge} ${cls}`}>
      {TYPE_LABELS[type] ?? type}
    </span>
  );
}

function StatusBadge({ status }) {
  const cls = {
    OPEN:                 styles.statusOpen,
    QUESTION_SENT:        styles.statusQuestion,
    RESOLVED_IMPLEMENTED: styles.statusResolved,
    RESOLVED_REJECTED:    styles.statusRejected,
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
    cls:   "noteQuestion",
  },
  VOLUNTEER_NOTE: {
    label: "Your note:",
    cls:   "noteReply",
  },
  EMAIL_TO_VOLUNTEER: {
    label: "Message from admin:",
    cls:   "noteEmail",
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
  const [gql, setGql]           = useState(null);
  const [userName, setUserName] = useState("");
  const [isAdmin, setIsAdmin]   = useState(false);
  const [feedbackOpen, setFeedbackOpen] = useState(false);

  const [item, setItem]         = useState(null);
  const [loading, setLoading]   = useState(true);
  const [error, setError]       = useState("");
  const [successMsg, setSuccessMsg] = useState("");
  const [downloadingId, setDownloadingId] = useState(null);

  const [noteText, setNoteText]   = useState("");
  const [noteBusy, setNoteBusy]   = useState(false);
  const [noteError, setNoteError] = useState("");

  const handleDownload = useCallback(async (attachmentId) => {
    setDownloadingId(attachmentId);
    try {
      await downloadAttachment(attachmentId, false);
    } catch {
      // non-fatal
    } finally {
      setDownloadingId(null);
    }
  }, []);

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
    if (!isAuthenticated()) { router.replace("/login"); return; }
    const bound = volunteerGql;
    setGql(() => bound);
    setUserName(getAuthName() ?? "");
    setIsAdmin(hasAuthRole(Roles.ADMINISTRATOR));
    loadData(bound);
  }, [router, loadData]);

  const handleSignOut = async () => { await signOut(); router.replace("/login"); };

  const handleAddNote = async () => {
    if (!noteText.trim()) { setNoteError("Note text is required."); return; }
    setNoteBusy(true);
    setNoteError("");
    setSuccessMsg("");
    try {
      const res = await gql(ADD_VOLUNTEER_NOTE, {
        input: { feedbackId: parseInt(id, 10), note: noteText.trim() },
      });
      const result = res.data?.addVolunteerFeedbackNote;
      if (res.errors || !result?.success) {
        setNoteError(result?.message ?? res.errors?.[0]?.message ?? "Failed to add note.");
      } else {
        setNoteText("");
        setSuccessMsg("Your note was added.");
        loadData(gql);
      }
    } catch {
      setNoteError("Unable to reach the server.");
    } finally {
      setNoteBusy(false);
    }
  };

  if (!gql) return null;

  const isResolved = item
    ? ["RESOLVED_IMPLEMENTED", "RESOLVED_REJECTED"].includes(item.status)
    : false;

  return (
    <div className={styles.page}>
      <AdminTopBar userName={userName} isAdmin={isAdmin} onSignOut={handleSignOut} onFeedbackOpen={() => setFeedbackOpen(true)} />

      <main className={styles.main}>
        {successMsg && <div className={styles.successBox}>{successMsg}</div>}
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
            </div>

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

            {/* Add a note — not shown when resolved */}
            {!isResolved && (
              <div className={styles.addNoteSection}>
                <h2 className={styles.addNoteTitle}>
                  {item.status === "QUESTION_SENT"
                    ? "Reply to admin's question"
                    : "Add a note"}
                </h2>
                {noteError && <div className={styles.noteErrorBox}>{noteError}</div>}
                <textarea
                  className={styles.noteTextarea}
                  value={noteText}
                  onChange={(e) => setNoteText(e.target.value)}
                  placeholder="Add context, a reply, or any additional information…"
                  rows={4}
                />
                <div className={styles.noteActions}>
                  <button
                    className={styles.noteSubmitBtn}
                    onClick={handleAddNote}
                    disabled={noteBusy}
                  >
                    {noteBusy
                      ? "Submitting…"
                      : item.status === "QUESTION_SENT"
                        ? "Submit Reply"
                        : "Add Note"}
                  </button>
                </div>
              </div>
            )}

            {/* Attachments */}
            {item.attachments && item.attachments.length > 0 && (
              <div className={styles.card}>
                <h2 className={styles.attachTitle}>Attachments</h2>
                <ul className={styles.attachList}>
                  {item.attachments.map((att) => (
                    <li key={att.id} className={styles.attachItem}>
                      <div className={styles.attachInfo}>
                        <span className={styles.attachName}>{att.filename}</span>
                        <span className={styles.attachMeta}>
                          {Math.round(att.fileSize / 1024)} KB
                        </span>
                      </div>
                      <button
                        className={styles.attachDownload}
                        onClick={() => handleDownload(att.id)}
                        disabled={downloadingId === att.id}
                      >
                        {downloadingId === att.id ? "Downloading…" : "Download"}
                      </button>
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </>
        )}
      </main>

      <FeedbackButton open={feedbackOpen} onClose={() => setFeedbackOpen(false)} />
    </div>
  );
}
