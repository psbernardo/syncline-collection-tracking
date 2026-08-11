package dashboard

import (
	"context"
	"fmt"

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

func classificationExpression() string {
	return `CASE
        WHEN payment_date_utc IS NOT NULL THEN 'Payment Received'
        WHEN due_date_utc < ? THEN 'Overdue'
        WHEN due_date_utc <= ? THEN 'Near Due'
        ELSE 'Pending'
    END`
}
