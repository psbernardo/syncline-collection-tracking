package migrations

import "gorm.io/gorm"

func up0005(db *gorm.DB) error {
	for _, statement := range []string{
		`IF EXISTS (
            SELECT invoice_number
            FROM dbo.delivery_receivables
            WHERE lifecycle_status IN ('Active', 'Archived')
            GROUP BY invoice_number
            HAVING COUNT(*) > 1
        ) THROW 51005, 'Resolve duplicate non-cancelled invoice numbers before applying migration 0005', 1;`,
		`CREATE UNIQUE INDEX UX_delivery_receivables_invoice_not_cancelled ON dbo.delivery_receivables (invoice_number) WHERE lifecycle_status IN ('Active', 'Archived');`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func down0005(db *gorm.DB) error {
	return db.Exec(`DROP INDEX UX_delivery_receivables_invoice_not_cancelled ON dbo.delivery_receivables;`).Error
}
