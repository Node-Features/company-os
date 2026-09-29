"use client";

import { useEffect, useState } from "react";
import type { Approval, Objective } from "@/lib/companyd-client";
import Disclosure from "./ui/Disclosure";
import Empty from "./ui/Empty";
import { panelCard, badgeStyle, traceBtn } from "@/lib/ui-styles";

const PAGE_SIZE = 5;

type Section = "pending" | "approved";

// docs/architecture/ui-ux.md's "Decisions caveat": GET /v1/objectives only
// ever surfaces already-approved Decisions; a pending or declined one
// exists only as an Approval record. The two sources are paginated
// independently against their own real offset/limit — a merged "one list"
// pager isn't meaningful across two different APIs with no shared cursor,
// so this renders as two real, separately-paged sections instead of
// fetching everything and slicing client-side.
function usePage<T>(fetchPage: (limit: number, offset: number) => Promise<T[]>) {
  const [page, setPage] = useState(0);
  const [rows, setRows] = useState<T[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [hasNext, setHasNext] = useState(false);

  async function load(p: number) {
    setError(null);
    try {
      const result = await fetchPage(PAGE_SIZE, p * PAGE_SIZE);
      setRows(result);
      setHasNext(result.length === PAGE_SIZE);
    } catch {
      setError("Could not load — the request failed.");
    }
  }

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    load(page);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page]);

  return { page, setPage, rows, error, hasNext, reload: () => load(page) };
}

function Pager({ page, hasNext, onPrev, onNext }: { page: number; hasNext: boolean; onPrev: () => void; onNext: () => void }) {
  return (
    <div style={{ display: "flex", alignItems: "center", justifyContent: "flex-end", gap: 10, marginTop: 10 }}>
      <span className="ds-mono" style={{ fontSize: 11, color: "var(--muted-foreground)" }}>
        Page {page + 1}
      </span>
      <button style={traceBtn} onClick={onPrev} disabled={page === 0}>
        ← Prev
      </button>
      <button style={traceBtn} onClick={onNext} disabled={!hasNext}>
        Next →
      </button>
    </div>
  );
}

export default function DecisionsList() {
  const pending = usePage<Approval>(async (limit, offset) => {
    const res = await fetch(`/api/approvals?status=PENDING,REJECTED&limit=${limit}&offset=${offset}`, { cache: "no-store" });
    if (!res.ok) throw new Error("failed");
    return res.json();
  });
  const approved = usePage<Objective>(async (limit, offset) => {
    const res = await fetch(`/api/objectives?limit=${limit}&offset=${offset}`, { cache: "no-store" });
    if (!res.ok) throw new Error("failed");
    return res.json();
  });

  function renderSection(kind: Section, s: ReturnType<typeof usePage<Approval | Objective>>) {
    if (s.error) {
      return (
        <div className="ds-error">
          <span>{s.error}</span>
          <button style={{ background: "none", border: "none", color: "var(--primary)", fontSize: 12 }} onClick={s.reload}>
            Retry
          </button>
        </div>
      );
    }
    if (s.rows === null) {
      return (
        <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
          {[0, 1].map((i) => (
            <div key={i} className="ds-skeleton" style={{ height: 84 }} />
          ))}
        </div>
      );
    }
    if (s.rows.length === 0 && s.page === 0) {
      return (
        <Empty
          text={
            kind === "pending"
              ? "No pending or declined Decisions right now."
              : "No approved Decisions yet."
          }
        />
      );
    }
    return (
      <>
        <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
          {kind === "pending"
            ? (s.rows as Approval[]).map((a) => (
                <div key={a.ApprovalID} className="card-hover" style={panelCard}>
                  <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
                    <div style={{ fontSize: 14, fontWeight: 600 }}>Proposed objective</div>
                    <span style={badgeStyle(a.Status === "REJECTED" ? "failed" : "pending")}>
                      {a.Status === "REJECTED" ? "DECLINED" : "PENDING"}
                    </span>
                  </div>
                  <div style={{ fontSize: 12, color: "var(--muted-foreground)", marginTop: 4 }}>Requested {a.CreatedAt}</div>
                  <Disclosure
                    detail={`action=${a.Action} · resource=${a.ResourceType} ${a.ResourceID} · requested_by=${a.RequestingPrincipalID}`}
                  />
                </div>
              ))
            : (s.rows as Objective[]).map((o) => (
                <div key={o.ObjectiveID} className="card-hover" style={panelCard}>
                  <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
                    <div style={{ fontSize: 14, fontWeight: 600 }}>{o.Title}</div>
                    <span style={badgeStyle("approved")}>APPROVED</span>
                  </div>
                  <div style={{ fontSize: 12, color: "var(--muted-foreground)", marginTop: 4 }}>
                    Sourced from {o.SourceType} · {o.CreatedAt}
                  </div>
                  <Disclosure
                    detail={`objective_id=${o.ObjectiveID} · source=${o.SourceType} ${o.SourceID} · owner=${o.OwnerPrincipalID}`}
                  />
                </div>
              ))}
        </div>
        <Pager
          page={s.page}
          hasNext={s.hasNext}
          onPrev={() => s.setPage((p) => Math.max(0, p - 1))}
          onNext={() => s.setPage((p) => p + 1)}
        />
      </>
    );
  }

  return (
    <div className="rise" style={{ display: "flex", flexDirection: "column", gap: 28 }}>
      <div>
        <div style={{ fontSize: 14, fontWeight: 600, marginBottom: 12 }}>Awaiting or declined</div>
        {renderSection("pending", pending)}
      </div>
      <div>
        <div style={{ fontSize: 14, fontWeight: 600, marginBottom: 12 }}>Approved</div>
        {renderSection("approved", approved)}
      </div>
    </div>
  );
}
