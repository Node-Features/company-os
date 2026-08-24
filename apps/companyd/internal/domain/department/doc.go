// Package department is reserved for the canonical Department domain
// contract (docs/domain/department.md, Status: APPROVED) but has no
// implementation yet — no DepartmentDefinition, DepartmentRegistry, or
// DepartmentMembership type exists here or anywhere else in this codebase
// (docs/audit/backlog-p2-p4.md's "doc-only stubs" row). The real pattern
// today is three independent, hand-duplicated package sets —
// internal/domain/{research,monitoringevaluation,finance} plus
// internal/departments/{research,monitoringevaluation,finance} — with no
// shared registry or base type binding them together (ROADMAP.md Phase 11
// Slice 0 is the first slice that would build one).
package department
