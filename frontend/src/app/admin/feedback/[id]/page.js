"use client";

import { useEffect, useState, useCallback } from "react";
import { useRouter, useParams } from "next/navigation";
import {
  isAuthenticated,
  hasAuthRole,
  Roles,
  getAuthName,
  signOut,
  adminGql,
  downloadAttachment,
} from "../../../lib/api";
import AdminTopBar from "../../../components/AdminTopBar";
import FeedbackButton from "../../../components/FeedbackButton";
import styles from "./admin-feedback-detail.module.css";

/* =========================================================
   GraphQL
   ========================================================= */

const FEEDBACK_DETAIL = `
  query FeedbackDetail($id: ID!) {
    feedbackDetail(feedbackId: $id) {
      id volunteerName type status subject appPageName text
      createdAt lastUpdatedAt resolvedAt
      notes { id creator noteType note createdAt }
      attachments { id filename mimeType fileSize createdAt }
    }
  }
`;

const UPDATE_FEEDBACK_STATUS = `
  mutation UpdateFeedbackStatus($input: FeedbackStatusUpdateInput!) {
    updateFeedbackStatus(su: $input) { success message }
  }
`;

const ADD_FEEDBACK_NOTE = `
  mutation AddFeedbackNote($input: FeedbackNoteInput!) {
    addFeedbackNote(note: $input) { success message }
  }
`;

const EMAIL_FEEDBACK_SUBMITTER = `
  mutation EmailFeedbackSubmitter($input: FeedbackEmailInput!) {
    emailFeedbackSubmitter(input: $input) { success message }
  }
`;

/* =========================================================
   Helpers / constants
   ========================================================= */

function formatDate(iso) {
  if (!iso) return "—";
  return new Date(iso).toLocaleString(undefined, {
    month: "short", day: "numeric", year: "numeric",
    hour: "numeric", minute: "2-digit",
  });
}

function formatDateShort(iso) {
  if (!iso) return "—";
  return new Date(iso).toLocaleDateString(undefined, {
    month: "short", day: "numeric", year: "numeric",
  });
}

const STATUS_LABEL = {
  OPEN:                 "Open",
  QUESTION_SENT:        "Question Sent",
  RESOLVED_IMPLEMENTED: "Resolved",
  RESOLVED_REJECTED:    "Rejected",
};

const TYPE_LABEL = {
  BUG:         "Bug",
  ENHANCEMENT: "Enhancement",
  GENERAL:     "General",
};

const NOTE_TYPE_LABEL = {
  ADMIN_NOTE:          "Admin note (internal):",
  QUESTION:            "Question sent to volunteer:",
  VOLUNTEER_NOTE:      "Volunteer note:",
  EMAIL_TO_VOLUNTEER:  "Email to volunteer:",
};

// QUESTION_SENT is set automatically by emailFeedbackSubmitter — not selectable here.
const UPDATABLE_STATUSES = [
  { value: "OPEN",                  label: "Open" },
  { value: "RESOLVED_IMPLEMENTED",  label: "Resolved" },
  { value: "RESOLVED_REJECTED",     label: "Rejected" },
];

/* =========================================================
   Sub-components
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
    OPEN:                 styles.statusOpen,
    QUESTION_SENT:        styles.statusQuestion,
    RESOLVED_IMPLEMENTED: styles.statusResolved,
    RESOLVED_REJECTED:    styles.statusRejected,
  }[status] ?? styles.statusOpen;
  return <span className={`${styles.badge} ${cls}`}>{STATUS_LABEL[status] ?? status}</span>;
}

function NoteItem({ note }) {
  const cls = {
    ADMIN_NOTE:         styles.noteAdmin,
    QUESTION:           styles.noteQuestion,
    VOLUNTEER_NOTE:     styles.noteVolunteer,
    EMAIL_TO_VOLUNTEER: styles.noteEmail,
  }[note.noteType] ?? styles.noteAdmin;

  return (
    <div className={`${styles.noteItem} ${cls}`}>
      <div className={styles.noteHeader}>
        <span className={styles.noteTypeLabel}>
          {NOTE_TYPE_LABEL[note.noteType] ?? note.noteType}
        </span>
        <span className={styles.noteMeta}>
          {note.creator} · {formatDate(note.createdAt)}
        </span>
      </div>
      <div className={styles.noteBody}>{note.note}</div>
    </div>
  );
}

function TextField({ label, value, onChange, required, placeholder, rows = 4 }) {
  return (
    <div className={styles.field}>
      <label className={styles.label}>
        {label}
        {required && <span className={styles.required}> *</span>}
      </label>
      <textarea
        className={styles.textarea}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        rows={rows}
      />
    </div>
  );
}

/* =========================================================
   Accordion wrapper
   ========================================================= */

