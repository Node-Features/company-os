import Link from "next/link";
import WorkflowTrigger from "@/components/WorkflowTrigger";
import { logout } from "./login/actions";

export default function HomePage() {
  return (
    <main>
      <div className="actions" style={{ justifyContent: "space-between", marginBottom: "0.5rem" }}>
        <h1 className="title">CompanyOS</h1>
        <div className="actions">
          <Link className="btn" href="/approvals">
            Approvals
          </Link>
          <form action={logout}>
            <button className="btn" type="submit">
              Sign out
            </button>
          </form>
        </div>
      </div>
      <p className="subtitle">UI and Application adapter — see AGENTS.md and docs/INDEX.md.</p>
      <div className="panel">
        <WorkflowTrigger />
      </div>
    </main>
  );
}
