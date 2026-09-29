import AppShell from "@/components/AppShell";
import PageIntro from "@/components/ui/PageIntro";
import Soon from "@/components/ui/Soon";

// docs/domain/department.md is APPROVED with zero implementation — no
// DepartmentRegistry/DepartmentMembership infrastructure exists yet
// (ROADMAP.md Phase 11).
export default function DepartmentsPage() {
  return (
    <AppShell active="departments" pageTitle="Departments">
      <PageIntro title="Departments" subtitle="How your company's work is organized." />
      <Soon
        title="Departments is on the way"
        detail="department.md is approved but nothing backs it yet — no DepartmentRegistry or DepartmentMembership exists. This screen will show real departments once ROADMAP.md Phase 11 lands them."
      />
    </AppShell>
  );
}