function AccordionPanel({ title, isOpen, onToggle, children }) {
  return (
    <div className={styles.accordionPanel}>
      <button
        className={styles.accordionHeader}
        onClick={onToggle}
        aria-expanded={isOpen}
      >
        <span className={styles.accordionTitle}>{title}</span>
        <span className={`${styles.accordionChevron} ${isOpen ? styles.chevronOpen : ""}`}>▼</span>
      </button>
      {isOpen && (
        <div className={styles.accordionBody}>
          {children}
        </div>
      )}
    </div>
  );
}

/* =========================================================
   Add Internal Note Panel
   ========================================================= */

function AddNotePanel({ feedbackId, gql, onSuccess, onError }) {
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);

  const handleSubmit = async () => {
    if (!note.trim()) { onError("Note text is required."); return; }
    setBusy(true);
    try {
      const res = await gql(ADD_FEEDBACK_NOTE, {
        input: { feedbackId: parseInt(feedbackId, 10), note: note.trim() },
      });
      const result = res.data?.addFeedbackNote;
      if (res.errors || !result?.success) {
        onError(result?.message ?? res.errors?.[0]?.message ?? "Failed to add note.");
      } else {
        setNote("");
        onSuccess("Internal note added.");
      }
    } catch {
      onError("Unable to reach the server.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <>
      <TextField
        label="Note"
        value={note}
        onChange={setNote}
        required
        placeholder="Private note — not visible to the volunteer"
        rows={3}
      />
      <div className={styles.panelActions}>
        <button className={styles.btnPrimary} onClick={handleSubmit} disabled={busy}>
          {busy ? "Saving…" : "Add Note"}
        </button>
      </div>
    </>
  );
}

/* =========================================================
   Change Status Panel
   ========================================================= */

function ChangeStatusPanel({ feedbackId, currentStatus, gql, onSuccess, onError }) {
  const [status, setStatus] = useState(
    UPDATABLE_STATUSES.some((s) => s.value === currentStatus) ? currentStatus : "OPEN"
  );
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);

  const handleSubmit = async () => {
    if (!note.trim()) { onError("A note is required when changing status."); return; }
    setBusy(true);
    try {
      const res = await gql(UPDATE_FEEDBACK_STATUS, {
        input: {
          feedbackId: parseInt(feedbackId, 10),
          status,
          note: note.trim(),
        },
      });
      const result = res.data?.updateFeedbackStatus;
      if (res.errors || !result?.success) {
        onError(result?.message ?? res.errors?.[0]?.message ?? "Failed to update status.");
      } else {
        setNote("");
        onSuccess("Status updated.");
      }
    } catch {
      onError("Unable to reach the server.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <>
      <div className={styles.field}>
        <label className={styles.label}>New Status</label>
        <select
          className={styles.select}
          value={status}
          onChange={(e) => setStatus(e.target.value)}
        >
          {UPDATABLE_STATUSES.map(({ value, label }) => (
            <option key={value} value={value}>{label}</option>
          ))}
        </select>
      </div>
      <TextField
        label="Note (required)"
        value={note}
        onChange={setNote}
        required
        placeholder="Explain the status change"
        rows={3}
      />
      <div className={styles.panelActions}>
        <button className={styles.btnPrimary} onClick={handleSubmit} disabled={busy}>
          {busy ? "Saving…" : "Update Status"}
        </button>
      </div>
    </>
  );
}

/* =========================================================
   Email Volunteer Panel
   ========================================================= */

function EmailPanel({ feedbackId, gql, onSuccess, onError }) {
  const [emailText, setEmailText]     = useState("");
  const [requireReply, setRequireReply] = useState(false);
  const [busy, setBusy]               = useState(false);

  const handleSubmit = async () => {
    if (!emailText.trim()) { onError("Email text is required."); return; }
    setBusy(true);
    try {
      const res = await gql(EMAIL_FEEDBACK_SUBMITTER, {
        input: {
          feedbackId: parseInt(feedbackId, 10),
          emailText: emailText.trim(),
          requireReply,
        },
      });
      const result = res.data?.emailFeedbackSubmitter;
      if (res.errors || !result?.success) {
        onError(result?.message ?? res.errors?.[0]?.message ?? "Failed to send email.");
      } else {
        setEmailText("");
        setRequireReply(false);
        onSuccess(requireReply ? "Question sent to volunteer." : "Email sent to volunteer.");
      }
    } catch {
      onError("Unable to reach the server.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <>
      <TextField
        label="Email text"
        value={emailText}
        onChange={setEmailText}
        required
        placeholder="What would you like to say to the volunteer?"
      />
      <div className={styles.field}>
        <label className={styles.checkboxLabel}>
          <input
            type="checkbox"
            checked={requireReply}
            onChange={(e) => setRequireReply(e.target.checked)}
          />
          I need a reply before work can continue
        </label>
        <p className={styles.checkboxHint}>
          {requireReply
            ? "Status will change to “Question Sent” — work is paused until the volunteer responds."
            : "Leave unchecked to send informational email without pausing work on this feedback."}
        </p>
      </div>
      <div className={styles.panelActions}>
        <button className={styles.btnPrimary} onClick={handleSubmit} disabled={busy}>
          {busy ? "Sending…" : "Send Email"}
        </button>
      </div>
    </>
  );
}

/* =========================================================
   Page
   ========================================================= */

export default function AdminFeedbackDetailPage() {
  const router   = useRouter();
  const params   = useParams();
  const id       = params?.id;
  const [gql, setGql]           = useState(null);
  const [userName, setUserName] = useState("");
  const [feedbackOpen, setFeedbackOpen] = useState(false);

  const [feedback, setFeedback]   = useState(null);
  const [loading, setLoading]     = useState(true);
  const [pageError, setPageError] = useState("");
  const [actionMsg, setActionMsg] = useState(null);
  const [downloadingId, setDownloadingId] = useState(null);
  const [openPanel, setOpenPanel] = useState(null);

  const togglePanel = (name) => setOpenPanel((prev) => (prev === name ? null : name));

  const handleDownload = useCallback(async (attachmentId) => {
    setDownloadingId(attachmentId);
    try {
      await downloadAttachment(attachmentId, true);
    } catch {
      // non-fatal
    } finally {
      setDownloadingId(null);
    }
  }, []);

  /* ----- Load detail ----- */
  const loadData = useCallback((bound, feedbackId) => {
    setLoading(true);
    setPageError("");
    bound(FEEDBACK_DETAIL, { id: feedbackId })
      .then((res) => {
        setFeedback(res.data?.feedbackDetail ?? null);
        if (res.errors) setPageError(res.errors[0]?.message ?? "Error loading feedback.");
        if (!res.data?.feedbackDetail && !res.errors) setPageError("Feedback not found.");
      })
      .catch(() => setPageError("Unable to reach the server."))
      .finally(() => setLoading(false));
  }, []);

  /* ----- Auth guard ----- */
  useEffect(() => {
    if (!isAuthenticated()) { router.replace("/login"); return; }
    if (!hasAuthRole(Roles.ADMINISTRATOR)) { router.replace("/events"); return; }
    const bound = adminGql;
    setGql(() => bound);
    setUserName(getAuthName() ?? "");
    if (id) loadData(bound, id);
  }, [router, loadData, id]);

  const handleSignOut = async () => { await signOut(); router.replace("/login"); };

  const handleSuccess = (msg) => {
    setActionMsg({ type: "success", text: msg });
    if (gql && id) loadData(gql, id);
  };

  const handleError = (msg) => {
    setActionMsg({ type: "error", text: msg });
    window.scrollTo({ top: 0, behavior: "smooth" });
  };

  if (!gql) return null;

  const isResolved = feedback
    ? ["RESOLVED_IMPLEMENTED", "RESOLVED_REJECTED"].includes(feedback.status)
    : false;

  return (
    <div className={styles.page}>
      <AdminTopBar userName={userName} isAdmin={true} onSignOut={handleSignOut} onFeedbackOpen={() => setFeedbackOpen(true)} />

      <div className={styles.content}>
        {/* Banners */}
        {actionMsg?.type === "success" && (
          <div className={styles.successBanner}>{actionMsg.text}</div>
        )}
        {(actionMsg?.type === "error" || pageError) && (
          <div className={styles.errorBanner}>{actionMsg?.text ?? pageError}</div>
        )}

        {/* Loading */}
        {loading && (
          <div className={styles.stateBox}>
            <div className={styles.spinner} />
            <p>Loading feedback…</p>
          </div>
        )}

        {!loading && feedback && (
          <>
            {/* Header card */}
            <div className={styles.headerCard}>
              <div className={styles.headerTop}>
                <div className={styles.badges}>
                  <TypeBadge type={feedback.type} />
                  <StatusBadge status={feedback.status} />
                </div>
              </div>
              <h1 className={styles.subject}>{feedback.subject || "(no subject)"}</h1>
              <div className={styles.headerMeta}>
                <span><strong>From:</strong> {feedback.volunteerName}</span>
                <span><strong>Submitted:</strong> {formatDateShort(feedback.createdAt)}</span>
                {feedback.lastUpdatedAt && (
                  <span><strong>Last updated:</strong> {formatDateShort(feedback.lastUpdatedAt)}</span>
                )}
                {feedback.resolvedAt && (
                  <span><strong>Resolved:</strong> {formatDateShort(feedback.resolvedAt)}</span>
                )}
                {feedback.appPageName && (
                  <span><strong>Page:</strong> {feedback.appPageName}</span>
                )}
              </div>
            </div>

            {/* Original feedback text */}
            <div className={styles.card}>
              <h2 className={styles.cardTitle}>Original Feedback</h2>
              <p className={styles.feedbackText}>{feedback.text}</p>
            </div>

            {/* Activity thread */}
            {feedback.notes && feedback.notes.length > 0 && (
              <div className={styles.card}>
                <h2 className={styles.cardTitle}>Activity Thread</h2>
                <div className={styles.notesList}>
                  {[...feedback.notes]
                    .sort((a, b) => new Date(a.createdAt) - new Date(b.createdAt))
                    .map((note) => (
                      <NoteItem key={note.id} note={note} />
                    ))}
                </div>
              </div>
            )}

            {/* Attachments */}
            {feedback.attachments && feedback.attachments.length > 0 && (
              <div className={styles.card}>
                <h2 className={styles.cardTitle}>Attachments</h2>
                <ul className={styles.attachmentList}>
                  {feedback.attachments.map((att) => (
                    <li key={att.id} className={styles.attachmentItem}>
                      <div className={styles.attachmentInfo}>
                        <span className={styles.attachmentName}>{att.filename}</span>
                        <span className={styles.attachmentMeta}>
                          {att.mimeType} · {Math.round(att.fileSize / 1024)} KB
                        </span>
                      </div>
                      <button
                        className={styles.attachmentDownload}
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

            {/* Action panels */}
            {isResolved && (
              <div className={styles.resolvedNote}>
                This feedback has been resolved.
              </div>
            )}

            {/* Add internal note is always available */}
            <AccordionPanel
              title="Add Internal Note"
              isOpen={openPanel === "note"}
              onToggle={() => togglePanel("note")}
            >
              <AddNotePanel
                feedbackId={feedback.id}
                gql={gql}
                onSuccess={handleSuccess}
                onError={handleError}
              />
            </AccordionPanel>

            {/* Status and email actions only for open feedback */}
            {!isResolved && (
              <>
                <AccordionPanel
                  title="Change Status"
                  isOpen={openPanel === "status"}
                  onToggle={() => togglePanel("status")}
                >
                  <ChangeStatusPanel
                    feedbackId={feedback.id}
                    currentStatus={feedback.status}
                    gql={gql}
                    onSuccess={handleSuccess}
                    onError={handleError}
                  />
                </AccordionPanel>
                <AccordionPanel
                  title="Email Volunteer"
                  isOpen={openPanel === "email"}
                  onToggle={() => togglePanel("email")}
                >
                  <EmailPanel
                    feedbackId={feedback.id}
                    gql={gql}
                    onSuccess={handleSuccess}
                    onError={handleError}
                  />
                </AccordionPanel>
              </>
            )}
          </>
        )}
      </div>
      <FeedbackButton open={feedbackOpen} onClose={() => setFeedbackOpen(false)} />
    </div>
  );
}
