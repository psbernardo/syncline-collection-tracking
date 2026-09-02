package migrations

import "gorm.io/gorm"

func up0026(db *gorm.DB) error {
	for _, statement := range []string{
		`ALTER TABLE dbo.delivery_receivables DROP CONSTRAINT CK_delivery_receivables_invoice_number;`,
		`ALTER TABLE dbo.delivery_receivables ALTER COLUMN invoice_number VARCHAR(100) NULL;`,
		`DROP INDEX UX_delivery_receivables_invoice_not_cancelled ON dbo.delivery_receivables;`,
		`CREATE UNIQUE INDEX UX_delivery_receivables_invoice_not_cancelled ON dbo.delivery_receivables (invoice_number) WHERE lifecycle_status IN ('Active', 'Archived') AND invoice_number IS NOT NULL AND invoice_number <> '';`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func down0026(db *gorm.DB) error {
	for _, statement := range []string{
		`DROP INDEX UX_delivery_receivables_invoice_not_cancelled ON dbo.delivery_receivables;`,
		`IF EXISTS (SELECT 1 FROM dbo.delivery_receivables WHERE invoice_number IS NULL) THROW 51006, 'Cannot restore required receivable invoice numbers while NULL values exist', 1;`,
		`ALTER TABLE dbo.delivery_receivables ALTER COLUMN invoice_number VARCHAR(100) NOT NULL;`,
		`ALTER TABLE dbo.delivery_receivables ADD CONSTRAINT CK_delivery_receivables_invoice_number CHECK (LTRIM(RTRIM(invoice_number)) <> '');`,
		`CREATE UNIQUE INDEX UX_delivery_receivables_invoice_not_cancelled ON dbo.delivery_receivables (invoice_number) WHERE lifecycle_status IN ('Active', 'Archived');`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
