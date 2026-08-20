// Package ports declares the interfaces Application, Kernel, and Runtime
// consume, and that internal/adapters implements against concrete
// infrastructure. See docs/architecture/persistence.md#required-persistence-ports.
package ports

// TODO: define AuthoritativeStateRepository, ExecutionRepository,
// EventRepository/Outbox, ArtifactRepository, EvidenceRepository,
// KnowledgeRepository, and CommunicationRepository interfaces here,
// per docs/architecture/persistence.md. Concrete implementations live
// under internal/adapters/persistence/supabase.
