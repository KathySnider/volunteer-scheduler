/**
 * E2E tests — Authentication flow
 *
 * Covers:
 *  - Happy path: magic-link request → email arrives → click link → land on /events
 *  - Role routing: ADMINISTRATOR lands on /events (same page, admin menu shown)
 *  - Unknown email: "No account found" message shown, account request form offered
 *  - Invalid/expired token: sign-in error shown
 *  - Logged-out user hitting a protected page: redirected to /login
 */

// Use the extended test so we can reach adminPage/volunteerPage fixtures.
import { test, expect } from "./helpers/fixtures";
import { clearMailbox, waitForEmail, extractMagicLink } from "./helpers/mailhog";
import { requestMagicLink, createVolunteer, uniqueEmail } from "./helpers/api";
import { getAdminSession } from "./helpers/fixtures";

// We need a seeded admin token to create test volunteers.
// For auth-specific tests we create fresh volunteers inline.
const ADMIN_EMAIL = process.env.E2E_ADMIN_EMAIL!;

test.describe("Magic-link login — happy path", () => {
  let volunteerEmail: string;

  test.beforeAll(async () => {
    if (!ADMIN_EMAIL) throw new Error("E2E_ADMIN_EMAIL not set");

    // Create a fresh volunteer for these tests.
    // Re-use the shared admin session so we don't create a competing session
    // that could invalidate the cached token used by other test suites.
    volunteerEmail = uniqueEmail("auth-happy");
    const { token: adminToken } = await getAdminSession();

    await createVolunteer(adminToken, {
      firstName: "Happy",
      lastName: "Path",
      email: volunteerEmail,
      role: "VOLUNTEER",
    });
  });

  test("requests a magic link, receives email, clicks link, lands on /events", async ({
    page,
  }) => {
    await clearMailbox();

    // 1. Go to the login page
    await page.goto("/login");
    await expect(page.getByRole("heading", { name: "Sign In" })).toBeVisible();

    // 2. Enter the volunteer email
    await page.getByLabel("Email address").fill(volunteerEmail);
    await page.getByRole("button", { name: "Continue" }).click();

    // 3. "Check your email" confirmation is shown
    await expect(
      page.getByRole("heading", { name: "Check your email" })
    ).toBeVisible();
    await expect(page.getByText(volunteerEmail)).toBeVisible();

    // 4. Fetch the email from Mailhog and extract the magic link
    const msg = await waitForEmail(volunteerEmail);
    const magicUrl = extractMagicLink(msg);

    // 5. Navigate to the magic-link URL
    await page.goto(magicUrl);

    // 6. Should show "Signed in!" then redirect to /events
    await expect(page.getByRole("heading", { name: "Signed in!" })).toBeVisible();
    await page.waitForURL("**/events", { timeout: 8_000 });
    expect(page.url()).toContain("/events");
  });

  test("session token is stored in localStorage after login", async ({ page }) => {
    await clearMailbox();
    await requestMagicLink(volunteerEmail);
    const msg = await waitForEmail(volunteerEmail);
    const magicUrl = extractMagicLink(msg);

    await page.goto(magicUrl);
    await page.waitForURL("**/events", { timeout: 8_000 });

    const token = await page.evaluate(() => localStorage.getItem("authToken"));
    expect(token).toBeTruthy();

    const role = await page.evaluate(() => localStorage.getItem("authRole"));
    expect(role).toBe("VOLUNTEER");
  });
});

test.describe("Magic-link login — admin routing", () => {
  test("administrator sees admin menu items when logged in", async ({ adminPage }) => {
    await adminPage.goto("/events");

    // Open the user menu and check for admin-only items
    await adminPage.getByRole("button", { name: /menu|account|settings/i }).first().click();
    await expect(adminPage.getByRole("link", { name: "Manage Events" })).toBeVisible();
    await expect(adminPage.getByRole("link", { name: "Manage Volunteers" })).toBeVisible();
  });
});

