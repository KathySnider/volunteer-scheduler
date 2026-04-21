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
  createEventWithoutShifts,
  findEventIdByName,
  deleteEvent,
  deleteVenue,
  deleteJobType,
  uniqueName,
} from "./helpers/api";

test.describe("Admin event creation — happy path", () => {
  let happyVenueId: string;
  let happyEventName: string;
  let listedVenueId: string;
  let listedEventName: string;
  let listedJobTypeId: number;

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
    happyEventName = uniqueName("AdminCreatedEvent");
    const venueName = uniqueName("TestVenue");

    // Pre-create a venue so it appears in the search dropdown.
    happyVenueId = await createVenue(adminToken, {
      name: venueName,
      city: "Washington",
      state: "DC",
      ianaZone: "America/New_York",
    });

    await adminPage.goto("/admin/events/new");

    // Wait for the form to be ready.
    const nameInput = adminPage.getByPlaceholder(/medicare|workshop/i);
    await expect(nameInput).toBeVisible({ timeout: 10_000 });
    await nameInput.fill(happyEventName);

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
    listedEventName = uniqueName("ListedEvent");
    listedVenueId = await createVenue(adminToken, {
      name: uniqueName("ListVenue"),
      city: "Baltimore",
      state: "MD",
    });
    listedJobTypeId = await createJobType(adminToken, uniqueName("tblr"), uniqueName("Tabling"));

    await createEventWithShift(adminToken, {
      eventName: listedEventName,
      venueId: listedVenueId,
      jobTypeId: listedJobTypeId,
      startDateTime: "2027-10-05 10:00:00",
      endDateTime: "2027-10-05 14:00:00",
    });

    await adminPage.goto("/admin/events");
    await expect(adminPage.getByText(listedEventName)).toBeVisible({ timeout: 5_000 });
  });

  test.afterAll(async ({ adminToken }) => {
    // Delete events first (FK on venue), then venues, then job types.
    for (const name of [happyEventName, listedEventName].filter(Boolean)) {
      try {
        const id = await findEventIdByName(adminToken, name);
        if (id) await deleteEvent(adminToken, id);
      } catch { /* ignore */ }
    }
    if (happyVenueId) {
      try { await deleteVenue(adminToken, happyVenueId); } catch { /* ignore */ }
    }
    if (listedVenueId) {
      try { await deleteVenue(adminToken, listedVenueId); } catch { /* ignore */ }
    }
    if (listedJobTypeId) {
      try { await deleteJobType(adminToken, listedJobTypeId); } catch { /* ignore */ }
    }
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
  let cityFilterVenueId: string;
  let cityFilterJobTypeId: number;
  let cityFilterUpcomingEventId: string;
  let cityFilterPastEventId: string;

  test.beforeAll(async ({ adminToken }) => {
    filterCity        = uniqueName("AdminFilterCity");
    upcomingEventName = uniqueName("AdminUpcomingEvent");
    pastEventName     = uniqueName("AdminPastEvent");

    cityFilterJobTypeId = await createJobType(
      adminToken,
      uniqueName("afl"),
      uniqueName("Admin Filter Role"),
    );
    cityFilterVenueId = await createVenue(adminToken, {
      name: uniqueName("AdminFilterVenue"),
      city: filterCity,
      state: "WA",
    });

    ({ eventId: cityFilterUpcomingEventId } = await createEventWithShift(adminToken, {
      eventName: upcomingEventName,
      venueId: cityFilterVenueId,
      jobTypeId: cityFilterJobTypeId,
      startDateTime: "2027-08-05 09:00:00",
      endDateTime:   "2027-08-05 13:00:00",
    }));

    ({ eventId: cityFilterPastEventId } = await createEventWithShift(adminToken, {
      eventName: pastEventName,
      venueId: cityFilterVenueId,
      jobTypeId: cityFilterJobTypeId,
      startDateTime: "2020-04-10 09:00:00",
      endDateTime:   "2020-04-10 13:00:00",
    }));
  });

  test.afterAll(async ({ adminToken }) => {
    for (const id of [cityFilterUpcomingEventId, cityFilterPastEventId].filter(Boolean)) {
      try { await deleteEvent(adminToken, id); } catch { /* ignore */ }
    }
    if (cityFilterVenueId) {
      try { await deleteVenue(adminToken, cityFilterVenueId); } catch { /* ignore */ }
    }
    if (cityFilterJobTypeId) {
      try { await deleteJobType(adminToken, cityFilterJobTypeId); } catch { /* ignore */ }
    }
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
  let tfVenueId: string;
  let tfJobTypeId: number;
  let tfUpcomingEventId: string;
  let tfPastEventId: string;

  test.beforeAll(async ({ adminToken }) => {
    tfCity         = uniqueName("AdminTFCity");
    tfUpcomingName = uniqueName("AdminTFUpcoming");
    tfPastName     = uniqueName("AdminTFPast");

    tfJobTypeId = await createJobType(
      adminToken,
      uniqueName("atf"),
      uniqueName("Admin TF Role"),
    );
    tfVenueId = await createVenue(adminToken, {
      name: uniqueName("AdminTFVenue"),
      city: tfCity,
      state: "OR",
    });

    ({ eventId: tfUpcomingEventId } = await createEventWithShift(adminToken, {
      eventName: tfUpcomingName,
      venueId: tfVenueId,
      jobTypeId: tfJobTypeId,
      startDateTime: "2027-09-01 09:00:00",
      endDateTime:   "2027-09-01 13:00:00",
    }));

    ({ eventId: tfPastEventId } = await createEventWithShift(adminToken, {
      eventName: tfPastName,
      venueId: tfVenueId,
      jobTypeId: tfJobTypeId,
      startDateTime: "2020-05-15 09:00:00",
      endDateTime:   "2020-05-15 13:00:00",
    }));
  });

  test.afterAll(async ({ adminToken }) => {
    for (const id of [tfUpcomingEventId, tfPastEventId].filter(Boolean)) {
      try { await deleteEvent(adminToken, id); } catch { /* ignore */ }
    }
    if (tfVenueId) {
      try { await deleteVenue(adminToken, tfVenueId); } catch { /* ignore */ }
    }
    if (tfJobTypeId) {
      try { await deleteJobType(adminToken, tfJobTypeId); } catch { /* ignore */ }
    }
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
  let fmtVenueId: string;
  let fmtJobTypeId: number;
  let fmtEventId: string;

  test.beforeAll(async ({ adminToken }) => {
    fmtCity      = uniqueName("AdminFmtCity");
    inPersonName = uniqueName("AdminInPersonEvent");

    fmtJobTypeId = await createJobType(
      adminToken,
      uniqueName("afmt"),
      uniqueName("Admin Fmt Role"),
    );
    fmtVenueId = await createVenue(adminToken, {
      name: uniqueName("AdminFmtVenue"),
      city: fmtCity,
      state: "CA",
    });

    ({ eventId: fmtEventId } = await createEventWithShift(adminToken, {
      eventName: inPersonName,
      venueId: fmtVenueId,
      jobTypeId: fmtJobTypeId,
      startDateTime: "2027-10-01 09:00:00",
      endDateTime:   "2027-10-01 13:00:00",
    }));
  });

  test.afterAll(async ({ adminToken }) => {
    if (fmtEventId) {
      try { await deleteEvent(adminToken, fmtEventId); } catch { /* ignore */ }
    }
    if (fmtVenueId) {
      try { await deleteVenue(adminToken, fmtVenueId); } catch { /* ignore */ }
    }
    if (fmtJobTypeId) {
      try { await deleteJobType(adminToken, fmtJobTypeId); } catch { /* ignore */ }
    }
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
  let noShiftsVenueId: string;

  test.beforeAll(async ({ adminToken }) => {
    noShiftsName = uniqueName("AdminNoShiftsEvent");
    noShiftsVenueId = await createVenue(adminToken, {
      name: uniqueName("AdminNoShiftsVenue"),
      city: uniqueName("AdminNoShiftsCity"),
      state: "TX",
    });
    await createEventWithoutShifts(adminToken, { eventName: noShiftsName, venueId: noShiftsVenueId });
  });

  test.afterAll(async ({ adminToken }) => {
    try {
      const id = await findEventIdByName(adminToken, noShiftsName);
      if (id) await deleteEvent(adminToken, id);
    } catch { /* ignore */ }
    if (noShiftsVenueId) {
      try { await deleteVenue(adminToken, noShiftsVenueId); } catch { /* ignore */ }
    }
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

test.describe("Manage Events listing — Region column", () => {
  let regionEventName: string;
  let regionVenueId: string;

  test.beforeAll(async ({ adminToken }) => {
    regionEventName = uniqueName("AdminRegionEvent");
    regionVenueId = await createVenue(adminToken, {
      name: uniqueName("AdminRegionVenue"),
      city: uniqueName("AdminRegionCity"),
      state: "WA",
    });
    // createEventWithoutShifts seeds fundingEntityId: 1 = "Seattle Area" (migration 000006).
    await createEventWithoutShifts(adminToken, { eventName: regionEventName, venueId: regionVenueId });
  });

  test.afterAll(async ({ adminToken }) => {
    try {
      const id = await findEventIdByName(adminToken, regionEventName);
      if (id) await deleteEvent(adminToken, id);
    } catch { /* ignore */ }
    if (regionVenueId) {
      try { await deleteVenue(adminToken, regionVenueId); } catch { /* ignore */ }
    }
  });

  test("event row shows its funding entity name in the Region column", async ({
    adminPage,
  }) => {
    await adminPage.goto("/admin/events");
    await expect(
      adminPage.getByRole("heading", { name: /manage events/i })
    ).toBeVisible({ timeout: 8_000 });

    const row = adminPage.locator("tr").filter({ hasText: regionEventName });
    await expect(row).toBeVisible({ timeout: 5_000 });
    await expect(row.getByText("Seattle Area")).toBeVisible();
  });
});
