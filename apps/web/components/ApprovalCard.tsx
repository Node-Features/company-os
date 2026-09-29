"use client";

import { useState } from "react";
import type { Approval } from "@/lib/companyd-client";
import { approvalSummary } from "@/lib/governance-copy";
import Disclosure from "./ui/Disclosure";
import { panelCard, btnPrimary, btnGhostDanger } from "@/lib/ui-styles";

// docs/ux/screen-inventory.md#approvals — one pending REQUIRE_APPROVAL item.
// Approve/Decline call the shared POST /v1/approvals/{id}/decide endpoint
// via the parent's onDecide — never writes to persistence itself
// (docs/architecture/ui-ux.md invariant 1).
export default function ApprovalCard({
  approval,
  onDecide,
}: {
  approval: Approval;
  onDecide: (approvalId: string, approve: boolean) => Promise<{ error?: string }>;
}) {
  const [busy, setBusy] = useState<"approve" | "decline" | null>(null);
  const [error, setError] = useState<string | null>(null);

  async function decide(approve: boolean) {
    setBusy(approve ? "approve" : "decline");
    setError(null);
    try {
      const result = await onDecide(approval.ApprovalID, approve);
      if (result.error) setError(result.error);
    } finally {
      setBusy(null);
    }
  }

  return (
    <div className="card-hover" style={panelCard}>
      <div style={{ display: "flex", alignItems: "flex-start", gap: 14 }}>
        <div
          style={{
            width: 36,
            height: 36,
            borderRadius: 10,
            background: "oklch(0.78 0.14 85 / 0.15)",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            flexShrink: 0,
          }}
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--warning)" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <path d="M12 9v4M12 17h.01" />
            <path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
          </svg>
        </div>
        <div style={{ flexGrow: 1 }}>
          <div style={{ fontSize: 14, fontWeight: 600 }}>{approvalSummary(approval.Action)}</div>
          <Disclosure
            detail={
              <>
                action={approval.Action} · resource={approval.ResourceType} {approval.ResourceID} · requested_by={approval.RequestingPrincipalID} · requested={approval.CreatedAt}
              </>
            }
          />
        </div>
        <div style={{ display: "flex", gap: 8, flexShrink: 0 }}>
          <button className="btn" style={btnGhostDanger} onClick={() => decide(false)} disabled={busy !== null}>
            {busy === "decline" ? "Declining…" : "Decline"}
          </button>
          <button className="btn" style={btnPrimary} onClick={() => decide(true)} disabled={busy !== null}>
            {busy === "approve" ? "Approving…" : "Approve"}
          </button>
        </div>
      </div>
      {error && <p className="ds-error" style={{ marginTop: 10 }}>{error}</p>}
    </div>
  );
}
