package receivables

import "testing"

func TestReceivableModelMapsMissingInvoiceAsNull(t *testing.T) {
	model := toModel(DeliveryReceivable{})
	if model.InvoiceID != nil {
		t.Fatal("missing invoice ID should map to NULL")
	}
	if model.InvoiceNumber != nil {
		t.Fatal("missing invoice number should map to NULL")
	}
}

func TestReceivableModelMapsInvoiceNumberValue(t *testing.T) {
	model := toModel(DeliveryReceivable{InvoiceID: 7, InvoiceNumber: "INV007"})
	if model.InvoiceID == nil || *model.InvoiceID != 7 {
		t.Fatalf("invoice ID = %v, want 7", model.InvoiceID)
	}
	if model.InvoiceNumber == nil || *model.InvoiceNumber != "INV007" {
		t.Fatalf("invoice number = %v, want INV007", model.InvoiceNumber)
	}
}

func TestReceivableModelMapsNullInvoiceNumberAsEmpty(t *testing.T) {
	receivable := (receivableModel{}).toDomain()
	if receivable.InvoiceNumber != "" {
		t.Fatalf("invoice number = %q, want empty", receivable.InvoiceNumber)
	}
}
