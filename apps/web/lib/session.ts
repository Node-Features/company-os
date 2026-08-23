import { cookies } from "next/headers";
import { createClient } from "@/lib/supabase/server";

// The current request's Supabase session access token, or null if there
// is none. Every companyd-client.ts call needs this — companyd requires
// and verifies it on every /v1/workflows*/approvals* route
// (ROADMAP.md Phase 3 Slice 4).
export async function getAccessToken(): Promise<string | null> {
  const supabase = createClient(await cookies());
  const {
    data: { session },
  } = await supabase.auth.getSession();
  return session?.access_token ?? null;
}
