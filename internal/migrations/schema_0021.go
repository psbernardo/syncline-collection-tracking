package migrations

import "gorm.io/gorm"

func up0021(db *gorm.DB) error {
	for _, statement := range []string{
		`IF COL_LENGTH('dbo.delivery_receivables', 'invoice_id') IS NULL
         ALTER TABLE dbo.delivery_receivables ADD invoice_id BIGINT NULL;`,
		`IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = 'FK_delivery_receivables_invoice')
         ALTER TABLE dbo.delivery_receivables ADD CONSTRAINT FK_delivery_receivables_invoice
         FOREIGN KEY (invoice_id) REFERENCES dbo.invoices(invoice_id);`,
		`UPDATE r
         SET invoice_id = i.invoice_id
         FROM dbo.delivery_receivables AS r
         INNER JOIN dbo.invoices AS i
             ON i.invoice_number = r.invoice_number
            AND i.company_account_id = r.company_account_id
         WHERE r.invoice_id IS NULL
           AND (SELECT COUNT(*) FROM dbo.invoices AS match
                WHERE match.invoice_number = r.invoice_number
                  AND match.company_account_id = r.company_account_id) = 1;`,
		`IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = 'IX_delivery_receivables_invoice_id' AND object_id = OBJECT_ID('dbo.delivery_receivables'))
         CREATE INDEX IX_delivery_receivables_invoice_id ON dbo.delivery_receivables (invoice_id, delivery_receivable_id);`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func down0021(db *gorm.DB) error {
	for _, statement := range []string{
		`DROP INDEX IF EXISTS IX_delivery_receivables_invoice_id ON dbo.delivery_receivables;`,
		`IF EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = 'FK_delivery_receivables_invoice')
         ALTER TABLE dbo.delivery_receivables DROP CONSTRAINT FK_delivery_receivables_invoice;`,
		`IF COL_LENGTH('dbo.delivery_receivables', 'invoice_id') IS NOT NULL
         ALTER TABLE dbo.delivery_receivables DROP COLUMN invoice_id;`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
