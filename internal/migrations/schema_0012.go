package migrations

import "gorm.io/gorm"

func up0012(db *gorm.DB) error {
	for _, statement := range []string{
		`ALTER TABLE dbo.quotations ADD terms_days INT NULL;`,
		`ALTER TABLE dbo.quotations ADD CONSTRAINT CK_quotations_terms_days CHECK (terms_days IS NULL OR terms_days IN (7, 15, 30, 45));`,
		`CREATE INDEX IX_quotations_terms_days ON dbo.quotations (terms_days, quotation_id);`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func down0012(db *gorm.DB) error {
	for _, statement := range []string{
		`DROP INDEX IX_quotations_terms_days ON dbo.quotations;`,
		`ALTER TABLE dbo.quotations DROP CONSTRAINT CK_quotations_terms_days;`,
		`ALTER TABLE dbo.quotations DROP COLUMN terms_days;`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
