"use client";

import UserMenu from "./UserMenu";
import styles from "./admin-top-bar.module.css";

export default function AdminTopBar({ userName, onSignOut, onFeedbackOpen }) {
  return (
    <div className={styles.topBar}>
      <div className={styles.topBarLeft}>
        <div className={styles.appTitle}>Volunteer Scheduler</div>
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
      </div>
      <UserMenu name={userName} isAdmin={false} onSignOut={onSignOut} />
    </div>
  );
}
