package migrations

import "gorm.io/gorm"

func up0015(db *gorm.DB) error {
	for _, statement := range []string{
		`CREATE SEQUENCE dbo.SEQ_sales_order_numbers AS BIGINT START WITH 1 INCREMENT BY 1;`,
		`CREATE TABLE dbo.sales_orders (
            sales_order_id BIGINT IDENTITY(1,1) NOT NULL CONSTRAINT PK_sales_orders PRIMARY KEY,
            sales_order_number VARCHAR(50) NOT NULL CONSTRAINT UQ_sales_orders_number UNIQUE,
            quotation_id BIGINT NOT NULL,
            company_account_id BIGINT NOT NULL,
            customer_po_number NVARCHAR(100) NULL,
            sales_person NVARCHAR(255) NOT NULL,
            terms_days INT NOT NULL,
            status VARCHAR(20) NOT NULL CONSTRAINT DF_sales_orders_status DEFAULT ('OPEN'),
            subtotal_scaled BIGINT NOT NULL,
            tax_scaled BIGINT NOT NULL,
            total_scaled BIGINT NOT NULL,
            created_at_utc DATETIME2(0) NOT NULL CONSTRAINT DF_sales_orders_created_at_utc DEFAULT (SYSUTCDATETIME()),
            updated_at_utc DATETIME2(0) NOT NULL CONSTRAINT DF_sales_orders_updated_at_utc DEFAULT (SYSUTCDATETIME()),
            CONSTRAINT FK_sales_orders_quotations FOREIGN KEY (quotation_id) REFERENCES dbo.quotations(quotation_id),
            CONSTRAINT FK_sales_orders_company_accounts FOREIGN KEY (company_account_id) REFERENCES dbo.company_accounts(company_account_id),
            CONSTRAINT UQ_sales_orders_quotation UNIQUE (quotation_id),
            CONSTRAINT CK_sales_orders_status CHECK (status IN ('OPEN', 'COMPLETED'))
        );`,
		`CREATE TABLE dbo.sales_order_lines (
            sales_order_line_id BIGINT IDENTITY(1,1) NOT NULL CONSTRAINT PK_sales_order_lines PRIMARY KEY,
            sales_order_id BIGINT NOT NULL,
            product_id BIGINT NOT NULL,
            quantity_scaled BIGINT NOT NULL,
            uom VARCHAR(20) NOT NULL,
            unit_price_scaled BIGINT NOT NULL,
            tax_rate_scaled BIGINT NOT NULL,
            tax_code VARCHAR(50) NOT NULL,
            line_total_scaled BIGINT NOT NULL,
            vat_inclusive_total_scaled BIGINT NOT NULL,
            CONSTRAINT FK_sales_order_lines_orders FOREIGN KEY (sales_order_id) REFERENCES dbo.sales_orders(sales_order_id),
            CONSTRAINT FK_sales_order_lines_products FOREIGN KEY (product_id) REFERENCES dbo.products(product_id)
        );`,
		`CREATE INDEX IX_sales_orders_quotation ON dbo.sales_orders (quotation_id, sales_order_id);`,
		`CREATE INDEX IX_sales_order_lines_order ON dbo.sales_order_lines (sales_order_id, sales_order_line_id);`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func down0015(db *gorm.DB) error {
	for _, statement := range []string{
		`DROP TABLE dbo.sales_order_lines;`,
		`DROP TABLE dbo.sales_orders;`,
		`DROP SEQUENCE dbo.SEQ_sales_order_numbers;`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
