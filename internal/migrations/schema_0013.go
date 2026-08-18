package migrations

import "gorm.io/gorm"

func up0013(db *gorm.DB) error {
	return db.Exec(`ALTER TABLE dbo.quotations ADD withholding_tax_scaled BIGINT NOT NULL CONSTRAINT DF_quotations_withholding_tax DEFAULT (0);`).Error
}

func down0013(db *gorm.DB) error {
	return db.Exec(`ALTER TABLE dbo.quotations DROP CONSTRAINT DF_quotations_withholding_tax; ALTER TABLE dbo.quotations DROP COLUMN withholding_tax_scaled;`).Error
}
