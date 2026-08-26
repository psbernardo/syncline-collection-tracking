package purchaseorders

import (
	"bytes"
	"testing"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
)

func TestPurchaseOrderPDFDocumentUsesProcurementSnapshot(t *testing.T) {
	delivery := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	order := PurchaseOrder{
		Number: "PO-20260823-0001", SupplierName: "Acme Supplier", Status: "CONFIRMED",
		PODate: time.Date(2026, 8, 23, 0, 0, 0, 0, time.UTC), PaymentTerms: "Net 30",
		ExpectedDelivery: &delivery, SalesOrderNumbers: []string{"SO-00000009", "SO-00000010"},
		Notes: "Deliver to warehouse", Subtotal: money.Amount(123456), Total: money.Amount(123456),
		Lines: []Line{{SKU: "SKU-1", Name: "Widget", SupplierSKU: "SUP-1", UOM: "PC", Quantity: 20000, UnitCost: 50000, Total: 100000}},
	}
	document := purchaseOrderPDFDocument(order)
	if document.Supplier.Name != "Acme Supplier" || document.Number != order.Number || document.Total != order.Total {
		t.Fatalf("unexpected document mapping: %+v", document)
	}
	if got := purchaseOrderSource(document.SourceReferences); got != "SO-00000009, SO-00000010" {
		t.Fatalf("unexpected source display: %q", got)
	}
	if len(document.Lines) != 1 || document.Lines[0].UnitCost != money.Amount(50000) {
		t.Fatalf("unexpected line mapping: %+v", document.Lines)
	}
}

func TestPurchaseOrderPDFRendererPaginatesLongOrders(t *testing.T) {
	document := PurchaseOrderPDFDocument{
		Number: "PO-20260823-0002", Supplier: SupplierDetails{Name: "Supplier"},
		Lines: make([]PurchaseOrderPDFLine, 35),
	}
	for index := range document.Lines {
		document.Lines[index] = PurchaseOrderPDFLine{Name: "Purchased product", SKU: "SKU-1", UOM: "PC", Quantity: 10000}
	}
	var output bytes.Buffer
	if err := NewPurchaseOrderPDFRenderer().Render(&output, document); err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(output.Bytes(), []byte("%PDF-")) {
		t.Fatal("purchase-order renderer did not produce a PDF")
	}
	pages := bytes.Count(output.Bytes(), []byte("/Type /Page")) - bytes.Count(output.Bytes(), []byte("/Type /Pages"))
	if pages < 2 {
		t.Fatalf("expected long purchase order to span multiple pages, got %d", pages)
	}
}
