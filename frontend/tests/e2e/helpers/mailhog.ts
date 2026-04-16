/**
 * Mailhog API helpers.
 *
 * Mailhog runs at http://localhost:8025 (web UI + API).
 * Its REST API lets us list and read messages sent during tests.
 */

const MAILHOG_URL =
  process.env.MAILHOG_URL || "http://localhost:8025";

/** A single message from the Mailhog API */
export interface MailhogMessage {
  ID: string;
  From: { Mailbox: string; Domain: string };
  To: Array<{ Mailbox: string; Domain: string }>;
  Content: { Headers: Record<string, string[]>; Body: string };
  Created: string;
}

/** Fetch all messages currently in Mailhog. */
async function fetchMessages(): Promise<MailhogMessage[]> {
  const res = await fetch(`${MAILHOG_URL}/api/v2/messages?limit=50`);
  if (!res.ok) throw new Error(`Mailhog API error: ${res.status}`);
  const data = await res.json();
  return data.items ?? [];
}

/** Delete all messages in Mailhog (call before each test that checks email). */
export async function clearMailbox(): Promise<void> {
  await fetch(`${MAILHOG_URL}/api/v1/messages`, { method: "DELETE" });
}

/**
 * Wait up to `timeoutMs` for an email addressed to `toEmail` to arrive,
 * then return it. Polls every 500 ms.
 */
export async function waitForEmail(
  toEmail: string,
  timeoutMs = 10_000
): Promise<MailhogMessage> {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const msgs = await fetchMessages();
    const match = msgs.find((m) =>
      m.To.some(
        (t) =>
          `${t.Mailbox}@${t.Domain}`.toLowerCase() === toEmail.toLowerCase()
      )
    );
    if (match) return match;
    await new Promise((r) => setTimeout(r, 500));
  }
  throw new Error(
    `Timed out waiting for email to ${toEmail} after ${timeoutMs}ms`
  );
}

/**
 * Extract the magic-link URL from an email body.
 * The backend embeds a link like http://localhost:3000/auth/magic-link?token=<token>
 */
export function extractMagicLink(msg: MailhogMessage): string {
  const body = msg.Content.Body;
  const match = body.match(/https?:\/\/\S*\/auth\/magic-link\?token=[^\s"<>]+/);
  if (!match) {
    throw new Error(
      `Could not find magic-link URL in email body:\n${body.slice(0, 500)}`
    );
  }
  // URL-decode any encoded ampersands from HTML email bodies
  return match[0].replace(/&amp;/g, "&");
}
