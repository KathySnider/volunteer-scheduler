"use client";

import { usePathname } from "next/navigation";
import styles from "./admin-tabs.module.css";

const TABS = [
  { label: "Volunteer Events",    href: "/events" },
  { label: "Events",             href: "/admin/events" },
  { label: "Volunteers",         href: "/admin/volunteers" },
  { label: "Venues",             href: "/admin/venues" },
  { label: "Staff",              href: "/admin/staff" },
  { label: "Job Types",          href: "/admin/job-types" },
  { label: "Feedback",           href: "/admin/feedback" },
];

export default function AdminTabs() {
  const pathname = usePathname();

  return (
    <nav className={styles.tabBar} aria-label="Admin navigation">
      {TABS.map(({ label, href }) => {
        const isActive = href === "/events"
          ? pathname === "/events"
          : pathname.startsWith(href);
        return (
          <a
            key={href}
            href={href}
            className={`${styles.tab} ${isActive ? styles.tabActive : ""}`}
          >
            {label}
          </a>
        );
      })}
    </nav>
  );
}
