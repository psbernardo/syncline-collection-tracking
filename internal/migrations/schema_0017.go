package migrations

import "gorm.io/gorm"

func up0017(db *gorm.DB) error {
	for _, statement := range []string{
		`IF EXISTS (SELECT 1 FROM dbo.products WHERE uom = 'SET') OR EXISTS (SELECT 1 FROM dbo.quotation_lines WHERE uom = 'SET') THROW 51001, 'SET UOM data must be remediated before migration 17', 1;`,
		`ALTER TABLE dbo.products DROP CONSTRAINT CK_products_uom;`,
		`ALTER TABLE dbo.products ADD CONSTRAINT CK_products_uom CHECK (uom IN ('PC', 'PACK', 'BTL', 'REAM', 'ROLL', 'BOX'));`,
		`ALTER TABLE dbo.quotation_lines DROP CONSTRAINT CK_quotation_lines_uom;`,
		`ALTER TABLE dbo.quotation_lines ADD CONSTRAINT CK_quotation_lines_uom CHECK (uom IN ('PC', 'PACK', 'BTL', 'REAM', 'ROLL', 'BOX'));`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func down0017(db *gorm.DB) error {
	for _, statement := range []string{
		`ALTER TABLE dbo.quotation_lines DROP CONSTRAINT CK_quotation_lines_uom;`,
		`ALTER TABLE dbo.quotation_lines ADD CONSTRAINT CK_quotation_lines_uom CHECK (uom IN ('PC', 'BOX', 'SET'));`,
		`ALTER TABLE dbo.products DROP CONSTRAINT CK_products_uom;`,
		`ALTER TABLE dbo.products ADD CONSTRAINT CK_products_uom CHECK (uom IN ('PC', 'BOX', 'SET'));`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
