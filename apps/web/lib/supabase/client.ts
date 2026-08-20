import { createBrowserClient } from "@supabase/ssr";

// Browser-side Supabase client. Reads and Auth only — see the note in
// lib/supabase/server.ts. Authoritative writes go through
// lib/companyd-client.ts, never this client.

const supabaseUrl = process.env.NEXT_PUBLIC_SUPABASE_URL;
const supabaseKey = process.env.NEXT_PUBLIC_SUPABASE_PUBLISHABLE_KEY;

export const createClient = () =>
  createBrowserClient(
    supabaseUrl!,
    supabaseKey!,
  );
