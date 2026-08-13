# Delivery Receivable Filter Plan

## 1. Goal

Add server-side filters to the delivery receivables list for:

- Company name
- Invoice number
- PO number
- Multiple classification/status values

The filter flow must follow `analysis/vertical-slice-coding-standard.md` and use Go, GORM, SQL Server, `html/template`, HTMX, and Alpine.js presentation-only behavior.

The existing receivable slice already implements the company, PO, and status filters. This plan extends that flow with an invoice-number filter. No database migration is required because `invoice_number` is already persisted by migration `0004`.

## 2. Filter Contract

### Company name

- Query parameter: `company`
- Case-insensitive search according to the configured SQL Server collation.
- Trim whitespace.
- Match company name by prefix or contained text according to the selected query strategy.
- Recommended initial behavior: contained search for user convenience.

### PO number

- Query parameter: `po`
- Trim whitespace.
- Search using normalized uppercase PO value.
- Recommended behavior: exact or prefix search, not arbitrary wildcard construction.
- Do not expose or accept raw SQL wildcard expressions from the user.

### Invoice number

- Query parameter: `invoice`
- Trim whitespace before querying.
- Use the same escaped, case-insensitive contains-search behavior as the current PO filter, subject to the configured SQL Server collation.
- Do not expose or accept raw SQL wildcard expressions from the user.
- Preserve values such as `0220` as strings; never parse invoice numbers as integers so leading zeroes remain searchable.

### Multiple status/classification values

- Query parameter: repeated `status` values.
- Example:

```text
/receivables?status=overdue&status=near_due
```

- Values are combined with OR logic.
- Different filter categories are combined with AND logic.
- Valid values:
  - `pending`
  - `near_due`
  - `overdue`
  - `payment_received`
  - `cancelled`
  - `archived`
- Invalid status values are ignored or rejected consistently; recommended behavior is to ignore invalid values and preserve valid filters.
- No selected status means all records allowed by the page's default lifecycle scope.

## 3. Default Scope

The normal receivables list should show active records by default:

- Active records may classify as Pending, Near Due, Overdue, or Payment Received.
- Cancelled and Archived records are hidden unless explicitly selected through filters.
- If `cancelled` or `archived` is selected, include those lifecycle records in the result.
- A query selecting `overdue` and `near_due` must not include cancelled or archived records.

## 4. Query Model

Extend the existing slice-owned `ListQuery` type:

```go
type ListQuery struct {
    Company string
    Invoice string
    PO      string
    Statuses []string
    StatusFilterProvided bool
    Cursor string
    PageSize int
    Now time.Time
}
```

The query service should:

- Normalize the search values.
- Preserve invoice-number leading zeroes during normalization.
- Validate and normalize status values.
- Set a safe default page size and maximum.
- Supply one current Philippines business date for classification.
- Return filtered view models and pagination metadata.

## 5. Ordered Subtasks

### A. Define filter input and normalization

1. Add `Invoice` to the existing `ListQuery`.
2. Parse query parameters from `r.URL.Query()`.
3. Trim company, invoice, and PO values.
4. Normalize PO input to uppercase; keep invoice input as a trimmed string.
5. Parse repeated `status` values.
6. Remove duplicate status values.
7. Ignore or reject invalid status values according to the chosen policy.
8. Preserve normalized filters in the response view model.
9. Include invoice in `filterSignature` so a cursor cannot be reused with a different invoice filter.

### B. Define repository query

1. Join `delivery_receivables` with `company_accounts`.
2. Apply company-name filtering.
3. Apply escaped invoice-number filtering against `r.invoice_number`.
4. Apply normalized PO filtering.
5. Apply lifecycle filters for Cancelled and Archived.
6. Apply derived classification filters for active records.
7. Use parameterized query values.
8. Preserve deterministic ordering by due date and receivable ID.
9. Apply cursor pagination after filters.
10. Return `items`, `next_cursor`, and `remaining_count`.

The repository must not load all records and filter them in Go. Filtering belongs in SQL Server for scalability and consistent pagination.

### C. Handle derived classification filters

For active records, use the same classification rules as the receivable classification slice:

- Overdue: no payment date and due date before today.
- Near due: no payment date and due date today through five days ahead.
- Pending: no payment date and due date after the near-due window.
- Payment received: payment date exists.

The status filter expression must be centralized so the list and dashboard cannot disagree.

### D. Update receivable handler

1. Parse query parameters.
2. Build the existing `ListQuery` with the normalized invoice value.
3. Call the query service.
4. Render the full list for normal requests.
5. Render only the result fragment for HTMX requests.
6. Preserve filters in the URL.
7. Return safe errors with the reusable snackbar behavior.

### E. Update filter UI

Use a server-rendered filter form:

