package knowledge

import "testing"

func TestValidateCapture_NoPriorVersion_AcceptsVersion1(t *testing.T) {
	nextVersion, reasons := ValidateCapture(false, 0, "", "digest-a")
	if len(reasons) != 0 {
		t.Fatalf("expected no rejection reasons, got %v", reasons)
	}
	if nextVersion != 1 {
		t.Fatalf("expected version 1, got %d", nextVersion)
	}
}

func TestValidateCapture_UnchangedDigest_Rejected(t *testing.T) {
	nextVersion, reasons := ValidateCapture(true, 3, "digest-a", "digest-a")
	if nextVersion != 0 {
		t.Fatalf("expected version 0 on rejection, got %d", nextVersion)
	}
	if len(reasons) != 1 || reasons[0] != "content_unchanged_since_last_capture" {
		t.Fatalf("expected content_unchanged_since_last_capture reason, got %v", reasons)
	}
}

func TestValidateCapture_ChangedDigest_IncrementsVersion(t *testing.T) {
	nextVersion, reasons := ValidateCapture(true, 3, "digest-a", "digest-b")
	if len(reasons) != 0 {
		t.Fatalf("expected no rejection reasons, got %v", reasons)
	}
	if nextVersion != 4 {
		t.Fatalf("expected version 4, got %d", nextVersion)
	}
}
