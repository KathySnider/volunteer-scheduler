"use client";

import { useEffect } from "react";
import { usePathname } from "next/navigation";
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
  const pathname = usePathname();

  // Returns the CSS class(es) for a nav link — adds .active only on an exact
  // match so that detail pages (e.g. /admin/feedback/42) keep the nav link
  // clickable as a "back to list" affordance.
  const linkClass = (base, href) =>
    pathname === href ? `${styles[base]} ${styles.active}` : styles[base];

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
        <a href="/events" className={linkClass("appTitle", "/events")}>Volunteer Scheduler</a>

        {/* Volunteer nav — always visible for every role on every page */}
        <nav className={styles.topBarNav}>
          <a href="/my-shifts"   className={linkClass("topBarLink", "/my-shifts")}>My Shifts</a>
          <a href="/profile"     className={linkClass("topBarLink", "/profile")}>My Profile</a>
          <button
            type="button"
            className={styles.topBarLink}
            onClick={onFeedbackOpen}
          >
            Submit Feedback
          </button>
          <a href="/my-feedback" className={linkClass("topBarLink", "/my-feedback")}>My Feedback</a>
        </nav>

        {/* Admin section nav — only shown when signed in as admin */}
        {isAdmin && (
          <>
            <div className={styles.navDivider} />
            <nav className={styles.topBarNav}>
              <a href="/admin/events"     className={linkClass("topBarLinkAdmin", "/admin/events")}>Events</a>
              <a href="/admin/volunteers" className={linkClass("topBarLinkAdmin", "/admin/volunteers")}>Volunteers</a>
              <a href="/admin/venues"     className={linkClass("topBarLinkAdmin", "/admin/venues")}>Venues</a>
              <a href="/admin/staff"      className={linkClass("topBarLinkAdmin", "/admin/staff")}>Staff</a>
              <a href="/admin/job-types"  className={linkClass("topBarLinkAdmin", "/admin/job-types")}>Job Types</a>
              <a href="/admin/feedback"   className={linkClass("topBarLinkAdmin", "/admin/feedback")}>Feedback</a>
            </nav>
          </>
        )}
      </div>

      {/* Gear-icon dropdown no longer needed — admin nav is explicit in the header */}
      <UserMenu name={userName} isAdmin={false} onSignOut={onSignOut} />
    </div>
  );
}
