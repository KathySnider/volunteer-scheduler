/**
 * E2E tests — Admin recurring event UI
 *
 * Covers:
 *  - New event form: "Repeat this event" toggle hides/shows recurrence fields
 *  - New event form: recurrence fields are greyed out (disabled) when toggle is off
 *  - New event form: changing pattern updates the default occurrences value
 *  - New event form: "Week of Month" selector appears only for Monthly pattern
 *  - New event form: section title changes to "First Occurrence Dates" when recurring
 *  - New event form: creating a recurring event succeeds and shows "Event Created!"
 *  - Event detail page: recurrence info block appears for a recurring event
 *  - Event detail page: edit form shows "Apply changes to" scope selector for recurring events
 *  - Event detail page: Delete button for recurring event shows inline scope picker (not window.confirm)
 *  - Event detail page: "Cancel" on inline delete dismisses the picker without deleting
 */

import { test, expect } from "./helpers/fixtures";
import {
  createRecurringEvent,
  deleteEventsByName,
  findEventIdByName,
  uniqueName,
} from "./helpers/api";

/* ------------------------------------------------------------------ */
/*  New event form — recurrence section UI                              */
/* ------------------------------------------------------------------ */

test.describe("New event form — recurrence section", () => {
  test.beforeEach(async ({ adminPage }) => {
    await adminPage.goto("/admin/events/new");
    // Wait for the form to be fully hydrated.
    await expect(
      adminPage.getByPlaceholder(/medicare|workshop/i)
    ).toBeVisible({ timeout: 10_000 });
  });

  test("recurrence fields are disabled (greyed out) when toggle is off", async ({
    adminPage,
  }) => {
    // Toggle is unchecked by default — the fields block carries recurDisabled class.
    const patternSelect = adminPage.getByRole("combobox", { name: /pattern/i });
    const occurrencesInput = adminPage.getByRole("spinbutton");

    await expect(patternSelect).toBeDisabled();
    await expect(occurrencesInput).toBeDisabled();
  });

  test("checking 'Repeat this event' enables the recurrence fields", async ({
    adminPage,
  }) => {
    await adminPage.getByLabel(/repeat this event/i).check();

    const patternSelect = adminPage.getByRole("combobox", { name: /pattern/i });
    const occurrencesInput = adminPage.getByRole("spinbutton");

    await expect(patternSelect).toBeEnabled();
    await expect(occurrencesInput).toBeEnabled();
  });

  test("pattern defaults to Weekly with 52 occurrences", async ({
    adminPage,
  }) => {
    await adminPage.getByLabel(/repeat this event/i).check();

    const patternSelect = adminPage.getByRole("combobox", { name: /pattern/i });
    const occurrencesInput = adminPage.getByRole("spinbutton");

    await expect(patternSelect).toHaveValue("WEEKLY");
    await expect(occurrencesInput).toHaveValue("52");
  });

  test("changing pattern to Daily sets 365 occurrences", async ({
    adminPage,
  }) => {
    await adminPage.getByLabel(/repeat this event/i).check();

    const patternSelect = adminPage.getByRole("combobox", { name: /pattern/i });
    await patternSelect.selectOption("DAILY");

    await expect(adminPage.getByRole("spinbutton")).toHaveValue("365");
  });

  test("changing pattern to Every 2 Weeks sets 26 occurrences", async ({
    adminPage,
  }) => {
    await adminPage.getByLabel(/repeat this event/i).check();

    const patternSelect = adminPage.getByRole("combobox", { name: /pattern/i });
    await patternSelect.selectOption("BIWEEKLY");

    await expect(adminPage.getByRole("spinbutton")).toHaveValue("26");
  });

  test("changing pattern to Monthly sets 12 occurrences", async ({
    adminPage,
  }) => {
    await adminPage.getByLabel(/repeat this event/i).check();

    const patternSelect = adminPage.getByRole("combobox", { name: /pattern/i });
    await patternSelect.selectOption("MONTHLY");

    await expect(adminPage.getByRole("spinbutton")).toHaveValue("12");
  });

  test("changing pattern to Yearly clears occurrences (required field)", async ({
    adminPage,
  }) => {
    await adminPage.getByLabel(/repeat this event/i).check();

    const patternSelect = adminPage.getByRole("combobox", { name: /pattern/i });
    await patternSelect.selectOption("YEARLY");

    await expect(adminPage.getByRole("spinbutton")).toHaveValue("");
  });

  test("'Week of Month' selector appears only for Monthly pattern", async ({
    adminPage,
  }) => {
    await adminPage.getByLabel(/repeat this event/i).check();

    const patternSelect = adminPage.getByRole("combobox", { name: /pattern/i });

    // Not present for Weekly (default).
    await expect(adminPage.getByText(/week of month/i)).not.toBeVisible();

    // Appears for Monthly.
    await patternSelect.selectOption("MONTHLY");
    await expect(adminPage.getByText(/week of month/i)).toBeVisible();

    // Disappears when switching back to Weekly.
    await patternSelect.selectOption("WEEKLY");
    await expect(adminPage.getByText(/week of month/i)).not.toBeVisible();
  });

  test("section title changes to 'First Occurrence Dates' when recurring", async ({
    adminPage,
  }) => {
    // Before enabling: title should be "Event Dates".
    await expect(adminPage.getByText("Event Dates")).toBeVisible();
    await expect(adminPage.getByText("First Occurrence Dates")).not.toBeVisible();

    await adminPage.getByLabel(/repeat this event/i).check();

    await expect(adminPage.getByText("First Occurrence Dates")).toBeVisible();
    await expect(adminPage.getByText("Event Dates")).not.toBeVisible();
  });

  test("a note appears below the title explaining first occurrence behaviour", async ({
    adminPage,
  }) => {
    await adminPage.getByLabel(/repeat this event/i).check();
    await expect(
      adminPage.getByText(/enter the dates for the first occurrence/i)
    ).toBeVisible();
  });

  test("unchecking the toggle restores 'Event Dates' title and hides the note", async ({
    adminPage,
  }) => {
    await adminPage.getByLabel(/repeat this event/i).check();
    await adminPage.getByLabel(/repeat this event/i).uncheck();

    await expect(adminPage.getByText("Event Dates")).toBeVisible();
    await expect(adminPage.getByText("First Occurrence Dates")).not.toBeVisible();
    await expect(
      adminPage.getByText(/enter the dates for the first occurrence/i)
    ).not.toBeVisible();
  });

  test("YEARLY pattern with no occurrences shows a validation error", async ({
    adminPage,
  }) => {
    await adminPage.getByLabel(/repeat this event/i).check();

    const patternSelect = adminPage.getByRole("combobox", { name: /pattern/i });
    await patternSelect.selectOption("YEARLY");

    // Fill minimum required fields so only the recurrenceMax error fires.
    await adminPage.getByPlaceholder(/medicare|workshop/i).fill("Yearly Test Event");
    // Submit — validation should block the request.
    await adminPage.getByRole("button", { name: /create event/i }).click();

    await expect(
      adminPage.getByText(/number of occurrences is required/i)
    ).toBeVisible({ timeout: 5_000 });
  });
});

