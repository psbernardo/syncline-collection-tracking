package migrations

import "gorm.io/gorm"

func up0029(db *gorm.DB) error {
	for _, statement := range []string{
		`CREATE TABLE dbo.supplier_payments (
            supplier_payment_id BIGINT IDENTITY(1,1) NOT NULL CONSTRAINT PK_supplier_payments PRIMARY KEY,
            supplier_id BIGINT NOT NULL,
            check_number NVARCHAR(100) NOT NULL,
            issue_date_utc DATETIME2(0) NOT NULL,
            due_date_utc DATETIME2(0) NOT NULL,
            amount_scaled BIGINT NOT NULL,
            lifecycle_status VARCHAR(16) NOT NULL CONSTRAINT DF_supplier_payments_status DEFAULT ('Active'),
            created_at_utc DATETIME2(0) NOT NULL CONSTRAINT DF_supplier_payments_created_at DEFAULT (SYSUTCDATETIME()),
            updated_at_utc DATETIME2(0) NOT NULL CONSTRAINT DF_supplier_payments_updated_at DEFAULT (SYSUTCDATETIME()),
            row_version ROWVERSION NOT NULL,
            CONSTRAINT FK_supplier_payments_suppliers FOREIGN KEY (supplier_id) REFERENCES dbo.suppliers(supplier_id),
            CONSTRAINT CK_supplier_payments_check_number CHECK (LEN(LTRIM(RTRIM(check_number))) > 0),
            CONSTRAINT CK_supplier_payments_dates CHECK (due_date_utc >= issue_date_utc),
            CONSTRAINT CK_supplier_payments_amount CHECK (amount_scaled > 0),
            CONSTRAINT CK_supplier_payments_status CHECK (lifecycle_status IN ('Active', 'Voided'))
        );`,
		`CREATE UNIQUE INDEX UX_supplier_payments_supplier_check_active
            ON dbo.supplier_payments (supplier_id, check_number)
            WHERE lifecycle_status = 'Active';`,
		`CREATE INDEX IX_supplier_payments_due_active
            ON dbo.supplier_payments (due_date_utc, supplier_payment_id)
            INCLUDE (supplier_id, check_number, issue_date_utc, amount_scaled)
            WHERE lifecycle_status = 'Active';`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func down0029(db *gorm.DB) error {
	return db.Exec(`DROP TABLE dbo.supplier_payments;`).Error
}
