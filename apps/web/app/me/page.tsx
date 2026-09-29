import AppShell from "@/components/AppShell";
import PageIntro from "@/components/ui/PageIntro";
import Soon from "@/components/ui/Soon";

// GET /v1/me/subjects/{id}/performance-profile needs a specific subject
// id — there is no "list all subjects" endpoint yet.
export default function MonitoringEvaluationPage() {
  return (
    <AppShell active="me" pageTitle="Monitoring & Evaluation">
      <PageIntro title="Monitoring & Evaluation" subtitle="How well each capability is performing, and whether it's improving." />
      <Soon
        title="No capability selected"
        detail="companyd can show one capability's PerformanceProfile (GET /v1/me/subjects/{id}/performance-profile), but there's no endpoint yet that lists every tracked capability — so this view has nothing to start from until one exists."
      />
    </AppShell>
  );
}
