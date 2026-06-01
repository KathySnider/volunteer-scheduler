"use client";

import { useEffect, useState, Suspense } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";
import { authGql, volunteerGql, setAuthInfo } from "../../lib/api";
import styles from "./magic-link.module.css";

/* ----- GraphQL operations ----- */

const CONSUME_MAGIC_LINK = `
  mutation ConsumeMagicLink($token: String!) {
    consumeMagicLink(token: $token) {
      success
      message
      email
    }
  }
`;

const VOLUNTEER_PROFILE = `
  query {
    ownProfile {
      firstName
      lastName
      roles
    }
  }
`;

/* ----- Inner component — uses useSearchParams, must be inside Suspense ----- */

function MagicLinkContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [status, setStatus] = useState("processing"); // processing | success | error
  const [errorMsg, setErrorMsg] = useState("");

  useEffect(() => {
    const token = searchParams.get("token");
    if (!token) {
      setStatus("error");
      setErrorMsg(
        "No token was found in this link. Please request a new sign-in link."
      );
      return;
    }
    consumeToken(token);
  }, [searchParams]);

  const consumeToken = async (token) => {
    try {
      const result = await authGql(CONSUME_MAGIC_LINK, { token });
      const { success, message, email } = result.data.consumeMagicLink;

      if (!success) {
        setStatus("error");
        setErrorMsg(message || "Authentication failed.");
        return;
      }

      // The server has set an HttpOnly session cookie. Now fetch the volunteer's
      // profile — the cookie is sent automatically via credentials: 'include'.
      let roles = null;
      let name = null;
      try {
        const profileResult = await volunteerGql(VOLUNTEER_PROFILE);
        const profile = profileResult.data?.ownProfile;
        roles = profile?.roles ?? null;
        if (profile?.firstName || profile?.lastName) {
          name = `${profile.firstName ?? ""} ${profile.lastName ?? ""}`.trim();
        }
      } catch {
        // Non-fatal — proceed without profile; the events page will still load.
      }

      // Save only display values — the session token lives in the HttpOnly cookie.
      setAuthInfo(email, roles, name);
      setStatus("success");
      setTimeout(() => router.push("/events"), 2000);
    } catch {
      setStatus("error");
      setErrorMsg("Unable to reach the server. Please try again.");
    }
  };

  if (status === "processing") {
    return (
      <div className={styles.card}>
        <div className={`${styles.iconWrapper} ${styles.iconLoading}`}>
          <div className={styles.spinner} />
        </div>
        <h1 className={styles.title}>Signing you in&hellip;</h1>
        <p className={styles.message}>Please wait a moment.</p>
      </div>
    );
  }

  if (status === "success") {
    return (
      <div className={styles.card}>
        <div className={`${styles.iconWrapper} ${styles.iconSuccess}`}>
          <span className={styles.checkmark}>✓</span>
        </div>
        <h1 className={styles.title}>Signed in!</h1>
        <p className={styles.message}>Redirecting you now&hellip;</p>
      </div>
    );
  }

  return (
    <div className={styles.card}>
      <div className={`${styles.iconWrapper} ${styles.iconError}`}>
        <span className={styles.crossmark}>✕</span>
      </div>
      <h1 className={styles.title}>Sign-in failed</h1>
      <p className={styles.message}>{errorMsg}</p>
      <Link href="/login" className={styles.backLink}>
        Request a new sign-in link
      </Link>
    </div>
  );
}

/* ----- Loading fallback while searchParams resolves ----- */

function LoadingCard() {
  return (
    <div className={styles.card}>
      <div className={`${styles.iconWrapper} ${styles.iconLoading}`}>
        <div className={styles.spinner} />
      </div>
      <h1 className={styles.title}>Loading&hellip;</h1>
    </div>
  );
}

/* ----- Page export ----- */

export default function MagicLinkPage() {
  return (
    <div className={styles.page}>
      <Suspense fallback={<LoadingCard />}>
        <MagicLinkContent />
      </Suspense>
    </div>
  );
}
