-- Widen governance_decisions.outcome to accept the renamed Decision
-- vocabulary (docs/adr/ADR-0010-authority-model-formalization.md):
-- AUTOMATIC/HUMAN_ONLY/DENIED alongside the original ALLOW/DENY/
-- REQUIRE_APPROVAL. Additive only — old values stay valid and no existing
-- row is rewritten. governance.md's own "audit records are append-only
-- organizational evidence" invariant means historical rows keep their
-- original ALLOW/DENY spelling exactly as recorded; only new rows use the
-- new vocabulary (AUTOMATIC/HUMAN_ONLY/DENIED), and REQUIRE_APPROVAL is
-- unchanged in both.
ALTER TABLE governance_decisions DROP CONSTRAINT governance_decisions_outcome_check;
ALTER TABLE governance_decisions ADD CONSTRAINT governance_decisions_outcome_check
  CHECK (outcome IN ('ALLOW', 'DENY', 'REQUIRE_APPROVAL', 'AUTOMATIC', 'HUMAN_ONLY', 'DENIED'));
