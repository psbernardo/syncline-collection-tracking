package dashboard

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fakeRepository struct{}

func (fakeRepository) GetTotals(context.Context, Query) (DashboardTotals, error) {
	return DashboardTotals{
		Overdue: ClassificationTotal{ReceivableCount: 2, ClientCount: 1, AmountScaled: 98_003_439},
		Clients: []ClientReceivableSummary{{
			CompanyAccountID: 7,
			CompanyName:      "Acme Foods",
			TotalCount:       2,
			OverdueCount:     2,
		}},
	}, nil
}

func TestDashboardRendersSummaryCards(t *testing.T) {
	handler, err := NewHandler(NewService(fakeRepository{}))
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	handler.dashboard(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	body := recorder.Body.String()
	for _, value := range []string{"Collection dashboard", "Overdue", "9800.34 PHP", "2 receivables", "1 clients", "Receivables by client", "Acme Foods", "Total receivables", "Details", `href="/receivables?company=Acme`} {
		if !strings.Contains(body, value) {
			t.Errorf("dashboard does not contain %q", value)
		}
	}
}

func TestDashboardSummaryReturnsFragment(t *testing.T) {
	service := NewService(fakeRepository{})
	service.now = func() time.Time { return time.Date(2026, time.August, 10, 0, 0, 0, 0, time.UTC) }
	handler, err := NewHandler(service)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/dashboard/summary", nil)
	handler.summary(recorder, request)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "summary-card") || !strings.Contains(recorder.Body.String(), "Receivables by client") {
		t.Fatalf("unexpected summary response: %d %s", recorder.Code, recorder.Body.String())
	}
}
