import { NextResponse } from "next/server";
import { getConstraintStatus, CompanydError } from "@/lib/companyd-client";
import { getAccessToken } from "@/lib/session";

// Thin adapter only — no governed decisions happen here.
// See docs/architecture/application.md. Forwards companyd's real
// status/error rather than always answering 200 (see approvals/route.ts).

export async function GET() {
  const accessToken = await getAccessToken();
  if (!accessToken) {
    return NextResponse.json({ error: "unauthenticated" }, { status: 401 });
  }
  try {
    const status = await getConstraintStatus(accessToken);
    return NextResponse.json(status);
  } catch (err) {
    if (err instanceof CompanydError) {
      return NextResponse.json({ error: err.message }, { status: err.status });
    }
    return NextResponse.json({ error: "companyd unreachable" }, { status: 502 });
  }
}
