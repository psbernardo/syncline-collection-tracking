package dashboard

import "testing"

func TestDashboardViewModelFormatsTotals(t *testing.T) {
	view := (DashboardTotals{
		Pending:         ClassificationTotal{AmountScaled: 1_000_000},
		NearDue:         ClassificationTotal{AmountScaled: 2_000_000},
		Overdue:         ClassificationTotal{ReceivableCount: 2, ClientCount: 1, AmountScaled: 98_003_439},
		PaymentReceived: ClassificationTotal{AmountScaled: 50_000_000},
		Clients:         []ClientReceivableSummary{{CompanyName: "Acme Foods", TotalCount: 2, OverdueCount: 2, PaymentReceivedCount: 0}},
	}).ViewModel()
	if view.OutstandingAmountDisplay != "₱10,100.34" {
		t.Fatalf("outstanding amount = %q, want ₱10,100.34", view.OutstandingAmountDisplay)
	}
	if view.Overdue.AmountDisplay != "₱9,800.34" {
		t.Fatalf("amount = %q, want ₱9,800.34", view.Overdue.AmountDisplay)
	}
	if view.Overdue.ReceivableCount != 2 || view.Overdue.ClientCount != 1 {
		t.Fatalf("counts = %+v", view.Overdue)
	}
	if len(view.Clients) != 1 || view.Clients[0].CompanyName != "Acme Foods" || view.Clients[0].OverdueCount != 2 {
		t.Fatalf("client summaries = %+v", view.Clients)
	}
}

func TestCompanyTotalsViewModelCalculatesOutstandingAmount(t *testing.T) {
	view := (CompanyTotals{
		Pending:         ClassificationTotal{AmountScaled: 1_000_000, ReceivableCount: 1},
		NearDue:         ClassificationTotal{AmountScaled: 2_000_000, ReceivableCount: 2},
		Overdue:         ClassificationTotal{AmountScaled: 3_000_000, ReceivableCount: 3},
		PaymentReceived: ClassificationTotal{AmountScaled: 50_000_000, ReceivableCount: 4},
	}).ViewModel()
	if view.OutstandingAmountDisplay != "₱600.00" {
		t.Fatalf("outstanding amount = %q, want ₱600.00", view.OutstandingAmountDisplay)
	}
	if view.PaymentReceived.AmountDisplay != "₱5,000.00" || view.PaymentReceived.ReceivableCount != 4 {
		t.Fatalf("payment received = %+v", view.PaymentReceived)
	}
}
