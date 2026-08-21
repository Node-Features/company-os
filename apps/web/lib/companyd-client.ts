// Typed client for calling companyd. Every exported function here
// should map to exactly one Application use case
// (docs/architecture/application.md) and translate its normalized
// outcome (ACCEPTED/REJECTED/DENIED/APPROVAL_REQUIRED/CONFLICT/
// UNAVAILABLE/INDETERMINATE/INVALID) without reinterpreting it.

const COMPANYD_URL = process.env.COMPANYD_URL ?? "http://localhost:8080";

export async function companydHealth(): Promise<boolean> {
  const res = await fetch(`${COMPANYD_URL}/health`);
  return res.ok;
}

export type WorkflowOutcome =
  | "ACCEPTED"
  | "REJECTED"
  | "DENIED"
  | "APPROVAL_REQUIRED"
  | "CONFLICT"
  | "UNAVAILABLE"
  | "INDETERMINATE"
  | "INVALID";

export type WorkflowCommandResult = {
  outcome: WorkflowOutcome;
  workflowId?: string;
  version?: number;
  state?: string;
  requestId: string;
  reasons?: string[];
};

export type LatestResult = {
  resultId: string;
  outcome: string;
  reportedAt: string;
  output?: { text?: string };
};

export type WorkflowStatus = {
  workflowId: string;
  state: string;
  version: number;
  waitReason: string | null;
  terminalReason: string | null;
  latestResult?: LatestResult;
};

export async function createWorkflow(): Promise<WorkflowCommandResult> {
  const res = await fetch(`${COMPANYD_URL}/v1/workflows`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({}),
  });
  return res.json();
}

export async function startWorkflow(
  workflowId: string,
  expectedVersion: number,
): Promise<WorkflowCommandResult> {
  const res = await fetch(`${COMPANYD_URL}/v1/workflows/${workflowId}/start`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ expectedVersion }),
  });
  return res.json();
}

export async function getWorkflowStatus(workflowId: string): Promise<WorkflowStatus> {
  const res = await fetch(`${COMPANYD_URL}/v1/workflows/${workflowId}`, {
    cache: "no-store",
  });
  return res.json();
}
