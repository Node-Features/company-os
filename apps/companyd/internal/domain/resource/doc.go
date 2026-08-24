// Package resource is reserved for the canonical Resource domain contract
// (docs/domain/resource.md, Status: APPROVED) but has no implementation
// yet — no ResourceConstraint type or any of its five named
// specializations exist here (docs/audit/backlog-p2-p4.md's "doc-only
// stubs" row). The real, partial equivalent is
// internal/domain/finance.CostConstraint plus finance.Budget/
// PriceProfile/ResourceUsage/ResourceEvaluation — only one of the five
// specializations this contract names (CostConstraint); ComputeConstraint,
// TimeConstraint, StorageConstraint, and ConcurrencyConstraint have no
// implementation anywhere.
package resource
