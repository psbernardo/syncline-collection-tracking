# Acknowledge Payment Page Design Plan

## 1. Goal

Redesign the full-payment acknowledgment experience as a professional, focused workflow rather than an inline form appended to the receivable detail page.

The page must make three things immediately clear:

- Which receivable is being acknowledged.
- The exact full amount that will be marked paid.
- The date on which payment was received.

The design must preserve the existing full-payment-only rules, server-side authority, audit trail, idempotency, optimistic concurrency, normal HTML fallback, HTMX support, and responsive behavior.

## 2. Current UI Assessment

The current implementation in `internal/slices/receivables/templates/partials/payment-form.html` is a small card below the receivable detail grid.

### Strengths

- Uses a visible label for payment date.
- Displays the amount.
- Explains that partial payments are unsupported.
- Uses normal POST and HTMX attributes.
- Keeps the payment action separate from the general receivable edit form.

### Gaps

- The acknowledgment action does not have its own page hierarchy or focused title.
- Company, invoice, PO, delivery date, due date, and current status are not presented together in the payment context.
- The amount is rendered as a small badge instead of the primary confirmation value.
- There is no explicit confirmation statement such as “I confirm this receivable was fully paid.”
- There is no clear visual distinction between reviewing payment details and submitting the state change.
- The action is placed after the detail grid, which makes it feel secondary and easy to miss.
- There is no dedicated mobile layout for the payment workflow.
- The submit state has no visible loading label or progress treatment.
- Validation is limited to an inline field error; there is no clear form-level error context.
- The current detail page creates the payment form inline, so navigation context and payment intent are mixed together.

## 3. Recommended Experience

Add a dedicated acknowledgment page:

```text
Receivables / Invoice 0127 / Acknowledge payment

Acknowledge full payment
Confirm that this receivable has been fully paid.

Receivable summary                 Payment confirmation
Company                            Amount received
Invoice                            ₱98,000.00
PO                                 Payment received date [date]
Delivery date                      [ ] I confirm full payment
Due date                           [Cancel] [Acknowledge payment]
Current status

This action moves the receivable to Payment Received and cannot be edited afterward.
```

Use a two-column layout on desktop and one stacked column on mobile:

- Left: receivable identity and financial summary.
- Right: payment confirmation form.

The amount should be the strongest visual element in the form card. The status transition and protected-state warning should be visible before the submit button.

Do not use a browser-native confirmation dialog as the primary confirmation. A dedicated page is more accessible, easier to understand, and works consistently without JavaScript.

## 4. Route and Navigation Changes

Add a GET route:

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/receivables/{id}/payment` | Render the full-payment acknowledgment page |
| `POST` | `/receivables/{id}/payment` | Record the full payment |

Update the receivable list and detail actions:

- Change `Acknowledge payment` links to `/receivables/{id}/payment`.
- Keep the detail page focused on receivable information.
- Remove the inline payment card from the detail page.
- Keep the payment date visible on paid detail and list views.

GET behavior:

- Return `404` when the receivable does not exist.
- Return `409` when the receivable is not active or has already been paid.
- Generate a fresh idempotency key.
- Render the current row version.
- Default the payment date to the current Philippines business date.

POST behavior remains unchanged from the existing payment command:

- Normal request: redirect to the receivable detail page with `303`.
- Successful HTMX request: return `HX-Redirect` to the detail page.
- Validation failure: return the payment form fragment for HTMX and the full acknowledgment page for normal requests.
- Stale, already-paid, or protected state: return a safe `409` response with feedback.

## 5. Page Structure

Add a page template such as:

```text
internal/slices/receivables/templates/payment.html
internal/slices/receivables/templates/partials/payment-page.html
internal/slices/receivables/templates/partials/payment-summary.html
internal/slices/receivables/templates/partials/payment-form.html
```

Recommended semantic structure:

```html
<main>
  <nav aria-label="Breadcrumb">...</nav>
  <header class="payment-page-heading">...</header>
  <div class="payment-layout">
    <section aria-labelledby="receivable-summary-heading">...</section>
    <section aria-labelledby="payment-confirmation-heading">...</section>
  </div>
