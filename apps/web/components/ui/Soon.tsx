// Honest "not built yet" state, matching the mockup's own pattern for
// screens with no backing use case at all (People) or no list endpoint yet
// (Departments, Activity log) — never fabricated sample content
// (docs/architecture/ui-ux.md invariant 7).
export default function Soon({ title, detail }: { title: string; detail: string }) {
  return (
    <div className="rise" style={{ display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center", padding: "80px 20px", textAlign: "center" }}>
      <div
        style={{
          width: 52,
          height: 52,
          borderRadius: 14,
          background: "var(--muted)",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          marginBottom: 16,
        }}
      >
        <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="var(--muted-foreground)" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
          <circle cx="12" cy="12" r="10" />
          <path d="M12 6v6l4 2" />
        </svg>
      </div>
      <div style={{ fontSize: 16, fontWeight: 600 }}>{title}</div>
      <div style={{ fontSize: 13, color: "var(--muted-foreground)", marginTop: 6, maxWidth: 360 }}>{detail}</div>
    </div>
  );
}
