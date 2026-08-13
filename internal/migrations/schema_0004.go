package migrations

import "gorm.io/gorm"

func up0004(db *gorm.DB) error {
	for _, statement := range []string{
		`IF COL_LENGTH('dbo.delivery_receivables', 'invoice_number') IS NULL

         ALTER TABLE dbo.delivery_receivables
         ADD invoice_number VARCHAR(100) NOT NULL
             CONSTRAINT DF_delivery_receivables_invoice_number DEFAULT ('0000') WITH VALUES;`,
		`ALTER TABLE dbo.delivery_receivables ADD CONSTRAINT CK_delivery_receivables_invoice_number CHECK (LTRIM(RTRIM(invoice_number)) <> '');`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func down0004(db *gorm.DB) error {
	for _, statement := range []string{
		`ALTER TABLE dbo.delivery_receivables DROP CONSTRAINT CK_delivery_receivables_invoice_number;`,
		`ALTER TABLE dbo.delivery_receivables DROP CONSTRAINT DF_delivery_receivables_invoice_number;`,
		`ALTER TABLE dbo.delivery_receivables DROP COLUMN invoice_number;`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
