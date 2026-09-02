# Sales Order Invoice and Receivable Plan

## Goal

When a user creates an Invoice from an eligible Sales Order, create the Invoice and exactly one linked Delivery Receivable in the same transaction. Preserve manual Receivable creation as a separate supported workflow.

## Confirmed Business Rules

- Sales Orders have only two statuses: `OPEN` and `CONVERTED`.
- Only an `OPEN` Sales Order may create an Invoice.
- A successful conversion changes the Sales Order from `OPEN` to `CONVERTED`.
- A `CONVERTED` Sales Order cannot create another Invoice.
- Every Invoice created from a Sales Order must create exactly one Receivable.
- Manually created Receivables remain supported.
- A linked Quotation is not required for Invoice creation.
- Receivable payment terms come from `sales_orders.terms_days`.
- Receivable PO number comes from `sales_orders.customer_po_number`.
- Receivable tax rule follows the Sales Order tax treatment: a Sales Order with no tax rule uses `tax.RuleNone`; a Sales Order with the 12% VAT rule uses `tax.RuleVATInclusiveEWT1` for the Receivable.
- Receivable delivery date defaults to the current business date.

## Current-State Findings

- Invoice creation already exists in `internal/slices/invoices/repository.go`.
- Invoice and immutable Invoice line tables already exist in migration `0020`.
- Receivables already support nullable `invoice_id` through migration `0021`.
- Receivable creation already supports invoice references, tax calculation, idempotency, and audit events.
- The current Invoice transaction creates an Invoice and changes the Sales Order to `CONVERTED`, but does not create a Receivable.
- The current Invoice repository permits both `OPEN` and `COMPLETED` orders.
- The current schema and code still contain `COMPLETED` references even though the agreed lifecycle is `OPEN` and `CONVERTED`.
- The current Invoice and Receivable repositories each own their own transaction, so direct service-to-service calls would not provide atomicity.
- The existing Invoice creation form has no confirmation summary or confirmation modal.

## Field Mapping

| Receivable field | Source or rule |
|---|---|
| `company_account_id` | Sales Order `company_account_id` |
| `invoice_id` | Newly created Invoice ID |
| `invoice_number` | Newly created Invoice number snapshot |
| `po_number` | Sales Order `customer_po_number` |
| `gross_amount` | Invoice VAT-inclusive `total` |
| `tax_rule_code` | `tax.RuleNone` when the Sales Order has no tax rule; `tax.RuleVATInclusiveEWT1` when the Sales Order tax rule is 12% VAT |
| `delivery_date_utc` | Current business date |
| `payment_term_days` | Sales Order `terms_days` |
| `due_date_utc` | Delivery date plus payment terms |
| VAT, EWT, tax base, net amount | Receivable tax engine using gross amount and tax rule |

The Invoice remains the source document for the Invoice number, customer, Sales Order reference, and total. The Receivable stores an Invoice ID foreign key and a copied Invoice number for historical display.

## Transaction Design

Add an application-level conversion operation, for example:

```text
CreateInvoiceAndReceivableFromSalesOrder(
    ctx,
    salesOrderID,
    invoiceNumber,
    invoiceDate,
    idempotencyKey,
    requestID,
    actorID,
) (Invoice, DeliveryReceivable, error)
```

One transaction must:

1. Lock the Sales Order row.
2. Verify that the Sales Order exists and has status `OPEN`.
3. Verify that it has at least one persisted line.
4. Validate and normalize the Invoice number.
5. Resolve idempotency and return the existing result for an identical retry.
6. Check Invoice number uniqueness.
7. Create the immutable Invoice header and exact Sales Order line copies.
8. Build a Receivable from the newly created Invoice and Sales Order values.
9. Resolve the Receivable tax rule from the Sales Order: no tax maps to `tax.RuleNone`; 12% VAT maps to `tax.RuleVATInclusiveEWT1`.
10. Insert the Receivable with the new `invoice_id`.
11. Update the Sales Order from `OPEN` to `CONVERTED`.
12. Write audit events for the Invoice and Receivable.
13. Persist the idempotency result.
14. Commit all changes atomically.

Any failure must roll back the Invoice, Invoice lines, Receivable, audit records, idempotency record, and Sales Order status update.

Do not call the existing Invoice repository method and Receivable service one after the other while each owns a separate transaction. Refactor persistence methods to accept the shared transaction, or introduce a conversion service that owns the transaction.

## Idempotency and Concurrency

- Repeating the same request with the same idempotency key and Invoice data returns the existing Invoice and Receivable.
- Reusing the key with different Invoice data returns an idempotency conflict.
- Two concurrent conversions for one Sales Order produce at most one Invoice and one Receivable.
- The database must enforce one Invoice per Sales Order and one active Receivable per Invoice.
- A unique Invoice number remains enforced at the database level.