</main>
```

### Page heading

- Eyebrow: `Collections`.
- Heading: `Acknowledge full payment`.
- Supporting text: `Confirm the payment received for this delivery receivable.`.
- Breadcrumb/back link: `Back to receivable`.
- Avoid competing `Edit` actions on this page.

### Receivable summary card

Show the information needed to prevent paying the wrong record:

- Company name as the primary identity.
- Invoice number.
- PO number.
- Delivery date.
- Due date.
- Current classification/status badge.
- Full amount due, formatted in PHP.

Use a clear summary layout rather than a dense generic detail grid. The invoice and PO should remain selectable/copyable text, and the amount should be right-aligned on wider screens.

### Payment confirmation card

Show:

- Section title: `Payment confirmation`.
- Large full amount: `Amount to mark as paid`.
- Required `Payment received date` field.
- Short date guidance: `Use the date the full payment was received.`.
- Full-payment-only notice.
- Protected-state warning: `After acknowledgment, this receivable will be classified as Payment Received and normal financial editing will be unavailable.`.
- Primary action: `Acknowledge payment`.
- Secondary action: `Cancel` or `Back to receivable`.

Use a visible confirmation checkbox only if product testing shows the warning is insufficient. The first implementation should avoid adding a checkbox that duplicates the submit intent; the dedicated page and prominent warning provide the confirmation boundary.

## 6. Visual Design Direction

Follow the existing application visual language while giving the action more hierarchy:

- White cards on the existing light gray page background.
- Indigo for navigation and primary controls.
- Green for the payment-received status and successful state.
- Amber or slate notice for the irreversible/protected-state message, not red because this is an intentional valid operation.
- Large amount typography with strong contrast.
- Consistent existing `eyebrow`, `status-badge`, `button`, `card`, and form tokens.
- Use a subtle green accent on the confirmation card, such as a left border or top rule, without turning the page into a decorative success screen.

Add scoped classes instead of changing generic card behavior:

- `.payment-page-heading`
- `.payment-layout`
- `.payment-summary-card`
- `.payment-amount`
- `.payment-confirmation-card`
- `.payment-notice`
- `.payment-actions`
- `.payment-date-help`

Do not introduce a new CSS framework or external assets for this page.

## 7. Responsive Behavior

Desktop:

- Two columns, approximately 5/7 or 6/6 based on available width.
- Confirmation card remains visually prominent.
- Actions align to the bottom/right of the form card.

Mobile:

- Stack summary before confirmation form.
- Keep the amount prominent and avoid horizontal overflow.
- Make the primary and secondary actions full-width or evenly sized.
- Keep the breadcrumb/back link easy to reach.
- Preserve visible labels and error messages.
- Do not rely on hover states.

Accessibility:

- One page-level `h1`.
- Logical heading order.
- Visible form label.
- `aria-describedby` from the date input to date guidance and validation text.
- Use `aria-live="polite"` for returned validation feedback.
- Keep status meaning in text, not color alone.
- Ensure keyboard focus remains usable after HTMX form replacement.
- Respect reduced-motion preferences already defined in `app.css`.

## 8. View Models and Handler Changes

Add a dedicated page model, for example:

```go
type paymentPage struct {
    Title       string
    ActiveNav   string
    Receivable  ReceivableViewModel
    PaymentForm PaymentFormViewModel
}
```

Extend `PaymentFormViewModel` only with presentation data required by the new page:

- `AmountLabel` or reuse `AmountDisplay`.
- `DateHelp` if the wording is not static.
- `FormError` for a form-level error when needed.

Do not duplicate authoritative payment rules in the view model. `CanReceivePayment`, date defaults, classification, and row version remain server-derived.

The existing payment command, repository update, audit event, and idempotency behavior should not change as part of the visual redesign.

## 9. HTMX Fragment Strategy

Use stable replacement roots:

- `#payment-page` for the full page shell if a full page render is needed.
- `#payment-form` for validation failures.

Recommended behavior:

- GET always renders the full page.
- HTMX POST validation returns only the payment form card or form root, preserving the summary card.
- Successful HTMX POST redirects to the receivable detail page.
- The form submit button is disabled during the request.
- Add visible button text such as `Saving payment...` only if it can be implemented without client-side business state; otherwise use the existing disabled behavior and a CSS/HTMX indicator.

## 10. Tests

### Handler tests

- GET payment page renders company, invoice, PO, amount, status, date input, and payment action.
- GET paid receivable returns `409`.
- GET cancelled or archived receivable returns `409`.
- GET missing receivable returns `404`.
- GET generates an idempotency key and row version.
- Invalid normal POST rerenders the full payment page with the date error.
- Invalid HTMX POST returns the payment form fragment with the date error.
- Successful normal POST redirects to detail.
- Successful HTMX POST returns `HX-Redirect`.
- List and detail links point to the dedicated payment page.

### Template/accessibility checks

- Exactly one page-level `h1` exists.
- Payment amount is visible with PHP formatting.
- Payment date has a visible label.
- Warning text is present and not color-only.
- Cancel/back navigation is present.
- The form remains usable without JavaScript.

### Browser verification

- Desktop layout at wide and medium widths.
- Mobile layout at narrow width.
- Keyboard-only navigation through breadcrumb, date input, and actions.
- Validation error after HTMX replacement.
- Successful acknowledgment returns to detail with `Payment Received` and the received date.

## 11. Implementation Order

1. Add the dedicated GET route and page handler.
2. Add payment page and summary templates.
3. Refactor the existing payment form partial into the new confirmation card.
4. Change list/detail acknowledgment links to the dedicated page.
5. Remove the inline payment form from receivable detail.
6. Add scoped responsive CSS and notice/amount styles.
7. Improve normal and HTMX validation rendering.
8. Add handler, template, and accessibility tests.
9. Run `gofmt`, `go test ./...`, and `go vet ./...`.
10. Perform desktop, mobile, and keyboard browser verification.

## 12. Definition Of Done

- A user reaches a dedicated, focused payment acknowledgment page.
- The page clearly identifies the receivable and full amount.
- The payment date input and protected-state warning are prominent.
- The design is professional, consistent with the existing application, and responsive.
- List and detail actions navigate to the dedicated page.
- The detail page no longer contains an oversized or competing inline payment form.
- Normal and HTMX validation flows remain functional.
- Existing full-payment command behavior, audit, idempotency, and concurrency protection remain unchanged.
- No partial-payment behavior is introduced.
- Automated and browser verification pass.
