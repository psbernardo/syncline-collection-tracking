package migrations

import "gorm.io/gorm"

func up0007(db *gorm.DB) error {
	for _, statement := range []string{
		`CREATE TABLE dbo.products (
            product_id BIGINT IDENTITY(1,1) NOT NULL CONSTRAINT PK_products PRIMARY KEY,
            sku VARCHAR(50) NOT NULL,
            name NVARCHAR(255) NOT NULL,
            description NVARCHAR(MAX) NULL,
            uom VARCHAR(20) NOT NULL,
            is_active BIT NOT NULL CONSTRAINT DF_products_is_active DEFAULT (1),
            created_at_utc DATETIME2(0) NOT NULL CONSTRAINT DF_products_created_at_utc DEFAULT (SYSUTCDATETIME()),
            updated_at_utc DATETIME2(0) NOT NULL CONSTRAINT DF_products_updated_at_utc DEFAULT (SYSUTCDATETIME()),
            row_version ROWVERSION NOT NULL,
            CONSTRAINT CK_products_uom CHECK (uom IN ('PC', 'BOX', 'SET')),
            CONSTRAINT CK_products_sku CHECK (LEN(LTRIM(RTRIM(sku))) > 0)
        );`,
		`CREATE UNIQUE INDEX UX_products_sku ON dbo.products (sku);`,
		`CREATE INDEX IX_products_active_name ON dbo.products (is_active, name, product_id);`,
		`CREATE TABLE dbo.suppliers (
            supplier_id BIGINT IDENTITY(1,1) NOT NULL CONSTRAINT PK_suppliers PRIMARY KEY,
            name NVARCHAR(255) NOT NULL,
            contact_person NVARCHAR(255) NULL,
            contact_number NVARCHAR(50) NULL,
            email NVARCHAR(255) NULL,
            billing_address NVARCHAR(255) NULL,
            delivery_address NVARCHAR(255) NULL,
            tax_identifier NVARCHAR(100) NULL,
            is_active BIT NOT NULL CONSTRAINT DF_suppliers_is_active DEFAULT (1),
            created_at_utc DATETIME2(0) NOT NULL CONSTRAINT DF_suppliers_created_at_utc DEFAULT (SYSUTCDATETIME()),
            updated_at_utc DATETIME2(0) NOT NULL CONSTRAINT DF_suppliers_updated_at_utc DEFAULT (SYSUTCDATETIME()),
            row_version ROWVERSION NOT NULL,
            CONSTRAINT CK_suppliers_name CHECK (LEN(LTRIM(RTRIM(name))) > 0)
        );`,
		`CREATE INDEX IX_suppliers_active_name ON dbo.suppliers (is_active, name, supplier_id);`,
		`CREATE TABLE dbo.supplier_products (
            supplier_product_id BIGINT IDENTITY(1,1) NOT NULL CONSTRAINT PK_supplier_products PRIMARY KEY,
            supplier_id BIGINT NOT NULL,
            product_id BIGINT NOT NULL,
            supplier_sku VARCHAR(100) NULL,
            reference_cost_scaled BIGINT NOT NULL,
            is_active BIT NOT NULL CONSTRAINT DF_supplier_products_is_active DEFAULT (1),
            updated_at_utc DATETIME2(0) NOT NULL CONSTRAINT DF_supplier_products_updated_at_utc DEFAULT (SYSUTCDATETIME()),
            row_version ROWVERSION NOT NULL,
            CONSTRAINT FK_supplier_products_suppliers FOREIGN KEY (supplier_id) REFERENCES dbo.suppliers (supplier_id),
            CONSTRAINT FK_supplier_products_products FOREIGN KEY (product_id) REFERENCES dbo.products (product_id),
            CONSTRAINT CK_supplier_products_reference_cost CHECK (reference_cost_scaled >= 0)
        );`,
		`CREATE UNIQUE INDEX UX_supplier_products_supplier_product ON dbo.supplier_products (supplier_id, product_id);`,
		`CREATE INDEX IX_supplier_products_product_active ON dbo.supplier_products (product_id, is_active, supplier_id);`,
		`CREATE INDEX IX_supplier_products_supplier_active ON dbo.supplier_products (supplier_id, is_active, product_id);`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func down0007(db *gorm.DB) error {
	for _, statement := range []string{
		`DROP TABLE dbo.supplier_products;`,
		`DROP TABLE dbo.suppliers;`,
		`DROP TABLE dbo.products;`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
