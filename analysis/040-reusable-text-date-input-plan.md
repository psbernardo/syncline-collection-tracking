# 040 Reusable Text Date Input Plan

## Goal

Add a reusable date input that lets users type digits only while automatically inserting `/` after the month and day portions.

Example:

```text
User types: 08162026
Displayed:   08/16/2026
Submitted:   2026-08-16
```

The component should improve data entry without changing the application's canonical date storage or server-side validation rules.

## 1. Current implementation analysis

### Existing date controls

- `internal/slices/quotations/templates/form.html` has a custom `MM/DD/YYYY` text field paired with a hidden native date picker.
- `internal/web/static/app.js` contains quotation-specific date synchronization and validation in `quotationForm.initializeDatePicker`, `syncDatePicker`, and `formatDatePickerValue`.
- `internal/slices/purchaseorders/templates/form.html` uses native `type="date"` inputs for `po_date` and `expected_delivery_date`.
- `internal/slices/receivables/templates/partials/receivable-form.html` uses a native date input for `delivery_date`.
- `internal/slices/receivables/templates/partials/payment-form.html` uses a native date input for `payment_date`.
- Server handlers currently receive ISO values from native date inputs. Quotation parsing already accepts `YYYY-MM-DD`, while its display helper renders `MM/DD/YYYY`.

### Main risks

- Formatting only in the browser could be bypassed or broken by paste, deletion, cursor edits, or JavaScript being disabled.
- Submitting the display value directly would require inconsistent parsing across handlers.
- A shared component could accidentally alter existing date semantics, optional/required behavior, or validation-error rendering.
- Date validation must reject impossible dates such as `02/30/2026`, not merely enforce the shape.
- Careless caret handling can make typing in the middle of the value frustrating.

## 2. MVP decisions

- Use the fixed `MM/DD/YYYY` display format, matching the quotation UI and the requested example.
- Accept digits as the meaningful input; ignore non-digit characters except that existing separators may be preserved while editing.
- Automatically insert `/` after 2 and 4 digits.
- Limit the display value to 10 characters (`MM/DD/YYYY`) and use `inputmode="numeric"`.
- Submit a hidden canonical value in `YYYY-MM-DD` format using the field's original `name`.
- Give the visible text input a derived name or no name so only one value is posted. The exact implementation should preserve compatibility with existing handlers.
- Leave the hidden canonical value blank until the complete date is valid. This allows normal server-side required validation to handle incomplete input.
- Keep server-side parsing and calendar validation mandatory. Client-side formatting is a usability feature, not a validation boundary.
- Preserve the existing quotation calendar button only if it can be cleanly exposed as an optional component feature; the core reusable input must work without a picker.
- Keep the component dependency-free and compatible with the existing Alpine/HTMX/static JavaScript setup.

## 3. Reusable component design

### Shared template partial

Add a partial under `internal/web/templates/partials/`, for example `text-date-input.html`, with a view model containing:

```go
type TextDateInputViewModel struct {
    ID           string
    Name         string
    Label        string
    Value        string // canonical YYYY-MM-DD value or display MM/DD/YYYY value
    Placeholder  string
    Required     bool
    HelpID       string
    ErrorID      string
    Error        string
    Picker       bool
}
```

The partial should render:

- A wrapper with `x-data="textDateInput()"` and initialization from the canonical value.
- A visible text input with `type="text"`, `inputmode="numeric"`, `autocomplete="off"`, placeholder `MM/DD/YYYY`, and an accessible label.
- A hidden canonical input with the original field `name` and `YYYY-MM-DD` value.
- Optional `required`, `aria-describedby`, and error attributes based on the view model.
- An optional calendar button/native date picker only when the caller requests it.
- A field-level error element when an error is supplied.

The partial must HTML-escape all server-rendered values through the Go template system and preserve values after a failed form submission.

### Client-side behavior

Add a shared `window.textDateInput` Alpine component in `internal/web/static/app.js`:

- Initialize the visible value from the hidden ISO value or a supplied display value.
- On `input`, remove non-digits, cap at 8 digits, and format as `MM/DD/YYYY`.
- Keep the caret usable after automatic slash insertion and deletion.
- On paste, normalize pasted text in the same way as keyboard input.
- On `blur`, normalize the value and synchronize the hidden ISO field.
- Convert only a complete, calendar-valid value to ISO; otherwise clear the hidden value.
- Dispatch a normal `input`/`change` event after synchronization so existing form behavior remains compatible.
- If an optional native picker is present, convert its `YYYY-MM-DD` value to display format and update the hidden field.
- Do not rely on `Date.parse`; use explicit numeric checks and a UTC date comparison to avoid timezone-dependent behavior.

