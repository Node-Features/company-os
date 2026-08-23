"use client";

import { useEffect, useRef, useState } from "react";
import type { PendingApproval } from "@/lib/companyd-client";

// No realtime channel covers Approval events yet (docs/architecture/ui-ux.md's
// own open question) — this poll is the only update mechanism, not a
// fallback for a realtime one like WorkflowTrigger's.
const POLL_INTERVAL_MS = 10000;

function Loader({ label }: { label: string }) {
  return (
    <span className="loader">
      <span className="loader-ring" />
      <span className="loader-text">{label}</span>
    </span>
  );
}

// The Approval inbox (ROADMAP.md Phase 10 Slice 1, docs/architecture/ui-ux.md):
// every pending REQUIRE_APPROVAL across Workflow cancel, Objective
// proposals, and Knowledge approvals in one list — PendingCommand/Approval
// are already generic across all three, so one endpoint and one screen
// cover all of them. Reuses the existing resolveApproval decision flow
// (POST /api/approvals/{id}/decide, unchanged since Phase 3 Slice 2).
export default function ApprovalInbox() {
  const [approvals, setApprovals] = useState<PendingApproval[] | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  async function refresh() {
    const list: PendingApproval[] = await fetch("/api/approvals").then((r) => r.json());
    setApprovals(list);
  }

  useEffect(() => {
    refresh();
    pollRef.current = setInterval(refresh, POLL_INTERVAL_MS);
    return () => {
      if (pollRef.current) clearInterval(pollRef.current);
    };
  }, []);

  async function handleDecide(approvalId: string, approve: boolean) {
    setBusyId(approvalId);
    setError(null);
    try {
      const result = await fetch(`/api/approvals/${approvalId}/decide`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ approve }),
      }).then((r) => r.json());
      if (result.outcome !== "ACCEPTED" && result.outcome !== "REJECTED") {
        setError(`decision failed: ${result.outcome} ${result.reasons?.join(", ") ?? ""}`);
      }
      await refresh();
    } finally {
      setBusyId(null);
    }
  }

  if (approvals === null) {
    return (
      <p className="workflow-id">
        <Loader label="Loading approvals" />
      </p>
    );
  }

  return (
    <div>
      {error && <p className="error-text">{error}</p>}

      {approvals.length === 0 && <p className="unit-empty">No pending approvals.</p>}

      {approvals.map((a) => (
        <div key={a.approvalId} className="status-block">
          <div className="status-row">
            <span className="status-label">Action</span>
            <span className="state-badge state-READY">{a.action}</span>
          </div>
          <div className="status-row">
            <span className="status-label">Resource</span>
            <span>
              {a.resourceType} <code>{a.resourceId}</code>
            </span>
          </div>
          <div className="status-row">
            <span className="status-label">Requested</span>
            <span>{a.createdAt}</span>
          </div>
          <div className="actions" style={{ marginTop: "0.5rem" }}>
            <button
              className="btn btn-accent"
              onClick={() => handleDecide(a.approvalId, true)}
              disabled={busyId === a.approvalId}
            >
              Approve
            </button>
            <button
              className="btn"
              onClick={() => handleDecide(a.approvalId, false)}
              disabled={busyId === a.approvalId}
            >
              Reject
            </button>
            {busyId === a.approvalId && <Loader label="Deciding" />}
          </div>
        </div>
      ))}
    </div>
  );
}
