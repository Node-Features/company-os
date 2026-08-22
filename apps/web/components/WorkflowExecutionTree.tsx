"use client";

import { useEffect, useState } from "react";
import type { ExecutionUnit } from "@/lib/companyd-client";

const ACTIVE_INTENT_STATUSES = new Set(["PENDING", "CLAIMED"]);
const ACTIVE_ATTEMPT_STATUSES = new Set(["CLAIMED", "DISPATCHED", "WAITING"]);
const TERMINAL_WORKFLOW_STATES = new Set(["COMPLETED", "FAILED", "CANCELLED"]);

function statusClass(status: string): string {
  switch (status) {
    case "SUCCEEDED":
      return "unit-green";
    case "PENDING":
    case "FAILED_RETRYABLE":
      return "unit-amber";
    case "FAILED_TERMINAL":
    case "CANCELLED":
    case "LEASE_EXPIRED":
      return "unit-danger";
    case "CLOSED":
      return "unit-dim";
    default:
      return "unit-cyan";
  }
}

function formatDuration(ms: number): string {
  const clamped = Math.max(0, ms);
  if (clamped < 1000) return `${Math.round(clamped)}ms`;
  const totalSeconds = clamped / 1000;
  if (totalSeconds < 60) return `${totalSeconds.toFixed(1)}s`;
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = Math.round(totalSeconds % 60);
  return `${minutes}m${seconds.toString().padStart(2, "0")}s`;
}

type Row = {
  key: string;
  prefix: string;
  label: string;
  sub?: string;
  status: string;
  pulse: boolean;
  durationMs?: number;
};

// Builds one line per Workflow/Intent/Attempt, tree-connector prefixes
// computed the same way `tree`/git-log-graph-style output is: a corner
// (└─) for the last sibling, a tee (├─) otherwise, with a continuation
// column (│) carried down for any ancestor that still has siblings below.
function buildRows(workflowState: string, units: ExecutionUnit[], now: number): Row[] {
  const rows: Row[] = [];

  let workflowDurationMs: number | undefined;
  if (units.length > 0) {
    const start = Math.min(...units.map((u) => new Date(u.dueAt).getTime()));
    const terminalTimes = units
      .flatMap((u) => u.attempts ?? [])
      .map((a) => (a.terminalAt ? new Date(a.terminalAt).getTime() : undefined))
      .filter((t): t is number => t !== undefined);
    const end =
      TERMINAL_WORKFLOW_STATES.has(workflowState) && terminalTimes.length > 0
        ? Math.max(...terminalTimes)
        : now;
    workflowDurationMs = end - start;
  }

  rows.push({
    key: "workflow",
    prefix: "",
    label: "Workflow",
    status: workflowState,
    pulse: !TERMINAL_WORKFLOW_STATES.has(workflowState),
    durationMs: workflowDurationMs,
  });

  units.forEach((unit, unitIdx) => {
    const isLastUnit = unitIdx === units.length - 1;
    const attempts = unit.attempts ?? [];

    rows.push({
      key: unit.intentId,
      prefix: isLastUnit ? "└─ " : "├─ ",
      label: "Intent",
      sub: unit.intentId.slice(0, 8),
      status: unit.intentStatus,
      pulse: ACTIVE_INTENT_STATUSES.has(unit.intentStatus),
    });

    const childColumn = isLastUnit ? "   " : "│  ";
    attempts.forEach((attempt, attemptIdx) => {
      const isLastAttempt = attemptIdx === attempts.length - 1;
      const start = new Date(attempt.createdAt).getTime();
      const end = attempt.terminalAt ? new Date(attempt.terminalAt).getTime() : now;

      rows.push({
        key: attempt.attemptId,
        prefix: childColumn + (isLastAttempt ? "└─ " : "├─ "),
        label: `Attempt #${attempt.attemptNumber}`,
        status: attempt.status,
        pulse: ACTIVE_ATTEMPT_STATUSES.has(attempt.status),
        durationMs: end - start,
      });
    });
  });

  return rows;
}

// Renders the Workflow and every ExecutionUnit beneath it (an
// ExecutionIntent plus its ExecutionAttempts, docs/domain/execution.md) as
// a terminal-style connector tree with a live execution timer per row, the
// same shape Claude Code renders a tool-call tree with duration. Ticks once
// a second while anything is still in flight; stops ticking once
// everything is terminal so the finished tree doesn't keep re-rendering
// forever.
//
// Deliberately named "ExecutionUnit," not "Node": in CompanyOS a Node is a
// runtime/compute participant with its own resources and capabilities that
// a scheduler places work onto (docs/architecture/node.md) — a different,
// not-yet-built concept from this per-Workflow dispatch bookkeeping.
export default function WorkflowExecutionTree({
  workflowState,
  units,
}: {
  workflowState: string;
  units: ExecutionUnit[] | undefined;
}) {
  const safeUnits = units ?? [];
  const [now, setNow] = useState(() => Date.now());

  const anyActive =
    !TERMINAL_WORKFLOW_STATES.has(workflowState) ||
    safeUnits.some(
      (u) =>
        ACTIVE_INTENT_STATUSES.has(u.intentStatus) ||
        (u.attempts ?? []).some((a) => ACTIVE_ATTEMPT_STATUSES.has(a.status)),
    );

  useEffect(() => {
    if (!anyActive) return;
    const id = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(id);
  }, [anyActive]);

  const rows = buildRows(workflowState, safeUnits, now);

  return (
    <div className="unit-tree">
      {rows.map((row) => (
        <div className="unit-tree-row" key={row.key}>
          <span className="unit-tree-prefix">{row.prefix}</span>
          <span className={`unit-tree-marker ${statusClass(row.status)} ${row.pulse ? "unit-pulse" : ""}`}>
            {"●"}
          </span>
          <span className="unit-tree-label">
            {row.label}
            {row.sub && <span className="unit-tree-sub"> {row.sub}</span>}
          </span>
          <span className={`unit-tree-status ${statusClass(row.status)}`}>{row.status}</span>
          {row.durationMs !== undefined && (
            <span className="unit-tree-duration">({formatDuration(row.durationMs)})</span>
          )}
        </div>
      ))}
      {safeUnits.length === 0 && <p className="unit-empty">No execution units yet.</p>}
    </div>
  );
}
