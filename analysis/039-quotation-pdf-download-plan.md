# 039 Quotation PDF Download Plan

## Goal

Allow an authorized user to download a created quotation as a professional PDF from the quotation detail page.

The PDF should be generated on the server from the persisted quotation snapshot, not from browser HTML or form values. This keeps the document consistent with the stored quotation and makes the download work on desktop and mobile browsers.

## Current-State Findings

- The application is Go-first and uses `net/http`, `html/template`, embedded templates, GORM, and SQL Server.
- Quotations already have all core document data: number, customer, creation date, validity date, terms, tax rule, notes, line products, quantities, UOMs, prices, line totals, tax, and total.
- `GET /quotations/{id}` loads the complete quotation through `Repository.FindByID`.
- The quotation detail template already has the natural location for a `Download PDF` action.
- The current Docker runtime is `gcr.io/distroless/static-debian12:nonroot`, so it deliberately contains no shell, Node.js, Chromium, or system font packages.
- Money is stored as scaled integer values and formatted by the shared money package. PDF generation must reuse those values and formatters; it must not recalculate totals with floating point values.
- The current detail page displays supplier cost, commission, and profitability. Those are internal fields and must not be placed in a customer-facing quotation PDF by accident.

## Recommendation

Use **`github.com/signintech/gopdf`** as the first implementation choice, wrapped behind a small local `quotationpdf` renderer interface.

Why this is the best fit for this repository:

- Native Go: no second service, Node installation, browser download, or external process.
- MIT license and no paid runtime or per-document fee.
- Produces directly to an `io.Writer`/file path and supports embedded TrueType fonts, Unicode, images, tables, headers, footers, and page management.
- Has a large public adoption signal: approximately 2.9k GitHub stars, 306 forks, and 816 commits at research time.
- Works with the existing small document scope without introducing a new rendering language.
- Preserves the current static distroless deployment model.

The generator should be isolated so that switching to Maroto or another renderer later does not affect the handler, repository, domain calculations, or PDF endpoint contract.

This is not a claim that any library works “perfectly”. PDF correctness depends on font selection, pagination, escaping, totals, and viewer testing. The implementation plan explicitly verifies those concerns.

## Library Comparison

Research snapshot: 17 August 2026. GitHub star/fork counts are adoption indicators, not proof of production quality.

### Go Libraries

| Library | License / cost | Adoption and maintenance signal | Fit for quotation PDF | Maintainability | Main risks | Decision |
|---|---|---|---|---|---|---|
| `github.com/signintech/gopdf` | MIT, free | About 2.9k stars, 306 forks, 816 commits; public tests and examples | Strong. Has text, Unicode fonts, images, tables, headers/footers, links, and protection | Moderate. Low-level coordinate API, but predictable and self-contained | Layout is manual; table pagination and wrapping need local code/tests | **Recommended** |
| `github.com/johnfercher/maroto/v2` | MIT, free | About 2.7k stars, 257 forks, CI, docs, examples, active v2 releases | Very strong document-layout API with rows, columns, components, tables, and automatic page flow | High at the document layer | v2 currently declares Go 1.26.1 and depends on `phpdave11/gofpdf`; version compatibility and transitive maintenance need validation against this app | Consider after a compatibility spike |
| `github.com/go-pdf/fpdf` | MIT, free | About 643 stars, 59 forks | Feature-rich low-level generator with fonts, images, tables, and UTF-8 support | Low for a new dependency | Repository was archived on 4 March 2025; explicitly directs users to Codeberg | **Reject for new work** |
| `github.com/phpdave11/gofpdf` | MIT, free | Historically widespread and the base of several wrappers | Capable low-level generator | Low-to-moderate | Older API/ecosystem and maintenance uncertainty; Maroto still depends on it | Use only transitively or after review |
| `github.com/pdfcpu/pdfcpu` | Apache-2.0, free | About 8.8k stars, 624 forks, 1,121 commits, active tests and security policy | Excellent for validate, optimize, merge, encrypt, sign, stamp, and inspect PDFs; not the simplest document authoring API | High for PDF processing | Not primarily a quotation layout library; using it alone makes layout unnecessarily difficult | Optional post-generation validation/optimization |
| UniPDF / `unidoc` | AGPL or commercial dual licensing | Mature commercial-grade ecosystem | Technically strong for complex PDF features | High | AGPL obligations or paid commercial license conflict with the stated no-paid-library preference and require legal review | **Reject for this scope** |

### JavaScript Libraries and Rendering Approaches

