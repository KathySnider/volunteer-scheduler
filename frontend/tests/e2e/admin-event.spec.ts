/**
 * E2E tests — Admin creates an event with shifts
 *
 * Covers:
 *  - Happy path: admin fills in the new event form and submits → event appears in list
 *  - Happy path: admin creates an opportunity (job+shifts) on the event
 *  - Error: submit with missing required fields shows validation errors
 *  - Non-admin (volunteer) cannot reach admin pages — redirect to /events
 */

import { test, expect } from "./helpers/fixtures";
import {
  createVenue,
  createJobType,
  uniqueName,
} from "./helpers/api";

test.describe("Admin event creation — happy path", () => {
  test("admin can navigate to the new event form", async ({ adminPage }) => {
    await adminPage.goto("/admin/events");
    await expect(
      adminPage.getByRole("link", { name: /create new event/i })
    ).toBeVisible({ timeout: 5_000 });

    await adminPage.getByRole("link", { name: /create new event/i }).click();
    await adminPage.waitForURL("**/admin/events/new", { timeout: 5_000 });
  });

  test("admin fills in and submits the new event form", async ({
    adminPage,
    adminToken,
  }) => {
    const eventName = uniqueName("AdminCreatedEvent");
    const venueName = uniqueName("TestVenue");

    // Pre-create a venue so it appears in the search dropdown.
    await createVenue(adminToken, {
      name: venueName,
      city: "Washington",
      state: "DC",
      ianaZone: "America/New_York",
    });

    await adminPage.goto("/admin/events/new");

    // Wait for the form to be ready.
    const nameInput = adminPage.getByPlaceholder(/medicare|workshop/i);
    await expect(nameInput).toBeVisible({ timeout: 10_000 });
    await nameInput.fill(eventName);

    // Select In-Person format.
    await adminPage.getByRole("radio", { name: /in.person/i }).check();

    // Select the venue via the search dropdown.
    const venueSearch = adminPage.getByPlaceholder(/search venues/i);
    await venueSearch.fill(venueName);
    await adminPage.getByText(venueName).first().click();

    // Fill in one event date.
    // TimeInput commits its value on blur, so press Tab after each fill.
    // Use different start/end dates as a safety net against start===end validation.
    const dateInputs = adminPage.locator('input[type="date"]');
    await dateInputs.first().fill("2027-09-20");

    const timeInputs = adminPage.locator('input[placeholder="h:MM"]');
    await timeInputs.nth(0).fill("9:00");
    await timeInputs.nth(0).press("Tab"); // commit via onBlur

    await dateInputs.nth(1).fill("2027-09-21"); // end date one day later
    await timeInputs.nth(1).fill("5:00");
    await timeInputs.nth(1).press("Tab"); // commit via onBlur

    // Submit.
    await adminPage.getByRole("button", { name: /create event/i }).click();

    // Success card shows "Event Created!"
    await expect(
      adminPage.getByText("Event Created!")
    ).toBeVisible({ timeout: 8_000 });
  });

  test("created event appears in the admin event list", async ({
    adminPage,
    adminToken,
  }) => {
    const eventName = uniqueName("ListedEvent");
    const venueId = await createVenue(adminToken, {
      name: uniqueName("ListVenue"),
      city: "Baltimore",
      state: "MD",
    });
    const jobTypeId = await createJobType(adminToken, uniqueName("tblr"), uniqueName("Tabling"));

    const { createEventWithShift } = await import("./helpers/api");
    await createEventWithShift(adminToken, {
      eventName,
      venueId,
      jobTypeId,
      startDateTime: "2027-10-05 10:00:00",
      endDateTime: "2027-10-05 14:00:00",
    });

    await adminPage.goto("/admin/events");
    await expect(adminPage.getByText(eventName)).toBeVisible({ timeout: 5_000 });
  });
});

test.describe("Admin event creation — validation", () => {
  test("submitting with empty event name shows validation", async ({
    adminPage,
  }) => {
    await adminPage.goto("/admin/events/new");

    // Click Create Event without filling anything in
    await adminPage.getByRole("button", { name: /create event/i }).click();

    // The form uses JS validation and renders an inline error message
    await expect(
      adminPage.getByText("Event name is required.")
    ).toBeVisible({ timeout: 5_000 });
  });
});

test.describe("Admin event creation — access control", () => {
  test("volunteer user is redirected away from admin events page", async ({
    volunteerPage,
  }) => {
    await volunteerPage.goto("/admin/events");
    // Should redirect to /events or /login — not stay on the admin page
    // Use a predicate so /admin/events doesn't accidentally satisfy the condition.
    await volunteerPage.waitForURL(
      (u) => u.pathname === "/events" || u.pathname === "/login",
      { timeout: 5_000 }
    );
    expect(volunteerPage.url()).not.toContain("/admin");
  });

  test("volunteer user cannot access new event form", async ({
    volunteerPage,
  }) => {
    await volunteerPage.goto("/admin/events/new");
    // Use a predicate so /admin/events doesn't accidentally satisfy the condition.
    await volunteerPage.waitForURL(
      (u) => u.pathname === "/events" || u.pathname === "/login",
      { timeout: 5_000 }
    );
    expect(volunteerPage.url()).not.toContain("/admin");
  });

  test("unauthenticated user is redirected from admin pages to /login", async ({
    page,
  }) => {
    await page.goto("/");
    await page.evaluate(() => localStorage.clear());
    await page.goto("/admin/events");
    await page.waitForURL("**/login", { timeout: 5_000 });
    expect(page.url()).toContain("/login");
  });
});
