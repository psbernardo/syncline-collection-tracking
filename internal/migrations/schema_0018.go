package migrations

import "gorm.io/gorm"

func up0018(db *gorm.DB) error {
	for _, statement := range []string{
		`UPDATE dbo.quotations SET status = 'APPROVED' WHERE status = 'ACCEPTED';`,
		`ALTER TABLE dbo.quotations DROP CONSTRAINT CK_quotations_status;`,
		`ALTER TABLE dbo.quotations ADD CONSTRAINT CK_quotations_status CHECK (status IN ('DRAFT', 'SENT', 'APPROVED', 'EXPIRED', 'REJECTED', 'CANCELLED'));`,
		`ALTER TABLE dbo.sales_orders DROP CONSTRAINT UQ_sales_orders_quotation;`,
		`ALTER TABLE dbo.sales_orders ALTER COLUMN quotation_id BIGINT NULL;`,
		`ALTER TABLE dbo.sales_order_lines ADD quotation_line_id BIGINT NULL;`,
		`ALTER TABLE dbo.sales_order_lines ADD CONSTRAINT FK_sales_order_lines_quotation_lines FOREIGN KEY (quotation_line_id) REFERENCES dbo.quotation_lines(quotation_line_id);`,
		`ALTER TABLE dbo.sales_order_lines ADD sku VARCHAR(50) NOT NULL CONSTRAINT DF_sales_order_lines_sku DEFAULT (''), name NVARCHAR(255) NOT NULL CONSTRAINT DF_sales_order_lines_name DEFAULT ('');`,
		`UPDATE sol SET sku = p.sku, name = p.name FROM dbo.sales_order_lines AS sol JOIN dbo.products AS p ON p.product_id = sol.product_id;`,
		`CREATE TABLE dbo.quotation_line_allocations (
            quotation_line_allocation_id BIGINT IDENTITY(1,1) NOT NULL CONSTRAINT PK_quotation_line_allocations PRIMARY KEY,
            quotation_id BIGINT NOT NULL,
            quotation_line_id BIGINT NOT NULL,
            sales_order_id BIGINT NOT NULL,
            sales_order_line_id BIGINT NOT NULL,
            quantity_scaled BIGINT NOT NULL,
            created_at_utc DATETIME2(0) NOT NULL CONSTRAINT DF_quotation_line_allocations_created_at DEFAULT (SYSUTCDATETIME()),
            CONSTRAINT FK_allocations_quotations FOREIGN KEY (quotation_id) REFERENCES dbo.quotations(quotation_id),
            CONSTRAINT FK_allocations_quotation_lines FOREIGN KEY (quotation_line_id) REFERENCES dbo.quotation_lines(quotation_line_id),
            CONSTRAINT FK_allocations_sales_orders FOREIGN KEY (sales_order_id) REFERENCES dbo.sales_orders(sales_order_id),
            CONSTRAINT FK_allocations_sales_order_lines FOREIGN KEY (sales_order_line_id) REFERENCES dbo.sales_order_lines(sales_order_line_id),
            CONSTRAINT CK_allocations_quantity CHECK (quantity_scaled > 0)
        );`,
		`CREATE INDEX IX_allocations_quotation_line ON dbo.quotation_line_allocations (quotation_line_id, sales_order_id);`,
		`CREATE INDEX IX_allocations_sales_order ON dbo.quotation_line_allocations (sales_order_id, quotation_line_id);`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func down0018(db *gorm.DB) error {
	for _, statement := range []string{
		`DROP INDEX IX_allocations_sales_order ON dbo.quotation_line_allocations;`,
		`DROP INDEX IX_allocations_quotation_line ON dbo.quotation_line_allocations;`,
		`DROP TABLE dbo.quotation_line_allocations;`,
		`ALTER TABLE dbo.sales_order_lines DROP CONSTRAINT FK_sales_order_lines_quotation_lines;`,
		`ALTER TABLE dbo.sales_order_lines DROP CONSTRAINT DF_sales_order_lines_sku, DF_sales_order_lines_name;`,
		`ALTER TABLE dbo.sales_order_lines DROP COLUMN sku, name;`,
		`ALTER TABLE dbo.sales_order_lines DROP COLUMN quotation_line_id;`,
		`ALTER TABLE dbo.sales_orders ALTER COLUMN quotation_id BIGINT NOT NULL;`,
		`ALTER TABLE dbo.sales_orders ADD CONSTRAINT UQ_sales_orders_quotation UNIQUE (quotation_id);`,
		`ALTER TABLE dbo.quotations DROP CONSTRAINT CK_quotations_status;`,
		`ALTER TABLE dbo.quotations ADD CONSTRAINT CK_quotations_status CHECK (status IN ('DRAFT', 'SENT', 'ACCEPTED', 'EXPIRED', 'REJECTED', 'CANCELLED'));`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
