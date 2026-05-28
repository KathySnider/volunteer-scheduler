/**
 * GraphQL fetch utilities.
 *
 * Three helper functions correspond to the three API endpoints:
 *   authGql      — /graphql/auth      (no auth required)
 *   volunteerGql — /graphql/volunteer (session cookie required)
 *   adminGql     — /graphql/admin     (session cookie + admin role required)
 *
 * Authentication is handled via an HttpOnly session cookie sent automatically
 * by the browser on every request (credentials: "include"). No token is passed
 * from JavaScript.
 */

// All GraphQL endpoints are proxied through the Next.js server via rewrites
// in next.config.mjs (/graphql/* → backend).  Relative paths are used so the
// browser always calls the same origin, keeping session cookies on one domain.
// NEXT_PUBLIC_GRAPHQL_*_URL env vars are no longer needed and should be left
// unset; they are kept here only as an escape hatch.
const AUTH_URL =
  process.env.NEXT_PUBLIC_GRAPHQL_AUTH_URL ||
  "/graphql/auth";

const VOLUNTEER_URL =
  process.env.NEXT_PUBLIC_GRAPHQL_VOLUNTEER_URL ||
  "/graphql/volunteer";

const ADMIN_URL =
  process.env.NEXT_PUBLIC_GRAPHQL_ADMIN_URL ||
  "/graphql/admin";

async function gqlFetch(url, query, variables) {
  const response = await fetch(url, {
    method: "POST",
    credentials: "include", // send the HttpOnly session cookie on every request
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ query, variables }),
  });

  // 401 means the session has expired or is invalid.
  // Clean up local state and redirect to login with an explanation.
  if (response.status === 401) {
    clearAuthToken();
    if (typeof window !== "undefined") {
      window.location.href = "/login?expired=1";
    }
    return null;
  }

  if (!response.ok) {
    throw new Error(`Server returned HTTP ${response.status}`);
  }

  return response.json();
}

export function authGql(query, variables) {
  return gqlFetch(AUTH_URL, query, variables);
}

export function volunteerGql(query, variables) {
  return gqlFetch(VOLUNTEER_URL, query, variables);
}

export function adminGql(query, variables) {
  return gqlFetch(ADMIN_URL, query, variables);
}

/**
 * Upload a single file as a GraphQL multipart request
 * (graphql-multipart-request-spec). Used for the attachFileToFeedback mutation.
 * The caller passes variables WITHOUT the file key — this function sets it to
 * null in `operations` and maps the real File object via the `map` part.
 */
export async function volunteerGqlUpload(query, variables, file) {
  const operations = JSON.stringify({
    query,
    variables: { ...variables, file: null },
  });
  const map = JSON.stringify({ "0": ["variables.file"] });

  const form = new FormData();
  form.append("operations", operations);
  form.append("map", map);
  form.append("0", file, file.name);

  const response = await fetch(VOLUNTEER_URL, {
    method: "POST",
    credentials: "include", // send the HttpOnly session cookie
    // Do NOT set Content-Type — the browser sets it with the correct boundary.
    body: form,
  });

  if (!response.ok) {
    throw new Error(`Server returned HTTP ${response.status}`);
  }
  return response.json();
}

/**
 * Fetch one attachment's binary data (returned as Base64 by the server) and
 * trigger a browser file-download. Pass useAdminEndpoint=true on admin pages.
 */
export async function downloadAttachment(attachmentId, useAdminEndpoint = false) {
  const url = useAdminEndpoint ? ADMIN_URL : VOLUNTEER_URL;
  const queryName = useAdminEndpoint ? "attachment" : "ownAttachment";
  const res = await gqlFetch(
    url,
    `query GetAttachment($id: Int!) {
      ${queryName}(attachmentId: $id) {
        filename
        mimeType
        data
      }
    }`,
    { id: attachmentId }
  );

  const att = res.data?.[queryName];
  if (!att) throw new Error("Attachment not found");

  // Base64 → Uint8Array → Blob → object URL → programmatic click
  const binary = atob(att.data);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
  const blob = new Blob([bytes], { type: att.mimeType });
  const objectUrl = URL.createObjectURL(blob);

  const a = document.createElement("a");
  a.href = objectUrl;
  a.download = att.filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(objectUrl);
}

/**
 * Persist the non-sensitive display values (email, role, name) to localStorage
 * and set the sessionActive flag. The real session lives in the HttpOnly cookie.
 */
export function setAuthInfo(email, role, name) {
  localStorage.setItem("sessionActive", "1");
  if (email) localStorage.setItem("authEmail", email);
  if (role)  localStorage.setItem("authRole", role);
  if (name)  localStorage.setItem("authName", name);
}

/**
 * Returns true when the user has an active session.
 * Checks the sessionActive flag — the session token lives in an HttpOnly cookie
 * that JavaScript cannot read.
 */
export function isAuthenticated() {
  if (typeof window === "undefined") return false;
  return localStorage.getItem("sessionActive") === "1";
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
  localStorage.removeItem("sessionActive");
  localStorage.removeItem("authEmail");
  localStorage.removeItem("authRole");
  localStorage.removeItem("authName");
}

/* =========================================================
   Venue cache
   =========================================================
   Module-level cache so admin pages (Create Event, Edit Event)
   share a single venue fetch for the browser session instead of
   each issuing their own request.

   Usage:
     getVenues()            — returns cached list; fetches on first call
     invalidateVenueCache() — call after any venue mutation so the next
                              navigation re-fetches fresh data
     addVenueToCache(v)     — optimistically append a just-created venue
   ========================================================= */

let _venueCache = null;
let _venueFetchPromise = null;

export async function getVenues() {
  if (_venueCache !== null) return _venueCache;
  if (_venueFetchPromise) return _venueFetchPromise;
  _venueFetchPromise = adminGql(`
    query { venues { id name address city state zipCode } }
  `)
    .then((res) => {
      _venueCache = res?.data?.venues ?? [];
      _venueFetchPromise = null;
      return _venueCache;
    })
    .catch((err) => {
      _venueFetchPromise = null;
      throw err;
    });
  return _venueFetchPromise;
}

export function invalidateVenueCache() {
  _venueCache = null;
}

export function addVenueToCache(venue) {
  if (_venueCache !== null) {
    _venueCache = [..._venueCache, venue];
  }
}

/**
 * Sign the user out: invalidate the server session cookie, then clear localStorage.
 * The server call is best-effort — localStorage is always cleared regardless.
 */
export async function signOut() {
  try {
    await authGql(`mutation { logout { success } }`);
  } catch {
    // Non-fatal — clear locally even if the server call fails.
  }
  clearAuthToken();
}
