import { NextResponse } from "next/server";
import { createWorkflow } from "@/lib/companyd-client";

// Thin adapter only — no governed decisions happen here.
// See docs/architecture/application.md.

export async function POST() {
  const result = await createWorkflow();
  return NextResponse.json(result);
}
