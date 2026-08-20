# Artifact Domain

Status: APPROVED

## Definition

An `Artifact` is an addressable, durable work product such as a document, patch, design, dataset, build, report, or deployment manifest. Storage does not imply acceptance, correctness, approval, or Knowledge status.

## Minimum contract

An Artifact contains:

- stable artifact identity, organization, artifact type, and version;
- immutable content digest, media type, size, and storage reference;
- producing Principal, department, capability, workflow, execution, and Result references when applicable;
- created-at timestamp and tool/provider/version provenance;
- parent, input, derived-from, and supersession relationships;
- security classification, access scope, retention, licensing, and deletion constraints;
- lifecycle status: `CANDIDATE`, `ACCEPTED`, `REJECTED`, or `SUPERSEDED`.

Acceptance is a separate governed domain transition over the exact artifact version and digest. Mutable external resources are captured by immutable version or digest before authoritative use.

## Invariants

- Artifact identity/version resolves to exactly one content digest.
- New content creates a new version; history and lineage are never overwritten.
- Artifact status cannot authorize execution or substitute for Result, Evidence, Metric, or Knowledge.
- Missing or inaccessible content makes dependent validation fail explicitly.
- Provider paths, URLs, and workspace files are locations, not stable Artifact identity.

## OPEN QUESTIONS

- Which artifact types and retention classes are required for the first slice?
- Which content is stored directly versus referenced through an external object port?

## Dependencies

- [Principal](principal.md)
- [Organization](organization.md)