/* ------------------------------------------------------------------ */
/*  Creating a recurring event through the UI                           */
/* ------------------------------------------------------------------ */

test.describe("New event form — create recurring event happy path", () => {
  let createdEventName: string;

  test.afterAll(async ({ adminToken }) => {
    if (createdEventName) {
      try {
        await deleteEventsByName(adminToken, createdEventName);
      } catch { /* ignore */ }
    }
  });

  test("admin can create a weekly virtual recurring event", async ({
    adminPage,
    adminToken,
  }) => {
    createdEventName = uniqueName("WeeklyRecurEvent");

    await adminPage.goto("/admin/events/new");
    await expect(
      adminPage.getByPlaceholder(/medicare|workshop/i)
    ).toBeVisible({ timeout: 10_000 });

    // Fill event name.
    await adminPage.getByPlaceholder(/medicare|workshop/i).fill(createdEventName);

    // Select Virtual format (no venue required).
    await adminPage.getByRole("radio", { name: /virtual/i }).check();

    // Enable recurrence.
    await adminPage.getByLabel(/repeat this event/i).check();

    // Pattern: Weekly (default). Set occurrences to 3.
    const occurrencesInput = adminPage.getByRole("spinbutton");
    await occurrencesInput.fill("3");

    // Fill in the first occurrence dates.
    const dateInputs = adminPage.locator('input[type="date"]');
    await dateInputs.first().fill("2031-07-02");

    const timeInputs = adminPage.locator('input[placeholder="h:MM"]');
    await timeInputs.nth(0).fill("9:00");
    await timeInputs.nth(0).press("Tab");

    await dateInputs.nth(1).fill("2031-07-02");
    await timeInputs.nth(1).fill("11:00");
    await timeInputs.nth(1).press("Tab");

    // Submit.
    await adminPage.getByRole("button", { name: /create event/i }).click();

    // Success card should appear.
    await expect(
      adminPage.getByText("Event Created!")
    ).toBeVisible({ timeout: 10_000 });
  });

  test("created recurring event appears in the Manage Events list", async ({
    adminPage,
    adminToken,
  }) => {
    // createdEventName must have been set by the previous test.
    test.skip(!createdEventName, "depends on previous test");

    await adminPage.goto("/admin/events");
    await expect(
      adminPage.getByRole("heading", { name: /manage events/i })
    ).toBeVisible({ timeout: 8_000 });

    // At least one instance of the recurring event should be listed.
    await expect(adminPage.getByText(createdEventName).first()).toBeVisible({
      timeout: 5_000,
    });
  });
});

