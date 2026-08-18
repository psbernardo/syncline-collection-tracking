package migrations

import "gorm.io/gorm"

func up0014(db *gorm.DB) error {
	return db.Exec(`ALTER TABLE dbo.company_accounts ADD email NVARCHAR(255) NULL;`).Error
}

func down0014(db *gorm.DB) error {
	return db.Exec(`ALTER TABLE dbo.company_accounts DROP COLUMN email;`).Error
}
