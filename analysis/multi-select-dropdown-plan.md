# Reusable Multiple Selection Dropdown Implementation Plan

## 1. Goal

Create a reusable custom multiple-selection dropdown for server-rendered forms.

The component must allow users to add and remove values while preserving normal
HTML form submission. The first integration target is the receivables status
filter, which currently uses repeated status checkboxes.

The implementation must follow `analysis/vertical-slice-coding-standard.md`
and use Go `html/template`, HTMX, and Alpine.js only for local presentation
state.

## 2. Scope

### In scope

- Reusable server-rendered multiple-selection dropdown markup.
- Adding an available option to the selected value list.
- Removing a selected value without a page reload.
- Repeated hidden inputs for normal form submission.
- Server-rendered initial selections.
- Keyboard and screen-reader support.
- Receivables status-filter integration.
- Responsive styling that matches the existing application shell.
- Template, browser behavior, and HTTP coverage where applicable.

### Out of scope

- Server-side filtering or classification rule changes.
- Client-side calculation of receivable statuses.
- Remote option searching or asynchronous option loading.
- Creating new status values from the browser.
- Persistence of dropdown state outside the current form and URL.
- Replacing all native `<select multiple>` controls before a second use case
  confirms the shared component is stable.

## 3. Component Contract

The component should accept a server-rendered option list and selected values.

Recommended view-model shape:

```go
type MultiSelectOption struct {
    Value    string
    Label    string
    Disabled bool
}

type MultiSelectViewModel struct {
    ID          string
    Name        string
    Label       string
    Placeholder string
    Options     []MultiSelectOption
    Selected    []string
}
```

Contract rules:

- `Name` is the submitted form field name.
- Each selected value renders as one hidden input using `Name`.
- `Selected` values must be normalized and deduplicated by the owning slice
  before rendering.
- The browser must not invent values that were not in `Options`.
- Option values are submitted as strings and validated by the server.
- Labels are display text only and must be escaped by `html/template`.
- Empty selections submit no value, preserving existing filter semantics.

## 4. Template Structure

Add a shared presentation partial only after keeping its contract independent of
receivable-specific types:

```text
internal/web/templates/
├── layout.html
└── partials/
    └── multi-select.html
```

The partial should render:

- A labelled wrapper with a stable component ID.
- A button that opens and closes the option list.
- A selected-value region containing removable chips.
- A listbox or equivalent option list.
- Hidden inputs for selected values.
- An empty-selection placeholder.
- An empty-options message when no options are available.

Keep template expressions simple. The partial must not access repositories,
GORM models, or domain objects.

## 5. Ordered Subtasks

### A. Confirm the server-side value contract

1. Confirm repeated query/form values remain the public contract:

   ```text
   status=overdue&status=near_due
   ```

2. Reuse the existing status option values and labels.
3. Preserve the current valid-status normalization and filtering behavior.
4. Confirm empty status selection still means no status filter.
5. Confirm invalid values cannot broaden the server-side query.

### B. Define the reusable view model

1. Add the smallest shared view-model type needed by the template.
2. Keep the type in the shared web/template boundary, not in the receivables
   domain or repository package.
3. Define deterministic option ordering.
4. Ensure selected values are represented independently from display labels.
5. Add tests for duplicate and unknown selected values before rendering.

### C. Build the server-rendered partial

1. Render the trigger button with the current selection summary.
2. Render each selected value as a chip with a remove button.
3. Render each available option exactly once.
4. Mark selected options with an accessible selected state.
5. Render one hidden input for every selected value.
6. Keep the component usable as ordinary HTML when JavaScript is unavailable.
7. Keep the root element stable so HTMX can replace a containing fragment safely.

### D. Add Alpine presentation behavior

Use a small `x-data` scope for local component state only:

- Open or close the option list.
- Add an option to the local selected-value list.
- Remove an option from the local selected-value list.
- Close on Escape and outside click.
- Track the active option for keyboard navigation.
- Keep `aria-expanded` synchronized with the open state.

Rules:

- Alpine must not validate receivable statuses.
- Alpine must not calculate classification or query the database.
- Server-rendered values remain authoritative after an HTMX response.
- The form must retain a usable fallback when Alpine fails to load.
- Use `x-cloak` only for content that is intentionally hidden before Alpine
  initializes.

### E. Add accessibility behavior

