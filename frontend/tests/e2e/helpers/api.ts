/**
 * Direct GraphQL API helpers for test setup.
 *
 * These call the API the same way the frontend does, bypassing the browser.
 * Use them in beforeEach / beforeAll blocks to seed data.
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

async function gql(
  url: string,
  query: string,
  variables?: Record<string, unknown>,
  token?: string
): Promise<Record<string, unknown>> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };
  if (token) headers["Authorization"] = `Bearer ${token}`;

  const res = await fetch(url, {
    method: "POST",
    headers,
    body: JSON.stringify({ query, variables }),
  });

  if (!res.ok) {
    const text = await res.text();
    throw new Error(`HTTP ${res.status}: ${text}`);
  }

  const json = (await res.json()) as {
    data?: Record<string, unknown>;
    errors?: Array<{ message: string }>;
  };
  if (json.errors?.length) {
    throw new Error(json.errors.map((e) => e.message).join("; "));
  }
  return json.data ?? {};
}

/* ------------------------------------------------------------------ */
/*  Auth helpers                                                         */
/* ------------------------------------------------------------------ */

/** Request a magic link (puts email in Mailhog). */
export async function requestMagicLink(email: string): Promise<void> {
  await gql(AUTH_URL, `mutation { requestMagicLink(email: "${email}") { success message } }`);
}

/** Consume a magic link token, returning the session token. */
export async function consumeMagicLink(token: string): Promise<string> {
  const data = await gql(
    AUTH_URL,
    `mutation ConsumeMagicLink($token: String!) {
      consumeMagicLink(token: $token) {
        success
        message
        sessionToken
      }
    }`,
    { token }
  );
  const result = data.consumeMagicLink as {
    success: boolean;
    message: string;
    sessionToken?: string;
  };
  if (!result.success || !result.sessionToken) {
    throw new Error(`consumeMagicLink failed: ${result.message}`);
  }
  return result.sessionToken;
}

/* ------------------------------------------------------------------ */
/*  Admin seeding helpers                                                */
/* ------------------------------------------------------------------ */

/**
 * Create a volunteer directly via the admin API.
 * Returns the volunteer's ID (from allVolunteers query after creation).
 */
export async function createVolunteer(
  adminToken: string,
  opts: {
    firstName: string;
    lastName: string;
    email: string;
    role?: "VOLUNTEER" | "ADMINISTRATOR";
  }
): Promise<string> {
  const data = await gql(
    ADMIN_URL,
    `mutation CreateVol($v: NewVolunteerInput!) { createVolunteer(newVol: $v) { success message id } }`,
    {
      v: {
        firstName: opts.firstName,
        lastName: opts.lastName,
        email: opts.email,
        role: opts.role ?? "VOLUNTEER",
      },
    },
    adminToken
  );
  const result = data.createVolunteer as { success: boolean; message: string; id?: string };
  if (!result.success || !result.id) throw new Error(`createVolunteer failed: ${result.message}`);
  return result.id;
}

/**
 * Create a venue, returning its ID.
 */
export async function createVenue(
  adminToken: string,
  opts: { name: string; city: string; state: string; ianaZone?: string }
): Promise<string> {
  const data = await gql(
    ADMIN_URL,
    `mutation CreateVenue($v: NewVenueInput!) { createVenue(newVenue: $v) { success message id } }`,
    {
      v: {
        name: opts.name,
        address: uniqueName("123 Test St "),
        city: opts.city,
        state: opts.state,
        ianaZone: opts.ianaZone ?? "America/New_York",
      },
    },
    adminToken
  );
  const result = data.createVenue as { success: boolean; message: string; id?: string };
  if (!result.success || !result.id) throw new Error(`createVenue failed: ${result.message}`);
  return result.id;
}

/**
 * Create a job type, returning its ID.
 */
export async function createJobType(
  adminToken: string,
  code: string,
  name: string
): Promise<number> {
  const data = await gql(
    ADMIN_URL,
    `mutation CreateJob($j: NewJobTypeInput!) { createJobType(newJob: $j) { success message id } }`,
    { j: { code: code.toLowerCase(), name, sortOrder: 25 } },
    adminToken
  );
  const result = data.createJobType as { success: boolean; message: string; id?: string };
  if (!result.success || !result.id) throw new Error(`createJobType failed: ${result.message}`);
  return parseInt(result.id, 10);
}

