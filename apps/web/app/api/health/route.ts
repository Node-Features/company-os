import { NextResponse } from "next/server";

// Thin adapter only — no governed decisions happen here.
// See docs/architecture/application.md.

type CheckResult = { status: "ok" | "error"; detail?: string };

async function checkSupabase(): Promise<CheckResult> {
  const url = process.env.NEXT_PUBLIC_SUPABASE_URL;
  const key = process.env.NEXT_PUBLIC_SUPABASE_PUBLISHABLE_KEY;
  if (!url || !key) {
    return { status: "error", detail: "missing NEXT_PUBLIC_SUPABASE_URL or _PUBLISHABLE_KEY" };
  }

  try {
    // Auth's own health endpoint — proves the URL + publishable key can
    // reach the project without depending on any table existing yet.
    // (PostgREST's /rest/v1/ root rejects publishable keys outright:
    // "Secret API key required" — it's not a connectivity signal here.)
    const res = await fetch(`${url}/auth/v1/health`, {
      headers: { apikey: key },
      signal: AbortSignal.timeout(3000),
    });
    return res.ok ? { status: "ok" } : { status: "error", detail: `HTTP ${res.status}` };
  } catch (err) {
    return { status: "error", detail: err instanceof Error ? err.message : "unreachable" };
  }
}

async function checkCompanyd(): Promise<CheckResult & { db?: string }> {
  const url = process.env.COMPANYD_URL;
  if (!url) {
    return { status: "error", detail: "missing COMPANYD_URL" };
  }

  try {
    const res = await fetch(`${url}/health`, { signal: AbortSignal.timeout(3000) });
    const body = await res.json().catch(() => null);
    return res.ok
      ? { status: "ok", db: body?.db }
      : { status: "error", detail: `HTTP ${res.status}` };
  } catch (err) {
    return { status: "error", detail: err instanceof Error ? err.message : "unreachable" };
  }
}

export async function GET() {
  const [supabase, companyd] = await Promise.all([checkSupabase(), checkCompanyd()]);

  const status = supabase.status === "ok" && companyd.status === "ok" ? "ok" : "degraded";

  return NextResponse.json(
    { status, web: "ok", supabase, companyd },
    { status: status === "ok" ? 200 : 503 },
  );
}
