# Sales Order to Invoice Conversion Plan

## Goal

Allow a user to create exactly one immutable customer invoice from a Sales Order. The invoice receives all commercial and customer data from the Sales Order, keeps the Sales Order number as its reference, and requires the user to enter the invoice number manually. After a successful conversion, the Sales Order status becomes `CONVERTED`.

Invoice lines are a read-only copy of the Sales Order lines. There is no add-item or remove-item operation on the invoice.

## Current-State Findings

- `internal/slices/salesorders` currently supports Sales Order creation, editing while `OPEN`, listing, detail display, and PDF download.
- Sales Order statuses are currently only `OPEN` and `COMPLETED`; there is no `CONVERTED` status or conversion command.
- Sales Order header data is stored in `sales_orders`, while line snapshots are stored in `sales_order_lines`.
- There is no invoice table, invoice domain, invoice repository, invoice handler, or invoice template.
- Existing `delivery_receivables` records have a manually entered invoice number and amount, but they do not store invoice lines or a Sales Order foreign key. They are a collection workflow, not a complete invoice snapshot.
- The earlier receiving/invoice plan allowed line adjustments before posting. This request overrides that rule: invoice lines must be copied exactly and cannot be added or removed.
- The existing Sales Order schema and repository use `COMPLETED` as a status, but no receiving workflow currently transitions an order to it.

## Recommended Business Rules

### Conversion eligibility

1. A Sales Order may be converted only once.
2. Conversion should require the downstream-ready Sales Order state. Based on `analysis/035-receiving-invoice-plan.md`, that should be `COMPLETED` after full receiving. If invoice creation must be available before receiving is implemented, define a temporary explicit eligibility rule rather than treating every `OPEN` order as invoiceable.
3. `CONVERTED` is terminal for Sales Order editing. Existing edit and any future receiving actions must reject a converted order.
4. The Sales Order must contain at least one line and valid persisted totals before conversion.
5. The invoice must copy the Sales Order values; it must not recalculate against current product, tax, customer, or quotation configuration.

### Invoice number

- Invoice number is required input on the conversion form.
- Trim surrounding whitespace and enforce a documented maximum length.
- Invoice numbers must be unique among active/non-cancelled invoices. The existing receivable uniqueness rule should not be the only protection if invoices become a separate aggregate.
- The same invoice number may not be reused for a different Sales Order.

### Reference and copied data

The invoice should store both a foreign-key reference and immutable display snapshots where appropriate:

- Source Sales Order ID and Sales Order number.
- Customer account ID and customer name, billing address, delivery address, contact person, contact number, and email as available from the Sales Order snapshot.
- Customer PO number.
- Sales person.
- Payment terms and derived due date.
- Invoice date, using the approved business-date policy; default to the current business date if the user does not enter one.
- Subtotal, tax, VAT-inclusive total, and any withholding or other totals owned by the Sales Order.
- Every Sales Order line in the same order: product ID, SKU, name, UOM, quantity, unit price, tax code, tax rate, line total, and VAT-inclusive total.

Do not copy quotation allocation records into the invoice. The Sales Order is the sole source for this conversion.

## Data Model

Add a migration after migration `0019` (planned version `0020`) with tables similar to:

### `invoices`

- `invoice_id` primary key.
- `invoice_number` required and unique for active invoices.
- `sales_order_id` required foreign key with a unique constraint to enforce one invoice per Sales Order.
- `sales_order_number` required snapshot/reference.
- `company_account_id` required foreign key.
- Customer and address snapshots required for document stability.
- `customer_po_number`, `sales_person`, and payment-term fields.
- `invoice_date_utc`, `due_date_utc`, and creation/update timestamps.
- `subtotal_scaled`, `tax_scaled`, `total_scaled`, and any required withholding fields.
- Lifecycle status, initially `POSTED` or the agreed initial invoice status.

### `invoice_lines`

- `invoice_line_id` primary key.
- `invoice_id` required foreign key.
- `sales_order_line_id` required foreign key for traceability.
- Product ID and immutable SKU, name, and UOM snapshots.
- Quantity, unit price, tax code, tax rate, line total, and VAT-inclusive total.
- A unique constraint on `(invoice_id, sales_order_line_id)`.

The one-to-one Sales Order/invoice constraint is the database backstop for repeated submissions and concurrent conversion attempts. If cancelled invoices can be recreated, define that explicitly before relaxing the constraint; do not silently allow duplicate commercial invoices.

## Domain and Transaction Design

Add an invoice service/repository operation with a command containing:

```text
CreateFromSalesOrder(ctx, salesOrderID, invoiceNumber, invoiceDate, idempotencyKey) (Invoice, error)
```

The conversion transaction should:

1. Lock the Sales Order row with an update lock.
2. Verify that the order exists, is eligible, is not already `CONVERTED`, and has lines.
3. Load the persisted Sales Order header and line snapshots inside the same transaction.
4. Validate and normalize the manually supplied invoice number.
5. Check the invoice idempotency key and payload hash.
6. Check invoice-number uniqueness.
7. Insert the invoice header and exact line copies.
8. Update the Sales Order status from its eligible state to `CONVERTED`, using the locked row as the concurrency guard.
9. Persist an audit event containing the source Sales Order, invoice number, actor, and before/after values.
10. Commit all changes atomically.

A retry with the same idempotency key and identical input must return the existing invoice without creating another invoice or changing the Sales Order again. Reusing the key with a different invoice number must return an idempotency conflict.

If the existing collection workflow must receive an invoice-created receivable, create that handoff in the same transaction after the invoice model is finalized. It should reference the invoice and Sales Order rather than reconstructing an amount from form input. This is a separate integration step and must not weaken the invoice line-copy guarantees.

