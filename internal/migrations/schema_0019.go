package migrations

import "gorm.io/gorm"

func up0019(db *gorm.DB) error {
	for _, statement := range []string{
		`IF EXISTS (SELECT 1 FROM dbo.sales_orders WHERE customer_po_number IS NULL OR LEN(LTRIM(RTRIM(customer_po_number))) = 0) THROW 51002, 'Existing sales orders require customer PO remediation before migration 19', 1;`,
		`ALTER TABLE dbo.sales_orders ALTER COLUMN customer_po_number NVARCHAR(100) NOT NULL;`,
		`ALTER TABLE dbo.sales_orders ADD CONSTRAINT CK_sales_orders_customer_po CHECK (LEN(LTRIM(RTRIM(customer_po_number))) > 0);`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func down0019(db *gorm.DB) error {
	for _, statement := range []string{
		`ALTER TABLE dbo.sales_orders DROP CONSTRAINT CK_sales_orders_customer_po;`,
		`ALTER TABLE dbo.sales_orders ALTER COLUMN customer_po_number NVARCHAR(100) NULL;`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
