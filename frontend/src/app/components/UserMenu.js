"use client";

import { useState, useEffect, useRef } from "react";
import styles from "./UserMenu.module.css";

/**
 * UserMenu
 *
 * Shows the user's display name. If the user is an admin, also renders a
 * gear icon that opens a dropdown for admin navigation links. Sign out is
 * always available.
 *
 * Props:
 *   name       {string}   — display name from localStorage
 *   isAdmin    {boolean}  — whether to show the gear icon
 *   onSignOut  {function} — called when the user clicks Sign Out
 *
 * Admin nav items are defined in the ADMIN_ITEMS array below.
 * Add { label, href } entries there as admin pages are built.
 */

// ---- Admin navigation items ----
// Add entries here as admin pages are built, e.g.:
// { label: "Manage Events", href: "/admin/events" },
const ADMIN_ITEMS = [
  { label: "Manage Events", href: "/admin/events" },
  { label: "Manage Venues", href: "/admin/venues" },
];

export default function UserMenu({ name, isAdmin, onSignOut }) {
  const [open, setOpen] = useState(false);
  const wrapperRef = useRef(null);

  // Close the dropdown when the user clicks anywhere outside it.
  useEffect(() => {
    if (!open) return;
    function handleOutsideClick(e) {
      if (wrapperRef.current && !wrapperRef.current.contains(e.target)) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", handleOutsideClick);
    return () => document.removeEventListener("mousedown", handleOutsideClick);
  }, [open]);

  return (
    <div className={styles.container}>
      {name && <span className={styles.name}>{name}</span>}

      {isAdmin && (
        <div className={styles.gearWrapper} ref={wrapperRef}>
          <button
            className={`${styles.gearButton} ${open ? styles.open : ""}`}
            onClick={() => setOpen((v) => !v)}
            aria-label="Admin menu"
            title="Admin menu"
          >
            ⚙
          </button>

          {open && (
            <div className={styles.dropdown}>
              <div className={styles.dropdownHeader}>Admin</div>
              {ADMIN_ITEMS.length === 0 ? (
                <div className={styles.dropdownEmpty}>
                  More pages coming soon
                </div>
              ) : (
                ADMIN_ITEMS.map((item) => (
                  <a
                    key={item.href}
                    href={item.href}
                    className={styles.dropdownItem}
                    onClick={() => setOpen(false)}
                  >
                    {item.label}
                  </a>
                ))
              )}
            </div>
          )}
        </div>
      )}

      <button className={styles.signOutButton} onClick={onSignOut}>
        Sign out
      </button>
    </div>
  );
}
