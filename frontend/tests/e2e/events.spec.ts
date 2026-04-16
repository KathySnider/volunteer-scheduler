/**
 * E2E tests — Events listing page
 *
 * Covers:
 *  - lookupValues returns a cities list (API level)
 *  - Cities appear in the Cities dropdown panel
 *  - Events page loads and defaults to UPCOMING timeframe
 *  - City filter narrows results to the selected city
 *  - Timeframe filter: PAST shows past events, ALL shows both
 *  - Format filter: IN_PERSON shows only in-person events
 *  - Event count appears in the heading
 *  - Volunteer count displayed on event cards
 */

import { test, expect } from "./helpers/fixtures";
import {
  createVenue,
  createJobType,
  createEventWithShift,
  uniqueName,
} from "./helpers/api";

/* ------------------------------------------------------------------ */
/*  API helper                                                          */
/* ------------------------------------------------------------------ */

const VOLUNTEER_URL =
  process.env.NEXT_PUBLIC_GRAPHQL_VOLUNTEER_URL ||
  "http://localhost:8080/graphql/volunteer";

async function fetchLookupCities(token: string): Promise<string[]> {
  const res = await fetch(VOLUNTEER_URL, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ query: `query { lookupValues { cities } }` }),
  });
  const json = (await res.json()) as { data?: { lookupValues?: { cities?: string[] } } };
  return json.data?.lookupValues?.cities ?? [];
}

/* ------------------------------------------------------------------ */
/*  lookupValues — cities                                              */
/* ------------------------------------------------------------------ */

test.describe("lookupValues — cities", () => {
  test("API returns a non-empty list of city strings", async ({
    volunteerToken,
  }) => {
    const cities = await fetchLookupCities(volunteerToken);
    expect(cities.length).toBeGreaterThan(0);
    for (const city of cities) {
      expect(typeof city).toBe("string");
      expect(city.trim().length).toBeGreaterThan(0);
    }
  });

  test("cities are deduplicated and sorted alphabetically", async ({
    volunteerToken,
  }) => {
    const cities = await fetchLookupCities(volunteerToken);
    // No duplicates
    const unique = [...new Set(cities)];
    expect(cities.length).toBe(unique.length);
    // Sorted (case-insensitive check)
    for (let i = 1; i < cities.length; i++) {
      expect(cities[i].toLowerCase() >= cities[i - 1].toLowerCase()).toBe(true);
    }
  });

  test("Cities dropdown on events page lists cities from the API", async ({
    volunteerPage,
    volunteerToken,
  }) => {
    const cities = await fetchLookupCities(volunteerToken);
    const firstCity = cities[0];

    await volunteerPage.goto("/events");
    // Open the Cities multi-select panel
    await volunteerPage.getByRole("button", { name: "All Cities" }).click();
    // The first city should appear as a checkbox label in the panel
    await expect(volunteerPage.getByLabel(firstCity)).toBeVisible({ timeout: 5_000 });
  });
});

/* ------------------------------------------------------------------ */
/*  Events page — filtering                                            */
/* ------------------------------------------------------------------ */

