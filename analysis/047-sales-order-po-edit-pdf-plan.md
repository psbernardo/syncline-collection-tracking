# Sales Order PO, Edit, and PDF Plan

## Goal

Make the customer PO number mandatory whenever a sales order is created or edited, provide a reliable edit workflow for eligible sales orders, and make the sales-order PDF available from the detail page in the same way as the quotation PDF.

This applies to both entry paths:

- A sales order converted from an approved quotation.
- A standalone repeat sales order created without a quotation.

## Current-State Analysis

The current worktree has partial support:

- `customer_po_number` is currently optional in both sales-order forms.
- `GET /sales-orders/{id}/edit` and `POST /sales-orders/{id}` exist, but editing is limited to standalone repeat orders.
- Quotation-originated orders are blocked from editing because their allocations would otherwise become inconsistent.
- `GET /sales-orders/{id}/pdf` exists and is linked from the detail page, but the document renderer is fed through a quotation-shaped object and needs an explicit sales-order snapshot contract.
- Migration `0015` permits a null customer PO, so application-only validation would not fully enforce the rule.

## Business Rules

### Required PO number

- The PO number is required on standalone order creation.
- The PO number is required when converting a quotation into a sales order.
- The PO number is required on every permitted sales-order edit.
- Trim whitespace before validation and persistence.
- Reject an empty value and values longer than the database limit of 100 characters.
- Preserve the entered value when the form returns a validation error.
- Do not silently reuse a quotation number as the customer PO number.

The migration must handle existing null values before changing the database constraint. The preferred rollout is:

1. Inventory existing orders with null or blank PO values.
2. Remediate them with the owner or an explicitly approved legacy value.
3. Change `customer_po_number` to `NOT NULL`.
4. Add a check constraint preventing blank or whitespace-only values if SQL Server supports the chosen expression safely.

If legacy data cannot be remediated during deployment, keep the column nullable temporarily but enforce the rule for all new and edited orders, and record the exception as technical debt. Do not use a fake PO such as `N/A` without business approval.

### Editability

Only an `OPEN` sales order may be edited. A `COMPLETED` order is read-only.

For a standalone repeat order:

- Customer, PO number, payment terms, products, quantities, UOMs, prices, and tax values may be edited.
- Products must still be active and the customer must still exist.
- At least one line is required.
- The edit replaces the order snapshot atomically and recalculates totals.

For an order converted from a quotation:

- The source quotation reference cannot be changed.
- New products not present on the source quotation cannot be added.
- Each source line may be removed from this order or have its quantity changed, subject to the quotation’s remaining allocation.
- The existing allocation must be released and the edited allocation reapplied in the same transaction.
- The order cannot be edited if its quotation is cancelled, expired, or otherwise locked by a later workflow.
- If editing source-linked orders is not approved, show the order as read-only and keep edit support limited to standalone orders. This must be an explicit product decision, not an accidental repository limitation.

## Page Design

### Sales-order detail page

Route: `GET /sales-orders/{id}`

- Show a prominent `Customer PO` field and flag legacy missing values.
- Show `Edit sales order` only when the order is `OPEN` and its source/edit policy permits editing.
- Show `Download PDF` as a primary document action beside edit.
- Show the source relationship:
  - `From quotation QT-...` with a link for converted orders.
  - `Repeat order` / `No quotation linked` for standalone orders.
- Show a clear read-only message for completed or locked orders.

### Create sales-order page

Routes:

- `GET /sales-orders/new`
- `GET /quotations/{id}/sales-order/new`

Change the PO field label to `Customer PO number` and mark it required with browser validation and server validation. Place the field in the order context section so it is visible before the item table.

For quotation conversion, show the PO requirement before allowing `Create sales order`. For standalone creation, keep the searchable customer and item controls unchanged.

### Edit sales-order page

Route: `GET /sales-orders/{id}/edit`

Reuse the create form’s visual language, but make the page state explicit:

- Title: `Edit sales order SO-...`.
- Required customer PO number populated with the current value.
- Customer selector populated with the current customer.
- Existing lines populated with current snapshots.
- Per-item `Remove` control.
- `Add product` control for standalone orders.
- For quotation orders, restrict product choices to source quotation lines and show quoted, allocated, and remaining quantities.
- Live selected subtotal and total.
- Primary action: `Save changes`.
- Secondary action: `Cancel` returning to detail without mutation.
- Warn before leaving with unsaved changes.

When the last line is removed, keep the form usable but disable save and show `Add at least one item`.

## PDF Design

Route: `GET /sales-orders/{id}/pdf`

The endpoint should behave like the quotation PDF endpoint:

- Return `Content-Type: application/pdf`.
- Return `Content-Disposition: attachment; filename="SO-XXXXXXXX.pdf"` using the existing safe filename behavior.
- Set `X-Content-Type-Options: nosniff`.
- Return not found for an invalid or missing order.
- Return a useful server error if rendering fails.

