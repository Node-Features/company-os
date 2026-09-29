import AppShell from "@/components/AppShell";
import PageIntro from "@/components/ui/PageIntro";
import Soon from "@/components/ui/Soon";

export default function PeoplePage() {
  return (
    <AppShell active="people" pageTitle="People">
      <PageIntro title="People" subtitle="Every human and Agent Principal working across the company." />
      <Soon
        title="People is on the way"
        detail="This depends on DepartmentMembership, which doesn't exist yet — this screen will show up once ROADMAP.md Phase 11 lands it."
      />
    </AppShell>
  );
}