/** Find an event ID by exact name via filteredEvents query. Returns null if not found. */
export async function findEventIdByName(
  adminToken: string,
  name: string
): Promise<string | null> {
  const data = await gql(
    ADMIN_URL,
    `query { filteredEvents { id name } }`,
    undefined,
    adminToken
  );
  const events = data.filteredEvents as Array<{ id: string; name: string }>;
  return events.find((e) => e.name === name)?.id ?? null;
}

/** Delete an event by ID. */
export async function deleteEvent(adminToken: string, eventId: string): Promise<void> {
  await gql(
    ADMIN_URL,
    `mutation DeleteEvent($id: ID!) { deleteEvent(eventId: $id) { success message } }`,
    { id: eventId },
    adminToken
  );
}

/** Delete a venue by ID. */
export async function deleteVenue(adminToken: string, venueId: string): Promise<void> {
  await gql(
    ADMIN_URL,
    `mutation DeleteVenue($id: ID!) { deleteVenue(venueId: $id) { success message } }`,
    { id: venueId },
    adminToken
  );
}

/** Delete a job type by numeric ID. */
export async function deleteJobType(adminToken: string, jobTypeId: number): Promise<void> {
  await gql(
    ADMIN_URL,
    `mutation DeleteJobType($id: Int!) { deleteJobType(JobId: $id) { success message } }`,
    { id: jobTypeId },
    adminToken
  );
}

/** Delete a volunteer by ID. */
export async function deleteVolunteer(adminToken: string, volunteerId: string): Promise<void> {
  await gql(
    ADMIN_URL,
    `mutation DeleteVol($id: ID!) { deleteVolunteer(volunteerId: $id) { success message } }`,
    { id: volunteerId },
    adminToken
  );
}

/**
 * Create a bare event (no opportunities / shifts) via the admin API.
 * Useful for testing admin-only visibility of incomplete events.
 */
export async function createEventWithoutShifts(
  adminToken: string,
  opts: {
    eventName: string;
    venueId: string;
    startDateTime?: string;
    endDateTime?: string;
  }
): Promise<void> {
  await gql(
    ADMIN_URL,
    `mutation CreateEvent($e: NewEventInput!) { createEvent(newEvent: $e) { success message } }`,
    {
      e: {
        name: opts.eventName,
        description: "Test event — no shifts",
        eventType: "IN_PERSON",
        venueId: opts.venueId,
        fundingEntityId: 1,
        serviceTypes: [],
        eventDates: [
          {
            startDateTime: opts.startDateTime ?? "2027-11-01 09:00:00",
            endDateTime:   opts.endDateTime   ?? "2027-11-01 13:00:00",
            ianaZone: "America/New_York",
          },
        ],
      },
    },
    adminToken
  );
}

/**
 * Create an event with one date, one opportunity, and one shift.
 * Returns { eventId, shiftId }.
 */
export async function createEventWithShift(
  adminToken: string,
  opts: {
    eventName: string;
    venueId: string;
    jobTypeId: number;
    startDateTime: string; // e.g. "2027-06-15 09:00:00"
    endDateTime: string;
    maxVolunteers?: number;
  }
): Promise<{ eventId: string; shiftId: string }> {
  const data = await gql(
    ADMIN_URL,
    `mutation CreateEvent($e: NewEventInput!) { createEvent(newEvent: $e) { success message } }`,
    {
      e: {
        name: opts.eventName,
        description: "Test event",
        eventType: "IN_PERSON",
        venueId: opts.venueId,
        fundingEntityId: 1,
        serviceTypes: [],
        eventDates: [
          {
            startDateTime: opts.startDateTime,
            endDateTime: opts.endDateTime,
            ianaZone: "America/New_York",
          },
        ],
      },
    },
    adminToken
  );
  void data; // result is just success/message

  // Fetch the event to get its ID
  const eventsData = await gql(
    ADMIN_URL,
    `query { filteredEvents { id name } }`,
    undefined,
    adminToken
  );
  const events = eventsData.filteredEvents as Array<{ id: string; name: string }>;
  const event = events.find((e) => e.name === opts.eventName);
  if (!event) throw new Error(`Created event ${opts.eventName} not found`);

  // Create an opportunity (job assignment for the event)
  await gql(
    ADMIN_URL,
    `mutation CreateOpp($o: NewOpportunityInput!) { createOpportunity(newOpp: $o) { success message } }`,
    {
      o: {
        eventId: event.id,
        jobId: opts.jobTypeId,
        isVirtual: false,
        shifts: [
          {
            startDateTime: opts.startDateTime,
            endDateTime: opts.endDateTime,
            ianaZone: "America/New_York",
            maxVolunteers: opts.maxVolunteers ?? 5,
          },
        ],
      },
    },
    adminToken
  );

  // Fetch the shift ID
  const oppsData = await gql(
    ADMIN_URL,
    `query OppsForEvent($id: ID!) { opportunitiesForEvent(eventId: $id) { id shifts { id } } }`,
    { id: event.id },
    adminToken
  );
  const opps = oppsData.opportunitiesForEvent as Array<{
    id: string;
    shifts: Array<{ id: string }>;
  }>;
  const shiftId = opps[0]?.shifts[0]?.id;
  if (!shiftId) throw new Error("Could not find shift after creating opportunity");

  return { eventId: event.id, shiftId };
}