test.describe("Logout", () => {
  const AUTH_URL =
    process.env.NEXT_PUBLIC_GRAPHQL_AUTH_URL || "http://localhost:8080/graphql/auth";
  const VOLUNTEER_URL =
    process.env.NEXT_PUBLIC_GRAPHQL_VOLUNTEER_URL || "http://localhost:8080/graphql/volunteer";

  test("logout mutation invalidates the server session token", async ({
    volunteerToken,
  }) => {
    // Verify the token works before logout.
    const beforeRes = await fetch(VOLUNTEER_URL, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${volunteerToken}`,
      },
      body: JSON.stringify({ query: "query { lookupValues { cities } }" }),
    });
    expect(beforeRes.status).toBe(200);

    // Call the logout mutation.
    const logoutRes = await fetch(AUTH_URL, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        query: `mutation Logout($token: String!) { logout(token: $token) { success } }`,
        variables: { token: volunteerToken },
      }),
    });
    const logoutJson = (await logoutRes.json()) as {
      data?: { logout?: { success: boolean } };
    };
    expect(logoutJson.data?.logout?.success).toBe(true);

    // The same token should now be rejected by the authenticated endpoint.
    const afterRes = await fetch(VOLUNTEER_URL, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${volunteerToken}`,
      },
      body: JSON.stringify({ query: "query { lookupValues { cities } }" }),
    });
    expect(afterRes.status).toBe(401);
  });

  test("sign-out via UI clears session and redirects to /login", async ({
    volunteerPage,
  }) => {
    await volunteerPage.goto("/events");
    await expect(
      volunteerPage.getByRole("heading", { name: /volunteer events/i })
    ).toBeVisible({ timeout: 8_000 });

    // Sign Out is always visible in the UserMenu (no menu to open for volunteers).
    await volunteerPage.getByRole("button", { name: /sign out/i }).click();

    // Should land on /login.
    await volunteerPage.waitForURL("**/login", { timeout: 5_000 });
    expect(volunteerPage.url()).toContain("/login");

    // localStorage should be cleared by signOut().
    // Note: we can't re-navigate to /events here to test the redirect because
    // the volunteerPage fixture uses addInitScript which re-seeds localStorage
    // on every navigation. The redirect-when-unauthenticated behaviour is
    // covered by the "unauthenticated user visiting /events" test in this file.
    const token = await volunteerPage.evaluate(() => localStorage.getItem("authToken"));
    expect(token).toBeNull();
  });
});

test.describe("Magic-link login — error cases", () => {
  test("unknown email shows 'No account found' with request-account option", async ({
    page,
  }) => {
    await page.goto("/login");
    await page.getByLabel("Email address").fill("nobody@definitely-not-real.test");
    await page.getByRole("button", { name: "Continue" }).click();

    await expect(
      page.getByRole("heading", { name: "No account found" })
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Request an Account" })
    ).toBeVisible();
  });

  test("invalid magic-link token shows sign-in failed page", async ({
    page,
  }) => {
    await page.goto("/auth/magic-link?token=totally-invalid-token");
    await expect(
      page.getByRole("heading", { name: "Sign-in failed" })
    ).toBeVisible();
    await expect(
      page.getByRole("link", { name: /new sign-in link/i })
    ).toBeVisible();
  });

  test("missing token in magic-link URL shows error", async ({ page }) => {
    await page.goto("/auth/magic-link");
    await expect(
      page.getByRole("heading", { name: "Sign-in failed" })
    ).toBeVisible();
    await expect(page.getByText(/no token/i)).toBeVisible();
  });

  test("unauthenticated user visiting /events is redirected to /login", async ({
    page,
  }) => {
    // Make sure localStorage is clear
    await page.goto("/");
    await page.evaluate(() => localStorage.clear());

    await page.goto("/events");
    await page.waitForURL("**/login", { timeout: 5_000 });
    expect(page.url()).toContain("/login");
  });

  test("account request form can be submitted for unknown email", async ({
    page,
  }) => {
    const newEmail = uniqueEmail("newrequest");

    await page.goto("/login");
    await page.getByLabel("Email address").fill(newEmail);
    await page.getByRole("button", { name: "Continue" }).click();
    await page.getByRole("button", { name: "Request an Account" }).click();

    // Fill in the request form
    await page.getByLabel("First name").fill("New");
    await page.getByLabel("Last name").fill("User");
    await page.getByRole("button", { name: "Submit Request" }).click();

    await expect(
      page.getByRole("heading", { name: "Request Submitted" })
    ).toBeVisible();
  });
});
