package ports

import (
	"context"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/objective"
	"github.com/google/uuid"
)

// ObjectiveRepository persists proposed Objectives (ROADMAP.md Phase 4
// Slice 4). No compare-and-swap/version machinery — an Objective proposal
// creates a brand new row, never transitions an existing one, so this
// slice's flow is linear, the same reasoning ports.ResearchRepository's
// doc comment gives.
type ObjectiveRepository interface {
	// CreateObjective persists a new Objective. closePendingCommand and
	// consumeApproval are non-nil only when called from the
	// REQUIRE_APPROVAL resume path (internal/application's
	// resumeProposeObjective) — when set, the implementation closes that
	// PendingCommand/Approval in the same transaction as the insert,
	// mirroring ports.AuthoritativeStateRepository.CommitTransition's
	// closePendingCommand/consumeApproval parameters.
	CreateObjective(ctx context.Context, o *objective.Objective, closePendingCommand *uuid.UUID, consumeApproval *uuid.UUID) error

	GetObjective(ctx context.Context, organizationID, objectiveID uuid.UUID) (objective.Objective, error)

	// GetObjectiveBySource is the dedup lookup docs/architecture/departments.md
	// requires ("duplicate feedback... cannot create another... Objective")
	// — ErrNotFound means no Objective has been proposed from this exact
	// source yet.
	GetObjectiveBySource(ctx context.Context, organizationID uuid.UUID, sourceType objective.SourceType, sourceID uuid.UUID) (objective.Objective, error)
}
