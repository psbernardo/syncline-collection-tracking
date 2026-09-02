# 054 Purchase Order Multi-Source and Direct Purchase Plan

## Goal

Allow one supplier purchase order to contain procurement lines from one or more sales orders, or direct product purchases not tied to a sales order. A purchase order must use exactly one mode. Every sales-order allocation remains a separate purchase-order line.

Direct purchasing affects supplier purchasing only. Inventory records and inventory movements are out of scope for this phase.

## Confirmed Decisions

- Each allocation remains a separate purchase-order line.
- A PO cannot mix direct product lines and sales-order allocation lines.
- PO confirmation is independent of sales-order allocation completeness.
- Any existing sales-order status is eligible for procurement, provided the sales order exists.
- Direct products do not create inventory records at this stage.
- The sales-order relationship moves from the PO header to the PO line.
- Supplier unit cost may be changed after confirmation.
- PO cost edits must never change quotation or sales-order customer prices.

## Current-State Impact

The current implementation is single-source:

- `purchase_orders.sales_order_id` stores one header-level source.
- `purchase_order_lines.sales_order_line_id` is required.
- PO creation starts from `/sales-orders/{id}/purchase-order/new`.
- The form renders only lines from the selected sales order.
- Supplier-product lookup is driven only by source products.
- Allocation readiness and confirmation queries accept one sales-order ID.
- Purchase-order list/detail pages display one sales-order reference.
- The current `Add item` action restores removed source rows; it cannot add a catalog product.

This feature therefore requires a data-model and workflow change, not only a grid change.

## Purchase Order Modes

### Sales-order allocation mode

- One PO may contain lines from multiple sales orders.
- Each selected sales-order line creates one PO line.
- The same product from two sales orders remains two separate PO lines.
- Each line stores its `sales_order_id` and `sales_order_line_id`.
- Quantity cannot exceed the currently unallocated quantity for that sales-order line across non-cancelled POs.
- PO confirmation does not wait for all quantities of linked sales orders to be allocated.

### Direct purchase mode

- The user selects active products directly.
- Each product creates one PO line with no sales-order references.
- The product and supplier-product relationship must be active.
- The line is excluded from sales-order readiness calculations.
- No inventory or stock ledger action is performed.

### Mode invariant

A PO must contain either only sales-order allocation lines or only direct product lines. The invariant must be validated in the domain and repository, not only in the browser.

## Data Model

### Purchase-order header

Stop using `purchase_orders.sales_order_id` as the source of truth for new records.

Recommended compatibility approach:

- Keep the existing column nullable temporarily.
- Backfill line-level source IDs for existing records.
- Stop writing the header source for new POs.
- Update reads to derive source information from lines.
- Remove the legacy column in a later cleanup migration after all consumers are migrated.

### Purchase-order lines

Add `sales_order_id` and make both source fields nullable:

```text
purchase_order_line_id
purchase_order_id
sales_order_id nullable
sales_order_line_id nullable
product_id
supplier_product_id
sku
name
supplier_sku
uom
quantity_scaled
unit_cost_scaled
line_total_scaled
```

Rules:

- Direct line: both sales-order fields are null.
- Allocation line: both fields are populated.
- The referenced sales-order line must belong to the referenced sales order.
- Quantity is positive; unit cost and total are non-negative.

### Existing allocation table

The current `purchase_order_allocations` table duplicates the one-to-one source relationship now required by each PO line. Retain it temporarily to avoid breaking readiness queries and future receiving work, but treat the line-level source fields as the authoritative display relationship.

Because every source allocation is a separate PO line, the current unique allocation-per-PO-line rule can remain. If later receiving requires multiple source allocations on one aggregated PO line, remove that unique index in a separate migration.

## Migration Plan

Add a migration after the current purchase-order migrations:

1. Add nullable `sales_order_id` to `purchase_order_lines`.
2. Backfill it from `purchase_orders.sales_order_id` for existing lines.
3. Validate existing line ownership during the migration.
4. Add the sales-order foreign key and source indexes.
5. Make `sales_order_line_id` nullable.
6. Add a check constraint requiring both source fields to be null or both populated.
7. Keep the PO header source column nullable for legacy compatibility.
8. Update down migration in dependency order.

Do not drop the header source column until legacy records and downstream queries have been verified.

## Domain and Repository Design

Replace the source-specific input with:

```text
PurchaseOrderInput
- number
- supplier_id
- PO date
- payment terms
- expected delivery
- notes
- mode
- lines[]

PurchaseOrderLineInput
- product_id
- sales_order_id optional
- sales_order_line_id optional
- quantity
- unit_cost optional
```

Validate:

- Valid mode and at least one line.
- No mixed modes.
- No duplicate source sales-order lines.
- No duplicate direct products unless explicitly approved.
- Any existing sales-order status is accepted.
- Every source line belongs to its posted sales order.
- Allocation quantity does not exceed remaining quantity.
- Direct products and supplier-product mappings are active.
- Product/UOM snapshots are valid.

Creation transaction:

