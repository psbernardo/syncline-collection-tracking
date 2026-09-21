package receivables

import (
	"bytes"
	"testing"
	"time"
)

func TestReceivablesPDFRendererProducesReport(t *testing.T) {
	now := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	report := buildExportReport([]DeliveryReceivable{
		{ID: 1, CompanyAccountID: 1, CompanyName: "Acme Corp", InvoiceNumber: "INV1", PONumber: "PO1", DeliveryDateUTC: time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC), DueDateUTC: time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC), PaymentTermDays: 5, AmountDue: 1000, GrossAmount: 2000, LifecycleStatus: "Active"},
		{ID: 2, CompanyAccountID: 2, CompanyName: "Beta Inc", InvoiceNumber: "INV2", PONumber: "PO2", DeliveryDateUTC: time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC), DueDateUTC: time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC), PaymentTermDays: 5, AmountDue: 3000, GrossAmount: 6000, LifecycleStatus: "Active"},
	}, ListQuery{Now: now}, now)
	var output bytes.Buffer
	if err := NewReceivablesPDFRenderer().Render(&output, report); err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(output.Bytes(), []byte("%PDF-")) {
		t.Fatal("receivables renderer did not produce a PDF")
	}
	if !bytes.HasSuffix(bytes.TrimSpace(output.Bytes()), []byte("%%EOF")) {
		t.Fatal("receivables PDF is missing the EOF marker")
	}
}

func TestReceivablesPDFRendererPaginatesLongReports(t *testing.T) {
	now := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	receivables := make([]DeliveryReceivable, 0, 70)
	for index := 0; index < 70; index++ {
		receivables = append(receivables, DeliveryReceivable{
			ID: int64(index + 1), CompanyAccountID: 1, CompanyName: "Acme Corp", InvoiceNumber: "INV", PONumber: "PO",
			DeliveryDateUTC: now, DueDateUTC: now, PaymentTermDays: 5, AmountDue: 1000, GrossAmount: 2000, LifecycleStatus: "Active",
		})
	}
	report := buildExportReport(receivables, ListQuery{Now: now}, now)
	var output bytes.Buffer
	if err := NewReceivablesPDFRenderer().Render(&output, report); err != nil {
		t.Fatal(err)
	}
	pages := bytes.Count(output.Bytes(), []byte("/Type /Page")) - bytes.Count(output.Bytes(), []byte("/Type /Pages"))
	if pages < 2 {
		t.Fatalf("expected long report to span multiple pages, got %d", pages)
	}
}
