/**
 * E2E tests — "Submit Feedback" entry-point presence
 *
 * Regression guard: verifies that a "Submit Feedback" trigger is visible on
 * every page that should have one.
 *
 * Volunteer pages:  uses the floating FeedbackButton (aria-label "Submit feedback").
 * Admin pages:      uses the "Submit Feedback" button in the persistent AdminTopBar
 *                   header nav (no floating button on admin pages).
 *
 * Add a new entry to VOLUNTEER_PAGES or ADMIN_PAGES whenever a new page is
 * added to the app.
 */

import { test, expect } from "./helpers/fixtures";

// ---------------------------------------------------------------------------
// Pages that a logged-in volunteer should see the floating button on
// ---------------------------------------------------------------------------
const VOLUNTEER_PAGES = [
  "/events",
  "/my-shifts",
  "/profile",
  "/my-feedback",
];

// ---------------------------------------------------------------------------
// Pages that a logged-in admin should see the AdminTopBar "Submit Feedback"
// nav link on (no floating button — feedback is in the persistent header)
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

/**
 * On volunteer pages: the floating FeedbackButton (aria-label "Submit feedback").
 * On admin pages:     the AdminTopBar nav button (text "Submit Feedback").
 * Playwright name-matching is case-insensitive, so one locator covers both.
 */
const feedbackTrigger = (page: import("@playwright/test").Page) =>
  page.getByRole("button", { name: /submit feedback/i });

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

test.describe("Submit Feedback — volunteer pages (floating button)", () => {
  for (const path of VOLUNTEER_PAGES) {
    test(`visible on ${path}`, async ({ volunteerPage }) => {
      await volunteerPage.goto(path);
      await expect(feedbackTrigger(volunteerPage)).toBeVisible({ timeout: 8_000 });
    });
  }
});

test.describe("Submit Feedback — admin pages (AdminTopBar nav link)", () => {
  for (const path of ADMIN_PAGES) {
    test(`visible on ${path}`, async ({ adminPage }) => {
      await adminPage.goto(path);
      await expect(feedbackTrigger(adminPage)).toBeVisible({ timeout: 8_000 });
    });
  }
});
