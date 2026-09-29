"use client";

import { useId, useState } from "react";
import type { ReactNode } from "react";

// "View technical details" toggle (docs/architecture/ui-ux.md invariant 8):
// a governed decision is stated in plain language first, with technical
// detail available behind an explicit toggle.
export default function Disclosure({
  hiddenLabel = "View technical details",
  shownLabel = "Hide technical details",
  detail,
}: {
  hiddenLabel?: string;
  shownLabel?: string;
  detail: ReactNode;
}) {
  const [expanded, setExpanded] = useState(false);
  const detailId = useId();

  return (
    <div>
      <button
        type="button"
        aria-expanded={expanded}
        aria-controls={detailId}
        onClick={() => setExpanded((e) => !e)}
        style={{ background: "none", border: "none", color: "var(--primary)", fontSize: 12, padding: 0, marginTop: 8 }}
      >
        {expanded ? shownLabel : hiddenLabel}
      </button>
      {expanded && (
        <div id={detailId} className="ds-mono rise ds-disclosure-detail">
          {detail}
        </div>
      )}
    </div>
  );
}
