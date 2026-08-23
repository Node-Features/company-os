import { NextResponse } from "next/server";
import { listPendingApprovals } from "@/lib/companyd-client";
import { getAccessToken } from "@/lib/session";

// Thin adapter only — no governed decisions happen here.
// See docs/architecture/application.md.

export async function GET() {
  const accessToken = await getAccessToken();
  if (!accessToken) {
    return NextResponse.json({ error: "unauthenticated" }, { status: 401 });
  }
  const approvals = await listPendingApprovals(accessToken);
  return NextResponse.json(approvals);
}
