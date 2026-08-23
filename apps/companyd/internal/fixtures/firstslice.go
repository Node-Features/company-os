// Package fixtures holds the hardcoded Organization, Objective,
// WorkflowDefinition, CapabilityDefinition, and Principal values that
// prove the Phase 1 first vertical slice. These are bootstrap constants,
// not organizational logic — persisted domain tables for these concepts
// are out of scope this slice (first-slice plan decision #6); Kernel and
// Application read them through Registry instead.
package fixtures

import (
	"context"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/capability"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/command"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/objective"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/organization"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/principal"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/workflow"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
)

// These are fixed constants, not generated via uuid.New() — every run must
// address the same fixture rows. The 4/8 nibbles (version 4, RFC 4122
// variant) keep them valid UUIDs by format, not just by accident of
// google/uuid and Postgres both accepting arbitrary 128-bit values.
var (
	OrganizationID       = uuid.MustParse("00000000-0000-4000-8000-000000000001")
	ObjectiveID          = uuid.MustParse("00000000-0000-4000-8000-000000000002")
	DefinitionID         = uuid.MustParse("00000000-0000-4000-8000-000000000003")
	CapabilityID         = uuid.MustParse("00000000-0000-4000-8000-000000000004")
	PrincipalID          = uuid.MustParse("00000000-0000-4000-8000-000000000005")
	ApproverPrincipalID  = uuid.MustParse("00000000-0000-4000-8000-000000000006")
)

// GovernanceActionDispatch is the Action capability.md requires every
// CapabilityDefinition to declare for its dispatch-time governance check.
const GovernanceActionDispatch = "capability.intelligence.dispatch"

// Registry exposes the first-slice fixtures by read-only accessor.
type Registry struct {
	org      organization.Organization
	obj      objective.Objective
	def      workflow.WorkflowDefinition
	cap      capability.CapabilityDefinition
	trig     principal.Principal
	approver principal.Principal
}

// NewRegistry builds the fixed first-slice fixture set, Organization
// included as a Go literal. Used by tests and fakes that don't have a real
// database to load from — cmd/companyd/main.go uses NewRegistryFromDB
// instead (ROADMAP.md Phase 3 Slice 6), since the DB-backed value the
// migration seeds is identical to this literal, switching every test to
// hit a real table wouldn't prove anything new.
func NewRegistry() Registry {
	return newFixedFixtures()
}

// NewRegistryFromDB builds the same fixture set as NewRegistry, except the
// Organization is loaded from the real organizations table instead of a Go
// literal (ROADMAP.md Phase 3 Slice 6 — "replace fixtures.Registry's
// hardcoded Organization with real persistence"). Fails loudly if the row
// is missing rather than silently falling back to the literal.
func NewRegistryFromDB(ctx context.Context, orgRepo ports.OrganizationRepository) (Registry, error) {
	reg := newFixedFixtures()
	org, err := orgRepo.GetOrganization(ctx, OrganizationID)
	if err != nil {
		return Registry{}, err
	}
	reg.org = org
	return reg, nil
}

func newFixedFixtures() Registry {
	return Registry{
		org: organization.Organization{
			OrganizationID: OrganizationID,
			Name:           "CompanyOS Dev",
			Status:         organization.StatusActive,
		},
		obj: objective.Objective{
			ObjectiveID:      ObjectiveID,
			OrganizationID:   OrganizationID,
			Title:            "Prove the Phase 1 vertical slice",
			Status:           objective.StatusApproved,
			OwnerPrincipalID: PrincipalID,
		},
		def: workflow.WorkflowDefinition{
			DefinitionID: DefinitionID,
			Version:      1,
			Name:         "first-slice-single-capability",
			States: []workflow.State{
				workflow.StatePlanned, workflow.StateReady,
				workflow.StateCompleted, workflow.StateFailed, workflow.StateCancelled,
			},
			Commands: []string{
				string(command.CreateWorkflow), string(command.StartWorkflow),
				string(command.AcceptWorkflowResult), string(command.RejectWorkflowResult),
				string(command.CancelWorkflow),
			},
			RequiredCapabilityID:      CapabilityID,
			RequiredCapabilityVersion: 1,
			TerminalStates: []workflow.State{
				workflow.StateCompleted, workflow.StateFailed, workflow.StateCancelled,
			},
			Active: true,
		},
		cap: capability.CapabilityDefinition{
			CapabilityDefinitionID: CapabilityID,
			Version:                1,
			Name:                   "bounded-text-generation",
			InputSchema:            map[string]string{"prompt": "string", "maxOutputTokens": "int"},
			OutputSchema:           map[string]string{"text": "string"},
			Timeout:                30 * time.Second,
			Retryable:              true,
			RiskLevel:              "low",
			GovernanceAction:       GovernanceActionDispatch,
			Retry: capability.RetryPolicy{
				MaxAttempts: 3,
				BackoffBase: 2 * time.Second,
				BackoffMax:  30 * time.Second,
				FullJitter:  true,
			},
			Active: true,
		},
		trig: principal.Principal{
			PrincipalID:    PrincipalID,
			OrganizationID: OrganizationID,
			Kind:           principal.KindService,
			DisplayName:    "web-trigger",
		},
		// approver stands in for a real, distinct authenticated human
		// reviewer (docs/domain/approval.md: "Only authenticated, authorized
		// human Principals can approve or reject") — Kind HUMAN and a
		// different PrincipalID than trig (a Service principal) makes
		// approval.md's self-approval prohibition checkable this slice,
		// the same way trig stands in for real Identity evidence.
		approver: principal.Principal{
			PrincipalID:    ApproverPrincipalID,
			OrganizationID: OrganizationID,
			Kind:           principal.KindHuman,
			DisplayName:    "human-approver",
		},
	}
}

func (r Registry) Organization() organization.Organization       { return r.org }
func (r Registry) Objective() objective.Objective                { return r.obj }
func (r Registry) WorkflowDefinition() workflow.WorkflowDefinition { return r.def }
func (r Registry) Capability() capability.CapabilityDefinition   { return r.cap }
func (r Registry) TriggerPrincipal() principal.Principal         { return r.trig }
func (r Registry) ApproverPrincipal() principal.Principal        { return r.approver }

// TriggerEvidence is the stub (unverified) AuthenticatedPrincipalEvidence
// used as every command's requester this slice (decision #5 — real
// Supabase-Auth JWT verification is deferred to Phase 2).
func (r Registry) TriggerEvidence() principal.Evidence {
	return principal.Evidence{PrincipalID: PrincipalID, Verified: true, Source: "fixture"}
}
