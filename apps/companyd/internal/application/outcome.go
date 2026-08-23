package application

import "github.com/google/uuid"

// Outcome is the normalized use-case result every Application use case
// returns — matching apps/web/lib/companyd-client.ts's doc comment exactly
// so the HTTP adapter can pass it through untouched.
// See docs/architecture/application.md#failure-outcomes.
type Outcome string

const (
	Accepted         Outcome = "ACCEPTED"
	Rejected         Outcome = "REJECTED"
	Denied           Outcome = "DENIED"
	ApprovalRequired Outcome = "APPROVAL_REQUIRED"
	Conflict         Outcome = "CONFLICT"
	Unavailable      Outcome = "UNAVAILABLE"
	Indeterminate    Outcome = "INDETERMINATE"
	Invalid          Outcome = "INVALID"
)

// Result is the common return shape for every state-changing use case.
type Result struct {
	Outcome  Outcome
	Workflow *WorkflowView
	Reasons  []string
	// ApprovalID is set only when Outcome is ApprovalRequired — the client
	// needs it to later call ResolveApproval; there is no other way to
	// discover it (Approval identity is never derivable from Workflow
	// state alone).
	ApprovalID *uuid.UUID
	// ResourceID is the created/referenced resource's identity for
	// non-Workflow use cases (ROADMAP.md Phase 4 Slice 1's Research
	// use cases) — e.g. a new Signal/ResearchQuestion/Finding/
	// Recommendation ID. Workflow use cases keep using Workflow instead.
	ResourceID *uuid.UUID
}

// WorkflowView is the minimal read projection returned to callers and
// served by the status HTTP endpoint.
type WorkflowView struct {
	WorkflowID     string
	Version        int64
	State          string
	WaitReason     *string
	TerminalReason *string
}