1. Validate the supplier is active.
2. Lock referenced sales-order lines in deterministic ID order.
3. Recalculate active allocations inside the transaction.
4. Validate source-line availability.
5. Validate direct products and supplier mappings when in direct mode.
6. Create the PO header without a source sales-order ID.
7. Create one PO line per input line.
8. Store line-level source IDs only for allocation lines.
9. Preserve the supplier unit-cost snapshot and calculate totals.
10. Write an audit event and commit atomically.

Update transaction:

- Preserve source IDs for allocation lines.
- Revalidate allocation ownership and available quantity.
- Allow supplier metadata and supplier unit-cost changes while editing is permitted.
- Do not mutate quotation or sales-order customer prices.
- Record old and new costs in the audit history.

## Routes and Entry Points

Add:

```text
GET  /purchase-orders/new
POST /purchase-orders
```

Keep:

```text
GET /sales-orders/{id}/purchase-order/new
```

The existing route should open the general form with the sales order preselected.

Keep lifecycle routes:

```text
POST /purchase-orders/{id}/confirm
POST /purchase-orders/{id}/cancel
```

Confirmation validates the PO itself and does not require linked sales orders to be fully allocated.

## Form and Grid UX

### Mode selection

Provide mutually exclusive modes:

- `From sales orders`
- `Direct supplier purchase`

Changing modes must warn before clearing existing rows.

### Sales-order mode

- Search/select one or more sales orders.
- Do not filter by sales-order status.
- Display customer and sales-order number.
- Load available lines for each selected order.
- Show ordered, allocated, and remaining quantities.
- Add one source allocation per PO row.
- Show the source sales order on every row.
- Allow partial quantities within the remaining amount.

### Direct mode

- Search/select active products.
- Include products absent from all sales orders.
- Load supplier SKU and reference cost.
- Allow quantity and unit-cost entry.
- Mark rows as `Direct purchase`.

### Add/remove behavior

- Keep the final-column ellipsis action menu.
- `Remove` removes only the selected PO line.
- `Add item` adds an available source line in sales-order mode or a new product row in direct mode.
- Prevent duplicate source lines.
- Prevent invalid blank rows from submission.
- Disable submit when all rows are removed.
- Preserve both row types and entered values after validation errors.

## Queries and Display

Update list/detail queries to:

- Show all source sales-order numbers associated with a PO.
- Show the source sales order on each allocation line.
- Show `Direct purchase` for direct POs.
- Show the PO mode.
- Display supplier cost and line totals as PO snapshots.

Sales-order readiness remains:

```text
ordered quantity
- non-cancelled PO allocations
= unallocated quantity
```

Direct lines must never contribute to this calculation.

## Lifecycle and Cost Changes

PO confirmation is independent:

- A direct PO can be confirmed without a sales order.
- A multi-sales-order PO can be confirmed while source orders remain partially unallocated.
- Cancelled POs release their source allocations.

Supplier cost edits:

- Are allowed after confirmation until a later receiving lock is introduced.
- Recalculate the PO line and header totals.
- Never update quotation or sales-order customer prices.
- Record the previous and new cost in `audit_events`.

The receiving plan must define the status at which cost and line edits become immutable.

## Testing and Acceptance Criteria

### Domain and repository tests

- One PO can contain lines from two sales orders.
- The same product from two sales orders creates two PO lines.
- A direct product not present in a sales order can be purchased.
- Mixed direct and sales-order lines are rejected.
- Any valid sales-order status is accepted.
- Over-allocation is rejected across active POs.
- Concurrent allocation cannot exceed source quantity.
- Direct lines do not change sales-order readiness.
- PO confirmation succeeds while source orders remain partially unallocated.
- Cancelled POs release allocations.
- Supplier cost changes affect only PO snapshots.
- Existing single-source POs remain readable after migration.

### Handler and template tests

- Standalone create renders both modes.
- Sales-order entry preloads its source order.
- Multiple sales orders can be selected.
- Product options include products outside selected sales orders.
- Mixed-mode submissions are rejected.
- Ellipsis actions remove the correct row.
- Add item creates the correct row type.
- Removing all rows disables submission.
- Validation preserves direct and source-linked rows.
- List/detail pages show all sources or `Direct purchase`.

## Delivery Sequence

1. Confirm the receiving lock point for post-confirmation edits.
2. Add the line-level source migration.
3. Add mode-aware domain inputs and validation.
4. Refactor repository create/update transactions.
5. Add active product, sales-order, and source-line option providers.
6. Add standalone PO routes and preserve the sales-order shortcut.
7. Replace the source-only form with the two-mode form and grid.
8. Update list/detail pages and readiness queries.
9. Add cost-change audit behavior.
10. Add domain, repository, handler, template, and acceptance tests.
11. Run migrations, `gofmt`, `go test ./...`, `go vet ./...`, and browser checks.

## Readiness

Ready for implementation when:

- The receiving lock point is agreed.
- Legacy header-source migration is approved.
- Duplicate direct-product line behavior is decided.
- Product and sales-order search sizes are known.

Done when:

- A PO supports lines from multiple sales orders.
- A PO supports direct products without a sales order.
- Direct and source modes cannot be mixed.
- Each source allocation is a separate PO line.
- Confirmation is independent of overall sales-order readiness.
- Cost edits do not mutate customer pricing.
- Legacy single-source POs remain accessible.
