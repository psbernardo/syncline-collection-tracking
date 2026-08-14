package migrations

import "gorm.io/gorm"

func up0006(db *gorm.DB) error {
	for _, statement := range []string{
		`ALTER TABLE dbo.delivery_receivables ADD gross_amount_scaled BIGINT NULL, tax_rule_code VARCHAR(64) NULL, ewt_amount_scaled BIGINT NOT NULL CONSTRAINT DF_delivery_receivables_ewt_amount_scaled DEFAULT (0), tax_base_scaled BIGINT NOT NULL CONSTRAINT DF_delivery_receivables_tax_base_scaled DEFAULT (0), vat_amount_scaled BIGINT NOT NULL CONSTRAINT DF_delivery_receivables_vat_amount_scaled DEFAULT (0);`,
		`UPDATE dbo.delivery_receivables SET gross_amount_scaled = amount_due_scaled, tax_base_scaled = amount_due_scaled WHERE gross_amount_scaled IS NULL;`,
		`ALTER TABLE dbo.delivery_receivables ALTER COLUMN gross_amount_scaled BIGINT NOT NULL;`,
		`ALTER TABLE dbo.delivery_receivables ADD CONSTRAINT CK_delivery_receivables_gross_amount_scaled CHECK (gross_amount_scaled >= 0), CONSTRAINT CK_delivery_receivables_ewt_amount_scaled CHECK (ewt_amount_scaled >= 0), CONSTRAINT CK_delivery_receivables_tax_base_scaled CHECK (tax_base_scaled >= 0), CONSTRAINT CK_delivery_receivables_vat_amount_scaled CHECK (vat_amount_scaled >= 0);`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func down0006(db *gorm.DB) error {
	for _, statement := range []string{
		`ALTER TABLE dbo.delivery_receivables DROP CONSTRAINT CK_delivery_receivables_gross_amount_scaled, CK_delivery_receivables_ewt_amount_scaled, CK_delivery_receivables_tax_base_scaled, CK_delivery_receivables_vat_amount_scaled;`,
		`ALTER TABLE dbo.delivery_receivables DROP CONSTRAINT DF_delivery_receivables_ewt_amount_scaled, DF_delivery_receivables_tax_base_scaled, DF_delivery_receivables_vat_amount_scaled;`,
		`ALTER TABLE dbo.delivery_receivables DROP COLUMN gross_amount_scaled, tax_rule_code, ewt_amount_scaled, tax_base_scaled, vat_amount_scaled;`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
