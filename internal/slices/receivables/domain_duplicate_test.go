package receivables

import "testing"

func TestNormalizePONumber(t *testing.T) {
	if got := NormalizePONumber("  po001  "); got != "PO001" {
		t.Fatalf("NormalizePONumber() = %q, want PO001", got)
	}
}

func TestNewDeliveryReceivableRejectsNonAlphanumericPO(t *testing.T) {
	_, err := NewDeliveryReceivable(1, "PO-001", "100", "2026-08-10", 5)
	validation, ok := err.(ValidationErrors)
	if !ok || validation["PONumber"] == "" {
		t.Fatalf("expected PO validation error, got %v", err)
	}
}