test.describe("Events page — filtering", () => {
  let filterCity: string;
  let upcomingEventName: string;
  let pastEventName: string;

  test.beforeAll(async ({ adminToken }) => {
    filterCity       = uniqueName("FilterCity");
    upcomingEventName = uniqueName("UpcomingEvent");
    pastEventName    = uniqueName("PastEvent");

    const jobTypeId = await createJobType(
      adminToken,
      uniqueName("flt"),
      uniqueName("Filter Role"),
    );
    const venueId = await createVenue(adminToken, {
      name: uniqueName("FilterVenue"),
      city: filterCity,
      state: "WA",
    });

    // Upcoming in-person event in the unique test city.
    await createEventWithShift(adminToken, {
      eventName: upcomingEventName,
      venueId,
      jobTypeId,
      startDateTime: "2027-08-01 09:00:00",
      endDateTime: "2027-08-01 13:00:00",
      maxVolunteers: 4,
    });

    // Past in-person event in the same city.
    await createEventWithShift(adminToken, {
      eventName: pastEventName,
      venueId,
      jobTypeId,
      startDateTime: "2020-03-10 09:00:00",
      endDateTime: "2020-03-10 13:00:00",
      maxVolunteers: 2,
    });
  });

  test("page loads with UPCOMING as the default timeframe", async ({
    volunteerPage,
  }) => {
    await volunteerPage.goto("/events");
    await expect(
      volunteerPage.getByRole("heading", { name: /volunteer events/i })
    ).toBeVisible({ timeout: 8_000 });
    await expect(volunteerPage.locator("#timeFrameFilter")).toHaveValue("UPCOMING");
  });

  test("event count appears in the heading after load", async ({
    volunteerPage,
  }) => {
    await volunteerPage.goto("/events");
    // The heading renders "(N)" once events are loaded.
    await expect(
      volunteerPage.locator("h1").filter({ hasText: /\(\d+\)/ })
    ).toBeVisible({ timeout: 8_000 });
  });

  test("city filter — selecting a city shows only that city's events", async ({
    volunteerPage,
  }) => {
    await volunteerPage.goto("/events");
    await expect(
      volunteerPage.getByRole("heading", { name: /volunteer events/i })
    ).toBeVisible({ timeout: 8_000 });

    // Switch to ALL so both our seeded events are in scope.
    await volunteerPage.locator("#timeFrameFilter").selectOption("ALL");

    // Open the Cities panel and select our unique city.
    await volunteerPage.getByRole("button", { name: "All Cities" }).click();
    await volunteerPage.getByLabel(filterCity).check();

    // Both events for that city should appear.
    await expect(volunteerPage.getByText(upcomingEventName)).toBeVisible({
      timeout: 5_000,
    });
    await expect(volunteerPage.getByText(pastEventName)).toBeVisible({
      timeout: 5_000,
    });
  });

  test("timeframe filter — UPCOMING hides past events", async ({
    volunteerPage,
  }) => {
    await volunteerPage.goto("/events");
    await expect(
      volunteerPage.getByRole("heading", { name: /volunteer events/i })
    ).toBeVisible({ timeout: 8_000 });

    // Filter to our unique city (UPCOMING is already the default).
    await volunteerPage.getByRole("button", { name: "All Cities" }).click();
    await volunteerPage.getByLabel(filterCity).check();

    await expect(volunteerPage.getByText(upcomingEventName)).toBeVisible({
      timeout: 5_000,
    });
    await expect(volunteerPage.getByText(pastEventName)).not.toBeVisible();
  });

  test("timeframe filter — PAST shows past events and hides upcoming", async ({
    volunteerPage,
  }) => {
    await volunteerPage.goto("/events");
    await expect(
      volunteerPage.getByRole("heading", { name: /volunteer events/i })
    ).toBeVisible({ timeout: 8_000 });

    // Select our city first, then switch to PAST.
    await volunteerPage.getByRole("button", { name: "All Cities" }).click();
    await volunteerPage.getByLabel(filterCity).check();
    await volunteerPage.locator("#timeFrameFilter").selectOption("PAST");

    await expect(volunteerPage.getByText(pastEventName)).toBeVisible({
      timeout: 5_000,
    });
    await expect(volunteerPage.getByText(upcomingEventName)).not.toBeVisible();
  });

  test("timeframe filter — ALL shows both past and upcoming events", async ({
    volunteerPage,
  }) => {
    await volunteerPage.goto("/events");
    await expect(
      volunteerPage.getByRole("heading", { name: /volunteer events/i })
    ).toBeVisible({ timeout: 8_000 });

    // Select our city + ALL.
    await volunteerPage.getByRole("button", { name: "All Cities" }).click();
    await volunteerPage.getByLabel(filterCity).check();
    await volunteerPage.locator("#timeFrameFilter").selectOption("ALL");

    await expect(volunteerPage.getByText(upcomingEventName)).toBeVisible({
      timeout: 5_000,
    });
    await expect(volunteerPage.getByText(pastEventName)).toBeVisible({
      timeout: 5_000,
    });
  });
});

/* ------------------------------------------------------------------ */
/*  Events page — format filter                                        */
/* ------------------------------------------------------------------ */

