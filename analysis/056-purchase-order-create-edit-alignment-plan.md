# 056 Purchase Order Create/Edit Alignment Plan

## Goal

Use one purchase-order form and one interaction model for creating and editing an open purchase order.

The only create-flow difference is the entry context:

- `/purchase-orders/new` starts a direct supplier purchase with no source rows.
- `/sales-orders/{id}/purchase-order/new` starts a sales-order allocation and loads that sales order's items automatically.

Editing must use the same form structure and functionality. The existing purchase-order mode and source-allocation rules remain enforced; the sales-order shortcut must not become a second UI workflow.

This plan is a focused follow-up to `analysis/054-purchase-order-multi-source-direct-purchase-plan.md`. The broader multi-source data-model work is a dependency, not repeated here.

## Current-State Analysis

### Create

- `GET /purchase-orders/new` calls `standaloneNewPage` and renders `templates/form.html` in `DIRECT` mode.
- The direct form renders every active product immediately, with quantity and unit-cost inputs.
- `POST /purchase-orders` parses `products[id].quantity` and `products[id].unit_cost`, then calls `Creator.Create`.
- `GET /sales-orders/{id}/purchase-order/new` calls `newPage` and renders the same template in sales-order mode.
- The sales-order form loads `order.Quotation.Lines`, initially selecting all source lines and using supplier mappings for supplier SKU and reference cost.
- Supplier selection posts to a separate preview route so supplier mappings and missing-product errors can be displayed before submission.
- `POST /sales-orders/{id}/purchase-order` parses source-line quantities and costs, validates supplier availability, and calls `CreateFromSalesOrder`.

### Edit

- `GET /purchase-orders/{id}/edit` calls `editPage` and renders a separate `templates/edit.html`.
- The edit page assumes every line has a `SalesOrderLineID`; product identity is read-only and there is no direct-product editor.
- The edit handler parses only `lines[sourceLineID]`, so direct purchase rows cannot be submitted.
- `Repository.Update` rejects inputs without `SalesOrderID`, making direct purchase orders impossible to edit even though their status is `OPEN`.
- The edit page does not use the create page's mode selector, direct product controls, or selected-total calculation.
- Edit error rendering reuses the original persisted order rather than rebuilding the submitted rows, so changed supplier, date, costs, quantities, and notes can be lost after validation errors.
- The edit page shows the current persisted total only; `purchaseOrderForm().refresh()` updates line count but not amounts or total.
- Create and edit have different headings, context fields, button behavior, item-row markup, and add/remove semantics.

### Source and persistence constraints

- `purchase_order_lines` now supports nullable source fields under migration `0025`; direct lines have no sales-order references and allocation lines have both `sales_order_id` and `sales_order_line_id`.
- `FindByID` derives the mode from line sources, but the list page still displays only the legacy header `SalesOrderID`/`SalesOrderNumber`.
- The repository currently has separate create paths but a source-only update transaction.
- Sales-order allocation updates must continue to lock source lines, exclude the current PO's existing allocation when calculating availability, and replace allocations atomically.
- Direct updates must validate active products and active supplier-product mappings without creating or changing any sales-order allocation.

## Target Behavior

### One page and one client model

Replace `form.html` and `edit.html` with one shared purchase-order form template, or make one canonical partial responsible for the entire form and use thin page wrappers only when necessary.

The shared form must provide:

- The same order-context fields for create and edit: supplier, PO date, mode/source context, payment terms, expected delivery date, and notes.
- Read-only purchase-order number on edit and generated number on create.
- The same item table, row actions, empty-state behavior, selected-line count, live line totals, and live order total.
- The same ellipsis action menu for removing rows.
- `Add item` behavior appropriate to the active mode:
  - Direct mode: add an active catalog product row.
  - Sales-order mode: add an available source line, or restore a removed source row if the current single-source UX is retained.
- Disabled submit and an explicit message when no rows remain.
- Server validation errors that preserve all submitted scalar fields and rows.
- A consistent cancel link: purchase-order list for standalone create, source sales order for source create, and purchase-order detail for edit.

The JavaScript controller must be shared by create and edit. It should calculate line amount and total from the current inputs, update UOM and supplier values where applicable, track removed rows, prevent duplicate rows, and clean up viewport listeners when the component is destroyed.

### Create context

- Direct create starts with an empty grid and product options available for `Add item`.
- Sales-order create starts with the source sales order already selected and its eligible lines populated automatically.
- The sales-order source should be displayed as read-only context. The user must not be able to convert that form into a direct PO by changing a hidden or untrusted mode value.
- Supplier selection may continue to use the preview request, but preview and final submit must render the same canonical form and preserve all entered fields and rows.
- The final create request must derive the mode and source ownership from the route/context, not from a browser-only selector.

### Edit context

- `OPEN` direct and sales-order purchase orders open in the same canonical form.
- Existing PO lines are loaded as editable snapshots: quantity and supplier unit cost can be changed; product identity and source identity remain controlled by the persisted line/source relationship.
- Direct edits may remove existing rows and add active catalog products.
- Sales-order edits may remove or change allocation rows and add only eligible source lines. They may not add an unrelated direct product or change a source line to a different sales order.
- Supplier, PO date, payment terms, expected delivery, notes, and allowed costs are populated from the current order and preserved on validation failure.
- The edit form must submit a mode-aware line collection that supports both direct product IDs and source line IDs.
- Save must use the same row-level validation and live-total behavior as create, while the repository remains authoritative for status, source ownership, supplier mapping, and allocation availability.

## Domain and Repository Changes

1. Introduce a mode-aware input contract, extending or replacing `CreateInput` with explicit header fields and line fields for:
   - direct `ProductID`, or
   - `SalesOrderID` plus `SalesOrderLineID`,
   - quantity and unit cost.
