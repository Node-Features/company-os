import AppShell from "@/components/AppShell";
import DecisionsList from "@/components/DecisionsList";
import PageIntro from "@/components/ui/PageIntro";

export default function DecisionsPage() {
  return (
    <AppShell active="decisions" pageTitle="Decisions">
      <PageIntro
        title="Decisions"
        subtitle="Every objective proposed from research, monitoring, or budget evidence — and what came of it."
      />
      <DecisionsList />
    </AppShell>
  );
}
