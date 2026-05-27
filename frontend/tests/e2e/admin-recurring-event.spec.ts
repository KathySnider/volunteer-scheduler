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
 *  - Event detail page: date edit/delete buttons are hidden for recurring events
 *  - Event detail page: "+ Add Date" button is hidden for recurring events
 *  - Event detail page: note explains dates are managed by the recurrence schedule
 *  - Event detail page: edit form shows "Apply changes to" scope selector for recurring events
 *  - Event detail page: Delete button for recurring event shows inline scope picker (not window.confirm)
 *  - Event detail page: "Cancel" on inline delete dismisses the picker without deleting
 *  - Manage Events list page: Delete on a recurring event shows scope modal (not window.confirm)
 *  - Manage Events list page: modal defaults to "Just this occurrence"
 *  - Manage Events list page: "Cancel" dismisses modal without deleting
 *  - Manage Events list page: "This and all future" option is selectable
 *  - Manage Events list page: confirming "Just this occurrence" removes one row and shows success
 *  - Manage Events list page: Delete on a non-recurring event uses window.confirm (no modal)
 *  - Event detail page: editing with THIS_ONLY scope changes only that occurrence's description
 *  - Event detail page: editing with THIS_AND_FUTURE scope changes that and future occurrences, not past
 */

import { test, expect, withAdminRetry } from "./helpers/fixtures";
import type { Page } from "@playwright/test";
import {
  createRecurringEvent,
  createEventWithoutShifts,
  createVenue,
  deleteEventsByName,
  findAllEventsByName,
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

  test("admin can create a recurring event with multiple dates per occurrence", async ({
    adminPage,
  }) => {
    const multiDateName = uniqueName("MultiDateRecurEvent");

    await adminPage.goto("/admin/events/new");
    await expect(
      adminPage.getByPlaceholder(/medicare|workshop/i)
    ).toBeVisible({ timeout: 10_000 });

    // Fill event name.
    await adminPage.getByPlaceholder(/medicare|workshop/i).fill(multiDateName);

    // Select Virtual format.
    await adminPage.getByRole("radio", { name: /virtual/i }).check();

    // Enable recurrence. Set occurrences to 2.
    await adminPage.getByLabel(/repeat this event/i).check();
    await adminPage.getByRole("spinbutton").fill("2");

    // Fill first occurrence date (row 1).
    const dateInputs = adminPage.locator('input[type="date"]');
    const timeInputs = adminPage.locator('input[placeholder="h:MM"]');
    await dateInputs.nth(0).fill("2031-07-07");
    await timeInputs.nth(0).fill("9:00");
    await timeInputs.nth(0).press("Tab");
    await dateInputs.nth(1).fill("2031-07-07");
    await timeInputs.nth(1).fill("11:00");
    await timeInputs.nth(1).press("Tab");

    // Add a second date row.
    await adminPage.getByRole("button", { name: /add another date/i }).click();
    await expect(dateInputs.nth(2)).toBeVisible({ timeout: 3_000 });

    // Fill second date (row 2).
    await dateInputs.nth(2).fill("2031-07-09");
    await timeInputs.nth(2).fill("9:00");
    await timeInputs.nth(2).press("Tab");
    await dateInputs.nth(3).fill("2031-07-09");
    await timeInputs.nth(3).fill("11:00");
    await timeInputs.nth(3).press("Tab");

    // Submit.
    await adminPage.getByRole("button", { name: /create event/i }).click();

    // Success card must appear.
    await expect(
      adminPage.getByText("Event Created!")
    ).toBeVisible({ timeout: 10_000 });

    // Navigate to the first occurrence's detail page to verify both dates persisted.
    // "Add Volunteer Opportunities" is an <a> tag (role=link) styled as a button.
    const detailLink = adminPage.getByRole("link", { name: /add volunteer opportunities/i });
    await expect(detailLink).toBeVisible({ timeout: 10_000 });
    await detailLink.click();
    await expect(adminPage.getByText(multiDateName)).toBeVisible({ timeout: 10_000 });

    // Both seed dates must appear on the detail page. Each date renders as two
    // elements (start line + "to …" end line), so use .first() to avoid strict-mode errors.
    await expect(adminPage.getByText(/Jul 7, 2031/i).first()).toBeVisible();
    await expect(adminPage.getByText(/Jul 9, 2031/i).first()).toBeVisible();
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
  let eventName: string;
  // ID of the first instance (occurrence 1) — returned directly by createRecurringEvent.
  let firstEventId: string;

  test.beforeAll(async ({ adminToken }) => {
    eventName = uniqueName("DetailRecurEvent");

    // Create a 3-occurrence weekly series starting in the future.
    // createRecurringEvent now returns the event ID of occurrence #1 directly.
    firstEventId = await withAdminRetry(adminToken, (token) =>
      createRecurringEvent(token, {
        eventName,
        occurrences: 3,
        pattern: "WEEKLY",
        startDateTime: "2031-08-06 09:00:00",
        endDateTime:   "2031-08-06 11:00:00",
      })
    );
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

  test("date edit and delete buttons are absent for recurring events", async ({
    adminPage,
  }) => {
    test.skip(!firstEventId, "beforeAll did not seed event");

    await adminPage.goto(`/admin/events/${firstEventId}`);
    await expect(adminPage.getByText(eventName)).toBeVisible({ timeout: 10_000 });

    // The ✏ edit-date and 🗑 remove-date icon buttons must not appear.
    await expect(adminPage.getByTitle("Edit date")).not.toBeVisible();
    await expect(adminPage.getByTitle("Remove date")).not.toBeVisible();
  });

  test("'+ Add Date' button is absent for recurring events", async ({
    adminPage,
  }) => {
    test.skip(!firstEventId, "beforeAll did not seed event");

    await adminPage.goto(`/admin/events/${firstEventId}`);
    await expect(adminPage.getByText(eventName)).toBeVisible({ timeout: 10_000 });

    await expect(
      adminPage.getByRole("button", { name: /add date/i })
    ).not.toBeVisible();
  });

  test("recurring event shows note that dates are managed by the recurrence schedule", async ({
    adminPage,
  }) => {
    test.skip(!firstEventId, "beforeAll did not seed event");

    await adminPage.goto(`/admin/events/${firstEventId}`);
    await expect(adminPage.getByText(eventName)).toBeVisible({ timeout: 10_000 });

    await expect(
      adminPage.getByText(/dates are managed by the recurrence schedule/i)
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
    deleteFirstEventId = await withAdminRetry(adminToken, (token) =>
      createRecurringEvent(token, {
        eventName: deleteEventName,
        occurrences: 3,
        pattern: "WEEKLY",
        startDateTime: "2031-09-03 10:00:00",
        endDateTime:   "2031-09-03 12:00:00",
      })
    );
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

/* ------------------------------------------------------------------ */
/*  Manage Events list page — recurring event delete scope modal        */
/* ------------------------------------------------------------------ */

test.describe("Manage Events list — recurring event delete scope modal", () => {
  let listDeleteName: string;

  test.beforeAll(async ({ adminToken }) => {
    listDeleteName = uniqueName("ListDeleteRecurEvent");
    await withAdminRetry(adminToken, (token) =>
      createRecurringEvent(token, {
        eventName: listDeleteName,
        occurrences: 3,
        pattern: "WEEKLY",
        startDateTime: "2031-10-01 09:00:00",
        endDateTime:   "2031-10-01 11:00:00",
      })
    );
  });

  test.afterAll(async ({ adminToken }) => {
    if (listDeleteName) {
      try { await deleteEventsByName(adminToken, listDeleteName); } catch { /* already gone */ }
    }
  });

  /** Navigate to the list and wait for the recurring event to appear. */
  async function gotoList(adminPage: Page) {
    await adminPage.goto("/admin/events");
    await expect(
      adminPage.getByRole("heading", { name: /manage events/i })
    ).toBeVisible({ timeout: 10_000 });
    await expect(adminPage.getByText(listDeleteName).first()).toBeVisible({ timeout: 8_000 });
  }

  test("clicking Delete on a recurring event shows the scope modal", async ({
    adminPage,
  }) => {
    await gotoList(adminPage);

    // Click the delete icon on the first row that matches our event name.
    const row = adminPage.getByRole("row").filter({ hasText: listDeleteName }).first();
    await row.getByTitle(/delete event/i).click();

    // The modal overlay must appear — no native browser dialog.
    await expect(
      adminPage.getByText(/this is part of a recurring series/i)
    ).toBeVisible({ timeout: 5_000 });
    await expect(
      adminPage.getByRole("radio", { name: /just this occurrence/i })
    ).toBeVisible();
    await expect(
      adminPage.getByRole("radio", { name: /this and all future/i })
    ).toBeVisible();
  });

  test("scope modal defaults to 'Just this occurrence'", async ({
    adminPage,
  }) => {
    await gotoList(adminPage);

    const row = adminPage.getByRole("row").filter({ hasText: listDeleteName }).first();
    await row.getByTitle(/delete event/i).click();

    await expect(
      adminPage.getByRole("radio", { name: /just this occurrence/i })
    ).toBeChecked({ timeout: 5_000 });
    await expect(
      adminPage.getByRole("radio", { name: /this and all future/i })
    ).not.toBeChecked();
  });

  test("Cancel dismisses the modal without deleting the event", async ({
    adminPage,
  }) => {
    await gotoList(adminPage);

    const row = adminPage.getByRole("row").filter({ hasText: listDeleteName }).first();
    await row.getByTitle(/delete event/i).click();

    await expect(
      adminPage.getByText(/this is part of a recurring series/i)
    ).toBeVisible({ timeout: 5_000 });

    await adminPage.getByRole("button", { name: /^cancel$/i }).click();

    // Modal gone, event row still present.
    await expect(
      adminPage.getByText(/this is part of a recurring series/i)
    ).not.toBeVisible();
    await expect(adminPage.getByText(listDeleteName).first()).toBeVisible();
  });

  test("'This and all future' option can be selected in the modal", async ({
    adminPage,
  }) => {
    await gotoList(adminPage);

    const row = adminPage.getByRole("row").filter({ hasText: listDeleteName }).first();
    await row.getByTitle(/delete event/i).click();

    const futureRadio = adminPage.getByRole("radio", { name: /this and all future/i });
    await futureRadio.check();
    await expect(futureRadio).toBeChecked({ timeout: 3_000 });
  });

  test("confirming 'Just this occurrence' removes one row and shows success banner", async ({
    adminPage,
  }) => {
    await gotoList(adminPage);

    // Count rows before delete.
    const rowsBefore = await adminPage
      .getByRole("row")
      .filter({ hasText: listDeleteName })
      .count();

    const row = adminPage.getByRole("row").filter({ hasText: listDeleteName }).first();
    await row.getByTitle(/delete event/i).click();

    // Default is THIS_ONLY — just confirm.
    await adminPage.getByRole("button", { name: /^delete$/i }).click();

    // Success banner should appear.
    await expect(adminPage.getByText(/event deleted/i)).toBeVisible({ timeout: 8_000 });

    // One fewer row with this event name.
    const rowsAfter = await adminPage
      .getByRole("row")
      .filter({ hasText: listDeleteName })
      .count();
    expect(rowsAfter).toBe(rowsBefore - 1);
  });
});

/* ------------------------------------------------------------------ */
/*  Manage Events list page — non-recurring event uses window.confirm   */
/* ------------------------------------------------------------------ */

test.describe("Manage Events list — non-recurring event delete uses confirm dialog", () => {
  let standaloneEventName: string;
  let standaloneVenueId: string;

  test.beforeAll(async ({ adminToken }) => {
    standaloneEventName = uniqueName("StandaloneDeleteEvent");
    await withAdminRetry(adminToken, async (token) => {
      standaloneVenueId = await createVenue(token, {
        name: uniqueName("DeleteTestVenue"),
        city: "Seattle",
        state: "WA",
      });
      await createEventWithoutShifts(token, {
        eventName: standaloneEventName,
        venueId: standaloneVenueId,
        startDateTime: "2031-11-05 09:00:00",
        endDateTime:   "2031-11-05 11:00:00",
      });
    });
  });

  test.afterAll(async ({ adminToken }) => {
    if (standaloneEventName) {
      try { await deleteEventsByName(adminToken, standaloneEventName); } catch { /* already gone */ }
    }
  });

  test("clicking Delete on a non-recurring event shows a browser confirm (no modal)", async ({
    adminPage,
  }) => {
    await adminPage.goto("/admin/events");
    await expect(
      adminPage.getByRole("heading", { name: /manage events/i })
    ).toBeVisible({ timeout: 10_000 });
    await expect(adminPage.getByText(standaloneEventName)).toBeVisible({ timeout: 8_000 });

    // Intercept the native confirm dialog — accepting it lets the delete proceed.
    let dialogSeen = false;
    adminPage.once("dialog", (dialog) => {
      dialogSeen = true;
      dialog.accept();
    });

    const row = adminPage.getByRole("row").filter({ hasText: standaloneEventName }).first();
    await row.getByTitle(/delete event/i).click();

    // Give the page a moment to fire the dialog.
    await adminPage.waitForTimeout(1_000);

    // The scope modal must NOT appear (only the browser dialog should have shown).
    await expect(
      adminPage.getByText(/this is part of a recurring series/i)
    ).not.toBeVisible();

    expect(dialogSeen).toBe(true);
  });
});

/* ------------------------------------------------------------------ */
/*  Event detail page — update scope propagation                       */
/* ------------------------------------------------------------------ */

test.describe("Event detail page — update scope propagation", () => {
  // Two separate 3-occurrence series: one per propagation test so mutations
  // don't interfere with each other.
  let thisOnlyName: string;
  let thisAndFutureName: string;
  let thisOnlyOccs: Array<{ id: string; recurrenceOrder: number }>;
  let thisAndFutureOccs: Array<{ id: string; recurrenceOrder: number }>;

  test.beforeAll(async ({ adminToken }) => {
    thisOnlyName      = uniqueName("UpdateThisOnlyEvent");
    thisAndFutureName = uniqueName("UpdateFutureEvent");

    await withAdminRetry(adminToken, async (token) => {
      await createRecurringEvent(token, {
        eventName:     thisOnlyName,
        occurrences:   3,
        pattern:       "WEEKLY",
        startDateTime: "2031-12-03 09:00:00",
        endDateTime:   "2031-12-03 11:00:00",
      });
      await createRecurringEvent(token, {
        eventName:     thisAndFutureName,
        occurrences:   3,
        pattern:       "WEEKLY",
        startDateTime: "2032-01-07 09:00:00",
        endDateTime:   "2032-01-07 11:00:00",
      });

      thisOnlyOccs      = await findAllEventsByName(token, thisOnlyName);
      thisAndFutureOccs = await findAllEventsByName(token, thisAndFutureName);
    });
  });

  test.afterAll(async ({ adminToken }) => {
    for (const name of [thisOnlyName, thisAndFutureName]) {
      if (name) {
        try { await deleteEventsByName(adminToken, name); } catch { /* ignore */ }
      }
    }
  });

  test("editing with THIS_ONLY scope changes only that occurrence's description", async ({
    adminPage,
  }) => {
    test.skip(thisOnlyOccs.length < 3, "beforeAll did not seed all 3 occurrences");

    // Target: occurrence #2 (middle). occurrences are sorted by recurrenceOrder.
    const occ2 = thisOnlyOccs[1];
    const newDesc = uniqueName("DescThisOnly");

    await adminPage.goto(`/admin/events/${occ2.id}`);
    await expect(adminPage.getByText(thisOnlyName)).toBeVisible({ timeout: 10_000 });

    // Open the edit form.
    await adminPage.getByRole("button", { name: /edit/i }).first().click();
    await expect(adminPage.getByText(/apply changes to/i)).toBeVisible({ timeout: 5_000 });

    // Change the description (the only textarea in the event edit section).
    await adminPage.locator("textarea").first().fill(newDesc);

    // Scope is THIS_ONLY by default — save immediately.
    await adminPage.getByRole("button", { name: /save changes/i }).click();
    await expect(adminPage.getByText("Event updated.")).toBeVisible({ timeout: 15_000 });

    // Occurrence #2 must show the new description.
    await expect(adminPage.getByText(newDesc)).toBeVisible();

    // Occurrence #1 (before) must NOT have been changed.
    await adminPage.goto(`/admin/events/${thisOnlyOccs[0].id}`);
    await expect(adminPage.getByText(thisOnlyName)).toBeVisible({ timeout: 10_000 });
    await expect(adminPage.getByText(newDesc)).not.toBeVisible();

    // Occurrence #3 (after) must NOT have been changed.
    await adminPage.goto(`/admin/events/${thisOnlyOccs[2].id}`);
    await expect(adminPage.getByText(thisOnlyName)).toBeVisible({ timeout: 10_000 });
    await expect(adminPage.getByText(newDesc)).not.toBeVisible();
  });

  test("editing with THIS_AND_FUTURE scope updates that and future occurrences, not past", async ({
    adminPage,
  }) => {
    test.skip(thisAndFutureOccs.length < 3, "beforeAll did not seed all 3 occurrences");

    // Target: occurrence #2 (middle).
    const occ2 = thisAndFutureOccs[1];
    const newDesc = uniqueName("DescFuture");

    await adminPage.goto(`/admin/events/${occ2.id}`);
    await expect(adminPage.getByText(thisAndFutureName)).toBeVisible({ timeout: 10_000 });

    await adminPage.getByRole("button", { name: /edit/i }).first().click();
    await expect(adminPage.getByText(/apply changes to/i)).toBeVisible({ timeout: 5_000 });

    await adminPage.locator("textarea").first().fill(newDesc);

    // Switch scope to THIS_AND_FUTURE.
    await adminPage.getByRole("radio", { name: /this and all future/i }).check();
    await expect(
      adminPage.getByRole("radio", { name: /this and all future/i })
    ).toBeChecked();

    await adminPage.getByRole("button", { name: /save changes/i }).click();
    await expect(adminPage.getByText("Event updated.")).toBeVisible({ timeout: 15_000 });

    // Occurrence #2 must show the new description.
    await expect(adminPage.getByText(newDesc)).toBeVisible();

    // Occurrence #3 (future) must also show the new description.
    await adminPage.goto(`/admin/events/${thisAndFutureOccs[2].id}`);
    await expect(adminPage.getByText(thisAndFutureName)).toBeVisible({ timeout: 10_000 });
    await expect(adminPage.getByText(newDesc)).toBeVisible({ timeout: 5_000 });

    // Occurrence #1 (past — not reached by THIS_AND_FUTURE from occ #2) must be unchanged.
    await adminPage.goto(`/admin/events/${thisAndFutureOccs[0].id}`);
    await expect(adminPage.getByText(thisAndFutureName)).toBeVisible({ timeout: 10_000 });
    await expect(adminPage.getByText(newDesc)).not.toBeVisible();
  });
});
