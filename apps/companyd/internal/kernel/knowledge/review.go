package knowledge

import (
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/command"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/knowledge"
)

// reviewTTL is longer than kernel/workflow's 5-minute commandTTL since this
// proposal awaits human approval rather than immediate dispatch — same
// reasoning as kernel/objective's proposalTTL.
const reviewTTL = 24 * time.Hour

func newReviewProposal(cmd command.KnowledgeApprovalCommandEnvelope) *command.GovernedCommandProposal {
	args := map[string]any{
		"knowledgeItemId": cmd.KnowledgeItemID,
		"version":         cmd.Version,
		"contentDigest":   cmd.ContentDigest,
	}
	p := &command.GovernedCommandProposal{
		ProposalID:           cmd.CommandID,
		SchemaVersion:        1,
		CommandID:            cmd.CommandID,
		Action:               command.ActionFor[cmd.CommandType],
		ResourceType:         "KnowledgeItem",
		ResourceID:           cmd.KnowledgeItemID.String(),
		OrganizationID:       cmd.OrganizationID,
		Arguments:            args,
		TrustedContextDigest: proposalDigest(map[string]any{"org": cmd.OrganizationID, "principal": cmd.RequestingPrincipalID, "time": cmd.DeclaredTime}),
		EffectClassification: "governed-state-change",
		ExpiresAt:            cmd.DeclaredTime.Add(reviewTTL),
	}
	p.CommandDigest = proposalDigest(cmd)
	p.ProposalDigest = proposalDigest(struct {
		Action, ResourceType, ResourceID string
		Args                             map[string]any
		CommandDigest                    string
	}{p.Action, p.ResourceType, p.ResourceID, args, p.CommandDigest})
	return p
}

// ValidateApprovalRequest is knowledge.approve's Kernel legality check
// (docs/architecture/kernel.md: "Kernel owns legal transitions, not
// authorization" — separation-of-duties, the authorization half, is
// Governance's job, see internal/governance/evaluate.go's
// ExcludedPrincipalID). found/currentStatus/currentVersion/currentContentDigest
// are already-resolved facts the caller looks up (mirrors
// kernel/objective.ValidateProposal's alreadyProposed parameter) — this
// function does no I/O.
//
// Three rejection cases, in order: the item doesn't exist; it isn't
// currently DRAFT (only a fresh, undecided candidate may be submitted —
// this is also what rejects a second concurrent review request, and what
// rejects a stale resume if the item changed since the original request);
// the caller's asserted version/digest doesn't match the current one
// exactly (docs/architecture/knowledge.md: "exact item-version/content
// digest" verification — "a stale item version... requires a new
// Governance evaluation").
func ValidateApprovalRequest(found bool, currentStatus knowledge.Status, currentVersion int, currentContentDigest string, requestedVersion int, requestedContentDigest string, cmd command.KnowledgeApprovalCommandEnvelope) (*command.GovernedCommandProposal, []string) {
	if !found {
		return nil, []string{"knowledge_item_not_found"}
	}
	if currentStatus != knowledge.StatusDraft {
		return nil, []string{"knowledge_item_not_in_draft_status"}
	}
	if currentVersion != requestedVersion || currentContentDigest != requestedContentDigest {
		return nil, []string{"knowledge_item_stale_version_or_digest"}
	}
	return newReviewProposal(cmd), nil
}
