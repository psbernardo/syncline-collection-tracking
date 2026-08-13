# First Vertical Slice Implementation Plan

## 1. Goal

Deliver the first usable workflow:

```text
Open application
-> Create company account
-> Create delivery receivable
-> Calculate due date
-> Persist receivable
-> Display Pending / Near Due / Overdue classification
-> View dashboard totals
```

This slice should use the existing migration, date helpers, money helpers, local HTMX asset, and local Alpine.js asset.

Authentication remains deferred until the middle of development. The application remains local-only during this slice.

## 2. Implementation Order

### Phase 1: Server Foundation

1. Create `cmd/server/main.go`.
2. Load `.env` using the existing configuration package.
3. Open the GORM SQL Server connection.
4. Configure connection pool limits and request context handling.
5. Register standard `net/http` routes.
6. Load Go `html/template` files.
7. Serve `internal/web/static` locally.
8. Bind to `127.0.0.1:8080`.
9. Add a health page that confirms the server is running without exposing database credentials.

### Phase 2: Shared Web Layout

Create:

- `internal/web/templates/layout.html`
- `internal/web/templates/partials/flash.html`
- `internal/web/templates/partials/validation-errors.html`
- `internal/web/static/app.css`

The layout should include:

- Application name and navigation.
- Dashboard link.
- Company accounts link.
- Receivables link.
- HTMX script loaded locally.
- Alpine.js script loaded locally.
- A main content block.

Use server-rendered HTML as the initial page state. JavaScript enhances interactions but is not the source of business truth.

### Phase 3: Company Account Slice

Feature path:

```text
internal/slices/accounts/
├── domain.go
├── commands.go
├── queries.go
├── repository.go
├── repository_gorm.go
├── handler.go
├── models.go
├── templates/
│   ├── list.html
│   └── form.html
└── *_test.go
```

Implement:

- List company accounts.
- Create company account.
- Edit company account.
- Validate all required fields.
- Trim text fields before persistence.
- Display validation errors beside the relevant fields.
- Write an audit event for create and update commands.

Required fields:

- Company name
- Contact person
- TIN number
- Billing address
- Delivery address
- Contact number

### Phase 4: Delivery Receivable Slice

Feature path:

```text
internal/slices/receivables/
├── domain.go
├── commands.go
├── queries.go
├── repository.go
├── repository_gorm.go
├── handler.go
├── models.go
├── templates/
│   ├── list.html
│   ├── form.html
│   └── row.html
└── *_test.go
```

Implement:

- List receivables.
- Create a receivable for an existing company.
- Validate alphanumeric PO number.
- Validate payment term from 1 through 120.
- Parse amount through the shared scaled-money helper.
- Parse delivery date through the shared business-date helper.
- Calculate due date with delivery date counted as day one.
- Persist the term and calculated due date.
- Write an audit event for creation and correction.

Required input:

- Company
- PO number
- Amount
- Delivery date
- Payment term

Do not calculate due dates in Alpine.js. The server/domain layer is authoritative.

### Phase 5: Dashboard Slice

Feature path:

```text
internal/slices/dashboard/
├── queries.go
├── repository.go
├── repository_gorm.go
├── handler.go
├── view_models.go
├── templates/
│   ├── dashboard.html
│   ├── summary-cards.html
│   ├── client-row.html
│   └── load-more.html
└── *_test.go
```

Implement:

- Summary cards for overdue, near due, pending, and payment received.
- Both counts and PHP totals.
- Client aggregation before rendering.
- Client drill-down to PO-level receivables.
- Overdue ordering by days overdue, amount, then company name.
- Near-due ordering by nearest due date.
- Server-side search and filters.
- Cursor pagination with `next_cursor` and `remaining_count`.

## 3. Route Plan

### Server and layout routes

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/` | Redirect to dashboard |
| `GET` | `/health` | Local health response |
| `GET` | `/static/` | Local CSS and JavaScript assets |

### Account routes

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/accounts` | Account list |
| `GET` | `/accounts/new` | Account form |
| `POST` | `/accounts` | Create account |
| `GET` | `/accounts/{id}/edit` | Edit form |
| `POST` | `/accounts/{id}` | Update account |

### Receivable routes

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/receivables` | Receivable list |
| `GET` | `/receivables/new` | Receivable form |
| `POST` | `/receivables` | Create receivable |
| `GET` | `/receivables/{id}` | Receivable detail |
| `GET` | `/receivables/{id}/edit` | Edit form |
| `POST` | `/receivables/{id}` | Update receivable |

### Dashboard routes

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/dashboard` | Full dashboard page |
| `GET` | `/dashboard/summary` | Summary card fragment |
| `GET` | `/dashboard/clients` | Filtered client rows |
| `GET` | `/dashboard/clients/{id}/receivables` | PO drill-down |

