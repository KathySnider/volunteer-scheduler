/**
 * E2E tests — Feedback flow
 *
 * Covers:
 *  - Happy path: volunteer submits feedback via FeedbackButton → appears in My Feedback
 *  - Admin sees new feedback in admin list
 *  - Admin asks a question → volunteer sees question in My Feedback detail
 *  - Admin resolves feedback → status shown as resolved
 *  - Error: submit with missing fields (browser validation)
 *  - Volunteer cannot see ADMIN_NOTE entries
 *  - Attachments: file picker, size validation, submit with file, Download buttons
 */

import { test, expect } from "./helpers/fixtures";
import { submitFeedback, attachFileToFeedback, deleteFeedback, uniqueName } from "./helpers/api";

const ADMIN_URL =
  process.env.NEXT_PUBLIC_GRAPHQL_ADMIN_URL ||
  "http://localhost:8080/graphql/admin";

async function adminGql(
  query: string,
  variables: Record<string, unknown>,
  token: string
): Promise<Record<string, unknown>> {
  const res = await fetch(ADMIN_URL, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ query, variables }),
  });
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  const json = (await res.json()) as {
    data?: Record<string, unknown>;
    errors?: Array<{ message: string }>;
  };
  if (json.errors?.length) throw new Error(json.errors.map((e) => e.message).join("; "));
  return json.data ?? {};
}

test.describe("Feedback — volunteer submits feedback", () => {
  // Collect IDs of feedback created via API so afterAll can clean them up.
  const createdFeedbackIds: string[] = [];

  test.afterAll(async ({ adminToken }) => {
    for (const id of createdFeedbackIds) {
      try { await deleteFeedback(adminToken, id); } catch { /* ignore */ }
    }
  });

  test("FeedbackButton modal opens, volunteer submits, sees confirmation", async ({
    volunteerPage,
  }) => {
    await volunteerPage.goto("/events");

    // Open the feedback button
    const feedbackBtn = volunteerPage.getByRole("button", { name: "Submit feedback" });
    await expect(feedbackBtn).toBeVisible({ timeout: 5_000 });
    await feedbackBtn.click();

    // Modal should appear — scope all further interactions to it
    const dialog = volunteerPage.getByRole("dialog", { name: "Feedback form" });
    await expect(dialog.getByRole("heading", { name: /feedback/i })).toBeVisible();

    // Fill in the form (use exact label name "Type" to avoid matching events page "Job Type")
    await dialog.getByLabel("Type").selectOption("BUG");
    await dialog.getByLabel(/subject/i).fill("Test feedback subject");
    await dialog.getByLabel(/description/i).fill("This is a test bug report from an E2E test.");

    await dialog.getByRole("button", { name: "Send Feedback" }).click();

    // Should show a success state
    await expect(
      volunteerPage.getByText(/thank you|submitted|success/i)
    ).toBeVisible({ timeout: 5_000 });
  });

  test("submitted feedback appears on My Feedback page", async ({
    volunteerPage,
    volunteerToken,
  }) => {
    const subject = uniqueName("E2EFeedback");
    const id = await submitFeedback(volunteerToken, {
      subject,
      text: "Feedback body for E2E test",
    });
    createdFeedbackIds.push(id);

    await volunteerPage.goto("/my-feedback");
    await expect(volunteerPage.getByText(subject)).toBeVisible({ timeout: 5_000 });
  });

  test("My Feedback detail shows original feedback text", async ({
    volunteerPage,
    volunteerToken,
  }) => {
    const subject = uniqueName("DetailFeedback");
    const feedbackId = await submitFeedback(volunteerToken, {
      subject,
      text: "Detail view body text",
    });
    createdFeedbackIds.push(feedbackId);

    await volunteerPage.goto(`/my-feedback/${feedbackId}`);
    await expect(volunteerPage.getByText("Detail view body text")).toBeVisible({ timeout: 5_000 });
  });
});

