# 038 Supplier-Product Searchable Dropdown Plan

## Epic E-02 follow-up: User-friendly relationship selection

This plan replaces the raw numeric ID inputs in the supplier-product form with searchable, human-readable single-select controls. It builds on the implementation in `031-trading-master-data-plan.md` and does not change the supplier-product database relationship.

## 1. Current implementation analysis

### Current flow

- `internal/slices/suppliers/templates/product-form.html` renders numeric inputs for `supplier_id` and `product_id`.
- `internal/slices/suppliers/handler.go` parses both values with `strconv.ParseInt` and ignores parse errors.
- `CreateProductCommand` receives the numeric IDs and the repository verifies that both parent records are active.
- `ListProducts` joins suppliers and products and already returns the human-readable supplier name, product SKU, and product name needed for display.
- The edit form loads the relationship by scanning all supplier products and displays the same raw IDs.
- `UpdateProduct` only updates supplier SKU, reference cost, and active state. It does not update `supplier_id` or `product_id`, so the edit form currently suggests that the relationship pair is editable when it is not.
- The existing `multi-select` component supports keyboard interaction and no-JavaScript fallback, but it is designed for multiple values and has no search field. It should not be reused unchanged for this single relationship selector.

### Current risks

- A user must know internal database IDs to create a relationship.
- Invalid non-numeric input becomes `0` because parse errors are discarded, producing a generic active-parent failure instead of a useful field error.
- The form can submit an inactive or deleted selection if the page is stale; server-side validation remains mandatory.
- Rendering every option may become slow or unwieldy as the product and supplier masters grow.
- The relationship pair has a unique constraint, so allowing users to change the pair during edit would create confusing duplicate behavior and could alter the identity of a relationship referenced by later transactions.

## 2. MVP decision

Use a reusable **single searchable select** for supplier and product selection:

- The visible control searches human-readable labels.
- A hidden input posts the selected database ID.
- The label shows enough context to distinguish records:
  - Supplier: `Supplier name` and optional contact number.
  - Product: `SKU - Product name (UOM)`.
- Only active suppliers and products are offered for a new relationship.
- Search is case-insensitive and matches the beginning or contents of the displayed label.
- The control supports keyboard navigation, Enter selection, Escape close, and clear/reset behavior.
- A native select or text fallback remains available when JavaScript is disabled.

For the current owner-only MVP, load active options server-side and filter them locally with Alpine.js. This avoids adding a remote search API before the catalog size requires it. Add a server-side HTMX search endpoint only when the option set becomes materially large or page payloads become a measured problem.

## 3. Form behavior

### New relationship

1. The server loads active suppliers and active products.
2. The form renders empty searchable controls with labels and a clear placeholder, such as `Search supplier...` and `Search product...`.
3. Selecting an option updates the visible label and hidden `supplier_id` or `product_id` value.
4. Submitting without a selection returns a field-level error: `Select a supplier` or `Select a product`.
5. The server parses IDs strictly and verifies that the selected records are still active inside the create transaction.
6. Duplicate supplier/product pairs return a relationship-specific error next to the form rather than a generic 500 response.

### Edit relationship

The supplier and product pair is the relationship identity and should not be changed in place. On edit:

- Display supplier and product as read-only selected labels.
- Keep hidden IDs only for display/context if required, but do not present editable ID fields.
- Allow editing supplier SKU, current reference unit cost, and active state.
- If the owner needs a different supplier/product pair, create a new relationship and deactivate the old one.
- The server must use the relationship ID and row version as the update authority; it must not trust posted parent IDs to change identity.

## 4. UI component plan

### New reusable component

Add a single-select view model and partial alongside the existing multi-select component:

```go
type SearchableSelectOption struct {
    Value    string
    Label    string
    Search   string
    Disabled bool
}

type SearchableSelectViewModel struct {
    ID          string
    Name        string
    Label       string
    Placeholder string
    Options     []SearchableSelectOption
    Selected    string
}
```

The partial should:

- Render a visible search input and a hidden form input with the actual ID.
- Render a listbox with stable option IDs and `aria-activedescendant` behavior.
- Use Alpine state for open/closed, query, filtered options, and selected option.
- Close on outside click and Escape.
- Preserve the selected option when validation re-renders the form.
- Render a native `<select>` fallback or an accessible list of radio-like options inside `<noscript>`.
- Escape all server-provided labels through the Go template system.

Do not modify the existing multi-select behavior for this feature; it is already used by receivable filters and changing its contract would create unrelated regressions.

### Styling

Reuse the existing `.multi-select` visual language where appropriate, but add distinct single-select classes so the control has:

- A clear search affordance.
- A visible selected value.
- A scrollable option menu with an empty-search state.
- Focus and keyboard states consistent with existing controls.
- Mobile-friendly touch targets.

## 5. Data and handler changes

### Repository and service