/* ------------------------------------------------------------------ */
/*  Event detail page — recurring event display                         */
/* ------------------------------------------------------------------ */

test.describe("Event detail page — recurring event info", () => {
  let groupId: string;
  let eventName: string;
  // ID of the first instance (occurrence 1) — obtained from the DB via API.
  let firstEventId: string;

  test.beforeAll(async ({ adminToken }) => {
    eventName = uniqueName("DetailRecurEvent");

    // Create a 3-occurrence weekly series starting in the future.
    groupId = await createRecurringEvent(adminToken, {
      eventName,
      occurrences: 3,
      pattern: "WEEKLY",
      startDateTime: "2031-08-06 09:00:00",
      endDateTime:   "2031-08-06 11:00:00",
    });

    // The createRecurringEvent helper returns the group UUID; to navigate to
    // the detail page we need an individual event_id.  Query filteredEvents
    // and find the first matching instance by name.
    firstEventId = (await findEventIdByName(adminToken, eventName)) ?? "";
  });

  test.afterAll(async ({ adminToken }) => {
    if (eventName) {
      try { await deleteEventsByName(adminToken, eventName); } catch { /* ignore */ }
    }
  });

  test("recurrence info block shows pattern and occurrence count", async ({
    adminPage,
  }) => {
    test.skip(!firstEventId, "beforeAll did not seed event");

    await adminPage.goto(`/admin/events/${firstEventId}`);
    // Wait for the event detail to load.
    await expect(adminPage.getByText(eventName)).toBeVisible({ timeout: 10_000 });

    // The recurrence meta row should show "Weekly · occurrence N of 3".
    await expect(
      adminPage.getByText(/weekly.*occurrence.*of 3/i)
    ).toBeVisible({ timeout: 5_000 });
  });

  test("read-only note about changing recurrence is shown", async ({
    adminPage,
  }) => {
    test.skip(!firstEventId, "beforeAll did not seed event");

    await adminPage.goto(`/admin/events/${firstEventId}`);
    await expect(adminPage.getByText(eventName)).toBeVisible({ timeout: 10_000 });

    await expect(
      adminPage.getByText(/delete this and future occurrences/i)
    ).toBeVisible({ timeout: 5_000 });
  });

  test("edit form shows 'Apply changes to' scope selector for recurring event", async ({
    adminPage,
  }) => {
    test.skip(!firstEventId, "beforeAll did not seed event");

    await adminPage.goto(`/admin/events/${firstEventId}`);
    await expect(adminPage.getByText(eventName)).toBeVisible({ timeout: 10_000 });

    // Click the edit (pencil) icon button next to the event info section.
    await adminPage.getByRole("button", { name: /edit/i }).first().click();

    // "Apply changes to" label and both radio options must be visible.
    await expect(adminPage.getByText(/apply changes to/i)).toBeVisible({
      timeout: 5_000,
    });
    await expect(
      adminPage.getByRole("radio", { name: /just this occurrence/i })
    ).toBeVisible();
    await expect(
      adminPage.getByRole("radio", { name: /this and all future/i })
    ).toBeVisible();
  });

  test("edit form scope defaults to 'Just this occurrence'", async ({
    adminPage,
  }) => {
    test.skip(!firstEventId, "beforeAll did not seed event");

    await adminPage.goto(`/admin/events/${firstEventId}`);
    await expect(adminPage.getByText(eventName)).toBeVisible({ timeout: 10_000 });

    await adminPage.getByRole("button", { name: /edit/i }).first().click();

    await expect(
      adminPage.getByRole("radio", { name: /just this occurrence/i })
    ).toBeChecked({ timeout: 5_000 });
  });
});