test.describe("Feedback — admin workflow", () => {
  let feedbackId: string;
  // Save the beforeAll volunteer's token so test 7 can view the feedback
  // as the same volunteer who submitted it (ownFeedback is scoped per-user).
  let workflowVolunteerToken: string;
  const feedbackSubject = uniqueName("AdminWorkflow");

  test.beforeAll(async ({ volunteerToken }) => {
    workflowVolunteerToken = volunteerToken;
    feedbackId = await submitFeedback(volunteerToken, {
      subject: feedbackSubject,
      text: "Please help me understand this feature.",
    });
  });

  test.afterAll(async ({ adminToken }) => {
    if (feedbackId) {
      try { await deleteFeedback(adminToken, feedbackId); } catch { /* ignore */ }
    }
  });

  test("admin sees submitted feedback in admin feedback list", async ({
    adminPage,
  }) => {
    await adminPage.goto("/admin/feedback");
    await expect(adminPage.getByText(feedbackSubject)).toBeVisible({ timeout: 5_000 });
  });

  test("admin can open feedback detail page", async ({
    adminPage,
  }) => {
    await adminPage.goto(`/admin/feedback/${feedbackId}`);
    await expect(adminPage.getByText("Please help me understand this feature.")).toBeVisible({ timeout: 5_000 });
  });

  test("admin asks a question — status changes to QUESTION_SENT", async ({
    adminPage,
    adminToken,
  }) => {
    const result = await adminGql(
      `mutation Q($input: FeedbackEmailInput!) { emailFeedbackSubmitter(input: $input) { success message } }`,
      {
        input: {
          feedbackId: parseInt(feedbackId, 10),
          emailText: "Can you provide more details?",
          requireReply: true,
        },
      },
      adminToken
    ) as { emailFeedbackSubmitter?: { success: boolean; message?: string } };

    if (!result.emailFeedbackSubmitter?.success) {
      throw new Error(
        `emailFeedbackSubmitter mutation failed: ${result.emailFeedbackSubmitter?.message ?? "unknown error"}`
      );
    }

    await adminPage.goto(`/admin/feedback/${feedbackId}`);
    // Use exact match — the note type label also contains "question sent" (lowercase)
    // so a partial/case-insensitive match would resolve to two elements.
    await expect(
      adminPage.getByText("Question Sent", { exact: true })
    ).toBeVisible({ timeout: 5_000 });
  });

  test("volunteer can see the question in My Feedback detail but not admin notes", async ({
    browser,
    adminToken,
  }) => {
    // First add an admin-only note to ensure it is hidden from volunteer
    await adminGql(
      `mutation AddNote($input: FeedbackNoteInput!) { addFeedbackNote(note: $input) { success message } }`,
      {
        input: {
          feedbackId: parseInt(feedbackId, 10),
          note: "INTERNAL: this is an admin-only note",
        },
      },
      adminToken
    );

    // Create a page logged in as the volunteer who actually submitted the feedback.
    // We must use their session — ownFeedback is scoped per-user, so a different
    // volunteer's session would not find feedbackId.
    const ctx = await browser.newContext();
    await ctx.addCookies([{
      name: "session", value: workflowVolunteerToken,
      domain: "localhost", path: "/",
      httpOnly: true, secure: false, sameSite: "Lax",
    }]);
    const page = await ctx.newPage();
    await page.addInitScript(() => {
      localStorage.setItem("sessionActive", "1");
      localStorage.setItem("authRole", "VOLUNTEER");
      localStorage.setItem("authName", "Test Volunteer");
    });

    try {
      await page.goto(`/my-feedback/${feedbackId}`);

      // The question text should be visible
      await expect(
        page.getByText("Can you provide more details?")
      ).toBeVisible({ timeout: 5_000 });

      // The internal admin note must NOT be visible
      await expect(
        page.getByText("INTERNAL: this is an admin-only note")
      ).not.toBeVisible();
    } finally {
      await ctx.close();
    }
  });

  test("admin resolves feedback — status shows as resolved", async ({
    adminPage,
    adminToken,
  }) => {
    await adminGql(
      `mutation R($input: FeedbackStatusUpdateInput!) { updateFeedbackStatus(su: $input) { success message } }`,
      {
        input: {
          feedbackId: parseInt(feedbackId, 10),
          status: "RESOLVED_REJECTED",
          note: "Not reproducible, closing.",
        },
      },
      adminToken
    );

    await adminPage.goto(`/admin/feedback/${feedbackId}`);
    // Use exact match to target only the status badge ("Rejected"),
    // avoiding <strong>Resolved:</strong> and the "This feedback has been resolved." div.
    await expect(
      adminPage.getByText("Rejected", { exact: true })
    ).toBeVisible({ timeout: 5_000 });
  });
});

