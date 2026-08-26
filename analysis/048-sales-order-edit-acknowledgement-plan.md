# Sales Order Edit and Acknowledgement Plan

## Goal

Allow users to edit an `OPEN` sales order, including a sales order created from a quotation, while making the consequences explicit before the edit is saved. The user must acknowledge a warning before the system accepts changes.

The edit must preserve the quotation relationship and must never over-allocate quotation quantities when several sales orders share one quotation.

## Current-State Analysis

The current sales-order detail in the screenshot is quotation-originated:

- `sales_orders.quotation_id` is populated.
- The order status is `OPEN`.
- The detail template only shows `Edit sales order` when `quotation_id = 0`.
- `GET /sales-orders/{id}/edit` rejects quotation-originated orders.
- The repository implements `UpdateStandalone` but has no `UpdateFromQuotation` operation.
- Quotation conversion allocations are stored in `quotation_line_allocations`.
- There is no acknowledgement field or server-side acknowledgement check.

Therefore, adding only a visible button would be unsafe. The update operation must coordinate the sales-order lines and quotation allocations in one transaction.

## Edit Policy

### Eligibility

- Only `OPEN` sales orders may be edited.
- `COMPLETED` sales orders are read-only.
- A quotation-originated order can be edited while its source quotation is still valid and editable for allocation purposes.
- The source quotation ID and source quotation-line relationship cannot be changed.
- A standalone order may continue to use the existing broader edit rules.

If downstream procurement or invoicing has started, the sales order must be treated as locked even if its status is still `OPEN`. Add a future-ready lock check such as `is_edit_locked` or a downstream status check before enabling the action.

### What can change

For quotation-originated orders:

- Customer PO number, which remains required.
- Quantity of an existing quotation line.
- Removal of an existing line.
- Order-level terms only if the business permits them to differ from the quotation snapshot.

Do not allow:

- Adding a product not present on the source quotation.
- Changing the source quotation.
- Changing a product or quotation-line reference.
- Editing the quoted unit price or tax rule without an explicit commercial override policy.

For standalone repeat orders, retain the current ability to edit customer, PO, terms, products, quantities, prices, and tax values, subject to normal validation.

At least one line and a valid customer PO are required after every edit.

## Acknowledgement and Warning Flow

### Detail page

Show `Edit sales order` for every eligible `OPEN` order, including quotation-originated orders.

The button should identify the source context:

- `Edit sales order` for standalone orders.
- `Edit converted sales order` for quotation-originated orders.

Completed or downstream-locked orders show a disabled/read-only explanation instead of an edit action.

### Edit page warning card

At the top of `GET /sales-orders/{id}/edit`, display a warning card before the editable table:

> You are editing a live sales order. Saving changes will update this order and, for a quotation-based order, change the quantity allocated from the source quotation. Other sales orders from the same quotation are not changed.

For quotation-originated orders, also show:

- Source quotation number.
- Current allocation represented by this order.
- Other converted quantity.
- Remaining quantity after this order’s current allocation.
- A statement that the source quotation’s quoted price and product values are not being rewritten.

Require an unchecked checkbox:

```text
I understand that saving this edit changes the sales order allocation and may change the quantity still available from the quotation.
```

The `Save changes` button remains disabled in the browser until the checkbox is checked. The server must require a submitted acknowledgement value as well; browser-only enforcement is insufficient.

Do not use a hidden field as proof of acknowledgement. The checkbox must be present in the submitted form and validated immediately before mutation.

### Optional confirmation step

If users frequently edit orders after procurement begins, add a separate confirmation route:

- `GET /sales-orders/{id}/edit/confirm`
- `POST /sales-orders/{id}/edit/confirm`

The confirmation page summarizes the order and warning, then opens the edit form after acknowledgement. This requires a short-lived signed token or server-side session state. For the first implementation, the warning card and required checkbox on the edit form are sufficient and avoid introducing session state.

## Edit Page UX

Route: `GET /sales-orders/{id}/edit`

Reuse the create-order form but make source type visible.

### Header

- `Edit sales order SO-XXXXXXXX`.
- Status badge and customer name.
- Clickable source quotation reference when present.
- Required customer PO field populated with the current value.
- Back to detail link.

### Item table for quotation-originated orders

Show one row per current sales-order line:

- Product SKU and name, read-only.
- UOM, read-only.
- Original quoted quantity.
- Quantity already allocated to other sales orders.
- Current quantity in this order.
- Maximum quantity available to this order.
- Editable quantity for this order.
- Unit price and tax rule, read-only unless commercial override is approved.
- Recalculated line amount.
- `Remove` action per line.

Removing a line sets its submitted quantity to zero or removes it from the form, but the server must interpret both forms as releasing that allocation.

### Item table for standalone orders

Keep the current add-product and remove-item behavior. The same required acknowledgement card and required PO field apply.

### Actions

- Primary: `Save changes`.
- Secondary: `Cancel` returning to detail without mutation.
- Disable save until PO is valid, at least one line remains, and acknowledgement is checked.
- Show an unsaved-changes warning when leaving after edits.
- On success, redirect to detail with a confirmation message: `Sales order updated.`
- On concurrency conflict, reload current allocations and show which quantities changed.

## Domain and Repository Contract