```html
<form
  id="receivable-filters"
  method="get"
  action="/receivables"
  hx-get="/receivables"
   hx-target="#receivable-results"
   hx-trigger="change, keyup changed delay:300ms from:input[name='company'], keyup changed delay:300ms from:input[name='invoice'], keyup changed delay:300ms from:input[name='po']"
  hx-push-url="true">
  <input name="company" value="{{ .Filters.Company }}" placeholder="Company name">
  <input name="invoice" value="{{ .Filters.Invoice }}" placeholder="Invoice number">
  <input name="po" value="{{ .Filters.PO }}" placeholder="PO number">

  <label><input type="checkbox" name="status" value="pending"> Pending</label>
  <label><input type="checkbox" name="status" value="near_due"> Near due</label>
  <label><input type="checkbox" name="status" value="overdue"> Overdue</label>
  <label><input type="checkbox" name="status" value="payment_received"> Payment received</label>

  <button type="reset">Clear filters</button>
</form>
```

Rules:

- Use normal GET behavior as the fallback.
- Keep selected filters checked after an HTMX response.
- Alpine.js may open/close a mobile filter drawer only.
- Alpine.js must not filter rows or calculate statuses.
- Include Cancelled and Archived filters in a secondary filter area if needed.

### F. Update results fragment

Create a stable replacement target:

```html
<div id="receivable-results">
  {{ template "receivable-results" . }}
</div>
```

The fragment should contain:

- Result count.
- Active filter summary.
- Filtered rows.
- Empty state explaining that no records match.
- Load-more control with remaining count when another page exists.

### G. Expected implementation surface

- `internal/slices/receivables/queries.go`: add invoice state to `ListQuery`, normalization, and cursor filter signature.
- `internal/slices/receivables/handler.go`: parse `invoice` and pass it into the page view model.
- `internal/slices/receivables/repository_gorm.go`: add a parameterized escaped predicate on `r.invoice_number`.
- `internal/slices/receivables/templates/list.html`: add the debounced invoice input.
- `internal/slices/receivables/templates/partials/receivable-results.html`: show the active invoice filter and preserve it in load-more URLs.
- Receivables query, repository, handler, and template tests: cover filtering, cursor binding, and rendering.

No changes are expected in `internal/migrations`, `receivableModel`, or `DeliveryReceivable`; the invoice column is already implemented.

## 6. UX Rules

- Show active filters as removable chips or readable text.
- Provide a clear-filters action.
- Show the number of matching receivables.
- Keep company name, invoice number, and PO number visible in every result row.
- Preserve filters when opening a detail page where practical.
- Show a useful empty state instead of a blank table.
- Avoid submitting a request for every keystroke; debounce search input by approximately 300ms.
- Do not use color alone for status filters or result badges.

## 7. Tests

### Query/service tests

- Company name filter is trimmed and applied.
- Invoice number filter is trimmed, preserves leading zeroes, and is applied.
- PO filter is normalized and applied.
- One status filter works.
- Multiple statuses use OR logic.
- Company, invoice, and PO filters combine with status filters using AND logic.
- Invalid status values do not broaden the query.
- Cancelled and archived records are excluded by default.
- Explicit cancelled/archived filters include the intended records.
- Filtered pagination returns stable cursors.
- Remaining count uses the same filters as the page query.

### Repository tests with `sqlmock`

- Company join and filter parameters are bound correctly.
- Invoice filter parameter is escaped and bound correctly against `invoice_number`.
- PO normalization parameter is bound correctly.
- Multiple status parameters are bound correctly.
- No raw user input is concatenated into SQL.
- Empty results map to a valid empty view.
- Database errors map to safe application errors.

### HTTP/template tests

- Filters render with submitted values.
- Invoice filter value renders with leading zeroes preserved.
- Multiple status checkboxes remain selected.
- Normal GET returns a full page.
- HTMX GET returns only the results fragment.
- `hx-push-url` query state is preserved.
- Empty state renders for no matches.
- Load-more includes the remaining count.
- Errors trigger the reusable snackbar behavior.

## 8. Acceptance Scenarios

1. Given a company name, when the user searches, then only matching receivables appear.
2. Given an invoice number such as `0220`, when the user searches, then matching receivables appear and leading zeroes are preserved.
3. Given a PO number, when the user searches, then matching receivables appear with company names.
4. Given `overdue` and `near_due` selected, then records matching either classification appear.
5. Given invoice number and `overdue` selected, then only overdue receivables for that invoice appear.
6. Given no matching records, then a clear empty state appears.
7. Given filters are active, when the user loads more, then the same invoice, company, PO, and status filters remain applied.
8. Given Cancelled is not selected, then cancelled records remain hidden.
9. Given multiple status filters, then counts and rows match the same OR/AND rules.
10. Given JavaScript is disabled, then GET filters still work through normal browser navigation.

## 9. Definition Of Done

- Company name, invoice number, PO number, and multiple status filters are implemented server-side.
- Filter state is represented in the URL.
- Multiple statuses use OR logic within the status category.
- Different filter categories use AND logic.
- Cancelled and archived default behavior is explicit.
- Classification rules are shared with the receivable and dashboard slices.
- HTMX replaces only the results fragment.
- Alpine.js is presentation-only.
- Empty, loading, error, and filtered-result states exist.
- Cursor pagination and remaining count remain correct after filtering.
- Unit, repository, HTTP, and template tests pass.
- `go test ./...`, `go vet ./...`, and formatting pass.
- `analysis/progress.md` is updated.
