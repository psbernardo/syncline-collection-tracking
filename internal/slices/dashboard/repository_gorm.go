package dashboard

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type GormRepository struct{ db *gorm.DB }

func NewGormRepository(db *gorm.DB) *GormRepository { return &GormRepository{db: db} }

type aggregateRow struct {
	Classification  string `gorm:"column:classification"`
	ReceivableCount int64  `gorm:"column:receivable_count"`
	ClientCount     int64  `gorm:"column:client_count"`
	AmountScaled    int64  `gorm:"column:amount_scaled"`
}

type clientAggregateRow struct {
	CompanyAccountID     int64  `gorm:"column:company_account_id"`
	CompanyName          string `gorm:"column:company_name"`
	TotalCount           int64  `gorm:"column:total_count"`
	PendingCount         int64  `gorm:"column:pending_count"`
	NearDueCount         int64  `gorm:"column:near_due_count"`
	OverdueCount         int64  `gorm:"column:overdue_count"`
	PaymentReceivedCount int64  `gorm:"column:payment_received_count"`
}

func (repository *GormRepository) GetTotals(ctx context.Context, query Query) (DashboardTotals, error) {
	today, nearEnd := boundaries(query.Now)
	classificationSQL := classificationExpression()
	sql := fmt.Sprintf(`WITH classified AS (
	        SELECT company_account_id, amount_due_scaled, %s AS classification
	        FROM dbo.delivery_receivables
	        WHERE lifecycle_status = 'Active'
	    )
	    SELECT classification,
	        COUNT_BIG(*) AS receivable_count,
	        COUNT(DISTINCT company_account_id) AS client_count,
	        COALESCE(SUM(amount_due_scaled), 0) AS amount_scaled
	        FROM classified
	        GROUP BY classification`, classificationSQL)
	var rows []aggregateRow
	if err := repository.db.WithContext(ctx).Raw(sql, today, nearEnd).Scan(&rows).Error; err != nil {
		return DashboardTotals{}, fmt.Errorf("load dashboard totals: %w", err)
	}
	var totals DashboardTotals
	for _, row := range rows {
		total := ClassificationTotal{ReceivableCount: row.ReceivableCount, ClientCount: row.ClientCount, AmountScaled: row.AmountScaled}
		switch row.Classification {
		case "Pending":
			totals.Pending = total
		case "Near Due":
			totals.NearDue = total
		case "Overdue":
			totals.Overdue = total
		case "Payment Received":
			totals.PaymentReceived = total
		}
	}

	clientSQL := fmt.Sprintf(`WITH classified AS (
	        SELECT r.company_account_id, a.company_name, %s AS classification
	        FROM dbo.delivery_receivables r
	        INNER JOIN dbo.company_accounts a ON a.company_account_id = r.company_account_id
	        WHERE r.lifecycle_status = 'Active'
	    )
	    SELECT company_account_id, company_name,
	        COUNT_BIG(*) AS total_count,
	        COUNT_BIG(CASE WHEN classification = 'Pending' THEN 1 END) AS pending_count,
	        COUNT_BIG(CASE WHEN classification = 'Near Due' THEN 1 END) AS near_due_count,
	        COUNT_BIG(CASE WHEN classification = 'Overdue' THEN 1 END) AS overdue_count,
	        COUNT_BIG(CASE WHEN classification = 'Payment Received' THEN 1 END) AS payment_received_count
	    FROM classified
	    GROUP BY company_account_id, company_name
	    ORDER BY COUNT_BIG(*) DESC, company_name, company_account_id`, classificationSQL)
	var clientRows []clientAggregateRow
	if err := repository.db.WithContext(ctx).Raw(clientSQL, today, nearEnd).Scan(&clientRows).Error; err != nil {
		return DashboardTotals{}, fmt.Errorf("load dashboard client summaries: %w", err)
	}
	for _, row := range clientRows {
		totals.Clients = append(totals.Clients, ClientReceivableSummary{
			CompanyAccountID:     row.CompanyAccountID,
			CompanyName:          row.CompanyName,
			TotalCount:           row.TotalCount,
			PendingCount:         row.PendingCount,
			NearDueCount:         row.NearDueCount,
			OverdueCount:         row.OverdueCount,
			PaymentReceivedCount: row.PaymentReceivedCount,
		})
	}
	return totals, nil
}

type salesPeriodRow struct {
	Year, Month              int
	OrderCount, AmountScaled int64
}
type salesLeaderRow struct {
	Name                     string
	OrderCount, AmountScaled int64
}