/* ------------------------------------------------------------------ */
/*  Volunteer helpers                                                    */
/* ------------------------------------------------------------------ */

/**
 * Attach a small file to an existing feedback item via the volunteer endpoint.
 * Used in test setup to seed attachment data without going through the UI.
 */
export async function attachFileToFeedback(
  volunteerToken: string,
  feedbackId: string,
  filename = "test-attachment.txt",
  content = "E2E test attachment content"
): Promise<void> {
  const operations = JSON.stringify({
    query: `mutation AttachFile($feedbackId: ID!, $file: Upload!) {
      attachFileToFeedback(feedbackId: $feedbackId, file: $file) { success message }
    }`,
    variables: { feedbackId, file: null },
  });
  const map = JSON.stringify({ "0": ["variables.file"] });

  const form = new FormData();
  form.append("operations", operations);
  form.append("map", map);
  form.append("0", new Blob([content], { type: "text/plain" }), filename);

  const res = await fetch(VOLUNTEER_URL, {
    method: "POST",
    headers: { Authorization: `Bearer ${volunteerToken}` },
    body: form,
  });
  if (!res.ok) throw new Error(`attachFileToFeedback: HTTP ${res.status}`);
  const json = (await res.json()) as {
    data?: { attachFileToFeedback?: { success: boolean; message?: string } };
    errors?: Array<{ message: string }>;
  };
  if (json.errors?.length) throw new Error(json.errors.map((e) => e.message).join("; "));
  if (!json.data?.attachFileToFeedback?.success) {
    throw new Error(`attachFileToFeedback failed: ${json.data?.attachFileToFeedback?.message ?? "unknown"}`);
  }
}

/** Submit feedback as a volunteer. Returns the new feedback's ID. */
export async function submitFeedback(
  volunteerToken: string,
  opts: {
    type?: string;
    subject: string;
    text: string;
    appPageName?: string;
  }
): Promise<string> {
  await gql(
    VOLUNTEER_URL,
    `mutation GiveFeedback($f: NewFeedbackInput!) { giveFeedback(feedback: $f) { success message } }`,
    {
      f: {
        type: opts.type ?? "BUG",
        subject: opts.subject,
        text: opts.text,
        app_page_name: opts.appPageName ?? "/events",
      },
    },
    volunteerToken
  );

  const data = await gql(
    VOLUNTEER_URL,
    `query { ownFeedback { id subject } }`,
    undefined,
    volunteerToken
  );
  const feedbacks = data.ownFeedback as Array<{ id: string; subject: string }>;
  const found = feedbacks.find((f) => f.subject === opts.subject);
  if (!found) throw new Error(`Submitted feedback '${opts.subject}' not found`);
  return found.id;
}

/* ------------------------------------------------------------------ */
/*  Unique test data generators                                          */
/* ------------------------------------------------------------------ */

let counter = Date.now();
export function uniqueEmail(prefix = "testuser"): string {
  return `${prefix}.${++counter}@e2e.test`;
}

export function uniqueName(prefix = "Test"): string {
  return `${prefix}${++counter}`;
}