## Status Normalization

Update all status logic to use only:

| Status | Meaning |
|---|---|
| `OPEN` | Invoice creation is allowed |
| `CONVERTED` | Invoice and Receivable have been created; conversion is closed |

Add a migration after the current migrations that:

- Converts existing `COMPLETED` Sales Orders to `CONVERTED`, if any exist.
- Replaces the Sales Order status check constraint with `OPEN` and `CONVERTED` only.
- Updates all code, templates, and tests that reference `COMPLETED`.

The migration must be reviewed against production data before execution. If a `COMPLETED` record can exist without an Invoice, the migration should report those rows before converting them.

## Database Changes

Add a migration after the current latest migration to:

- Add or verify an index on `delivery_receivables.invoice_id`.
- Add a filtered unique index on `delivery_receivables.invoice_id` for blocking lifecycle states.
- Keep `invoice_number` as the Receivable display snapshot and legacy/manual value.
- Preserve nullable `invoice_id` for manually created or unresolved legacy Receivables.
- Retain the existing Invoice-to-Sales Order and Invoice-number uniqueness constraints.

Manual Receivables without an Invoice must not be rejected by the new Invoice conversion constraint.

## Application Changes

### Invoice domain and repository

- Add a conversion result type containing both documents.
- Add a transaction-aware create operation.
- Restrict conversion eligibility to `OPEN`.
- Use the Sales Order's persisted `terms_days`; do not require a Quotation lookup.
- Continue copying immutable Sales Order and line snapshots into the Invoice.
- Return a specific error for a `CONVERTED` Sales Order.

### Receivable domain and repository

- Add a transaction-aware create method used by Invoice conversion.
- Require the supplied Invoice ID for this path.
- Copy the Invoice number from the Invoice record, never from browser input.
- Use the Invoice total as gross amount.
- Map a no-tax Sales Order to `tax.RuleNone`.
- Map a 12% VAT Sales Order to `tax.RuleVATInclusiveEWT1` (`VAT-inclusive, 1% EWT`) for Receivable calculation.
- Use the Sales Order's terms and current date.
- Preserve the existing manual create/update path and its user-selected fields.

### Cross-links

- Sales Order detail should show the Invoice link after conversion.
- Invoice detail should show the linked Receivable.
- Receivable detail should show the linked Invoice when `invoice_id` exists.
- Existing manual Receivables without an Invoice should remain viewable without broken links.

## Tests

### Domain and service tests

- `OPEN` Sales Order creates one Invoice and one Receivable.
- `CONVERTED` Sales Order is rejected.
- Sales Order status changes only after both documents can be persisted.
- Sales Order terms are copied to the Receivable.
- Sales Order customer PO is copied to the Receivable.
- Current date is used as delivery date.
- Gross, VAT, EWT, and net amounts match the configured tax calculation.
- A failed Receivable insert rolls back Invoice creation and status change.
- A failed status update rolls back both documents.
- Manual Receivable creation continues to work without an Invoice.

### Repository and concurrency tests

- Invoice and Receivable are committed in one transaction.
- Duplicate Invoice numbers are rejected.
- Duplicate Invoice-to-Receivable links are rejected.
- Concurrent requests create one Invoice and one Receivable.
- Idempotent retries return the original conversion result.
- Idempotency conflicts reject changed Invoice data.

### HTTP and template tests

- Invoice form displays the confirmation action.
- `COMPLETED` is not presented as a Sales Order status.
- `CONVERTED` Sales Orders have no active create-Invoice action.
- Invoice detail links to the Receivable.
- Receivable detail links to the Invoice when linked.

## Delivery Sequence

1. Inspect and report existing Sales Orders with status `COMPLETED`.
2. Add status normalization and the one-Invoice/one-Receivable database constraints.
3. Refactor transaction ownership around the combined conversion operation.
4. Implement Receivable creation from Invoice and Sales Order snapshots.
5. Restrict conversion to `OPEN` and transition to `CONVERTED`.
6. Add confirmation preview and modal behavior described in `059-invoice-receivable-confirmation-plan.md`.
7. Add cross-links and update status labels.
8. Add unit, repository, migration, rollback, concurrency, and handler tests.
9. Run `go test ./...`, `go vet ./...`, and formatting checks.

## Definition of Done

- Only `OPEN` Sales Orders can create Invoices.
- Every converted Sales Order has exactly one Invoice and one linked Receivable.
- Invoice, Receivable, and Sales Order status are created or changed atomically.
- Receivable field mappings follow the confirmed rules.
- Manual Receivables remain supported.
- Confirmation data is server-validated and cannot be manipulated through browser-submitted totals.
- No active `COMPLETED` status remains in the application lifecycle.