- Add active supplier and active product option queries returning only the fields needed by the selector.
- Prefer dedicated methods such as `ListActiveOptions` rather than passing complete master entities to templates.
- Order suppliers by name and products by SKU/name for stable results.
- Keep server-side active-parent and foreign-key validation in `CreateProduct`; UI filtering is not a security boundary.
- Add a typed error for duplicate supplier-product pairs so the handler can render a useful form error.

### Handler

- Extend `productFormPage` with supplier and product selector view models.
- Build selector options on both GET and validation-error renders.
- Preserve selected values after validation failures.
- Replace ignored `strconv.ParseInt` errors with strict validation errors attached to `SupplierID` and `ProductID`.
- Do not load all supplier-product relationships to find one edit record; add `FindProduct` usage directly through the service/repository.
- Return a user-facing conflict response for stale row versions rather than an internal server error.
- On edit, render selected supplier/product labels from the loaded relationship and omit editable parent selectors.

### Service contract

- Keep `SupplierID` and `ProductID` required on create.
- Treat them as identity fields on the relationship and reject any attempted parent change on update, or remove them from the update command entirely.
- Keep current reference price validation through `internal/shared/money`.
- Preserve audit snapshots for price, SKU, and active-state changes.

## 6. Suggested routes and request shape

### MVP local-filter approach

No new HTTP search routes are required. The form page contains active options and Alpine filters them locally.

```text
GET  /supplier-products/new
POST /supplier-products
GET  /supplier-products/{id}/edit
POST /supplier-products/{id}
```

The submitted request still contains numeric IDs, but they are hidden implementation values generated by the selected option, not user-entered fields:

```text
supplier_id=12
product_id=34
supplier_sku=SUP-001
reference_cost=1250.00
```

### Deferred server-side search

If local option rendering becomes too large, add:

```text
GET /suppliers/options?q=acme
GET /products/options?q=paper
```

Those endpoints should return an HTML option partial for HTMX, limit results, require a minimum query length, and always enforce `is_active = 1`. Do not add these endpoints preemptively for the MVP.

## 7. Acceptance scenarios

1. Given the new relationship form, when the owner opens the supplier control, then names are shown instead of numeric IDs.
2. Given active suppliers, when the owner types part of a supplier name, then matching suppliers remain visible.
3. Given active products, when the owner searches by SKU or product name, then matching products remain visible.
4. Given a selected supplier and product, when the form is submitted, then hidden IDs are posted and the relationship is created.
5. Given no supplier selection, when the form is submitted, then the supplier field shows a specific validation error.
6. Given no product selection, when the form is submitted, then the product field shows a specific validation error.
7. Given a stale page where a selected parent was deactivated, when the form is submitted, then creation is rejected and the form explains that the record is inactive.
8. Given an existing relationship, when the owner edits it, then supplier and product are displayed as read-only labels rather than editable ID inputs.
9. Given an existing relationship, when the owner changes the reference cost, then only the cost/SKU/active fields change and the supplier/product pair remains unchanged.
10. Given a duplicate supplier/product pair, when the owner submits the form, then a clear duplicate relationship error is rendered without a server error page.
11. Given JavaScript is unavailable, when the owner uses the form, then a native fallback still permits selecting a supplier and product.
12. Given keyboard-only use, when the owner searches and selects an option, then focus and selection behavior are accessible and predictable.

## 8. Delivery sequence

1. Add this plan to the implementation ladder as sequence 038.
2. Define the searchable single-select view model and partial without changing multi-select.
3. Add Alpine filtering and keyboard interaction in `internal/web/static/app.js`.
4. Add CSS for the single-select states and responsive behavior.
5. Add active supplier/product option queries and handler view-model construction.
6. Replace raw ID inputs in `product-form.html` with the two searchable selectors.
7. Make create parsing strict and map stale/inactive/duplicate errors to fields.
8. Make edit parent fields read-only and remove the misleading identity update behavior.
9. Add template, handler, domain, and repository tests for selection, validation, filtering, and stale state.
10. Run `gofmt`, `go test ./...`, `go vet ./...`, and manual browser checks for mouse, keyboard, mobile, and no-JavaScript fallback.

## 9. Dependencies and readiness

**Dependencies:**

- `031-trading-master-data-plan.md` implementation.
- Existing Alpine.js, HTMX, Go template, and CSS conventions.
- Active supplier and product master queries.

**Ready for development when:**

- The product/supplier pair is approved as immutable identity after creation.
- Local filtering is accepted for the MVP option-set size.
- Visible labels and search fields are approved.
- The no-JavaScript fallback behavior is agreed.
- Duplicate, inactive, malformed, and stale-selection error behavior is approved.

**Done when:**

- No user-facing supplier/product field requires typing a database ID.
- The submitted hidden IDs are always generated from selected options.
- All server-side validation remains in place.
- The edit form no longer implies that relationship identity can be changed.
- Acceptance scenarios pass and sequence 038 is updated with implementation evidence.
