# 040 Quotation PDF Screenshot Layout Plan

## Objective

Implement the quotation PDF download using the recommended native Go library, `github.com/signintech/gopdf`, with a layout closely matching the supplied Syncline quotation screenshot.

This document refines `analysis/039-quotation-pdf-download-plan.md`. It is a layout and implementation plan only; no PDF code is added yet.

## Screenshot Analysis

The reference image is approximately 847 x 792 pixels and represents a single landscape quotation page with a thin purple border. The content is compact at the top, leaves flexible whitespace for the item table, and anchors totals and the closing message near the bottom.

### Visual Language

- Page: white background, thin dark-purple outer border.
- Primary color: dark purple for headings, labels, borders, and totals.
- Secondary fill: light gray for the title strip, customer section heading, and item table header.
- Typography: bold uppercase labels and headings; regular dark text for values; italic purple closing message.
- Logo: large company logo occupying the upper-left header block.
- Alignment: labels mostly left-aligned; quotation metadata values right of the labels; numeric columns right-aligned.
- Style: spreadsheet-like grid, minimal decoration, strong horizontal rules, no internal profitability section.

### Page Geometry

The screenshot should be reproduced as an A4 landscape page rather than A4 portrait.

Recommended PDF geometry:

| Area | Approximate screenshot proportion | A4 landscape implementation |
|---|---:|---:|
| Outer border | 1-2 px | 0.5-0.8 pt stroke, inset 4-6 mm |
| Header height | 15% | 38-45 mm |
| Seller contact block | 10% | 24-30 mm |
| Customer heading/details | 15% | 38-45 mm |
| Item table | 38-45% | flexible region from roughly 105 mm to 235 mm |
| Totals | 12% | right-aligned block around 245-265 mm |
| Closing message/footer | 10% | bottom 18-25 mm |

Use actual measured coordinates in millimeters, not screenshot pixels. Keep all coordinates in one layout file or renderer struct so future branding changes do not scatter magic numbers through the generator.

### Header

The header is split vertically:

- Left block: company logo, approximately 36% of page width and the full header height.
- Right block: gray title strip containing `QUOTATION`, followed by four metadata rows:
  - `DATE`
  - `QUOTE NUMBER`
  - `VALID UNTIL`
  - `TERMS`

The title is centered, uppercase, dark purple, and larger than all other text. Metadata labels are bold dark purple. Metadata values are regular black/dark gray.

### Seller Contact Block

Below the header, show three seller rows:

- `ADDRESS: ...`
- `EMAIL: ...`
- `CONTACT NUMBER: ...`

The labels are bold purple and the values are regular black. Use wrapped text for a long address, but keep the block's height deterministic by reserving two lines for address when necessary.

The seller data is not currently represented in the quotation or company-account models. It should come from application-level company profile configuration, not from the selected customer account. See the data-gap section below.

### Customer Block

Use a full-width gray section bar labeled `CUSTOMER`, followed by a two-column customer detail grid:

- `COMPANY NAME`
- `ADDRESS`
- `PHONE`
- `CONTACT PERSON`
- `EMAIL`

Labels occupy a fixed narrow column and are bold purple. Values start at a consistent x-coordinate and are bold black. The screenshot uses the customer's billing address, not the delivery address, so this must be stated in the document contract.

The current `CompanyAccount` model has company name, contact person, billing/delivery address, and contact number, but it does not have an email field. The screenshot therefore requires either an email field addition or a deliberate omission/placeholder.

### Item Table

The table has a gray header row with purple borders and these columns:

| Column | Screenshot meaning | Alignment | Source |
|---|---|---|---|
| `QTY` | Quantity | Right | `Quotation.Lines[].Quantity` |
| `UNIT` | Unit of measure | Center | `Quotation.Lines[].UOM` |
| `ITEM DESCRIPTION` | Product description/name | Left | `Quotation.Lines[].ProductName` |
| `UNIT PRICE` | Customer unit price | Right | `Quotation.Lines[].UnitPrice` |
| `TOTAL` | Customer line total | Right | `Quotation.Lines[].VATInclusiveTotal` or persisted line total per tax convention |

Important layout decisions:

