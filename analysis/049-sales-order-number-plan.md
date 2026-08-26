# Sales Order Number Plan

## Goal

Give every sales order a system-generated sales-order number, show that number while creating or editing the order, and print it clearly on the downloadable sales-order PDF.

This applies equally to:

- Orders converted from quotations.
- Standalone repeat orders created without quotations.

The user must not type or edit the sales-order number.

## Current-State Analysis

The current implementation already has the database foundation:

- `dbo.sales_orders.sales_order_number` is required and unique.
- Migration `0015` creates `SEQ_sales_order_numbers`.
- `createOrderTx` obtains the next sequence value and formats `SO-%08d`.
- The detail page displays `SalesOrder.Number` as its heading.
- The edit page does not show the persisted number in its form context.
- Create pages do not show a sales-order number before submission.
- The PDF handler currently uses `order.QuotationNumber` for `REF #`, which is incorrect for standalone orders and does not clearly identify the sales order.

The number must remain stable across edits and downloads. Editing an order must never allocate a new number.

## Number Contract

- Format: `SO-XXXXXXXX`, where `XXXXXXXX` is an eight-digit, zero-padded sequence value.
- Generation source: `dbo.SEQ_sales_order_numbers`.
- Persistence: `sales_orders.sales_order_number`.
- Database uniqueness: retain `UQ_sales_orders_number`.
- Mutability: generated once at order creation and immutable thereafter.
- Display: read-only in create/edit forms, prominent on detail/list pages, and visible in the PDF.
- The quotation number remains a separate source reference, for example `Quotation reference: QT-00000024`.

Sequence gaps are acceptable. A number preview may consume a sequence value if the user abandons the form; reusing numbers creates a greater audit and concurrency risk.

## Create-Page Behavior

### Standalone repeat order

Route: `GET /sales-orders/new`

1. Ask the repository number generator for the next sales-order number.
2. Render it in the order context card as:

   `Sales order number: SO-XXXXXXXX`

3. Render the value as read-only text, not an editable input.
4. Carry the generated value into the POST using a server-controlled reservation/token mechanism.
5. On submit, persist that exact number after strict format validation.
6. If the reservation is unavailable or already used, return a clear error and do not create a duplicate order.

### Quotation conversion

Route: `GET /quotations/{id}/sales-order/new`

Use the same number-generation and display behavior. The page should show both:

- `Sales order number: SO-XXXXXXXX`
- `Source quotation: QT-XXXXXXXX`

Do not use the quotation number as the sales-order number and do not overwrite the quotation number in the sales-order snapshot.

### Reservation strategy

Prefer one of these approaches, in order:

1. Create a short-lived server-side number reservation record containing the generated number, form purpose, expiry, and consumed timestamp.
2. If the application has signed state/session support, use a signed short-lived reservation token.
3. As a minimal implementation, include the generated number as a hidden value, validate its exact format, and still enforce the database unique constraint. The repository must not accept arbitrary user-entered numbering, and a mismatch should generate a fresh server number or reject the request according to the chosen policy.

The reservation should expire after a reasonable period, such as 30 minutes. Expired forms should show `This sales-order number expired. Reload the page to get a new number.`

## Edit-Page Behavior

Route: `GET /sales-orders/{id}/edit`

- Load the persisted `SalesOrder.Number`.
- Show `Sales order number` in the order context card.
- Render it as read-only text.
- Do not generate or request a new number.
- Do not include the number in update fields except as a display-only value or server-verified path context.
- Preserve the same number after saving PO, quantity, product, or allocation changes.

The edit warning should explicitly state:

> The sales-order number is permanent and will not change when you save this edit.

## Detail and List Pages

### Detail

Keep the number as the primary page heading and repeat it in the document metadata card:

- Sales order number: `SO-XXXXXXXX`
- Customer PO: `...`
- Source: `QT-XXXXXXXX` or `Repeat order`

### List

Make the order number the first, linked column. It should remain readable on mobile and should not be replaced by a database ID.

