package migrations

import "gorm.io/gorm"

func up0020(db *gorm.DB) error {
	for _, statement := range []string{
		`ALTER TABLE dbo.sales_orders DROP CONSTRAINT CK_sales_orders_status;`,
		`ALTER TABLE dbo.sales_orders ADD CONSTRAINT CK_sales_orders_status CHECK (status IN ('OPEN', 'COMPLETED', 'CONVERTED'));`,
		`CREATE TABLE dbo.invoices (
            invoice_id BIGINT IDENTITY(1,1) NOT NULL CONSTRAINT PK_invoices PRIMARY KEY,
            invoice_number NVARCHAR(100) NOT NULL,
            sales_order_id BIGINT NOT NULL,
            sales_order_number VARCHAR(50) NOT NULL,
            company_account_id BIGINT NOT NULL,
            customer_name NVARCHAR(255) NOT NULL,
            billing_address NVARCHAR(MAX) NOT NULL,
            delivery_address NVARCHAR(MAX) NOT NULL,
            contact_person NVARCHAR(255) NOT NULL,
            contact_number NVARCHAR(100) NOT NULL,
            email NVARCHAR(255) NOT NULL,
            customer_po_number NVARCHAR(100) NOT NULL,
            sales_person NVARCHAR(255) NOT NULL,
            terms_days INT NOT NULL,
            invoice_date_utc DATETIME2(0) NOT NULL,
            due_date_utc DATETIME2(0) NOT NULL,
            subtotal_scaled BIGINT NOT NULL,
            tax_scaled BIGINT NOT NULL,
            total_scaled BIGINT NOT NULL,
            status VARCHAR(20) NOT NULL CONSTRAINT DF_invoices_status DEFAULT ('POSTED'),
            created_at_utc DATETIME2(0) NOT NULL CONSTRAINT DF_invoices_created_at DEFAULT (SYSUTCDATETIME()),
            updated_at_utc DATETIME2(0) NOT NULL CONSTRAINT DF_invoices_updated_at DEFAULT (SYSUTCDATETIME()),
            CONSTRAINT FK_invoices_sales_orders FOREIGN KEY (sales_order_id) REFERENCES dbo.sales_orders(sales_order_id),
            CONSTRAINT FK_invoices_company_accounts FOREIGN KEY (company_account_id) REFERENCES dbo.company_accounts(company_account_id),
            CONSTRAINT UQ_invoices_sales_order UNIQUE (sales_order_id),
            CONSTRAINT UQ_invoices_number UNIQUE (invoice_number),
            CONSTRAINT CK_invoices_status CHECK (status IN ('POSTED', 'VOIDED'))
        );`,
		`CREATE TABLE dbo.invoice_lines (
            invoice_line_id BIGINT IDENTITY(1,1) NOT NULL CONSTRAINT PK_invoice_lines PRIMARY KEY,
            invoice_id BIGINT NOT NULL,
            sales_order_line_id BIGINT NOT NULL,
            product_id BIGINT NOT NULL,
            sku VARCHAR(50) NOT NULL,
            name NVARCHAR(255) NOT NULL,
            quantity_scaled BIGINT NOT NULL,
            uom VARCHAR(20) NOT NULL,
            unit_price_scaled BIGINT NOT NULL,
            tax_rate_scaled BIGINT NOT NULL,
            tax_code VARCHAR(50) NOT NULL,
            line_total_scaled BIGINT NOT NULL,
            vat_inclusive_total_scaled BIGINT NOT NULL,
            CONSTRAINT FK_invoice_lines_invoices FOREIGN KEY (invoice_id) REFERENCES dbo.invoices(invoice_id),
            CONSTRAINT FK_invoice_lines_sales_order_lines FOREIGN KEY (sales_order_line_id) REFERENCES dbo.sales_order_lines(sales_order_line_id),
            CONSTRAINT FK_invoice_lines_products FOREIGN KEY (product_id) REFERENCES dbo.products(product_id),
            CONSTRAINT UQ_invoice_lines_source UNIQUE (invoice_id, sales_order_line_id),
            CONSTRAINT CK_invoice_lines_quantity CHECK (quantity_scaled > 0)
        );`,
		`CREATE INDEX IX_invoice_lines_invoice ON dbo.invoice_lines (invoice_id, invoice_line_id);`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func down0020(db *gorm.DB) error {
	for _, statement := range []string{
		`DROP TABLE dbo.invoice_lines;`,
		`DROP TABLE dbo.invoices;`,
		`ALTER TABLE dbo.sales_orders DROP CONSTRAINT CK_sales_orders_status;`,
		`ALTER TABLE dbo.sales_orders ADD CONSTRAINT CK_sales_orders_status CHECK (status IN ('OPEN', 'COMPLETED'));`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
