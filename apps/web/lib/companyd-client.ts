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

// companyd's error responses (writeError) are `{"error": "..."}` with a
// non-2xx status. Every GET read below must check this before decoding —
// otherwise a 401/503/etc. silently hands the caller an object where it
// expected an array/typed shape (the bug this class fixes: `.map is not a
// function` when companyd errors and the proxy route naively forwards the
// body with an unconditional 200).
export class CompanydError extends Error {
  constructor(public status: number, message: string) {
    super(message);
    this.name = "CompanydError";
  }
}

async function readJSON<T>(res: Response): Promise<T> {
  if (!res.ok) {
    let message = res.statusText;
    try {
      const body = await res.json();
      if (body?.error) message = body.error;
    } catch {
      // Non-JSON error body — fall back to statusText.
    }
    throw new CompanydError(res.status, message);
  }
  return res.json();
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

export async function getWorkflowStatus(
  workflowId: string,
  accessToken: string,
): Promise<WorkflowStatus> {
  const res = await fetch(`${COMPANYD_URL}/v1/workflows/${workflowId}`, {
    cache: "no-store",
    headers: authHeaders(accessToken),
  });
  return readJSON<WorkflowStatus>(res);
}

// ---------------------------------------------------------------------
// The GET reads below all encode a Go struct with no `json` tags
// directly (json.NewEncoder(w).Encode(x)) — so field names are the
// PascalCase Go identifiers, unlike the camelCase command-response
// shapes above (which have explicit tags). Verified per-handler against
// apps/companyd/internal/adapters/httpapi/*.go and main.go's route table.
// ---------------------------------------------------------------------

// One row from GET /v1/approvals — apps/companyd/internal/domain/approval.Approval.
export type Approval = {
  ApprovalID: string;
  OrganizationID: string;
  PendingCommandID: string;
  RequestingPrincipalID: string;
  Action: string;
  ResourceType: string;
  ResourceID: string;
  ProposalDigest: string;
  Status: string;
  DecidedByPrincipalID?: string | null;
  DecidedAt?: string | null;
  Reason?: string | null;
  CreatedAt: string;
};

export async function listApprovals(
  accessToken: string,
  statuses?: string[],
  limit?: number,
  offset?: number,
): Promise<Approval[]> {
  const params = new URLSearchParams();
  if (statuses && statuses.length > 0) params.set("status", statuses.join(","));
  if (limit) params.set("limit", String(limit));
  if (offset) params.set("offset", String(offset));
  const qs = params.toString();
  const res = await fetch(`${COMPANYD_URL}/v1/approvals${qs ? `?${qs}` : ""}`, {
    cache: "no-store",
    headers: authHeaders(accessToken),
  });
  return readJSON<Approval[]>(res);
}

// One row from GET /v1/objectives — an Objective only ever exists once a
// proposal is approved (CreateObjective only runs on the approve-resume
// path); a pending/declined Decision exists only as an Approval record —
// see docs/architecture/ui-ux.md's "Decisions caveat".
export type Objective = {
  ObjectiveID: string;
  OrganizationID: string;
  Title: string;
  Intent: string;
  Status: "PROPOSED" | "APPROVED" | "RETIRED";
  OwnerPrincipalID: string;
  SourceType: "FINDING" | "RECOMMENDATION" | "EVALUATION" | "RESOURCE_EVALUATION";
  SourceID: string;
  GovernanceDecisionID: string;
  ApprovalID: string | null;
  CreatedAt: string;
};

export async function listObjectives(
  accessToken: string,
  statuses?: string[],
  limit?: number,
  offset?: number,
): Promise<Objective[]> {
  const params = new URLSearchParams();
  if (statuses && statuses.length > 0) params.set("statuses", statuses.join(","));
  if (limit) params.set("limit", String(limit));
  if (offset) params.set("offset", String(offset));
  const qs = params.toString();
  const res = await fetch(`${COMPANYD_URL}/v1/objectives${qs ? `?${qs}` : ""}`, {
    cache: "no-store",
    headers: authHeaders(accessToken),
  });
  return readJSON<Objective[]>(res);
}

// GET /v1/research/questions/{id} — application.ResearchQuestionStatusView.
// There is no "list all open questions" endpoint yet, so a Research screen
// can only render once a specific question id is known.
export type EvidenceView = { EvidenceID: string; Source: string; Content: string };
export type RecommendationView = { RecommendationID: string; ProposedAction: string; Status: string };
export type FindingView = {
  FindingID: string;
  Claim: string;
  Status: string;
  EvidenceIDs: string[];
  Recommendations: RecommendationView[];
};
export type ResearchQuestionStatusView = {
  QuestionID: string;
  SignalID: string;
  Text: string;
  Status: string;
  Evidence: EvidenceView[];
  Findings: FindingView[];
};

export async function getResearchQuestion(
  questionId: string,
  accessToken: string,
): Promise<ResearchQuestionStatusView> {
  const res = await fetch(`${COMPANYD_URL}/v1/research/questions/${questionId}`, {
    cache: "no-store",
    headers: authHeaders(accessToken),
  });
  return readJSON<ResearchQuestionStatusView>(res);
}

// GET /v1/finance/constraint-status — one organization-wide status, not a
// per-department breakdown (application.GetConstraintStatus takes no
// subject argument).
export type ConstraintStatusView = {
  SubjectID: string;
  CumulativeCost: number;
  MaxCost: number | null;
  Outcome: string;
};

export async function getConstraintStatus(accessToken: string): Promise<ConstraintStatusView> {
  const res = await fetch(`${COMPANYD_URL}/v1/finance/constraint-status`, {
    cache: "no-store",
    headers: authHeaders(accessToken),
  });
  return readJSON<ConstraintStatusView>(res);
}

// GET /v1/me/subjects/{id}/performance-profile — needs a specific subject
// id; there is no "list all subjects" endpoint yet.
export type PerformanceProfile = {
  SubjectID: string;
  OrganizationID: string;
  LatestEvaluationID: string;
  Outcome: "PASS" | "FAIL" | "INCONCLUSIVE";
  SuccessRate: number;
  UpdatedAt: string;
};

export async function getPerformanceProfile(
  subjectId: string,
  accessToken: string,
): Promise<PerformanceProfile> {
  const res = await fetch(`${COMPANYD_URL}/v1/me/subjects/${subjectId}/performance-profile`, {
    cache: "no-store",
    headers: authHeaders(accessToken),
  });
  return readJSON<PerformanceProfile>(res);
}

// GET /v1/knowledge/items — defaults to APPROVED-only (knowledge.md's
// retrieval contract); a draft-inclusive query needs an explicit `purpose`
// (docs/architecture/ui-ux.md invariant 4 — the UI must collect and show
// it, never silently default one in).
export type KnowledgeItem = {
  KnowledgeItemID: string;
  OrganizationID: string;
  Version: number;
  Claim: string;
  ContentDigest: string;
  Classification: string;
  SourceType: string;
  SourceID: string;
  ProducedByPrincipalID: string;
  ProducedByMethod: string;
  Status: "DRAFT" | "IN_REVIEW" | "APPROVED" | "REJECTED";
  DuplicateOfItemID?: string | null;
  CreatedAt: string;
  ReviewerPrincipalID?: string | null;
  GovernanceDecisionID?: string | null;
  ApprovalID?: string | null;
  ReviewedAt?: string | null;
};

export async function listKnowledgeItems(
  accessToken: string,
  statuses?: string[],
  purpose?: string,
  limit?: number,
): Promise<KnowledgeItem[]> {
  const params = new URLSearchParams();
  if (statuses && statuses.length > 0) params.set("statuses", statuses.join(","));
  if (purpose) params.set("purpose", purpose);
  if (limit) params.set("limit", String(limit));
  const qs = params.toString();
  const res = await fetch(`${COMPANYD_URL}/v1/knowledge/items${qs ? `?${qs}` : ""}`, {
    cache: "no-store",
    headers: authHeaders(accessToken),
  });
  const body = await readJSON<KnowledgeItem[] | null>(res);
  // The list handler doesn't coerce a nil slice to [] the way
  // ListObjectivesHandler does — normalize here so callers never branch
  // on null vs [].
  return body ?? [];
}
