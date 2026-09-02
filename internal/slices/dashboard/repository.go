package dashboard

import "context"

type QueryRepository interface {
	GetTotals(ctx context.Context, query Query) (DashboardTotals, error)
}

type SalesQueryRepository interface {
	GetSalesDashboard(ctx context.Context, query Query) (SalesDashboard, error)
}
