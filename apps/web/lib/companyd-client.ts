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
  // Set only when outcome is APPROVAL_REQUIRED — see
  // apps/companyd/internal/adapters/httpapi/approvals.go.
  approvalId?: string;
};

export type LatestResult = {
  resultId: string;
  outcome: string;
  reportedAt: string;
  output?: { text?: string };
};

// One ExecutionAttempt against an ExecutionUnit's ExecutionIntent. See
// apps/companyd/internal/application/workflow_status.go's AttemptView.
export type ExecutionAttempt = {
  attemptId: string;
  attemptNumber: number;
  status: string;
  providerRunId?: string;
  createdAt: string;
  lastHeartbeatAt?: string;
  terminalAt?: string;
};

// An ExecutionUnit groups one ExecutionIntent with every ExecutionAttempt
// made against it, for this Workflow. Deliberately not called "Node" — that
// word is reserved for a CompanyOS runtime/compute node (an addressable
// participant with its own resources and capabilities that a scheduler
// places work onto), a different, not-yet-built concept. See
// apps/companyd/internal/domain/execution/types.go.
export type ExecutionUnit = {
  intentId: string;
  intentStatus: string;
  dueAt: string;
  attempts: ExecutionAttempt[];
};

export type WorkflowStatus = {
  workflowId: string;
  state: string;
  version: number;
  waitReason: string | null;
  terminalReason: string | null;
  latestResult?: LatestResult;
  units: ExecutionUnit[];
};

// authHeaders builds the Authorization header every call below sends.
// companyd requires and verifies this Supabase-issued JWT on every
// /v1/workflows*/approvals* route (ROADMAP.md Phase 3 Slice 4) — it never
// trusts a client-asserted Principal, so accessToken always comes from
// the caller's own verified Supabase session, never a value chosen here.
function authHeaders(accessToken: string): HeadersInit {
  return { Authorization: `Bearer ${accessToken}` };
}

export async function createWorkflow(accessToken: string): Promise<WorkflowCommandResult> {
  const res = await fetch(`${COMPANYD_URL}/v1/workflows`, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...authHeaders(accessToken) },
    body: JSON.stringify({}),
  });
  return res.json();
}

export async function startWorkflow(
  workflowId: string,
  expectedVersion: number,
  accessToken: string,
): Promise<WorkflowCommandResult> {
  const res = await fetch(`${COMPANYD_URL}/v1/workflows/${workflowId}/start`, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...authHeaders(accessToken) },
    body: JSON.stringify({ expectedVersion }),
  });
  return res.json();
}

export async function cancelWorkflow(
  workflowId: string,
  expectedVersion: number,
  accessToken: string,
): Promise<WorkflowCommandResult> {
  const res = await fetch(`${COMPANYD_URL}/v1/workflows/${workflowId}/cancel`, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...authHeaders(accessToken) },
    body: JSON.stringify({ expectedVersion }),
  });
  return res.json();
}

// resolveApproval is the human decision on a PENDING Approval
// (docs/domain/approval.md). The deciding Principal is never sent from
// here — companyd always resolves it server-side to
// fixtures.Registry.ApproverPrincipal(), the same never-client-asserted-
// identity pattern every other call in this file follows. accessToken is
// only companyd's authentication gate, not the deciding identity.
export async function resolveApproval(
  approvalId: string,
  approve: boolean,
  accessToken: string,
): Promise<WorkflowCommandResult> {
  const res = await fetch(`${COMPANYD_URL}/v1/approvals/${approvalId}/decide`, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...authHeaders(accessToken) },
    body: JSON.stringify({ approve }),
  });
  return res.json();
}

// One row in the Approval inbox (ROADMAP.md Phase 10 Slice 1). See
// apps/companyd/internal/adapters/httpapi/approvals.go's pendingApprovalView.
export type PendingApproval = {
  approvalId: string;
  action: string;
  resourceType: string;
  resourceId: string;
  requestingPrincipalId: string;
  createdAt: string;
};

export async function listPendingApprovals(accessToken: string): Promise<PendingApproval[]> {
  const res = await fetch(`${COMPANYD_URL}/v1/approvals?status=PENDING`, {
    cache: "no-store",
    headers: authHeaders(accessToken),
  });
  return res.json();
}

export async function getWorkflowStatus(
  workflowId: string,
  accessToken: string,
): Promise<WorkflowStatus> {
  const res = await fetch(`${COMPANYD_URL}/v1/workflows/${workflowId}`, {
    cache: "no-store",
    headers: authHeaders(accessToken),
  });
  return res.json();
}
