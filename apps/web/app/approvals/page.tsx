import Link from "next/link";
import ApprovalInbox from "@/components/ApprovalInbox";

export default function ApprovalsPage() {
  return (
    <main>
      <div className="actions" style={{ justifyContent: "space-between", marginBottom: "0.5rem" }}>
        <h1 className="title">Approvals</h1>
        <Link className="btn" href="/">
          Workflow
        </Link>
      </div>
      <p className="subtitle">
        Every pending REQUIRE_APPROVAL across Workflow cancel, Objective proposals, and Knowledge
        approvals — see docs/architecture/ui-ux.md.
      </p>
      <div className="panel">
        <ApprovalInbox />
      </div>
    </main>
  );
}
