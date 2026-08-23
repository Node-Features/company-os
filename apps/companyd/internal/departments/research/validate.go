// Package research implements the Research department contract. It
// must import only internal/domain and internal/ports packages, never
// another department's package. See docs/departments/research.md.
//
// This file is Research's Kernel-equivalent (docs/workflows/research-loop.md):
// pure legality functions only, mirroring internal/kernel/workflow's shape.
// Orchestration (Governance evaluation, persistence) lives in
// internal/application/research_*.go, exactly as internal/kernel/workflow's
// pure functions are orchestrated by internal/application/workflow_*.go.
package research

import (
	"strings"

	domain "github.com/Node-Features/company-os/apps/companyd/internal/domain/research"
	"github.com/google/uuid"
)

// Reason strings for validation failures — mirrors command.Reason* constants'
// role for Workflow.
const (
	ReasonMissingDescription = "missing_description"
	ReasonUnknownSourceType  = "unknown_source_type"
	ReasonMissingText        = "missing_text"
	ReasonQuestionNotOpen    = "question_not_open"
	ReasonMissingSource      = "missing_source"
	ReasonMissingContent     = "missing_content"
	ReasonMissingClaim       = "missing_claim"
	ReasonNoEvidenceCited    = "no_evidence_cited"
	ReasonFindingSuperseded  = "finding_superseded"
	ReasonMissingAction      = "missing_action"
)

// ValidateSubmitSignal checks a Signal is well-formed before persistence.
// This slice handles exactly one SourceType (research-loop.md).
func ValidateSubmitSignal(sourceType, description string) []string {
	var reasons []string
	if sourceType != domain.SourceTypeProviderModelChange {
		reasons = append(reasons, ReasonUnknownSourceType)
	}
	if strings.TrimSpace(description) == "" {
		reasons = append(reasons, ReasonMissingDescription)
	}
	return reasons
}

// ValidateOpenQuestion checks a ResearchQuestion can be opened from sig.
func ValidateOpenQuestion(sig domain.Signal, text string) []string {
	var reasons []string
	if strings.TrimSpace(text) == "" {
		reasons = append(reasons, ReasonMissingText)
	}
	return reasons
}

// ValidateRecordEvidence checks Evidence can be recorded against q — the
// question must still be OPEN (research.md: findings/evidence collection
// happens while a question is active).
func ValidateRecordEvidence(q domain.ResearchQuestion, source, content string) []string {
	var reasons []string
	if q.Status != domain.QuestionOpen {
		reasons = append(reasons, ReasonQuestionNotOpen)
	}
	if strings.TrimSpace(source) == "" {
		reasons = append(reasons, ReasonMissingSource)
	}
	if strings.TrimSpace(content) == "" {
		reasons = append(reasons, ReasonMissingContent)
	}
	return reasons
}

// ValidatePublishFinding enforces research.md's invariant that "a finding
// cannot exceed what its evidence supports" at the one point this slice can
// mechanically check it: at least one cited Evidence record must exist. It
// cannot verify the claim is actually supported by that evidence — that
// judgment is the publishing Principal's, same as every other slice's
// "don't validate what can't be mechanically checked" convention.
func ValidatePublishFinding(q domain.ResearchQuestion, claim string, evidenceIDs []uuid.UUID) []string {
	var reasons []string
	if q.Status != domain.QuestionOpen {
		reasons = append(reasons, ReasonQuestionNotOpen)
	}
	if strings.TrimSpace(claim) == "" {
		reasons = append(reasons, ReasonMissingClaim)
	}
	if len(evidenceIDs) == 0 {
		reasons = append(reasons, ReasonNoEvidenceCited)
	}
	return reasons
}

// ValidateIssueRecommendation checks a Recommendation can be issued from f —
// f must not already be superseded.
func ValidateIssueRecommendation(f domain.Finding, proposedAction string) []string {
	var reasons []string
	if f.Status == domain.FindingSuperseded {
		reasons = append(reasons, ReasonFindingSuperseded)
	}
	if strings.TrimSpace(proposedAction) == "" {
		reasons = append(reasons, ReasonMissingAction)
	}
	return reasons
}
