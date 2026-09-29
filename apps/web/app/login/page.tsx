import { login } from "./actions";

// Matches the dashboard's design-system (docs/architecture/ui-ux.md's
// re-approved clean, dark direction) — this page previously kept Phase 1's
// original cyberpunk look (globals.css's `.title`/`.panel`/`.btn` classes)
// since nothing had touched it yet; now brought in line with the rest of
// the product UI instead of the other way around.
export default async function LoginPage({
  searchParams,
}: {
  searchParams: Promise<{ error?: string }>;
}) {
  const { error } = await searchParams;

  return (
    <div
      className="ds-root"
      style={{
        minHeight: "100vh",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        padding: 24,
        background:
          "radial-gradient(560px 340px at 0% 0%, var(--magenta-glow), transparent 60%), " +
          "radial-gradient(640px 380px at 100% 0%, var(--primary-glow), transparent 55%), " +
          "linear-gradient(oklch(1 0 0 / 0.025) 1px, transparent 1px) 0 0 / 40px 40px, " +
          "linear-gradient(90deg, oklch(1 0 0 / 0.025) 1px, transparent 1px) 0 0 / 40px 40px, " +
          "var(--background)",
      }}
    >
      <div style={{ width: "100%", maxWidth: 380 }}>
        <div className="brand-glow" style={{ fontSize: 20, fontWeight: 700, letterSpacing: "-0.01em", color: "var(--primary)", marginBottom: 6 }}>
          CompanyOS
        </div>
        <p style={{ fontSize: 14, color: "var(--muted-foreground)", margin: "0 0 24px" }}>Sign in to continue.</p>

        <div style={{ background: "var(--card)", border: "1px solid var(--border)", borderRadius: "var(--radius)", padding: "24px 24px 20px" }}>
          <form action={login} style={{ display: "flex", flexDirection: "column", gap: 14 }}>
            <div>
              <label
                htmlFor="email"
                style={{ display: "block", fontSize: 11, fontWeight: 600, letterSpacing: "0.05em", color: "var(--muted-foreground)", textTransform: "uppercase", marginBottom: 6 }}
              >
                Email
              </label>
              <input
                id="email"
                name="email"
                type="email"
                required
                style={{
                  width: "100%",
                  background: "var(--muted)",
                  border: "1px solid var(--border)",
                  borderRadius: 8,
                  color: "var(--foreground)",
                  fontSize: 13.5,
                  padding: "9px 12px",
                }}
              />
            </div>

            <div>
              <label
                htmlFor="password"
                style={{ display: "block", fontSize: 11, fontWeight: 600, letterSpacing: "0.05em", color: "var(--muted-foreground)", textTransform: "uppercase", marginBottom: 6 }}
              >
                Password
              </label>
              <input
                id="password"
                name="password"
                type="password"
                required
                style={{
                  width: "100%",
                  background: "var(--muted)",
                  border: "1px solid var(--border)",
                  borderRadius: 8,
                  color: "var(--foreground)",
                  fontSize: 13.5,
                  padding: "9px 12px",
                }}
              />
            </div>

            <button
              type="submit"
              className="btn"
              style={{
                marginTop: 6,
                background: "var(--primary)",
                color: "var(--primary-foreground)",
                border: "none",
                fontSize: 13,
                fontWeight: 600,
                padding: "10px 16px",
                borderRadius: 8,
                boxShadow: "0 0 18px var(--primary-glow)",
              }}
            >
              Sign in
            </button>
          </form>
          {error && (
            <p style={{ marginTop: 14, fontSize: 12.5, color: "var(--destructive)" }}>{error}</p>
          )}
        </div>
      </div>
    </div>
  );
}