| Library / approach | License / cost | Adoption signal | Fit for quotation PDF | Maintainability | Main risks | Decision |
|---|---|---|---|---|---|---|
| `pdfkit` | MIT, free | About 10.7k stars, 1.2k forks, 903 commits | Strong low-level PDF generation, tables, fonts, images, metadata, and Node streaming | Moderate | Requires Node runtime and a separate JS build/service in this Go application; manual layout remains | Good JS option, not selected |
| `pdf-lib` | MIT, free | About 8.6k stars, 909 forks, 484 commits; tests for Node/browser/Deno/React Native | Strong for creating and modifying PDFs, forms, fonts, metadata, and page operations | High API quality | More drawing/layout work for a polished quotation; adding Node still splits the stack | Good for PDF modification, not selected |
| `jsPDF` | MIT, free | About 31.3k stars, 4.8k forks, 2,333 commits | Strong for client-side or Node-generated simple PDFs and broad plugin ecosystem | Moderate | Browser/Node layout and font handling require care; downloading from client can expose data and produce inconsistent output | Not selected |
| Puppeteer + Chromium `page.pdf()` | Apache-2.0, free | About 95.5k stars, 9.6k forks, 6,480 commits | Excellent visual fidelity when converting a purpose-built HTML/CSS print template; easiest for complex styling | High document-template ergonomics, lower operations ergonomics | Large browser dependency, startup/memory cost, sandboxing, fonts, image/runtime management, and a second runtime | Best visual alternative, not best repository fit |
| Playwright + Chromium | Apache-2.0, free | Large, active browser automation ecosystem | Similar HTML/CSS print quality and browser coverage | High | Same browser/runtime/container complexity as Puppeteer | Not selected |

## Why Not Generate from the Existing Detail HTML?

Using a browser to print the current detail page is unsafe for this document because the page includes internal profitability data. It also couples a customer document to dashboard styling and browser rendering behavior.

If a browser renderer is selected later, create a separate customer-facing print template with an explicit view model. Never pass the full `Quotation` object to a customer PDF template.

## Proposed Design

### Package Boundary

Add a package such as:

```text
internal/slices/quotations/pdf.go
internal/slices/quotations/pdf_test.go
internal/slices/quotations/templates/detail.html  # download action only
```

Keep the renderer behind a narrow function or interface:

```go
type PDFRenderer interface {
    RenderQuotation(io.Writer, CustomerQuotationDocument) error
}
```

The document view model should contain only customer-visible fields:

- quotation number
- customer name and, when available, customer address/contact data
- created date and validity date
- payment terms and delivery terms
- line number, SKU, product name, quantity, UOM, unit price, tax label, and line amount
- subtotal, tax, withholding tax only if the business explicitly wants it shown, and total
- notes and customer-facing terms
- company name/logo and PDF metadata

Do not include supplier cost, commission rate/amount, estimated profit, or internal margin in this model.

### HTTP Flow

1. Add `GET /quotations/{id}/pdf` beside the existing quotation routes.
2. Parse and validate the numeric ID exactly as the existing `view` handler does.
3. Load the quotation using `FindByID`, ensuring the complete persisted snapshot is used.
4. Return `404` for a missing quotation and a generic `500` for generation/storage failures without exposing internal errors.
5. Set:
   - `Content-Type: application/pdf`
   - `Content-Disposition: attachment; filename="QT-00000001.pdf"`
   - `X-Content-Type-Options: nosniff`
6. Generate into a bounded `bytes.Buffer` before writing the response. This prevents a partial PDF response if generation fails and allows a size limit.
7. Use `http.ServeContent` only if a seekable reader and explicit modification timestamp are useful; direct buffer writing is sufficient for this endpoint.

### PDF Layout

Use A4 portrait with a stable margin and a professional one-page-first layout:

- branded header with company name/logo and `QUOTATION`
- quotation number, date, valid-until date, and customer block
- payment terms, delivery terms, and tax rule
- repeatable line-item table with wrapped product name/description
- subtotal, tax, withholding, and grand total block
- customer-facing notes and terms
- footer with page number and generation/document metadata

The renderer must support multiple pages. The line table header should repeat, and totals must remain together where possible. Long product names, notes, customer names, and Unicode text must wrap rather than overflow.

### Fonts and Assets

Embed a known TrueType font in the Go binary using `//go:embed`, for example a permitted Roboto or Noto Sans regular/bold pair. Do not depend on fonts installed in the container.

Before adding a font, confirm its license and add the required notice to `internal/web/static/THIRD-PARTY-NOTICES.md` or the project license inventory. Keep the logo optional so PDF generation still works if no logo asset is configured.

### Data and Calculation Rules

