package dashboard

import (
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/businessdate"
	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
)

type ClassificationTotal struct {
	ReceivableCount int64
	ClientCount     int64
	AmountScaled    int64
}

type DashboardTotals struct {
	Pending         ClassificationTotal
	NearDue         ClassificationTotal
	Overdue         ClassificationTotal
	PaymentReceived ClassificationTotal
	Clients         []ClientReceivableSummary
}

type CompanyTotals struct {
	Pending         ClassificationTotal
	NearDue         ClassificationTotal
	Overdue         ClassificationTotal
	PaymentReceived ClassificationTotal
}

type ClientReceivableSummary struct {
	CompanyAccountID     int64
	CompanyName          string
	TotalCount           int64
	PendingCount         int64
	NearDueCount         int64
	OverdueCount         int64
	PaymentReceivedCount int64
}

type Query struct {
	Now time.Time
}

type SalesDashboard struct {
	Monthly      []SalesPeriod
	Yearly       []SalesPeriod
	Orders       int64
	Total        int64
	TopCustomers []SalesLeader
	TopProducts  []SalesLeader
}

type SalesPeriod struct {
	Label        string
	AmountScaled int64
	OrderCount   int64
}

type SalesLeader struct {
	Name         string
	AmountScaled int64
	OrderCount   int64
}

type SalesDashboardViewModel struct {
	Monthly        []SalesPeriodViewModel
	Yearly         []SalesPeriodViewModel
	Orders         int64
	TotalDisplay   string
	AverageDisplay string
	TopCustomers   []SalesLeaderViewModel
	TopProducts    []SalesLeaderViewModel
}

type SalesPeriodViewModel struct {
	Label         string
	AmountDisplay string
	OrderCount    int64
}

type SalesLeaderViewModel struct {
	Name          string
	AmountDisplay string
	OrderCount    int64
}

func (dashboard SalesDashboard) ViewModel() SalesDashboardViewModel {
	view := SalesDashboardViewModel{Orders: dashboard.Orders, TotalDisplay: formatAmount(dashboard.Total), AverageDisplay: formatAmount(0)}
	if dashboard.Orders > 0 {
		view.AverageDisplay = formatAmount(dashboard.Total / dashboard.Orders)
	}
	for _, period := range dashboard.Monthly {
		view.Monthly = append(view.Monthly, SalesPeriodViewModel{Label: period.Label, AmountDisplay: formatAmount(period.AmountScaled), OrderCount: period.OrderCount})
	}
	for _, period := range dashboard.Yearly {
		view.Yearly = append(view.Yearly, SalesPeriodViewModel{Label: period.Label, AmountDisplay: formatAmount(period.AmountScaled), OrderCount: period.OrderCount})
	}
	for _, leader := range dashboard.TopCustomers {
		view.TopCustomers = append(view.TopCustomers, SalesLeaderViewModel{Name: leader.Name, AmountDisplay: formatAmount(leader.AmountScaled), OrderCount: leader.OrderCount})
	}
	for _, leader := range dashboard.TopProducts {
		view.TopProducts = append(view.TopProducts, SalesLeaderViewModel{Name: leader.Name, AmountDisplay: formatAmount(leader.AmountScaled), OrderCount: leader.OrderCount})
	}
	return view
}

func boundaries(now time.Time) (today, nearEnd time.Time) {
	return businessdate.ClassificationBoundaries(now)
}

func (totals DashboardTotals) ViewModel() DashboardViewModel {
	outstandingAmount := totals.Overdue.AmountScaled + totals.NearDue.AmountScaled + totals.Pending.AmountScaled
	return DashboardViewModel{
		OutstandingAmountDisplay: formatAmount(outstandingAmount),
		Pending:                  card("Pending", "pending", totals.Pending),
		NearDue:                  card("Near due", "near-due", totals.NearDue),
		Overdue:                  card("Overdue", "overdue", totals.Overdue),
		PaymentReceived:          card("Payment received", "payment-received", totals.PaymentReceived),
		Clients:                  totals.Clients,
	}
}

type CompanyTotalsViewModel struct {
	OutstandingAmountDisplay string
	Pending                  SummaryCard
	NearDue                  SummaryCard
	Overdue                  SummaryCard
	PaymentReceived          SummaryCard
}

func (totals CompanyTotals) ViewModel() CompanyTotalsViewModel {
	return CompanyTotalsViewModel{
		OutstandingAmountDisplay: formatAmount(totals.Overdue.AmountScaled + totals.NearDue.AmountScaled + totals.Pending.AmountScaled),
		Pending:                  card("Pending", "pending", totals.Pending),
		NearDue:                  card("Near due", "near-due", totals.NearDue),
		Overdue:                  card("Overdue", "overdue", totals.Overdue),
		PaymentReceived:          card("Payment received", "payment-received", totals.PaymentReceived),
	}
}

type DashboardViewModel struct {
	OutstandingAmountDisplay string
	Pending                  SummaryCard
	NearDue                  SummaryCard
	Overdue                  SummaryCard
	PaymentReceived          SummaryCard
	Clients                  []ClientReceivableSummary
}

type SummaryCard struct {
	Label           string
	Tone            string
	AmountDisplay   string
	ReceivableCount int64
	ClientCount     int64
}

func card(label, tone string, total ClassificationTotal) SummaryCard {
	return SummaryCard{Label: label, Tone: tone, AmountDisplay: formatAmount(total.AmountScaled), ReceivableCount: total.ReceivableCount, ClientCount: total.ClientCount}
}

func formatAmount(scaled int64) string {
	return money.Amount(scaled).FormatPHP()
}
