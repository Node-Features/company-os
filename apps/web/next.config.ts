import type { NextConfig } from "next";
import path from "path";

const nextConfig: NextConfig = {
  // The repo-root package.json (Supabase CLI tooling) plus this app's own
  // package.json look like a monorepo to Turbopack's root inference. Pin it
  // explicitly so it doesn't guess wrong and rescan the whole repo.
  turbopack: {
    root: path.resolve(__dirname),
  },
};

export default nextConfig;
