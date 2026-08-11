package receivables

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
	request := httptest.NewRequest(http.MethodGet, "/receivables?company=%20Acme%20&po=%20po-1%20", nil)
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

type fakeAccountRepository struct{}

func (*fakeAccountRepository) Exists(context.Context, *gorm.DB, int64) (bool, error) {
	return true, nil
}

func (*fakeAccountRepository) List(context.Context) ([]accounts.CompanyAccount, error) {
	return []accounts.CompanyAccount{{ID: 1, CompanyName: "Acme Corp"}}, nil
}

type fakeReceivableRepository struct{}

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

func (*fakeReceivableRepository) Update(context.Context, *gorm.DB, DeliveryReceivable, []byte) (DeliveryReceivable, error) {
	return DeliveryReceivable{}, nil
}

func (*fakeReceivableRepository) FindBlockingPONumber(context.Context, *gorm.DB, string, int64) (bool, error) {
	return false, nil
}
