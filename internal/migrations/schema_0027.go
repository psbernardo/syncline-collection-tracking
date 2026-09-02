package migrations

import "gorm.io/gorm"

func up0027(db *gorm.DB) error {
	for _, statement := range []string{
		`ALTER TABLE dbo.products DROP CONSTRAINT CK_products_uom;`,
		`ALTER TABLE dbo.products ADD CONSTRAINT CK_products_uom CHECK (uom IN ('PC', 'CASE', 'PACK', 'BTL', 'REAM', 'ROLL', 'BOX'));`,
		`ALTER TABLE dbo.quotation_lines DROP CONSTRAINT CK_quotation_lines_uom;`,
		`ALTER TABLE dbo.quotation_lines ADD CONSTRAINT CK_quotation_lines_uom CHECK (uom IN ('PC', 'CASE', 'PACK', 'BTL', 'REAM', 'ROLL', 'BOX'));`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func down0027(db *gorm.DB) error {
	for _, statement := range []string{
		`IF EXISTS (SELECT 1 FROM dbo.products WHERE uom = 'CASE') OR EXISTS (SELECT 1 FROM dbo.quotation_lines WHERE uom = 'CASE') THROW 51007, 'CASE UOM data must be remediated before migration 27', 1;`,
		`ALTER TABLE dbo.quotation_lines DROP CONSTRAINT CK_quotation_lines_uom;`,
		`ALTER TABLE dbo.quotation_lines ADD CONSTRAINT CK_quotation_lines_uom CHECK (uom IN ('PC', 'PACK', 'BTL', 'REAM', 'ROLL', 'BOX'));`,
		`ALTER TABLE dbo.products DROP CONSTRAINT CK_products_uom;`,
		`ALTER TABLE dbo.products ADD CONSTRAINT CK_products_uom CHECK (uom IN ('PC', 'PACK', 'BTL', 'REAM', 'ROLL', 'BOX'));`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
