# Receivable Tax Computation Preview Plan

## 1. Current State

- `internal/slices/receivables/templates/partials/receivable-form.html` renders the gross amount input and tax-rule select for both create and edit.
- The form currently has no computation summary and does not react to amount or tax-rule changes.
- `internal/shared/tax.CalculateRule` is the authoritative calculation path. It returns gross amount, VAT-exclusive base, VAT, withholding/EWT, and net amount.
- `NewDeliveryReceivableWithTax` recalculates the same breakdown during create and update validation, so the preview must never become the persistence source of truth.
- Existing detail and payment pages already render tax breakdown values after a receivable has been saved.
- HTMX is already used for form fragments and GET refreshes; Alpine is presentation-only and should not own financial calculations.

## 2. Goal

On both new and edit receivable forms, show a detailed computation preview when all of these are true:

1. The gross amount input contains a valid non-negative amount.
2. A tax-rule value is selected.
3. The user types or changes the gross amount, or changes the tax rule.

The preview should update without submitting or saving the receivable and should include:

- Gross amount
- VAT-exclusive tax base
- VAT
- EWT/withholding
- Net payable

When no tax rule is selected, the detailed tax preview should be hidden. The existing form and no-rule behavior remain unchanged.

## 3. Recommended Interaction

Add a server-rendered HTMX preview fragment instead of implementing tax arithmetic in browser JavaScript.

### Form behavior

- Keep the amount label as `Gross amount (PHP)`.
- Add a stable preview target immediately after the tax-rule field, such as `#receivable-tax-preview`.
- Configure the amount input to request a preview on debounced input/change events.
- Configure the tax-rule select to request a preview on change.
- Include only `amount` and `tax_rule_code` in the preview request, or include the form and let the endpoint read those two fields.
- Use a short debounce for typing so the server is not called for every keystroke.
- Preserve normal HTML form submission for users without JavaScript/HTMX.

### Preview states

- Empty amount: render no detailed computation, or a quiet prompt to enter a gross amount.
- Invalid amount: render no monetary result and leave the existing form validation to the POST path.
- No tax rule: render an empty preview so stale tax values disappear immediately.
- Valid amount plus selected rule: render the full breakdown with currency formatting from `money.Amount.FormatPHP()`.
- Unknown rule: render a safe error state; do not calculate based on browser-supplied rates.

## 4. Server Changes

### A. Preview view model

Extend `ReceivableFormViewModel` or add a focused preview view model containing:

- `Visible` or equivalent state
- `GrossAmountDisplay`
- `TaxBaseDisplay`
- `VATDisplay`
- `EWTDisplay`
- `NetPayableDisplay`
- Optional preview error/message

Prefer reusing the existing receivable display naming where practical, while keeping the preview independent from a persisted `DeliveryReceivable`.

### B. Preview endpoint

Add a GET route in `internal/slices/receivables/handler.go`, for example:

```text
GET /receivables/tax-preview
```

The handler should:

1. Read `amount` and `tax_rule_code`.
2. Return an empty preview when no rule is selected.
3. Parse the amount with `money.Parse`.
4. Call `tax.CalculateRule` with the posted `tax.RuleCode`.
5. Render only the preview fragment for HTMX.
6. Return a safe validation response for malformed input or an unknown rule.

Do not call create/update services, write to the database, create audit events, or consume idempotency keys from this endpoint.

### C. Shared calculation boundary

Use `tax.CalculateRule` for preview values. Do not duplicate the VAT-inclusive formula in:

- Handler code
- Templates
- `app.js`
- Inline JavaScript
- SQL

The create/update path remains authoritative and continues to use `NewDeliveryReceivableWithTax`.

## 5. Template And Styling Changes

### Templates

- Add a `receivable-tax-preview` partial under `internal/slices/receivables/templates/partials/`.
- Render a clear breakdown, for example:

```text
Tax computation
Gross amount          PHP 2,800.00
VAT-exclusive base    PHP 2,500.00
VAT                   PHP   300.00
EWT                   PHP    25.00
Net payable           PHP 2,775.00
```

- Add the target element to `receivable-form.html`.
- Ensure HTMX replacement preserves the form and its current field values.
- Do not show tax labels or zero values when no rule is selected.

### CSS

Add minimal styles in `internal/web/static/app.css` for a compact form computation card and responsive label/value rows. Reuse existing card, muted, and detail-grid visual language.

## 6. Tests

### Handler/template tests

- Valid amount and selected rule return a preview fragment.
- Preview contains gross, tax base, VAT, EWT, and net payable.
- No selected rule clears/hides the preview.
- Empty or malformed amount does not produce misleading monetary values.
- Unknown tax rule is rejected safely.
- Both create and edit forms contain the same preview target and triggers.
- Validation-error form rendering still preserves the posted amount and rule selection.

### Calculation coverage

Reuse existing shared tax tests and verify preview output with at least:

- `PHP 2,800.00`: gross `PHP 2,800.00`, tax base `PHP 2,500.00`, VAT `PHP 300.00`, EWT `PHP 25.00`, net payable `PHP 2,775.00`.
- `PHP 13,400.00`: net payable `PHP 13,280.36` and EWT `PHP 119.64`.
- A value with more than two decimal places to confirm existing precision is retained.

### Verification

Run:

```text
gofmt -w <changed Go files>
go test ./...
go vet ./...
```

Also manually verify create and edit flows in a browser with typing, backspacing to an invalid value, selecting/deselecting the rule, and changing an existing saved receivable.

## 7. Acceptance Criteria

1. A valid gross amount and selected tax rule show the detailed computation without form submission.
2. Changing the gross amount refreshes all displayed values.
3. Changing the tax rule refreshes or clears the computation.
4. No-rule state never leaves stale tax values visible.
5. Preview values match the values persisted after create/edit submission.
6. The browser cannot choose tax rates or override server calculations.
7. Existing no-rule create/edit, HTMX validation, full-page fallback, detail, list, and payment behavior remain intact.

## 8. Open Product Decision

Confirm whether the preview should show VAT and tax base in addition to gross, EWT, and net payable. The existing persisted/detail model supports all five values, so showing the complete breakdown is the recommended default.
