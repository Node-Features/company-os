import AppShell from "@/components/AppShell";
import BudgetOverview from "@/components/BudgetOverview";
import PageIntro from "@/components/ui/PageIntro";

export default function BudgetPage() {
  return (
    <AppShell active="budget" pageTitle="Budget">
      <PageIntro title="Budget" subtitle="What's allocated, what's spent, and what's still available." />
      <BudgetOverview />
    </AppShell>
  );
}
