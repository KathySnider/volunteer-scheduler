/**
 * E2E tests — Session cookie security
 *
 * Verifies properties of the HttpOnly cookie-based session system:
 *
 *  - Login sets an HttpOnly cookie that JavaScript cannot read
 *  - sessionActive (not authToken) is the localStorage login indicator
 *  - No raw session token ever appears in localStorage
 *  - The session cookie is present and cleared correctly on sign-out
 *  - Old-style authToken in localStorage is ignored by the auth guard
 */

import { test, expect } from "./helpers/fixtures";
import { clearMailbox, waitForEmail, extractMagicLink } from "./helpers/mailhog";
import { requestMagicLink, createVolunteer, deleteVolunteer, uniqueEmail } from "./helpers/api";
import { getAdminSession } from "./helpers/fixtures";

// ============================================================================
// Shared volunteer account for cookie-visibility tests
// ============================================================================

test.describe("HttpOnly session cookie", () => {
  let volunteerEmail: string;
  let volunteerId: string;

  test.beforeAll(async () => {
    volunteerEmail = uniqueEmail("sec-cookie");
    const { token: adminToken } = await getAdminSession();
    volunteerId = await createVolunteer(adminToken, {
      firstName: "Cookie",
      lastName: "Test",
      email: volunteerEmail,
      role: "VOLUNTEER",
    });
  });

  test.afterAll(async () => {
    if (volunteerId) {
      const { token: adminToken } = await getAdminSession();
      try { await deleteVolunteer(adminToken, volunteerId); } catch { /* ignore */ }
    }
  });

  test("session cookie is HttpOnly — document.cookie cannot read it", async ({ page }) => {
    await clearMailbox();
    await requestMagicLink(volunteerEmail);
    const msg = await waitForEmail(volunteerEmail);
    const magicUrl = extractMagicLink(msg);

    await page.goto(magicUrl);
    await page.waitForURL("**/events", { timeout: 8_000 });

    // JavaScript must not be able to read the session cookie.
    const jsCookies = await page.evaluate(() => document.cookie);
    expect(jsCookies).not.toContain("session");

    // Playwright's context-level cookies bypass HttpOnly and can see it —
    // verify the cookie exists and is correctly flagged.
    const cookies = await page.context().cookies();
    const sessionCookie = cookies.find((c) => c.name === "session");
    expect(sessionCookie).toBeDefined();
    expect(sessionCookie?.httpOnly).toBe(true);
    expect(sessionCookie?.value).toBeTruthy();
  });

  test("sessionActive is set in localStorage; authToken is absent after login", async ({
    page,
  }) => {
    await clearMailbox();
    await requestMagicLink(volunteerEmail);
    const msg = await waitForEmail(volunteerEmail);
    const magicUrl = extractMagicLink(msg);

    await page.goto(magicUrl);
    await page.waitForURL("**/events", { timeout: 8_000 });

    // The new login indicator.
    const sessionActive = await page.evaluate(() => localStorage.getItem("sessionActive"));
    expect(sessionActive).toBe("1");

    // The raw token must never appear in localStorage.
    const authToken = await page.evaluate(() => localStorage.getItem("authToken"));
    expect(authToken).toBeNull();

    // Role and name are still stored for display purposes.
    const role = await page.evaluate(() => localStorage.getItem("authRole"));
    expect(role).toBe("VOLUNTEER");
  });

  test("session cookie is present before sign-out and absent after", async ({ page }) => {
    await clearMailbox();
    await requestMagicLink(volunteerEmail);
    const msg = await waitForEmail(volunteerEmail);
    const magicUrl = extractMagicLink(msg);

    await page.goto(magicUrl);
    await page.waitForURL("**/events", { timeout: 8_000 });

    // Cookie is set after login.
    const before = await page.context().cookies();
    expect(before.find((c) => c.name === "session")?.value).toBeTruthy();

    // Sign out.
    await page.getByRole("button", { name: /sign out/i }).click();
    await page.waitForURL("**/login", { timeout: 5_000 });

    // Cookie must be cleared (gone or empty value).
    const after = await page.context().cookies();
    const sessionCookie = after.find((c) => c.name === "session");
    expect(!sessionCookie || !sessionCookie.value).toBeTruthy();

    // sessionActive must also be cleared.
    const sessionActive = await page.evaluate(() => localStorage.getItem("sessionActive"));
    expect(sessionActive).toBeNull();
  });
});

// ============================================================================
// Auth guard behaviour
// ============================================================================

test.describe("Auth guard uses sessionActive, not authToken", () => {
  test("authToken in localStorage does NOT grant access to protected pages", async ({
    page,
  }) => {
    // Simulate a browser that has the old-style token but not the new flag.
    await page.goto("/login");
    await page.evaluate(() => {
      localStorage.clear();
      localStorage.setItem("authToken", "an-old-token-that-no-longer-works");
    });

    await page.goto("/events");
    await page.waitForURL("**/login", { timeout: 5_000 });
    expect(page.url()).toContain("/login");
  });

  test("no localStorage state at all redirects to /login", async ({ page }) => {
    await page.goto("/login");
    await page.evaluate(() => localStorage.clear());

    await page.goto("/events");
    await page.waitForURL("**/login", { timeout: 5_000 });
    expect(page.url()).toContain("/login");
  });

  test("admin page rejects volunteer cookie", async ({ volunteerPage }) => {
    // A volunteer's session cookie must not allow access to admin pages.
    // The page code checks authRole and redirects non-admins to /events.
    await volunteerPage.goto("/admin/events");
    // Wait for the redirect. We cannot use "**/events" because that glob also
    // matches "/admin/events" (it ends with "/events"), so we anchor to the
    // origin — the redirect target is exactly <origin>/events with no prefix.
    await expect(volunteerPage).toHaveURL(
      /^https?:\/\/[^/]+\/events(?:\?|$)/,
      { timeout: 5_000 }
    );
    expect(volunteerPage.url()).not.toContain("/admin");
  });
});