## PDF Requirements

Route: `GET /sales-orders/{id}/pdf`

The PDF must use `SalesOrder.Number` as the document identity:

- Title: `SALES ORDER`.
- Metadata label: `SALES ORDER NO.` with value `SO-XXXXXXXX`.
- File name: `SO-XXXXXXXX.pdf`.
- Customer PO shown separately.
- Quotation reference shown separately when the order originated from a quotation.
- Repeat orders show `Source: Repeat order`.

Do not populate the sales-order reference from `QuotationNumber`. For standalone orders, quotation reference is empty but the sales-order number must still be present.

The renderer should receive a dedicated sales-order document view model or metadata contract rather than relying on a quotation-shaped object whose `Number` field may contain the wrong document number.

## Domain and Repository Changes

Expose a number-generator contract in the sales-orders slice:

```text
NextNumber(ctx) (string, error)
```

Keep final number generation and persistence inside the creation transaction. The create operation must:

- Generate or consume the approved reservation.
- Validate the `SO-XXXXXXXX` format.
- Persist the number on the order header.
- Rely on the unique database constraint as the final guard.

Update operations must not include `sales_order_number` in their update maps.

Add a shared formatter/validator if not already present:

```text
GeneratedSalesOrderNumber(sequence int64) string
ValidSalesOrderNumber(value string) bool
```

## Routes

- `GET /sales-orders/new`: display generated number for standalone creation.
- `POST /sales-orders`: persist generated number.
- `GET /quotations/{id}/sales-order/new`: display generated number for quotation conversion.
- `POST /quotations/{id}/sales-order`: persist generated number.
- `GET /sales-orders/{id}/edit`: display existing immutable number.
- `POST /sales-orders/{id}`: preserve existing number.
- `GET /sales-orders/{id}`: display existing number.
- `GET /sales-orders/{id}/pdf`: print existing number and use it as the filename.

## Tests and Acceptance Criteria

### Generation and persistence

- Standalone create displays an `SO-XXXXXXXX` number before submission.
- Quotation conversion displays an `SO-XXXXXXXX` number before submission.
- Created orders persist the displayed number.
- Two new orders receive different numbers.
- A failed creation does not create an order with the number.
- Sequence-generated numbers conform to the exact format.
- Duplicate number attempts are rejected or safely regenerated.

### Edit behavior

- Edit page displays the existing sales-order number.
- Saving an edit does not change the number.
- Editing PO, quantities, or lines does not allocate a new number.
- Quotation allocation edits preserve the original number.
- Completed/locked orders remain read-only but still display their number.

### PDF behavior

- PDF contains `SALES ORDER NO. SO-XXXXXXXX`.
- PDF filename is `SO-XXXXXXXX.pdf`.
- PDF contains customer PO separately.
- Quotation-originated PDFs contain both sales-order and quotation numbers.
- Repeat-order PDFs contain the sales-order number and `Repeat order` source.
- Downloading the PDF does not generate or mutate a number.

### UI behavior

- Create number is visibly read-only and cannot be typed over.
- Edit number is visibly read-only and cannot be changed.
- Number is visible without requiring the user to open a secondary panel.
- Number remains legible on desktop and mobile layouts.

## Delivery Sequence

1. Add number formatter/validator and `NextNumber` repository contract.
2. Decide and implement the short-lived reservation strategy.
3. Add generated-number display to both create pages.
4. Add immutable number display to the edit page.
5. Ensure all update paths preserve the existing number.
6. Update sales-order PDF metadata and document view model.
7. Add handler, repository, reservation, and PDF tests.
8. Verify live SQL sequence behavior and browser rendering.

## Decisions Required Before Coding

- Should opening a create page reserve a number, accepting sequence gaps when users abandon forms?
- Is a short-lived reservation table acceptable, or should the first implementation use a signed/hidden form value?
- Should number reservation happen separately for quotation conversion and standalone creation, or use one shared sequence reservation flow?
- Should the PDF show the number in the header, metadata block, or both?
