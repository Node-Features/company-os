import AppShell from "@/components/AppShell";
import PageIntro from "@/components/ui/PageIntro";
import Soon from "@/components/ui/Soon";

// docs/ux/screen-inventory.md#activity-log: BLOCKED — no read endpoint
// exists over governance_decisions yet (ROADMAP.md Phase 10 Slice 4).
export default function ActivityPage() {
  return (
    <AppShell active="activity" pageTitle="Activity log">
      <PageIntro title="Activity log" subtitle="Who changed what, when, and under what authority." />
      <Soon
        title="Activity log is on the way"
        detail="Every governed decision is already recorded in governance_decisions — but no read endpoint exposes it yet. This screen will show up once ROADMAP.md Phase 10 Slice 4 lands one."
      />
    </AppShell>
  );
}
