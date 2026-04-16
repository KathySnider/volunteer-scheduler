"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import { getAuthToken, volunteerGql } from "../lib/api";
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

/* ----- Constants ----- */

const TYPE_OPTIONS = [
  { value: "BUG", label: "Bug Report" },
  { value: "ENHANCEMENT", label: "Enhancement Request" },
  { value: "GENERAL", label: "General Feedback" },
];

const MAX_SUBJECT = 200;
const MAX_TEXT = 2000;

/* ----- FeedbackButton component ----- */

export default function FeedbackButton() {
  const [open, setOpen] = useState(false);
  const [type, setType] = useState("GENERAL");
  const [subject, setSubject] = useState("");
  const [text, setText] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState(false);
  const successTimer = useRef(null);
  const overlayRef = useRef(null);

  // Reset form when modal opens
  useEffect(() => {
    if (open) {
      setType("GENERAL");
      setSubject("");
      setText("");
      setError("");
      setSuccess(false);
    }
  }, [open]);

  // Close after success
  useEffect(() => {
    if (success) {
      successTimer.current = setTimeout(() => {
        setOpen(false);
        setSuccess(false);
      }, 2000);
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

  const handleSubmit = useCallback(async (e) => {
    e.preventDefault();
    setError("");

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
      } else if (res.data?.giveFeedback?.success === false) {
        setError(res.data.giveFeedback.message ?? "Failed to submit feedback.");
      } else {
        setSuccess(true);
      }
    } catch {
      setError("Unable to reach the server. Please try again.");
    } finally {
      setSubmitting(false);
    }
  }, [type, subject, text]);

  return (
    <>
      {/* Floating button */}
      <button
        className={styles.floatingBtn}
        onClick={() => setOpen(true)}
        aria-label="Submit feedback"
        title="Submit feedback"
      >
        ?
      </button>

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
                Thank you! Your feedback has been submitted.
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
                  <div className={styles.charCount}>
                    {subject.length}/{MAX_SUBJECT}
                  </div>
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
                    placeholder="Please describe your feedback in detail..."
                    maxLength={MAX_TEXT}
                    rows={5}
                    required
                    disabled={submitting}
                  />
                  <div className={styles.charCount}>
                    {text.length}/{MAX_TEXT}
                  </div>
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
                    {submitting ? "Submitting..." : "Submit Feedback"}
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
