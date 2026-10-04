package migrations

import "gorm.io/gorm"

func up0030(db *gorm.DB) error {
	for _, statement := range []string{
		`IF EXISTS (
            SELECT check_number FROM dbo.supplier_payments
            GROUP BY check_number HAVING COUNT(*) > 1
        ) THROW 51030, 'Duplicate supplier payment check numbers must be resolved before migration 30', 1;`,
		`DROP INDEX UX_supplier_payments_supplier_check_active ON dbo.supplier_payments;`,
		`CREATE UNIQUE INDEX UX_supplier_payments_check_number
            ON dbo.supplier_payments (check_number);`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func down0030(db *gorm.DB) error {
	for _, statement := range []string{
		`DROP INDEX UX_supplier_payments_check_number ON dbo.supplier_payments;`,
		`CREATE UNIQUE INDEX UX_supplier_payments_supplier_check_active
            ON dbo.supplier_payments (supplier_id, check_number)
            WHERE lifecycle_status = 'Active';`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
