"use client";

import { useState } from "react";
import { authGql } from "../lib/api";
import styles from "./login.module.css";

/* ----- GraphQL mutations ----- */

const REQUEST_MAGIC_LINK = `
  mutation RequestMagicLink($email: String!) {
    requestMagicLink(email: $email) {
      success
      message
    }
  }
`;

const REQUEST_ACCOUNT = `
  mutation RequestAccount($email: String!, $firstName: String!, $lastName: String!) {
    requestAccount(email: $email, firstName: $firstName, lastName: $lastName) {
      success
      message
    }
  }
`;

/* ----- Stages ----- */
// enterEmail → (found) linkSent
//            → (not found) notFound → requestForm → requestSent
//            → (inactive)  inactive
//            → (server error) enterEmail with errorMsg

export default function LoginPage() {
  const [stage, setStage] = useState("enterEmail");

  // Form values
  const [email, setEmail] = useState("");
  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName] = useState("");

  // UI state
  const [loading, setLoading] = useState(false);
  const [errorMsg, setErrorMsg] = useState("");

  /* Reset everything back to the initial email entry stage */
  const reset = () => {
    setStage("enterEmail");
    setEmail("");
    setFirstName("");
    setLastName("");
    setErrorMsg("");
  };

  /* Submit the email address to request a magic link */
  const handleEmailSubmit = async (e) => {
    e.preventDefault();
    setErrorMsg("");
    setLoading(true);
    try {
      const result = await authGql(REQUEST_MAGIC_LINK, { email });
      const { success, message } = result.data.requestMagicLink;

      if (success) {
        setStage("linkSent");
      } else if (message.toLowerCase().includes("no volunteer account")) {
        setStage("notFound");
      } else if (message.toLowerCase().includes("inactive")) {
        setStage("inactive");
      } else {
        setErrorMsg(message);
      }
    } catch {
      setErrorMsg("Unable to reach the server. Please try again.");
    } finally {
      setLoading(false);
    }
  };

  /* Submit the account request form */
  const handleRequestAccount = async (e) => {
    e.preventDefault();
    setErrorMsg("");
    setLoading(true);
    try {
      const result = await authGql(REQUEST_ACCOUNT, {
        email,
        firstName,
        lastName,
      });
      const { success, message } = result.data.requestAccount;

      if (success) {
        setStage("requestSent");
      } else {
        setErrorMsg(message);
      }
    } catch {
      setErrorMsg("Unable to reach the server. Please try again.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className={styles.page}>
      <div className={styles.appHeader}>
        <div className={styles.appName}>Volunteer Scheduler System</div>
        <div className={styles.appTagline}>Volunteer event management</div>
      </div>

      <div className={styles.card}>
        {stage === "enterEmail" && (
          <EnterEmailStage
            email={email}
            setEmail={setEmail}
            loading={loading}
            errorMsg={errorMsg}
            onSubmit={handleEmailSubmit}
          />
        )}

        {stage === "linkSent" && (
          <LinkSentStage email={email} onReset={reset} />
        )}

        {stage === "notFound" && (
          <NotFoundStage
            email={email}
            onRequestAccount={() => setStage("requestForm")}
            onReset={reset}
          />
        )}

        {stage === "requestForm" && (
          <RequestFormStage
            email={email}
            firstName={firstName}
            setFirstName={setFirstName}
            lastName={lastName}
            setLastName={setLastName}
            loading={loading}
            errorMsg={errorMsg}
            onSubmit={handleRequestAccount}
            onBack={() => setStage("notFound")}
          />
        )}

        {stage === "requestSent" && (
          <RequestSentStage email={email} onReset={reset} />
        )}

        {stage === "inactive" && (
          <InactiveStage
            email={email}
            onRequestAccount={() => setStage("requestForm")}
            onReset={reset}
          />
        )}
      </div>
    </div>
  );
}

/* =========================================================
   Stage components
   ========================================================= */

function EnterEmailStage({ email, setEmail, loading, errorMsg, onSubmit }) {
  return (
    <>
      <h1 className={styles.cardTitle}>Sign In</h1>
      <p className={styles.cardBody}>
        Enter your email address and we&apos;ll send you a sign-in link.
      </p>
      <form className={styles.form} onSubmit={onSubmit}>
        <div className={styles.field}>
          <label className={styles.label} htmlFor="email">
            Email address
          </label>
          <input
            id="email"
            className={styles.input}
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="you@example.com"
            autoComplete="email"
            required
          />
        </div>
        {errorMsg && <p className={styles.errorMessage}>{errorMsg}</p>}
        <button className={styles.buttonPrimary} type="submit" disabled={loading}>
          {loading ? "Sending…" : "Continue"}
        </button>
      </form>
    </>
  );
}

function LinkSentStage({ email, onReset }) {
  return (
    <>
      <div className={`${styles.statusIcon} ${styles.statusIconSuccess}`}>
        ✉
      </div>
      <h1 className={styles.cardTitle}>Check your email</h1>
      <p className={styles.cardBody}>
        We sent a sign-in link to{" "}
        <span className={styles.highlight}>{email}</span>. Click the link in
        the email to sign in.
      </p>
      <p className={styles.cardBody}>
        The link expires in 15 minutes. Check your spam folder if you
        don&apos;t see it.
      </p>
      <button className={styles.linkButton} onClick={onReset}>
        Use a different email
      </button>
    </>
  );
}

function NotFoundStage({ email, onRequestAccount, onReset }) {
  return (
    <>
      <h1 className={styles.cardTitle}>No account found</h1>
      <p className={styles.cardBody}>
        There&apos;s no volunteer account associated with{" "}
        <span className={styles.highlight}>{email}</span>.
      </p>
      <p className={styles.cardBody}>Would you like to request access?</p>
      <div className={`${styles.buttonStack} ${styles.notFoundActions}`}>
        <button className={styles.buttonPrimary} onClick={onRequestAccount}>
          Request an Account
        </button>
        <button className={styles.buttonOutline} onClick={onReset}>
          Try a Different Email
        </button>
      </div>
    </>
  );
}

function RequestFormStage({
  email,
  firstName,
  setFirstName,
  lastName,
  setLastName,
  loading,
  errorMsg,
  onSubmit,
  onBack,
}) {
  return (
    <>
      <h1 className={styles.cardTitle}>Request an Account</h1>
      <p className={styles.cardBody}>
        Fill in your name and submit a request. An administrator will review
        it and create your account.
      </p>
      <form className={styles.form} onSubmit={onSubmit}>
        <div className={styles.field}>
          <label className={styles.label} htmlFor="firstName">
            First name
          </label>
          <input
            id="firstName"
            className={styles.input}
            type="text"
            value={firstName}
            onChange={(e) => setFirstName(e.target.value)}
            autoComplete="given-name"
            required
          />
        </div>
        <div className={styles.field}>
          <label className={styles.label} htmlFor="lastName">
            Last name
          </label>
          <input
            id="lastName"
            className={styles.input}
            type="text"
            value={lastName}
            onChange={(e) => setLastName(e.target.value)}
            autoComplete="family-name"
            required
          />
        </div>
        <div className={styles.field}>
          <label className={styles.label} htmlFor="emailReadonly">
            Email
          </label>
          <input
            id="emailReadonly"
            className={`${styles.input} ${styles.inputReadonly}`}
            type="email"
            value={email}
            readOnly
          />
        </div>
        {errorMsg && <p className={styles.errorMessage}>{errorMsg}</p>}
        <button
          className={styles.buttonPrimary}
          type="submit"
          disabled={loading}
        >
          {loading ? "Submitting…" : "Submit Request"}
        </button>
        <button
          type="button"
          className={styles.buttonOutline}
          onClick={onBack}
        >
          Back
        </button>
      </form>
    </>
  );
}

function InactiveStage({ email, onRequestAccount, onReset }) {
  return (
    <>
      <h1 className={styles.cardTitle}>Account Inactive</h1>
      <p className={styles.cardBody}>
        The account for <span className={styles.highlight}>{email}</span> is
        currently inactive.
      </p>
      <p className={styles.cardBody}>
        You can submit a request and an administrator will review it. They
        may reactivate your existing account or create a new one.
      </p>
      <div className={`${styles.buttonStack} ${styles.notFoundActions}`}>
        <button className={styles.buttonPrimary} onClick={onRequestAccount}>
          Request Account Access
        </button>
        <button className={styles.buttonOutline} onClick={onReset}>
          Try a Different Email
        </button>
      </div>
    </>
  );
}

function RequestSentStage({ email, onReset }) {
  return (
    <>
      <div className={`${styles.statusIcon} ${styles.statusIconSuccess}`}>
        ✓
      </div>
      <h1 className={styles.cardTitle}>Request Submitted</h1>
      <p className={styles.cardBody}>
        Thank you! An administrator will review your request and be in touch
        at <span className={styles.highlight}>{email}</span>.
      </p>
      <button className={styles.linkButton} onClick={onReset}>
        Back to sign in
      </button>
    </>
  );
}
