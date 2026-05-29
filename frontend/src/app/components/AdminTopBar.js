"use client";

import { useEffect } from "react";
import UserMenu from "./UserMenu";
import styles from "./admin-top-bar.module.css";
import { isAuthenticated, getVenues, getOwnShifts } from "../lib/api";

/**
 * Persistent header bar used on every page (volunteer and admin).
 *
 * The volunteer nav links are always shown so they stay anchored in the same
 * position regardless of which page the user is on.  When isAdmin is true the
 * admin section links appear after a divider, between the volunteer links and
 * the user name — non-admin volunteers never see them.
 *
 * The app title is a link to /events (volunteer events page).
 *
 * Props:
 *   userName       {string}   — display name from localStorage
 *   onSignOut      {function} — sign-out handler
 *   onFeedbackOpen {function} — opens the Submit Feedback modal
 *   isAdmin        {boolean}  — show admin section links (default false)
 */
export default function AdminTopBar({ userName, onSignOut, onFeedbackOpen, isAdmin = false }) {
  // Warm caches in the background on every page so subsequent navigations
  // don't need to wait for separate fetches.
  useEffect(() => {
    if (isAuthenticated()) {
      // Everyone's own upcoming shifts — used for conflict detection on event detail.
      getOwnShifts().catch(() => {});
    }
    if (isAdmin) {
      // Full venue list — used by Create/Edit Event forms.
      getVenues().catch(() => {});
    }
  }, [isAdmin]);

  return (
    <div className={styles.topBar}>
      <div className={styles.topBarLeft}>
        {/* App title doubles as a home link to the volunteer events page */}
        <a href="/events" className={styles.appTitle}>Volunteer Scheduler</a>

        {/* Volunteer nav — always visible for every role on every page */}
        <nav className={styles.topBarNav}>
          <a href="/my-shifts"   className={styles.topBarLink}>My Shifts</a>
          <a href="/profile"     className={styles.topBarLink}>My Profile</a>
          <button
            type="button"
            className={styles.topBarLink}
            onClick={onFeedbackOpen}
          >
            Submit Feedback
          </button>
          <a href="/my-feedback" className={styles.topBarLink}>My Feedback</a>
        </nav>

        {/* Admin section nav — only shown when signed in as admin */}
        {isAdmin && (
          <>
            <div className={styles.navDivider} />
            <nav className={styles.topBarNav}>
              <a href="/admin/events"     className={styles.topBarLinkAdmin}>Events</a>
              <a href="/admin/volunteers" className={styles.topBarLinkAdmin}>Volunteers</a>
              <a href="/admin/venues"     className={styles.topBarLinkAdmin}>Venues</a>
              <a href="/admin/staff"      className={styles.topBarLinkAdmin}>Staff</a>
              <a href="/admin/job-types"  className={styles.topBarLinkAdmin}>Job Types</a>
              <a href="/admin/feedback"   className={styles.topBarLinkAdmin}>Feedback</a>
            </nav>
          </>
        )}
      </div>

      {/* Gear-icon dropdown no longer needed — admin nav is explicit in the header */}
      <UserMenu name={userName} isAdmin={false} onSignOut={onSignOut} />
    </div>
  );
}
