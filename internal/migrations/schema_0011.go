package migrations

import "gorm.io/gorm"

func up0011(db *gorm.DB) error {
	for _, statement := range []string{
		`ALTER TABLE dbo.quotations ADD CONSTRAINT CK_quotations_validity_date CHECK (validity_date IS NULL OR validity_date >= CONVERT(date, created_at_utc));`,
		`CREATE INDEX IX_quotations_created_validity ON dbo.quotations (created_at_utc, validity_date, status, quotation_id);`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func down0011(db *gorm.DB) error {
	for _, statement := range []string{
		`DROP INDEX IX_quotations_created_validity ON dbo.quotations;`,
		`ALTER TABLE dbo.quotations DROP CONSTRAINT CK_quotations_validity_date;`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
