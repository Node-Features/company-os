// Typed client for calling companyd. Every exported function here
// should map to exactly one Application use case
// (docs/architecture/application.md) and translate its normalized
// outcome (ACCEPTED/REJECTED/DENIED/APPROVAL_REQUIRED/CONFLICT/
// UNAVAILABLE/INDETERMINATE/INVALID) without reinterpreting it.

const COMPANYD_URL = process.env.COMPANYD_URL ?? "http://localhost:8080";

export async function companydHealth(): Promise<boolean> {
  const res = await fetch(`${COMPANYD_URL}/health`);
  return res.ok;
}
