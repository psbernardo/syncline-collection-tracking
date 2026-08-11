package dashboard

import "testing"

func TestDashboardViewModelFormatsTotals(t *testing.T) {
	view := (DashboardTotals{
		Overdue: ClassificationTotal{ReceivableCount: 2, ClientCount: 1, AmountScaled: 98_003_439},
		Clients: []ClientReceivableSummary{{CompanyName: "Acme Foods", TotalCount: 2, OverdueCount: 2, PaymentReceivedCount: 0}},
	}).ViewModel()
	if view.Overdue.AmountDisplay != "9800.34" {
		t.Fatalf("amount = %q, want 9800.34", view.Overdue.AmountDisplay)
	}
	if view.Overdue.ReceivableCount != 2 || view.Overdue.ClientCount != 1 {
		t.Fatalf("counts = %+v", view.Overdue)
	}
	if len(view.Clients) != 1 || view.Clients[0].CompanyName != "Acme Foods" || view.Clients[0].OverdueCount != 2 {
		t.Fatalf("client summaries = %+v", view.Clients)
	}
}
