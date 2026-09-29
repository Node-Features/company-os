import { NextResponse } from "next/server";
import { listApprovals, CompanydError } from "@/lib/companyd-client";
import { getAccessToken } from "@/lib/session";

// Thin adapter only — no governed decisions happen here.
// See docs/architecture/application.md.
//
// Forwards companyd's real status/error rather than always answering 200 —
// a caller that only checks `res.ok` before treating the body as an array
// must actually see a non-2xx when companyd errored (this route previously
// always returned 200, so a companyd 401/503 arrived here as `{error}` and
// crashed callers expecting Approval[]).

export async function GET(request: Request) {
  const accessToken = await getAccessToken();
  if (!accessToken) {
    return NextResponse.json({ error: "unauthenticated" }, { status: 401 });
  }
  const { searchParams } = new URL(request.url);
  const statusParam = searchParams.get("status");
  const statuses = statusParam ? statusParam.split(",") : undefined;
  const limit = searchParams.get("limit") ? Number(searchParams.get("limit")) : undefined;
  const offset = searchParams.get("offset") ? Number(searchParams.get("offset")) : undefined;
  try {
    const approvals = await listApprovals(accessToken, statuses, limit, offset);
    return NextResponse.json(approvals);
  } catch (err) {
    if (err instanceof CompanydError) {
      return NextResponse.json({ error: err.message }, { status: err.status });
    }
    return NextResponse.json({ error: "companyd unreachable" }, { status: 502 });
  }
}
