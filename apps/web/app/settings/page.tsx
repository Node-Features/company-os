import AppShell from "@/components/AppShell";
import PageIntro from "@/components/ui/PageIntro";
import { panelCard, badgeStyle } from "@/lib/ui-styles";

// docs/architecture/ui-ux.md's Settings row: "a grouped IA shell can exist
// and honestly say so per-item; do not build working forms ahead of the
// backend that would back them." Group/item names come from that same
// screen-inventory row, not invented here.
const groups = [
  {
    label: "ORGANIZATION",
    items: [
      { name: "Organization profile", desc: "Name, domain, billing contact" },
      { name: "Roles & permissions", desc: "Who can approve, propose, or configure" },
    ],
  },
  {
    label: "RESEARCH & OPERATIONS",
    items: [
      { name: "Research configuration", desc: "Signal sources, review requirements" },
      { name: "M&E configuration", desc: "KPI thresholds, sample-size minimums" },
      { name: "Budget configuration", desc: "Currency, constraint enforcement level" },
      { name: "Workflow configuration", desc: "Retry policy, timeouts, capability defaults" },
    ],
  },
  {
    label: "SYSTEM",
    items: [
      { name: "AI & automation", desc: "Provider priority, autonomy defaults" },
      { name: "Notifications", desc: "Approval alerts, digest frequency" },
      { name: "Integrations", desc: "Connected data sources and tools" },
      { name: "Security & audit", desc: "Session policy, audit retention" },
    ],
  },
];

export default function SettingsPage() {
  return (
    <AppShell active="settings" pageTitle="Settings">
      <PageIntro title="Settings" subtitle="Organization, access, and how each department and safeguard is configured." />
      {groups.map((g) => (
        <div key={g.label} className="rise" style={{ marginBottom: 20 }}>
          <div style={{ fontSize: 12, fontWeight: 600, color: "var(--muted-foreground)", letterSpacing: "0.04em", marginBottom: 10 }}>
            {g.label}
          </div>
          <div style={{ display: "grid", gridTemplateColumns: "repeat(2,1fr)", gap: 10 }}>
            {g.items.map((it) => (
              <div key={it.name} className="card-hover" style={{ ...panelCard, display: "flex", alignItems: "center", gap: 12 }}>
                <div
                  style={{
                    width: 32,
                    height: 32,
                    borderRadius: 8,
                    background: "var(--muted)",
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "center",
                    flexShrink: 0,
                    color: "var(--muted-foreground)",
                  }}
                >
                  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
                    <circle cx="12" cy="12" r="9" />
                  </svg>
                </div>
                <div style={{ flexGrow: 1 }}>
                  <div style={{ fontSize: 13, fontWeight: 500 }}>{it.name}</div>
                  <div style={{ fontSize: 11.5, color: "var(--muted-foreground)", marginTop: 1 }}>{it.desc}</div>
                </div>
                <span style={badgeStyle("soon")}>SOON</span>
              </div>
            ))}
          </div>
        </div>
      ))}
      <p style={{ fontSize: 12, color: "var(--muted-foreground)" }}>
        None of this is persisted data yet — policy is hardcoded Go, and there is no config table of
        any kind. This is a navigation shell only, not working forms.
      </p>
    </AppShell>
  );
}
