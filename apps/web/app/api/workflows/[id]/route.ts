import { NextResponse } from "next/server";
import { getWorkflowStatus } from "@/lib/companyd-client";
import { getAccessToken } from "@/lib/session";

// Thin adapter only — a read-only projection, not a governed decision.
// See docs/architecture/application.md.

export async function GET(
  _request: Request,
  { params }: { params: Promise<{ id: string }> },
) {
  const accessToken = await getAccessToken();
  if (!accessToken) {
    return NextResponse.json({ error: "unauthenticated" }, { status: 401 });
  }
  const { id } = await params;
  const status = await getWorkflowStatus(id, accessToken);
  return NextResponse.json(status);
}
