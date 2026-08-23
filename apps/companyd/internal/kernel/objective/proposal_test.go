package objective

import (
	"testing"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/command"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/objective"
	"github.com/google/uuid"
)

func testEnvelope() command.ObjectiveProposalCommandEnvelope {
	return command.ObjectiveProposalCommandEnvelope{
		SchemaVersion:         1,
		CommandID:             uuid.New(),
		RequestID:             uuid.New(),
		IdempotencyKey:        "idem-1",
		CommandType:           command.ProposeObjective,
		OrganizationID:        uuid.New(),
		ObjectiveID:           uuid.New(),
		SourceType:            objective.SourceFinding,
		SourceID:              uuid.New(),
		Title:                 "Investigate cost regression",
		Intent:                "Reduce effective cost per successful result",
		RequestingPrincipalID: uuid.New(),
		DeclaredTime:          time.Now().UTC(),
		CorrelationID:         uuid.New(),
	}
}

func TestValidateProposal_AlreadyProposedRejected(t *testing.T) {
	proposal, reasons := ValidateProposal(testEnvelope(), true)
	if proposal != nil {
		t.Fatalf("proposal = %+v, want nil", proposal)
	}
	if len(reasons) != 1 || reasons[0] != command.ReasonObjectiveAlreadyProposed {
		t.Fatalf("reasons = %v, want [%s]", reasons, command.ReasonObjectiveAlreadyProposed)
	}
}

func TestValidateProposal_NotProposedAccepted(t *testing.T) {
	cmd := testEnvelope()
	proposal, reasons := ValidateProposal(cmd, false)
	if proposal == nil {
		t.Fatalf("proposal = nil, reasons = %v, want a proposal", reasons)
	}
	if proposal.Action != "objective.propose" {
		t.Fatalf("Action = %q, want objective.propose", proposal.Action)
	}
	if proposal.ResourceType != "Objective" || proposal.ResourceID != cmd.ObjectiveID.String() {
		t.Fatalf("ResourceType/ResourceID = %s/%s, want Objective/%s", proposal.ResourceType, proposal.ResourceID, cmd.ObjectiveID)
	}
	if proposal.ProposalDigest == "" || proposal.CommandDigest == "" || proposal.TrustedContextDigest == "" {
		t.Fatalf("digest fields not populated: %+v", proposal)
	}

	// Same cmd, evaluated again, reproduces the identical digest — a pure
	// function of its inputs, the property REQUIRE_APPROVAL replay depends
	// on (kernel/workflow/proposal.go's newProposal has the same property).
	proposal2, _ := ValidateProposal(cmd, false)
	if proposal2.ProposalDigest != proposal.ProposalDigest {
		t.Fatalf("ProposalDigest not stable across identical replay: %s != %s", proposal2.ProposalDigest, proposal.ProposalDigest)
	}
}
