package receivables

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/slices/accounts"
	"gorm.io/gorm"
)

func TestNewReceivableFormRendersAccounts(t *testing.T) {
	handler, err := NewHandler(NewService(nil, &fakeReceivableRepository{}, &fakeAccountRepository{}))
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/receivables/new", nil)
	handler.newForm(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if !strings.Contains(recorder.Body.String(), "Acme Corp") {
		t.Fatal("company option was not rendered")
	}
	if !strings.Contains(recorder.Body.String(), `name="invoice_number"`) {
		t.Fatal("invoice number field was not rendered")
	}
	if !strings.Contains(recorder.Body.String(), "Company accounts") || !strings.Contains(recorder.Body.String(), "Receivables") {
		t.Fatal("shared navigation was not rendered")
	}
}

func TestInvalidReceivableCreateReturnsHTMXFragment(t *testing.T) {
	handler, err := NewHandler(NewService(nil, &fakeReceivableRepository{}, &fakeAccountRepository{}))
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/receivables", strings.NewReader("idempotency_key=test"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("HX-Request", "true")
	handler.create(recorder, request)
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnprocessableEntity)
	}
	if !strings.Contains(recorder.Body.String(), "Select a company") {
		t.Fatal("company validation was not rendered")
	}
}

func TestReceivableListRendersMultiSelectStatusFilter(t *testing.T) {
	handler, err := NewHandler(NewService(nil, &fakeReceivableRepository{}, &fakeAccountRepository{}))
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/receivables?status=overdue&status=near_due", nil)
	handler.list(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	body := recorder.Body.String()
	for _, expected := range []string{
		`hx-get="/receivables" hx-include="#receivable-filters"`,
		`id="receivable-status-filter"`,
		`data-multi-select-option`,
		`data-multi-select-input name="status" value="overdue"`,
		`data-multi-select-input name="status" value="near_due"`,
		`<span>Overdue</span>`,
		`<span>Near due</span>`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("list response does not contain %q", expected)
		}
	}
}