Use path parsing from standard `net/http` initially. Keep route parsing separate from handlers so a router can be introduced later without changing use cases.

## 4. HTMX UI Patterns

### Form submission

Use normal form behavior first, then enhance it with HTMX:

```html
<form
  method="post"
  action="/receivables"
  hx-post="/receivables"
  hx-target="#receivable-form"
  hx-swap="outerHTML"
  hx-disabled-elt="find button[type='submit']">
  <label for="amount">Amount</label>
  <input id="amount" name="amount" inputmode="decimal" required>
  {{ template "field-error" .Errors.amount }}

  <button type="submit">Save receivable</button>
</form>
```

Rules:

- The server validates every field.
- On validation failure, return the form fragment with errors and submitted values.
- On success, use `HX-Redirect` or a normal redirect to avoid duplicate form submissions.
- Disable the submit button while the request is active.
- Do not return JSON for normal page interactions.

### Success redirect

For a successful command, return:

```http
HX-Redirect: /receivables/42
```

The redirect ensures refresh does not repeat the POST.

### Dashboard load more

```html
<section id="overdue-clients">
  {{ range .OverdueClients }}
    {{ template "client-row" . }}
  {{ end }}
</section>

{{ if .NextCursor }}
<button
  id="load-more-overdue"
  hx-get="/dashboard/clients?classification=overdue&cursor={{ .NextCursor }}"
  hx-target="#load-more-overdue"
  hx-swap="outerHTML">
  Load more
  <span>({{ .RemainingCount }} remaining)</span>
</button>
{{ end }}
```

The server must apply the same filters to the page query and remaining-count query.

### Filter changes

```html
<form
  id="dashboard-filters"
  hx-get="/dashboard/clients"
  hx-target="#client-results"
  hx-trigger="change, keyup changed delay:300ms from:input[name='q']"
  hx-push-url="true">
  <input name="q" placeholder="Company or PO number">
  <select name="classification">
    <option value="">All active</option>
    <option value="overdue">Overdue</option>
    <option value="near_due">Near due</option>
    <option value="pending">Pending</option>
    <option value="payment_received">Payment received</option>
  </select>
</form>

<div id="client-results">
  {{ template "client-results" . }}
</div>
```

Use `hx-push-url` only when the filter state should be bookmarkable and restorable with browser navigation.

## 5. Alpine.js UI Patterns

Use Alpine.js for local UI state only.

### Confirmation modal

```html
<div x-data="{ open: false }">
  <button type="button" @click="open = true">Archive</button>

  <div x-show="open" x-cloak @keydown.escape.window="open = false">
    <div role="dialog" aria-modal="true" aria-labelledby="archive-title">
      <h2 id="archive-title">Archive receivable?</h2>
      <button type="button" @click="open = false">Cancel</button>
      <form method="post" action="/receivables/42/archive">
        <button type="submit">Confirm archive</button>
      </form>
    </div>
  </div>
</div>
```

Rules:

- Alpine controls modal visibility, local toggles, and presentation state.
- Alpine does not calculate due dates, totals, balances, or statuses.
- Keep forms usable if Alpine fails to load.
- Use `x-cloak` to prevent modal flash before Alpine initializes.
- Use semantic buttons, labels, focus handling, and `aria` attributes.

## 6. TDD Checkpoints

### Server foundation

- Configuration loads `.env` and defaults.
- Routes return expected status codes.
- Templates render without leaking persistence models.

### Account slice

- Required field validation.
- Create and update commands.
- Repository success, not-found, and database-error behavior.
- Audit event generated exactly once per successful command.

### Receivable slice

- Payment term boundaries: 1, 120, 0, and 121.
- Amount scaling and four-decimal rounding.
- Delivery date and due-date calculation.
- PO validation.
- Idempotent create behavior.
- `rowversion` stale update behavior.

### Dashboard slice

- Client aggregation.
- Mixed client classifications.
- Overdue and near-due boundaries.
- Search and filters.
- Cursor pagination.
- Exact remaining count.
- PO drill-down.

## 7. Definition Of Done

The first vertical slice is complete when:

- A company account can be created through the browser.
- A receivable can be created for that account.
- The due date is calculated according to the approved business rule.
- The receivable is stored in SQL Server.
- The dashboard shows the correct derived classification.
- All state-changing commands write one audit event transactionally.
- Repeated form submissions do not create duplicate records.
- Unit, repository, and HTTP tests pass.
- The UI works without an external network connection.

## 8. Reference Documentation

- HTMX documentation: https://htmx.org/docs/
- Alpine.js documentation: https://alpinejs.dev/start-here
- Go HTML templates: https://pkg.go.dev/html/template
- Go `net/http`: https://pkg.go.dev/net/http
