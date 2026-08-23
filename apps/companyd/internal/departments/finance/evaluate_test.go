package finance

import (
	"testing"

	domain "github.com/Node-Features/company-os/apps/companyd/internal/domain/finance"
)

func TestComputeUsageCost(t *testing.T) {
	p := domain.PriceProfile{InputPricePerKTokens: 0.10, OutputPricePerKTokens: 0.40}
	got := ComputeUsageCost(p, 1000, 500)
	want := 0.30
	if diff := got - want; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("ComputeUsageCost = %v, want %v", got, want)
	}
}

func TestEvaluateConstraintOutcome(t *testing.T) {
	constraint := &domain.CostConstraint{MaxCost: 10.0}
	tests := []struct {
		name           string
		cumulativeCost float64
		constraint     *domain.CostConstraint
		want           domain.ConstraintOutcome
	}{
		{"no constraint", 5, nil, domain.NotApplicable},
		{"well within", 1, constraint, domain.WithinLimit},
		{"at warning threshold", 8, constraint, domain.ApproachingLimit},
		{"at limit exactly", 10, constraint, domain.ApproachingLimit},
		{"over limit", 10.01, constraint, domain.LimitExceeded},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluateConstraintOutcome(tt.cumulativeCost, tt.constraint)
			if got != tt.want {
				t.Fatalf("EvaluateConstraintOutcome(%v) = %s, want %s", tt.cumulativeCost, got, tt.want)
			}
		})
	}
}

func TestComputeEffectiveCostPerSuccess(t *testing.T) {
	if got := ComputeEffectiveCostPerSuccess(10, 0); got != nil {
		t.Fatalf("ComputeEffectiveCostPerSuccess(10, 0) = %v, want nil", *got)
	}
	got := ComputeEffectiveCostPerSuccess(10, 4)
	if got == nil || *got != 2.5 {
		t.Fatalf("ComputeEffectiveCostPerSuccess(10, 4) = %v, want 2.5", got)
	}
}
