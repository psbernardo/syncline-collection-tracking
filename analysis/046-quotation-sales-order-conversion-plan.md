# Quotation to Sales Order Conversion Plan

## Goal

Allow an approved quotation to produce one or more independent sales orders, while also supporting sales orders created directly for repeat customers. The user can move selected quotation items, or all remaining items, into a new sales order. Quotation-originated orders keep a direct reference to their source quotation; standalone orders have no quotation reference and use explicitly selected customer and product data.

In this plan, **sales order** is the operational document created from the quotation. If the product later uses the label **sales invoice**, the same conversion and allocation rules apply, but invoice issuance should remain a separate accounting workflow.

## Current-State Gaps

The existing implementation is not sufficient for this workflow:

- `internal/slices/salesorders/handler.go` renders only a customer PO field; quotation lines are not shown or selectable.
- `CreateFromQuotation` copies every quotation line into one order.
- Migration `0015` has `UQ_sales_orders_quotation`, which prevents multiple orders from one quotation.
- Sales-order lines do not retain a quotation-line reference or source quantity.
- There is no persisted quantity allocation, so the application cannot safely prevent over-conversion.
- The quotation detail page always offers conversion, regardless of status or remaining quantity.
- The current quotation statuses use `ACCEPTED`; the requested business language is “approved.”
- The current sales-order schema requires a quotation, so it cannot represent a repeat order without one.

## Recommended Lifecycle

Do not use “not converted” as an automatic cancellation rule. If every quotation that has not yet been converted is immediately marked cancelled, approved quotations could never remain available for later customer action. Use quotation status for the commercial lifecycle and a separate conversion state for fulfillment progress.

### Quotation status

| Status | Meaning | Can create sales order? |
|---|---|---:|
| `DRAFT` | Still being prepared | No |
| `SENT` | Sent to customer and awaiting response | No |
| `APPROVED` | Customer-approved commercial offer | Yes |
| `PARTIALLY_CONVERTED` | Some, but not all, quantities converted | Yes |
| `FULLY_CONVERTED` | No remaining quantity is available | No |
| `EXPIRED` | Validity date has passed without completion | No |
| `REJECTED` | Customer declined the offer | No |
| `CANCELLED` | Explicitly cancelled by the user | No |

If retaining compatibility with existing `ACCEPTED`, migrate it to `APPROVED` or treat it as an alias during the migration. Do not display both labels.

### Conversion state

Calculate this from allocations rather than allowing users to edit it:

- `NOT_CONVERTED`: every quotation line has its full quantity remaining.
- `PARTIALLY_CONVERTED`: at least one quantity is allocated and at least one quantity remains.
- `FULLY_CONVERTED`: all quotation quantities are allocated.

An explicit **Cancel quotation** action may set `CANCELLED`, but a quotation should not be cancelled merely because it has not been converted. If the business specifically requires an unconverted quote to be tagged cancelled after expiry, use `EXPIRED` as the event and show a derived `Unconverted` tag rather than silently changing history.

## Sales Order Entry Paths

There are two intentionally separate entry paths:

| Entry path | Source | Quotation allocation | Customer/product selection |
|---|---|---|---|
| Convert quotation | Approved or partially converted quotation | Required | Loaded from quotation; quantities selected by user |
| Repeat order | No quotation | Not applicable | User selects active customer and products |

Both paths create the same independent sales-order data set and follow the same order status, pricing snapshot, validation, and confirmation rules. A standalone repeat order must never create a fake quotation simply to satisfy the schema.

## Page Experience

### Entry points

- Quotation detail: show `Create sales order` only when status is `APPROVED` or `PARTIALLY_CONVERTED` and at least one line has remaining quantity.
- Sales-order navigation: show `New sales order` for a standalone repeat order.
- Repeat-order page: provide a clear `Start without quotation` path rather than making the user search for a quotation.
- Quotation detail: show a conversion summary such as `2 orders · 6 of 10 units converted`.
- Quotation detail: provide links to each related sales order.
- Sales-order list/detail: display `From quotation QT-00000001` as a clickable reference.

### Create Sales Order page

Route: `GET /quotations/{id}/sales-order/new`

Use the application layout and existing table/card styling rather than the current inline HTML template.

1. **Context header**
   - Back link to the quotation.
   - Page title: `Create sales order`.
   - Source badge: quotation number and status.
   - Customer name, quotation date, validity date, payment terms, and total quoted amount.
   - A short explanation: `Choose the quantities to include. You can create another sales order later for the remaining quantities.`

2. **Line allocation table**
   - Product SKU and name.
   - UOM.
   - Quoted quantity.
   - Already converted quantity.
   - Remaining quantity.
   - Quantity for this order, initialized to zero.
   - Unit price and tax rule copied from the quotation and presented as read-only transaction values.
   - This order line total, updated as quantity changes.
   - Disable rows with zero remaining quantity and mark them `Fully converted`.

