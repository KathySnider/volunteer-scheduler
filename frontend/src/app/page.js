"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { authGql, getAuthToken, clearAuthToken } from "./lib/api";
import styles from "./home.module.css";

const LOGOUT = `
  mutation Logout($token: String!) {
    logout(token: $token) {
      success
    }
  }
`;

export default function HomePage() {
  const router = useRouter();
  const [email, setEmail] = useState(null);

  useEffect(() => {
    const token = getAuthToken();
    if (!token) {
      router.replace("/login");
      return;
    }
    setEmail(localStorage.getItem("authEmail") || "");
  }, [router]);

  const handleLogout = async () => {
    const token = getAuthToken();
    if (token) {
      try {
        await authGql(LOGOUT, { token });
      } catch {
        // Best-effort — clear local state regardless
      }
    }
    clearAuthToken();
    router.push("/login");
  };

  // Render nothing while the redirect to /login is in flight
  if (email === null) return null;

  return (
    <div className={styles.page}>
      <div className={styles.card}>
        <h1 className={styles.title}>You&apos;re signed in</h1>
        <p className={styles.email}>{email}</p>
        <button className={styles.logoutButton} onClick={handleLogout}>
          Sign out
        </button>
      </div>
    </div>
  );
}
