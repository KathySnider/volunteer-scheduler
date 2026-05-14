/**
 * Next.js Edge Middleware — admin route protection
 *
 * Runs server-side before any admin page renders. Reads the HttpOnly session
 * cookie, asks the backend for the volunteer's role, and redirects if:
 *   - No session cookie present  → /login
 *   - Session invalid/expired    → /login
 *   - Role is not ADMINISTRATOR  → /events
 *   - Backend unreachable        → /login  (fail secure)
 *
 * Because this runs before the page renders, a volunteer who manually sets
 * authRole="ADMINISTRATOR" in localStorage will never see the admin page shell
 * — they are redirected by the server before any HTML is sent.
 *
 * Server-side URL note:
 *   GRAPHQL_VOLUNTEER_URL (no NEXT_PUBLIC prefix) can be set to an internal
 *   service URL in production (e.g. a Railway private network address) to
 *   avoid the public internet hop. Falls back to NEXT_PUBLIC_GRAPHQL_VOLUNTEER_URL
 *   which works fine in development and is acceptable in production.
 */

import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

const VOLUNTEER_API_URL =
  process.env.GRAPHQL_VOLUNTEER_URL ||
  process.env.NEXT_PUBLIC_GRAPHQL_VOLUNTEER_URL ||
  "http://localhost:8080/graphql/volunteer";

export async function middleware(request: NextRequest) {
  const sessionCookie = request.cookies.get("session");

  if (!sessionCookie?.value) {
    return NextResponse.redirect(new URL("/login", request.url));
  }

  try {
    const res = await fetch(VOLUNTEER_API_URL, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Cookie: `session=${sessionCookie.value}`,
      },
      body: JSON.stringify({ query: "{ volunteerProfile { role } }" }),
    });

    if (!res.ok) {
      return NextResponse.redirect(new URL("/login", request.url));
    }

    const json = (await res.json()) as {
      data?: { volunteerProfile?: { role?: string } };
    };
    const role = json?.data?.volunteerProfile?.role;

    if (role !== "ADMINISTRATOR") {
      return NextResponse.redirect(new URL("/events", request.url));
    }
  } catch {
    // Backend unreachable — fail secure rather than letting the request through.
    return NextResponse.redirect(new URL("/login", request.url));
  }

  return NextResponse.next();
}

export const config = {
  // Match all routes under /admin/ — does not match /admin itself (no such page).
  matcher: ["/admin/:path*"],
};
