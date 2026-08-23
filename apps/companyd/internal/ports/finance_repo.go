package ports

import (
	"context"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/finance"
	"github.com/google/uuid"
)

// FinanceRepository persists Finance's core contracts (ROADMAP.md Phase 4
// Slice 3).
type FinanceRepository interface {
	CreateBudget(ctx context.Context, b *finance.Budget) error
	GetBudgetBySubject(ctx context.Context, organizationID, subjectID uuid.UUID) (finance.Budget, error)

	CreateCostConstraint(ctx context.Context, c *finance.CostConstraint) error
	// GetCostConstraintBySubject returns the most recently created
	// CostConstraint for the subject — no versioning/supersession-link
	// machinery this slice.
	GetCostConstraintBySubject(ctx context.Context, organizationID, subjectID uuid.UUID) (finance.CostConstraint, error)

	// GetPriceProfile looks up the migration-seeded PriceProfile by
	// ProviderAdapter only (see finance.PriceProfile's doc comment).
	GetPriceProfile(ctx context.Context, providerAdapter string) (finance.PriceProfile, error)

	RecordResourceUsage(ctx context.Context, u *finance.ResourceUsage) error
	GetResourceUsageBySubject(ctx context.Context, organizationID, subjectID uuid.UUID) ([]finance.ResourceUsage, error)

	CreateResourceEvaluation(ctx context.Context, e *finance.ResourceEvaluation) error
	GetResourceEvaluation(ctx context.Context, organizationID, evaluationID uuid.UUID) (finance.ResourceEvaluation, error)
}
