# Database Schema and Index Plan

This document defines the first SQL Server schema and index contract for migration `0001`. It is derived from `analysis/plan.md` and `analysis/technical-plan.md`.

## 1. SQL Script Standards

- Use the `dbo` schema explicitly for every table, constraint, and index.
- Use explicit, stable names such as `PK_company_accounts`, `FK_delivery_receivables_company_accounts`, and `IX_delivery_receivables_active_due`.
- Use `SET XACT_ABORT ON` in migration transaction code so runtime errors cannot leave an open transaction.
- Run each migration in one controlled transaction where SQL Server supports the operation.
- Do not use `GO` batch separators inside SQL sent through the Go database driver.
- Never use `SELECT *` in application queries or migration verification queries.
- Define column types, nullability, defaults, collations, and constraints explicitly. Do not rely on GORM defaults for the initial schema.
- Make migrations repeat-safe through the migration registry, not by silently ignoring failed or partial schema changes.
- Never edit an applied migration. Create a new version for every schema change.
- Use `BIGINT` for scaled money values and `DATETIME2` for UTC timestamps.
- Use foreign keys and check constraints to enforce data integrity in addition to application validation.
- Review every index against an actual query shape and execution plan. Do not create indexes only because a column exists.
- Do not create a unique index on PO number alone; the domain permits repeated PO values across companies or deliveries.

## 2. Table Contract

### `dbo.company_accounts`

| Column | Type | Null | Rule |
|---|---|---:|---|
| `company_account_id` | `BIGINT` | No | Identity primary key |
| `company_name` | `NVARCHAR(255)` | No | Required, trimmed |
| `contact_person` | `NVARCHAR(255)` | No | Required, trimmed |
| `tin_number` | `NVARCHAR(50)` | No | Required |
| `billing_address` | `NVARCHAR(255)` | No | Required |
| `delivery_address` | `NVARCHAR(255)` | No | Required |
| `contact_number` | `NVARCHAR(50)` | No | Required |
| `created_at_utc` | `DATETIME2(0)` | No | UTC |
| `updated_at_utc` | `DATETIME2(0)` | No | UTC |
| `row_version` | `ROWVERSION` | No | Optimistic concurrency for edits |

Company-account archiving is out of scope for the current 1-4 client MVP, so no account archive status is required in migration `0001`.

### `dbo.delivery_receivables`

| Column | Type | Null | Rule |
|---|---|---:|---|
| `delivery_receivable_id` | `BIGINT` | No | Identity primary key |
| `company_account_id` | `BIGINT` | No | Foreign key to `company_accounts` |
| `invoice_number` | `VARCHAR(100)` | No | Required, trimmed ASCII alphanumeric value; legacy rows default to `0000` during migration `0004` |
| `po_number` | `VARCHAR(100)` | No | ASCII alphanumeric, application and database validation |
| `po_number_normalized` | `VARCHAR(100)` | No | Uppercase comparison value |
| `delivery_date_utc` | `DATETIME2(0)` | No | UTC-normalized Philippines date |
| `payment_term_days` | `TINYINT` | No | Between 1 and 120 |
| `due_date_utc` | `DATETIME2(0)` | No | Calculated and persisted |
| `amount_due_scaled` | `BIGINT` | No | PHP multiplied by 10,000; non-negative |
| `payment_date_utc` | `DATETIME2(0)` | Yes | Null until fully paid |
| `lifecycle_status` | `VARCHAR(16)` | No | `Active`, `Cancelled`, or `Archived` |
| `created_at_utc` | `DATETIME2(0)` | No | UTC |
| `updated_at_utc` | `DATETIME2(0)` | No | UTC |
| `row_version` | `ROWVERSION` | No | Optimistic concurrency |

Recommended constraints:

- `CK_delivery_receivables_payment_term_days` enforces `payment_term_days BETWEEN 1 AND 120`.
- `CK_delivery_receivables_amount_due_scaled` enforces `amount_due_scaled >= 0`.
- `CK_delivery_receivables_lifecycle_status` restricts the lifecycle status values.
- `CK_delivery_receivables_payment_date` prevents a payment date earlier than the delivery date when both values are present.
- `CK_delivery_receivables_invoice_number` prevents an empty invoice number.
- `CK_delivery_receivables_po_number_normalized` prevents an empty normalized PO value.
- `FK_delivery_receivables_company_accounts` enforces the company relationship.

The due date is calculated in Go using the Philippines timezone, normalized to UTC, and persisted so historical terms cannot silently change the result.

### `dbo.audit_events`

