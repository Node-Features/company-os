-- Phase 5 Slice 2: knowledge.review/knowledge.approve governed action
-- (docs/architecture/knowledge.md steps 5-7). Adds the reviewer/decision
-- fields Slice 1 deliberately left off knowledge_items — no CHECK
-- constraint change needed, Slice 1's status column already allows
-- 'APPROVED'/'REJECTED'.

ALTER TABLE knowledge_items
  ADD COLUMN reviewer_principal_id uuid,
  ADD COLUMN governance_decision_id uuid,
  ADD COLUMN approval_id uuid,
  ADD COLUMN reviewed_at timestamptz;