3. **Fast actions**
   - `Move all remaining` fills every editable quantity with its remaining quantity.
   - `Clear selection` resets all quantities to zero.
   - Optional per-row `Move remaining` control for touch-friendly partial workflows.
   - Show an inline selected-line count and selected total; no page reload is needed for these calculations.

4. **Order details panel**
   - Customer PO number, optional or required according to the approved PO policy.
   - Order notes, if the sales-order model supports them.
   - Preview subtotal, VAT, total, and selected quantity count.
   - Clearly label that these are copied values and that later edits to the quotation will not change this order.

5. **Actions**
   - Primary: `Create sales order`.
   - Secondary: `Cancel` / `Back to quotation`.
   - Disable submit until at least one quantity is greater than zero.
   - On validation error, preserve all entered quantities and return errors beside the affected row.
   - On success, redirect to the sales-order detail page, with a confirmation banner and source quotation link.

### Interaction behavior

- Use HTMX for server-rendered fragments where useful, but keep quantity totals as small client-side progressive enhancement so the page remains usable without JavaScript.
- Debounce or avoid network calls for every quantity keystroke.
- Never silently round quantities. Validate against the product/UOM precision already used by the domain.
- Warn before leaving the page when non-zero selections or a PO number have been entered.
- On a concurrency conflict, reload current allocations and show: `Some quantities were converted in another order. Review the remaining quantities.`

### New Repeat Order page

Route: `GET /sales-orders/new`

Use the same layout, table, totals, and confirmation language as the quotation conversion page, but replace quotation context with:

- Required searchable customer selector.
- Optional repeat-order reference or customer PO number.
- Empty line editor with searchable active-product selection.
- Quantity, UOM, unit price, tax rule, and line total for each selected product.
- Clear label: `No quotation linked. Prices and quantities are entered for this order.`

The user should be able to add lines quickly, remove lines before submission, and see totals update immediately. If the application has a standard supplier-price lookup, show it as an internal reference only and do not silently replace the customer price.

## Data Model

Keep quotation and sales-order data as separate transaction sets.

### Quotation allocation

Add a conversion allocation table, for example `quotation_line_allocations`, only for quotation-originated orders:

- `quotation_line_allocation_id` primary key.
- `quotation_id` foreign key for efficient source-level queries.
- `quotation_line_id` foreign key.
- `sales_order_id` foreign key.
- `sales_order_line_id` foreign key, if line creation is not guaranteed to be one-to-one.
- `quantity_scaled` allocated quantity.
- `created_at_utc`.

Constraints and indexes:

- Quantity must be greater than zero.
- Foreign keys must reference the source line and destination line.
- Index `(quotation_line_id, sales_order_id)`.
- Index `(sales_order_id, quotation_line_id)`.
- Enforce that total allocated quantity for a quotation line never exceeds its quoted quantity inside the conversion transaction. SQL Server locking or a serializable transaction is required; application-side checks alone are unsafe.

### Sales-order snapshot

`sales_orders` and `sales_order_lines` remain independent records. On quotation conversion, copy:

- Customer/account reference.
- Product reference, SKU, name, and UOM snapshot.
- Selected quantity, unit price, tax code, tax rate, line totals, and VAT-inclusive totals.
- Payment and delivery terms needed by the order.
- Customer PO data.
- Source quotation ID and quotation number snapshot.

For repeat orders, `quotation_id` and quotation number are null. Store the customer, product, price, tax, quantity, UOM, and terms entered for that order as its own snapshot.

Add to `sales_order_lines`:

- `quotation_line_id` for traceability.
- Any required source quotation number or product snapshots.

Change `sales_orders.quotation_id` from required to nullable, retaining the foreign key when a value is present. Remove the one-to-one `UQ_sales_orders_quotation` constraint from migration `0015`; a non-unique filtered index on `(quotation_id, sales_order_id)` is appropriate. Allowing null quotation IDs is what distinguishes repeat orders from quotation conversions.

The quotation must not be recalculated or mutated when an order is created. The order is a snapshot of the selected quantities and quoted commercial values at conversion time.

## Domain and Repository Design

Add two operations in the sales-orders slice, keeping the transaction boundary there:

```text
CreateFromQuotation(ctx, quotationID, customerPO, []LineSelection) (SalesOrder, error)
CreateStandalone(ctx, StandaloneOrderInput) (SalesOrder, error)
```

Recommended domain checks:

- Source quotation exists and is `APPROVED` or `PARTIALLY_CONVERTED`.
- Source quotation is not expired, rejected, cancelled, or fully converted.
- Every selected quotation line belongs to the source quotation.
- Every selected quantity is positive, valid for the UOM, and no greater than remaining quantity.
- At least one line is selected.
- Product and customer references remain valid.