- Do not show product SKU unless the business wants it in the description column; the screenshot shows only the product description.
- Do not show supplier cost, tax code, commission, or margin.
- Use the VAT-inclusive amount in the `TOTAL` column when the quotation tax rule is VAT-inclusive, matching the screenshot.
- Repeat the table header on every additional page.
- Allow item descriptions to wrap to two or more lines and increase row height accordingly.
- Use a minimum row height for visual consistency, but do not reserve a large empty table area on multi-page documents.

### Totals Block

The screenshot places totals on the lower-right side with no enclosing box:

- `TOTAL SALES (VAT INC)`
- `LESS: VAT`
- `AMOUNT: NET OF VAT`

Values are Philippine peso amounts, right-aligned, bold purple, and formatted with comma separators and two decimal places.

The screenshot example validates the current VAT-inclusive rule:

```text
VAT-inclusive total:  ₱75,000.00
Net of VAT:          ₱66,964.29
VAT:                  ₱8,035.71
```

This corresponds to 12% VAT included in the gross amount. The renderer must use the persisted `Totals.Total`, `Totals.Tax`, and `Totals.Subtotal` values rather than independently recalculating them.

If withholding tax is to appear in the PDF, add a separate agreed row. Do not silently insert it into the screenshot layout because the current reference does not show it.

### Closing Message

At the bottom center, show an italic purple closing block:

```text
If you have any questions about this price quote, please contact us
(seller email, seller phone)
Thank You For Your Business!
```

The email and phone must come from the seller profile configuration. Do not hard-code the screenshot's values in the renderer.

## Existing Data Mapping

### Available Today

The quotation repository already loads:

- quotation number
- created date
- validity date
- terms days
- customer company name
- quotation lines and product name
- quantity and UOM
- customer unit price
- persisted line totals
- persisted VAT/tax amount
- persisted net subtotal
- persisted gross total

The company-account domain already stores:

- company name
- contact person
- billing address
- delivery address
- contact number
- TIN number

### Missing or Incomplete

The screenshot requires these additions or decisions:

1. **Seller profile:** seller name, logo, address, email, phone, and optional footer wording are not currently visible in the quotation model or configuration contract.
2. **Customer email:** `CompanyAccount` has no email field, but the screenshot includes it.
3. **Quotation notes:** `Quotation.Notes` exists in persistence, but the current quotation form does not expose it. The screenshot's closing message is seller boilerplate, not quotation notes.
4. **Customer detail loading:** `Quotation` currently stores only `CompanyName`; the repository should load a dedicated customer document projection for address, phone, contact person, and email rather than making the PDF renderer query the database.
5. **Currency:** current money formatting uses PHP pesos. The PDF document contract should explicitly identify `PHP`/`₱` and use an embedded font that supports the peso symbol.

## Proposed Document Model

Do not pass the full `Quotation` domain object directly to `gopdf`. Create a customer-facing projection:

```go
type QuotationPDFDocument struct {
    Seller   SellerProfile
    Customer CustomerDetails
    Number   string
    Date     time.Time
    Validity *time.Time
    Terms    string
    Lines    []QuotationPDFLine
    Totals   QuotationPDFTotals
    Footer   string
}
```

The projection must not contain:

- supplier cost
- commission type/rate/amount
- estimated profit
- internal margin
- database IDs unless needed for metadata/debugging

Use a repository method such as `FindPDFDocument(ctx, id)` or an application-level assembler that loads the quotation and customer profile in a controlled way. The handler should only coordinate HTTP and rendering.

## `gopdf` Implementation Plan

### 1. Dependency and Assets

- Pin `github.com/signintech/gopdf` to a reviewed version.
- Add a permitted TrueType font with regular, bold, and italic variants using `//go:embed`.
- Verify that the selected font contains `₱`, accented characters, and expected customer/product Unicode.
- Add the font and logo license notices to the existing third-party notice inventory.
- Embed the logo as PNG or JPEG. Prefer PNG if the supplied logo has transparency; otherwise use a sufficiently high-resolution JPEG.

### 2. Renderer Structure

Create a focused renderer with these responsibilities:

- initialize A4 landscape page and margins
- register fonts
- draw outer border
- draw header and seller profile
- draw customer block
- draw table header and rows
- paginate rows and repeat table headers
- draw totals at the end of the table
- draw closing message and footer
- write PDF bytes to the supplied writer

