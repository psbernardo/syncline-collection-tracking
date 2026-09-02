package migrations

import "gorm.io/gorm"

func up0023(db *gorm.DB) error {
	for _, statement := range []string{
		`ALTER TABLE dbo.purchase_orders DROP CONSTRAINT CK_purchase_orders_status;`,
		`ALTER TABLE dbo.purchase_orders ADD CONSTRAINT CK_purchase_orders_status CHECK (status IN ('OPEN', 'CONFIRMED', 'RECEIVING', 'RECEIVED', 'COMPLETED', 'CANCELLED'));`,
		`ALTER TABLE dbo.purchase_orders ADD payment_terms NVARCHAR(255) NULL, expected_delivery_date DATE NULL, notes NVARCHAR(2000) NULL;`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func down0023(db *gorm.DB) error {
	for _, statement := range []string{
		`ALTER TABLE dbo.purchase_orders DROP CONSTRAINT CK_purchase_orders_status;`,
		`ALTER TABLE dbo.purchase_orders ADD CONSTRAINT CK_purchase_orders_status CHECK (status IN ('OPEN', 'COMPLETED', 'CANCELLED'));`,
		`ALTER TABLE dbo.purchase_orders DROP COLUMN payment_terms, expected_delivery_date, notes;`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
