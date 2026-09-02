# Invoice Creation PDF New Tab Plan

## Goal

After a successful Invoice conversion, open the newly generated Invoice PDF in a new browser tab. The browser PDF viewer must allow the user to download or print the document.

## Current State

- Invoice creation currently redirects to `/invoices/{id}` after success.
- Invoice PDF is available at `/invoices/{id}/pdf`.
- The PDF endpoint returns `Content-Type: application/pdf`.
- The PDF endpoint currently uses `Content-Disposition: attachment`, which encourages download instead of inline browser display.
- The Invoice creation form already uses an Alpine confirmation modal and submits through the same form.
- The existing PDF tests expect an attachment disposition and must be updated for inline viewing.

## Recommended Flow

1. User opens an `OPEN` Sales Order.
2. User enters the Invoice number.
3. User reviews and confirms the Invoice/Receivable summary.
4. The confirmation submit changes the form target to `_blank`.
5. The server creates the Invoice and linked Receivable atomically.
6. On success, the server redirects to `/invoices/{id}/pdf`.
7. The new tab follows the redirect and displays the PDF in the browser PDF viewer.
8. The user can use the viewer's native Download and Print controls.

The original Sales Order page remains open in the original tab.

## HTTP Changes

### Successful creation response

Change the successful Invoice creation redirect in `internal/slices/invoices/handler.go` from:

```text
/invoices/{id}
```

to:

```text
/invoices/{id}/pdf
```

Continue using `303 See Other` so the browser performs a GET for the PDF after the POST.

### PDF response headers

Change the Invoice PDF response to:

```http
Content-Type: application/pdf
Content-Disposition: inline; filename="INV-....pdf"
X-Content-Type-Options: nosniff
```

`inline` is required for the browser to display the document in the new tab. The filename remains available when the user chooses Download.

Do not change quotation, Sales Order, or Purchase Order PDF behavior unless explicitly requested. This behavior is specific to the Invoice created by the conversion flow.

## Form and Browser Behavior

Update `internal/slices/invoices/templates/form.html` so only the confirmed submission opens a new tab.

Recommended Alpine behavior:

```text
on confirmation:
    form.target = "_blank"
    submit the form natively
```

The initial `Create invoice` action must continue opening the confirmation modal without opening a tab. The `Cancel` action must not change the form target or submit the form.

The final native submit is intentional. It allows the browser to apply the `_blank` target to the POST response and its redirect, avoiding popup-blocker issues caused by calling `window.open()` after an asynchronous request.

## Validation and Failure Behavior

- The server remains authoritative for all Invoice and Receivable validation.
- If Invoice number validation fails, no new tab should be opened by the normal browser flow; the form should render the validation error in the current page where possible.
- If a database or concurrency error occurs after confirmation, show the existing error response and do not treat the operation as successful.
- If the Invoice and Receivable transaction succeeds but PDF rendering fails, the new tab should show the PDF error response. The Invoice must not be recreated by refreshing the PDF tab.
- Refreshing the PDF tab must only perform a GET and must never create another Invoice or Receivable.
- Idempotency must continue protecting repeated confirmation submissions.

If setting `_blank` before submission causes validation errors to open in a new tab, use a small success-only response flow instead: submit the form in the current tab, return a response that sets the PDF URL, and open it from the original user gesture. The preferred first implementation is native `_blank` because it is simpler and is consistently supported by browsers for redirects.

## Route and Navigation Behavior

- Keep `/invoices/{id}` for Invoice detail navigation.
- Keep `/invoices/{id}/pdf` for direct PDF access and later reprinting.
- After successful creation, go directly to the PDF route in the new tab.
- The Invoice PDF should include the Invoice number, customer, Sales Order reference, PO number, terms, lines, totals, and tax values already used by the Invoice document.
- Invoice detail should continue linking to the Receivable for users who return to the application tab.

## Security and Reliability

- Continue loading the Invoice by ID on the PDF endpoint rather than accepting document values from the browser.
- Preserve the existing safe filename logic.
- Keep `nosniff` enabled.
- Do not expose the PDF as a data URI or embed untrusted HTML.
- Do not create a second PDF-specific conversion endpoint that can mutate data.

## Tests

### Handler tests

- Successful Invoice creation returns `303 See Other` with `Location: /invoices/{id}/pdf`.
- Successful creation still creates the Invoice/Receivable only once when the request is retried.
- Converted Sales Orders cannot submit another Invoice.
- Invoice PDF returns status `200` and `Content-Type: application/pdf`.
- Invoice PDF returns `Content-Disposition: inline` with a safe filename.
- Invoice PDF begins with `%PDF-`.
- PDF generation failure returns `500` and does not return an application PDF response.

### Template tests

- The initial create button opens the confirmation dialog.
- The confirmation action sets the form target to `_blank`.
- The form action remains the Invoice creation POST route.
- The modal Cancel action does not submit.
- The summary is still visible before confirmation.

### Browser acceptance tests

1. Open an `OPEN` Sales Order and choose `Create invoice`.
2. Enter an Invoice number and confirm the summary.
3. Verify a new tab opens at the Invoice PDF URL.
4. Verify the PDF is displayed by the browser viewer rather than immediately downloaded.
5. Use the viewer's Print action and verify the print dialog is available.
6. Use the viewer's Download action and verify the Invoice filename.
7. Return to the original tab and verify the Sales Order is `CONVERTED` with the Invoice link.
8. Verify the corresponding Receivable exists and is linked.

## Delivery Sequence

1. Change the successful Invoice redirect to the PDF route.
2. Change the Invoice PDF disposition from `attachment` to `inline`.
3. Update the confirmation submit behavior to use `_blank` only after confirmation.
4. Update Invoice handler and PDF tests.
5. Verify direct Invoice detail and PDF routes remain available.
6. Run `go test ./...`, `go vet ./...`, and browser/manual acceptance testing.

## Definition of Done

- A confirmed Invoice conversion opens the generated Invoice PDF in a new tab.
- The PDF displays inline in the browser.
- Browser Download and Print controls are available.
- The original application tab remains available.
- Validation, transaction atomicity, idempotency, and manual Receivable creation remain unchanged.
