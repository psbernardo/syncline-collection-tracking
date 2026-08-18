package migrations

import "gorm.io/gorm"

func up0010(db *gorm.DB) error {
	for _, statement := range []string{
		`ALTER TABLE dbo.quotation_lines ADD CONSTRAINT CK_quotation_lines_uom CHECK (uom IN ('PC', 'BOX', 'SET'));`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func down0010(db *gorm.DB) error {
	for _, statement := range []string{
		`ALTER TABLE dbo.quotation_lines DROP CONSTRAINT CK_quotation_lines_uom;`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