func TestReceivableListHTMXResponseDoesNotDuplicateResultsTarget(t *testing.T) {
	handler, err := NewHandler(NewService(nil, &fakeReceivableRepository{}, &fakeAccountRepository{}))
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/receivables?company=1&company=2&invoice=%200220%20&po=%20po-1%20", nil)
	request.Header.Set("HX-Request", "true")
	handler.list(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	body := recorder.Body.String()
	if strings.Contains(body, `id="receivable-results"`) {
		t.Fatal("HTMX fragment duplicated the results target")
	}
	if !strings.Contains(body, "PO-1") {
		t.Fatal("normalized PO filter was not rendered")
	}
	if !strings.Contains(body, `Invoice contains &quot;0220&quot;.`) {
		t.Fatal("invoice filter was not rendered")
	}
}

func TestReceivableListLoadMoreReturnsRowsAndReplacementSentinel(t *testing.T) {
	handler, err := NewHandler(NewService(nil, &loadMoreReceivableRepository{}, &fakeAccountRepository{}))
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/receivables?company=1&invoice=0220&po=PO-1&status=overdue&cursor=cursor-value&load_more=1", nil)
	request.Header.Set("HX-Request", "true")
	handler.list(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	body := recorder.Body.String()
	if strings.Contains(body, `id="receivable-results"`) || strings.Contains(body, "<section") {
		t.Fatal("load-more response rendered the full results section")
	}
	for _, expected := range []string{"PO-next", "hx-trigger=\"revealed\"", "load_more=1", "invoice=0220", "status=overdue", "page_size=5"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("load-more response does not contain %q", expected)
		}
	}
}

func TestReceivableLoadMoreURLUsesFiveItemPages(t *testing.T) {
	query := ListQuery{CompanyAccountIDs: []int64{2}, Invoice: "0220", PO: "PO-1", Statuses: []string{"overdue"}, PageSize: initialListPageSize}
	loadMoreURL := receivableLoadMoreURL(query, "cursor-value")
	if !strings.Contains(loadMoreURL, "page_size=5") {
		t.Fatalf("load-more URL = %q, want page_size=5", loadMoreURL)
	}
}

func TestParseListQueryMarksInvalidStatusFilterAsProvided(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/receivables?status=invalid", nil)
	query := parseListQuery(request)
	if !query.StatusFilterProvided {
		t.Fatal("status filter was not marked as provided")
	}
	if len(query.Statuses) != 0 {
		t.Fatalf("statuses = %#v, want empty after normalization", query.Statuses)
	}
}

func TestParseListQueryPreservesInvoiceLeadingZeroes(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/receivables?invoice=%200220%20", nil)
	query := parseListQuery(request)
	if query.Invoice != "0220" {
		t.Fatalf("invoice = %q, want 0220", query.Invoice)
	}
}

func TestParseListQueryParsesMultipleCompanies(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/receivables?company=2&company=1&company=2", nil)
	query := parseListQuery(request)
	if !query.CompanyFilterProvided {
		t.Fatal("company filter was not marked as provided")
	}
	if len(query.CompanyAccountIDs) != 2 || query.CompanyAccountIDs[0] != 2 || query.CompanyAccountIDs[1] != 1 {
		t.Fatalf("company IDs = %#v, want [2 1]", query.CompanyAccountIDs)
	}
}

func TestReceivableListRendersCompanyMultiSelect(t *testing.T) {
	handler, err := NewHandler(NewService(nil, &fakeReceivableRepository{}, &fakeAccountRepository{}))
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/receivables?company=1", nil)
	handler.list(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	body := recorder.Body.String()
	for _, expected := range []string{`id="receivable-company-filter"`, `data-multi-select-option`, `data-multi-select-input name="company" value="1"`, `data-label="Acme Corp"`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("list response does not contain %q", expected)
		}
	}
}

func TestReceivablePaymentPageRendersConfirmationWorkflow(t *testing.T) {
	handler, err := NewHandler(NewService(nil, &paymentReceivableRepository{}, &fakeAccountRepository{}))
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/receivables/1/payment", nil)
	request.SetPathValue("id", "1")
	handler.payment(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	body := recorder.Body.String()
	for _, expected := range []string{"Acknowledge full payment", "Receivable summary", "₱0.01", `name="payment_date"`, "Payment received date", "Full payment only", "normal financial editing"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("detail does not contain %q", expected)
		}
	}
}

func TestReceivableDetailRendersProfessionalSummaryAndActions(t *testing.T) {
	handler, err := NewHandler(NewService(nil, &paymentReceivableRepository{}, &fakeAccountRepository{}))
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/receivables/1", nil)
	request.SetPathValue("id", "1")
	handler.detail(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	body := recorder.Body.String()
	for _, expected := range []string{"Receivable details", "Collection record", "Receivable information", "Delivery and payment terms", "Has the full payment been received?", "Acknowledge payment"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("detail does not contain %q", expected)
		}
	}
}

type fakeAccountRepository struct{}

func (*fakeAccountRepository) Exists(context.Context, *gorm.DB, int64) (bool, error) {
	return true, nil
}

func (*fakeAccountRepository) List(context.Context) ([]accounts.CompanyAccount, error) {
	return []accounts.CompanyAccount{{ID: 1, CompanyName: "Acme Corp"}}, nil
}

type fakeReceivableRepository struct{}

type loadMoreReceivableRepository struct{}

func (*fakeReceivableRepository) Create(context.Context, *gorm.DB, DeliveryReceivable) (DeliveryReceivable, error) {
	return DeliveryReceivable{ID: 1}, nil
}

func (*fakeReceivableRepository) FindByID(context.Context, *gorm.DB, int64) (DeliveryReceivable, error) {
	return DeliveryReceivable{}, ErrNotFound
}

func (*fakeReceivableRepository) List(context.Context) ([]DeliveryReceivable, error) {
	return []DeliveryReceivable{}, nil
}

func (*fakeReceivableRepository) ListFiltered(context.Context, ListQuery) ([]DeliveryReceivable, int64, error) {
	return []DeliveryReceivable{}, 0, nil
}

func (*loadMoreReceivableRepository) Create(context.Context, *gorm.DB, DeliveryReceivable) (DeliveryReceivable, error) {
	return DeliveryReceivable{}, nil
}

func (*loadMoreReceivableRepository) FindByID(context.Context, *gorm.DB, int64) (DeliveryReceivable, error) {
	return DeliveryReceivable{}, ErrNotFound
}

func (*loadMoreReceivableRepository) List(context.Context) ([]DeliveryReceivable, error) {
	return nil, nil
}

func (*loadMoreReceivableRepository) ListFiltered(context.Context, ListQuery) ([]DeliveryReceivable, int64, error) {
	return []DeliveryReceivable{{ID: 2, CompanyName: "Acme Corp", InvoiceNumber: "0220", PONumber: "PO-next", DeliveryDateUTC: time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC), DueDateUTC: time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC), PaymentTermDays: 5, AmountDue: 100, LifecycleStatus: "Active", RowVersion: []byte("version")}}, 2, nil
}

