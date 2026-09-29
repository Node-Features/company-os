"use client";

import { useEffect, useState } from "react";
import type { Approval } from "@/lib/companyd-client";
import ApprovalCard from "./ApprovalCard";
import Empty from "./ui/Empty";

// docs/ux/ux-states.md: skeleton loading, a named empty state, and an
// inline/retryable error.
export default function ApprovalsList() {
  const [approvals, setApprovals] = useState<Approval[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  async function load() {
    setError(null);
    try {
      const res = await fetch("/api/approvals", { cache: "no-store" });
      if (!res.ok) {
        setError("Could not load approvals — the request failed.");
        return;
      }
      setApprovals(await res.json());
    } catch {
      setError("Could not load approvals — the request failed.");
    }
  }

  useEffect(() => {
    // load() only calls setState after its own await; the linter's static
    // check can't see past the call, so this is not the synchronous-
    // setState-in-effect case the rule guards against.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    load();
  }, []);

  async function handleDecide(approvalId: string, approve: boolean): Promise<{ error?: string }> {
    const res = await fetch(`/api/approvals/${approvalId}/decide`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ approve }),
    });
    const result = await res.json();
    if (result.outcome !== "ACCEPTED" && result.outcome !== "REJECTED") {
      return { error: `${result.outcome}: ${result.reasons?.join(", ") ?? "decision failed"}` };
    }
    setApprovals((prev) => prev?.filter((a) => a.ApprovalID !== approvalId) ?? prev);
    return {};
  }

  if (error) {
    return (
      <div className="ds-error">
        <span>{error}</span>
        <button style={{ background: "none", border: "none", color: "var(--primary)", fontSize: 12 }} onClick={load}>
          Retry
        </button>
      </div>
    );
  }

  if (approvals === null) {
    return (
      <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
        {[0, 1, 2].map((i) => (
          <div key={i} className="ds-skeleton" style={{ height: 92 }} />
        ))}
      </div>
    );
  }

  if (approvals.length === 0) {
    return <Empty text="Nothing needs your approval right now." />;
  }

  return (
    <div className="rise" style={{ display: "flex", flexDirection: "column", gap: 12 }}>
      {approvals.map((approval) => (
        <ApprovalCard key={approval.ApprovalID} approval={approval} onDecide={handleDecide} />
      ))}
    </div>
  );
}
