package purchaseorders

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
	"github.com/psbernardo/syncline-collection-tracking/internal/slices/quotations"
	"github.com/psbernardo/syncline-collection-tracking/internal/slices/salesorders"
)

type testOrders struct{ value salesorders.SalesOrder }

func (r testOrders) CreateFromQuotation(context.Context, quotations.Quotation, string) (salesorders.SalesOrder, error) {
	return r.value, nil
}
func (r testOrders) FindByID(context.Context, int64) (salesorders.SalesOrder, error) {
	return r.value, nil
}

type testPurchase struct {
	value       PurchaseOrder
	products    map[int64]SupplierProduct
	createCalls int
	updateCalls int
	lastUpdate  CreateInput
}

func (r *testPurchase) NextNumber(context.Context, time.Time) (string, error) {
	return "PO-20260823-0001", nil
}
func (r *testPurchase) CreateFromSalesOrder(context.Context, CreateInput) (PurchaseOrder, error) {
	r.createCalls++
	return r.value, nil
}
func (r *testPurchase) FindByID(context.Context, int64) (PurchaseOrder, error) { return r.value, nil }
func (r *testPurchase) List(context.Context) ([]PurchaseOrder, error) {
	return []PurchaseOrder{r.value}, nil
}
func (r *testPurchase) Update(_ context.Context, _ int64, input CreateInput) (PurchaseOrder, error) {
	r.updateCalls++
	r.lastUpdate = input
	return r.value, nil
}
func (r *testPurchase) SupplierProducts(context.Context, int64) (map[int64]SupplierProduct, error) {
	if r.products == nil {
		return map[int64]SupplierProduct{}, nil
	}
	return r.products, nil
}

func testPurchaseOrder() PurchaseOrder {
	return PurchaseOrder{ID: 3, Number: "PO-20260823-0001", SupplierID: 4, SupplierName: "Supplier", SalesOrderID: 9, SalesOrderNumber: "SO-00000009", PODate: time.Date(2026, 8, 23, 0, 0, 0, 0, time.UTC), Status: "OPEN", Total: money.Amount(100000), Lines: []Line{{SalesOrderLineID: 12, ProductID: 1, SKU: "SKU-1", Name: "Widget", UOM: "PC", Quantity: 10000, UnitCost: 100000, Total: 100000}}}
}

func TestPurchaseOrderListAndEditPages(t *testing.T) {
	purchase := &testPurchase{value: testPurchaseOrder(), products: map[int64]SupplierProduct{1: {ProductID: 1}, 2: {ProductID: 2}}}
	h, err := NewHandler(testOrders{}, purchase, func(context.Context) ([]SupplierOption, error) {
		return []SupplierOption{{ID: 4, Name: "Supplier"}}, nil
	}, purchase)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	for path, expected := range map[string]string{"/purchase-orders": "All purchase orders", "/purchase-orders/3": "Download PDF", "/purchase-orders/3/edit": "Save changes"} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), expected) {
			t.Fatalf("%s: status=%d body=%s", path, w.Code, w.Body.String())
		}
	}
}

type failingPurchaseOrderRenderer struct{}

func (failingPurchaseOrderRenderer) Render(io.Writer, PurchaseOrderPDFDocument) error {
	return errors.New("render failed")
}

func TestPurchaseOrderPDFDownload(t *testing.T) {
	purchase := &testPurchase{value: testPurchaseOrder()}
	h, err := NewHandler(testOrders{}, purchase, func(context.Context) ([]SupplierOption, error) {
		return []SupplierOption{{ID: 4, Name: "Supplier"}}, nil
	}, purchase)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/purchase-orders/3/pdf", nil))
	if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "application/pdf" || !strings.Contains(w.Header().Get("Content-Disposition"), "PO-20260823-0001.pdf") || w.Header().Get("X-Content-Type-Options") != "nosniff" || !strings.HasPrefix(w.Body.String(), "%PDF-") {
		t.Fatalf("unexpected PDF response: status=%d headers=%v", w.Code, w.Header())
	}
}

