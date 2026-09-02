# 052 Separate Quotation, Sales Order, and Invoice PDF Templates Plan

## Goal

Give Quotations, Sales Orders, and Invoices independent PDF templates so each document can be customized without changing the layout or data contract of the other two.

The three documents may continue to share branding and small drawing primitives, but they must not share a document-shaped renderer or be converted into a `quotations.Quotation` merely to produce a PDF.

## Current-State Findings

- `GET /quotations/{id}/pdf` is implemented in `internal/slices/quotations/handler.go`.
- `GET /sales-orders/{id}/pdf` is implemented in `internal/slices/salesorders/handler.go`, but it calls `quotations.NewDocumentPDFRenderer("SALES ORDER")` and passes the order's quotation snapshot.
- `GET /invoices/{id}/pdf` is implemented in `internal/slices/invoices/handler.go`, but it reconstructs a `quotations.Quotation` and calls the same renderer with title `INVOICE`.
- The actual PDF template is Go drawing code in `internal/slices/quotations/pdf.go`; there are no HTML PDF templates.
- The quotation renderer currently owns document-wide coordinates, metadata rules, pagination, seller configuration, and formatting helpers. A title string is the main customization switch.
- The Sales Order PDF can therefore inherit quotation assumptions and cannot safely represent order-only fields independently from quotation fields.
- The Invoice PDF can display invoice data, but its layout and semantics are still quotation-driven.
- Detail pages already expose all three download actions, so the primary UI work is limited to preserving those links and improving document-specific labels if needed.
- The worktree contains existing uncommitted changes. This plan does not assume those changes should be reverted.

## Desired Boundary

Use one renderer and one customer-facing view model per document:

```text
internal/slices/quotations/pdf.go       -> QuotationPDFDocument, QuotationPDFRenderer
internal/slices/salesorders/pdf.go      -> SalesOrderPDFDocument, SalesOrderPDFRenderer
internal/slices/invoices/pdf.go         -> InvoicePDFDocument, InvoicePDFRenderer
internal/shared/pdf/                    -> font, branding, color, text, table, and pagination primitives only
```

The shared package must not know about quotation, order, or invoice fields. It may provide:

- seller/branding configuration
- safe filename handling
- money/date display helpers
- text wrapping and font measurement
- primitive cells, rows, headers, footers, and page breaks
- PDF response header helpers

Each slice owns its document title, metadata, table columns, totals wording, optional sections, validation, and projection from its persisted aggregate.

## Document Contracts

### Quotation PDF

Customer-facing quotation document containing:

- quotation number, created date, validity date, and payment terms
- customer identity and billing/delivery information available in the quotation snapshot
- quoted product lines, customer price, tax treatment, and persisted line totals
- persisted subtotal, VAT, withholding only if approved, and total
- customer-facing notes/terms when available

Must exclude supplier cost, commission, estimated profit, margin, and other internal profitability fields.

### Sales Order PDF

Final-order document containing:

- Sales Order number and order date
- quotation reference, or explicit `Repeat order` source
- customer PO number as a distinct field
- sales person and payment terms
- Bill To and Ship To blocks from the Sales Order snapshot
- only Sales Order lines, with quantity, UOM, rate, tax treatment, and amount
- persisted Sales Order totals

The handler must obtain an order-specific document projection. It must not use the quotation as a substitute source, especially after order edits or partial quotation allocation.

### Invoice PDF

Issued customer invoice containing:

- invoice number, invoice date, and due date
- Sales Order reference and customer PO number
- customer identity, contact, billing, and delivery snapshots
- sales person and payment terms
- immutable invoice lines and persisted invoice totals
- invoice status only if approved for customer display

The invoice renderer must use the immutable invoice snapshot returned by `invoices.Repository.FindByID`. It must not query or reconstruct data from the Sales Order or quotation.

## Template Strategy

Keep the existing visual language as a starting point, but remove the title-based branching from `quotations/pdf.go`.

Each renderer should have a small document-specific layout configuration, for example:

```go
type QuotationPDFRenderer struct { Brand Brand; Layout QuotationLayout }
type SalesOrderPDFRenderer struct { Brand Brand; Layout SalesOrderLayout }
type InvoicePDFRenderer struct { Brand Brand; Layout InvoiceLayout }
```

The layouts may initially look similar, but their coordinates and sections must be independently adjustable. This allows a future invoice tax/due-date block or Sales Order shipping block without adding conditionals to a quotation renderer.

Prefer A4 portrait initially to preserve the current verified contract. If the supplied quotation screenshot requires landscape, make that a quotation-only layout decision rather than changing all documents.

## Implementation Plan

### Phase 1: Freeze contracts and layout decisions

1. Confirm the required fields and customer-facing wording for each document.
2. Decide whether the screenshot-style layout applies only to quotations or to all documents.
3. Confirm billing versus delivery address behavior for each document.
4. Confirm invoice display rules for due date, status, withholding tax, and tax-inclusive totals.
5. Confirm seller profile and logo source. Avoid hard-coded seller data in individual renderers.

### Phase 2: Extract only shared PDF infrastructure

1. Move generic drawing helpers from `internal/slices/quotations/pdf.go` into `internal/shared/pdf/`.
2. Move common seller configuration, font loading, branding colors, safe filename logic, and generic response headers there.
3. Keep document-specific structs and `draw*` functions in their owning slice.
4. Replace `asciiText`, which currently turns Unicode into `?`, with embedded-font-aware text handling. Verify peso and customer/product Unicode support.
5. Make font and logo availability deterministic for the distroless Docker image. The current runtime filesystem lookup is not sufficient unless assets are explicitly packaged or mounted.

