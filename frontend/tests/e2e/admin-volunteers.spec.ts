/**
 * E2E tests — Admin Manage Volunteers page
 *
 * Covers:
 *  - Page loads and shows the volunteer list
 *  - Search/filter narrows the list
 *  - Create: "+ New Volunteer" form adds a volunteer that appears in the list
 *  - Edit: inline edit form updates name and role
 *  - Delete: confirmation dialog then volunteer is removed from the list
 *  - Non-admin (volunteer) is redirected to /events
 */

import { test, expect } from "./helpers/fixtures";
import { createVolunteer, uniqueEmail, uniqueName } from "./helpers/api";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/**
 * Fill a form field whose label and input are siblings inside a `.field`
 * container (CSS-module class), as used on the volunteers page.
 * Finds the container that contains the label text, then fills its input.
 */
async function fillField(
  scope: import("@playwright/test").Locator,
  labelText: string | RegExp,
  value: string
) {
  await scope
    .locator("[class*='field']")
    .filter({ hasText: labelText })
    .locator("input")
    .fill(value);
}

// ---------------------------------------------------------------------------

test.describe("Manage Volunteers — page load", () => {
  test("admin can navigate to the volunteers page", async ({ adminPage }) => {
    await adminPage.goto("/admin/volunteers");
    await expect(
      adminPage.getByRole("heading", { name: /manage volunteers/i })
    ).toBeVisible({ timeout: 8_000 });
  });

  test("volunteer list is shown with at least one card", async ({ adminPage }) => {
    await adminPage.goto("/admin/volunteers");
    await expect(
      adminPage.getByRole("heading", { name: /manage volunteers/i })
    ).toBeVisible({ timeout: 8_000 });
    // Wait for at least one volunteer card to render.
    await expect(
      adminPage.locator("[class*='volCard']").first()
    ).toBeVisible({ timeout: 8_000 });
    // At least one role badge should be visible.
    await expect(
      adminPage.locator("[class*='roleBadge']").first()
    ).toBeVisible({ timeout: 8_000 });
  });

  test("non-admin is redirected to /events", async ({ volunteerPage }) => {
    await volunteerPage.goto("/admin/volunteers");
    await volunteerPage.waitForURL("**/events", { timeout: 5_000 });
    expect(volunteerPage.url()).toContain("/events");
  });
});

test.describe("Manage Volunteers — search", () => {
  let searchFirst: string;
  let searchLast: string;

  test.beforeAll(async ({ adminToken }) => {
    // Use highly unique names so previous test runs don't cause strict-mode
    // violations — the timestamp suffix makes collisions essentially impossible.
    searchFirst = uniqueName("Srchable");
    searchLast  = uniqueName("McFindme");
    await createVolunteer(adminToken, {
      firstName: searchFirst,
      lastName:  searchLast,
      email:     uniqueEmail("vol-search"),
      role:      "VOLUNTEER",
    });
  });

  test("search box filters the list by name", async ({ adminPage }) => {
    await adminPage.goto("/admin/volunteers");

    // Wait for list to load — scope to the name element to avoid strict-mode
    // issues if multiple matches exist from previous runs.
    const nameLocator = adminPage
      .locator("[class*='volName']")
      .filter({ hasText: `${searchFirst} ${searchLast}` })
      .first();
    await expect(nameLocator).toBeVisible({ timeout: 8_000 });

    // Narrow by last name (unique across runs thanks to timestamp suffix).
    const searchBox = adminPage.getByPlaceholder(/search/i);
    await searchBox.fill(searchLast);
    await expect(nameLocator).toBeVisible();

    // Nothing-matching term hides the volunteer.
    await searchBox.fill("zzz-nobody-by-this-name");
    await expect(nameLocator).not.toBeVisible();
  });

  test("clearing search restores the full list", async ({ adminPage }) => {
    await adminPage.goto("/admin/volunteers");

    const nameLocator = adminPage
      .locator("[class*='volName']")
      .filter({ hasText: `${searchFirst} ${searchLast}` })
      .first();
    await expect(nameLocator).toBeVisible({ timeout: 8_000 });

    const searchBox = adminPage.getByPlaceholder(/search/i);
    await searchBox.fill("zzz-nobody");
    await expect(nameLocator).not.toBeVisible();

    await searchBox.clear();
    await expect(nameLocator).toBeVisible();
  });
});

test.describe("Manage Volunteers — create", () => {
  test("admin can open the New Volunteer panel", async ({ adminPage }) => {
    await adminPage.goto("/admin/volunteers");
    await adminPage.getByRole("button", { name: /new volunteer/i }).click();
    await expect(
      adminPage.getByRole("heading", { name: /new volunteer/i })
    ).toBeVisible({ timeout: 5_000 });
  });

  test("create form validates required fields", async ({ adminPage }) => {
    await adminPage.goto("/admin/volunteers");
    await adminPage.getByRole("button", { name: /new volunteer/i }).click();

    // Submit without filling anything in.
    await adminPage.getByRole("button", { name: "Create Volunteer" }).click();

    // Error banner should appear.
    await expect(
      adminPage.getByText("First name, last name, and email are required.")
    ).toBeVisible({ timeout: 3_000 });
  });

  test("admin fills in the form and the new volunteer appears in the list", async ({
    adminPage,
  }) => {
    const firstName = uniqueName("NewFirst");
    const lastName  = uniqueName("NewLast");
    const email     = uniqueEmail("create-vol");

    await adminPage.goto("/admin/volunteers");
    await adminPage.getByRole("button", { name: /new volunteer/i }).click();

    // Wait for the panel — use div to avoid matching the h2.createPanelTitle
    // which also contains the substring "createPanel".
    const panel = adminPage.locator("div[class*='createPanel']");
    await expect(panel).toBeVisible({ timeout: 5_000 });

    // The label and input are siblings inside a .field container — use fillField().
    await fillField(panel, /first name/i, firstName);
    await fillField(panel, /last name/i,  lastName);
    await fillField(panel, /^email/i,     email);
    // Role defaults to VOLUNTEER — leave it.

    await adminPage.getByRole("button", { name: "Create Volunteer" }).click();

    // Success banner should appear.
    await expect(
      adminPage.getByText("Volunteer created.")
    ).toBeVisible({ timeout: 8_000 });

    // The new volunteer should now appear in the list.
    await expect(
      adminPage.locator("[class*='volName']").filter({ hasText: `${firstName} ${lastName}` })
    ).toBeVisible({ timeout: 8_000 });
  });
});

