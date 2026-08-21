import WorkflowTrigger from "@/components/WorkflowTrigger";

export default function HomePage() {
  return (
    <main>
      <h1 className="title">CompanyOS</h1>
      <p className="subtitle">UI and Application adapter — see AGENTS.md and docs/INDEX.md.</p>
      <div className="panel">
        <WorkflowTrigger />
      </div>
    </main>
  );
}