/* ------------------------------------------------------------------ */
/*  Event detail page — delete recurring event inline scope picker      */
/* ------------------------------------------------------------------ */

test.describe("Event detail page — recurring event delete UI", () => {
  let deleteEventName: string;
  let deleteFirstEventId: string;

  test.beforeAll(async ({ adminToken }) => {
    deleteEventName = uniqueName("DeleteScopeEvent");
    await createRecurringEvent(adminToken, {
      eventName: deleteEventName,
      occurrences: 3,
      pattern: "WEEKLY",
      startDateTime: "2031-09-03 10:00:00",
      endDateTime:   "2031-09-03 12:00:00",
    });
    deleteFirstEventId = (await findEventIdByName(adminToken, deleteEventName)) ?? "";
  });

  test.afterAll(async ({ adminToken }) => {
    if (deleteEventName) {
      try { await deleteEventsByName(adminToken, deleteEventName); } catch { /* ignore */ }
    }
  });

  test("clicking Delete for a recurring event shows inline scope picker", async ({
    adminPage,
  }) => {
    test.skip(!deleteFirstEventId, "beforeAll did not seed event");

    await adminPage.goto(`/admin/events/${deleteFirstEventId}`);
    await expect(adminPage.getByText(deleteEventName)).toBeVisible({ timeout: 10_000 });

    // Click the Delete button (danger button in the event info header area).
    await adminPage.getByRole("button", { name: /delete event/i }).click();

    // Inline picker should appear — no native dialog.
    await expect(
      adminPage.getByText(/delete recurring event/i)
    ).toBeVisible({ timeout: 5_000 });
    await expect(
      adminPage.getByRole("radio", { name: /just this occurrence/i })
    ).toBeVisible();
    await expect(
      adminPage.getByRole("radio", { name: /this and all future/i })
    ).toBeVisible();
  });

  test("Cancel on inline delete picker dismisses it without deleting", async ({
    adminPage,
  }) => {
    test.skip(!deleteFirstEventId, "beforeAll did not seed event");

    await adminPage.goto(`/admin/events/${deleteFirstEventId}`);
    await expect(adminPage.getByText(deleteEventName)).toBeVisible({ timeout: 10_000 });

    await adminPage.getByRole("button", { name: /delete event/i }).click();

    // Inline picker is visible.
    await expect(
      adminPage.getByText(/delete recurring event/i)
    ).toBeVisible({ timeout: 5_000 });

    // Click Cancel — picker should disappear, event still on screen.
    await adminPage.getByRole("button", { name: /^cancel$/i }).click();

    await expect(adminPage.getByText(/delete recurring event/i)).not.toBeVisible();
    await expect(adminPage.getByText(deleteEventName)).toBeVisible();
  });

  test("scope defaults to 'Just this occurrence' in the delete picker", async ({
    adminPage,
  }) => {
    test.skip(!deleteFirstEventId, "beforeAll did not seed event");

    await adminPage.goto(`/admin/events/${deleteFirstEventId}`);
    await expect(adminPage.getByText(deleteEventName)).toBeVisible({ timeout: 10_000 });

    await adminPage.getByRole("button", { name: /delete event/i }).click();

    await expect(
      adminPage.getByRole("radio", { name: /just this occurrence/i })
    ).toBeChecked({ timeout: 5_000 });
  });

  test("'This and all future' scope option can be selected", async ({
    adminPage,
  }) => {
    test.skip(!deleteFirstEventId, "beforeAll did not seed event");

    await adminPage.goto(`/admin/events/${deleteFirstEventId}`);
    await expect(adminPage.getByText(deleteEventName)).toBeVisible({ timeout: 10_000 });

    await adminPage.getByRole("button", { name: /delete event/i }).click();

    const futureRadio = adminPage.getByRole("radio", { name: /this and all future/i });
    await futureRadio.check();
    await expect(futureRadio).toBeChecked({ timeout: 3_000 });
  });
});
