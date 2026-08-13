package accounts

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/psbernardo/syncline-collection-tracking/internal/slices/dashboard"
	"gorm.io/gorm"
)

func TestNewAccountFormRenders(t *testing.T) {
	handler, err := NewHandler(NewService(nil, nil))
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/accounts/new", nil)
	handler.newForm(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if !strings.Contains(recorder.Body.String(), "New company account") {
		t.Fatal("form title was not rendered")
	}
	if !strings.Contains(recorder.Body.String(), `name="idempotency_key"`) {
		t.Fatal("idempotency key was not rendered")
	}
	if !strings.Contains(recorder.Body.String(), "Company accounts") || !strings.Contains(recorder.Body.String(), "Receivables") {
		t.Fatal("shared navigation was not rendered")
	}
}

func TestInvalidCreateReturnsHTMXValidationFragment(t *testing.T) {
	handler, err := NewHandler(NewService(nil, nil))
	if err != nil {
		t.Fatal(err)
	}
	form := "idempotency_key=test-key&company_name="
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/accounts", strings.NewReader(form))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("HX-Request", "true")
	handler.create(recorder, request)
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnprocessableEntity)
	}
	if !strings.Contains(recorder.Body.String(), "This field is required.") {
		t.Fatal("validation message was not rendered")
	}
}

func TestEditFormRendersCurrentAccount(t *testing.T) {
	repository := &fakeRepository{account: CompanyAccount{
		ID: 1, CompanyName: "Acme Corp", ContactPerson: "Jane Doe", TINNumber: "TIN-1",
		BillingAddress: "Billing", DeliveryAddress: "Delivery", ContactNumber: "555-0100",
		RowVersion: []byte{1, 2, 3},
	}}
	handler, err := NewHandler(NewService(nil, repository))
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/accounts/1/edit", nil)
	request.SetPathValue("id", "1")
	handler.editForm(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "Edit company account") || !strings.Contains(body, "Acme Corp") {
		t.Fatalf("edit form did not render current account: %s", body)
	}
}

func TestEditFormRendersCompanyReceivablesSummary(t *testing.T) {
	repository := &fakeRepository{account: CompanyAccount{ID: 1, CompanyName: "Acme Corp", RowVersion: []byte{1}}}
	companyTotals := dashboard.NewCompanyTotalsService(fakeCompanyTotalsRepository{})
	handler, err := NewHandler(NewService(nil, repository), companyTotals)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/accounts/1/edit", nil)
	request.SetPathValue("id", "1")
	handler.editForm(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	body := recorder.Body.String()
	for _, value := range []string{"Collection summary", "Total outstanding: ₱600.00", "₱300.00", "3 receivables"} {
		if !strings.Contains(body, value) {
			t.Errorf("edit form does not contain %q", value)
		}
	}
}

type fakeRepository struct {
	account CompanyAccount
}

type fakeCompanyTotalsRepository struct{}

func (fakeCompanyTotalsRepository) GetCompanyTotals(context.Context, int64, dashboard.Query) (dashboard.CompanyTotals, error) {
	return dashboard.CompanyTotals{
		Pending: dashboard.ClassificationTotal{AmountScaled: 1_000_000},
		NearDue: dashboard.ClassificationTotal{AmountScaled: 2_000_000},
		Overdue: dashboard.ClassificationTotal{AmountScaled: 3_000_000, ReceivableCount: 3},
	}, nil
}

func (repository *fakeRepository) Create(_ context.Context, _ *gorm.DB, account CompanyAccount) (CompanyAccount, error) {
	account.ID = 1
	return account, nil
}

func (repository *fakeRepository) List(_ context.Context) ([]CompanyAccount, error) {
	return []CompanyAccount{repository.account}, nil
}

func (repository *fakeRepository) FindByID(_ context.Context, _ *gorm.DB, id int64) (CompanyAccount, error) {
	if id != repository.account.ID {
		return CompanyAccount{}, ErrAccountNotFound
	}
	return repository.account, nil
}

func (repository *fakeRepository) Exists(_ context.Context, _ *gorm.DB, id int64) (bool, error) {
	return id == repository.account.ID, nil
}

func (repository *fakeRepository) Update(_ context.Context, _ *gorm.DB, account CompanyAccount, _ []byte) (CompanyAccount, error) {
	repository.account = account
	return account, nil
}
