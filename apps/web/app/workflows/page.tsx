import AppShell from "@/components/AppShell";
import WorkflowTrigger from "@/components/WorkflowTrigger";
import PageIntro from "@/components/ui/PageIntro";

// The mockup's Workflows screen shows a table of every workflow across
// every department — but companyd has no "list all workflows" endpoint
// yet (only create/start/cancel/get-one-by-id). Rather than fabricate a
// table, this keeps the real Phase 1 create→start→watch flow
// (WorkflowTrigger.tsx) as the honest, fully-functional slice of this
// screen (docs/architecture/ui-ux.md invariant 7).
export default function WorkflowsPage() {
  return (
    <AppShell active="workflows" pageTitle="Workflows">
      <PageIntro title="Workflows" subtitle="Create, start, and watch one workflow end to end." />
      <div className="rise" style={{ background: "var(--card)", border: "1px solid var(--border)", borderRadius: "var(--radius)", padding: "18px 20px" }}>
        <WorkflowTrigger />
      </div>
      <p style={{ fontSize: 12, color: "var(--muted-foreground)", marginTop: 14 }}>
        A company-wide table of every running/completed workflow needs a list endpoint that doesn&apos;t
        exist in companyd yet — this screen tracks one workflow at a time until that lands.
      </p>
    </AppShell>
  );
}
