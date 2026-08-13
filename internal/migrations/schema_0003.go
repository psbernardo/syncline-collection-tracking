package migrations

import "gorm.io/gorm"

func up0003(db *gorm.DB) error {
	for _, statement := range []string{
		`IF EXISTS (
            SELECT po_number
            FROM dbo.delivery_receivables
            WHERE lifecycle_status <> 'Cancelled'
            GROUP BY po_number
            HAVING COUNT(*) > 1
        ) THROW 51003, 'Resolve duplicate non-cancelled PO numbers before applying migration 0003', 1;`,
		`ALTER TABLE dbo.delivery_receivables ADD po_number_normalized VARCHAR(100) NULL;`,
		`UPDATE dbo.delivery_receivables SET po_number_normalized = UPPER(LTRIM(RTRIM(po_number)));`,
		`ALTER TABLE dbo.delivery_receivables ALTER COLUMN po_number_normalized VARCHAR(100) NOT NULL;`,
		`ALTER TABLE dbo.delivery_receivables ADD CONSTRAINT CK_delivery_receivables_po_number_normalized CHECK (po_number_normalized <> '');`,
		`CREATE UNIQUE INDEX UX_delivery_receivables_po_not_cancelled ON dbo.delivery_receivables (po_number_normalized) WHERE lifecycle_status IN ('Active', 'Archived');`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func down0003(db *gorm.DB) error {
	for _, statement := range []string{
		`DROP INDEX UX_delivery_receivables_po_not_cancelled ON dbo.delivery_receivables;`,
		`ALTER TABLE dbo.delivery_receivables DROP CONSTRAINT CK_delivery_receivables_po_number_normalized;`,
		`ALTER TABLE dbo.delivery_receivables DROP COLUMN po_number_normalized;`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
