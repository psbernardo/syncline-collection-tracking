package migrations

import "gorm.io/gorm"

func up0002(db *gorm.DB) error {
	return db.Exec(`ALTER TABLE dbo.company_accounts ADD row_version ROWVERSION NOT NULL;`).Error
}

func down0002(db *gorm.DB) error {
	return db.Exec(`ALTER TABLE dbo.company_accounts DROP COLUMN row_version;`).Error
}
