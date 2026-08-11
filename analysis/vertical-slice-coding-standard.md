# Vertical Slice Page Coding Standard

This standard applies to every feature page and action in the collection tracking system. It follows the vertical slice architecture, Go `html/template`, HTMX, Alpine.js, GORM, SQL Server, and TDD plans.

## 1. Required Slice Structure

Each slice owns the complete feature flow:

```text
internal/slices/<feature>/
├── domain.go
├── commands.go
├── queries.go
├── repository.go
├── repository_gorm.go
├── handler.go
├── view_models.go
├── models.go
├── templates/
│   ├── page.html
│   ├── form.html
│   └── partials/
└── *_test.go
```

Rules:

- Keep feature code inside its slice.
- Do not create global `handlers`, `services`, or `repositories` packages.
- Keep GORM models separate from domain entities and template view models.
- Define repository interfaces in the slice that consumes them.
- Share code only when it is genuinely cross-slice and stable.
- Do not let templates access repositories or GORM models.

## 2. Naming Conventions

- Use nouns for domain types: `CompanyAccount`, `DeliveryReceivable`.
- Use verbs for commands: `CreateAccount`, `RecordPayment`, `CancelReceivable`.
- Use query names that describe the result: `ListAccounts`, `GetDashboardClients`.
- Use `Handler` suffix for HTTP handlers.
- Use `Repository` suffix for persistence ports and adapters.
- Use `ViewModel` suffix for template-specific data.
- Use `Command` and `Query` suffixes for input DTOs where useful.
- Keep exported names documented when they are part of a package API.
- Return domain-specific errors such as `ErrNotFound`, `ErrValidation`, and `ErrConflict`.

## 3. Request Flow

Every page or command follows this flow:

```text
HTTP request
-> transport parsing
-> command/query input
-> domain/application behavior
-> repository port
-> GORM adapter
-> view model
-> full page or HTMX fragment
```

Handlers must:

- Parse path, query, and form values.
- Validate transport-specific syntax.
- Create a command or query input.
- Call exactly one slice use-case boundary.
- Map known errors to user-facing responses.
- Return a full page or fragment.

Handlers must not:

- Calculate due dates.
- Calculate dashboard totals.
- Apply financial rules.
- Construct SQL.
- Read GORM models directly into templates.
- Decide authorization in templates.

## 4. Commands and Queries

### Commands

Commands change state:

- Create account.
- Update account.
- Create receivable.
- Record payment.
- Cancel or reopen receivable.
- Archive or restore receivable.

Command rules:

- Validate business invariants in the domain/application layer.
- Use a request id and idempotency key.
- Use a transaction when changing data and writing an audit event.
- Capture previous values before updates.
- Check `rowversion` before committing edits.
- Return a stable result or domain error.
- Use Post/Redirect/Get after successful normal form submissions.

### Queries

Queries read state:

- List accounts.
- Get receivable detail.
- Get dashboard summary.
- List dashboard clients.
- Load PO drill-down rows.

Query rules:

- Use read-specific DTOs and projections.
- Apply filters on the server.
- Exclude archived and cancelled records from ordinary totals.
- Use deterministic ordering and cursor pagination.
- Return `items`, `next_cursor`, and `remaining_count` for load-more lists.
- Do not mutate data or create audit events.

## 5. Repository Standards

Repository methods should be small and intention-revealing:

```go
type Repository interface {
    Create(ctx context.Context, account Account) (Account, error)
    FindByID(ctx context.Context, id int64) (Account, error)
    Update(ctx context.Context, account Account, version []byte) error
}
```

Rules:

- Accept `context.Context` as the first argument.
- Return domain entities or query projections, not GORM models.
- Use parameterized GORM queries.
- Select only required columns for list and dashboard queries.
- Map `gorm.ErrRecordNotFound` to a domain/application not-found error.
- Map unique, foreign-key, check-constraint, and rowversion failures to known application errors.
- Keep SQL Server-specific syntax inside the GORM adapter.
- Use transactions from the command boundary, not hidden inside unrelated repository methods.
- Add `sqlmock` tests for success, not-found, constraint errors, rollback, and commit.

## 6. GORM Model Standards

- Use explicit table names when they differ from GORM conventions.
- Match migration column names and types exactly.
- Do not use `AutoMigrate` at server startup.
- Do not use floating-point types for money.
- Store scaled PHP amounts in `int64`/SQL Server `BIGINT`.
- Use UTC timestamps according to the date conversion contract.
- Include `rowversion` handling for editable records.
- Keep audit and idempotency writes inside the command transaction.

## 7. Go Template Standards

Use Go `html/template` and view models:

```go
type ReceivableDetailViewModel struct {
    ID             int64
    CompanyName    string
    PONumber       string
    AmountDisplay  string
    DeliveryDate   string
    DueDate        string
    Classification string
}
```

Rules:

