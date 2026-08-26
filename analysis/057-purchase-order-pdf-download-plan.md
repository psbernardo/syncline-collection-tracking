# 057 Purchase Order PDF Download Plan

## Goal

Add a `Download PDF` action to the purchase-order detail page. The PDF must be a supplier-facing procurement document generated on the server from the persisted purchase-order snapshot.

The implementation should align with the existing quotation download behavior for HTTP semantics, PDF technology, pagination, typography, and document quality, but it must remain a separate purchase-order implementation. Do not call the quotation renderer or pass a quotation into the purchase-order renderer.

## Current-State Findings

- Quotations already expose `GET /quotations/{id}/pdf`, use `gopdf`, buffer output before writing, and return an attachment with a sanitized document number filename.
- `internal/shared/pdf/helpers.go` contains reusable low-level PDF setup, font discovery, seller defaults, formatting, wrapping, and safe filename helpers.
- Purchase orders currently expose only `GET /purchase-orders/{id}` for HTML detail. There is no PDF route, renderer, PDF view model, or PDF-specific test.
- Purchase-order detail data is loaded by `purchaseorders.Repository.FindByID` and includes the complete persisted header and line snapshot: number, supplier, PO date, status, payment terms, expected delivery, notes, source order references, supplier SKU, UOM, quantity, unit cost, line total, subtotal, and total.
- Direct purchase orders have no sales-order source. Sales-order allocations can contain multiple source-order references through line-level sources, even though the legacy header field supports only one `SalesOrderID`.
- Purchase-order totals are supplier-cost totals. They are not quotation customer sales totals and do not currently contain tax fields.
- The purchase-order model currently contains supplier name but not supplier address, contact person, phone, or email. A supplier-facing layout therefore needs an explicit decision about whether to add/load those fields or omit them.
- The current quotation renderer is package-local and has quotation-specific labels, customer data, VAT totals, and closing text. Reusing it would risk showing customer-facing fields or incorrect tax semantics.

## Alignment With Quotation Download

Reuse the following established behavior:

- Native Go `gopdf` generation inside the existing single binary.
- A dedicated customer/supplier-facing document projection rather than passing the complete domain object to the renderer.
- `bytes.Buffer` generation before response headers/body are committed.
- `Content-Type: application/pdf`.
- `Content-Disposition: attachment` with a safe persisted-number filename.
- `X-Content-Type-Options: nosniff`.
- A4 layout, embedded/configured font setup, wrapped text, and multi-page line-item pagination.
- Repository snapshot as the source of truth; do not recalculate persisted totals in the PDF layer.
- Renderer and handler tests plus PDF validity and long-document coverage.

Do not reuse the quotation renderer's document model or business presentation:

- Title must be `PURCHASE ORDER`, not `QUOTATION`.
- The primary party is the supplier, not the customer.
- Amounts must use `UnitCost` and persisted purchase-order line `Total`.
- Totals must show purchase-order subtotal/total, not VAT-inclusive sales totals.
- Include procurement/source context and expected delivery where approved.
- Never include quotation commission, supplier margin, customer price, or profitability fields.

The low-level helpers in `internal/shared/pdf` may be used. If a helper is too quotation/invoice-specific, keep the purchase-order equivalent local instead of broadening shared code prematurely.

## Proposed Package Boundary

Add a separate implementation under the purchase-orders slice:

```text
internal/slices/purchaseorders/pdf.go
internal/slices/purchaseorders/pdf_test.go
internal/slices/purchaseorders/handler.go       # route, handler dependency, endpoint
internal/slices/purchaseorders/templates/detail.html  # action only
```

Suggested contract:

```go
type PurchaseOrderPDFDocument struct {
    Seller             sharedpdf.SellerProfile
    Supplier           SupplierDetails
    Number             string
    PODate             time.Time
    Status             string
    PaymentTerms       string
    ExpectedDelivery   *time.Time
    SourceReferences   []string
    Notes              string
    Lines              []PurchaseOrderPDFLine
    Subtotal           money.Amount
    Total              money.Amount
}

type PurchaseOrderPDFLine struct {
    SKU, Name, SupplierSKU, UOM string
    Quantity, UnitCost, Total   money.Amount
    SourceReference             string
}
```

