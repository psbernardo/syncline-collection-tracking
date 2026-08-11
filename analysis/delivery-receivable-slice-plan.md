# Create Delivery Receivable Slice Implementation Plan

## 1. Goal

Implement the vertical slice for recording money owed after a client delivery.

The slice must follow `analysis/vertical-slice-coding-standard.md` and use:

- Go vertical slice architecture
- GORM and SQL Server
- Go `html/template`
- HTMX for server interactions
- Alpine.js only for local presentation state
- Shared Philippines date and scaled-money helpers
- TDD with `testing`, `sqlmock`, and `httptest`
- Transactional immutable audit events
- Idempotent state-changing commands

## 2. Scope

### In scope

- List delivery receivables.
- Display the create-receivable form.
- Select an existing company account.
- Enter PO number, amount, delivery date, and payment term.
- Validate payment terms from 1 through 120 calendar days.
- Calculate the due date with the delivery date counted as day one.
- Persist the calculated due date and original payment term.
- Derive `Pending`, `Near Due`, or `Overdue` classification.
- Write one audit event for a successful create command.
- Prevent duplicate receivable creation with idempotency keys.
- Support full-page and HTMX form responses.

### Out of scope

- Partial payments.
- Payment recording.
- Payment proof or reconciliation.
- Dashboard aggregation.
- Client-level dashboard rows.
- Receivable cancellation, archive, restore, or reopening actions.
- Authentication middleware.
- Customer self-service access.

## 3. Dependencies

Before starting:

- Company-account create/list/edit slice is working.
- Migration `0001` and `0002` are applied to `CTS_DEV`.
- `company_accounts` and `delivery_receivables` exist.
- `audit_events` and `idempotency_keys` exist.
- Date helper tests pass.
- Money helper tests pass.
- Server composition root and local templates are available.

## 4. Slice Structure

Create:

```text
internal/slices/receivables/
├── domain.go
├── commands.go
├── queries.go
├── repository.go
├── repository_gorm.go
├── handler.go
├── view_models.go
├── models.go
├── templates/
│   ├── list.html
│   ├── form.html
│   └── partials/
│       ├── receivable-form.html
│       ├── receivable-row.html
│       └── validation-errors.html
└── *_test.go
```

## 5. Ordered Subtasks

### A. Confirm database contract

1. Verify `dbo.delivery_receivables` matches `analysis/database-schema.md`.
2. Confirm the company foreign key exists.
3. Confirm payment term check constraint `1-120`.
4. Confirm non-negative scaled amount constraint.
5. Confirm lifecycle status defaults to `Active`.
6. Confirm due-date, payment-date, company, PO, and active-record indexes exist.
7. Confirm `row_version` exists for future edits.

### B. Define the domain entity

Create `DeliveryReceivable` with:

- ID
- Company account ID
- PO number
- Delivery date
- Payment term in days
- Due date
- Amount due as scaled PHP integer
- Optional payment date
- Persisted lifecycle status
- Created/updated timestamps
- Rowversion

Domain rules:

- PO number is required and ASCII alphanumeric.
- Payment term must be a whole number from 1 through 120.
- Amount must be non-negative.
- Amount input is parsed by the shared money helper.
- Delivery date is parsed by the shared business-date helper.
- Due date is calculated as:

```text
delivery date + (payment term - 1) calendar days
```

- Due-date calculation uses `Asia/Manila`.
- Persisted dates use the approved UTC conversion contract.
- New records start with lifecycle status `Active`.
- Derived classification is calculated from due date and payment date.
- Do not accept payment date on the create command.

### C. Define commands and queries

Create:

- `CreateReceivableCommand`
- `ListReceivablesQuery`
- `GetReceivableQuery`
- `ReceivableViewModel`
- `ReceivableFormViewModel`

The create command contains:

- Company account ID
- PO number
- Amount input
- Delivery date input
- Payment term input
- Request ID
- Idempotency key
- Actor ID

Command flow:

1. Parse and validate input.
2. Load and verify the company account exists.
3. Calculate the due date.
4. Normalize the amount to scaled PHP units.
5. Start a database transaction.
6. Check or create the idempotency record.
7. Insert the receivable.
8. Insert one audit event.
9. Store the result entity ID.
10. Commit all writes together.

### D. Define repository interfaces

```go
type Repository interface {
    Create(ctx context.Context, db *gorm.DB, receivable DeliveryReceivable) (DeliveryReceivable, error)
    FindByID(ctx context.Context, db *gorm.DB, id int64) (DeliveryReceivable, error)
    List(ctx context.Context) ([]DeliveryReceivable, error)
}
```

The company-account lookup should use an explicit account repository port or a small account existence port. Do not import the accounts slice persistence implementation directly.

Repository rules:

- Accept `context.Context`.
- Use the transaction supplied by the command service.
- Return domain entities or query projections, not GORM models.
- Map missing company accounts to a known application error.
- Keep SQL Server-specific behavior in the GORM adapter.
- Select only required columns for list queries.

### E. Implement GORM models and persistence adapter

1. Map all migration column names explicitly.
2. Represent `amount_due_scaled` as `int64`.
3. Represent UTC dates consistently with the shared date helper.
4. Map `row_version` as `[]byte`.
5. Use the company foreign key.
6. Add repository methods for create, list, and detail.
7. Preserve the original payment term and due date.

### F. Implement routes and handlers