| Column | Type | Null | Rule |
|---|---|---:|---|
| `event_id` | `BIGINT` | No | Identity primary key |
| `entity_type` | `VARCHAR(64)` | No | For example `delivery_receivable` |
| `entity_id` | `BIGINT` | No | Changed entity identifier |
| `action` | `VARCHAR(32)` | No | Create, update, payment, cancel, reopen, archive |
| `actor_id` | `NVARCHAR(255)` | No | Initial admin username |
| `occurred_at_utc` | `DATETIME2(0)` | No | UTC |
| `request_id` | `VARCHAR(64)` | No | Request correlation identifier |
| `idempotency_key` | `VARCHAR(128)` | No | Command key |
| `previous_values_json` | `NVARCHAR(MAX)` | Yes | Previous state |
| `new_values_json` | `NVARCHAR(MAX)` | Yes | New state |
| `note` | `NVARCHAR(1000)` | Yes | Optional user note |

Audit rows are append-only through the application. The business mutation and audit insert are one transaction.

### `dbo.idempotency_keys`

| Column | Type | Null | Rule |
|---|---|---:|---|
| `idempotency_key` | `VARCHAR(128)` | No | Unique command key |
| `command_type` | `VARCHAR(64)` | No | Command name |
| `request_hash` | `VARCHAR(64)` | No | Hash of normalized request |
| `result_entity_id` | `BIGINT` | Yes | Result resource |
| `response_status` | `INT` | No | Original response status |
| `created_at_utc` | `DATETIME2(0)` | No | UTC |
| `expires_at_utc` | `DATETIME2(0)` | No | Replay-window expiry |

## 3. Index Plan

### Company accounts

| Index | Keys | Purpose |
|---|---|---|
| `PK_company_accounts` | `company_account_id` clustered | Primary lookup and foreign-key target |
| `IX_company_accounts_name` | `company_name, company_account_id` | Prefix search and stable account ordering |
| `IX_company_accounts_tin` | `tin_number, company_account_id` | TIN lookup/filter |

Do not add a unique TIN index unless the business explicitly guarantees one account per TIN.

### Delivery receivables

| Index | Keys / filter | Purpose |
|---|---|---|
| `PK_delivery_receivables` | `delivery_receivable_id` clustered | Primary lookup |
| `IX_delivery_receivables_company_date` | `company_account_id, delivery_date_utc, delivery_receivable_id` | Client drill-down and delivery history |
| `IX_delivery_receivables_po` | `po_number, company_account_id, delivery_receivable_id` | PO search and client traceability |
| `UX_delivery_receivables_po_not_cancelled` | `po_number_normalized`; unique filtered to `lifecycle_status IN ('Active', 'Archived')` | Prevents non-cancelled PO duplicates |
| `IX_delivery_receivables_active_due` | `due_date_utc, company_account_id, delivery_receivable_id`; filtered to `lifecycle_status = 'Active' AND payment_date_utc IS NULL` | Near-due and overdue dashboard queries |
| `IX_delivery_receivables_paid_date` | `payment_date_utc, company_account_id, delivery_receivable_id`; filtered to `payment_date_utc IS NOT NULL` | Payment-received views |

Include only dashboard columns justified by the execution plan, such as `amount_due_scaled`, `po_number`, `delivery_date_utc`, and `payment_term_days`. Avoid adding every table column to an index.

### Audit events

| Index | Keys | Purpose |
|---|---|---|
| `PK_audit_events` | `event_id` clustered | Event identity |
| `IX_audit_events_entity_time` | `entity_type, entity_id, occurred_at_utc DESC, event_id DESC` | Entity history timeline |
| `IX_audit_events_request` | `request_id, occurred_at_utc` | Request tracing |
| `IX_audit_events_idempotency` | `idempotency_key` | Command/audit correlation |

### Idempotency keys

| Index | Keys | Purpose |
|---|---|---|
| `PK_idempotency_keys` | `idempotency_key` clustered or unique | Duplicate command protection |
| `IX_idempotency_keys_expiry` | `expires_at_utc` | Safe cleanup of expired keys |

## 4. Migration Verification Queries

The migration test should verify schema metadata rather than relying only on successful execution:

- Required tables exist in `dbo`.
- Required columns have the expected SQL types and nullability.
- `invoice_number` is required and existing receivables are backfilled to `0000` by migration `0004`.
- Foreign keys and check constraints exist.
- Required indexes exist with the intended keys and filters.
- `rowversion` is present on editable records.
- A duplicate idempotency key is rejected.
- Invalid payment terms and negative amounts are rejected.
- A receivable cannot reference a missing company account.

Run these checks against both development and test databases. Do not run destructive rollback verification against development data.

Before applying the non-cancelled PO unique index, run a duplicate preflight query and resolve every conflict manually. The migration must not guess which historical receivable should be cancelled or retained.

## 5. References

- GORM migration documentation: https://gorm.io/docs/migration.html
- GORM SQL Server connection documentation: https://gorm.io/docs/connecting_to_the_database.html#SQL-Server
- Microsoft Go SQL Server driver: https://github.com/microsoft/go-mssqldb