func (repository *GormRepository) GetSalesDashboard(ctx context.Context, query Query) (SalesDashboard, error) {
	now := query.Now.UTC()
	monthStart := time.Date(now.Year(), now.Month()-2, 1, 0, 0, 0, 0, time.UTC)
	yearStart := time.Date(now.Year()-2, 1, 1, 0, 0, 0, 0, time.UTC)
	result := SalesDashboard{}
	var monthly []salesPeriodRow
	if err := repository.db.WithContext(ctx).Raw(`SELECT YEAR(invoice_date_utc) AS year, MONTH(invoice_date_utc) AS month, COUNT_BIG(*) AS order_count, COALESCE(SUM(total_scaled), 0) AS amount_scaled FROM dbo.invoices WHERE status = 'POSTED' AND invoice_date_utc >= ? GROUP BY YEAR(invoice_date_utc), MONTH(invoice_date_utc) ORDER BY year DESC, month DESC`, monthStart).Scan(&monthly).Error; err != nil {
		return SalesDashboard{}, fmt.Errorf("load monthly sales: %w", err)
	}
	monthlyByDate := make(map[string]salesPeriodRow, len(monthly))
	for _, row := range monthly {
		monthlyByDate[fmt.Sprintf("%d-%02d", row.Year, row.Month)] = row
	}
	for offset := 0; offset < 3; offset++ {
		date := monthStart.AddDate(0, offset, 0)
		row := monthlyByDate[date.Format("2006-01")]
		result.Monthly = append(result.Monthly, SalesPeriod{Label: date.Format("January 2006"), AmountScaled: row.AmountScaled, OrderCount: row.OrderCount})
	}
	var yearly []salesPeriodRow
	if err := repository.db.WithContext(ctx).Raw(`SELECT YEAR(invoice_date_utc) AS year, 1 AS month, COUNT_BIG(*) AS order_count, COALESCE(SUM(total_scaled), 0) AS amount_scaled FROM dbo.invoices WHERE status = 'POSTED' AND invoice_date_utc >= ? GROUP BY YEAR(invoice_date_utc) ORDER BY year DESC`, yearStart).Scan(&yearly).Error; err != nil {
		return SalesDashboard{}, fmt.Errorf("load yearly sales: %w", err)
	}
	yearlyByYear := make(map[int]salesPeriodRow, len(yearly))
	for _, row := range yearly {
		yearlyByYear[row.Year] = row
	}
	for offset := 0; offset < 3; offset++ {
		year := now.Year() - offset
		row := yearlyByYear[year]
		result.Yearly = append(result.Yearly, SalesPeriod{Label: fmt.Sprintf("%d", year), AmountScaled: row.AmountScaled, OrderCount: row.OrderCount})
	}
	if err := repository.db.WithContext(ctx).Raw(`SELECT COUNT_BIG(*), COALESCE(SUM(total_scaled), 0) FROM dbo.invoices WHERE status = 'POSTED'`).Row().Scan(&result.Orders, &result.Total); err != nil {
		return SalesDashboard{}, fmt.Errorf("load sales totals: %w", err)
	}
	var customers []salesLeaderRow
	if err := repository.db.WithContext(ctx).Raw(`SELECT TOP 5 customer_name AS name, COUNT_BIG(*) AS order_count, COALESCE(SUM(total_scaled), 0) AS amount_scaled FROM dbo.invoices WHERE status = 'POSTED' AND invoice_date_utc >= ? GROUP BY customer_name ORDER BY SUM(total_scaled) DESC, customer_name`, time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)).Scan(&customers).Error; err != nil {
		return SalesDashboard{}, fmt.Errorf("load top customers: %w", err)
	}
	result.TopCustomers = make([]SalesLeader, 0, len(customers))
	for _, row := range customers {
		result.TopCustomers = append(result.TopCustomers, SalesLeader{Name: row.Name, AmountScaled: row.AmountScaled, OrderCount: row.OrderCount})
	}
	var products []salesLeaderRow
	if err := repository.db.WithContext(ctx).Raw(`SELECT TOP 5 il.name AS name, COUNT(DISTINCT i.invoice_id) AS order_count, COALESCE(SUM(il.vat_inclusive_total_scaled), 0) AS amount_scaled FROM dbo.invoice_lines il INNER JOIN dbo.invoices i ON i.invoice_id = il.invoice_id WHERE i.status = 'POSTED' AND i.invoice_date_utc >= ? GROUP BY il.name ORDER BY SUM(il.vat_inclusive_total_scaled) DESC, il.name`, time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)).Scan(&products).Error; err != nil {
		return SalesDashboard{}, fmt.Errorf("load top products: %w", err)
	}
	result.TopProducts = make([]SalesLeader, 0, len(products))
	for _, row := range products {
		result.TopProducts = append(result.TopProducts, SalesLeader{Name: row.Name, AmountScaled: row.AmountScaled, OrderCount: row.OrderCount})
	}
	return result, nil
}

func (repository *GormRepository) GetCompanyTotals(ctx context.Context, companyAccountID int64, query Query) (CompanyTotals, error) {
	today, nearEnd := boundaries(query.Now)
	classificationSQL := classificationExpression()
	sql := fmt.Sprintf(`WITH classified AS (
	        SELECT amount_due_scaled, %s AS classification
	        FROM dbo.delivery_receivables
	        WHERE lifecycle_status = 'Active' AND company_account_id = ?
	    )
	    SELECT classification,
	        COUNT_BIG(*) AS receivable_count,
	        COUNT(DISTINCT 1) AS client_count,
	        COALESCE(SUM(amount_due_scaled), 0) AS amount_scaled
	        FROM classified
	        GROUP BY classification`, classificationSQL)
	var rows []aggregateRow
	if err := repository.db.WithContext(ctx).Raw(sql, today, nearEnd, companyAccountID).Scan(&rows).Error; err != nil {
		return CompanyTotals{}, fmt.Errorf("load company dashboard totals: %w", err)
	}
	var totals CompanyTotals
	for _, row := range rows {
		total := ClassificationTotal{ReceivableCount: row.ReceivableCount, ClientCount: row.ClientCount, AmountScaled: row.AmountScaled}
		switch row.Classification {
		case "Pending":
			totals.Pending = total
		case "Near Due":
			totals.NearDue = total
		case "Overdue":
			totals.Overdue = total
		case "Payment Received":
			totals.PaymentReceived = total
		}
	}
	return totals, nil
}

func classificationExpression() string {
	return `CASE
        WHEN payment_date_utc IS NOT NULL THEN 'Payment Received'
        WHEN due_date_utc < ? THEN 'Overdue'
        WHEN due_date_utc <= ? THEN 'Near Due'
        ELSE 'Pending'
    END`
}
