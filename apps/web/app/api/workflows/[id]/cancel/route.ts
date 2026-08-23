import { NextResponse } from "next/server";
import { cancelWorkflow } from "@/lib/companyd-client";
import { getAccessToken } from "@/lib/session";

// Thin adapter only — no governed decisions happen here.
// See docs/architecture/application.md.

export async function POST(
  request: Request,
  { params }: { params: Promise<{ id: string }> },
) {
  const accessToken = await getAccessToken();
  if (!accessToken) {
    return NextResponse.json({ error: "unauthenticated" }, { status: 401 });
  }
  const { id } = await params;
  const { expectedVersion } = await request.json();
  const result = await cancelWorkflow(id, expectedVersion, accessToken);
  return NextResponse.json(result);
}