- Escape output through `html/template`.
- Format money and dates before rendering.
- Keep template expressions simple.
- Use named partials for repeated rows, cards, alerts, and forms.
- Keep each partial's root element stable for HTMX replacement.
- Show field errors beside the related input.
- Preserve submitted values when validation fails.
- Use semantic headings, labels, buttons, tables, and landmarks.
- Never render raw user input with unsafe HTML.

## 8. HTMX Standards

Use HTMX for server communication and partial updates.

### Forms

```html
<form
  method="post"
  action="/receivables"
  hx-post="/receivables"
  hx-target="#receivable-form"
  hx-swap="outerHTML"
  hx-disabled-elt="find button[type='submit']">
  <!-- server-rendered fields and errors -->
  <button type="submit">Save</button>
</form>
```

Rules:

- Keep normal `method="post"` and `action` attributes as progressive enhancement.
- Return the form fragment for validation errors.
- Return `HX-Redirect` after successful commands.
- Disable submit controls while requests are active.
- Include idempotency keys in state-changing forms.
- Use `hx-target` and `hx-swap` explicitly.
- Do not use client-side JavaScript to duplicate domain rules.
- Avoid polling unless a real business requirement exists.

### Lists and filters

- Use `hx-get` for filter and search requests.
- Use `hx-trigger` with a debounce for search inputs.
- Use `hx-push-url="true"` when filters should be bookmarkable.
- Return the complete result fragment for the active filter state.
- Preserve filter state in query parameters.
- Use `hx-swap="afterend"` or a stable replacement target for load-more results.

### Loading and errors

- Show a loading state with `htmx-indicator`.
- Return accessible error alerts for failed requests.
- Keep the current page usable if an HTMX request fails.
- Do not expose raw database errors in fragments.

## 9. Alpine.js Standards

Alpine.js is only for local presentation state:

- Mobile navigation.
- Filter drawers.
- Confirmation modals.
- Disclosure panels.
- Local theme preference.
- Focus and Escape-key behavior.

Rules:

- Use small `x-data` scopes.
- Use `x-cloak` for initially hidden content.
- Use semantic controls and `aria-expanded`/`aria-controls`.
- Keep forms usable when Alpine fails to load.
- Do not store authoritative financial state in Alpine.
- Do not calculate amounts, due dates, classifications, or dashboard totals in Alpine.
- Let HTMX responses replace server-owned HTML.

## 10. Page-Specific Standards

### Account page

- Show all required company fields.
- Validate blank and trimmed values.
- Show account list with company name, TIN, contact person, and contact number.
- Keep account archiving out of the MVP.
- Audit create and update commands.

### Receivable page

- Require company, PO number, amount, delivery date, and payment term.
- Display the calculated due date after save.
- Use the shared money and business-date helpers.
- Keep payment date optional during creation.
- Show lifecycle status and derived classification separately.
- Audit create, update, payment, cancel, reopen, archive, and restore commands.

### Dashboard page

- Show counts and PHP amounts on every summary card.
- Put overdue work first.
- Aggregate rows by client.
- Provide PO-level drill-down.
- Use server-side filters.
- Use deterministic ordering and cursor load-more.
- Display remaining row count.
- Do not render a dashboard based on stale client-side totals.

### Audit timeline

- Show action, actor, local Philippines timestamp, and note.
- Show previous and new values in a readable format.
- Never show edit or delete controls for audit events.
- Keep audit data append-only.

## 11. Testing Standard Per Slice

Every slice must include:

- Domain table-driven tests.
- Command success and validation tests.
- Query filter and ordering tests.
- Repository `sqlmock` tests.
- Transaction commit and rollback tests.
- Idempotency replay and request-conflict tests.
- `rowversion` stale-update tests for editable records.
- HTTP handler tests using `httptest`.
- Full-page and HTMX fragment response tests.
- Template rendering tests for empty, valid, and error states.
- Accessibility checks for labels, headings, status text, and dialog attributes.

## 12. Per-Page Definition Of Done

A vertical slice page is complete only when:

- The domain behavior is covered by tests.
- The command/query boundary is explicit.
- The repository interface and GORM adapter are tested.
- Database changes have a reviewed migration.
- Audit behavior is defined for every state change.
- Idempotency is implemented for every state-changing form.
- `rowversion` is checked for editable records.
- Full-page requests work without JavaScript.
- HTMX fragment requests work correctly.
- Alpine behavior is limited to presentation state.
- Validation errors are accessible and preserve input.
- Loading, empty, success, and failure states exist.
- Mobile and desktop layouts are usable.
- `go test ./...`, `go vet ./...`, and formatting pass.
- The page is reviewed against the business acceptance scenarios.

## 13. References

- Go `html/template`: https://pkg.go.dev/html/template
- Go `net/http`: https://pkg.go.dev/net/http
- HTMX documentation: https://htmx.org/docs/
- Alpine.js documentation: https://alpinejs.dev/start-here
- Tailwind UI plan: `analysis/tailwind-ui-plan.md`
- First vertical slice plan: `analysis/first-vertical-slice-plan.md`