### Phase 3: Implement three independent renderers

1. Convert quotation rendering to `QuotationPDFDocument` and `QuotationPDFRenderer`.
2. Add `SalesOrderPDFDocument` and `SalesOrderPDFRenderer` under `internal/slices/salesorders`.
3. Add `InvoicePDFDocument` and `InvoicePDFRenderer` under `internal/slices/invoices`.
4. Give each renderer its own metadata layout, table columns, totals block, footer wording, and pagination decisions.
5. Reuse only shared primitives; do not expose `DocumentTitle` as the customization mechanism.
6. Keep all totals sourced from persisted domain values. No renderer may recalculate money using floating point or current product/tax data.

### Phase 4: Integrate handlers and repository projections

1. Keep `GET /quotations/{id}/pdf`, but map the quotation aggregate to the quotation-only document model.
2. Update `salesorders.Handler.pdf` to map an order-specific snapshot and call `NewSalesOrderPDFRenderer()`.
3. Add any missing Sales Order repository projection needed to load order-owned lines and customer snapshots without quotation substitution.
4. Update `invoices.Handler.pdf` to map `Invoice` directly and call `NewInvoicePDFRenderer()`.
5. Preserve safe document-specific filenames: quotation number, Sales Order number, and invoice number.
6. Standardize `Content-Type`, attachment disposition, `X-Content-Type-Options: nosniff`, and generate into a buffer before writing.
7. Keep the existing detail-page download links. Add document-specific button text only if product wording requires it.

### Phase 5: Tests and visual verification

1. Add renderer mapping tests proving internal quotation fields cannot enter the quotation PDF model.
2. Add Sales Order tests proving only order lines and order metadata are rendered, including PO/reference separation and repeat-order source.
3. Add Invoice tests proving invoice output remains stable if the Sales Order, quotation, product, or customer records later change.
4. Add handler tests for status, content type, safe filename, missing records, renderer failures, and no partial output.
5. Add PDF validity/signature tests and page-count tests for long documents.
6. Test empty optional values, long names, long PO numbers, long notes, Unicode text, no-tax/VAT cases, and 1/10/50/200 lines.
7. Inspect representative quotation, Sales Order, and Invoice PDFs in desktop and mobile viewers for clipping, wrapping, totals placement, and document-specific fields.
8. Run the final Docker image and verify that embedded assets work in the non-root distroless runtime.

## Files Expected To Change

- `internal/slices/quotations/pdf.go`: reduce to quotation-specific projection/rendering or replace with a quotation renderer using shared primitives.
- `internal/slices/quotations/handler.go`: inject/use the quotation renderer and map the quotation document model.
- `internal/slices/quotations/handler_test.go`: quotation renderer and endpoint regression coverage.
- `internal/slices/salesorders/handler.go`: remove the quotation renderer call and map the Sales Order document model.
- `internal/slices/salesorders/repository.go`: add or strengthen an order PDF snapshot projection if current aggregate data is insufficient.
- `internal/slices/salesorders/pdf.go` and `internal/slices/salesorders/pdf_test.go`: new independent Sales Order template and tests.
- `internal/slices/invoices/handler.go`: remove quotation-shaped reconstruction and use the invoice renderer.
- `internal/slices/invoices/pdf.go` and `internal/slices/invoices/pdf_test.go`: new independent Invoice template and tests.
- `internal/shared/pdf/`: new generic PDF primitives, branding, assets, and response helpers.
- `go.mod`/`go.sum`: only if the selected PDF/font implementation requires dependency changes.
- `Dockerfile` and asset/license notices: if fonts or logos are embedded or packaged differently.

## Risks and Controls

- **Data leakage:** dedicated customer-facing models and mapping tests prevent internal costs and margins from being rendered.
- **Stale or incorrect Sales Order data:** use order-owned persisted snapshots and test partial/edited orders.
- **Invoice immutability break:** render only the invoice aggregate, never live upstream records.
- **Layout drift:** independent layout structs allow local changes; golden fixtures or visual review should cover each document.
- **Pagination regressions:** test long line descriptions and totals placement for each renderer.
- **Font/runtime failures:** embed or explicitly package fonts and test the actual Docker image.
- **Scope expansion:** defer a template editor or database-configurable templates. First establish three code-owned, independently customizable templates.

## Acceptance Criteria

- Quotation, Sales Order, and Invoice downloads each call a distinct renderer and document view model.
- Changing one document's layout or fields does not require title conditionals or changes to another document's renderer.
- Sales Order PDFs use Sales Order data and contain only Sales Order lines.
- Invoice PDFs use immutable invoice snapshots and contain invoice-specific metadata such as due date.
- Quotation PDFs remain customer-facing and exclude profitability data.
- All three endpoints return valid, downloadable PDFs with safe filenames and consistent security headers.
- Long, empty, Unicode, VAT, no-tax, and multi-page cases render without clipping.
- The application remains deployable as the existing single Go binary in the distroless runtime.

## Decisions Superseding Earlier Analysis

`analysis/041-shared-quotation-sales-order-pdf-decisions.md` recommended one shared document renderer with a title input. This request changes that decision: the renderer and document contract must now be separate for all three document types. Shared low-level PDF primitives remain acceptable, but shared document layouts and quotation-shaped adapters are not.
