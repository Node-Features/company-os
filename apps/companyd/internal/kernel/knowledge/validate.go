package knowledge

// ValidateCapture is the state-legality check for capturing a new
// KnowledgeItem candidate version: reject only when the source's content is
// unchanged since the latest known version for that source (no point
// creating an identical duplicate version) — everything else about whether
// this capture should proceed is Governance's job (knowledge.item.capture,
// AUTOMATIC autonomy this slice), not Kernel's.
//
// hasLatest/latestVersion/latestContentDigest are already-resolved facts
// the caller looks up (mirrors kernel/objective.ValidateProposal's
// alreadyProposed parameter) — this function does no I/O.
func ValidateCapture(hasLatest bool, latestVersion int, latestContentDigest, newContentDigest string) (nextVersion int, reasons []string) {
	if !hasLatest {
		return 1, nil
	}
	if latestContentDigest == newContentDigest {
		return 0, []string{"content_unchanged_since_last_capture"}
	}
	return latestVersion + 1, nil
}
