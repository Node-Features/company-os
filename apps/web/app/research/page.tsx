import AppShell from "@/components/AppShell";
import PageIntro from "@/components/ui/PageIntro";
import Soon from "@/components/ui/Soon";

// GET /v1/research/questions/{id} needs a specific question id — there is
// no "list all open questions" endpoint yet, so this screen can't render
// a real timeline without one (docs/architecture/ui-ux.md invariant 7:
// never fabricate the mockup's illustrative timeline).
export default function ResearchPage() {
  return (
    <AppShell active="research" pageTitle="Research">
      <PageIntro title="Research" subtitle="How a signal becomes evidence, becomes a discovery, becomes something worth acting on." />
      <Soon
        title="Research timeline needs a question to open"
        detail="companyd can show one ResearchQuestion's full timeline (GET /v1/research/questions/{id}), but there's no endpoint yet that lists every open question — so this view has nothing to start from until one exists."
      />
    </AppShell>
  );
}
