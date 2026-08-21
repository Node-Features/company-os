"use client";

import { useEffect, useRef, useState } from "react";
import type { WorkflowStatus } from "@/lib/companyd-client";

const TERMINAL_STATES = new Set(["COMPLETED", "FAILED", "CANCELLED"]);

function Loader({ label }: { label: string }) {
  return (
    <span className="loader">
      <span className="loader-ring" />
      <span className="loader-text">{label}</span>
    </span>
  );
}

// Minimal Phase 1 vertical-slice trigger: create a Workflow, start it, and
// poll its status until terminal. No design polish by design — this proves
// the kernel-to-daemon-to-web-to-db loop, not a finished UI. See
// docs/architecture/application.md and ROADMAP.md's Phase 1 slices.
export default function WorkflowTrigger() {
  const [workflow, setWorkflow] = useState<{ id: string; version: number } | null>(null);
  const [status, setStatus] = useState<WorkflowStatus | null>(null);
  const [busy, setBusy] = useState(false);
  const [polling, setPolling] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    return () => {
      if (pollRef.current) clearInterval(pollRef.current);
    };
  }, []);

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
      startPolling(workflow.id);
    } finally {
      setBusy(false);
    }
  }

  function startPolling(workflowId: string) {
    if (pollRef.current) clearInterval(pollRef.current);
    setPolling(true);
    pollRef.current = setInterval(async () => {
      const s: WorkflowStatus = await fetch(`/api/workflows/${workflowId}`).then((r) => r.json());
      setStatus(s);
      if (TERMINAL_STATES.has(s.state)) {
        if (pollRef.current) clearInterval(pollRef.current);
        setPolling(false);
      }
    }, 2000);
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
        {busy && <Loader label="Transmitting" />}
      </div>

      {error && <p className="error-text">{error}</p>}

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
