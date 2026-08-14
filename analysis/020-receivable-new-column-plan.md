# Receivable New Column Implementation Plan

## 1. Current State

The receivable list is a server-rendered HTML table in the `receivables` vertical slice. The current visible columns are:

- Company
- PO number
- Amount
- Delivery date
- Due date
- Payment term
- Status/classification
- Action

The data path is:

```text
SQL Server migration
-> receivableModel
-> DeliveryReceivable
-> ReceivableViewModel
-> list/detail templates
```

The repository uses explicit column projections and the migration registry. A schema change must therefore be implemented as a new versioned migration; `AutoMigrate` and edits to applied migrations are not allowed.

## 2. Approved Field Contract

The new field is approved as follows:

- Display label: `Invoice number`.
- Database column: `invoice_number`.
- Type: required `VARCHAR(100)`.
- Input: user-entered and required when creating a receivable.
- Validation: trimmed, non-empty, ASCII alphanumeric, maximum 100 characters, matching PO input validation.
- Visibility: list, detail, create form, and edit form, in the same operational flow as PO number.
- Edit behavior: editable through the existing active-receivable edit flow; protected lifecycle states remain protected.
- Existing data: migration `0004` backfills missing legacy values to `0000` and enforces `NOT NULL`.

Invoice-number uniqueness is not added because the approved requirement specifies the same validation rules as PO number, not the PO-specific duplicate constraint.

## 3. Recommended Scope Classification

### UI-only or existing-data column

Use this path when the new column is a relabeling, layout change, or presentation of an existing field such as payment date or lifecycle status:

1. Add the field to `ReceivableViewModel` only if it is not already exposed.
2. Add it to the relevant explicit repository projections.
3. Format dates, money, and statuses before template rendering.
4. Update `receivable-results.html`, `receivable-row.html`, and `detail.html` as applicable.
5. Add handler and template tests for full-page and HTMX responses.

### New persisted field

Use this path when the value is stored on `dbo.delivery_receivables`:

1. Record the approved field contract in `analysis/006-database-schema.md` and the delivery receivable mapping in `analysis/002-technical-plan.md`.
2. Add `schema_0004.go` with an explicit `up0004` and `down0004`, then register it in `internal/migrations/migrations.go`.
3. Use an explicit SQL Server type, nullability, default/backfill rule, constraint, and index only if an approved query requires one.
4. Update `receivableModel`, `DeliveryReceivable`, conversion functions, and all explicit `SELECT` projections.
5. Add the field to create/update commands and domain validation only if it is user-controlled.
6. Include it in form view models, form templates, field-error mapping, and normal/HTMX handler flows when editable or required.
7. Include it in list/detail view models and templates where the acceptance criteria require visibility.
8. Include previous and new values in audit JSON for every command that changes it.
9. Include it in normalized idempotency payloads so replay and request-conflict behavior remain correct.
10. Review dashboard, filter, export, and cross-slice projections for any query that should expose or constrain the field.

## 4. Ordered Implementation Steps

### A. Approve the field contract

Document the answers from Section 2, including at least one example value and behavior for existing records. Decide whether the field is:

- Persisted input.
- Persisted derived value.
- Query-only derived value.
- Existing data shown under a new table heading.

### B. Update domain and persistence contracts

For a persisted field:

- Add the field to the domain entity and persistence model with explicit mapping.
- Keep business rules in `domain.go` or the command boundary, not in handlers or templates.
- Update `toDomain` and `toModel`.
- Update `FindByID`, `List`, and `ListFiltered` projections.
- Update repository fakes and test fixtures.

For a derived field:

- Define one deterministic calculation using the supplied clock/date context.
- Do not persist it if it can change over time or be derived from authoritative fields.
- Reuse the same rule in list, detail, and dashboard paths.

### C. Add and verify the migration

For a new database column:

- Create the next migration version; never modify `schema_0001.go` or an applied migration.
- Apply `SET XACT_ABORT ON` through the existing migration runner transaction.
- Add explicit SQL Server type, nullability, default/backfill, and named constraints.
- Add a `down` operation suitable for development/recovery only.
- Register the migration with a descriptive name.
- Extend migration tests to verify the column metadata, constraints, default/backfill behavior, and rollback SQL expectations.
- Run migration status and apply verification against the development/test SQL Server database without destructive cleanup.

### D. Update the receivable UI

- Add the column to the table header and row partial in a consistent order.
- Add the value to detail view if it is part of the receivable record.
- Add labeled create/edit controls only when the field is user-editable.
- Preserve submitted values and show field-level validation errors.
- Keep normal HTML navigation and form submission working without JavaScript.
- Keep HTMX targets and stable partial roots unchanged.
- Ensure the column remains usable on narrow screens; use the existing table responsive treatment rather than adding client-side calculation.

### E. Update related behavior

- Review filters and URL state if the new column is searchable.
- Review dashboard aggregation and PO drill-down if the field affects totals or grouping.
- Review lifecycle restrictions if it can be changed after payment, cancellation, or archival.
- Review duplicate/unique rules before adding any index or constraint.
- Include the field in audit snapshots when it changes.

## 5. Test Plan

### Domain and command tests

- Valid value and boundary values.
- Blank, malformed, over-length, and out-of-range values.
- Create and update behavior.
- Protected lifecycle behavior.
- Due-date/classification behavior remains unchanged unless the approved field explicitly affects it.
- Idempotent replay returns the original result.
- Same idempotency key with a changed field is rejected as a payload conflict.
- Audit failure rolls back the field change.

### Repository and migration tests

- Create/update writes the field with the expected bound value.
- Detail and list projections return the field.
- Null/default/backfill behavior is correct.
- Constraint and database errors map to safe application errors.
- Migration registry and transaction behavior remain correct.

### HTTP and template tests

- Full list includes the new heading and value.
- HTMX results include the same column without duplicating the results target.
- Detail includes the value where required.
- Create/edit forms render the field, preserve input, and show errors when applicable.
- Normal requests and JavaScript-disabled flows remain usable.
- Empty, error, mobile, and protected-state behavior remain valid.

## 6. Acceptance Criteria

- The approved field contract is documented before code changes begin.
- The field is correctly classified as UI-only, derived, or persisted.
- A persisted field has a reviewed versioned migration and matching GORM/domain mappings.
- All explicit receivable projections return the required value.
- List, detail, create, edit, filter, dashboard, audit, and idempotency behavior are updated only where relevant.
- No financial calculation is duplicated in templates or Alpine.js.
- Existing receivable data is preserved and migration backfill is verified.
- `go test ./...`, `go vet ./...`, and `gofmt` pass.
- `analysis/028-progress.md` records the implementation and migration verification status.

## 7. Documentation Updates

This plan is the initial documentation update. After the field is approved, update only the applicable documents:

- `analysis/006-database-schema.md`: persisted column, constraints, indexes, and verification queries.
- `analysis/002-technical-plan.md`: delivery receivable domain mapping and data contract.
- `analysis/004-vertical-slice-coding-standard.md`: only if the new field establishes a reusable page or data rule.
- `analysis/028-progress.md`: plan status, implementation status, migration version, and verification result.
- `README.md`: only if migration or runtime instructions change.

## 8. Migration Behavior

Code implementation and migration handling are complete. Migration `0004` assigns `0000` to existing rows through a named SQL Server default constraint, then enforces the required column and non-empty check constraint.