Use helper functions with explicit names such as `drawHeader`, `drawCustomerBlock`, `drawLineTable`, `drawTotals`, and `drawFooter`. Keep business calculations outside these functions.

### 3. Coordinate and Style Constants

Define a renderer style/configuration object containing:

- page dimensions and margins
- purple and gray RGB colors
- border width
- font names and sizes
- column widths
- row heights
- header/footer positions
- currency/date formats

This makes the screenshot's visual language adjustable without changing the document data mapping.

### 4. Text and Wrapping

- Use embedded fonts for all visible text.
- Measure text before drawing it.
- Wrap descriptions and addresses against their column widths.
- Clip or wrap values rather than allowing them to overlap adjacent columns.
- Keep label and value baselines consistent in the header/customer blocks.
- Test long company names and long addresses using the same font metrics as production.

### 5. Pagination

The screenshot is a one-page reference, but production quotations cannot assume three lines.

- Reserve enough bottom space for totals and the closing message.
- Add line rows until the remaining space is below the minimum table-row threshold.
- Start a new page and redraw the outer border/header or a compact repeated header according to the approved document contract.
- Repeat the item table header on every page.
- Draw totals and closing content only after the last line.
- If the closing message cannot fit with totals, move both to a clean final page rather than clipping them.

### 6. Download Endpoint

Add `GET /quotations/{id}/pdf` next to the existing quotation routes.

The handler should:

1. Parse the route ID.
2. Load the customer-facing PDF document projection.
3. Generate into a bounded `bytes.Buffer`.
4. Set `Content-Type: application/pdf`.
5. Set `Content-Disposition: attachment; filename="<quotation-number>.pdf"` after sanitizing the number.
6. Set `X-Content-Type-Options: nosniff`.
7. Write the bytes only after successful generation.

Add a `Download PDF` button to the quotation detail action area. Keep the existing HTML page as the internal review page and do not use browser print-to-PDF.

## Testing Plan

### Unit Tests

- Customer projection includes seller/customer fields and excludes internal fields.
- VAT-inclusive example produces the expected displayed values from persisted totals.
- Filename uses the quotation number and rejects path separators/control characters.
- Dates render in the agreed format.
- Missing validity date and optional customer fields do not break layout.
- Long descriptions and addresses produce wrapped lines.

### Handler Tests

- `GET /quotations/7/pdf` returns status 200.
- Response has `application/pdf` content type.
- Response has attachment disposition with a safe filename.
- Missing quotation returns 404.
- Renderer failure returns 500 without partial PDF output.
- Internal fields do not appear in generated PDF text.

### PDF Validation and Visual Tests

- Validate output with `pdfcpu validate` or an equivalent PDF parser.
- Confirm the file opens in Chrome, Firefox, Adobe Acrobat Reader, and a mobile PDF viewer.
- Verify the page is A4 landscape, with correct margins and no clipping.
- Verify embedded fonts and the `₱` symbol.
- Test no-tax and VAT-inclusive quotations.
- Test one, three, 20, and 200 lines.
- Test long addresses, long descriptions, Unicode values, and empty optional values.
- Compare a generated fixture visually against the supplied screenshot for header proportions, colors, spacing, totals placement, and footer placement.

## Acceptance Criteria

- The PDF visually follows the supplied screenshot: landscape page, purple border, logo/header split, gray section bars, customer block, five-column item table, right-side totals, and centered closing message.
- The PDF is generated with `gopdf` inside the existing Go service and does not require Node, Chromium, or an external PDF service.
- The existing distroless Docker image remains usable.
- Customer PDF values come from persisted quotation/account data and use the existing money calculations.
- Supplier cost, commission, and profitability never appear in the customer PDF.
- The PDF remains valid and readable for multi-page quotations.
- The download uses a safe quotation-number filename and appropriate PDF response headers.
- Seller and customer email data gaps are resolved before implementation is marked complete.

## Decisions Required Before Coding

1. Confirm the document is customer-facing and must exclude internal profitability data.
2. Confirm the seller legal name, address, email, phone, and logo source.
3. Decide whether to add customer email to `CompanyAccount` and its migration/form flow.
4. Confirm billing address versus delivery address in the customer block.
5. Confirm whether withholding tax should be displayed.
6. Confirm whether the exact screenshot closing message is approved for all quotations.
7. Confirm whether the PDF should repeat the full seller header on page two or use a compact repeated header.
