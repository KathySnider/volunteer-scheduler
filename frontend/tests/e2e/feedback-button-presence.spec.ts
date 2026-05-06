/**
 * E2E tests — FeedbackButton presence
 *
 * Regression guard: verifies the floating feedback button is visible on every
 * page that should have it.  Add a new entry to VOLUNTEER_PAGES or ADMIN_PAGES
 * whenever a new page is added to the app.
 */

import { test, expect } from "./helpers/fixtures";

// ---------------------------------------------------------------------------
// Pages that a logged-in volunteer should see the button on
// ---------------------------------------------------------------------------
const VOLUNTEER_PAGES = [
  "/events",
  "/my-shifts",
  "/profile",
  "/my-feedback",
];

// ---------------------------------------------------------------------------
// Pages that a logged-in admin should see the button on
// ---------------------------------------------------------------------------
const ADMIN_PAGES = [
  "/admin/events",
  "/admin/events/new",
  "/admin/feedback",
  "/admin/job-types",
  "/admin/staff",
  "/admin/venues",
  "/admin/volunteers",
];

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Locator for the floating feedback button (aria-label set in FeedbackButton.js). */
const feedbackBtn = (page: import("@playwright/test").Page) =>
  page.getByRole("button", { name: "Submit feedback" });

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

test.describe("FeedbackButton — volunteer pages", () => {
  for (const path of VOLUNTEER_PAGES) {
    test(`visible on ${path}`, async ({ volunteerPage }) => {
      await volunteerPage.goto(path);
      await expect(feedbackBtn(volunteerPage)).toBeVisible({ timeout: 8_000 });
    });
  }
});

test.describe("FeedbackButton — admin pages", () => {
  for (const path of ADMIN_PAGES) {
    test(`visible on ${path}`, async ({ adminPage }) => {
      await adminPage.goto(path);
      await expect(feedbackBtn(adminPage)).toBeVisible({ timeout: 8_000 });
    });
  }
});