test.describe("Feedback — error cases", () => {
  test("FeedbackButton submit with empty subject shows browser validation", async ({
    volunteerPage,
  }) => {
    await volunteerPage.goto("/events");
    await volunteerPage.getByRole("button", { name: "Submit feedback" }).click();

    // Scope to the modal to avoid ambiguity with events page elements
    const dialog = volunteerPage.getByRole("dialog", { name: "Feedback form" });

    // Leave subject blank and try to submit
    await dialog
      .getByLabel(/description/i)
      .fill("Description without a subject");
    await dialog.getByRole("button", { name: "Send Feedback" }).click();

    // The subject field should be invalid (HTML5 required validation)
    const subjectInput = dialog.getByLabel(/subject/i);
    const validity = await subjectInput.evaluate(
      (el: HTMLInputElement) => el.validity.valid
    );
    expect(validity).toBe(false);
  });

  test("unauthenticated user visiting /my-feedback is redirected to /login", async ({
    page,
  }) => {
    await page.goto("/");
    await page.evaluate(() => localStorage.clear());
    await page.goto("/my-feedback");
    await page.waitForURL("**/login", { timeout: 5_000 });
    expect(page.url()).toContain("/login");
  });
});

/* ------------------------------------------------------------------ */
/*  Feedback — attachments                                             */
/* ------------------------------------------------------------------ */

test.describe("FeedbackButton — attachments", () => {
  // Subjects of feedback submitted via the UI — used in afterAll to locate and
  // delete via the admin API (which can see all feedback regardless of owner).
  const uiSubmittedSubjects: string[] = [];

  test.afterAll(async ({ adminToken }) => {
    if (uiSubmittedSubjects.length === 0) return;
    try {
      const res = await fetch(
        process.env.NEXT_PUBLIC_GRAPHQL_ADMIN_URL || "http://localhost:8080/graphql/admin",
        {
          method: "POST",
          headers: { "Content-Type": "application/json", Authorization: `Bearer ${adminToken}` },
          body: JSON.stringify({ query: `query { feedback { id subject } }` }),
        }
      );
      const json = (await res.json()) as { data?: { feedback?: Array<{ id: string; subject: string }> } };
      for (const fb of json.data?.feedback ?? []) {
        if (uiSubmittedSubjects.includes(fb.subject)) {
          try { await deleteFeedback(adminToken, fb.id); } catch { /* ignore */ }
        }
      }
    } catch { /* ignore */ }
  });

  /** Open the feedback modal and return the dialog locator. */
  async function openFeedbackModal(page: { goto: Function; getByRole: Function }) {
    await page.goto("/events");
    await page.getByRole("button", { name: "Submit feedback" }).click();
    const dialog = page.getByRole("dialog", { name: "Feedback form" });
    await expect(dialog.getByRole("heading", { name: /feedback/i })).toBeVisible({ timeout: 5_000 });
    return dialog;
  }

  test("selecting a file adds it to the file list", async ({ volunteerPage }) => {
    const dialog = await openFeedbackModal(volunteerPage);

    await dialog.locator('input[type="file"]').setInputFiles({
      name: "screenshot.png",
      mimeType: "image/png",
      buffer: Buffer.from("fake-png-content"),
    });

    await expect(dialog.getByText("screenshot.png")).toBeVisible({ timeout: 3_000 });
  });

  test("file over 5 MB shows an error in the file list", async ({ volunteerPage }) => {
    const dialog = await openFeedbackModal(volunteerPage);

    // 6 MB buffer — exceeds the 5 MB per-file limit
    const bigBuffer = Buffer.alloc(6 * 1024 * 1024, "x");
    await dialog.locator('input[type="file"]').setInputFiles({
      name: "too-big.bin",
      mimeType: "application/octet-stream",
      buffer: bigBuffer,
    });

    await expect(dialog.getByText(/exceeds the 5 MB limit/i)).toBeVisible({ timeout: 3_000 });
  });

  test("submit with attachment — attachment shows on My Feedback detail page", async ({
    volunteerPage,
  }) => {
    const subject = uniqueName("AttachSubmit");
    uiSubmittedSubjects.push(subject);
    const dialog = await openFeedbackModal(volunteerPage);

    // Fill in the required fields
    await dialog.getByLabel(/subject/i).fill(subject);
    await dialog.getByLabel(/description/i).fill("Feedback with an attached file.");

    // Attach a small file
    await dialog.locator('input[type="file"]').setInputFiles({
      name: "report.txt",
      mimeType: "text/plain",
      buffer: Buffer.from("Attachment content for E2E test"),
    });
    await expect(dialog.getByText("report.txt")).toBeVisible({ timeout: 3_000 });

    // Submit
    await dialog.getByRole("button", { name: "Send Feedback" }).click();
    await expect(volunteerPage.getByText(/thank you/i)).toBeVisible({ timeout: 8_000 });

    // Navigate to My Feedback and open the detail page
    await volunteerPage.goto("/my-feedback");
    await volunteerPage.getByText(subject).click();

    // The Download button for our attachment should be visible
    await expect(
      volunteerPage.getByRole("button", { name: "Download" })
    ).toBeVisible({ timeout: 8_000 });
  });
});

