import { NextResponse } from "next/server";
import { listKnowledgeItems, CompanydError } from "@/lib/companyd-client";
import { getAccessToken } from "@/lib/session";

// Thin adapter only — no governed decisions happen here.
// See docs/architecture/application.md. Forwards companyd's real
// status/error rather than always answering 200 (see approvals/route.ts).

export async function GET(request: Request) {
  const accessToken = await getAccessToken();
  if (!accessToken) {
    return NextResponse.json({ error: "unauthenticated" }, { status: 401 });
  }
  const { searchParams } = new URL(request.url);
  const statusParam = searchParams.get("statuses");
  const statuses = statusParam ? statusParam.split(",") : undefined;
  const purpose = searchParams.get("purpose") ?? undefined;
  try {
    const items = await listKnowledgeItems(accessToken, statuses, purpose);
    return NextResponse.json(items);
  } catch (err) {
    if (err instanceof CompanydError) {
      return NextResponse.json({ error: err.message }, { status: err.status });
    }
    return NextResponse.json({ error: "companyd unreachable" }, { status: 502 });
  }
}