## Routes and Views

Extend `internal/slices/salesorders` or add a dedicated `internal/slices/invoices` slice. Prefer a dedicated invoice slice for ownership of invoice persistence and presentation, with the Sales Order slice owning only the conversion entry point.

Recommended routes:

- `GET /sales-orders/{id}/invoice/new`: show conversion form.
- `POST /sales-orders/{id}/invoice`: validate invoice number and create the invoice.
- `GET /invoices`: list invoices.
- `GET /invoices/{id}`: show invoice detail.
- `GET /invoices/{id}/pdf`: render the invoice snapshot.

Conversion page behavior:

- Show Sales Order number, status, customer, PO, terms, totals, and all lines as read-only.
- Show an invoice-number input as the only required commercial identifier entered by the user.
- Show invoice date only if the business requires a user-selectable date; otherwise default it server-side.
- Do not render add-line, remove-line, quantity, price, tax, or product controls.
- Explain that the invoice will contain an exact copy of the Sales Order.
- Disable or hide the action when the order is already `CONVERTED` or otherwise ineligible.
- On validation/concurrency failure, preserve the entered invoice number and show a specific error.
- On success, redirect to invoice detail with a confirmation message.

Sales Order detail behavior:

- Show `Create invoice` only for eligible non-converted orders.
- Show `Converted` status after success.
- Link to the invoice when one exists.
- Remove or disable Sales Order editing after conversion.

Invoice detail behavior:

- Display the invoice number prominently.
- Display `Reference: SO-...` with a link back to the Sales Order.
- Render customer, terms, dates, totals, and all copied lines as read-only.
- Do not provide item editing controls.

## Receivables Integration

The existing `delivery_receivables` workflow can be integrated after invoice creation:

- Add invoice ID and Sales Order ID references to the receivable if collection navigation requires drill-down.
- Map invoice number, Sales Order reference, PO number, customer account, due date, amount due, and tax amounts from the invoice snapshot.
- Preserve manual receivable creation as a separate path.
- Make invoice-to-receivable creation idempotent and part of the same transaction if posting must never leave an invoice without its collection record.
- Do not use the current receivable form to edit invoice lines; invoice immutability belongs to the invoice aggregate.

## Testing and Acceptance Criteria

### Domain/repository/service tests

- An eligible Sales Order creates one invoice with the supplied invoice number.
- Every Sales Order header field and every line field is copied exactly.
- Invoice values remain unchanged when products, tax rules, customer data, or the source quotation changes.
- The Sales Order status changes to `CONVERTED` only after invoice creation succeeds.
- A failed invoice insert leaves the Sales Order status unchanged.
- An already converted Sales Order cannot create another invoice.
- An ineligible Sales Order cannot create an invoice.
- Missing, blank, overlong, or duplicate invoice numbers are rejected.
- Concurrent conversion requests create at most one invoice.
- Retrying with the same idempotency key returns the original invoice.
- Retrying with the same key and different input returns an idempotency conflict.
- Invoice lines cannot be added, removed, or changed through the invoice API/repository.
- The invoice-to-receivable handoff, if enabled, is atomic and idempotent.

### Handler/template tests

- Eligible Sales Order detail shows the conversion action.
- Converted Sales Order detail shows `CONVERTED` and the invoice link, not an active conversion action.
- The conversion form displays all Sales Order lines and no add/remove controls.
- Invalid invoice-number input preserves the submitted value.
- Successful conversion redirects to invoice detail.
- Invoice detail displays the Sales Order reference and read-only copied lines.

### User acceptance scenarios

1. The user opens an eligible Sales Order and selects `Create invoice`.
2. The form shows the complete Sales Order and requires the user to enter an invoice number.
3. The user submits the form and sees an invoice containing exactly the same customer data, terms, totals, and items.
4. The Sales Order status becomes `CONVERTED` and links to the invoice.
5. Reopening the Sales Order does not allow a second conversion or Sales Order edits.
6. The user cannot add or remove invoice items.
7. Two simultaneous submissions result in one invoice and one converted Sales Order.

## Delivery Sequence

1. Confirm invoice terminology, invoice lifecycle, invoice date policy, and the Sales Order state that is eligible for conversion.
2. Complete or explicitly defer the receiving prerequisite; do not infer readiness from `OPEN` without approval.
3. Add migration `0020` for invoice headers, immutable invoice lines, constraints, and indexes.
4. Add invoice domain types, validation, repository, service, idempotency, and audit behavior.
5. Add the atomic Sales Order conversion operation and `CONVERTED` status transition.
6. Add conversion form, Sales Order action/status/link updates, invoice list/detail, and invoice PDF.
7. Add receivable handoff and cross-slice navigation if included in this release.
8. Run unit, repository, handler, migration, concurrency, and end-to-end acceptance tests.
9. Update `analysis/035-receiving-invoice-plan.md` and `analysis/000-implementation-ladder.md` to reflect the immutable-line rule and completed scope.

## Decisions Required Before Coding

- Is invoice creation allowed only after full receiving and `COMPLETED`, or may an `OPEN` Sales Order be invoiced?
- Is the invoice considered `POSTED` immediately, or does it need `DRAFT` and `POSTED` states?
- Is invoice date system-generated, user-entered, or both?
- Should invoice creation always create a delivery receivable in the same transaction?
- Should cancellation/voiding be supported, and can a voided invoice free the Sales Order for re-invoicing?
- Which customer/address fields are guaranteed to exist on the Sales Order snapshot?
- What maximum length and format rules apply to manually entered invoice numbers?
- Is a separate invoice aggregate required, or is the existing delivery receivable intended to become the invoice record?