`SupplierDetails` should contain only fields explicitly approved for the document. The renderer must not depend on repository/database queries.

## Data Contract Decisions

Confirm these before coding:

1. **Audience:** supplier-facing PO, internal procurement copy, or both. The initial plan assumes supplier-facing but includes procurement cost fields by necessity.
2. **Supplier identity:** whether the PDF should show only supplier name, or also supplier address/contact fields. If required, extend `PurchaseOrder`/repository projection or add a dedicated `FindPDFDocument` repository method. Do not query from the renderer.
3. **Seller identity:** use the configured Syncline seller profile from `sharedpdf.DefaultSeller()` unless procurement documents require separate legal details.
4. **Status:** include the persisted status (`OPEN`, `CONFIRMED`, or `CANCELLED`) or omit it. If included, display it as document metadata, not as a mutable form value.
5. **Source references:** show all distinct `SalesOrderNumbers` for allocation POs; show `Direct purchase` when none exist. Do not rely only on the legacy single `SalesOrderNumber` field.
6. **Notes:** include persisted PO notes as a supplier-facing notes section. Confirm whether internal notes are possible; if so, split the field before exposing it.
7. **Tax/currency:** use the existing PHP money formatter. Do not add VAT rows unless purchase-order tax persistence is added and explicitly approved.
8. **Document date:** use `PODate`; use the existing local date display convention. Expected delivery is optional and must not render as a zero date.
9. **Filename:** use the purchase-order number through `sharedpdf.SafeFilename(number, "purchase-order")`.

## PDF Layout

Use the quotation/invoice visual language for consistency, while making the content procurement-specific:

- A4 portrait initially, matching the current quotation/invoice renderer geometry and reducing layout risk.
- Branded Syncline header and `PURCHASE ORDER` title.
- Metadata block containing PO date, PO number, status, payment terms, expected delivery, and source reference(s).
- Supplier block with approved supplier identity fields.
- Line-item table with line number, SKU, item description, supplier SKU, quantity, UOM, unit cost, and amount.
- Notes block when notes are present.
- Right-aligned totals block containing `SUBTOTAL` and `TOTAL` using persisted `Subtotal` and `Total`.
- Footer using approved seller contact details and a purchase-order-specific closing label, or no closing text if a supplier-facing message is not approved.

The renderer must:

- Wrap product names, supplier SKUs, source references, notes, and supplier details.
- Repeat the table header on continuation pages.
- Keep totals and notes together where possible.
- Move the final totals/notes section to a clean final page rather than clipping it.
- Render empty optional fields cleanly.
- Preserve line order from `FindByID`.

Keep all coordinates, widths, colors, and font sizes in purchase-order renderer constants/configuration. Do not copy quotation constants and silently retain quotation labels or assumptions.

## Repository/View-Model Work

Preferred minimal option:

- Continue loading the persisted order with `FindByID`.
- Add a pure `purchaseOrderPDFDocument(order PurchaseOrder) PurchaseOrderPDFDocument` mapper in the purchase-orders package.
- Map only the fields needed by the PDF and use `SalesOrderNumbers` for source context.

If supplier contact/address data is required:

- Extend `PurchaseOrder` with explicit supplier display fields and populate them in `FindByID`, or
- Add a repository method such as `FindPDFDocument(ctx, id)` that returns a controlled projection.

Do not make `pdf.go` issue SQL queries, use `salesorders.Repository`, or infer customer quotation data. The handler should coordinate ID parsing, repository loading, mapping, rendering, and HTTP response only.

## HTTP and UI Changes

1. Add `GET /purchase-orders/{id}/pdf` beside the existing purchase-order routes.
2. Parse IDs with the existing `parseID` helper; invalid IDs return `404`.
3. Load the order using `h.purchase.FindByID` so direct and sales-order allocation POs use the same endpoint.
4. Return `404` when the order is missing, matching `detailPage` behavior.
5. Render to a `bytes.Buffer`; return a generic `500` on renderer failure without partial PDF output.
6. Set PDF response headers only after successful rendering.
7. Add `Download PDF` to the purchase-order detail action area for every viewable order, independent of editability/status.
8. Keep the HTML detail page for interactive/internal review; do not use browser print-to-PDF.

