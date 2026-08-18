package migrations

import "gorm.io/gorm"

func up0008(db *gorm.DB) error {
	for _, statement := range []string{
		`CREATE TABLE dbo.quotations (
            quotation_id BIGINT IDENTITY(1,1) NOT NULL CONSTRAINT PK_quotations PRIMARY KEY,
            quotation_number VARCHAR(50) NOT NULL,
            company_account_id BIGINT NOT NULL,
            validity_date DATE NULL,
            payment_terms NVARCHAR(255) NULL,
            delivery_terms NVARCHAR(255) NULL,
            tax_default_code VARCHAR(50) NOT NULL CONSTRAINT DF_quotations_tax_default_code DEFAULT (''),
            status VARCHAR(20) NOT NULL CONSTRAINT DF_quotations_status DEFAULT ('DRAFT'),
            notes NVARCHAR(MAX) NULL,
            commission_type VARCHAR(30) NOT NULL CONSTRAINT DF_quotations_commission_type DEFAULT ('NONE'),
            commission_rate_scaled BIGINT NOT NULL CONSTRAINT DF_quotations_commission_rate DEFAULT (0),
            commission_amount_scaled BIGINT NOT NULL CONSTRAINT DF_quotations_commission_amount DEFAULT (0),
            estimated_supplier_cost_scaled BIGINT NOT NULL CONSTRAINT DF_quotations_supplier_cost DEFAULT (0),
            subtotal_scaled BIGINT NOT NULL CONSTRAINT DF_quotations_subtotal DEFAULT (0),
            tax_scaled BIGINT NOT NULL CONSTRAINT DF_quotations_tax DEFAULT (0),
            total_scaled BIGINT NOT NULL CONSTRAINT DF_quotations_total DEFAULT (0),
            estimated_profit_before_commission_scaled BIGINT NOT NULL CONSTRAINT DF_quotations_profit_before_commission DEFAULT (0),
            estimated_profit_after_commission_scaled BIGINT NOT NULL CONSTRAINT DF_quotations_profit_after_commission DEFAULT (0),
            created_at_utc DATETIME2(0) NOT NULL CONSTRAINT DF_quotations_created_at_utc DEFAULT (SYSUTCDATETIME()),
            updated_at_utc DATETIME2(0) NOT NULL CONSTRAINT DF_quotations_updated_at_utc DEFAULT (SYSUTCDATETIME()),
            row_version ROWVERSION NOT NULL,
            CONSTRAINT UQ_quotations_number UNIQUE (quotation_number),
            CONSTRAINT FK_quotations_company_accounts FOREIGN KEY (company_account_id) REFERENCES dbo.company_accounts(company_account_id),
            CONSTRAINT CK_quotations_status CHECK (status IN ('DRAFT', 'SENT', 'ACCEPTED', 'EXPIRED', 'REJECTED', 'CANCELLED')),
            CONSTRAINT CK_quotations_commission_type CHECK (commission_type IN ('NONE', 'PER_UNIT', 'PERCENTAGE', 'FIXED_QUOTATION')),
            CONSTRAINT CK_quotations_non_negative CHECK (commission_rate_scaled >= 0 AND commission_amount_scaled >= 0 AND estimated_supplier_cost_scaled >= 0)
        );`,
		`CREATE INDEX IX_quotations_account_status ON dbo.quotations (company_account_id, status, quotation_id);`,
		`CREATE TABLE dbo.quotation_lines (
            quotation_line_id BIGINT IDENTITY(1,1) NOT NULL CONSTRAINT PK_quotation_lines PRIMARY KEY,
            quotation_id BIGINT NOT NULL,
            product_id BIGINT NOT NULL,
            quantity_scaled BIGINT NOT NULL,
            uom VARCHAR(20) NOT NULL,
            unit_price_scaled BIGINT NOT NULL,
            line_total_scaled BIGINT NOT NULL,
            tax_code VARCHAR(50) NOT NULL CONSTRAINT DF_quotation_lines_tax_code DEFAULT (''),
            tax_rate_scaled BIGINT NOT NULL CONSTRAINT DF_quotation_lines_tax_rate DEFAULT (0),
            vat_inclusive_total_scaled BIGINT NOT NULL,
            supplier_cost_scaled BIGINT NOT NULL CONSTRAINT DF_quotation_lines_supplier_cost DEFAULT (0),
            CONSTRAINT FK_quotation_lines_quotations FOREIGN KEY (quotation_id) REFERENCES dbo.quotations(quotation_id),
            CONSTRAINT FK_quotation_lines_products FOREIGN KEY (product_id) REFERENCES dbo.products(product_id),
            CONSTRAINT CK_quotation_lines_quantity CHECK (quantity_scaled > 0),
            CONSTRAINT CK_quotation_lines_amounts CHECK (unit_price_scaled >= 0 AND line_total_scaled >= 0 AND tax_rate_scaled >= 0 AND vat_inclusive_total_scaled >= 0 AND supplier_cost_scaled >= 0)
        );`,
		`CREATE INDEX IX_quotation_lines_quotation ON dbo.quotation_lines (quotation_id, quotation_line_id);`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func down0008(db *gorm.DB) error {
	for _, statement := range []string{`DROP TABLE dbo.quotation_lines;`, `DROP TABLE dbo.quotations;`} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
