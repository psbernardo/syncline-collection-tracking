package migrations

import "gorm.io/gorm"

func up0025(db *gorm.DB) error {
	for _, statement := range []string{
		`ALTER TABLE dbo.purchase_order_lines ADD sales_order_id BIGINT NULL;`,
		`UPDATE pol SET sales_order_id = po.sales_order_id FROM dbo.purchase_order_lines pol JOIN dbo.purchase_orders po ON po.purchase_order_id = pol.purchase_order_id WHERE po.sales_order_id IS NOT NULL;`,
		`IF EXISTS (SELECT 1 FROM dbo.purchase_order_lines WHERE sales_order_line_id IS NOT NULL AND sales_order_id IS NULL) THROW 51000, 'Could not backfill purchase order line sales order IDs', 1;`,
		`IF EXISTS (SELECT 1 FROM dbo.purchase_order_lines pol JOIN dbo.sales_order_lines sol ON sol.sales_order_line_id = pol.sales_order_line_id WHERE pol.sales_order_id <> sol.sales_order_id) THROW 51000, 'Purchase order line source ownership is invalid', 1;`,
		`ALTER TABLE dbo.purchase_order_lines DROP CONSTRAINT FK_purchase_order_lines_sales_lines;`,
		`ALTER TABLE dbo.purchase_order_lines ALTER COLUMN sales_order_line_id BIGINT NULL;`,
		`ALTER TABLE dbo.purchase_order_lines ADD CONSTRAINT FK_purchase_order_lines_sales_orders FOREIGN KEY (sales_order_id) REFERENCES dbo.sales_orders(sales_order_id);`,
		`ALTER TABLE dbo.purchase_order_lines ADD CONSTRAINT FK_purchase_order_lines_sales_lines FOREIGN KEY (sales_order_line_id) REFERENCES dbo.sales_order_lines(sales_order_line_id);`,
		`ALTER TABLE dbo.purchase_order_lines ADD CONSTRAINT CK_purchase_order_lines_source_pair CHECK ((sales_order_id IS NULL AND sales_order_line_id IS NULL) OR (sales_order_id IS NOT NULL AND sales_order_line_id IS NOT NULL));`,
		`CREATE INDEX IX_purchase_order_lines_sales_order ON dbo.purchase_order_lines (sales_order_id, purchase_order_line_id);`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func down0025(db *gorm.DB) error {
	for _, statement := range []string{
		`DROP INDEX IX_purchase_order_lines_sales_order ON dbo.purchase_order_lines;`,
		`ALTER TABLE dbo.purchase_order_lines DROP CONSTRAINT CK_purchase_order_lines_source_pair;`,
		`ALTER TABLE dbo.purchase_order_lines DROP CONSTRAINT FK_purchase_order_lines_sales_lines;`,
		`ALTER TABLE dbo.purchase_order_lines DROP CONSTRAINT FK_purchase_order_lines_sales_orders;`,
		`ALTER TABLE dbo.purchase_order_lines ALTER COLUMN sales_order_line_id BIGINT NOT NULL;`,
		`ALTER TABLE dbo.purchase_order_lines ADD CONSTRAINT FK_purchase_order_lines_sales_lines FOREIGN KEY (sales_order_line_id) REFERENCES dbo.sales_order_lines(sales_order_line_id);`,
		`ALTER TABLE dbo.purchase_order_lines DROP COLUMN sales_order_id;`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
