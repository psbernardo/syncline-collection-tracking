# Duplicate Invoice Number Validation Plan

## Goal

Prevent a receivable from being saved when its invoice number is already in use. The rule must apply to both create and edit commands, show a field-level validation error, and remain safe when two requests are submitted concurrently.

## Current State

- `NewDeliveryReceivable` validates that `invoice_number` is required, trimmed, and 1-100 ASCII alphanumeric characters in `internal/slices/receivables/domain.go`.
- `Create` and `Update` in `internal/slices/receivables/commands.go` already perform the PO duplicate check inside the transaction.
- `Repository` and `GormRepository` expose `FindBlockingPONumber`, but no invoice equivalent exists.
- `handler.go` maps `ErrDuplicatePO` to the PO field and HTMX feedback response. Invoice errors currently only come from syntax validation.
- Migration `0003` demonstrates the database uniqueness pattern for PO numbers.
- Migration `0004` added the invoice column and backfilled existing rows with `0000`; adding a unique index requires checking those backfilled values first.

## Proposed Rule

Use the same lifecycle scope as PO uniqueness:

- An invoice number must be unique across all `Active` and `Archived` receivables.
- A `Cancelled` receivable does not reserve its invoice number, so the number may be reused after cancellation.
- On edit, the current receivable is excluded from the duplicate lookup.
- Comparison uses the already trimmed invoice value. SQL Server's configured collation should define case behavior consistently for both the lookup and unique index; if the business requires case-insensitive behavior regardless of database collation, add an explicit normalized comparison column before implementation.

## Implementation Steps

### 1. Add domain error

In `internal/slices/receivables/domain.go`, add `ErrDuplicateInvoiceNumber` with a user-facing message equivalent to:

`Invoice number is already used by a non-cancelled receivable.`

No change is needed to the existing format validation. The duplicate rule belongs to the command/repository layer because it depends on persisted records.

### 2. Extend the repository contract

In `internal/slices/receivables/repository.go` and `repository_gorm.go`, add:

`FindBlockingInvoiceNumber(ctx, db, invoiceNumber string, excludeID int64) (bool, error)`

The query should:

- Match `invoice_number` exactly after the domain has trimmed input.
- Restrict matches to `lifecycle_status IN ('Active', 'Archived')`.
- Exclude `delivery_receivable_id = excludeID` when editing.
- Use a bound parameter and return a wrapped repository error on query failure.

### 3. Enforce the check in both commands

In `Create`, check the invoice immediately before the existing PO check, inside the same transaction and after company validation. Return `ErrDuplicateInvoiceNumber` when a blocker exists.

In `Update`, perform the same check after loading/protecting the existing receivable and pass `command.ID` as the exclusion ID.

Keep the database error mapping as a backstop. Add an `isDuplicateInvoiceNumberError` helper and map the unique-index violation to `ErrDuplicateInvoiceNumber`, just as PO violations are mapped today.

### 4. Map errors in the handler

Update both create and update error branches in `internal/slices/receivables/handler.go` to recognize `ErrDuplicateInvoiceNumber`.

- Set `ValidationErrors["InvoiceNumber"]` to the duplicate message.
- For HTMX, return the existing `422 Unprocessable Entity` form fragment and set `X-Feedback-Message` to the invoice duplicate message.
- Preserve submitted values, row version, and idempotency key exactly as the current validation path does.
- Keep PO and invoice duplicate handling separate so the correct field and feedback message are displayed.

### 5. Add a database uniqueness migration

Add the next versioned migration, rather than changing migration `0004` after it has been applied. Before creating the index, fail with a clear migration error if duplicate blocking invoice values already exist, especially multiple legacy rows containing `0000`.

Then create a filtered unique index such as:

`UX_delivery_receivables_invoice_not_cancelled ON dbo.delivery_receivables (invoice_number) WHERE lifecycle_status IN ('Active', 'Archived')`

Add a matching down migration. Do not silently delete, merge, or invent invoice numbers for existing rows. Existing duplicate data must be resolved as a data/business decision before the index is applied.

## Tests

### Domain/service tests

- Create rejects a duplicate invoice number.
- Update rejects an invoice used by another active or archived receivable.
- Update allows the same invoice number when the only match is the current receivable.
- Cancelled records do not block create or update.
- Invoice syntax validation still takes precedence over duplicate lookup.
- A database unique-index violation is translated to `ErrDuplicateInvoiceNumber`.

### Repository tests

Using the existing SQL Server dry-run/sqlmock style, verify the duplicate query contains:

- `invoice_number` equality predicate.
- Active/Archived lifecycle predicate.
- Exclusion predicate when `excludeID > 0`.
- Bound variables rather than interpolated input.

### Handler tests

- Duplicate create returns HTTP 422 and renders the invoice field error.
- Duplicate HTMX create returns the form fragment and invoice feedback header.
- Duplicate edit returns HTTP 422 and preserves the edit form values/version.
- Duplicate PO behavior remains unchanged.

### Migration tests

- Migration registration includes the new version once.
- Up migration performs duplicate-data preflight before creating the filtered unique index.
- Down migration drops the invoice index.

## Verification

1. Run `go test ./...`.
2. Apply the migration against the development SQL Server only after inspecting any existing duplicate invoice values.
3. Manually verify create and edit with an existing active invoice, an archived invoice, a cancelled invoice, and the same invoice on the record being edited.
4. Confirm concurrent attempts cannot both persist the same active/archived invoice because the database index is authoritative.

## Open Business Decision

This plan assumes invoice uniqueness is global, matching the existing PO rule. If invoice numbers are only unique per company, change both the repository predicate and unique index to use `(company_account_id, invoice_number)` before implementation.
