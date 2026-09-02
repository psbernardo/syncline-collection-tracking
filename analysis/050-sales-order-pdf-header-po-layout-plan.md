# Sales Order PDF Header and PO Layout Plan

## Goal

Ensure the downloadable sales-order PDF always displays the required customer PO number and that the sales-order metadata header has enough vertical and horizontal space for every value without overlapping the item table.

## Current-State Analysis

The sales-order handler already passes the PO into PDF metadata as `PO NUMBER`. The problem is in the shared PDF layout:

- `drawCustomer` starts metadata at `y = 188`.
- Each metadata row advances by a fixed `27` points.
- The sales-order metadata currently contains six rows:
  - `SALES ORDER NO.`
  - `ORDER DATE`
  - `QUOTATION REF #`
  - `SALES PERSON`
  - `PO NUMBER`
  - `PAYMENT TERMS #`
- The item table is hard-coded to start at `pdfFirstTableY = 297`.
- With six rows, the fifth and sixth metadata rows are drawn at approximately `296` and `323`, so the PO/payment fields collide with the table header and first item row.
- `drawCustomer` writes metadata values in a fixed single line and does not calculate space based on value length.
- The renderer is shared by quotation and sales-order documents, so a layout change must preserve quotation PDF output and pagination behavior.

The screenshot confirms the overlap: the sales-order header metadata continues into the table area, and the PO number is not presented as a clean, readable header field.

## Required PDF Header Content

For a sales order, the right-side header must show:

1. Sales order number, for example `SO-00000003`.
2. Order date.
3. Quotation reference when quotation-originated.
4. Sales person.
5. Customer PO number, for example `001`.
6. Payment terms.

For a standalone repeat order, show `REPEAT ORDER` or `Source: Repeat order` in the quotation-reference slot. Do not leave the sales-order identity blank.

The PO must be separate from the quotation reference and must not be inferred from the quotation number.

## Layout Strategy

### Dynamic header boundary

Replace the fixed table start assumption with a calculated header boundary:

- `drawCustomer` returns the bottom Y coordinate occupied by the customer/ship-to block and metadata.
- `drawTableHeader` starts at the greater of:
  - The existing minimum table position for the document type.
  - The calculated metadata bottom plus a safe gap, such as 18 points.
- The item table must never begin before the final metadata row has ended.

The calculation must happen on the first page only. Continuation pages continue to use the compact table header position.

### Metadata row sizing

Use a metadata layout with:

- A fixed label column sized to fit `PAYMENT TERMS #`.
- A value column extending to the right page margin.
- A row height calculated from wrapped label/value text, rather than an unconditional 27-point increment.
- A minimum row height of approximately 20 to 22 points for normal values.
- A safe gap after the last row before the item table.

Values such as long sales-person names, long PO references, or future payment-term labels must wrap or shrink safely within the value column. They must never draw over the table.

Do not solve the problem by simply clipping the PO or removing metadata.

### Header column widths

Measure the right header area from its actual X origin to `pdfWidth - pdfMargin`. Adjust the label/value widths together so:

- Labels remain readable.
- Values are right-aligned consistently.
- A normal SO number and PO number fit on one line.
- Long values use controlled wrapping within the value column.

Keep the left `BILL TO` and `SHIP TO` sections independent from the right metadata block.

## Sales-Order Renderer Contract

Keep the quotation renderer reusable, but make the sales-order metadata explicit:

```text
SALES ORDER NO.  SO-00000003
ORDER DATE       August 23 2026
QUOTATION REF #  QT-00000024
SALES PERSON     Alma Mae Bernardo
PO NUMBER        001
PAYMENT TERMS #  Net 30
```

The handler should pass `order.Number` for `SALES ORDER NO.` and `order.CustomerPONumber` for `PO NUMBER`. The quotation reference must remain a separate field.

The renderer should reject a sales order document view with an empty required PO rather than generating a visually incomplete document, unless legacy-data compatibility has been explicitly approved.

## Implementation Changes

### PDF layout code

Update `internal/slices/quotations/pdf.go`:

- Introduce a metadata layout helper that returns the consumed bottom Y coordinate.
- Introduce wrapped metadata value rendering using the existing font and text helpers.
- Calculate the first table Y position from the metadata layout.
- Keep continuation pages compact and free of the full customer metadata block.
- Preserve quotation-specific behavior that omits the duplicate quote-number metadata row when the title is `QUOTATION`.
- Keep the existing A4 dimensions, margins, table columns, and totals pagination contract unless the dynamic header requires a small first-page adjustment.

### Sales-order handler

Update `internal/slices/salesorders/handler.go`:

- Pass `order.Number` as the sales-order number metadata.
- Pass `order.CustomerPONumber` as the PO metadata.
- Pass `order.QuotationNumber` only as quotation reference.
- Use `Repeat order` as the source value when `QuotationNumber` is empty.
- Keep the existing safe `SO-XXXXXXXX.pdf` filename.

### Test fixtures

Use fixtures containing:

- A short PO such as `001`.
- A longer PO such as `CUSTOMER-PO-2026-000001`.
- A long sales-person value.
- A quotation-originated order.
- A standalone repeat order.

## PDF Acceptance Tests

Add or extend tests to verify:

- Sales-order PDF output begins with a valid `%PDF-` signature.
- Output contains the sales-order number metadata.
- Output contains the customer PO metadata.
- Output contains quotation reference separately for converted orders.
- Standalone output identifies the repeat-order source.
- The PDF remains valid with a long PO and long metadata values.
- The first item table begins after the last metadata row. Prefer testing the calculated Y coordinate directly through a small layout helper rather than relying only on visual inspection.
- Quotation PDF rendering remains valid and keeps its existing page count expectations.
- Multi-page sales orders still render continuation headers and totals correctly.

If PDF text is compressed or difficult to inspect, test the metadata layout model directly and use a PDF text extraction tool in an integration/browser verification step.

## Visual Verification Checklist

- PO number is visible in the right header block.
- Sales-order number and quotation reference are not confused.
- Payment terms do not overlap the first item row.
- The item table header has clear whitespace below the metadata block.
- Long PO values remain inside the right header column.
- Long sales-person values wrap without changing the left customer block.
- The first page and continuation pages maintain the existing visual hierarchy.
- Quotation PDFs are unchanged except for any intentional shared spacing improvement.

## Delivery Sequence

1. Add a renderer-level metadata layout calculation and test its consumed height.
2. Replace fixed first-page table positioning with the calculated header boundary.
3. Ensure the sales-order handler supplies number, PO, quotation reference, and repeat-order source separately.
4. Add short and long PO PDF fixtures.
5. Run quotation and sales-order PDF regression tests.
6. Perform browser download verification and inspect the rendered PDF visually.

## Decisions Required Before Coding

- Should long metadata values wrap to a second line or use a smaller font after a threshold?
- Should `PAYMENT TERMS #` be renamed to `PAYMENT TERMS` in the PDF?
- Should the sales-order number appear both in the title area and metadata block, or only in metadata?
- Should a missing legacy PO block PDF generation or display a legacy placeholder?