func TestPurchaseOrderPDFRendererFailureDoesNotWritePartialResponse(t *testing.T) {
	purchase := &testPurchase{value: testPurchaseOrder()}
	h, err := NewHandler(testOrders{}, purchase, func(context.Context) ([]SupplierOption, error) {
		return nil, nil
	}, purchase)
	if err != nil {
		t.Fatal(err)
	}
	h.pdfRenderer = failingPurchaseOrderRenderer{}
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/purchase-orders/3/pdf", nil))
	if w.Code != http.StatusInternalServerError || strings.Contains(w.Header().Get("Content-Type"), "application/pdf") {
		t.Fatalf("unexpected renderer failure response: status=%d headers=%v", w.Code, w.Header())
	}
}

func TestPurchaseOrderSupplierSelectionUsesPostPreview(t *testing.T) {
	purchase := &testPurchase{value: testPurchaseOrder()}
	h, err := NewHandler(testOrders{value: salesorders.SalesOrder{ID: 9, Status: salesorders.Open}}, purchase, func(context.Context) ([]SupplierOption, error) {
		return []SupplierOption{{ID: 2, Name: "Alternate Supplier"}}, nil
	}, purchase)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	form := strings.NewReader("supplier_id=2&purchase_order_number=PO-20260823-0001")
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/sales-orders/9/purchase-order/supplier-preview", form)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "Alternate Supplier") || !strings.Contains(w.Body.String(), "purchase-order/supplier-preview") {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if purchase.createCalls != 0 {
		t.Fatal("supplier preview attempted to create a purchase order")
	}
}

func TestPurchaseOrderGridUsesInlineAction(t *testing.T) {
	order := salesorders.SalesOrder{ID: 9, Number: "SO-00000009", Status: salesorders.Open, Quotation: quotations.Quotation{Lines: []quotations.Line{
		{ID: 12, ProductID: 1, ProductSKU: "SKU-1", ProductName: "Widget", UOM: "PC", Quantity: 10000},
	}}}
	purchase := &testPurchase{value: testPurchaseOrder()}
	h, err := NewHandler(testOrders{value: order}, purchase, func(context.Context) ([]SupplierOption, error) {
		return []SupplierOption{{ID: 4, Name: "Supplier"}}, nil
	}, purchase)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/sales-orders/9/purchase-order/new", nil))
	body := w.Body.String()
	if w.Code != http.StatusOK || strings.Contains(body, ".selected\" type=\"checkbox\"") || strings.Contains(body, "data-source-line-id=") || !strings.Contains(body, "No items added") || !strings.Contains(body, "quotation-remove-action") || strings.Contains(body, "class=\"ellipsis-button\"") {
		t.Fatalf("status=%d body=%s", w.Code, body)
	}
}

func TestPurchaseOrderSupplierPreviewPreservesRemovedLines(t *testing.T) {
	order := salesorders.SalesOrder{ID: 9, Number: "SO-00000009", Status: salesorders.Open, Quotation: quotations.Quotation{Lines: []quotations.Line{
		{ID: 12, ProductID: 1, ProductSKU: "SKU-1", ProductName: "Widget", UOM: "PC", Quantity: 10000},
		{ID: 13, ProductID: 2, ProductSKU: "SKU-2", ProductName: "Gadget", UOM: "BOX", Quantity: 20000},
	}}}
	purchase := &testPurchase{value: testPurchaseOrder(), products: map[int64]SupplierProduct{1: {ProductID: 1}, 2: {ProductID: 2}}}
	h, err := NewHandler(testOrders{value: order}, purchase, func(context.Context) ([]SupplierOption, error) {
		return []SupplierOption{{ID: 4, Name: "Supplier"}}, nil
	}, purchase)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	form := strings.NewReader("supplier_id=4&excluded_line_ids=12")
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/sales-orders/9/purchase-order/supplier-preview", form)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mux.ServeHTTP(w, r)
	body := w.Body.String()
	if w.Code != http.StatusOK || strings.Contains(body, "SKU-1") || !strings.Contains(body, "SKU-2") {
		t.Fatalf("status=%d body=%s", w.Code, body)
	}
}

