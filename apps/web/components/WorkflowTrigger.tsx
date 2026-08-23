"use client";

import { useEffect, useRef, useState } from "react";
import type { RealtimeChannel } from "@supabase/supabase-js";
import { createClient } from "@/lib/supabase/client";
import type { WorkflowStatus } from "@/lib/companyd-client";
import WorkflowExecutionTree from "./WorkflowExecutionTree";

const TERMINAL_STATES = new Set(["COMPLETED", "FAILED", "CANCELLED"]);

// Realtime push is best-effort (docs/architecture/events.md: notification
// "is a recoverable hint, not a dependency"), so this slow reconciliation
// poll is the safety net for a missed broadcast or a dropped socket — not
// the primary driver of UI updates anymore. Mirrors
// internal/runtime.Runtime's own wake-up-plus-poll-fallback shape
// server-side.
const RECONCILE_INTERVAL_MS = 15000;

function Loader({ label }: { label: string }) {
  return (
    <span className="loader">
      <span className="loader-ring" />
      <span className="loader-text">{label}</span>
    </span>
  );
}

// Minimal Phase 1 vertical-slice trigger: create a Workflow, start it, and
// watch its status until terminal, pushed over Supabase Realtime rather
// than polled. No design polish by design — this proves the
// kernel-to-daemon-to-web-to-db loop, not a finished UI. See
// docs/architecture/application.md and ROADMAP.md's Phase 1 slices.
export default function WorkflowTrigger() {
  const [workflow, setWorkflow] = useState<{ id: string; version: number } | null>(null);
  const [status, setStatus] = useState<WorkflowStatus | null>(null);
  const [busy, setBusy] = useState(false);
  const [polling, setPolling] = useState(false);
  const [error, setError] = useState<string | null>(null);
  // Set when CANCEL_WORKFLOW returns APPROVAL_REQUIRED (Phase 3 Slice 2) —
  // an in-flight cancel needs a human decision before it takes effect. See
  // apps/companyd/internal/application/workflow_cancel.go's
  // cancelAutonomyRequirement.
  const [pendingApprovalId, setPendingApprovalId] = useState<string | null>(null);
  const channelRef = useRef<RealtimeChannel | null>(null);
  const reconcileRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    return () => {
      stopWatching();
    };
  }, []);

  function stopWatching() {
    if (channelRef.current) {
      channelRef.current.unsubscribe();
      channelRef.current = null;
    }
    if (reconcileRef.current) {
      clearInterval(reconcileRef.current);
      reconcileRef.current = null;
    }
    setPolling(false);
  }

  async function refreshStatus(workflowId: string) {
    const s: WorkflowStatus = await fetch(`/api/workflows/${workflowId}`).then((r) => r.json());
    setStatus(s);
    if (TERMINAL_STATES.has(s.state)) {
      stopWatching();
    }
  }

  async function handleCreate() {
    setBusy(true);
    setError(null);
    try {
      const result = await fetch("/api/workflows", { method: "POST" }).then((r) => r.json());
      if (result.outcome !== "ACCEPTED" || !result.workflowId) {
        setError(`create failed: ${result.outcome} ${result.reasons?.join(", ") ?? ""}`);
        return;
      }
      setWorkflow({ id: result.workflowId, version: result.version });
      setStatus(null);
    } finally {
      setBusy(false);
    }
  }

  async function handleStart() {
    if (!workflow) return;
    setBusy(true);
    setError(null);
    try {
      const result = await fetch(`/api/workflows/${workflow.id}/start`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ expectedVersion: workflow.version }),
      }).then((r) => r.json());
      if (result.outcome !== "ACCEPTED") {
        setError(`start failed: ${result.outcome} ${result.reasons?.join(", ") ?? ""}`);
        return;
      }
      setWorkflow({ id: workflow.id, version: result.version });
      startWatching(workflow.id);
    } finally {
      setBusy(false);
    }
  }

  async function handleCancel() {
    if (!workflow) return;
    setBusy(true);
    setError(null);
    try {
      const version = status?.version ?? workflow.version;
      const result = await fetch(`/api/workflows/${workflow.id}/cancel`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ expectedVersion: version }),
      }).then((r) => r.json());
      if (result.outcome === "APPROVAL_REQUIRED" && result.approvalId) {
        setPendingApprovalId(result.approvalId);
        return;
      }
      if (result.outcome !== "ACCEPTED") {
        setError(`cancel failed: ${result.outcome} ${result.reasons?.join(", ") ?? ""}`);
        return;
      }
      stopWatching();
      setWorkflow({ id: workflow.id, version: result.version });
      setStatus((prev) => (prev ? { ...prev, state: result.state, version: result.version } : prev));
    } finally {
      setBusy(false);
    }
  }

  // Resolves the pending Approval a READY-Workflow cancel produced. On
  // approve, companyd immediately resumes and completes the cancel in the
  // same call (docs/domain/approval.md's resolution steps) — this handler
  // just reflects whatever the server actually did, it doesn't assume.
  async function handleResolveApproval(approve: boolean) {
    if (!pendingApprovalId || !workflow) return;
    setBusy(true);
    setError(null);
    try {
      const result = await fetch(`/api/approvals/${pendingApprovalId}/decide`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ approve }),
      }).then((r) => r.json());
      setPendingApprovalId(null);
      if (approve && result.outcome === "ACCEPTED") {
        stopWatching();
        setWorkflow({ id: workflow.id, version: result.version });
        setStatus((prev) => (prev ? { ...prev, state: result.state, version: result.version } : prev));
        return;
      }
      if (!approve && result.outcome === "REJECTED") {
        return; // Workflow is untouched — still READY, still being watched.
      }
      setError(`approval decision failed: ${result.outcome} ${result.reasons?.join(", ") ?? ""}`);
    } finally {
      setBusy(false);
    }
  }

  // Subscribes to a per-Workflow Supabase Realtime broadcast channel
  // instead of polling — see internal/adapters/notify/realtime.Publisher.
  // A broadcast is only a hint ("something changed"), so the handler always
  // refetches the real status through companyd rather than trusting the
  // broadcast payload itself.
  function startWatching(workflowId: string) {
    stopWatching();
    setPolling(true);

    const channel = createClient()
      .channel(`workflow:${workflowId}`)
      .on("broadcast", { event: "workflow_changed" }, () => {
        refreshStatus(workflowId);
      })
      .subscribe((subscribeStatus) => {
        if (subscribeStatus === "SUBSCRIBED") {
          // Catches anything that changed between create/start and the
          // subscription actually going live.
          refreshStatus(workflowId);
        }
      });
    channelRef.current = channel;

    reconcileRef.current = setInterval(() => {
      refreshStatus(workflowId);
    }, RECONCILE_INTERVAL_MS);
  }

  return (
    <div>
      <div className="actions">
        <button className="btn" onClick={handleCreate} disabled={busy}>
          Create Workflow
        </button>
        <button className="btn btn-accent" onClick={handleStart} disabled={busy || !workflow || Boolean(status)}>
          Start Workflow
        </button>
        <button
          className="btn"
          onClick={handleCancel}
          disabled={busy || !workflow || Boolean(pendingApprovalId) || (status !== null && TERMINAL_STATES.has(status.state))}
        >
          Cancel Workflow
        </button>
        {busy && <Loader label="Transmitting" />}
      </div>

      {error && <p className="error-text">{error}</p>}

      {pendingApprovalId && (
        <div className="status-row">
          <span className="status-label">Cancellation needs approval</span>
          <button className="btn btn-accent" onClick={() => handleResolveApproval(true)} disabled={busy}>
            Approve
          </button>
          <button className="btn" onClick={() => handleResolveApproval(false)} disabled={busy}>
            Reject
          </button>
        </div>
      )}

      {workflow && (
        <p className="workflow-id">
          Workflow <code>{workflow.id}</code>
        </p>
      )}

      {polling && !status && (
        <p className="workflow-id">
          <Loader label="Initializing Runtime" />
        </p>
      )}

      {status && (
        <div className="status-block">
          <div className="status-row">
            <span className="status-label">State</span>
            <span className={`state-badge state-${status.state}`}>{status.state}</span>
            {polling && !TERMINAL_STATES.has(status.state) && <Loader label="Awaiting Runtime" />}
          </div>

          <WorkflowExecutionTree workflowState={status.state} units={status.units} />

          {status.latestResult && (
            <>
              <div className="status-row">
                <span className="status-label">Outcome</span>
                <span>{status.latestResult.outcome}</span>
              </div>
              {status.latestResult.output?.text && (
                <div className="result-text">{status.latestResult.output.text}</div>
              )}
            </>
          )}
        </div>
      )}
    </div>
  );
}