The formatter should have small pure helpers where practical, such as:

```text
formatDateDigits("08162026") -> "08/16/2026"
parseDisplayDate("08/16/2026") -> "2026-08-16"
```

These helpers should be easy to unit test independently from Alpine DOM behavior.

## 4. Server-side contract

- Keep the canonical posted field format as `YYYY-MM-DD` so existing domain and repository contracts remain unchanged.
- Add shared Go helpers only if multiple handlers currently duplicate display parsing or formatting. They should parse strictly and return a clear invalid-date error.
- Do not trust the hidden input as proof of validity; parse it server-side and verify the resulting calendar date.
- Preserve each field's existing required/optional semantics:
  - Quotation `validity_date`: optional unless the current domain requires it.
  - Purchase order `po_date`: required.
  - Purchase order `expected_delivery_date`: optional.
  - Receivable `delivery_date`: required.
  - Payment `payment_date`: required.
- Ensure validation failures re-render the user's typed display value rather than replacing it with an empty or stale model value.

## 5. Migration sequence

1. Add shared view-model and template partial under `internal/web/templates`.
2. Add and test date-formatting/parsing helpers in `internal/web/static/app.js`.
3. Replace the quotation-specific date markup with the shared partial and remove or narrow the quotation-only date-picker synchronization.
4. Replace purchase-order `po_date` and `expected_delivery_date` controls with the partial.
5. Replace receivable delivery and payment date controls with the partial.
6. Update each page's view model/handler rendering to pass canonical values, labels, IDs, and field errors.
7. Confirm all existing handlers still receive ISO values and no database schema or migration is needed.
8. Update the static asset version in `layout.html` if browser caching would otherwise retain the old JavaScript.

Do not change unrelated searchable-select or form behavior.

## 6. Accessibility and responsive behavior

- Associate every visible input with a unique label and stable `id`.
- Keep the visible input keyboard-editable and do not make the native picker the only input path.
- Use `aria-invalid="true"` when the field has a server-side error.
- Connect help and error text with `aria-describedby`.
- Use a clear placeholder and an explicit format hint, such as `MM/DD/YYYY`.
- Ensure the control inherits existing input sizing and remains usable on narrow screens.

## 7. Tests

### JavaScript/component tests

Add tests using the project's available frontend test approach, or document a small browser-level verification if no JavaScript test runner exists:

- Typing `08162026` produces `08/16/2026`.
- Typing one digit at a time inserts separators at the correct positions.
- Letters, spaces, and extra digits are ignored/truncated.
- Pasting `08/16/2026` produces the same normalized value.
- Backspace around separators does not create duplicate separators or corrupt the value.
- Incomplete input leaves the canonical hidden field blank.
- Invalid dates such as `02/30/2026` leave the canonical field blank.
- Valid dates produce `2026-08-16`.
- Existing ISO values initialize as the correct display value.

### Go/template/handler tests

- Shared partial renders the label, placeholder, IDs, required state, hidden canonical value, and error attributes.
- Quotation form still renders `MM/DD/YYYY` and submits the expected ISO date.
- Purchase-order create/edit forms preserve both date fields and their optional/required behavior.
- Receivable and payment forms preserve their date values after rendering and validation errors.
- Existing handler parsing tests continue to pass with the hidden ISO field.

### Manual acceptance

- Type only digits on desktop and mobile numeric keyboards.
- Paste a formatted date and an unformatted date.
- Edit the month, day, and year in the middle of the value.
- Submit incomplete and invalid dates and confirm useful server-side validation.
- Verify the control works without JavaScript fallback or, at minimum, fails safely with a visible format-compatible field.

## 8. Acceptance criteria

1. A user can type `08162026` and sees `08/16/2026` without typing `/`.
2. Only numeric date content is accepted and the input cannot exceed `MM/DD/YYYY`.
3. Valid complete dates submit as `YYYY-MM-DD` to existing handlers.
4. Invalid or incomplete dates are not submitted as valid canonical dates.
5. The component is reusable across quotation, purchase-order, receivable, and payment forms.
6. Existing optional/required behavior and server-side date validation remain unchanged.
7. Existing quotation calendar selection continues to work if retained.
8. Labels, errors, keyboard editing, mobile input, and validation-error re-rendering remain accessible and usable.
