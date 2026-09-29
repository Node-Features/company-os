import AppShell from "@/components/AppShell";
import PageIntro from "@/components/ui/PageIntro";
import Soon from "@/components/ui/Soon";

// The Insights graph is presentation over Research + Decisions reads,
// walked and assembled client-side (docs/architecture/ui-ux.md). Since
// Research has no "list open questions" endpoint yet (see /research), the
// graph has no real chain to assemble from — and even once one exists,
// ui-ux.md's own open question says Decision→Action→Outcome→Evaluation
// isn't modeled anywhere in the domain yet, so the chain can only ever
// render as far as Decision without fabricating edges the backend doesn't
// establish.
export default function GraphPage() {
  return (
    <AppShell active="graph" pageTitle="Insights graph">
      <PageIntro
        title="Insights graph"
        subtitle="Trace how an observation became a decision, and whether that decision produced a measurable outcome."
      />
      <Soon
        title="Nothing to trace yet"
        detail="This graph assembles from a Research question's chain through to its Decision — but there's no endpoint yet that lists open questions to start from, and the Decision→Action→Outcome→Evaluation half of the chain isn't modeled in the domain yet either (see docs/architecture/ui-ux.md's open question)."
      />
    </AppShell>
  );
}
