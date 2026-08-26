package migrations

import "gorm.io/gorm"

func up0024(db *gorm.DB) error {
	for _, statement := range []string{
		`CREATE TABLE dbo.purchase_order_allocations (
            purchase_order_allocation_id BIGINT IDENTITY(1,1) NOT NULL CONSTRAINT PK_purchase_order_allocations PRIMARY KEY,
            purchase_order_line_id BIGINT NOT NULL,
            sales_order_line_id BIGINT NOT NULL,
            quantity_scaled BIGINT NOT NULL,
            created_at_utc DATETIME2(0) NOT NULL CONSTRAINT DF_purchase_order_allocations_created_at DEFAULT (SYSUTCDATETIME()),
            CONSTRAINT FK_purchase_order_allocations_po_line FOREIGN KEY (purchase_order_line_id) REFERENCES dbo.purchase_order_lines(purchase_order_line_id),
            CONSTRAINT FK_purchase_order_allocations_sales_line FOREIGN KEY (sales_order_line_id) REFERENCES dbo.sales_order_lines(sales_order_line_id),
            CONSTRAINT CK_purchase_order_allocations_quantity CHECK (quantity_scaled > 0)
        );`,
		`CREATE UNIQUE INDEX UX_purchase_order_allocations_line ON dbo.purchase_order_allocations (purchase_order_line_id);`,
		`CREATE INDEX IX_purchase_order_allocations_sales_line ON dbo.purchase_order_allocations (sales_order_line_id);`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func down0024(db *gorm.DB) error { return db.Exec(`DROP TABLE dbo.purchase_order_allocations;`).Error }