The handler should own a separate `*PurchaseOrderPDFRenderer` dependency. For testability, prefer a small renderer interface or constructor injection rather than hard-coding an unreplaceable renderer in endpoint tests. This is a purchase-order dependency and should not be shared with `quotations.Handler`.

## Implementation Sequence

### Phase 1: Confirm document contract

- Resolve audience, supplier fields, status visibility, source-reference wording, notes visibility, currency/tax behavior, and footer wording.
- Confirm whether existing supplier data is sufficient or requires a repository/domain projection change.

### Phase 2: Add purchase-order document mapping

- Define `PurchaseOrderPDFDocument`, `SupplierDetails`, and `PurchaseOrderPDFLine` in `purchaseorders/pdf.go` or a nearby focused file.
- Map persisted order/line values without recalculating totals.
- Ensure customer quotation prices, commission, margin, and other unrelated fields cannot enter the projection.

### Phase 3: Implement the separate renderer

- Configure `gopdf` through shared low-level helpers.
- Implement purchase-order-specific header, supplier block, metadata, line table, totals, notes, and footer.
- Implement wrapping and multi-page pagination.
- Use the shared money formatter and safe filename utility.

### Phase 4: Integrate endpoint and detail action

- Add renderer dependency to `purchaseorders.Handler` and initialize it in `NewHandler`.
- Register the PDF route and implement the endpoint.
- Add the detail-page download link.

### Phase 5: Verify and harden

- Run unit, handler, repository projection, PDF parsing/validity, and regression tests.
- Test direct and sales-order allocation orders, including multiple source references.
- Test missing optional fields, long names/notes, Unicode, zero/one/many lines, and multi-page output.
- Open representative output in desktop and mobile PDF viewers.

## Testing Plan

### Mapper and renderer tests

- Maps number, supplier, PO date, status, terms, expected delivery, notes, source references, lines, and persisted totals.
- Uses supplier unit cost and purchase-order line total, not quotation customer prices.
- Does not expose unrelated quotation/customer profitability fields.
- Direct orders display the agreed direct-purchase source label.
- Multi-source allocation orders preserve all distinct source references.
- Optional dates, notes, supplier fields, and source references do not break rendering.
- Long descriptions, supplier SKUs, notes, and Unicode values wrap without overflow.
- Renderer returns PDF bytes beginning with `%PDF-`.
- 30+ lines produce multiple pages and repeat the table header.

### Handler tests

- `GET /purchase-orders/3/pdf` returns `200` and a PDF content type.
- Response has attachment disposition using a safe `PO-...pdf` filename.
- `X-Content-Type-Options: nosniff` is present.
- Invalid or missing order IDs return `404`.
- Renderer failures return `500` without partial PDF output.
- The purchase-order detail page contains the download action.
- The route works for both direct and sales-order allocation purchase orders.

### Integration/document checks

- Validate generated PDF bytes with the available PDF parser or `pdfcpu validate` if adopted for verification.
- Verify A4 dimensions, embedded/configured fonts, readable PHP amounts, no clipped totals, and no overlapping wrapped text.
- Verify `OPEN`, `CONFIRMED`, and `CANCELLED` documents according to the approved status contract.
- Verify the existing quotation and invoice PDF routes remain unchanged.

## Acceptance Criteria

- Purchase-order detail provides a working `Download PDF` action.
- `GET /purchase-orders/{id}/pdf` returns a valid downloadable PDF for direct and allocation orders.
- The PDF is generated by a separate purchase-order renderer and document projection.
- The PDF contains the persisted purchase-order number, supplier/procurement metadata, line snapshots, and correct persisted subtotal/total.
- Source references are accurate for direct, single-source, and multi-source orders.
- No quotation customer pricing, commission, margin, or profitability data appears.
- Long text and multi-page orders render without clipping, and table headers repeat.
- The existing distroless single-Go-binary deployment remains viable.
- Tests cover mapping, endpoint behavior, safe filenames, renderer failure, direct/allocation modes, and representative PDF output.

## Non-Goals

- Do not refactor quotation PDF rendering as part of this feature.
- Do not create a shared business document renderer for quotation and purchase order.
- Do not add tax calculations or alter purchase-order persistence.
- Do not change purchase-order create/edit behavior.
- Do not generate PDFs in the browser or from the existing HTML detail page.
