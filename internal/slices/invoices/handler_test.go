package invoices

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
	"github.com/psbernardo/syncline-collection-tracking/internal/slices/quotations"
	"github.com/psbernardo/syncline-collection-tracking/internal/slices/salesorders"
)

type orderRepo struct{ order salesorders.SalesOrder }

func (r orderRepo) CreateFromQuotation(context.Context, quotations.Quotation, string) (salesorders.SalesOrder, error) {
	return r.order, nil
}
func (r orderRepo) FindByID(context.Context, int64) (salesorders.SalesOrder, error) {
	return r.order, nil
}

type invoiceRepo struct {
	created bool
	value   Invoice
}

func (r *invoiceRepo) CreateFromSalesOrder(_ context.Context, id int64, number string, date time.Time, key, requestID, actor string) (Invoice, error) {
	r.created = true
	return Invoice{ID: 12, SalesOrderID: id, Number: number, InvoiceDateUTC: date}, nil
}
func (r *invoiceRepo) FindByID(context.Context, int64) (Invoice, error) { return r.value, nil }
func (r *invoiceRepo) FindBySalesOrder(context.Context, int64) (Invoice, error) {
	return Invoice{}, nil
}
func (r *invoiceRepo) List(context.Context) ([]Invoice, error) { return nil, nil }

func testOrder() salesorders.SalesOrder {
	return salesorders.SalesOrder{ID: 7, Number: "SO-00000007", Status: salesorders.Open, CustomerPONumber: "PO-7", Quotation: quotations.Quotation{CompanyName: "Customer", TermsDays: 30, Lines: []quotations.Line{{ProductSKU: "SKU-1", ProductName: "Product", UOM: "PC", Quantity: money.Amount(2), UnitPrice: money.Amount(10000), TaxCode: quotations.TaxVAT12, TaxRate: 120000, LineTotal: money.Amount(20000), VATInclusiveTotal: money.Amount(22400)}}, Totals: quotations.Totals{Total: money.Amount(22400)}}}
}

func TestConversionFormIsReadOnly(t *testing.T) {
	repo := &invoiceRepo{}
	h := NewHandler(orderRepo{order: testOrder()}, repo)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/sales-orders/7/invoice/new", nil))
	body := w.Body.String()
	if w.Code != http.StatusOK || !strings.Contains(body, "invoice_number") || !strings.Contains(body, "Product") {
		t.Fatalf("unexpected response: %d %s", w.Code, body)
	}
	if !strings.Contains(body, "<td>₱2.00</td>") || strings.Contains(body, "<td>₱2.24</td>") {
		t.Fatal("invoice grid amount is not quantity multiplied by unit price")
	}
	if strings.Contains(body, "add-item") || strings.Contains(body, "removeLine") || strings.Contains(body, "name=\"lines") {
		t.Fatal("invoice form exposes line editing")
	}
	if !strings.Contains(body, "Create invoice and receivable?") || !strings.Contains(body, "VAT-inclusive, 1% EWT") {
		t.Fatal("invoice confirmation summary is missing")
	}
	if !strings.Contains(body, "$el.target = '_blank'") {
		t.Fatal("confirmation submit does not open the result in a new tab")
	}
}

func TestConvertedSalesOrderCannotOpenInvoiceForm(t *testing.T) {
	h := NewHandler(orderRepo{order: salesorders.SalesOrder{ID: 7, Status: salesorders.Converted}}, &invoiceRepo{})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/sales-orders/7/invoice/new", nil))
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestCreateInvoiceRedirectsToInvoiceDetail(t *testing.T) {
	repo := &invoiceRepo{}
	h := NewHandler(orderRepo{order: testOrder()}, repo)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	r := httptest.NewRequest("POST", "/sales-orders/7/invoice", strings.NewReader("invoice_number=+INV-7+&idempotency_key=key"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusSeeOther || w.Header().Get("Location") != "/invoices/12" || !repo.created {
		t.Fatalf("unexpected conversion response: %d %q", w.Code, w.Header().Get("Location"))
	}
}

func TestInvoicePDFReturnsInlinePDF(t *testing.T) {
	repo := &invoiceRepo{value: Invoice{ID: 12, Number: "INV-00000012", SalesOrderNumber: "SO-00000007", CustomerName: "Customer", InvoiceDateUTC: time.Now().UTC(), DueDateUTC: time.Now().UTC(), TermsDays: 30, Lines: []Line{{SKU: "SKU-1", Name: "Product", Quantity: money.Amount(10000), UOM: "PC", UnitPrice: money.Amount(10000), LineTotal: money.Amount(10000), VATInclusiveTotal: money.Amount(11200)}}}}
	h := NewHandler(orderRepo{order: testOrder()}, repo)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/invoices/12/pdf", nil))
	if w.Code != http.StatusOK || !bytes.HasPrefix(w.Body.Bytes(), []byte("%PDF-")) || !bytes.HasSuffix(bytes.TrimSpace(w.Body.Bytes()), []byte("%%EOF")) {
		t.Fatalf("unexpected PDF response: status=%d bytes=%d", w.Code, w.Body.Len())
	}
	if got := w.Header().Get("Content-Length"); got != strconv.Itoa(w.Body.Len()) {
		t.Fatalf("unexpected content length: header=%q body=%d", got, w.Body.Len())
	}
	if got := w.Header().Get("Content-Disposition"); got != `inline; filename="INV-00000012.pdf"` {
		t.Fatalf("unexpected filename: %q", got)
	}
	if got := w.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("unexpected content protection header: %q", got)
	}
}

func TestInvoiceDetailShowsTotalComputation(t *testing.T) {
	amounts := Invoice{ID: 12, Number: "INV-00000012", SalesOrderID: 7, SalesOrderNumber: "SO-00000007", CustomerName: "Customer", InvoiceDateUTC: time.Now().UTC(), DueDateUTC: time.Now().UTC(), TermsDays: 30, Subtotal: money.Amount(10000000), Tax: money.Amount(1200000), Total: money.Amount(11200000), Status: Posted, Lines: []Line{{SKU: "SKU-1", Name: "Product", Quantity: money.Amount(10000), UOM: "PC", UnitPrice: money.Amount(10000000), LineTotal: money.Amount(10000000)}}}
	h := NewHandler(orderRepo{order: testOrder()}, &invoiceRepo{value: amounts})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/invoices/12", nil))
	body := w.Body.String()
	for _, label := range []string{"Vatable Sales", "Vat", "Total Sales", "Less: VAT", "Add Vat", "Total amount due"} {
		if !strings.Contains(body, label) {
			t.Errorf("invoice detail is missing total computation field %q", label)
		}
	}
	if !strings.Contains(body, "₱1,120.00") || !strings.Contains(body, "₱1,000.00") || !strings.Contains(body, "₱120.00") {
		t.Error("invoice detail is missing computed amount values")
	}
}