- Display values using the existing `money.Amount` formatting rules.
- Never recalculate totals in the PDF package.
- Use the persisted `Totals` and persisted line totals as the source of truth.
- If the PDF needs a display tax split, use persisted tax values and clearly label whether prices are tax-exclusive or tax-inclusive.
- Freeze the customer PDF's visible fields to a dedicated view model so future internal quotation fields cannot leak automatically.

## Implementation Plan

### Phase 1: Confirm document contract

- Decide whether the PDF is customer-facing only or whether an internal version is also required.
- Confirm company identity fields, logo, address, phone/email, currency, payment terms wording, delivery terms wording, and tax display convention.
- Confirm whether withholding tax appears on customer quotations. The repository stores it, but the current detail page does not display it.
- Confirm authorization rules if the application later gains users/tenants. The endpoint must use the same ownership checks as the detail page.

### Phase 2: Add the renderer

- Add `gopdf` at a pinned version.
- Add the font assets and notices.
- Build a customer-only document view model.
- Implement the header, customer metadata, line table, totals, notes, and footer.
- Add deterministic filename sanitization based on the persisted quotation number.

### Phase 3: Integrate the endpoint and UI

- Register `GET /quotations/{id}/pdf`.
- Add `Download PDF` to the detail action area.
- Use download semantics rather than opening a new tab by default.
- Preserve the existing HTML detail page for internal review.

### Phase 4: Verify and harden

- Unit-test document mapping, customer/internal field separation, filenames, missing data, long text, empty optional fields, and multi-line quotations.
- Handler-test status codes and PDF response headers.
- Validate generated bytes with `pdfcpu validate` in an integration test or CI check if adding pdfcpu as a development/verification dependency is acceptable.
- Open representative PDFs in Chrome, Firefox, Adobe Acrobat Reader, and a mobile PDF viewer.
- Test 1, 10, 50, and 200 line items, long product names, long notes, Unicode customer/product names, VAT/no-tax cases, and very large amounts.
- Benchmark concurrent downloads and enforce a reasonable maximum response size or line count.
- Test the final Docker image, confirming the endpoint works in the distroless non-root runtime.

## Acceptance Criteria

- A created quotation detail page provides a working `Download PDF` action.
- The endpoint returns a valid PDF with the quotation number in the filename.
- The PDF contains the persisted customer-facing quotation values and correct totals.
- No supplier cost, commission, or profitability value appears in the customer PDF.
- The PDF renders correctly with zero optional notes, long text, Unicode text, and multiple pages.
- The line-item header repeats on later pages and totals are not clipped.
- The service remains a single Go binary and the existing distroless Docker image remains viable.
- Tests cover mapping, HTTP behavior, PDF validity, and representative visual/document cases.

## Final Decision Matrix

Scoring: 1 = poor, 5 = excellent for this repository. “Adoption” is based on public project signals, not a guarantee.

| Option | Adoption | Result quality | Maintainability here | Deployment simplicity | Cost/licensing | Total |
|---|---:|---:|---:|---:|---:|---:|
| `gopdf` | 4 | 4 | 4 | 5 | 5 | **22/25** |
| Maroto v2 | 4 | 5 | 5 | 4 | 5 | 23/25, subject to Go-version spike |
| `pdfkit` | 4 | 4 | 3 | 2 | 5 | 18/25 |
| `pdf-lib` | 4 | 4 | 3 | 2 | 5 | 18/25 |
| `jsPDF` | 5 | 3 | 3 | 2 | 5 | 18/25 |
| Puppeteer | 5 | 5 | 3 | 1 | 5 | 19/25 |

Maroto has the highest document authoring score, but its current Go-version declaration and dependency chain must be proven compatible before adoption. `gopdf` is the lower-risk first implementation because it works with the current Go-native, distroless architecture and provides all required features. A short Maroto spike can be performed before coding if automatic pagination is considered more important than keeping the dependency surface minimal.

## Sources

- [gopdf](https://github.com/signintech/gopdf): MIT license, features, examples, adoption signals.
- [Maroto](https://github.com/johnfercher/maroto): MIT license, v2 layout model, docs, CI, and adoption signals.
- [pdfcpu](https://github.com/pdfcpu/pdfcpu): Apache-2.0 license and PDF validation/processing scope.
- [go-pdf/fpdf](https://github.com/go-pdf/fpdf): archived status and migration notice.
- [PDFKit](https://github.com/foliojs/pdfkit): MIT license and Node/browser generation features.
- [pdf-lib](https://github.com/Hopding/pdf-lib): MIT license and create/modify/font/metadata features.
- [jsPDF](https://github.com/parallax/jsPDF): MIT license, Node/browser support, and font considerations.
- [Puppeteer](https://github.com/puppeteer/puppeteer): Apache-2.0 license and Chromium installation/runtime model.
