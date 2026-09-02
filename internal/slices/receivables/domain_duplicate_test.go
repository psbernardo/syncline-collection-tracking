package receivables

import (
	"errors"
	"testing"
)

func TestNormalizePONumber(t *testing.T) {
	if got := NormalizePONumber("  po001  "); got != "PO001" {
		t.Fatalf("NormalizePONumber() = %q, want PO001", got)
	}
}

func TestNewDeliveryReceivableRejectsNonAlphanumericPO(t *testing.T) {
	_, err := NewDeliveryReceivable(1, "0127", "PO-001", "100", "2026-08-10", 5)
	validation, ok := err.(ValidationErrors)
	if !ok || validation["PONumber"] == "" {
		t.Fatalf("expected PO validation error, got %v", err)
	}
}

func TestNewDeliveryReceivableRejectsInvalidInvoiceNumber(t *testing.T) {
	_, err := NewDeliveryReceivable(1, "INV-001", "PO001", "100", "2026-08-10", 5)
	validation, ok := err.(ValidationErrors)
	if !ok || validation["InvoiceNumber"] == "" {
		t.Fatalf("expected invoice number validation error, got %v", err)
	}
}

func TestNewDeliveryReceivableAllowsMissingInvoiceNumber(t *testing.T) {
	receivable, err := NewDeliveryReceivable(1, "", "PO001", "100", "2026-08-10", 5)
	if err != nil {
		t.Fatalf("expected missing invoice number to be allowed, got %v", err)
	}
	if receivable.InvoiceNumber != "" {
		t.Fatalf("invoice number = %q, want empty", receivable.InvoiceNumber)
	}
}

func TestNewDeliveryReceivableTrimsManualInvoiceNumber(t *testing.T) {
	receivable, err := NewDeliveryReceivable(1, "  INV001  ", "PO001", "100", "2026-08-10", 5)
	if err != nil {
		t.Fatal(err)
	}
	if receivable.InvoiceNumber != "INV001" {
		t.Fatalf("invoice number = %q, want INV001", receivable.InvoiceNumber)
	}
}

func TestDuplicateInvoiceErrorRecognized(t *testing.T) {
	if !isDuplicateInvoiceError(errors.New("violation UX_delivery_receivables_invoice_not_cancelled")) {
		t.Fatal("expected invoice unique-index error to be recognized")
	}
}
