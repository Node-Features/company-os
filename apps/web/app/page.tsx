import AppShell from "@/components/AppShell";
import PageIntro from "@/components/ui/PageIntro";
import Empty from "@/components/ui/Empty";
import { getAccessToken } from "@/lib/session";
import { listApprovals, listKnowledgeItems, getConstraintStatus } from "@/lib/companyd-client";
import { statCard, panelCard, linkBtn } from "@/lib/ui-styles";
import { approvalSummary } from "@/lib/governance-copy";
import Link from "next/link";

// Overview — a client-side (here: server-side) rollup of existing reads,
// per docs/architecture/ui-ux.md's screen inventory. Each panel loads
// independently and fails independently (docs/ux/ux-states.md's
// per-panel rule) rather than one slow read blocking the whole screen.
// No "active research"/"achievements" panel — no endpoint backs either
// concept yet, and the mockup's illustrative numbers for them are not
// reproduced here (invariant 7).
export default async function HomePage() {
  const accessToken = await getAccessToken();

  let approvals: Awaited<ReturnType<typeof listApprovals>> | null = null;
  let approvalsError = false;
  let knowledgeCount: number | null = null;
  let knowledgeError = false;
  let budgetPct: number | null = null;
  let budgetOutcome: string | null = null;
  let budgetError = false;

  if (accessToken) {
    try {
      approvals = await listApprovals(accessToken);
    } catch {
      approvalsError = true;
    }
    try {
      const items = await listKnowledgeItems(accessToken);
      knowledgeCount = items.length;
    } catch {
      knowledgeError = true;
    }
    try {
      const status = await getConstraintStatus(accessToken);
      budgetOutcome = status.Outcome;
      budgetPct = status.MaxCost ? Math.min(100, Math.round((status.CumulativeCost / status.MaxCost) * 100)) : null;
    } catch {
      budgetError = true;
    }
  }

  return (
    <AppShell active="overview" pageTitle="Overview">
      <PageIntro title="Overview" subtitle="What's happening, what needs a decision, and what the company knows." />

      <div className="rise" style={{ display: "grid", gridTemplateColumns: "repeat(3,1fr)", gap: 14, marginBottom: 16 }}>
        <div className="card-hover" style={statCard}>
          <div style={{ fontSize: 12, color: "var(--muted-foreground)" }}>Pending approvals</div>
          {approvalsError ? (
            <div style={{ fontSize: 13, color: "var(--destructive)", marginTop: 6 }}>Unavailable</div>
          ) : (
            <div className="glow-num ds-mono" style={{ fontSize: 26, fontWeight: 700, marginTop: 6, color: "var(--primary)" }}>
              {approvals === null ? "—" : approvals.length}
            </div>
          )}
        </div>
        <div className="card-hover" style={statCard}>
          <div style={{ fontSize: 12, color: "var(--muted-foreground)" }}>Approved knowledge items</div>
          {knowledgeError ? (
            <div style={{ fontSize: 13, color: "var(--destructive)", marginTop: 6 }}>Unavailable</div>
          ) : (
            <div className="glow-num ds-mono" style={{ fontSize: 26, fontWeight: 700, marginTop: 6, color: "var(--primary)" }}>
              {knowledgeCount === null ? "—" : knowledgeCount}
            </div>
          )}
        </div>
        <div className="card-hover" style={statCard}>
          <div style={{ fontSize: 12, color: "var(--muted-foreground)" }}>Budget utilization</div>
          {budgetError ? (
            <div style={{ fontSize: 13, color: "var(--destructive)", marginTop: 6 }}>Unavailable</div>
          ) : (
            <>
              <div className="glow-num ds-mono" style={{ fontSize: 26, fontWeight: 700, marginTop: 6, color: "var(--warning)" }}>
                {budgetPct !== null ? `${budgetPct}%` : "—"}
              </div>
              {budgetOutcome && <div style={{ fontSize: 12, color: "var(--muted-foreground)", marginTop: 2 }}>{budgetOutcome}</div>}
            </>
          )}
        </div>
      </div>

      <div className="rise" style={panelCard}>
        <div style={{ display: "flex", alignItems: "center", marginBottom: 14 }}>
          <div style={{ fontSize: 14, fontWeight: 600, flexGrow: 1 }}>Needs your decision</div>
          <Link href="/approvals" style={linkBtn}>
            View all →
          </Link>
        </div>
        {approvalsError && <div className="ds-error">Could not load pending approvals.</div>}
        {!approvalsError && approvals !== null && approvals.length === 0 && <Empty text="Nothing needs your approval right now." />}
        {!approvalsError &&
          approvals !== null &&
          approvals.slice(0, 3).map((a) => (
            <div key={a.ApprovalID} style={{ display: "flex", alignItems: "center", gap: 14, padding: "12px 0", borderTop: "1px solid var(--border)" }}>
              <div style={{ flexGrow: 1 }}>
                <div style={{ fontSize: 13, fontWeight: 500 }}>{approvalSummary(a.Action)}</div>
                <div style={{ fontSize: 12, color: "var(--muted-foreground)", marginTop: 2 }}>Requested {a.CreatedAt}</div>
              </div>
            </div>
          ))}
      </div>
    </AppShell>
  );
}
