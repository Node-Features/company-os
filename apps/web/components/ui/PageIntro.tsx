export default function PageIntro({ title, subtitle }: { title: string; subtitle: string }) {
  return (
    <div className="rise" style={{ marginBottom: 18 }}>
      <div style={{ fontSize: 22, fontWeight: 700, letterSpacing: "-0.01em" }}>{title}</div>
      <div style={{ fontSize: 14, color: "var(--muted-foreground)", marginTop: 4 }}>{subtitle}</div>
    </div>
  );
}
