import { NextResponse } from "next/server";
import { getWorkflowStatus } from "@/lib/companyd-client";

// Thin adapter only — a read-only projection, not a governed decision.
// See docs/architecture/application.md.

export async function GET(
  _request: Request,
  { params }: { params: Promise<{ id: string }> },
) {
  const { id } = await params;
  const status = await getWorkflowStatus(id);
  return NextResponse.json(status);
}
