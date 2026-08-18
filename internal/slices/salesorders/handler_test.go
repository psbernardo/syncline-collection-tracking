package salesorders

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/slices/quotations"
)

type quotationRepo struct{ value quotations.Quotation }

func (r quotationRepo) Create(context.Context, quotations.Quotation) (quotations.Quotation, error) {
	return r.value, nil
}
func (r quotationRepo) List(context.Context) ([]quotations.Quotation, error) { return nil, nil }
func (r quotationRepo) FindByID(context.Context, int64) (quotations.Quotation, error) {
	return r.value, nil
}
func (r quotationRepo) Update(context.Context, quotations.Quotation, []byte) (quotations.Quotation, error) {
	return r.value, nil
}

type orderRepo struct{ value SalesOrder }

func (r *orderRepo) CreateFromQuotation(_ context.Context, q quotations.Quotation, po string) (SalesOrder, error) {
	r.value = SalesOrder{ID: 9, Number: "SO-00000009", CustomerPONumber: po, SalesPerson: FixedSalesPerson, Status: Open, Quotation: q}
	r.value.Quotation.Number = r.value.Number
	return r.value, nil
}
func (r *orderRepo) FindByID(context.Context, int64) (SalesOrder, error) { return r.value, nil }

func testQuotation() quotations.Quotation {
	return quotations.Quotation{
		ID: 7, Number: "QT-00000007", CompanyAccountID: 2, CompanyName: "Customer", CreatedAtUTC: time.Now().UTC(), TermsDays: 30,
		Lines:  []quotations.Line{{ProductName: "Product", Quantity: 500000, UOM: "PC", UnitPrice: 10000, LineTotal: 500000, VATInclusiveTotal: 500000}},
		Totals: quotations.Totals{Subtotal: 500000, Total: 500000},
	}
}

func TestCreateSalesOrderUsesFixedSalesPersonAndPO(t *testing.T) {
	orders := &orderRepo{}
	h := NewHandler(orders, quotationRepo{value: testQuotation()})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	request := httptest.NewRequest("POST", "/quotations/7/sales-order", strings.NewReader("customer_po_number=PO-123"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, request)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect, got %d", w.Code)
	}
	if orders.value.SalesPerson != FixedSalesPerson || orders.value.CustomerPONumber != "PO-123" {
		t.Fatalf("unexpected sales order: %+v", orders.value)
	}
}

func TestSalesOrderPDFUsesSalesOrderTitle(t *testing.T) {
	q := testQuotation()
	orders := &orderRepo{value: SalesOrder{ID: 9, Number: "SO-00000009", Status: Open, Quotation: q}}
	h := NewHandler(orders, quotationRepo{value: q})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/sales-orders/9/pdf", nil))
	if w.Code != http.StatusOK || !bytes.HasPrefix(w.Body.Bytes(), []byte("%PDF-")) {
		t.Fatalf("unexpected PDF response: status=%d bytes=%d", w.Code, w.Body.Len())
	}
	if got := w.Header().Get("Content-Disposition"); got != `attachment; filename="SO-00000009.pdf"` {
		t.Fatalf("unexpected filename: %q", got)
	}
}
