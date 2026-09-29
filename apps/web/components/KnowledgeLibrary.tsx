"use client";

import { useEffect, useState } from "react";
import type { KnowledgeItem } from "@/lib/companyd-client";
import { panelCard } from "@/lib/ui-styles";
import Empty from "./ui/Empty";

// docs/ux/screen-inventory.md#knowledge-library — browse APPROVED items
// (GET /v1/knowledge/items defaults to APPROVED-only per knowledge.md's
// retrieval contract). Read-only; no draft/review UI in this pass.
export default function KnowledgeLibrary() {
  const [items, setItems] = useState<KnowledgeItem[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  async function load() {
    setError(null);
    try {
      const res = await fetch("/api/knowledge/items", { cache: "no-store" });
      if (!res.ok) {
        setError("Could not load Knowledge items — the request failed.");
        return;
      }
      setItems(await res.json());
    } catch {
      setError("Could not load Knowledge items — the request failed.");
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

  if (items === null) {
    return (
      <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
        {[0, 1, 2].map((i) => (
          <div key={i} className="ds-skeleton" style={{ height: 64 }} />
        ))}
      </div>
    );
  }

  if (items.length === 0) {
    return <Empty text="No approved knowledge yet." />;
  }

  return (
    <div className="rise" style={{ display: "flex", flexDirection: "column", gap: 10 }}>
      {items.map((item) => (
        <div key={`${item.KnowledgeItemID}-${item.Version}`} style={panelCard}>
          <div style={{ fontSize: 14, fontWeight: 500 }}>{item.Claim}</div>
          <div style={{ fontSize: 12, color: "var(--muted-foreground)", marginTop: 6 }}>
            {item.SourceType} · v{item.Version} · {item.ReviewedAt ?? item.CreatedAt}
          </div>
        </div>
      ))}
    </div>
  );
}