2. Keep `CreateInput.Validate` as the shared invariant boundary: valid mode, one or more lines, positive quantities, non-negative costs, no duplicate direct products, no duplicate source lines, and no mixed line sources.
3. Add a shared repository update operation that branches by mode rather than requiring `SalesOrderID`:
   - direct update validates active products and supplier mappings and replaces direct lines atomically;
   - sales-order update locks the PO and source lines, recalculates availability excluding the PO's current allocations, then replaces lines and allocation rows atomically.
4. Ensure both create and update write line snapshots, recalculate subtotal/total, and record audit events. Cost changes must affect only the purchase-order snapshot.
5. Normalize repository errors so handlers can distinguish invalid supplier, missing supplier product, invalid line, stale allocation, and locked/non-open order cases.
6. Derive displayed mode/source information from line-level sources, consistent with migration `0025`; do not rely on the legacy PO header source column for new behavior.

## Handler and View-Model Changes

### `internal/slices/purchaseorders/handler.go`

- Replace separate `page` and `editPage` data paths with one form view model containing:
  - operation (`create` or `edit`), mode, order ID, number, source context;
  - submitted/current header values;
  - direct product options and source-line options;
  - canonical editable rows;
  - removed/excluded row IDs;
  - validation and missing-mapping details.
- Make create routes construct the same view model, differing only in initial mode and source rows.
- Make `editPage` load direct or source rows based on the persisted order mode.
- Make `standaloneCreate`, `create`, and `update` use one request parser and one validation/rendering path, with route context supplying source restrictions.
- Preserve submitted rows and values when rendering any error, including supplier preview errors.
- Reject mode/source tampering and reject updates for non-`OPEN` orders before attempting persistence.
- Keep the sales-order shortcut route and its automatic item loading; do not duplicate a second form implementation for it.

### Templates and static assets

- Consolidate `internal/slices/purchaseorders/templates/form.html` and `edit.html` into the canonical form.
- Update `list.html` and `detail.html` only as needed to expose the correct source/mode labels and edit link for both direct and source POs.
- Update `internal/web/static/app.js` so `purchaseOrderForm()` handles both modes and both operations with the same calculations and row actions.
- Reuse existing purchase-order CSS classes; add only the styles required for a consistent empty grid, add-item control, live totals, and validation state.

## Routes

Keep the public routes:

```text
GET  /purchase-orders/new
POST /purchase-orders
GET  /sales-orders/{id}/purchase-order/new
POST /sales-orders/{id}/purchase-order/supplier-preview
POST /sales-orders/{id}/purchase-order
GET  /purchase-orders/{id}/edit
POST /purchase-orders/{id}
```

The two create GET routes should call the same renderer. The sales-order route supplies the initial source and rows; the standalone route supplies direct-product options. The edit GET/POST routes should use the same renderer/parser and select the persisted mode.

## Tests and Acceptance Criteria

### Handler/template tests

- Direct create and direct edit render the same form structure and controls.
- Sales-order create renders the same form structure and automatically loads source items.
- Sales-order supplier preview preserves supplier, date, terms, notes, costs, quantities, and removed rows.
- Direct edit loads existing direct rows and active product options.
- Sales-order edit loads existing allocation rows and source context.
- Direct edit can remove a row, add a product, change quantity/cost, and save.
- Sales-order edit can remove/change an allocation and save without exposing arbitrary direct products.
- Removing all rows disables save and displays the required-line message.
- Validation errors preserve all submitted rows and header values.
- Non-open orders cannot be edited.
- Mode/source tampering is rejected.
- The same live line amount and total behavior is present on create and edit.

### Domain/repository tests

- Direct create and update succeed without a sales-order ID.
- Sales-order create and update preserve source ownership and allocation records.
- Direct update never writes allocation rows or changes sales-order readiness.
- Source update cannot over-allocate when another PO changes availability concurrently.
- Duplicate direct products, duplicate source lines, mixed modes, invalid mappings, and invalid quantities are rejected.
- Failed updates leave the original header, lines, totals, and allocations unchanged.
- Supplier cost changes do not mutate sales-order or quotation customer pricing.

## Delivery Sequence

1. Confirm the edit policy for source-linked POs and the allowed lock point for editing after confirmation/receiving.
2. Finish or verify the line-level source contract from plan `054`, including mode derivation and legacy header compatibility.
3. Define the shared form input/view-model and implement one request parser with error-preserving reconstruction.
4. Refactor the repository update transaction to support both direct and sales-order modes safely.
5. Consolidate the two templates into the canonical form and implement identical create/edit controls.
6. Update `purchaseOrderForm()` for shared totals, add/remove behavior, and mode-specific row creation.
7. Keep the sales-order shortcut as an initialization adapter that preloads source items and supplier-mapping data.
8. Add handler, template, domain, repository, concurrency, and regression tests.
9. Run `gofmt`, `go test -count=1 ./...`, `go vet ./...`, migrations, and browser checks for both entry paths and both edit modes.

## Decisions Required Before Coding

- Are supplier, PO date, payment terms, expected delivery, notes, quantity, and unit cost all editable for every `OPEN` PO, or should any field be restricted after confirmation?
- In sales-order edit, may the user add any currently unallocated source line, or only restore lines that were previously on this PO?
- Should changing supplier automatically reload reference costs and supplier SKUs, or preserve entered costs and require explicit confirmation?
- Should direct create initially show no rows with `Add item`, or retain the current behavior of rendering all active products as rows?
- When an existing legacy PO has header-level source data but incomplete line-level source data, should it be read-only until repaired?