1. Associate the visible label with the component.
2. Use a semantic button for the dropdown trigger.
3. Expose `aria-expanded` and `aria-controls` on the trigger.
4. Expose selected state for each option.
5. Give every remove button an accessible name containing the option label.
6. Support Enter and Space to open or select.
7. Support Arrow Up and Arrow Down to move through options.
8. Support Escape to close and return focus to the trigger.
9. Ensure focus rings are visible against the application background.
10. Do not rely on color alone to communicate selected values.

### F. Add component styling

1. Add component-scoped classes to `app.css`.
2. Match existing input, button, badge, border, and focus styles.
3. Keep the option list above surrounding table content with a predictable
   stacking context.
4. Prevent the list from extending beyond the viewport on narrow screens.
5. Allow selected chips to wrap without changing the form layout unexpectedly.
6. Support long labels without clipping their remove controls.
7. Ensure the component does not introduce horizontal scrolling.

### G. Integrate the receivables filter

1. Replace the status checkbox group in the receivables filter template.
2. Build the component view model from the existing `StatusOptions` and
   `Filters.Statuses` values.
3. Keep `name="status"` and repeated value submission unchanged.
4. Preserve filter values on normal GET requests.
5. Preserve filter values after HTMX result replacement.
6. Keep Search and Clear filters behavior unchanged.
7. Keep the result fragment independent from the dropdown implementation.

### H. Verify HTMX and progressive enhancement

1. Submit the filter normally with JavaScript disabled.
2. Submit the filter through HTMX with multiple values selected.
3. Confirm `hx-push-url` preserves repeated status query parameters.
4. Confirm a result refresh does not leave stale selected chips behind.
5. Confirm the component is reinitialized after any HTMX replacement that
   contains the filter form.
6. Confirm normal navigation does not replace the application shell.

### I. Add tests

Component/view-model tests:

- Empty selection renders a placeholder.
- One selected value renders one chip and one hidden input.
- Multiple selected values render in deterministic order.
- Duplicate selected values render once.
- Unknown selected values are rejected or omitted consistently.
- Disabled options cannot be selected.
- Labels and values are HTML escaped.

HTTP/template tests:

- Receivables filter renders all status options.
- Existing status query values render as selected.
- Normal GET submits repeated status values.
- HTMX GET preserves multiple status values and returns the results fragment.
- Clear filters removes all selected values.
- Invalid status values do not broaden the query.

Manual browser checks:

- Add and remove several options.
- Reopen the dropdown after removing a value.
- Use keyboard-only navigation.
- Close with Escape and outside click.
- Operate the form with JavaScript disabled.
- Verify the dropdown at desktop and mobile widths.
- Verify the list does not appear behind the receivables table.

## 6. Acceptance Scenarios

1. Given no selected statuses, when the filter opens, then the placeholder is
   visible and no status input is submitted.
2. Given `Overdue` is selected, when the user selects `Near due`, then both
   values appear as removable chips and both hidden inputs are submitted.
3. Given `Overdue` is selected, when the user removes it, then no `overdue`
   input remains in the form.
4. Given a status is already selected, when the option list opens, then it is
   visibly marked selected and cannot be added twice.
5. Given the page is loaded with repeated `status` query parameters, then the
   same values are selected in the dropdown.
6. Given JavaScript is unavailable, then the server-rendered form still
   submits and filters using the existing status contract.
7. Given an HTMX filter request completes, then the selected status state
   matches the URL and returned server state.
8. Given a keyboard user opens the dropdown, then options can be selected,
   removed, and dismissed without a pointer.
9. Given a narrow viewport, then the dropdown remains within the viewport and
   selected chips remain usable.

## 7. Definition Of Done

- A shared multiple-selection dropdown partial and view-model contract exist.
- Values can be added and removed without duplicate submissions.
- Repeated hidden inputs preserve the existing server-side form contract.
- The receivables status filter uses the component without domain changes.
- Server-rendered state remains authoritative after HTMX requests.
- Alpine.js is limited to local presentation state.
- Normal form behavior works without JavaScript.
- Keyboard and screen-reader behavior is covered and manually verified.
- Empty, selected, disabled, and error-adjacent states are styled.
- Desktop and mobile layouts are usable.
- Template and HTTP tests pass.
- `go test ./...`, `go vet ./...`, and formatting pass.
- `analysis/progress.md` is updated.

## 8. Implementation Decision Gate

Before implementation, confirm:

- The component should submit repeated values rather than a comma-separated
  value.
- The first supported use case is the receivables status filter.
- The option list is server-rendered and finite.
- Creating arbitrary new options is not required.
- Alpine.js remains the local interaction mechanism.
