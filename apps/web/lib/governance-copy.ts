// Plain-language summaries for a governed Approval's Action string
// (docs/architecture/ui-ux.md invariant 8). Deliberately does not fabricate
// a resource title (e.g. a Workflow/Objective name) — no domain type
// carries one today, so the summary names the *kind* of action truthfully
// and leaves the specific resource id to the technical-details toggle.
const SUMMARIES: Record<string, string> = {
  "workflow.cancel":
    "Cancel a running workflow — it's already dispatched, so cancelling it needs your sign-off.",
  "objective.propose":
    "Approve a proposed Decision before it becomes an active Objective.",
  "knowledge.review.request":
    "Approve a Knowledge item's review before it counts as established knowledge.",
};

export function approvalSummary(action: string): string {
  if (SUMMARIES[action]) return SUMMARIES[action];
  const humanized = action.replace(/[._]/g, " ");
  return `Approve "${humanized}" — this action needs your sign-off before it proceeds.`;
}
