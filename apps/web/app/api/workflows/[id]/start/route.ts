import { NextResponse } from "next/server";
import { startWorkflow } from "@/lib/companyd-client";

// Thin adapter only — no governed decisions happen here.
// See docs/architecture/application.md.

export async function POST(
  request: Request,
  { params }: { params: Promise<{ id: string }> },
) {
  const { id } = await params;
  const { expectedVersion } = await request.json();
  const result = await startWorkflow(id, expectedVersion);
  return NextResponse.json(result);
}
