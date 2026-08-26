package salesorders

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func TestStandaloneSalesOrderPageUsesSharedLayout(t *testing.T) {
	h := NewHandler(&orderRepo{}, quotationRepo{value: testQuotation()})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/sales-orders/new", nil))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "New repeat order") {
		t.Fatalf("unexpected repeat-order page: status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestQuotationSalesOrderPageShowsRemainingLines(t *testing.T) {
	q := testQuotation()
	q.Status = quotations.Approved
	h := NewHandler(&orderRepo{}, quotationRepo{value: q})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/quotations/7/sales-order/new", nil))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "Move all remaining") || !strings.Contains(w.Body.String(), "Product") {
		t.Fatalf("unexpected quotation conversion page: status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestStandaloneFormParsesCustomerAndLines(t *testing.T) {
	form := url.Values{"company_account_id": {"42"}, "customer_po_number": {"PO-123"}, "terms_days": {"30"}, "lines[0].product_id": {"9"}, "lines[0].quantity": {"2"}, "lines[0].uom": {"BOX"}, "lines[0].unit_price": {"125"}, "lines[0].tax_code": {"NONE"}, "lines[0].tax_rate": {"0"}}
	r := httptest.NewRequest("POST", "/sales-orders", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	input, err := standaloneFromForm(r)
	if err != nil || input.CompanyAccountID != 42 || len(input.Lines) != 1 || input.Lines[0].ProductID != 9 {
		t.Fatalf("unexpected standalone input: %+v, error=%v", input, err)
	}
}

func TestStandaloneEditPageShowsRemoveControls(t *testing.T) {
	q := testQuotation()
	orders := &orderRepo{value: SalesOrder{ID: 9, Number: "SO-00000009", Status: Open, Quotation: q}}
	h := NewHandler(orders, quotationRepo{value: q})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/sales-orders/9/edit", nil))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "Edit sales order") || !strings.Contains(w.Body.String(), "Remove") {
		t.Fatalf("unexpected edit page: status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestStandaloneOpenOrderViewShowsEditButton(t *testing.T) {
	q := testQuotation()
	orders := &orderRepo{value: SalesOrder{ID: 9, Number: "SO-00000009", Status: Open, Quotation: q}}
	h := NewHandler(orders, quotationRepo{value: q})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/sales-orders/9", nil))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `href="/sales-orders/9/edit"`) {
		t.Fatalf("edit button missing: status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestQuotationOpenOrderViewShowsEditButton(t *testing.T) {
	q := testQuotation()
	orders := &orderRepo{value: SalesOrder{ID: 9, Number: "SO-00000009", QuotationID: q.ID, QuotationNumber: q.Number, Status: Open, Quotation: q}}
	h := NewHandler(orders, quotationRepo{value: q})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/sales-orders/9", nil))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `href="/sales-orders/9/edit"`) {
		t.Fatalf("quotation edit button missing: status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestQuotationEditRequiresAcknowledgement(t *testing.T) {
	q := testQuotation()
	orders := &orderRepo{value: SalesOrder{ID: 9, Number: "SO-00000009", QuotationID: q.ID, Status: Open, Quotation: q}}
	h := NewHandler(orders, quotationRepo{value: q})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("POST", "/sales-orders/9", nil))
	if w.Code != http.StatusUnprocessableEntity || !strings.Contains(w.Body.String(), "acknowledge") {
		t.Fatalf("expected acknowledgement error: status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestQuotationSelectionsParseOnlyPositiveQuantities(t *testing.T) {
	form := url.Values{"lines[0].id": {"11"}, "lines[0].quantity": {"4"}, "lines[1].id": {"12"}, "lines[1].quantity": {"0"}}
	r := httptest.NewRequest("POST", "/quotations/7/sales-order", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	selections, err := selectionsFromForm(r)
	if err != nil || len(selections) != 1 || selections[0].QuotationLineID != 11 {
		t.Fatalf("unexpected selections: %+v, error=%v", selections, err)
	}
}

func TestCustomerPOIsRequiredAndTrimmed(t *testing.T) {
	if _, err := ValidateCustomerPO("   "); err == nil {
		t.Fatal("expected blank customer PO to be rejected")
	}
	value, err := ValidateCustomerPO("  PO-123  ")
	if err != nil || value != "PO-123" {
		t.Fatalf("unexpected customer PO: %q, error=%v", value, err)
	}
}
