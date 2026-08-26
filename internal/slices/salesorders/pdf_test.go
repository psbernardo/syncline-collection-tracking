package salesorders

import (
	"bytes"
	"testing"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
)

func TestSalesOrderPDFRendererUsesOrderDocumentContract(t *testing.T) {
	document := SalesOrderPDFDocument{
		Number: "SO-00000003", Source: "QT-00000024", CustomerPO: "CUSTOMER-PO-2026-000001",
		SalesPerson: "Alma Mae Bernardo", Customer: "Customer", BillingAddress: "Billing address", DeliveryAddress: "Delivery address",
		TermsDays: 30, OrderDate: time.Date(2026, time.August, 23, 0, 0, 0, 0, time.UTC),
		Lines: []SalesOrderPDFLine{{SKU: "SKU-1", Name: "Order product", UOM: "PC", Quantity: money.Amount(20000), UnitPrice: money.Amount(10000), Amount: money.Amount(22400), TaxRate: 120000}},
	}
	var output bytes.Buffer
	if err := NewSalesOrderPDFRenderer().Render(&output, document); err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(output.Bytes(), []byte("%PDF-")) {
		t.Fatal("sales order renderer did not produce a PDF")
	}
}
