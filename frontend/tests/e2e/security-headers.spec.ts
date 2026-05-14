/**
 * E2E tests — HTTP security response headers
 *
 * Verifies that every page response includes the security headers added in
 * next.config.mjs.  Tests cover:
 *
 *  - X-Frame-Options: DENY
 *  - X-Content-Type-Options: nosniff
 *  - Referrer-Policy: strict-origin-when-cross-origin
 *  - Permissions-Policy (camera, microphone, geolocation denied)
 *  - Strict-Transport-Security (max-age present)
 *  - Content-Security-Policy (key directives present)
 *
 * Strategy: intercept the first navigation response on both a public page
 * (/login) and a protected page (/events) to confirm headers are present on
 * all routes, not just one.
 */

import { test, expect } from "@playwright/test";

// ---------------------------------------------------------------------------
// Helper — navigate to a URL and capture the response headers.
// ---------------------------------------------------------------------------

async function getResponseHeaders(
  page: Parameters<typeof test>[1] extends (args: { page: infer P }) => unknown ? P : never,
  url: string
): Promise<Record<string, string>> {
  const [response] = await Promise.all([
    page.waitForResponse((r) => r.url().includes(url) && r.status() < 400),
    page.goto(url),
  ]);
  return response?.headers() ?? {};
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

test.describe("HTTP security headers — public page (/login)", () => {
  let headers: Record<string, string>;

  test.beforeEach(async ({ page }) => {
    const [response] = await Promise.all([
      page.waitForResponse((r) => new URL(r.url()).pathname === "/login"),
      page.goto("/login"),
    ]);
    headers = response?.headers() ?? {};
  });

  test("X-Frame-Options is DENY", async () => {
    expect(headers["x-frame-options"]).toBe("DENY");
  });

  test("X-Content-Type-Options is nosniff", async () => {
    expect(headers["x-content-type-options"]).toBe("nosniff");
  });

  test("Referrer-Policy is strict-origin-when-cross-origin", async () => {
    expect(headers["referrer-policy"]).toBe("strict-origin-when-cross-origin");
  });

  test("Permissions-Policy denies camera, microphone, and geolocation", async () => {
    const policy = headers["permissions-policy"] ?? "";
    expect(policy).toContain("camera=()");
    expect(policy).toContain("microphone=()");
    expect(policy).toContain("geolocation=()");
  });

  test("Strict-Transport-Security has a max-age", async () => {
    const hsts = headers["strict-transport-security"] ?? "";
    expect(hsts).toMatch(/max-age=\d+/);
  });

  test("Content-Security-Policy includes key directives", async () => {
    const csp = headers["content-security-policy"] ?? "";
    expect(csp).toContain("default-src");
    expect(csp).toContain("script-src");
    expect(csp).toContain("frame-ancestors 'none'");
    expect(csp).toContain("base-uri 'self'");
    expect(csp).toContain("form-action 'self'");
  });
});

test.describe("HTTP security headers — protected page (/events)", () => {
  let headers: Record<string, string>;

  test.beforeEach(async ({ page }) => {
    // /events redirects unauthenticated users to /login.
    // We capture the /login redirect response — headers are set on all routes.
    const [response] = await Promise.all([
      page.waitForResponse((r) => r.status() < 400),
      page.goto("/events"),
    ]);
    headers = response?.headers() ?? {};
  });

  test("X-Frame-Options is DENY", async () => {
    expect(headers["x-frame-options"]).toBe("DENY");
  });

  test("X-Content-Type-Options is nosniff", async () => {
    expect(headers["x-content-type-options"]).toBe("nosniff");
  });

  test("Content-Security-Policy is present", async () => {
    expect(headers["content-security-policy"]).toBeTruthy();
  });
});
