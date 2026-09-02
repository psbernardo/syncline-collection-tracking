package migrations

import "gorm.io/gorm"

func up0022(db *gorm.DB) error {
	for _, statement := range []string{
		`CREATE SEQUENCE dbo.SEQ_purchase_order_numbers AS BIGINT START WITH 1 INCREMENT BY 1;`,
		`CREATE TABLE dbo.purchase_orders (
            purchase_order_id BIGINT IDENTITY(1,1) NOT NULL CONSTRAINT PK_purchase_orders PRIMARY KEY,
            purchase_order_number VARCHAR(50) NOT NULL CONSTRAINT UQ_purchase_orders_number UNIQUE,
            supplier_id BIGINT NOT NULL,
            sales_order_id BIGINT NULL,
            po_date DATE NOT NULL,
            status VARCHAR(20) NOT NULL CONSTRAINT DF_purchase_orders_status DEFAULT ('OPEN'),
            subtotal_scaled BIGINT NOT NULL,
            total_scaled BIGINT NOT NULL,
            created_at_utc DATETIME2(0) NOT NULL CONSTRAINT DF_purchase_orders_created_at DEFAULT (SYSUTCDATETIME()),
            updated_at_utc DATETIME2(0) NOT NULL CONSTRAINT DF_purchase_orders_updated_at DEFAULT (SYSUTCDATETIME()),
            CONSTRAINT FK_purchase_orders_suppliers FOREIGN KEY (supplier_id) REFERENCES dbo.suppliers(supplier_id),
            CONSTRAINT FK_purchase_orders_sales_orders FOREIGN KEY (sales_order_id) REFERENCES dbo.sales_orders(sales_order_id),
            CONSTRAINT CK_purchase_orders_status CHECK (status IN ('OPEN', 'COMPLETED', 'CANCELLED')),
            CONSTRAINT CK_purchase_orders_amounts CHECK (subtotal_scaled >= 0 AND total_scaled >= 0)
        );`,
		`CREATE TABLE dbo.purchase_order_lines (
            purchase_order_line_id BIGINT IDENTITY(1,1) NOT NULL CONSTRAINT PK_purchase_order_lines PRIMARY KEY,
            purchase_order_id BIGINT NOT NULL,
            sales_order_line_id BIGINT NOT NULL,
            product_id BIGINT NOT NULL,
            supplier_product_id BIGINT NULL,
            sku VARCHAR(50) NOT NULL,
            name NVARCHAR(255) NOT NULL,
            supplier_sku VARCHAR(100) NULL,
            uom VARCHAR(20) NOT NULL,
            quantity_scaled BIGINT NOT NULL,
            unit_cost_scaled BIGINT NOT NULL,
            line_total_scaled BIGINT NOT NULL,
            CONSTRAINT FK_purchase_order_lines_orders FOREIGN KEY (purchase_order_id) REFERENCES dbo.purchase_orders(purchase_order_id),
            CONSTRAINT FK_purchase_order_lines_sales_lines FOREIGN KEY (sales_order_line_id) REFERENCES dbo.sales_order_lines(sales_order_line_id),
            CONSTRAINT FK_purchase_order_lines_products FOREIGN KEY (product_id) REFERENCES dbo.products(product_id),
            CONSTRAINT FK_purchase_order_lines_supplier_products FOREIGN KEY (supplier_product_id) REFERENCES dbo.supplier_products(supplier_product_id),
            CONSTRAINT CK_purchase_order_lines_amounts CHECK (quantity_scaled > 0 AND unit_cost_scaled >= 0 AND line_total_scaled >= 0)
        );`,
		`CREATE INDEX IX_purchase_orders_sales_order ON dbo.purchase_orders (sales_order_id, purchase_order_id);`,
		`CREATE INDEX IX_purchase_orders_supplier ON dbo.purchase_orders (supplier_id, purchase_order_id);`,
		`CREATE INDEX IX_purchase_order_lines_order ON dbo.purchase_order_lines (purchase_order_id, purchase_order_line_id);`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func down0022(db *gorm.DB) error {
	for _, statement := range []string{
		`DROP TABLE dbo.purchase_order_lines;`,
		`DROP TABLE dbo.purchase_orders;`,
		`DROP SEQUENCE dbo.SEQ_purchase_order_numbers;`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
