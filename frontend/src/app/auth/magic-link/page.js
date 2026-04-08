"use client";

import { useEffect, useState, Suspense } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";
import { authGql, volunteerGql, setAuthToken } from "../../lib/api";
import styles from "./magic-link.module.css";

/* ----- GraphQL operations ----- */

const CONSUME_MAGIC_LINK = `
  mutation ConsumeMagicLink($token: String!) {
    consumeMagicLink(token: $token) {
      success
      message
      email
      sessionToken
    }
  }
`;

const VOLUNTEER_PROFILE = `
  query {
    volunteerProfile {
      role
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
      const { success, message, sessionToken, email } =
        result.data.consumeMagicLink;

      if (!success || !sessionToken) {
        setStatus("error");
        setErrorMsg(message || "Authentication failed.");
        return;
      }

      // Fetch the volunteer's role so the app can route correctly.
      let role = null;
      try {
        const profileResult = await volunteerGql(
          VOLUNTEER_PROFILE,
          null,
          sessionToken
        );
        role = profileResult.data?.volunteerProfile?.role ?? null;
      } catch {
        // Non-fatal — proceed without role; the events page will still load.
      }

      setAuthToken(sessionToken, email, role);
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
