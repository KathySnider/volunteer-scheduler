/**
 * GraphQL fetch utilities.
 *
 * Three helper functions correspond to the three API endpoints:
 *   authGql      — /graphql/auth      (no token required)
 *   volunteerGql — /graphql/volunteer (Bearer token required)
 *   adminGql     — /graphql/admin     (Bearer token + admin role required)
 */

const AUTH_URL =
  process.env.NEXT_PUBLIC_GRAPHQL_AUTH_URL ||
  "http://localhost:8080/graphql/auth";

const VOLUNTEER_URL =
  process.env.NEXT_PUBLIC_GRAPHQL_VOLUNTEER_URL ||
  "http://localhost:8080/graphql/volunteer";

const ADMIN_URL =
  process.env.NEXT_PUBLIC_GRAPHQL_ADMIN_URL ||
  "http://localhost:8080/graphql/admin";

async function gqlFetch(url, query, variables, token) {
  const headers = { "Content-Type": "application/json" };
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const response = await fetch(url, {
    method: "POST",
    headers,
    body: JSON.stringify({ query, variables }),
  });

  if (!response.ok) {
    throw new Error(`Server returned HTTP ${response.status}`);
  }

  return response.json();
}

export function authGql(query, variables) {
  return gqlFetch(AUTH_URL, query, variables);
}

export function volunteerGql(query, variables, token) {
  return gqlFetch(VOLUNTEER_URL, query, variables, token);
}

export function adminGql(query, variables, token) {
  return gqlFetch(ADMIN_URL, query, variables, token);
}

/** Read the session token from localStorage (client-side only). */
export function getAuthToken() {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("authToken");
}

/** Persist the session token, email, role, and display name to localStorage. */
export function setAuthToken(token, email, role, name) {
  localStorage.setItem("authToken", token);
  if (email) localStorage.setItem("authEmail", email);
  if (role)  localStorage.setItem("authRole", role);
  if (name)  localStorage.setItem("authName", name);
}

/** Read the volunteer's role from localStorage (client-side only). */
export function getAuthRole() {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("authRole");
}

/** Read the volunteer's display name from localStorage (client-side only). */
export function getAuthName() {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("authName");
}

/** Clear all session data from localStorage. */
export function clearAuthToken() {
  localStorage.removeItem("authToken");
  localStorage.removeItem("authEmail");
  localStorage.removeItem("authRole");
  localStorage.removeItem("authName");
}