func TestPurchaseOrderEditRejectsUnavailableSupplierProduct(t *testing.T) {
	order := testPurchaseOrder()
	order.Lines = append(order.Lines, Line{SalesOrderLineID: 13, ProductID: 2, SKU: "SKU-2", Name: "Gadget", UOM: "BOX", Quantity: 20000, UnitCost: 200000, Total: 400000})
	purchase := &testPurchase{value: order}
	h, err := NewHandler(testOrders{}, purchase, func(context.Context) ([]SupplierOption, error) {
		return []SupplierOption{{ID: 4, Name: "Supplier"}}, nil
	}, purchase)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	form := strings.NewReader("supplier_id=4&po_date=2026-08-23&lines%5B12%5D.unit_cost=100.00&lines%5B13%5D.unit_cost=200.00")
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/purchase-orders/3", form)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "The following items are not configured for Supplier:") || !strings.Contains(w.Body.String(), "SKU-1") || !strings.Contains(w.Body.String(), "Widget") || !strings.Contains(w.Body.String(), "SKU-2") || !strings.Contains(w.Body.String(), "Gadget") {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if strings.Count(w.Body.String(), "<li>") != 2 {
		t.Fatalf("expected two missing product list items, body=%s", w.Body.String())
	}
	if purchase.updateCalls != 0 {
		t.Fatal("purchase order update was called for an unavailable supplier product")
	}
}

func TestPurchaseOrderCreateListsUnavailableSupplierProducts(t *testing.T) {
	purchase := &testPurchase{value: testPurchaseOrder()}
	orders := testOrders{value: salesorders.SalesOrder{
		ID: 9, Number: "SO-00000009", Status: salesorders.Open,
		Quotation: quotations.Quotation{Lines: []quotations.Line{
			{ID: 12, ProductID: 1, ProductSKU: "SKU-1", ProductName: "Widget", Quantity: 10000},
			{ID: 13, ProductID: 2, ProductSKU: "SKU-2", ProductName: "Gadget", Quantity: 20000},
		}},
	}}
	h, err := NewHandler(orders, purchase, func(context.Context) ([]SupplierOption, error) {
		return []SupplierOption{{ID: 4, Name: "Supplier"}}, nil
	}, purchase)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	form := strings.NewReader("supplier_id=4&po_date=2026-08-23&purchase_order_number=PO-20260823-0001&lines%5B12%5D.unit_cost=100.00&lines%5B13%5D.unit_cost=200.00")
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/sales-orders/9/purchase-order", form)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "The following items are not configured for Supplier:") || !strings.Contains(w.Body.String(), "SKU-1") || !strings.Contains(w.Body.String(), "Widget") || !strings.Contains(w.Body.String(), "SKU-2") || !strings.Contains(w.Body.String(), "Gadget") {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if strings.Count(w.Body.String(), "<li>") != 2 {
		t.Fatalf("expected two missing product list items, body=%s", w.Body.String())
	}
	if purchase.createCalls != 0 {
		t.Fatal("purchase order creation was called for unavailable supplier products")
	}
}

