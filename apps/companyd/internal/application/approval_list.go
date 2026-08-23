package application

import (
	"context"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/approval"
)

// ListPendingApprovals is the Approval inbox's entry point (ROADMAP.md
// Phase 10 Slice 1) — every PENDING Approval for the organization, across
// every CommandType (CancelWorkflow/ProposeObjective/ApproveKnowledgeItem)
// at once, since PendingCommand/Approval are already generic across them.
// Deliberately takes no PrincipalID, matching GetKnowledgeItem/GetObjective/
// QueryKnowledge's existing precedent that reads in this codebase don't
// carry one.
func (a *Application) ListPendingApprovals(ctx context.Context) ([]approval.Approval, error) {
	orgID := a.Fixtures.Organization().OrganizationID
	return a.Pending.ListPendingApprovals(ctx, orgID)
}
