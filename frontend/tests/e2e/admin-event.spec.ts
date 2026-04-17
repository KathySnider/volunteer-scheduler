/**
 * E2E tests — Admin creates an event with shifts
 *
 * Covers:
 *  - Happy path: admin fills in the new event form and submits → event appears in list
 *  - Happy path: admin creates an opportunity (job+shifts) on the event
 *  - Error: submit with missing required fields shows validation errors
 *  - Non-admin (volunteer) cannot reach admin pages — redirect to /events
 *  - Manage Events listing page filters (cities, timeframe, format, reset, "No shifts" badge)
 */

import { test, expect } from "./helpers/fixtures";
import {
  createVenue,
  createJobType,
  createEventWithShift,
  uniqueName,
} from "./helpers/api";

const ADMIN_URL =
  process.env.NEXT_PUBLIC_GRAPHQL_ADMIN_URL ||
  "http://localhost:8080/graphql/admin";

/** Create a bare event (no opportunities / shifts) and return its name. */
async function createEventOnly(
  adminToken: string,
  eventName: string,
  venueId: string
): Promise<void> {
  await fetch(ADMIN_URL, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${adminToken}`,
    },
    body: JSON.stringify({
      query: `mutation CreateEvent($e: NewEventInput!) { createEvent(newEvent: $e) { success message } }`,
      variables: {
        e: {
          name: eventName,
          description: "Test event — no shifts",
          eventType: "IN_PERSON",
          venueId,
          serviceTypes: [],
          eventDates: [
            {
              startDateTime: "2027-11-01 09:00:00",
              endDateTime:   "2027-11-01 13:00:00",
              ianaZone: "America/New_York",
            },
          ],
        },
      },
    }),
  });
}

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

/* ------------------------------------------------------------------ */
/*  Manage Events listing — filter bar                                  */
/* ------------------------------------------------------------------ */

test.describe("Manage Events listing — defaults and event count", () => {
  test("page defaults to ALL timeframe", async ({ adminPage }) => {
    await adminPage.goto("/admin/events");
    await expect(
      adminPage.getByRole("heading", { name: /manage events/i })
    ).toBeVisible({ timeout: 8_000 });
    await expect(adminPage.locator("#adminTimeFrameFilter")).toHaveValue("ALL");
  });

  test("event count appears in the heading after load", async ({ adminPage }) => {
    await adminPage.goto("/admin/events");
    await expect(
      adminPage.locator("h1").filter({ hasText: /\(\d+\)/ })
    ).toBeVisible({ timeout: 8_000 });
  });
});

test.describe("Manage Events listing — city filter", () => {
  let filterCity: string;
  let upcomingEventName: string;
  let pastEventName: string;

  test.beforeAll(async ({ adminToken }) => {
    filterCity        = uniqueName("AdminFilterCity");
    upcomingEventName = uniqueName("AdminUpcomingEvent");
    pastEventName     = uniqueName("AdminPastEvent");

    const jobTypeId = await createJobType(
      adminToken,
      uniqueName("afl"),
      uniqueName("Admin Filter Role"),
    );
    const venueId = await createVenue(adminToken, {
      name: uniqueName("AdminFilterVenue"),
      city: filterCity,
      state: "WA",
    });

    await createEventWithShift(adminToken, {
      eventName: upcomingEventName,
      venueId,
      jobTypeId,
      startDateTime: "2027-08-05 09:00:00",
      endDateTime:   "2027-08-05 13:00:00",
    });

    await createEventWithShift(adminToken, {
      eventName: pastEventName,
      venueId,
      jobTypeId,
      startDateTime: "2020-04-10 09:00:00",
      endDateTime:   "2020-04-10 13:00:00",
    });
  });

  test("selecting a city shows only that city's events", async ({ adminPage }) => {
    await adminPage.goto("/admin/events");
    await expect(
      adminPage.getByRole("heading", { name: /manage events/i })
    ).toBeVisible({ timeout: 8_000 });

    // Default is ALL — both events should appear after selecting the city.
    await adminPage.getByRole("button", { name: "All Cities" }).click();
    await adminPage.getByLabel(filterCity).check();

    await expect(adminPage.getByText(upcomingEventName)).toBeVisible({ timeout: 5_000 });
    await expect(adminPage.getByText(pastEventName)).toBeVisible({ timeout: 5_000 });
  });

  test("Reset filters button appears when a city is selected", async ({ adminPage }) => {
    await adminPage.goto("/admin/events");
    await expect(
      adminPage.getByRole("heading", { name: /manage events/i })
    ).toBeVisible({ timeout: 8_000 });

    // Should not be visible before any filter is active.
    await expect(adminPage.getByRole("button", { name: "Reset filters" })).not.toBeVisible();

    await adminPage.getByRole("button", { name: "All Cities" }).click();
    await adminPage.getByLabel(filterCity).check();

    await expect(adminPage.getByRole("button", { name: "Reset filters" })).toBeVisible();
  });

  test("Reset filters clears city selection and restores all events", async ({ adminPage }) => {
    await adminPage.goto("/admin/events");
    await expect(
      adminPage.getByRole("heading", { name: /manage events/i })
    ).toBeVisible({ timeout: 8_000 });

    await adminPage.getByRole("button", { name: "All Cities" }).click();
    await adminPage.getByLabel(filterCity).check();
    // Close the panel before clicking reset (avoids panel obscuring button).
    await adminPage.keyboard.press("Escape");

    await adminPage.getByRole("button", { name: "Reset filters" }).click();

    // Reset button should disappear.
    await expect(adminPage.getByRole("button", { name: "Reset filters" })).not.toBeVisible();
    // Both events still visible (now unfiltered).
    await expect(adminPage.getByText(upcomingEventName)).toBeVisible({ timeout: 5_000 });
    await expect(adminPage.getByText(pastEventName)).toBeVisible({ timeout: 5_000 });
  });
});

test.describe("Manage Events listing — timeframe filter", () => {
  let tfCity: string;
  let tfUpcomingName: string;
  let tfPastName: string;

  test.beforeAll(async ({ adminToken }) => {
    tfCity         = uniqueName("AdminTFCity");
    tfUpcomingName = uniqueName("AdminTFUpcoming");
    tfPastName     = uniqueName("AdminTFPast");

    const jobTypeId = await createJobType(
      adminToken,
      uniqueName("atf"),
      uniqueName("Admin TF Role"),
    );
    const venueId = await createVenue(adminToken, {
      name: uniqueName("AdminTFVenue"),
      city: tfCity,
      state: "OR",
    });

    await createEventWithShift(adminToken, {
      eventName: tfUpcomingName,
      venueId,
      jobTypeId,
      startDateTime: "2027-09-01 09:00:00",
      endDateTime:   "2027-09-01 13:00:00",
    });

    await createEventWithShift(adminToken, {
      eventName: tfPastName,
      venueId,
      jobTypeId,
      startDateTime: "2020-05-15 09:00:00",
      endDateTime:   "2020-05-15 13:00:00",
    });
  });

  test("UPCOMING hides past events", async ({ adminPage }) => {
    await adminPage.goto("/admin/events");
    await expect(
      adminPage.getByRole("heading", { name: /manage events/i })
    ).toBeVisible({ timeout: 8_000 });

    await adminPage.getByRole("button", { name: "All Cities" }).click();
    await adminPage.getByLabel(tfCity).check();
    await adminPage.locator("#adminTimeFrameFilter").selectOption("UPCOMING");

    await expect(adminPage.getByText(tfUpcomingName)).toBeVisible({ timeout: 5_000 });
    await expect(adminPage.getByText(tfPastName)).not.toBeVisible();
  });

  test("PAST shows past events and hides upcoming", async ({ adminPage }) => {
    await adminPage.goto("/admin/events");
    await expect(
      adminPage.getByRole("heading", { name: /manage events/i })
    ).toBeVisible({ timeout: 8_000 });

    await adminPage.getByRole("button", { name: "All Cities" }).click();
    await adminPage.getByLabel(tfCity).check();
    await adminPage.locator("#adminTimeFrameFilter").selectOption("PAST");

    await expect(adminPage.getByText(tfPastName)).toBeVisible({ timeout: 5_000 });
    await expect(adminPage.getByText(tfUpcomingName)).not.toBeVisible();
  });

  test("ALL shows both past and upcoming events", async ({ adminPage }) => {
    await adminPage.goto("/admin/events");
    await expect(
      adminPage.getByRole("heading", { name: /manage events/i })
    ).toBeVisible({ timeout: 8_000 });

    await adminPage.getByRole("button", { name: "All Cities" }).click();
    await adminPage.getByLabel(tfCity).check();
    // ALL is the default — explicitly select it to be explicit.
    await adminPage.locator("#adminTimeFrameFilter").selectOption("ALL");

    await expect(adminPage.getByText(tfUpcomingName)).toBeVisible({ timeout: 5_000 });
    await expect(adminPage.getByText(tfPastName)).toBeVisible({ timeout: 5_000 });
  });
});

test.describe("Manage Events listing — format filter", () => {
  let fmtCity: string;
  let inPersonName: string;

  test.beforeAll(async ({ adminToken }) => {
    fmtCity      = uniqueName("AdminFmtCity");
    inPersonName = uniqueName("AdminInPersonEvent");

    const jobTypeId = await createJobType(
      adminToken,
      uniqueName("afmt"),
      uniqueName("Admin Fmt Role"),
    );
    const venueId = await createVenue(adminToken, {
      name: uniqueName("AdminFmtVenue"),
      city: fmtCity,
      state: "CA",
    });

    await createEventWithShift(adminToken, {
      eventName: inPersonName,
      venueId,
      jobTypeId,
      startDateTime: "2027-10-01 09:00:00",
      endDateTime:   "2027-10-01 13:00:00",
    });
  });

  test("IN_PERSON format filter shows in-person events", async ({ adminPage }) => {
    await adminPage.goto("/admin/events");
    await expect(
      adminPage.getByRole("heading", { name: /manage events/i })
    ).toBeVisible({ timeout: 8_000 });

    await adminPage.getByRole("button", { name: "All Cities" }).click();
    await adminPage.getByLabel(fmtCity).check();
    await adminPage.locator("#adminFormatFilter").selectOption("IN_PERSON");

    await expect(adminPage.getByText(inPersonName)).toBeVisible({ timeout: 5_000 });
  });

  test("VIRTUAL format filter hides in-person events", async ({ adminPage }) => {
    await adminPage.goto("/admin/events");
    await expect(
      adminPage.getByRole("heading", { name: /manage events/i })
    ).toBeVisible({ timeout: 8_000 });

    await adminPage.getByRole("button", { name: "All Cities" }).click();
    await adminPage.getByLabel(fmtCity).check();
    await adminPage.locator("#adminFormatFilter").selectOption("VIRTUAL");

    await expect(adminPage.getByText(inPersonName)).not.toBeVisible();
  });
});

test.describe("Manage Events listing — 'No shifts' badge", () => {
  let noShiftsName: string;

  test.beforeAll(async ({ adminToken }) => {
    noShiftsName = uniqueName("AdminNoShiftsEvent");
    const venueId = await createVenue(adminToken, {
      name: uniqueName("AdminNoShiftsVenue"),
      city: uniqueName("AdminNoShiftsCity"),
      state: "TX",
    });
    await createEventOnly(adminToken, noShiftsName, venueId);
  });

  test("event with no shifts shows 'No shifts' badge in the Volunteers column", async ({
    adminPage,
  }) => {
    await adminPage.goto("/admin/events");
    await expect(
      adminPage.getByRole("heading", { name: /manage events/i })
    ).toBeVisible({ timeout: 8_000 });

    // Find the row for our event and check for the badge.
    const row = adminPage.locator("tr").filter({ hasText: noShiftsName });
    await expect(row).toBeVisible({ timeout: 5_000 });
    await expect(row.getByText("No shifts")).toBeVisible();
  });
});