func TestDirectPurchaseOrderUsesCanonicalEditFormAndUpdate(t *testing.T) {
	order := PurchaseOrder{ID: 8, Number: "PO-20260823-0008", SupplierID: 4, SupplierName: "Supplier", PODate: time.Date(2026, 8, 23, 0, 0, 0, 0, time.UTC), Status: "OPEN", Mode: DirectMode, Lines: []Line{{ProductID: 1, SKU: "SKU-1", Name: "Widget", UOM: "PC", Quantity: 10000, UnitCost: 50000}}}
	purchase := &testPurchase{value: order, products: map[int64]SupplierProduct{1: {ProductID: 1, SupplierSKU: "SUP-1", ReferenceCost: money.Amount(50000)}}}
	h, err := NewHandler(testOrders{}, purchase, func(context.Context) ([]SupplierOption, error) {
		return []SupplierOption{{ID: 4, Name: "Supplier"}}, nil
	}, purchase, func(context.Context) ([]ProductOption, error) {
		return []ProductOption{{ID: 1, SKU: "SKU-1", Name: "Widget", UOM: "PC"}, {ID: 2, SKU: "SKU-2", Name: "Gadget", UOM: "BOX"}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/purchase-orders/8/edit", nil))
	body := w.Body.String()
	if w.Code != http.StatusOK || !strings.Contains(body, "Save changes") || !strings.Contains(body, "products[0].quantity") || !strings.Contains(body, "purchase-direct-product-template") || !strings.Contains(body, "quotation-remove-action") || strings.Contains(body, "ellipsis-button") || strings.Contains(body, "Sales order reference") {
		t.Fatalf("status=%d body=%s", w.Code, body)
	}

	form := strings.NewReader("supplier_id=4&po_date=2026-08-24&products%5B1%5D.quantity=2.00&products%5B1%5D.unit_cost=6.00&payment_terms=Net+30")
	w = httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/purchase-orders/8", form)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusSeeOther || purchase.updateCalls != 1 || purchase.lastUpdate.Mode != DirectMode || len(purchase.lastUpdate.Lines) != 1 || purchase.lastUpdate.Lines[0].ProductID != 1 {
		t.Fatalf("status=%d update=%+v calls=%d", w.Code, purchase.lastUpdate, purchase.updateCalls)
	}
}

func TestSalesPurchaseOrderEditRendersExistingLines(t *testing.T) {
	order := testPurchaseOrder()
	purchase := &testPurchase{value: order, products: map[int64]SupplierProduct{1: {ProductID: 1}}}
	h, err := NewHandler(testOrders{}, purchase, func(context.Context) ([]SupplierOption, error) {
		return []SupplierOption{{ID: 4, Name: "Supplier"}}, nil
	}, purchase)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/purchase-orders/3/edit", nil))
	body := w.Body.String()
	if w.Code != http.StatusOK || !strings.Contains(body, "lines[12].quantity") || !strings.Contains(body, "lines[12].unit_cost") || !strings.Contains(body, "Remove item") {
		t.Fatalf("status=%d body=%s", w.Code, body)
	}
}

func TestNewDirectPurchaseOrderKeepsDirectProductGrid(t *testing.T) {
	purchase := &testPurchase{}
	h, err := NewHandler(testOrders{}, purchase, func(context.Context) ([]SupplierOption, error) {
		return []SupplierOption{{ID: 4, Name: "Supplier"}}, nil
	}, purchase, func(context.Context) ([]ProductOption, error) {
		return []ProductOption{{ID: 1, SKU: "SKU-1", Name: "Widget", UOM: "PC"}, {ID: 2, SKU: "SKU-2", Name: "Gadget", UOM: "BOX"}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/purchase-orders/new", nil))
	body := w.Body.String()
	if w.Code != http.StatusOK || !strings.Contains(body, "No items added") || !strings.Contains(body, "purchase-direct-product-template") || !strings.Contains(body, "products[__INDEX__].quantity\" type=\"number\" min=\"0\" step=\"0.0001\" value=\"0\"") || strings.Contains(body, "<tr data-purchase-line data-product-id=") || strings.Contains(body, "products[1].quantity") || strings.Contains(body, "purchase-orders/supplier-preview") {
		t.Fatalf("unexpected initial direct form: status=%d body=%s", w.Code, body)
	}
}