The PDF must render the sales-order snapshot, not current quotation or product data. Include:

- Sales-order number and order date.
- Customer name and address.
- Required customer PO number.
- Sales person and payment terms.
- Source quotation reference when present.
- Only the lines belonging to this sales order.
- UOM, quantity, unit price, tax treatment, line amount, subtotal, VAT, and total.
- A clear `SALES ORDER` document title.

For a repeat order, display `Source: Repeat order` instead of leaving the source metadata blank.

Add a PDF rendering test that checks the PDF signature, document title metadata/content, order number, PO number, and that unrelated quotation lines are not rendered.

## Domain and Repository Changes

Add a shared PO validator in the sales-orders slice so create and edit paths cannot diverge:

```text
ValidateCustomerPO(value string) (string, error)
```

Add explicit repository operations rather than overloading creation behavior:

```text
UpdateStandalone(ctx, orderID, StandaloneOrderInput) (SalesOrder, error)
UpdateFromQuotation(ctx, orderID, customerPO, []LineSelection) (SalesOrder, error)
```

For quotation-originated edits, the transaction must:

1. Lock the sales order and its allocation rows.
2. Confirm the order is `OPEN` and still linked to the same quotation.
3. Lock quotation lines and calculate current allocations excluding this order’s existing allocations.
4. Validate all edited quantities against the available quantity.
5. Update the required PO and order totals.
6. Replace order lines and allocation rows atomically.
7. Commit or roll back the entire edit.

For standalone edits, keep the current snapshot replacement strategy but validate the PO before opening the transaction and again inside the repository boundary.

## Database Migration

Add the next migration after `0018`:

- Remediate or explicitly report existing null/blank PO values according to the deployment decision.
- Alter `dbo.sales_orders.customer_po_number` to `NOT NULL` when data is clean.
- Add an appropriate non-blank constraint.
- Do not alter the quotation foreign-key behavior.
- Preserve the existing sales-order PDF and source-reference columns.

The migration down path must restore the prior nullable behavior only if that is safe for the data state. If down migration cannot safely restore nullability after cleanup, document the operational limitation and test it against the project migration policy.

## Routes

- `GET /sales-orders/new`: standalone create page.
- `POST /sales-orders`: standalone create with required PO.
- `GET /quotations/{id}/sales-order/new`: quotation allocation page with required PO.
- `POST /quotations/{id}/sales-order`: quotation conversion with required PO.
- `GET /sales-orders/{id}`: detail with edit and PDF actions.
- `GET /sales-orders/{id}/edit`: eligible edit page.
- `POST /sales-orders/{id}`: standalone or quotation-originated edit according to source policy.
- `GET /sales-orders/{id}/pdf`: downloadable sales-order PDF.

## Tests and Acceptance Criteria

### PO validation

- Empty PO is rejected on standalone create.
- Empty PO is rejected on quotation conversion.
- Empty PO is rejected on edit.
- Whitespace-only PO is rejected.
- A valid PO is trimmed and persisted.
- Overlong PO is rejected without losing selected lines.
- Legacy orders with a missing PO are handled according to the migration policy.

### Edit behavior

- Open standalone order opens with current customer, PO, terms, and lines.
- Removing one item and saving removes only that item.
- Adding a product and saving persists the new item.
- Removing all items prevents saving.
- Completed orders cannot be edited.
- Quotation-originated edit either updates allocations safely or is clearly blocked by the approved source policy.
- Concurrent quotation edits cannot over-allocate a quotation line.
- Failed edits leave the original order and allocations unchanged.

### PDF behavior

- Detail page exposes the download action.
- Download returns a valid PDF with an `SO-` filename.
- PDF contains the order number and customer PO.
- PDF contains only the order’s lines.
- Repeat-order PDF identifies itself as having no quotation source.
- PDF generation does not mutate order or quotation data.

## Delivery Sequence

1. Decide whether quotation-originated `OPEN` orders are editable or read-only.
2. Decide how existing null/blank customer PO values will be remediated.
3. Add shared required-PO validation and update create/conversion forms.
4. Add migration `0019` for the database PO constraint.
5. Complete standalone edit error preservation and source-policy gating.
6. Implement quotation-originated allocation-aware editing if approved.
7. Harden the sales-order PDF around an explicit order snapshot and add rendering tests.
8. Add route, handler, repository, migration, and acceptance tests.
9. Perform live SQL and browser verification, then update the implementation ladder.

## Decisions Required Before Coding

- Should quotation-originated open sales orders be editable, or permanently read-only after conversion?
- Must existing null/blank PO values be manually remediated before migration, and what approved legacy value is allowed if needed?
- Is the customer PO unique per customer, per quotation, or only required without uniqueness?
- Can the customer be changed during a standalone order edit?
- Can prices and tax values be changed during an edit, or should edits be limited to PO and quantities?
- Should a sales-order PDF be downloadable for every status, including completed orders?