The repository transaction should:

1. Lock the quotation lines and current allocation totals.
2. Recompute remaining quantities from the database.
3. Reject stale or over-allocated selections.
4. Create the sales-order header.
5. Create independent sales-order line snapshots.
6. Create allocation records.
7. Recalculate and persist the quotation conversion state.
8. Commit all changes atomically.

For `CreateStandalone`, validate the active customer and products, calculate totals from the submitted order lines, create the order and line snapshots, and commit without touching quotation tables or allocation records.

Make retries safe. A form resubmission must not create duplicate orders. Use an idempotency key stored with the conversion attempt, or require a unique client request token on the conversion record.

## Routes and Views

Extend `internal/slices/salesorders`:

- `GET /sales-orders/new`: standalone repeat-order form.
- `POST /sales-orders`: create a standalone repeat order.
- `GET /quotations/{id}/sales-order/new`: allocation form.
- `POST /quotations/{id}/sales-order`: validate selections and create order.
- `GET /sales-orders/{id}`: sales-order detail with source quotation reference.
- `GET /sales-orders`: list orders, filterable by quotation and status.
- `GET /sales-orders/{id}/pdf`: existing document endpoint, updated to render the order snapshot rather than the full quotation.

Update `internal/slices/quotations`:

- Return conversion summary and related orders from quotation detail/list queries.
- Gate conversion links by status and remaining quantity.
- Add explicit approve and cancel actions if those lifecycle transitions are not already implemented.

Use dedicated templates under `internal/slices/salesorders/templates/`; do not keep the current inline `template.Must` page.

## Testing and Acceptance Criteria

### Domain/repository tests

- Approved quotation can create an order from one selected line.
- Standalone repeat order can be created without a quotation.
- A standalone order has a null quotation reference and does not affect quotation conversion state.
- `Move all remaining` equivalent creates all remaining lines.
- A second order can use the remaining quantities from the same quotation.
- A third order cannot over-allocate a quotation line.
- Zero selections are rejected.
- Draft, sent, expired, rejected, cancelled, and fully converted quotations cannot create orders.
- Concurrent conversions cannot exceed the quoted quantity.
- Quotation and sales-order totals remain independent after creation.
- Retrying the same request does not create duplicate sales orders.
- Every sales-order line retains its quotation-line reference.

### Handler/template tests

- Form displays quoted, converted, and remaining quantities.
- `Move all remaining` and `Clear selection` controls are present.
- Submit is unavailable for a quotation with no remaining quantity.
- Invalid quantity errors preserve the submitted form.
- Successful conversion redirects to the order detail page.
- Quotation detail shows related orders and correct conversion status.
- Sales-order detail links back to its quotation.

### User acceptance scenarios

1. An approved quotation with five products opens with all quantities at zero and remaining quantities visible.
2. The user moves two products into an order, creates it, and sees the new order containing only those products.
3. The quotation now shows `Partially converted`; the user creates a second order from the remaining items.
4. The user can create multiple orders from the same quotation without duplicating already converted quantities.
5. Each order shows `From quotation QT-...`, and opening the link returns to the quotation conversion summary.
6. A quotation with no remaining quantity is visibly `Fully converted` and has no active create-order action.
7. A cancelled or expired quotation cannot be converted and explains why.
8. A repeat customer can create a sales order from the sales-order navigation without opening or creating a quotation.
9. A standalone order shows `No quotation linked` and remains fully usable for downstream processing.

## Delivery Sequence

1. Agree on terminology: `APPROVED` versus existing `ACCEPTED`, and whether the document is a sales order, sales invoice, or both.
2. Add migration for nullable quotation references, allocation records, sales-order line source references, and removal of the unique quotation constraint.
3. Add domain allocation calculations and repository transactions for both conversion and standalone creation.
4. Replace the inline create page with the quotation allocation form and progressive-enhancement controls.
5. Add the standalone repeat-order form with customer and product search.
6. Add order detail/list pages and quotation conversion summaries.
7. Add lifecycle actions and status transition tests.
8. Add acceptance tests for partial, full, repeated, standalone, and concurrent conversion.
9. Update the implementation ladder after the feature is implemented and verified.

## Decisions Required Before Coding

- Should the canonical status label be `APPROVED`, with `ACCEPTED` migrated as legacy data?
- Is the created document a sales order, a sales invoice, or should invoice issuance be a later step from the sales order?
- Can standalone repeat orders use the same `OPEN` status and downstream workflow as quotation-originated orders?
- Is customer PO required before creating an order?
- Are partial quantities allowed for every UOM, and what quantity precision applies?
- Should quotation validity/expiry block conversion after the customer has approved it?
- Should an explicit cancellation be allowed after partial conversion, and what happens to remaining quantities?
- Are order prices always locked to the quotation snapshot, or may an authorized user override them before confirmation?
