package migrations

import "gorm.io/gorm"

func up0028(db *gorm.DB) error {
	for _, statement := range []string{
		`UPDATE dbo.sales_orders SET status = 'CONVERTED' WHERE status = 'COMPLETED';`,
		`ALTER TABLE dbo.sales_orders DROP CONSTRAINT CK_sales_orders_status;`,
		`ALTER TABLE dbo.sales_orders ADD CONSTRAINT CK_sales_orders_status CHECK (status IN ('OPEN', 'CONVERTED'));`,
		`IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = 'UX_delivery_receivables_invoice_id_not_cancelled' AND object_id = OBJECT_ID('dbo.delivery_receivables'))
		 CREATE UNIQUE INDEX UX_delivery_receivables_invoice_id_not_cancelled
         ON dbo.delivery_receivables (invoice_id)
         WHERE invoice_id IS NOT NULL AND lifecycle_status IN ('Active', 'Archived');`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func down0028(db *gorm.DB) error {
	for _, statement := range []string{
		`DROP INDEX IF EXISTS UX_delivery_receivables_invoice_id_not_cancelled ON dbo.delivery_receivables;`,
		`ALTER TABLE dbo.sales_orders DROP CONSTRAINT CK_sales_orders_status;`,
		`ALTER TABLE dbo.sales_orders ADD CONSTRAINT CK_sales_orders_status CHECK (status IN ('OPEN', 'COMPLETED', 'CONVERTED'));`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
