package migrations

import "gorm.io/gorm"

func up0016(db *gorm.DB) error {
	for _, statement := range []string{
		`ALTER TABLE dbo.quotations ADD supplier_delivery_cost_scaled BIGINT NOT NULL CONSTRAINT DF_quotations_supplier_delivery_cost DEFAULT (0), customer_delivery_cost_scaled BIGINT NOT NULL CONSTRAINT DF_quotations_customer_delivery_cost DEFAULT (0), other_cost_scaled BIGINT NOT NULL CONSTRAINT DF_quotations_other_cost DEFAULT (0);`,
		`ALTER TABLE dbo.quotation_lines ADD supplier_product_cost_scaled BIGINT NOT NULL CONSTRAINT DF_quotation_lines_supplier_product_cost DEFAULT (0), commission_amount_scaled BIGINT NOT NULL CONSTRAINT DF_quotation_lines_commission_amount DEFAULT (0), profit_before_commission_scaled BIGINT NOT NULL CONSTRAINT DF_quotation_lines_profit_before DEFAULT (0), profit_after_commission_scaled BIGINT NOT NULL CONSTRAINT DF_quotation_lines_profit_after DEFAULT (0);`,
		`ALTER TABLE dbo.quotations ADD CONSTRAINT CK_quotations_profitability_costs CHECK (supplier_delivery_cost_scaled >= 0 AND customer_delivery_cost_scaled >= 0 AND other_cost_scaled >= 0);`,
		`ALTER TABLE dbo.quotation_lines ADD CONSTRAINT CK_quotation_lines_profitability_costs CHECK (supplier_product_cost_scaled >= 0 AND commission_amount_scaled >= 0);`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func down0016(db *gorm.DB) error {
	for _, statement := range []string{
		`ALTER TABLE dbo.quotation_lines DROP CONSTRAINT CK_quotation_lines_profitability_costs;`,
		`ALTER TABLE dbo.quotations DROP CONSTRAINT CK_quotations_profitability_costs;`,
		`ALTER TABLE dbo.quotation_lines DROP CONSTRAINT DF_quotation_lines_supplier_product_cost, DF_quotation_lines_commission_amount, DF_quotation_lines_profit_before, DF_quotation_lines_profit_after;`,
		`ALTER TABLE dbo.quotations DROP CONSTRAINT DF_quotations_supplier_delivery_cost, DF_quotations_customer_delivery_cost, DF_quotations_other_cost;`,
		`ALTER TABLE dbo.quotation_lines DROP COLUMN supplier_product_cost_scaled, commission_amount_scaled, profit_before_commission_scaled, profit_after_commission_scaled;`,
		`ALTER TABLE dbo.quotations DROP COLUMN supplier_delivery_cost_scaled, customer_delivery_cost_scaled, other_cost_scaled;`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
