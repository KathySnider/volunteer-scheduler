"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import { getAuthToken, volunteerGql, volunteerGqlUpload } from "../lib/api";
import styles from "./FeedbackButton.module.css";

/* ----- GraphQL ----- */

const GIVE_FEEDBACK = `
  mutation GiveFeedback($input: NewFeedbackInput!) {
    giveFeedback(feedback: $input) {
      success
      message
      id
    }
  }
`;

const ATTACH_FILE = `
  mutation AttachFile($feedbackId: ID!, $file: Upload!) {
    attachFileToFeedback(feedbackId: $feedbackId, file: $file) {
      success
      message
    }
  }
`;

/* ----- Constants ----- */

const TYPE_OPTIONS = [
  { value: "BUG", label: "🐛 Bug Report" },
  { value: "ENHANCEMENT", label: "✨ Enhancement Request" },
  { value: "GENERAL", label: "💬 General Feedback" },
];

const MAX_SUBJECT = 200;
const MAX_TEXT = 2000;
const MAX_FILES = 5;
const MAX_FILE_SIZE = 5 * 1024 * 1024; // 5 MB

/* ----- Pure helpers ----- */

function formatFileSize(bytes) {
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function validateFile(file) {
  if (file.size > MAX_FILE_SIZE) {
    return `${file.name} exceeds the 5 MB limit`;
  }
  return null;
}

/* ----- FeedbackButton component ----- */

/**
 * Can be used in two modes:
 *
 * Uncontrolled (default): renders a floating "?" button that opens the modal internally.
 *   <FeedbackButton />
 *
 * Controlled: parent supplies open/onClose; no floating button is rendered.
 *   <FeedbackButton open={feedbackOpen} onClose={() => setFeedbackOpen(false)} />
 */
export default function FeedbackButton({ open: controlledOpen, onClose: controlledOnClose } = {}) {
  const isControlled = controlledOpen !== undefined;

  const [internalOpen, setInternalOpen] = useState(false);

  // Unified open value — use controlled prop when provided, otherwise internal state
  const open = isControlled ? controlledOpen : internalOpen;

  // Unified setter — in controlled mode, closing notifies the parent; opening is a no-op
  // (the parent decides when to open). In uncontrolled mode, drive internal state directly.
  const setOpen = useCallback((val) => {
    if (isControlled) {
      if (!val) controlledOnClose?.();
    } else {
      setInternalOpen(val);
    }
  }, [isControlled, controlledOnClose]);

  const [type, setType] = useState("BUG");
  const [subject, setSubject] = useState("");
  const [text, setText] = useState("");
  const [files, setFiles] = useState([]);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState(false);
  const [successNote, setSuccessNote] = useState("");
  const successTimer = useRef(null);
  const overlayRef = useRef(null);
  const fileInputRef = useRef(null);

  // Reset form when modal opens
  useEffect(() => {
    if (open) {
      setType("BUG");
      setSubject("");
      setText("");
      setFiles([]);
      setError("");
      setSuccess(false);
      setSuccessNote("");
    }
  }, [open]);

  // Close after success
  useEffect(() => {
    if (success) {
      successTimer.current = setTimeout(() => {
        setOpen(false);
        setSuccess(false);
        setSuccessNote("");
      }, 2500);
    }
    return () => clearTimeout(successTimer.current);
  }, [success]);

  // Close on Escape
  useEffect(() => {
    if (!open) return;
    function onKey(e) {
      if (e.key === "Escape") setOpen(false);
    }
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [open]);

  const handleOverlayClick = useCallback((e) => {
    if (e.target === overlayRef.current) setOpen(false);
  }, []);

  const handleFileSelect = useCallback((e) => {
    const selected = Array.from(e.target.files ?? []);
    e.target.value = ""; // reset so same file can be re-selected
    setFiles((prev) => {
      const combined = [...prev];
      for (const f of selected) {
        if (combined.length >= MAX_FILES) break;
        combined.push(f);
      }
      return combined;
    });
  }, []);

  const handleRemoveFile = useCallback((index) => {
    setFiles((prev) => prev.filter((_, i) => i !== index));
  }, []);

  const handleSubmit = useCallback(async (e) => {
    e.preventDefault();
    setError("");

    // Validate files before hitting the server
    for (const f of files) {
      const err = validateFile(f);
      if (err) { setError(err); return; }
    }

    const token = getAuthToken();
    if (!token) {
      setError("You must be signed in to submit feedback.");
      return;
    }

    const trimmedSubject = subject.trim();
    const trimmedText = text.trim();
    if (!trimmedSubject) { setError("Subject is required."); return; }
    if (!trimmedText) { setError("Description is required."); return; }

    const appPageName =
      typeof window !== "undefined" ? window.location.pathname : "/";

    setSubmitting(true);
    try {
      // 1 — Submit the feedback text
      const res = await volunteerGql(GIVE_FEEDBACK, {
        input: {
          type,
          subject: trimmedSubject,
          app_page_name: appPageName,
          text: trimmedText,
        },
      }, token);

      if (res.errors) {
        setError(res.errors[0]?.message ?? "Failed to submit feedback.");
        return;
      }
      if (res.data?.giveFeedback?.success === false) {
        setError(res.data.giveFeedback.message ?? "Failed to submit feedback.");
        return;
      }

      const feedbackId = res.data?.giveFeedback?.id;

      // 2 — Upload attachments (best-effort; feedback text already saved)
      if (feedbackId && files.length > 0) {
        let failedCount = 0;
        for (const file of files) {
          try {
            const r = await volunteerGqlUpload(
              ATTACH_FILE,
              { feedbackId },
              file,
              token
            );
            if (!r.data?.attachFileToFeedback?.success) failedCount++;
          } catch {
            failedCount++;
          }
        }
        if (failedCount > 0) {
          setSuccessNote(
            `Note: ${failedCount} attachment${failedCount > 1 ? "s" : ""} could not be uploaded.`
          );
        }
      }

      setSuccess(true);
    } catch {
      setError("Unable to reach the server. Please try again.");
    } finally {
      setSubmitting(false);
    }
  }, [type, subject, text, files]);

  return (
    <>
      {/* Floating button — only rendered in uncontrolled mode */}
      {!isControlled && (
        <button
          className={styles.floatingBtn}
          onClick={() => setOpen(true)}
          aria-label="Submit feedback"
          title="Submit feedback"
        >
          {/* Edit / write icon */}
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="22"
            height="22"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
            aria-hidden="true"
          >
            <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
            <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
          </svg>
        </button>
      )}

      {/* Modal overlay */}
      {open && (
        <div
          className={styles.overlay}
          ref={overlayRef}
          onClick={handleOverlayClick}
          role="dialog"
          aria-modal="true"
          aria-label="Feedback form"
        >
          <div className={styles.modal}>
            <div className={styles.modalHeader}>
              <h2 className={styles.modalTitle}>Submit Feedback</h2>
              <button
                className={styles.closeBtn}
                onClick={() => setOpen(false)}
                aria-label="Close feedback form"
              >
                &#10005;
              </button>
            </div>

            {success ? (
              <div className={styles.successMessage}>
                <div>Thank you! Your feedback has been submitted.</div>
                {successNote && (
                  <div className={styles.successNote}>{successNote}</div>
                )}
              </div>
            ) : (
              <form className={styles.form} onSubmit={handleSubmit} noValidate>
                {error && <div className={styles.errorBox}>{error}</div>}

                {/* Type */}
                <div className={styles.field}>
                  <label className={styles.label} htmlFor="fb-type">Type</label>
                  <select
                    id="fb-type"
                    className={styles.select}
                    value={type}
                    onChange={(e) => setType(e.target.value)}
                    disabled={submitting}
                  >
                    {TYPE_OPTIONS.map((o) => (
                      <option key={o.value} value={o.value}>{o.label}</option>
                    ))}
                  </select>
                </div>

                {/* Subject */}
                <div className={styles.field}>
                  <label className={styles.label} htmlFor="fb-subject">
                    Subject <span className={styles.required}>*</span>
                  </label>
                  <input
                    id="fb-subject"
                    type="text"
                    className={styles.input}
                    value={subject}
                    onChange={(e) => setSubject(e.target.value.slice(0, MAX_SUBJECT))}
                    placeholder="Brief summary of your feedback"
                    maxLength={MAX_SUBJECT}
                    required
                    disabled={submitting}
                  />
                </div>

                {/* Description */}
                <div className={styles.field}>
                  <label className={styles.label} htmlFor="fb-text">
                    Description <span className={styles.required}>*</span>
                  </label>
                  <textarea
                    id="fb-text"
                    className={styles.textarea}
                    value={text}
                    onChange={(e) => setText(e.target.value.slice(0, MAX_TEXT))}
                    placeholder="What happened? What did you expect to happen? Steps to reproduce..."
                    maxLength={MAX_TEXT}
                    rows={5}
                    required
                    disabled={submitting}
                  />
                  <div className={styles.charCount}>{text.length}/{MAX_TEXT}</div>
                </div>

                {/* Attachments */}
                <div className={styles.field}>
                  <div className={styles.attachLabel}>
                    Attachments{" "}
                    <span className={styles.attachOptional}>(optional)</span>
                  </div>

                  {/* Hidden file input */}
                  <input
                    ref={fileInputRef}
                    type="file"
                    className={styles.hiddenInput}
                    accept="image/*,.pdf,.doc,.docx,.txt"
                    multiple
                    onChange={handleFileSelect}
                    disabled={submitting}
                  />

                  {files.length < MAX_FILES && (
                    <button
                      type="button"
                      className={styles.addFileBtn}
                      onClick={() => fileInputRef.current?.click()}
                      disabled={submitting}
                    >
                      + Add file
                    </button>
                  )}

                  {files.length > 0 && (
                    <ul className={styles.fileList}>
                      {files.map((file, i) => {
                        const err = validateFile(file);
                        return (
                          <li key={i} className={`${styles.fileItem}${err ? ` ${styles.fileItemError}` : ""}`}>
                            <span className={styles.fileName}>{file.name}</span>
                            <span className={styles.fileSize}>{formatFileSize(file.size)}</span>
                            {err && <span className={styles.fileError}>{err}</span>}
                            <button
                              type="button"
                              className={styles.fileRemove}
                              onClick={() => handleRemoveFile(i)}
                              aria-label={`Remove ${file.name}`}
                              disabled={submitting}
                            >
                              ✕
                            </button>
                          </li>
                        );
                      })}
                    </ul>
                  )}

                  <p className={styles.fileHint}>
                    Images, PDF, Word, or text files. Max 5 MB each, up to 5 files.
                  </p>
                </div>

                {/* Actions */}
                <div className={styles.actions}>
                  <button
                    type="button"
                    className={styles.cancelBtn}
                    onClick={() => setOpen(false)}
                    disabled={submitting}
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    className={styles.submitBtn}
                    disabled={submitting}
                  >
                    {submitting ? "Submitting…" : "Send Feedback"}
                  </button>
                </div>
              </form>
            )}
          </div>
        </div>
      )}
    </>
  );
}
