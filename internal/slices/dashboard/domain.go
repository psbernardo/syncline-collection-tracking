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
