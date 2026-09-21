# Sales Order Duplicate Process Plan

## Goal

Allow a user to create a new Sales Order by duplicating an existing Sales Order.
The new process copies the source customer and item snapshots. Before saving,
the user may edit the customer PO number and the quantity of each copied item.

The duplicate is a new independent Sales Order. It must not depend on, consume,
or create a reference to the source quotation.

## Source Rule

There is no quotation number available for this process. The only source is the
persisted Sales Order, so the source mapping is:

| New value | Source |
|---|---|
| Customer | Source Sales Order customer reference, prefilled and editable |
| Product, SKU, name, UOM | Source Sales Order line snapshot |
| Unit price and tax | Source Sales Order line snapshot, read-only |
| Quantity | Source line, editable |
| Customer PO | Source order, editable |
| Payment terms | Source order, prefilled and editable |
| Sales-order number | New sequence value |
| Quotation ID/number | Always null/empty |

If the source order originated from a quotation, its quotation number must not
be copied or displayed as the duplicate's source. The new order is a standalone
repeat order. `Repeat order` or `No quotation linked` is the correct source
label.

## Current-State Impact Analysis

The current implementation already provides most persistence primitives:

- `sales_orders.quotation_id` is nullable after migration `0018`.
- `sales_order_lines` stores product, quantity, UOM, price, tax, SKU, and name
  snapshots.
- `CreateStandalone` validates lines, validates the customer, recalculates
  totals, and creates an independent order.
- A database sequence and unique constraint generate Sales Order numbers.
- The shared form already supports PO entry and editable quantities.

The missing capability is an explicit source-order duplicate command and UI.
The minimum change does not need a schema migration, quotation lookup, or
quotation allocation change.

## Recommended User Flow

Add `Duplicate sales order` to the Sales Order detail page.

`OPEN` and `CONVERTED` source orders may be duplicated. A `CONVERTED` order is
still safe to use as a copy source because the operation reads its immutable
Sales Order snapshot and creates a separate standalone order; it does not
modify or reopen the converted order.

Routes:

```text
GET  /sales-orders/{id}/duplicate
POST /sales-orders/{id}/duplicate
```

The page should show:

- Source Sales Order number and customer.
- Searchable customer/company selector prefilled from the source order and
  editable, like standalone Sales Order creation.
- A new read-only Sales Order number.
- Customer PO prefilled from the source and editable.
- One row for every source item.
- Product, SKU, name, UOM, price, and tax as read-only values.
- Payment terms selector prefilled from the source order and editable.
- Quantity as an editable value for every row.
- Remove-row control or quantity-zero behavior.
- Recalculated line amounts and totals.
- `No quotation linked. This is a new repeat order.`
- `Changing quantities does not change the source sales order.`

Submit is disabled in the browser when the PO is invalid or no positive line
remains, but server-side validation is authoritative. Success redirects to the
new order detail page.

## Domain and Repository Plan

Add a dedicated operation instead of making the handler reconstruct a generic
standalone input from client values:

```text
Duplicate(ctx, sourceOrderID, customerPO, []DuplicateLineSelection) (SalesOrder, error)
```

Selections identify source `sales_order_line_id` values and submitted
quantities. The repository re-reads source lines from the database and ignores
client-submitted prices, taxes, product names, and UOMs. Customer/company and
payment terms are submitted as editable duplicate-order values and must be
validated independently of the source order.

Transaction steps:

1. Lock the source order and confirm it is eligible.
2. Load and lock its persisted lines.
3. Validate the submitted active customer/company.
4. Validate the submitted payment terms.
5. Validate and trim the customer PO.
6. Validate that every submitted line belongs to the source order.
7. Validate positive quantities and supported precision.
8. Require at least one positive line.
9. Copy source snapshots and apply only submitted quantities.
10. Recalculate line and order totals.
11. Generate a new Sales Order number inside the transaction.
12. Insert a new order with the submitted customer and terms and
    `quotation_id = NULL`.
13. Insert new line snapshots with no quotation-line references or allocation rows.
14. Commit atomically and return the new order.

The source order and lines must remain unchanged. Form resubmission must not
create unintended duplicate orders; use the existing project idempotency
convention or add a request token before production use.

## Data and Application Impact

| Area | Change |
|---|---|
| `internal/slices/salesorders/handler.go` | Add duplicate routes, page loading, POST parsing, eligibility, and errors |
| `internal/slices/salesorders/repository.go` | Add duplicate input types and atomic persistence |
| `templates/detail.html` | Add duplicate action for eligible orders |
| `templates/form.html` | Add duplicate mode with prefilled editable customer/terms and editable PO/quantities; keep line commercial values read-only |
| `internal/migrations` | No migration for the minimum feature |
| Sales-order tests | Add handler, repository, validation, concurrency, and PDF coverage |
| Quotation slice | No change; duplicate must never call it |
| Invoice/receivables | New duplicate follows existing standalone `OPEN` workflow; source status remains unchanged |

Do not add a quotation-number column. Do not copy invoice ID, invoice number,
status, quotation ID, or quotation allocations. A `duplicated_from_order_id`
audit field is not required for this feature.

## Validation and Safety

- Determine source type from the persisted source order, never a client flag.
- Validate the source order and line IDs again inside the repository transaction.
- Reject blank or overlong PO values, unknown lines, duplicate selections,
  negative quantities, invalid precision, and zero total quantity.
- Preserve copied prices, tax, product, and UOM values from the source database.
- Validate customer/company and payment terms from the submitted duplicate form;
  do not silently force the source values.
- Generate a different immutable order number; never reuse the source number.
- Do not create quotation allocations.
- Do not copy the source invoice relationship or converted status; the new
  order is created as `OPEN`.
- Apply the existing downstream edit-lock and status policy.

## Acceptance Criteria

- `OPEN` and `CONVERTED` orders with items show `Duplicate sales order`.
- Orders with another status or no items cannot be duplicated.
- The duplicate form preselects the source customer and allows changing it.
- The duplicate form preselects the source payment terms and allows changing
  them.
- Source PO is prefilled and can be changed.
- Each line quantity can be changed or removed.
- A changed PO is saved only on the new order.
- A changed customer and payment terms are saved only on the new order.
- Changed quantities recalculate line and order totals.
- The source order, lines, PO, totals, and quotation allocations are unchanged.
- The new order has a new number, null quotation ID, and no quotation allocations.
- A quotation-originated source still creates a standalone duplicate.
- Client attempts to change price, product, tax, or UOM are ignored/rejected.
- Invalid input returns the form with entered PO and quantities preserved.
- The duplicate PDF contains the new number, PO, quantities, and repeat-order
  source label.
- The duplicate can use the existing invoice flow as an `OPEN` order.

## Delivery Sequence

1. Confirm the documented `OPEN`/`CONVERTED` eligibility and no-audit-link decision.
2. Add duplicate types, repository transaction, and domain tests.
3. Add GET/POST routes and server-side validation.
4. Add duplicate form mode and detail-page action.
5. Add idempotency protection if the application does not already provide it.
6. Run `gofmt`, `go test ./...`, `go vet ./...`, and live SQL/browser checks.
7. Update the implementation ladder after implementation and verification.

## Confirmed Decisions

- `OPEN` and `CONVERTED` orders may be duplicated.
- Payment terms are prefilled from the source order and can be changed.
- Prices and tax values are copied read-only; only customer PO and quantities
  are editable, along with customer/company and payment terms.
- No `Duplicated from SO-...` audit link or persistence field is required.
- Quantity zero removes the line and is normalized to omission server-side.
