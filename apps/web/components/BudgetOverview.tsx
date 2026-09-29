"use client";

import { useEffect, useState } from "react";
import type { ConstraintStatusView } from "@/lib/companyd-client";
import { statCard, panelCard } from "@/lib/ui-styles";
import Empty from "./ui/Empty";

// GET /v1/finance/constraint-status returns one organization-wide status —
// application.GetConstraintStatus takes no subject argument, so there is
// no real per-department breakdown to show; the mockup's per-department
// table is not backed by any endpoint yet, so this screen shows the one
// real reconciled number instead of fabricating department rows
// (docs/architecture/ui-ux.md invariant 7).
export default function BudgetOverview() {
  const [status, setStatus] = useState<ConstraintStatusView | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loaded, setLoaded] = useState(false);

  async function load() {
    setError(null);
    try {
      const res = await fetch("/api/finance/constraint-status", { cache: "no-store" });
      if (!res.ok) {
        setError("Could not load budget status — the request failed.");
        return;
      }
      setStatus(await res.json());
    } catch {
      setError("Could not load budget status — the request failed.");
    } finally {
      setLoaded(true);
    }
  }

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    load();
  }, []);

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

  if (!loaded) {
    return (
      <div style={{ display: "grid", gridTemplateColumns: "repeat(3,1fr)", gap: 14 }}>
        {[0, 1, 2].map((i) => (
          <div key={i} className="ds-skeleton" style={{ height: 76 }} />
        ))}
      </div>
    );
  }

  if (!status) {
    return <Empty text="No budget set for this organization yet." />;
  }

  const utilizationPct = status.MaxCost ? Math.min(100, Math.round((status.CumulativeCost / status.MaxCost) * 100)) : null;

  return (
    <>
      <div className="rise" style={{ display: "grid", gridTemplateColumns: "repeat(3,1fr)", gap: 14, marginBottom: 16 }}>
        <div style={statCard}>
          <div style={{ fontSize: 11, color: "var(--muted-foreground)" }}>Cumulative cost</div>
          <div className="ds-mono" style={{ fontSize: 20, fontWeight: 700, marginTop: 4 }}>
            ${status.CumulativeCost.toFixed(2)}
          </div>
        </div>
        <div style={statCard}>
          <div style={{ fontSize: 11, color: "var(--muted-foreground)" }}>Max cost</div>
          <div className="ds-mono" style={{ fontSize: 20, fontWeight: 700, marginTop: 4 }}>
            {status.MaxCost !== null ? `$${status.MaxCost.toFixed(2)}` : "Not set"}
          </div>
        </div>
        <div style={statCard}>
          <div style={{ fontSize: 11, color: "var(--muted-foreground)" }}>Utilization</div>
          <div className="ds-mono" style={{ fontSize: 20, fontWeight: 700, marginTop: 4, color: "var(--warning)" }}>
            {utilizationPct !== null ? `${utilizationPct}%` : "—"}
          </div>
        </div>
      </div>

      <div className="rise" style={panelCard}>
        <div style={{ fontSize: 11, fontWeight: 600, color: "var(--muted-foreground)", letterSpacing: "0.05em", marginBottom: 6 }}>
          OUTCOME
        </div>
        <div style={{ fontSize: 14 }}>{status.Outcome}</div>
        <div style={{ fontSize: 12, color: "var(--muted-foreground)", marginTop: 10 }}>
          Subject: <span className="ds-mono">{status.SubjectID}</span>. This is the one organization-wide
          constraint status companyd tracks today — a per-department breakdown isn&apos;t backed by any
          endpoint yet.
        </div>
      </div>
    </>
  );
}