test.describe("Manage Volunteers — edit", () => {
  let editFirstName: string;
  let editLastName: string;

  test.beforeAll(async ({ adminToken }) => {
    editFirstName = uniqueName("EditFirst");
    editLastName  = uniqueName("EditLast");
    await createVolunteer(adminToken, {
      firstName: editFirstName,
      lastName:  editLastName,
      email:     uniqueEmail("vol-edit"),
      role:      "VOLUNTEER",
    });
  });

  test("admin can open the inline edit form for a volunteer", async ({
    adminPage,
  }) => {
    await adminPage.goto("/admin/volunteers");

    const card = adminPage.locator("[class*='volCard']").filter({
      hasText: `${editFirstName} ${editLastName}`,
    });
    await expect(card).toBeVisible({ timeout: 8_000 });
    await card.getByTitle("Edit volunteer").click();

    // The edit form appears inside the card; find First Name input via .field container.
    await expect(
      card.locator("[class*='field']").filter({ hasText: /first name/i }).locator("input")
    ).toBeVisible({ timeout: 5_000 });
  });

  test("admin can update the volunteer's last name", async ({ adminPage }) => {
    const updatedLast = uniqueName("UpdatedLast");

    await adminPage.goto("/admin/volunteers");

    const card = adminPage.locator("[class*='volCard']").filter({
      hasText: `${editFirstName} ${editLastName}`,
    });
    await expect(card).toBeVisible({ timeout: 8_000 });
    await card.getByTitle("Edit volunteer").click();

    // Update the last name field.
    const lastNameInput = card
      .locator("[class*='field']")
      .filter({ hasText: /last name/i })
      .locator("input");
    await lastNameInput.clear();
    await lastNameInput.fill(updatedLast);

    await card.getByRole("button", { name: "Save Changes" }).click();

    // Success banner.
    await expect(adminPage.getByText("Volunteer updated.")).toBeVisible({ timeout: 8_000 });

    // Updated name visible in the list.
    await expect(
      adminPage.locator("[class*='volName']").filter({ hasText: `${editFirstName} ${updatedLast}` })
    ).toBeVisible({ timeout: 8_000 });
  });
});

test.describe("Manage Volunteers — delete", () => {
  let deleteFirstName: string;
  let deleteLastName: string;

  test.beforeAll(async ({ adminToken }) => {
    deleteFirstName = uniqueName("DeleteFirst");
    deleteLastName  = uniqueName("DeleteLast");
    await createVolunteer(adminToken, {
      firstName: deleteFirstName,
      lastName:  deleteLastName,
      email:     uniqueEmail("vol-delete"),
      role:      "VOLUNTEER",
    });
  });

  test("admin can delete a volunteer after confirming", async ({ adminPage }) => {
    await adminPage.goto("/admin/volunteers");

    const card = adminPage.locator("[class*='volCard']").filter({
      hasText: `${deleteFirstName} ${deleteLastName}`,
    });
    await expect(card).toBeVisible({ timeout: 8_000 });

    // Accept the window.confirm dialog.
    adminPage.once("dialog", (dialog) => dialog.accept());
    await card.getByTitle("Delete volunteer").click();

    // Volunteer should be gone.
    await expect(
      adminPage.locator("[class*='volName']").filter({ hasText: `${deleteFirstName} ${deleteLastName}` })
    ).not.toBeVisible({ timeout: 8_000 });

    // Success banner.
    await expect(adminPage.getByText("Volunteer deleted.")).toBeVisible({ timeout: 8_000 });
  });

  test("dismissing the confirm dialog does NOT delete the volunteer", async ({
    adminToken,
    adminPage,
  }) => {
    const keepFirst = uniqueName("KeepFirst");
    const keepLast  = uniqueName("KeepLast");
    await createVolunteer(adminToken, {
      firstName: keepFirst,
      lastName:  keepLast,
      email:     uniqueEmail("vol-keep"),
      role:      "VOLUNTEER",
    });

    await adminPage.goto("/admin/volunteers");

    const card = adminPage.locator("[class*='volCard']").filter({
      hasText: `${keepFirst} ${keepLast}`,
    });
    await expect(card).toBeVisible({ timeout: 8_000 });

    // Dismiss the dialog.
    adminPage.once("dialog", (dialog) => dialog.dismiss());
    await card.getByTitle("Delete volunteer").click();

    // Volunteer should still be in the list.
    await expect(card).toBeVisible({ timeout: 5_000 });
  });
});
