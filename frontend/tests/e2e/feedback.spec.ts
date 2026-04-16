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
 */

import { test, expect } from "./helpers/fixtures";
import { submitFeedback, uniqueName } from "./helpers/api";

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

    await dialog.getByRole("button", { name: "Submit Feedback" }).click();

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
    await submitFeedback(volunteerToken, {
      subject,
      text: "Feedback body for E2E test",
    });

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
      `mutation Q($q: QuestionFeedbackInput!) { questionFeedback(question: $q) { success message } }`,
      {
        q: {
          id: feedbackId,
          emailText: "Can you provide more details?",
          note: "Asked for clarification",
        },
      },
      adminToken
    ) as { questionFeedback?: { success: boolean; message?: string } };

    if (!result.questionFeedback?.success) {
      throw new Error(
        `questionFeedback mutation failed: ${result.questionFeedback?.message ?? "unknown error"}`
      );
    }

    await adminPage.goto(`/admin/feedback/${feedbackId}`);
    // Scope to <span> so we match the status badge, not the hidden <option> in the dropdown
    await expect(
      adminPage.locator("span").filter({ hasText: "Question Sent" })
    ).toBeVisible({ timeout: 5_000 });
  });

  test("volunteer can see the question in My Feedback detail but not admin notes", async ({
    browser,
    adminToken,
  }) => {
    // First add an admin-only note to ensure it is hidden from volunteer
    await adminGql(
      `mutation U($f: UpdateFeedbackInput!) { updateFeedback(feedback: $f) { success message } }`,
      {
        f: {
          id: feedbackId,
          status: "QUESTION_SENT",
          note: "INTERNAL: this is an admin-only note",
        },
      },
      adminToken
    );

    // Create a page logged in as the volunteer who actually submitted the feedback.
    // We must use their token — ownFeedback is scoped per-user, so a different
    // volunteer's session would not find feedbackId.
    const ctx = await browser.newContext();
    const page = await ctx.newPage();
    await page.addInitScript(({ token }) => {
      localStorage.setItem("authToken", token);
      localStorage.setItem("authRole", "VOLUNTEER");
      localStorage.setItem("authName", "Test Volunteer");
    }, { token: workflowVolunteerToken });

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
      `mutation R($r: ResolveFeedbackInput!) { resolveFeedback(resolution: $r) { success message } }`,
      {
        r: {
          id: feedbackId,
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
    await dialog.getByRole("button", { name: "Submit Feedback" }).click();

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
