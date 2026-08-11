package migrations

import "gorm.io/gorm"

func up0001(db *gorm.DB) error {
	statements := []string{
		`CREATE TABLE dbo.company_accounts (
            company_account_id BIGINT IDENTITY(1,1) NOT NULL CONSTRAINT PK_company_accounts PRIMARY KEY,
            company_name NVARCHAR(255) NOT NULL,
            contact_person NVARCHAR(255) NOT NULL,
            tin_number NVARCHAR(50) NOT NULL,
            billing_address NVARCHAR(255) NOT NULL,
            delivery_address NVARCHAR(255) NOT NULL,
            contact_number NVARCHAR(50) NOT NULL,
            created_at_utc DATETIME2(0) NOT NULL CONSTRAINT DF_company_accounts_created_at_utc DEFAULT (SYSUTCDATETIME()),
            updated_at_utc DATETIME2(0) NOT NULL CONSTRAINT DF_company_accounts_updated_at_utc DEFAULT (SYSUTCDATETIME())
        );`,
		`CREATE TABLE dbo.delivery_receivables (
            delivery_receivable_id BIGINT IDENTITY(1,1) NOT NULL CONSTRAINT PK_delivery_receivables PRIMARY KEY,
            company_account_id BIGINT NOT NULL,
            po_number VARCHAR(100) NOT NULL,
            delivery_date_utc DATETIME2(0) NOT NULL,
            payment_term_days TINYINT NOT NULL,
            due_date_utc DATETIME2(0) NOT NULL,
            amount_due_scaled BIGINT NOT NULL,
            payment_date_utc DATETIME2(0) NULL,
            lifecycle_status VARCHAR(16) NOT NULL CONSTRAINT DF_delivery_receivables_lifecycle_status DEFAULT ('Active'),
            created_at_utc DATETIME2(0) NOT NULL CONSTRAINT DF_delivery_receivables_created_at_utc DEFAULT (SYSUTCDATETIME()),
            updated_at_utc DATETIME2(0) NOT NULL CONSTRAINT DF_delivery_receivables_updated_at_utc DEFAULT (SYSUTCDATETIME()),
            row_version ROWVERSION NOT NULL,
            CONSTRAINT FK_delivery_receivables_company_accounts FOREIGN KEY (company_account_id) REFERENCES dbo.company_accounts(company_account_id),
            CONSTRAINT CK_delivery_receivables_payment_term_days CHECK (payment_term_days BETWEEN 1 AND 120),
            CONSTRAINT CK_delivery_receivables_amount_due_scaled CHECK (amount_due_scaled >= 0),
            CONSTRAINT CK_delivery_receivables_lifecycle_status CHECK (lifecycle_status IN ('Active', 'Cancelled', 'Archived')),
            CONSTRAINT CK_delivery_receivables_payment_date CHECK (payment_date_utc IS NULL OR payment_date_utc >= delivery_date_utc)
        );`,
		`CREATE TABLE dbo.audit_events (
            event_id BIGINT IDENTITY(1,1) NOT NULL CONSTRAINT PK_audit_events PRIMARY KEY,
            entity_type VARCHAR(64) NOT NULL,
            entity_id BIGINT NOT NULL,
            action VARCHAR(32) NOT NULL,
            actor_id NVARCHAR(255) NOT NULL,
            occurred_at_utc DATETIME2(0) NOT NULL CONSTRAINT DF_audit_events_occurred_at_utc DEFAULT (SYSUTCDATETIME()),
            request_id VARCHAR(64) NOT NULL,
            idempotency_key VARCHAR(128) NOT NULL,
            previous_values_json NVARCHAR(MAX) NULL,
            new_values_json NVARCHAR(MAX) NULL,
            note NVARCHAR(1000) NULL
        );`,
		`CREATE TABLE dbo.idempotency_keys (
            idempotency_key VARCHAR(128) NOT NULL CONSTRAINT PK_idempotency_keys PRIMARY KEY,
            command_type VARCHAR(64) NOT NULL,
            request_hash VARCHAR(64) NOT NULL,
            result_entity_id BIGINT NULL,
            response_status INT NOT NULL,
            created_at_utc DATETIME2(0) NOT NULL CONSTRAINT DF_idempotency_keys_created_at_utc DEFAULT (SYSUTCDATETIME()),
            expires_at_utc DATETIME2(0) NOT NULL
        );`,
		`CREATE INDEX IX_company_accounts_name ON dbo.company_accounts (company_name, company_account_id);`,
		`CREATE INDEX IX_company_accounts_tin ON dbo.company_accounts (tin_number, company_account_id);`,
		`CREATE INDEX IX_delivery_receivables_company_date ON dbo.delivery_receivables (company_account_id, delivery_date_utc, delivery_receivable_id);`,
		`CREATE INDEX IX_delivery_receivables_po ON dbo.delivery_receivables (po_number, company_account_id, delivery_receivable_id);`,
		`CREATE INDEX IX_delivery_receivables_active_due ON dbo.delivery_receivables (due_date_utc, company_account_id, delivery_receivable_id) INCLUDE (amount_due_scaled, po_number, delivery_date_utc, payment_term_days) WHERE lifecycle_status = 'Active' AND payment_date_utc IS NULL;`,
		`CREATE INDEX IX_delivery_receivables_paid_date ON dbo.delivery_receivables (payment_date_utc, company_account_id, delivery_receivable_id) WHERE payment_date_utc IS NOT NULL;`,
		`CREATE INDEX IX_audit_events_entity_time ON dbo.audit_events (entity_type, entity_id, occurred_at_utc DESC, event_id DESC);`,
		`CREATE INDEX IX_audit_events_request ON dbo.audit_events (request_id, occurred_at_utc);`,
		`CREATE INDEX IX_audit_events_idempotency ON dbo.audit_events (idempotency_key);`,
		`CREATE INDEX IX_idempotency_keys_expiry ON dbo.idempotency_keys (expires_at_utc);`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func down0001(db *gorm.DB) error {
	for _, statement := range []string{
		`DROP TABLE dbo.idempotency_keys;`,
		`DROP TABLE dbo.audit_events;`,
		`DROP TABLE dbo.delivery_receivables;`,
		`DROP TABLE dbo.company_accounts;`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
