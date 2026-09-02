package invoices

import (
	"bytes"
	"testing"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
)

func TestInvoicePDFRendererUsesInvoiceDocumentContract(t *testing.T) {
	document := InvoicePDFDocument{
		Number: "INV-00000003", SalesOrderNumber: "SO-00000003", CustomerPO: "PO-3",
		Customer: "Customer", BillingAddress: "Billing address", DeliveryAddress: "Delivery address",
		TermsDays: 30, InvoiceDate: time.Date(2026, time.August, 23, 0, 0, 0, 0, time.UTC), DueDate: time.Date(2026, time.September, 22, 0, 0, 0, 0, time.UTC),
		Lines: []InvoicePDFLine{{SKU: "SKU-1", Name: "Invoiced product", UOM: "PC", Quantity: money.Amount(20000), UnitPrice: money.Amount(10000), Amount: money.Amount(20000), TaxRate: 120000}},
	}
	var output bytes.Buffer
	if err := NewInvoicePDFRenderer().Render(&output, document); err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(output.Bytes(), []byte("%PDF-")) {
		t.Fatal("invoice renderer did not produce a PDF")
	}
}

func TestInvoiceDescriptionMatchesQuotationTemplate(t *testing.T) {
	line := InvoicePDFLine{SKU: "SKU-1", Name: "Invoiced product"}
	if got := invoiceDescription(line); got != "Invoiced product" {
		t.Fatalf("expected quotation-style product description, got %q", got)
	}
}

func TestInvoiceTotalsUseComputedAmountsAndLeaveUnavailableValuesBlank(t *testing.T) {
	subtotal := money.Amount(12500000)
	tax := money.Amount(1500000)
	total := money.Amount(14000000)
	document := InvoicePDFDocument{Subtotal: subtotal, Tax: tax, Total: total, VatableSales: &subtotal, TotalAmountDue: &total}

	if got := formatOptionalAmount(document.VatableSales); got != "PHP 1,250.00" {
		t.Fatalf("vatable sales = %q", got)
	}
	if got := formatOptionalAmount(amountPointer(document.Tax)); got != "PHP 150.00" {
		t.Fatalf("VAT = %q", got)
	}
	if got := formatOptionalAmount(document.Discount); got != "" {
		t.Fatalf("blank discount = %q, want blank", got)
	}
	if got := formatOptionalAmount(document.WithholdingTax); got != "" {
		t.Fatalf("blank withholding tax = %q, want blank", got)
	}
	if got := formatOptionalAmount(nonZeroAmountPointer(0)); got != "" {
		t.Fatalf("zero VAT = %q, want blank", got)
	}
}

func TestInvoicePDFRendererPaginatesLongInvoices(t *testing.T) {
	document := InvoicePDFDocument{
		Number: "INV-00000004", Customer: "Customer",
		Lines: make([]InvoicePDFLine, 30),
	}
	for index := range document.Lines {
		document.Lines[index] = InvoicePDFLine{Name: "Invoiced product", UOM: "PC", Quantity: money.Amount(10000)}
	}
	var output bytes.Buffer
	if err := NewInvoicePDFRenderer().Render(&output, document); err != nil {
		t.Fatal(err)
	}
	if got := bytes.Count(output.Bytes(), []byte("/Type /Page")) - bytes.Count(output.Bytes(), []byte("/Type /Pages")); got < 2 {
		t.Fatalf("expected long invoice to span multiple pages, got %d", got)
	}
}
