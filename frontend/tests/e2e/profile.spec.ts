/**
 * E2E tests — Volunteer profile editing
 *
 * Covers:
 *  - Happy path: profile page loads pre-populated with current data
 *  - Happy path: volunteer updates name/phone/zip, sees success banner
 *  - Display name in UserMenu updates after save
 *  - Error: submit with blank required field shows validation
 *  - Unauthenticated user is redirected to /login
 */

import { test, expect } from "./helpers/fixtures";

test.describe("Profile page — loading", () => {
  test("profile page loads and shows pre-filled name and email", async ({
    volunteerPage,
    volunteerEmail,
  }) => {
    await volunteerPage.goto("/profile");

    // First and last name fields should be pre-filled
    await expect(volunteerPage.getByLabel("First Name")).toHaveValue("Test", { timeout: 5_000 });
    await expect(volunteerPage.getByLabel("Last Name")).toHaveValue("Volunteer");

    // Email should match the account we created
    await expect(volunteerPage.getByLabel("Email Address")).toHaveValue(volunteerEmail);
  });

  test("My Profile link is visible in the top bar for volunteers", async ({
    volunteerPage,
  }) => {
    await volunteerPage.goto("/events");
    await expect(volunteerPage.getByRole("link", { name: "My Profile" })).toBeVisible();
  });

  test("My Profile link is visible in the top bar for admins", async ({
    adminPage,
  }) => {
    await adminPage.goto("/events");
    await expect(adminPage.getByRole("link", { name: "My Profile" })).toBeVisible();
  });
});

test.describe("Profile page — editing", () => {
  test("volunteer can update phone and zip code", async ({ volunteerPage }) => {
    await volunteerPage.goto("/profile");
    await volunteerPage.waitForSelector("input#firstName", { timeout: 5_000 });

    await volunteerPage.getByLabel("Phone").fill("(555) 123-4567");
    await volunteerPage.getByLabel("Zip Code").fill("98101");
    await volunteerPage.getByRole("button", { name: "Save Changes" }).click();

    await expect(
      volunteerPage.getByText("Profile updated.")
    ).toBeVisible({ timeout: 5_000 });
  });

  test("volunteer can update their name and the top bar reflects the change", async ({
    volunteerPage,
  }) => {
    await volunteerPage.goto("/profile");
    await volunteerPage.waitForSelector("input#firstName", { timeout: 5_000 });

    await volunteerPage.getByLabel("First Name").fill("Updated");
    await volunteerPage.getByLabel("Last Name").fill("Name");
    await volunteerPage.getByRole("button", { name: "Save Changes" }).click();

    await expect(
      volunteerPage.getByText("Profile updated.")
    ).toBeVisible({ timeout: 5_000 });

    // The display name in the top bar should now show the new name
    await expect(volunteerPage.getByText("Updated Name")).toBeVisible();
  });
});

test.describe("Profile page — validation", () => {
  test("clearing first name and saving shows validation error", async ({
    volunteerPage,
  }) => {
    await volunteerPage.goto("/profile");
    await volunteerPage.waitForSelector("input#firstName", { timeout: 5_000 });

    await volunteerPage.getByLabel("First Name").fill("");
    await volunteerPage.getByRole("button", { name: "Save Changes" }).click();

    // HTML5 required validation fires before the API call
    const firstNameInput = volunteerPage.locator("input#firstName");
    const validity = await firstNameInput.evaluate(
      (el: HTMLInputElement) => el.validity.valid
    );
    expect(validity).toBe(false);
  });

  test("clearing email and saving shows validation error", async ({
    volunteerPage,
  }) => {
    await volunteerPage.goto("/profile");
    await volunteerPage.waitForSelector("input#email", { timeout: 5_000 });

    await volunteerPage.getByLabel("Email Address").fill("");
    await volunteerPage.getByRole("button", { name: "Save Changes" }).click();

    const emailInput = volunteerPage.locator("input#email");
    const validity = await emailInput.evaluate(
      (el: HTMLInputElement) => el.validity.valid
    );
    expect(validity).toBe(false);
  });
});

test.describe("Profile page — access control", () => {
  test("unauthenticated user is redirected to /login", async ({ page }) => {
    await page.goto("/");
    await page.evaluate(() => localStorage.clear());
    await page.goto("/profile");
    await page.waitForURL("**/login", { timeout: 5_000 });
    expect(page.url()).toContain("/login");
  });
});
