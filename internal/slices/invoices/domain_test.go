package invoices

import "testing"

func TestValidateInvoiceNumber(t *testing.T) {
	value, err := ValidateInvoiceNumber("  INV-42  ")
	if err != nil || value != "INV-42" {
		t.Fatalf("unexpected invoice number: %q, error=%v", value, err)
	}
	if _, err := ValidateInvoiceNumber("   "); err == nil {
		t.Fatal("expected blank invoice number to fail")
	}
	if _, err := ValidateInvoiceNumber(string(make([]rune, 101))); err == nil {
		t.Fatal("expected long invoice number to fail")
	}
}