func (*loadMoreReceivableRepository) Update(context.Context, *gorm.DB, DeliveryReceivable, []byte) (DeliveryReceivable, error) {
	return DeliveryReceivable{}, nil
}

func (*loadMoreReceivableRepository) MarkPaymentReceived(context.Context, *gorm.DB, int64, time.Time, []byte) (DeliveryReceivable, error) {
	return DeliveryReceivable{}, nil
}

func (*loadMoreReceivableRepository) FindBlockingPONumber(context.Context, *gorm.DB, string, int64) (bool, error) {
	return false, nil
}

func (*loadMoreReceivableRepository) FindBlockingInvoiceNumber(context.Context, *gorm.DB, string, int64) (bool, error) {
	return false, nil
}

func (*fakeReceivableRepository) Update(context.Context, *gorm.DB, DeliveryReceivable, []byte) (DeliveryReceivable, error) {
	return DeliveryReceivable{}, nil
}

func (*fakeReceivableRepository) MarkPaymentReceived(context.Context, *gorm.DB, int64, time.Time, []byte) (DeliveryReceivable, error) {
	return DeliveryReceivable{}, nil
}

type paymentReceivableRepository struct{}

func (*paymentReceivableRepository) Create(context.Context, *gorm.DB, DeliveryReceivable) (DeliveryReceivable, error) {
	return DeliveryReceivable{}, nil
}

func (*paymentReceivableRepository) FindByID(context.Context, *gorm.DB, int64) (DeliveryReceivable, error) {
	return DeliveryReceivable{ID: 1, CompanyAccountID: 1, CompanyName: "Acme Corp", InvoiceNumber: "INV001", PONumber: "PO001", DeliveryDateUTC: time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC), PaymentTermDays: 5, DueDateUTC: time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC), AmountDue: 100, LifecycleStatus: "Active", RowVersion: []byte("version")}, nil
}

func (*paymentReceivableRepository) List(context.Context) ([]DeliveryReceivable, error) {
	return nil, nil
}

func (*paymentReceivableRepository) ListFiltered(context.Context, ListQuery) ([]DeliveryReceivable, int64, error) {
	return nil, 0, nil
}

func (*paymentReceivableRepository) Update(context.Context, *gorm.DB, DeliveryReceivable, []byte) (DeliveryReceivable, error) {
	return DeliveryReceivable{}, nil
}

func (*paymentReceivableRepository) MarkPaymentReceived(context.Context, *gorm.DB, int64, time.Time, []byte) (DeliveryReceivable, error) {
	return DeliveryReceivable{}, nil
}

func (*paymentReceivableRepository) FindBlockingPONumber(context.Context, *gorm.DB, string, int64) (bool, error) {
	return false, nil
}

func (*paymentReceivableRepository) FindBlockingInvoiceNumber(context.Context, *gorm.DB, string, int64) (bool, error) {
	return false, nil
}

func (*fakeReceivableRepository) FindBlockingPONumber(context.Context, *gorm.DB, string, int64) (bool, error) {
	return false, nil
}

func (*fakeReceivableRepository) FindBlockingInvoiceNumber(context.Context, *gorm.DB, string, int64) (bool, error) {
	return false, nil
}
