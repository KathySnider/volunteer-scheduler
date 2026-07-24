"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { isAuthenticated, hasAuthRole, Roles, getAuthName, signOut } from "./lib/api";
import FeedbackButton from "./components/FeedbackButton";
import styles from "./dashboard.module.css";

const LINK_CARDS = [
  {
    href:  "/my-shifts",
    title: "My Signups",
    desc:  "View upcoming and past events you've signed up for",
    color: "blue",
  },
  {
    href:  "/events",
    title: "Volunteer Events",
    desc:  "Browse all volunteer events",
    color: "purple",
  },
  {
    href:  "/profile",
    title: "My Preferences",
    desc:  "Change preferences in your profile",
    color: "yellow",
  },
  {
    href:  "/my-feedback",
    title: "My Feedback",
    desc:  "View and manage feedback submissions you've submitted",
    color: "teal",
  },
];

export default function DashboardPage() {
  const router = useRouter();
  const [feedbackOpen, setFeedbackOpen] = useState(false);

  useEffect(() => {
    if (!isAuthenticated()) {
      router.replace("/login");
    } else if (hasAuthRole(Roles.ADMINISTRATOR)) {
      router.replace("/admin/events");
    }
  }, [router]);

  if (!isAuthenticated() || hasAuthRole(Roles.ADMINISTRATOR)) return null;

  const rawName = getAuthName() ?? "";
  const firstName = rawName.split(" ")[0] || "there";

  const handleSignOut = async () => { await signOut(); router.replace("/login"); };

  return (
    <div className={styles.page}>
      <h1 className={styles.greeting}>Welcome, {firstName}!</h1>
      <p className={styles.subtitle}>What would you like to do today?</p>
      <hr className={styles.divider} />
      <div className={styles.grid}>
        {LINK_CARDS.map((card) => (
          <Link key={card.href} href={card.href} className={`${styles.card} ${styles[card.color]}`}>
            <div className={styles.cardTitle}>{card.title}</div>
            <div className={styles.cardDesc}>{card.desc}</div>
          </Link>
        ))}
        <button className={`${styles.card} ${styles.green}`} onClick={() => setFeedbackOpen(true)}>
          <div className={styles.cardTitle}>Submit Feedback</div>
          <div className={styles.cardDesc}>Send a question or comment to the event organizers</div>
        </button>
        <button className={`${styles.card} ${styles.signOut}`} onClick={handleSignOut}>
          <div className={styles.cardTitle}>Sign Out</div>
          <div className={styles.cardDesc}>Sign out of your account</div>
        </button>
      </div>
      <FeedbackButton open={feedbackOpen} onClose={() => setFeedbackOpen(false)} />
    </div>
  );
}
