import { NextResponse } from "next/server";
import { getWorkflowStatus, CompanydError } from "@/lib/companyd-client";
import { getAccessToken } from "@/lib/session";

// Thin adapter only — a read-only projection, not a governed decision.
// See docs/architecture/application.md. Forwards companyd's real
// status/error rather than always answering 200 (see approvals/route.ts).

export async function GET(
  _request: Request,
  { params }: { params: Promise<{ id: string }> },
) {
  const accessToken = await getAccessToken();
  if (!accessToken) {
    return NextResponse.json({ error: "unauthenticated" }, { status: 401 });
  }
  const { id } = await params;
  try {
    const status = await getWorkflowStatus(id, accessToken);
    return NextResponse.json(status);
  } catch (err) {
    if (err instanceof CompanydError) {
      return NextResponse.json({ error: err.message }, { status: err.status });
    }
    return NextResponse.json({ error: "companyd unreachable" }, { status: 502 });
  }
}
