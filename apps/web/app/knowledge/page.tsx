import AppShell from "@/components/AppShell";
import KnowledgeLibrary from "@/components/KnowledgeLibrary";
import PageIntro from "@/components/ui/PageIntro";

export default function KnowledgePage() {
  return (
    <AppShell active="knowledge" pageTitle="Knowledge">
      <PageIntro title="Knowledge" subtitle="What the company has established as fact, and what it's built on." />
      <KnowledgeLibrary />
    </AppShell>
  );
}
