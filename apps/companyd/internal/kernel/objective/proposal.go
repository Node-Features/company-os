// Package objective is the Objective department's Kernel-equivalent
// (ROADMAP.md Phase 4 Slice 4): pure state-legality functions only, no I/O
// — the same role internal/kernel/workflow plays for Workflow. See
// docs/architecture/kernel.md: "Kernel owns legal transitions, not
// authorization."
package objective

import (
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/command"
)

// proposalTTL is longer than kernel/workflow's 5-minute commandTTL since
// this proposal awaits human approval rather than immediate dispatch — an
// implementation-detail default, not doctrine.
const proposalTTL = 24 * time.Hour

func newProposal(cmd command.ObjectiveProposalCommandEnvelope) *command.GovernedCommandProposal {
	args := map[string]any{
		"sourceType": cmd.SourceType,
		"sourceId":   cmd.SourceID,
		"title":      cmd.Title,
		"intent":     cmd.Intent,
	}
	p := &command.GovernedCommandProposal{
		ProposalID:           cmd.CommandID,
		SchemaVersion:        1,
		CommandID:            cmd.CommandID,
		Action:               command.ActionFor[cmd.CommandType],
		ResourceType:         "Objective",
		ResourceID:           cmd.ObjectiveID.String(),
		OrganizationID:       cmd.OrganizationID,
		Arguments:            args,
		TrustedContextDigest: canonicalDigest(map[string]any{"org": cmd.OrganizationID, "principal": cmd.RequestingPrincipalID, "time": cmd.DeclaredTime}),
		EffectClassification: "governed-state-change",
		ExpiresAt:            cmd.DeclaredTime.Add(proposalTTL),
	}
	p.CommandDigest = canonicalDigest(cmd)
	p.ProposalDigest = canonicalDigest(struct {
		Action, ResourceType, ResourceID string
		Args                             map[string]any
		CommandDigest                    string
	}{p.Action, p.ResourceType, p.ResourceID, args, p.CommandDigest})
	return p
}

// ValidateProposal is Objective-creation Kernel legality: state legality
// only (does an Objective already exist for this exact source —
// docs/architecture/departments.md: "duplicate feedback... cannot create
// another... Objective"), not authorization. alreadyProposed is a
// caller-resolved fact (Application looks it up via
// ports.ObjectiveRepository.GetObjectiveBySource before calling this),
// mirroring how kernel/workflow's Validate*Proposal functions take
// already-loaded state rather than doing persistence lookups themselves.
func ValidateProposal(cmd command.ObjectiveProposalCommandEnvelope, alreadyProposed bool) (*command.GovernedCommandProposal, []string) {
	if alreadyProposed {
		return nil, []string{command.ReasonObjectiveAlreadyProposed}
	}
	return newProposal(cmd), nil
}