test.describe("Feedback — attachment Download buttons", () => {
  let feedbackId: string;
  let ownerToken: string; // the volunteer who submitted — needed for ownFeedback (scoped per-user)

  test.beforeAll(async ({ volunteerToken }) => {
    ownerToken = volunteerToken;
    feedbackId = await submitFeedback(volunteerToken, {
      subject: uniqueName("DownloadBtnFeedback"),
      text: "Testing download button visibility.",
    });
    await attachFileToFeedback(volunteerToken, feedbackId, "evidence.txt", "evidence content");
  });

  test.afterAll(async ({ adminToken }) => {
    if (feedbackId) {
      try { await deleteFeedback(adminToken, feedbackId); } catch { /* ignore */ }
    }
  });

  test("volunteer detail page shows Download button for attachments", async ({
    browser,
  }) => {
    // ownFeedback is scoped per-user, so we must visit the page as the volunteer
    // who actually submitted the feedback, not the volunteerPage fixture's user.
    const ctx = await browser.newContext();
    await ctx.addCookies([{
      name: "session", value: ownerToken,
      domain: "localhost", path: "/",
      httpOnly: true, secure: false, sameSite: "Lax",
    }]);
    const page = await ctx.newPage();
    await page.addInitScript(() => {
      localStorage.setItem("sessionActive", "1");
      localStorage.setItem("authRole", "VOLUNTEER");
      localStorage.setItem("authName", "Test Volunteer");
    });

    try {
      await page.goto(`/my-feedback/${feedbackId}`);
      await expect(
        page.getByRole("button", { name: "Download" })
      ).toBeVisible({ timeout: 8_000 });
    } finally {
      await ctx.close();
    }
  });

  test("admin detail page shows Download button for attachments", async ({
    adminPage,
  }) => {
    // feedbackById is not user-scoped — any admin can see any feedback item.
    await adminPage.goto(`/admin/feedback/${feedbackId}`);
    await expect(
      adminPage.getByRole("button", { name: "Download" })
    ).toBeVisible({ timeout: 8_000 });
  });
});
