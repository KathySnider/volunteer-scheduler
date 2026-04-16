/**
 * Playwright fixtures that extend the base test with pre-authenticated pages.
 *
 * Usage:
 *   import { test, expect } from "../helpers/fixtures";
 *
 *   test("some test", async ({ volunteerPage, volunteerToken }) => { ... });
 */

import { test as base, type Page } from "@playwright/test";
import { clearMailbox, waitForEmail, extractMagicLink } from "./mailhog";
import { requestMagicLink, createVolunteer, uniqueEmail, consumeMagicLink } from "./api";

/**
 * Module-level cache — one admin session for the entire test run.
 * Exposed so auth.spec.ts can share it rather than consuming a separate
 * magic link (which would invalidate this session on single-session backends).
 */
let cachedAdminSession: { token: string; email: string } | null = null;

export async function getAdminSession(): Promise<{ token: string; email: string }> {
  if (cachedAdminSession) return cachedAdminSession;

  const adminEmail = process.env.E2E_ADMIN_EMAIL;
  if (!adminEmail) {
    throw new Error(
      "E2E_ADMIN_EMAIL env var is required. " +
        "Set it to an existing ADMINISTRATOR account in the test database."
    );
  }
  await clearMailbox();
  await requestMagicLink(adminEmail);
  const msg = await waitForEmail(adminEmail);
  const url = extractMagicLink(msg);
  const token = await consumeMagicLink(new URL(url).searchParams.get("token")!);
  cachedAdminSession = { token, email: adminEmail };
  return cachedAdminSession;
}

/**
 * Force a fresh admin session by clearing the cache and re-authenticating.
 * Call this when a request returns 401, indicating the cached session was
 * invalidated (e.g. a browser-based magic-link login created a new session
 * and the backend uses single-session-per-user semantics).
 */
async function refreshAdminSession(): Promise<{ token: string; email: string }> {
  cachedAdminSession = null;
  return getAdminSession();
}

/**
 * A single volunteer session object shared by volunteerToken, volunteerEmail,
 * and volunteerPage within the same test. This prevents each fixture from
 * creating its own independent volunteer account, which caused token/email
 * mismatches when multiple fixtures were requested for the same test.
 */
type VolunteerSession = { token: string; email: string };

type TestFixtures = {
  /** Internal: the single volunteer session for this test. */
  _volunteerSession: VolunteerSession;

  /** A browser page already logged in as a volunteer. */
  volunteerPage: Page;
  /** The session token for the volunteer logged into volunteerPage. */
  volunteerToken: string;
  /** Email address of the volunteer. */
  volunteerEmail: string;

  /** A browser page already logged in as an admin. */
  adminPage: Page;
  /** The session token for the admin. */
  adminToken: string;
  /** Email address of the admin. */
  adminEmail: string;
};

export const test = base.extend<TestFixtures>({
  // ------------------------------------------------------------------ admin
  adminToken: async ({}, use) => {
    const { token } = await getAdminSession();
    await use(token);
  },

  adminEmail: async ({}, use) => {
    const { email } = await getAdminSession();
    await use(email);
  },

  adminPage: async ({ browser, adminToken, adminEmail }, use) => {
    const ctx = await browser.newContext();
    const page = await ctx.newPage();
    await page.addInitScript(
      ({ token, email }) => {
        localStorage.setItem("authToken", token);
        localStorage.setItem("authEmail", email);
        localStorage.setItem("authRole", "ADMINISTRATOR");
        localStorage.setItem("authName", "Test Admin");
      },
      { token: adminToken, email: adminEmail }
    );
    await use(page);
    await ctx.close();
  },

  // --------------------------------------------------------------- volunteer
  // Single source of truth: one volunteer account per test.
  // Calls getAdminSession() directly (no fixture dependency) so it can
  // self-heal if the cached admin session was invalidated between tests
  // (e.g. by an auth test that consumed a new magic link in the browser).
  _volunteerSession: async ({}, use) => {
    const email = uniqueEmail("vol");

    // Obtain admin token, retrying once with a fresh session on 401.
    let { token: adminToken } = await getAdminSession();
    try {
      await createVolunteer(adminToken, {
        firstName: "Test",
        lastName: "Volunteer",
        email,
        role: "VOLUNTEER",
      });
    } catch (err) {
      if (String(err).includes("401") || String(err).includes("unauthorized")) {
        // Session was invalidated — refresh and retry once.
        ({ token: adminToken } = await refreshAdminSession());
        await createVolunteer(adminToken, {
          firstName: "Test",
          lastName: "Volunteer",
          email,
          role: "VOLUNTEER",
        });
      } else throw err;
    }

    await clearMailbox();
    await requestMagicLink(email);
    const msg = await waitForEmail(email);
    const url = extractMagicLink(msg);
    const token = await consumeMagicLink(new URL(url).searchParams.get("token")!);
    await use({ token, email });
  },

  volunteerToken: async ({ _volunteerSession }, use) => {
    await use(_volunteerSession.token);
  },

  volunteerEmail: async ({ _volunteerSession }, use) => {
    await use(_volunteerSession.email);
  },

  volunteerPage: async ({ browser, _volunteerSession }, use) => {
    const ctx = await browser.newContext();
    const page = await ctx.newPage();
    await page.addInitScript(
      ({ token, email }) => {
        localStorage.setItem("authToken", token);
        localStorage.setItem("authEmail", email);
        localStorage.setItem("authRole", "VOLUNTEER");
        localStorage.setItem("authName", "Test Volunteer");
      },
      { token: _volunteerSession.token, email: _volunteerSession.email }
    );
    await use(page);
    await ctx.close();
  },
});

export { expect } from "@playwright/test";
