-- Phase 5 Slice 1: KnowledgeItem ingestion and versioning for one source
-- type (docs/architecture/knowledge.md steps 1-4, docs/domain/knowledge.md).
-- Rows are append-only per version: knowledge_item_id is stable across
-- versions, each new normalization of the same source is a new row with an
-- incremented version, never an overwrite ("immutable version").
--
-- source_id is deliberately not FK'd — it is polymorphic across future
-- source types (only RESEARCH_FINDING exists this slice), the same
-- reasoning objectives.source_id already established for its own
-- polymorphic source. No FK to principals either, same reasoning as prior
-- migrations. RLS enabled with zero policies, matching the existing schema
-- convention.

CREATE TABLE knowledge_items (
  knowledge_item_id uuid NOT NULL,
  organization_id uuid NOT NULL,
  version int NOT NULL,
  claim text NOT NULL,
  content_digest text NOT NULL,
  classification text NOT NULL CHECK (classification IN ('INTERNAL')),
  source_type text NOT NULL CHECK (source_type IN ('RESEARCH_FINDING')),
  source_id uuid NOT NULL,
  produced_by_principal_id uuid NOT NULL,
  produced_by_method text NOT NULL CHECK (produced_by_method IN ('SOURCE_VERBATIM')),
  status text NOT NULL CHECK (status IN ('DRAFT', 'IN_REVIEW', 'APPROVED', 'REJECTED', 'SUPERSEDED', 'EXPIRED')),
  duplicate_of_item_id uuid,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (knowledge_item_id, version)
);
CREATE INDEX knowledge_items_source_idx ON knowledge_items (organization_id, source_type, source_id);
CREATE INDEX knowledge_items_content_digest_idx ON knowledge_items (organization_id, content_digest);
ALTER TABLE knowledge_items ENABLE ROW LEVEL SECURITY;
