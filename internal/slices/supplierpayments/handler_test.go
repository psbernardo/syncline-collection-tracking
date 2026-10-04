package supplierpayments

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/businessdate"
	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
)

type fakeApplication struct {
	result     ListResult
	payment    Payment
	suppliers  []SupplierOption
	created    Input
	updated    Input
	updatedID  int64
	updatedVer []byte
	voidedID   int64
	voidedVer  []byte
	listVoided bool
}

func (fake *fakeApplication) List(_ context.Context, includeVoided bool) (ListResult, error) {
	fake.listVoided = includeVoided
	return fake.result, nil
}
func (fake *fakeApplication) Get(_ context.Context, _ int64) (Payment, error) {
	return fake.payment, nil
}
func (fake *fakeApplication) Suppliers(context.Context) ([]SupplierOption, error) {
	return fake.suppliers, nil
}
func (fake *fakeApplication) Create(_ context.Context, input Input) (Payment, error) {
	fake.created = input
	return Payment{ID: 1}, nil
}
func (fake *fakeApplication) Update(_ context.Context, id int64, input Input, version []byte) (Payment, error) {
	fake.updatedID, fake.updated, fake.updatedVer = id, input, version
	return fake.payment, nil
}
func (fake *fakeApplication) Void(_ context.Context, id int64, version []byte) error {
	fake.voidedID, fake.voidedVer = id, version
	return nil
}

func TestListPageShowsDueTotalsAndVoidedToggle(t *testing.T) {
	amount, _ := money.Parse("100")
	service := &fakeApplication{result: ListResult{Summary: Summary{NearDue: amount, WithinFiveDays: amount}}}
	handler := testHandler(t, service)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/supplier-payments?show_voided=1", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("GET list status = %d, body %s", response.Code, response.Body.String())
	}
	for _, expected := range []string{"Near due · next 14 days", "Due within 5 days", "Due in 6–14 days", "₱100.00", "Hide voided"} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Errorf("list page does not contain %q", expected)
		}
	}
	if !service.listVoided {
		t.Error("show_voided query was not passed to the service")
	}
}

func TestNewFormAndCreatePost(t *testing.T) {
	service := &fakeApplication{suppliers: []SupplierOption{{ID: 7, Name: "Acme"}}}
	handler := testHandler(t, service)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	page := httptest.NewRecorder()
	mux.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/supplier-payments/new", nil))
	if page.Code != http.StatusOK {
		t.Fatalf("GET new status = %d", page.Code)
	}
	for _, expected := range []string{`name="supplier_id"`, `name="check_number"`, `name="issue_date"`, `name="due_date"`, `name="amount"`, "Acme"} {
		if !strings.Contains(page.Body.String(), expected) {
			t.Errorf("new form does not contain %q", expected)
		}
	}

	values := url.Values{"supplier_id": {"7"}, "check_number": {"00023"}, "issue_date": {"2026-05-12"}, "due_date": {"2026-05-20"}, "amount": {"125.50"}}
	request := httptest.NewRequest(http.MethodPost, "/supplier-payments", strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/supplier-payments" {
		t.Fatalf("POST create status/location = %d/%q", response.Code, response.Header().Get("Location"))
	}
	if service.created.SupplierID != 7 || service.created.CheckNumber != "00023" || service.created.Amount != "125.50" {
		t.Errorf("captured create input = %#v", service.created)
	}
}

func TestEditUpdateAndDeleteUseIDAndRowVersion(t *testing.T) {
	issue, _ := businessdate.Parse("2026-05-12")
	due, _ := businessdate.Parse("2026-05-20")
	service := &fakeApplication{
		payment:   Payment{ID: 42, SupplierID: 7, SupplierName: "Acme", CheckNumber: "00023", IssueDateUTC: issue, DueDateUTC: due, Amount: money.Amount(1255000), Status: "Active", RowVersion: []byte{1, 2, 3}},
		suppliers: []SupplierOption{{ID: 7, Name: "Acme"}},
	}
	handler := testHandler(t, service)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	edit := httptest.NewRecorder()
	mux.ServeHTTP(edit, httptest.NewRequest(http.MethodGet, "/supplier-payments/42/edit", nil))
	if edit.Code != http.StatusOK || !strings.Contains(edit.Body.String(), `action="/supplier-payments/42"`) {
		t.Fatalf("edit page status/action = %d; body %s", edit.Code, edit.Body.String())
	}
	version := encodeVersion([]byte{1, 2, 3})
	values := url.Values{"supplier_id": {"7"}, "check_number": {"00024"}, "issue_date": {"2026-05-12"}, "due_date": {"2026-05-21"}, "amount": {"125.50"}, "row_version": {version}}
	updateRequest := httptest.NewRequest(http.MethodPost, "/supplier-payments/42", strings.NewReader(values.Encode()))
	updateRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	update := httptest.NewRecorder()
	mux.ServeHTTP(update, updateRequest)
	if update.Code != http.StatusSeeOther || service.updatedID != 42 || string(service.updatedVer) != string([]byte{1, 2, 3}) {
		t.Fatalf("update status/id/version = %d/%d/%v", update.Code, service.updatedID, service.updatedVer)
	}

	deleteValues := url.Values{"row_version": {version}}
	deleteRequest := httptest.NewRequest(http.MethodPost, "/supplier-payments/42/delete", strings.NewReader(deleteValues.Encode()))
	deleteRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	deleted := httptest.NewRecorder()
	mux.ServeHTTP(deleted, deleteRequest)
	if deleted.Code != http.StatusSeeOther || service.voidedID != 42 || string(service.voidedVer) != string([]byte{1, 2, 3}) {
		t.Fatalf("delete status/id/version = %d/%d/%v", deleted.Code, service.voidedID, service.voidedVer)
	}
}

func TestDueLabelsUseBusinessDate(t *testing.T) {
	now, _ := businessdate.Parse("2026-05-12")
	due, _ := businessdate.Parse("2026-05-13")
	result := buildListResult([]Payment{{ID: 1, Status: "Active", DueDateUTC: due}}, now.Add(7*time.Hour), false)
	if len(result.Items) != 1 || result.Items[0].DueLabel != "Due tomorrow" {
		t.Fatalf("result = %#v", result.Items)
	}
}

func testHandler(t *testing.T, service Application) *Handler {
	t.Helper()
	handler, err := NewHandler(service)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	return handler
}
