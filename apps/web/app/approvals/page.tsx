import AppShell from "@/components/AppShell";
import ApprovalsList from "@/components/ApprovalsList";
import PageIntro from "@/components/ui/PageIntro";

// ROADMAP.md Phase 10 Slice 1 UI — the Approval inbox, real data via
// GET /v1/approvals / POST /v1/approvals/{id}/decide.
export default function ApprovalsPage() {
  return (
    <AppShell active="approvals" pageTitle="Approvals">
      <PageIntro title="Approvals" subtitle="Actions across the company that need a human decision before they proceed." />
      <ApprovalsList />
    </AppShell>
  );
}