test.describe("Events page — format filter", () => {
  let formatCity: string;
  let inPersonEventName: string;

  test.beforeAll(async ({ adminToken }) => {
    formatCity       = uniqueName("FormatCity");
    inPersonEventName = uniqueName("InPersonEvent");

    const jobTypeId = await createJobType(
      adminToken,
      uniqueName("fmt"),
      uniqueName("Format Role"),
    );
    const venueId = await createVenue(adminToken, {
      name: uniqueName("FormatVenue"),
      city: formatCity,
      state: "OR",
    });

    await createEventWithShift(adminToken, {
      eventName: inPersonEventName,
      venueId,
      jobTypeId,
      startDateTime: "2027-09-15 10:00:00",
      endDateTime: "2027-09-15 14:00:00",
    });
  });

  test("IN_PERSON filter shows in-person events", async ({ volunteerPage }) => {
    await volunteerPage.goto("/events");
    await expect(
      volunteerPage.getByRole("heading", { name: /volunteer events/i })
    ).toBeVisible({ timeout: 8_000 });

    // Select our city and IN_PERSON format.
    await volunteerPage.getByRole("button", { name: "All Cities" }).click();
    await volunteerPage.getByLabel(formatCity).check();
    await volunteerPage.locator("#formatFilter").selectOption("IN_PERSON");

    await expect(volunteerPage.getByText(inPersonEventName)).toBeVisible({
      timeout: 5_000,
    });
  });

  test("VIRTUAL filter hides in-person events", async ({ volunteerPage }) => {
    await volunteerPage.goto("/events");
    await expect(
      volunteerPage.getByRole("heading", { name: /volunteer events/i })
    ).toBeVisible({ timeout: 8_000 });

    // Select our city and VIRTUAL format — our in-person event should not show.
    await volunteerPage.getByRole("button", { name: "All Cities" }).click();
    await volunteerPage.getByLabel(formatCity).check();
    await volunteerPage.locator("#formatFilter").selectOption("VIRTUAL");

    await expect(volunteerPage.getByText(inPersonEventName)).not.toBeVisible();
  });
});

/* ------------------------------------------------------------------ */
/*  Events page — card display                                         */
/* ------------------------------------------------------------------ */

test.describe("Events page — card display", () => {
  let cardCity: string;
  let cardEventName: string;

  test.beforeAll(async ({ adminToken }) => {
    cardCity      = uniqueName("CardCity");
    cardEventName = uniqueName("CardEvent");

    const jobTypeId = await createJobType(
      adminToken,
      uniqueName("crd"),
      uniqueName("Card Role"),
    );
    const venueId = await createVenue(adminToken, {
      name: uniqueName("CardVenue"),
      city: cardCity,
      state: "CA",
    });

    await createEventWithShift(adminToken, {
      eventName: cardEventName,
      venueId,
      jobTypeId,
      startDateTime: "2027-10-20 09:00:00",
      endDateTime: "2027-10-20 12:00:00",
      maxVolunteers: 5,
    });
  });

  test("event card shows volunteer count (assigned/max)", async ({
    volunteerPage,
  }) => {
    await volunteerPage.goto("/events");
    await expect(
      volunteerPage.getByRole("heading", { name: /volunteer events/i })
    ).toBeVisible({ timeout: 8_000 });

    // Filter to our city so the card is visible.
    await volunteerPage.getByRole("button", { name: "All Cities" }).click();
    await volunteerPage.getByLabel(cardCity).check();

    // Event name and the 0/5 count should both be visible (nobody signed up yet).
    await expect(volunteerPage.getByText(cardEventName)).toBeVisible({ timeout: 5_000 });
    await expect(volunteerPage.getByText("0/5", { exact: true })).toBeVisible({ timeout: 5_000 });
  });

  test("event card shows the city in the location field", async ({
    volunteerPage,
  }) => {
    await volunteerPage.goto("/events");
    await expect(
      volunteerPage.getByRole("heading", { name: /volunteer events/i })
    ).toBeVisible({ timeout: 8_000 });

    await volunteerPage.getByRole("button", { name: "All Cities" }).click();
    await volunteerPage.getByLabel(cardCity).check();

    // Both the event name and the city should be visible in the main content area.
    // Scope to <main> to avoid matching the Cities button or open checkbox panel.
    const main = volunteerPage.locator("main");
    await expect(main.getByText(cardEventName)).toBeVisible({ timeout: 5_000 });
    await expect(main.getByText(new RegExp(cardCity))).toBeVisible({ timeout: 5_000 });
  });
});