Add a shared edit acknowledgement validator:

```text
ValidateEditAcknowledgement(value string) error
```

Extend the sales-orders slice with an explicit quotation update operation:

```text
UpdateFromQuotation(ctx, orderID, customerPO string, selections []LineSelection, acknowledged bool) (SalesOrder, error)
```

The existing standalone update should also accept or otherwise validate the acknowledgement at the handler boundary and repository boundary.

### Quotation-originated update transaction

The transaction must:

1. Lock the sales-order header and confirm it is `OPEN`.
2. Lock the order’s current allocation rows.
3. Lock all source quotation lines with `UPDLOCK, HOLDLOCK`.
4. Verify the source quotation is still eligible and has not been cancelled or expired.
5. Calculate each source line’s available quantity as:
   - Quoted quantity
   - Minus allocations from other sales orders
   - Plus this order’s current allocation
6. Validate the submitted quantity for every selected source line against that available quantity.
7. Require at least one positive selected quantity and a valid customer PO.
8. Update order header PO and totals.
9. Delete and recreate this order’s sales-order lines with snapshot values.
10. Delete this order’s old allocation rows.
11. Insert new allocation rows for the edited positive quantities.
12. Commit all changes atomically.

If any validation, lock, or insert fails, the original sales order and allocations must remain unchanged.

### Allocation invariants

- Total allocation for a quotation line can never exceed its quoted quantity.
- An order edit can release quantity for another future order.
- Editing one order cannot modify another order’s lines or PO.
- The quotation’s commercial line values remain unchanged.
- A quotation-based order line always has a source `quotation_line_id`.

## Persistence and Audit

The acknowledgement itself should be auditable when the application has an actor identity. Record an edit event containing:

- Sales-order ID.
- Source quotation ID, if present.
- Acknowledged timestamp UTC.
- User/actor ID when authentication is available.
- Old and new PO values.
- Old and new line quantities.
- Old and new totals.

If authentication is not yet available, persist the timestamp and leave actor identity nullable. Do not claim this is a security authorization mechanism; acknowledgement records awareness, while authorization remains a separate concern.

## Routes

- `GET /sales-orders/{id}`: show edit action according to eligibility.
- `GET /sales-orders/{id}/edit`: load standalone or quotation-based edit page.
- `POST /sales-orders/{id}`: update the correct source type after acknowledgement.
- `GET /sales-orders/{id}/pdf`: remain available from the detail page after edits.

The handler must choose the update operation from the persisted order source, not from a client-submitted source flag.

## Error Handling

- Missing acknowledgement: `Please acknowledge the editing warning before saving.`
- Missing/blank PO: `Customer PO number is required.`
- Completed order: `Completed sales orders cannot be edited.`
- Locked quotation order: `This sales order can no longer be edited because its source workflow has progressed.`
- Invalid source line: `The selected item is not part of the source quotation.`
- Excess quantity: `The requested quantity exceeds the quantity available from the quotation.`
- Concurrent change: `The quotation allocation changed while you were editing. Reload the page and review the quantities.`

Validation errors must return the form with the submitted values intact whenever safe.

## Tests and Acceptance Criteria

### Visibility and acknowledgement

- An `OPEN` standalone order shows the edit button.
- An `OPEN` quotation-originated order shows the edit button.
- A completed order does not show an edit button.
- The edit page displays the warning and unchecked acknowledgement checkbox.
- POST without acknowledgement is rejected and does not mutate data.
- POST with acknowledgement proceeds only after all other validations pass.

### Quotation allocation edits

- Edit a converted line from quantity 10 to quantity 6; allocation becomes 6.
- Remove one converted line; its allocation is released.
- Add quantity to one line only within the source quotation’s available quantity.
- A second sales order can consume quantity released by the first order.
- An edit cannot exceed quoted quantity after accounting for other orders.
- Editing one order does not change another order’s PO, lines, or totals.
- Failed or concurrent edits leave the original order and allocations intact.

### Standalone edits

- PO is required and trimmed.
- Products can be added and removed.
- The last line cannot be saved.
- Totals are recalculated from the edited snapshot.

### PDF regression

- The edited order PDF contains the new PO, quantities, totals, and only current order lines.
- The source quotation reference remains unchanged.
- PDF download remains available for open and completed orders.

## Delivery Sequence

1. Confirm that quotation-originated `OPEN` orders are editable under the acknowledgement policy.
2. Add edit warning/acknowledgement UI and server validation.
3. Add quotation-originated edit loading and detail-page visibility.
4. Implement the allocation-safe `UpdateFromQuotation` transaction.
5. Add edit audit persistence if actor/timestamp auditing is approved.
6. Preserve and improve standalone edit error handling.
7. Add route, domain, repository, concurrency, and template tests.
8. Verify edited sales-order PDFs and live SQL behavior.
9. Update the implementation ladder with the completed/verified status.

## Decisions Required Before Coding

- Is acknowledgement required only for quotation-originated orders, or for every sales-order edit?
- Should price/tax values be immutable on quotation-originated edits?
- What downstream state makes an `OPEN` order no longer editable?
- Is a timestamp-only acknowledgement sufficient before authentication exists?
- Should editing the customer PO be allowed after procurement has started?
