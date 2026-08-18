# 042 Apply Shared Sales Order Layout to Quotations

## Objective

Apply the new shared document layout to the quotation download PDF while keeping the existing quotation business rules and totals.

The quotation and Sales Order PDFs must use the same visual structure, logo, colors, fonts, table style, totals block, and footer. The document title and document-specific metadata remain different.

## Target Quotation Layout

Use the Sales Order layout with the title:

```text
QUOTATION
```

The quotation page should contain:

- Syncline logo and seller information
- quotation title
- quotation date
- quotation number
- validity date
- payment terms
- Bill To customer
- Ship To customer
- quotation line items
- quotation totals
- shared Syncline footer

## Quotation Header Metadata

The right-side header metadata should remain quotation-specific:

| Label | Value |
|---|---|
| `QUOTE DATE` | `Quotation.CreatedAtUTC` |
| `QUOTE NUMBER` | `Quotation.Number` |
| `VALID UNTIL` | `Quotation.ValidityDate` or `Not specified` |
| `PAYMENT TERMS` | `Quotation.TermsDays` formatted as `Net 30` or the approved wording |

Do not show Sales Order-only fields on the quotation:

- Sales Order number
- Customer PO number
- Sales person
- Sales Order status

## Customer Information

Replace the current single customer block with the shared two-part layout:

### Bill To

- Company name from `Quotation.CompanyName`
- Billing address from `Quotation.CustomerAddress`
- Customer phone from `Quotation.CustomerContactNumber`
- Contact person from `Quotation.CustomerContactPerson`
- Customer email from `Quotation.CustomerEmail`

### Ship To

- Company name from the same customer account
- Delivery address from the account delivery address
- Customer phone/contact details where useful

The current quotation projection only loads the billing address. Add `CustomerDeliveryAddress` to the quotation model and repository projection, or introduce a dedicated customer PDF projection that contains both addresses.

Do not query the database inside the PDF renderer. The handler/repository layer must prepare the complete customer-facing document data first.

## Line Item Table

Use the same table columns as the new Sales Order layout:

| Column | Quotation source | Display |
|---|---|---|
| `#` | line index | sequential number |
| `DESCRIPTION` | `Line.ProductName` | wrapped text |
| `QTY` | `Line.Quantity` | plain numeric value, no currency |
| `UNIT` | `Line.UOM` | displayed below or beside quantity |
| `RATE` | `Line.UnitPrice` | Philippine peso |
| `TAX %` | `Line.TaxRate` / quotation tax rule | `12%` for VAT12 |
| `AMOUNT` | `Line.VATInclusiveTotal` for VAT-inclusive rule | Philippine peso |

Rules:

- Quantity must never use the peso formatter.
- Prices and amounts use the existing peso formatter.
- `TAX %` displays `12%` for `VAT12` and `0%`/blank for `NONE`, according to the final visual decision.
- The table header repeats on additional pages.
- Long descriptions wrap without clipping.
- Supplier cost, commission, and profitability remain excluded.

## Totals

Keep the totals already implemented:

- `TOTAL SALES (VAT INC)` from `Quotation.Totals.Total`
- `LESS: VAT` from `Quotation.Totals.Tax`
- `AMOUNT: NET OF VAT` from `Quotation.Totals.Subtotal`

The renderer must use persisted quotation totals and must not recalculate them.

For `NONE` tax, display zero VAT and the subtotal as the total according to existing domain behavior.

## Shared Renderer Changes

Refactor the current renderer so it receives a shared document projection containing:

- document title
- metadata rows
- seller profile
- Bill To details
- Ship To details
- lines
- totals
- footer

Both document types should construct this projection:

```text
Quotation -> shared PDF document with title QUOTATION
SalesOrder -> shared PDF document with title SALES ORDER
```

Avoid separate drawing functions for quotation and Sales Order. Only the mapping layer should differ.

## Implementation Steps

### 1. Extend Quotation Customer Projection

- Add `CustomerDeliveryAddress` to the quotation view data.
- Update the quotation repository customer query to load both billing and delivery addresses.
- Preserve customer email from migration `0014`.
- Add unit tests for both addresses.

### 2. Create Shared PDF View Model

- Move common PDF fields into a shared `PDFDocument` structure.
- Add `BillTo` and `ShipTo` structures.
- Add metadata rows rather than hard-coding quotation labels in the drawing code.
- Keep internal quotation fields out of the view model.

### 3. Update Quotation Mapping

- Map quotation metadata into the shared view model.
- Map the customer billing and delivery addresses.
- Map tax percentage text.
- Keep quotation totals unchanged.

### 4. Update Renderer Layout

- Use the Sales Order table geometry for quotations.
- Add the `#` column.
- Split description and quantity/UOM according to the shared layout.
- Keep the Syncline logo and footer behavior unchanged.
- Keep landscape A4 and multi-page support.

### 5. Update Quotation Detail Download

- Keep the existing `Download PDF` action and route.
- Ensure it uses the shared projection with title `QUOTATION`.
- Do not change quotation creation or pricing behavior.

### 6. Verify Sales Order Regression

- Confirm Sales Order PDFs still render with title `SALES ORDER`.
- Confirm Sales Order-specific metadata remains present.
- Confirm both document types use the same line table and totals layout.

## Acceptance Criteria

- The quotation PDF visually matches the new Sales Order layout.
- The only document-level visual difference is the title and document-specific metadata.
- Quotation PDFs show both Bill To and Ship To sections.
- Quotation metadata shows quote date, quote number, validity, and payment terms.
- The line table contains `#`, description, quantity, unit, rate, tax percentage, and amount.
- Quantity values have no currency prefix.
- Peso formatting is used only for monetary values.
- VAT-inclusive and no-tax quotations display correct line and summary values.
- Supplier cost, commission, and profitability are not exposed.
- Logo, footer, borders, colors, fonts, and page orientation match the Sales Order PDF.
- Long descriptions and multi-page quotations render without clipping.
- Existing Sales Order PDF tests continue to pass.

## Remaining Confirmation

One display detail should be confirmed before coding:

- For quotations with `TaxDefaultCode = NONE`, should the `TAX %` column show `0%` or remain blank?
