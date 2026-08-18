package migrations

import "gorm.io/gorm"

func up0009(db *gorm.DB) error {
	return db.Exec(`CREATE SEQUENCE dbo.SEQ_quotation_numbers AS BIGINT START WITH 1 INCREMENT BY 1;`).Error
}

func down0009(db *gorm.DB) error {
	return db.Exec(`DROP SEQUENCE dbo.SEQ_quotation_numbers;`).Error
}
