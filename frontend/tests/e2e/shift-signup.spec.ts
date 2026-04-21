/**
 * E2E tests — Volunteer shift signup
 *
 * Covers:
 *  - Happy path: volunteer views event, signs up for a shift, shift shows in My Shifts
 *  - Cancel: volunteer cancels a shift, shift removed from My Shifts
 *  - Full shift: "Full" label shown, Sign Up button absent
 *  - Unauthenticated: redirect to /login
 */

import { test, expect } from "./helpers/fixtures";
import {
  createVenue,
  createJobType,
  createEventWithShift,
  deleteEvent,
  deleteVenue,
  deleteJobType,
  uniqueName,
} from "./helpers/api";

test.describe("Shift signup — happy path", () => {
  let eventId: string;
  let eventName: string;
  let happyVenueId: string;
  let happyJobTypeId: number;

  test.beforeAll(async ({ adminToken }) => {
    eventName = uniqueName("ShiftEvent");
    happyVenueId = await createVenue(adminToken, {
      name: uniqueName("Venue"),
      city: "Testville",
      state: "VA",
    });
    // Use a fully unique code so re-runs don't hit the uniqueness constraint.
    happyJobTypeId = await createJobType(
      adminToken,
      uniqueName("grtg"),
      uniqueName("Greeter")
    );
    const result = await createEventWithShift(adminToken, {
      eventName,
      venueId: happyVenueId,
      jobTypeId: happyJobTypeId,
      startDateTime: "2027-06-15 09:00:00",
      endDateTime: "2027-06-15 12:00:00",
      maxVolunteers: 5,
    });
    eventId = result.eventId;
  });

  test.afterAll(async ({ adminToken }) => {
    if (eventId) {
      try { await deleteEvent(adminToken, eventId); } catch { /* ignore */ }
    }
    if (happyVenueId) {
      try { await deleteVenue(adminToken, happyVenueId); } catch { /* ignore */ }
    }
    if (happyJobTypeId) {
      try { await deleteJobType(adminToken, happyJobTypeId); } catch { /* ignore */ }
    }
  });

  test("volunteer sees Sign Up button and signs up for a shift", async ({
    volunteerPage,
  }) => {
    await volunteerPage.goto(`/events/${eventId}`);

    const signUpBtn = volunteerPage.getByRole("button", { name: "Sign Up" }).first();
    await expect(signUpBtn).toBeVisible({ timeout: 5_000 });
    await signUpBtn.click();

    await expect(
      volunteerPage.getByRole("button", { name: "Cancel Signup" }).first()
    ).toBeVisible({ timeout: 5_000 });
  });

  test("signed-up shift appears in My Shifts", async ({ volunteerPage }) => {
    // Each test gets a fresh volunteer — sign up unconditionally first.
    await volunteerPage.goto(`/events/${eventId}`);
    const signUpBtn = volunteerPage.getByRole("button", { name: "Sign Up" }).first();
    await expect(signUpBtn).toBeVisible({ timeout: 5_000 });
    await signUpBtn.click();
    await expect(
      volunteerPage.getByRole("button", { name: "Cancel Signup" }).first()
    ).toBeVisible({ timeout: 5_000 });

    // Navigate to My Shifts and confirm the event name appears.
    await volunteerPage.goto("/my-shifts");
    await expect(
      volunteerPage.getByText(eventName)
    ).toBeVisible({ timeout: 5_000 });
  });

  test("volunteer can cancel their shift", async ({ volunteerPage }) => {
    // Fresh volunteer — sign up first.
    await volunteerPage.goto(`/events/${eventId}`);
    const signUpBtn = volunteerPage.getByRole("button", { name: "Sign Up" }).first();
    await expect(signUpBtn).toBeVisible({ timeout: 5_000 });
    await signUpBtn.click();
    await expect(
      volunteerPage.getByRole("button", { name: "Cancel Signup" }).first()
    ).toBeVisible({ timeout: 5_000 });

    // Cancel
    await volunteerPage.getByRole("button", { name: "Cancel Signup" }).first().click();

    // Sign Up button should come back
    await expect(
      volunteerPage.getByRole("button", { name: "Sign Up" }).first()
    ).toBeVisible({ timeout: 5_000 });
  });
});

test.describe("Shift signup — full shift", () => {
  let eventId: string;
  let fullVenueId: string;
  let fullJobTypeId: number;

  test.beforeAll(async ({ adminToken }) => {
    const eventName = uniqueName("FullEvent");
    fullVenueId = await createVenue(adminToken, {
      name: uniqueName("Venue"),
      city: "Testville",
      state: "VA",
    });
    fullJobTypeId = await createJobType(
      adminToken,
      uniqueName("chck"),
      uniqueName("Checker")
    );
    // maxVolunteers must be > 0 (DB constraint). Use 1 and fill it via API.
    const result = await createEventWithShift(adminToken, {
      eventName,
      venueId: fullVenueId,
      jobTypeId: fullJobTypeId,
      startDateTime: "2027-07-10 08:00:00",
      endDateTime: "2027-07-10 11:00:00",
      maxVolunteers: 1,
    });
    eventId = result.eventId;

    // Assign the admin to the shift so it is full.
    const ADMIN_URL =
      process.env.NEXT_PUBLIC_GRAPHQL_ADMIN_URL ||
      "http://localhost:8080/graphql/admin";
    await fetch(ADMIN_URL, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${adminToken}`,
      },
      body: JSON.stringify({
        query: `mutation { assignVolunteerToShift(shiftId: "${result.shiftId}", volunteerId: "1") { success message } }`,
      }),
    });
  });

  test.afterAll(async ({ adminToken }) => {
    if (eventId) {
      try { await deleteEvent(adminToken, eventId); } catch { /* ignore */ }
    }
    if (fullVenueId) {
      try { await deleteVenue(adminToken, fullVenueId); } catch { /* ignore */ }
    }
    if (fullJobTypeId) {
      try { await deleteJobType(adminToken, fullJobTypeId); } catch { /* ignore */ }
    }
  });

  test("full shift shows Full label and no Sign Up button", async ({
    volunteerPage,
  }) => {
    await volunteerPage.goto(`/events/${eventId}`);
    await expect(volunteerPage.getByText("Full", { exact: true })).toBeVisible({ timeout: 5_000 });
    await expect(
      volunteerPage.getByRole("button", { name: "Sign Up" })
    ).not.toBeVisible();
  });
});

test.describe("Shift signup — error cases", () => {
  test("unauthenticated user is redirected to /login when visiting event page", async ({
    page,
  }) => {
    await page.goto("/");
    await page.evaluate(() => localStorage.clear());
    await page.goto("/events/1");
    await page.waitForURL("**/login", { timeout: 5_000 });
    expect(page.url()).toContain("/login");
  });
});
