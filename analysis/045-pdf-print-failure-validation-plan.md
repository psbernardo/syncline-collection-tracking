# 045 PDF Print Failure Validation Plan

## Goal

Identify whether the `Print Failed` message is caused by the HTTP download, a truncated or invalid PDF, PDF font/page construction, or browser-specific printing behavior.

The validation must use the same quotation PDF endpoint shown in the report and must preserve the downloaded artifact for every check.

## Current Validation Gap

- `internal/slices/quotations/handler.go:149-170` generates the PDF into a buffer and returns it as an attachment.
- `internal/slices/quotations/pdf.go:69-139` creates the document with `gopdf`.
- `internal/slices/quotations/handler_test.go:175-208` checks status, headers, and the `%PDF-` prefix only.
- `internal/slices/quotations/handler_test.go:210-243` counts page markers but does not parse the PDF structure.
- There is no automated check for xref integrity, page boxes, embedded fonts, incomplete content streams, or print compatibility.

## Validation Sequence

### Phase 1: Capture the Exact Download

Use a real quotation ID that reproduces the issue and save the raw response without opening it in the browser first.

Record:

- HTTP status.
- `Content-Type`.
- `Content-Disposition` filename.
- `X-Content-Type-Options`.
- Response byte length.
- First five bytes, which must be `%PDF-`.
- Final bytes, which should contain the PDF EOF marker.

Compare the saved file size with the browser-downloaded file size. If they differ, investigate the proxy, browser download, or interrupted response before changing the renderer.

Pass criteria:

- Status is `200`.
- Content type is exactly `application/pdf`.
- The response is not an HTML error page.
- The saved file opens consistently from disk.
- Server and browser file sizes match.

### Phase 2: Validate PDF Structure

Run a standards-aware PDF validator against the saved file, preferably `pdfcpu validate`. If that is unavailable, use Adobe Acrobat Preflight or another independent PDF parser.

Check for:

- Valid header and EOF marker.
- Valid cross-reference table or stream.
- Valid trailer and page tree.
- Valid object offsets.
- Valid content streams.
- No unreadable or orphaned page objects.

Pass criteria:

- The validator reports no structural errors.
- The PDF can be opened and saved again by an independent desktop viewer.
- The re-saved PDF prints successfully.

If validation fails, preserve the validator output and treat PDF generation as the primary defect. Do not begin browser troubleshooting until this phase passes.

### Phase 3: Inspect Document Features

Inspect the generated PDF using a PDF information tool and a text extractor.

Verify:

- Page count is correct.
- Page size is the intended A4 geometry.
- All pages have a valid `MediaBox`.
- Fonts are embedded and usable.
- Text extraction does not fail on any page.
- No unexpected encryption or print restriction is present.
- The file contains no internal profitability fields.

Pay particular attention to the font setup at `pdf.go:80-83`, where one font file is registered under four font family names. Confirm that each referenced font exists in the final PDF and that the document has no broken font references.

### Phase 4: Test Renderer Edge Cases

Generate representative PDFs from the renderer and validate each artifact:

- One short line.
- Eleven or more lines to force pagination.
- A description long enough to wrap across multiple lines.
- A description taller than the remaining page area.
- Zero-tax quotation.
- VAT-inclusive quotation.
- Missing validity date.
- Empty optional address/contact fields.
- Large monetary values.
- Unicode customer and product names.
- 50 and 200 line quotations.

For each case verify:

- PDF validator passes.
- Text extraction completes.
- No text or totals are clipped.
- Table headers appear on continuation pages.
- Totals and footer remain inside the page boundary.
- The document prints successfully.

The long-row case is important because `pdf.go:100-128` can move a row to a new page but does not split a row that is taller than the available page space.

### Phase 5: Browser and Viewer Matrix

Print the same downloaded file, not a regenerated copy, in:

- Chrome PDF viewer.
- Microsoft Edge PDF viewer.
- Firefox PDF viewer.
- Adobe Acrobat Reader.
- A representative mobile PDF viewer if mobile use is required.

Record for each viewer:

- Opens successfully.
- Shows the correct page count and orientation.
- System print dialog opens.
- Print preview renders the page.
- Physical or virtual PDF printer completes successfully.

Interpretation:

- Fails in every viewer: malformed PDF, broken font, or invalid page/content structure.
- Fails only in Chrome or Edge: browser PDF print compatibility issue; compare with a re-saved PDF.
- Original fails but re-saved PDF prints: generated PDF contains a compatibility defect that the desktop viewer repairs.
- Only one machine/printer fails: local browser, spooler, driver, or printer issue rather than application PDF generation.

### Phase 6: Add Automated Regression Coverage

Add tests after the failing layer is identified:

- Handler test for response status, headers, non-empty body, and `Content-Length` if added.
- Renderer test that parses the complete PDF with an independent parser.
- Page-size and page-count assertions from parsed PDF objects, not byte-string counting.
- Font embedding assertion.
- Multi-page and long-row rendering tests.
- Text extraction test proving customer-facing output excludes supplier cost, commission, profit, and margin.

Keep a small valid PDF fixture or generated artifact available for local reproduction, but do not commit customer data or environment-specific paths.

## Recommended Diagnostic Order

1. Capture the endpoint response and compare it with the browser file.
2. Run an independent PDF structural validator.
3. Inspect fonts, page boxes, encryption, and extracted text.
4. Test long-row and multi-page renderer cases.
5. Compare Chrome, Edge, Firefox, and Acrobat printing.
6. Add the regression test at the layer that failed.

## Acceptance Criteria

- The exact downloaded artifact passes an independent PDF validator.
- All generated pages have valid page boxes and embedded usable fonts.
- The file prints from Chrome, Firefox, and Acrobat for the representative quotation.
- Long descriptions and multi-page documents print without clipping or print errors.
- Automated tests detect malformed output, broken fonts, invalid pagination, and incomplete responses.
- The root cause is documented as transport, renderer, browser, or local printer infrastructure.
