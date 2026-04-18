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

/**
 * Upload a single file as a GraphQL multipart request
 * (graphql-multipart-request-spec). Used for the attachFileToFeedback mutation.
 * The caller passes variables WITHOUT the file key — this function sets it to
 * null in `operations` and maps the real File object via the `map` part.
 */
export async function volunteerGqlUpload(query, variables, file, token) {
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
    // Do NOT set Content-Type — the browser sets it with the correct boundary.
    headers: { Authorization: `Bearer ${token}` },
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
export async function downloadAttachment(attachmentId, token, useAdminEndpoint = false) {
  const url = useAdminEndpoint ? ADMIN_URL : VOLUNTEER_URL;
  const res = await gqlFetch(
    url,
    `query GetAttachment($id: ID!) {
      attachment(attachmentId: $id) {
        filename
        mimeType
        data
      }
    }`,
    { id: attachmentId },
    token
  );

  const att = res.data?.attachment;
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

/**
 * Sign the user out: invalidate the server session, then clear localStorage.
 * The server call is best-effort — localStorage is always cleared regardless.
 */
export async function signOut(token) {
  if (token) {
    try {
      await authGql(
        `mutation Logout($token: String!) { logout(token: $token) { success } }`,
        { token }
      );
    } catch {
      // Non-fatal — clear locally even if the server call fails.
    }
  }
  clearAuthToken();
}
