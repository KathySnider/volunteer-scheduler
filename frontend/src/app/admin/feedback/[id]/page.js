"use client";

import { useEffect, useState, useCallback } from "react";
import { useRouter, useParams } from "next/navigation";
import {
  getAuthToken,
  getAuthRole,
  getAuthName,
  signOut,
  adminGql,
} from "../../../lib/api";
import UserMenu from "../../../components/UserMenu";
import styles from "./admin-feedback-detail.module.css";

/* =========================================================
   GraphQL
   ========================================================= */

const FEEDBACK_BY_ID = `
  query FeedbackById($id: ID!) {
    feedbackById(feedbackId: $id) {
      id volunteerName type status subject appPageName text
      githubIssueURL createdAt lastUpdatedAt resolvedAt
      notes { id creator noteType note createdAt }
      attachments { id filename mimeType fileSize createdAt }
    }
  }
`;

const QUESTION_FEEDBACK = `
  mutation QuestionFeedback($input: QuestionFeedbackInput!) {
    questionFeedback(question: $input) { success message }
  }
`;

const UPDATE_FEEDBACK = `
  mutation UpdateFeedback($input: UpdateFeedbackInput!) {
    updateFeedback(feedback: $input) { success message }
  }
`;

const RESOLVE_FEEDBACK = `
  mutation ResolveFeedback($input: ResolveFeedbackInput!) {
    resolveFeedback(resolution: $input) { success message }
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

const NOTE_TYPE_LABEL = {
  ADMIN_NOTE:          "Admin note (internal):",
  QUESTION:            "Question sent to volunteer:",
  VOLUNTEER_REPLY:     "Volunteer replied:",
  EMAIL_TO_VOLUNTEER:  "Email to volunteer:",
};

/* =========================================================
   Sub-components (module level)
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

function NoteItem({ note }) {
  const cls = {
    ADMIN_NOTE:         styles.noteAdmin,
    QUESTION:           styles.noteQuestion,
    VOLUNTEER_REPLY:    styles.noteVolunteer,
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

function InputField({ label, value, onChange, required, placeholder, type = "text" }) {
  return (
    <div className={styles.field}>
      <label className={styles.label}>
        {label}
        {required && <span className={styles.required}> *</span>}
      </label>
      <input
        className={styles.input}
        type={type}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
      />
    </div>
  );
}

/* =========================================================
   Ask a Question Panel
   ========================================================= */

function AskQuestionPanel({ feedbackId, gql, onSuccess, onError }) {
  const [emailText, setEmailText] = useState("");
  const [note, setNote]           = useState("");
  const [busy, setBusy]           = useState(false);

  const handleSubmit = async () => {
    if (!emailText.trim()) { onError("Email text is required."); return; }
    if (!note.trim())      { onError("Internal note is required."); return; }
    setBusy(true);
    try {
      const res = await gql(QUESTION_FEEDBACK, {
        input: { id: feedbackId, emailText: emailText.trim(), note: note.trim() },
      });
      const result = res.data?.questionFeedback;
      if (res.errors || !result?.success) {
        onError(result?.message ?? res.errors?.[0]?.message ?? "Failed to send question.");
      } else {
        onSuccess("Question sent to volunteer.");
      }
    } catch {
      onError("Unable to reach the server.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className={styles.actionPanel}>
      <h3 className={styles.panelTitle}>Ask a Question</h3>
      <TextField
        label="Email text to volunteer"
        value={emailText}
        onChange={setEmailText}
        required
        placeholder="What would you like to ask the volunteer?"
      />
      <TextField
        label="Internal note"
        value={note}
        onChange={setNote}
        required
        placeholder="Internal note (not visible to volunteer)"
        rows={3}
      />
      <div className={styles.panelActions}>
        <button className={styles.btnPrimary} onClick={handleSubmit} disabled={busy}>
          {busy ? "Sending…" : "Send Question"}
        </button>
      </div>
    </div>
  );
}

/* =========================================================
   Update / Add to GitHub Panel
   ========================================================= */

const UPDATABLE_STATUSES = [
  { value: "OPEN",              label: "Open" },
  { value: "QUESTION_SENT",     label: "Question Sent" },
  { value: "RESOLVED_GITHUB",   label: "Resolved (GitHub)" },
  { value: "RESOLVED_REJECTED", label: "Rejected" },
];

function UpdatePanel({ feedbackId, currentStatus, currentGithubURL, gql, onSuccess, onError }) {
  const [note, setNote]             = useState("");
  const [status, setStatus]         = useState(currentStatus);
  const [githubURL, setGithubURL]   = useState(currentGithubURL ?? "");
  const [busy, setBusy]             = useState(false);

  const handleSubmit = async () => {
    if (!note.trim()) { onError("Note is required."); return; }
    setBusy(true);
    try {
      const input = {
        id: feedbackId,
        status,
        note: note.trim(),
      };
      if (githubURL.trim()) input.githubIssueURL = githubURL.trim();

      const res = await gql(UPDATE_FEEDBACK, { input });
      const result = res.data?.updateFeedback;
      if (res.errors || !result?.success) {
        onError(result?.message ?? res.errors?.[0]?.message ?? "Failed to update feedback.");
      } else {
        onSuccess("Feedback updated.");
      }
    } catch {
      onError("Unable to reach the server.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className={styles.actionPanel}>
      <h3 className={styles.panelTitle}>Update / Add to GitHub</h3>
      <TextField
        label="Note"
        value={note}
        onChange={setNote}
        required
        placeholder="Add a note about this update"
        rows={3}
      />
      <div className={styles.field}>
        <label className={styles.label}>Status</label>
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
      {status === "RESOLVED_GITHUB" && (
        <InputField
          label="GitHub Issue URL"
          value={githubURL}
          onChange={setGithubURL}
          placeholder="https://github.com/org/repo/issues/123"
          type="url"
        />
      )}
      <div className={styles.panelActions}>
        <button className={styles.btnPrimary} onClick={handleSubmit} disabled={busy}>
          {busy ? "Saving…" : "Update"}
        </button>
      </div>
    </div>
  );
}

/* =========================================================
   Resolve Panel
   ========================================================= */

function ResolvePanel({ feedbackId, currentGithubURL, gql, onSuccess, onError }) {
  const [note, setNote]           = useState("");
  const [resolution, setResolution] = useState("RESOLVED_REJECTED");
  const [githubURL, setGithubURL] = useState(currentGithubURL ?? "");
  const [busy, setBusy]           = useState(false);

  const handleSubmit = async () => {
    if (!note.trim()) { onError("Note is required."); return; }
    if (resolution === "RESOLVED_GITHUB" && !githubURL.trim()) {
      onError("GitHub URL is required when resolving via GitHub.");
      return;
    }
    setBusy(true);
    try {
      const input = {
        id: feedbackId,
        status: resolution,
        note: note.trim(),
      };
      if (githubURL.trim()) input.githubIssueURL = githubURL.trim();

      const res = await gql(RESOLVE_FEEDBACK, { input });
      const result = res.data?.resolveFeedback;
      if (res.errors || !result?.success) {
        onError(result?.message ?? res.errors?.[0]?.message ?? "Failed to resolve feedback.");
      } else {
        onSuccess("Feedback resolved.");
      }
    } catch {
      onError("Unable to reach the server.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className={styles.actionPanel}>
      <h3 className={styles.panelTitle}>Resolve Feedback</h3>
      <TextField
        label="Resolution note (will be emailed to volunteer)"
        value={note}
        onChange={setNote}
        required
        placeholder="Explain how this feedback was resolved or why it was rejected"
      />
      <div className={styles.field}>
        <label className={styles.label}>Resolution</label>
        <div className={styles.radioGroup}>
          <label className={styles.radioLabel}>
            <input
              type="radio"
              name="resolution"
              value="RESOLVED_REJECTED"
              checked={resolution === "RESOLVED_REJECTED"}
              onChange={() => setResolution("RESOLVED_REJECTED")}
            />
            Rejected
          </label>
          <label className={styles.radioLabel}>
            <input
              type="radio"
              name="resolution"
              value="RESOLVED_GITHUB"
              checked={resolution === "RESOLVED_GITHUB"}
              onChange={() => setResolution("RESOLVED_GITHUB")}
            />
            Resolved via GitHub
          </label>
        </div>
      </div>
      {resolution === "RESOLVED_GITHUB" && (
        <InputField
          label="GitHub Issue URL"
          value={githubURL}
          onChange={setGithubURL}
          placeholder="https://github.com/org/repo/issues/123"
          type="url"
          required
        />
      )}
      <div className={styles.panelActions}>
        <button className={`${styles.btnPrimary} ${styles.btnDanger}`} onClick={handleSubmit} disabled={busy}>
          {busy ? "Resolving…" : "Resolve Feedback"}
        </button>
      </div>
    </div>
  );
}

/* =========================================================
   Page
   ========================================================= */

export default function AdminFeedbackDetailPage() {
  const router   = useRouter();
  const params   = useParams();
  const id       = params?.id;

  const [token, setToken]       = useState(null);
  const [gql, setGql]           = useState(null);
  const [userName, setUserName] = useState("");

  const [feedback, setFeedback]   = useState(null);
  const [loading, setLoading]     = useState(true);
  const [pageError, setPageError] = useState("");
  const [actionMsg, setActionMsg] = useState(null);

  /* ----- Load detail ----- */
  const loadData = useCallback((bound, feedbackId) => {
    setLoading(true);
    setPageError("");
    bound(FEEDBACK_BY_ID, { id: feedbackId })
      .then((res) => {
        setFeedback(res.data?.feedbackById ?? null);
        if (res.errors) setPageError(res.errors[0]?.message ?? "Error loading feedback.");
        if (!res.data?.feedbackById && !res.errors) setPageError("Feedback not found.");
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
    if (id) loadData(bound, id);
  }, [router, loadData, id]);

  const handleSignOut = async () => { await signOut(token); router.replace("/login"); };

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
    ? ["RESOLVED_GITHUB", "RESOLVED_REJECTED"].includes(feedback.status)
    : false;

  return (
    <div className={styles.page}>
      {/* Top bar */}
      <div className={styles.topBar}>
        <a href="/admin/feedback" className={styles.backLink}>← Manage Feedback</a>
        <UserMenu name={userName} isAdmin={true} onSignOut={handleSignOut} />
      </div>

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
              {feedback.githubIssueURL && (
                <div className={styles.githubLink}>
                  <strong>GitHub Issue:</strong>{" "}
                  <a
                    href={feedback.githubIssueURL}
                    target="_blank"
                    rel="noopener noreferrer"
                    className={styles.externalLink}
                  >
                    {feedback.githubIssueURL}
                  </a>
                </div>
              )}
            </div>

            {/* Original feedback text */}
            <div className={styles.card}>
              <h2 className={styles.cardTitle}>Original Feedback</h2>
              <p className={styles.feedbackText}>{feedback.text}</p>
            </div>

            {/* Notes thread */}
            {feedback.notes && feedback.notes.length > 0 && (
              <div className={styles.card}>
                <h2 className={styles.cardTitle}>Notes &amp; Thread</h2>
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
                      <span className={styles.attachmentName}>{att.filename}</span>
                      <span className={styles.attachmentMeta}>
                        {att.mimeType} · {Math.round(att.fileSize / 1024)} KB
                      </span>
                    </li>
                  ))}
                </ul>
              </div>
            )}

            {/* Action panels */}
            {isResolved ? (
              <div className={styles.resolvedNote}>
                This feedback has been resolved.
              </div>
            ) : (
              <>
                {feedback.status === "OPEN" && (
                  <AskQuestionPanel
                    feedbackId={feedback.id}
                    gql={gql}
                    onSuccess={handleSuccess}
                    onError={handleError}
                  />
                )}
                <UpdatePanel
                  feedbackId={feedback.id}
                  currentStatus={feedback.status}
                  currentGithubURL={feedback.githubIssueURL}
                  gql={gql}
                  onSuccess={handleSuccess}
                  onError={handleError}
                />
                <ResolvePanel
                  feedbackId={feedback.id}
                  currentGithubURL={feedback.githubIssueURL}
                  gql={gql}
                  onSuccess={handleSuccess}
                  onError={handleError}
                />
              </>
            )}
          </>
        )}
      </div>
    </div>
  );
}