Add:

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/receivables` | Render receivable list |
| `GET` | `/receivables/new` | Render create form |
| `POST` | `/receivables` | Create receivable |
| `GET` | `/receivables/{id}` | Render receivable detail |

Handler rules:

- Parse request values only.
- Build a command or query.
- Call the slice service boundary.
- Map validation errors to fields.
- Return full HTML for normal requests.
- Return form/row fragments for HTMX requests.
- Return `HX-Redirect` after successful HTMX creation.
- Return `404` when the company or receivable does not exist.
- Return a safe conflict/error response without exposing SQL details.

### G. Implement templates using the Tailwind standard

The create form must include:

- Company select.
- PO number input.
- PHP amount input.
- Delivery date input.
- Payment term number input with `1-120 days` guidance.
- Calculated due date after successful save.
- Derived classification on the detail/list view.
- Hidden idempotency key.

Example form:

```html
<form
  id="receivable-form"
  method="post"
  action="/receivables"
  hx-post="/receivables"
  hx-target="#receivable-form"
  hx-swap="outerHTML"
  hx-disabled-elt="find button[type='submit']">
  <input type="hidden" name="idempotency_key" value="{{ .IdempotencyKey }}">

  <label for="company-account-id">Company</label>
  <select id="company-account-id" name="company_account_id" required>
    {{ range .Accounts }}
      <option value="{{ .ID }}">{{ .CompanyName }}</option>
    {{ end }}
  </select>

  <label for="amount">Amount</label>
  <input id="amount" name="amount" inputmode="decimal" required>

  <label for="delivery-date">Delivery date</label>
  <input id="delivery-date" name="delivery_date" type="date" required>

  <label for="payment-term-days">Payment term</label>
  <input id="payment-term-days" name="payment_term_days" type="number" min="1" max="120" required>

  <button type="submit">Save receivable</button>
</form>
```

Rules:

- Use `html/template` escaping.
- Preserve submitted values after validation failure.
- Show errors beside the related input.
- Keep normal POST behavior as a fallback.
- Return `422` with the form fragment for HTMX validation errors.
- Return `HX-Redirect` after success.
- Do not calculate the due date in the browser.

### H. Apply Alpine.js only to presentation state

Allowed:

- Company select search UI if needed.
- Local form disclosure sections.
- Confirmation modal for future actions.
- Loading or local visual state.

Not allowed:

- Due-date calculation.
- Money conversion.
- Classification logic.
- Database persistence.
- Audit or idempotency behavior.

The form must remain usable when Alpine.js is unavailable.

### I. Add audit and idempotency

Audit action:

- `create`

Audit data must include:

- Entity type `delivery_receivable`.
- Receivable ID.
- Actor ID.
- UTC timestamp.
- New values including company ID, PO, amount, delivery date, term, due date, and lifecycle status.
- Request ID.
- Idempotency key.

Repeated submission with the same key and payload must return the original receivable without inserting another receivable or audit event.

### J. Add tests

Domain tests:

- Term 1 due date.
- Term 120 due date.
- Invalid terms 0 and 121.
- Delivery date counts as day one.
- UTC/Philippines conversion.
- Money parsing and four-decimal rounding.
- Negative amount rejection.
- PO validation.
- Active lifecycle default.

Application tests:

- Successful create.
- Missing company account.
- Invalid form values.
- Idempotent replay.
- Idempotency payload conflict.
- Audit failure rolls back receivable creation.
- Company foreign-key failure mapping.

Repository tests with `sqlmock`:

- Create success.
- List success and ordering.
- Detail success.
- Not-found behavior.
- Foreign-key/check-constraint errors.
- Transaction commit and rollback.

HTTP tests with `httptest`:

- `GET /receivables` full page.
- `GET /receivables/new` full form.
- Valid normal POST redirects with `303`.
- Valid HTMX POST returns `HX-Redirect`.
- Invalid normal POST re-renders full form.
- Invalid HTMX POST returns `422` form fragment.
- Missing company returns a user-safe validation error.
- Detail page displays due date and classification.

## 6. Acceptance Scenarios

1. Given an existing company account, when valid receivable data is submitted, then the receivable is persisted.
2. Given delivery date `2026-08-10` and term `1`, then the due date is `2026-08-10`.
3. Given delivery date `2026-08-10` and term `5`, then the due date is `2026-08-14`.
4. Given an invalid term outside `1-120`, then no receivable is created.
5. Given a negative amount, then no receivable is created.
6. Given a missing company account, then no receivable is created.
7. Given a successful create command, then exactly one immutable audit event exists.
8. Given an audit failure, then the receivable and idempotency record are rolled back.
9. Given the same idempotency key and payload twice, then exactly one receivable exists.
10. Given JavaScript is disabled, then the form still submits and validates.
11. Given the receivable list, then the amount, PO, delivery date, due date, and classification are visible.

## 7. Definition Of Done

- The slice follows the vertical-slice folder structure.
- Domain logic has no HTTP or GORM dependencies.
- Due date uses the shared business-date helper.
- Amount uses the shared scaled-money helper.
- The term and due date are persisted.
- Company foreign-key validation works.
- Create is idempotent.
- Audit is immutable and transactional.
- Full-page and HTMX flows work.
- Alpine.js remains presentation-only.
- Empty, validation, success, and error states exist.
- Unit, repository, service, HTTP, and template tests pass.
- `go test ./...`, `go vet ./...`, and formatting pass.
- The progress tracker is updated.

## 8. Next Slice

After this slice, implement the dashboard aggregation slice using the persisted receivable due dates and active classifications.
