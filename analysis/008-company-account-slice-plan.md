# Company Account Slice Implementation Plan

## 1. Goal

Implement the first business vertical slice for creating and managing company accounts.

The slice must follow `analysis/004-vertical-slice-coding-standard.md` and use:

- Go vertical slice architecture
- GORM and SQL Server
- Go `html/template`
- HTMX for server interactions
- Alpine.js only for local presentation state
- TDD with `testing`, `sqlmock`, and `httptest`
- Transactional immutable audit events
- Idempotent state-changing commands

## 2. Scope

### In scope

- List company accounts.
- Display the create-account form.
- Create a company account.
- Validate all required fields.
- Preserve submitted values when validation fails.
- Render full-page and HTMX form responses.
- Record one audit event for a successful create command.
- Prevent duplicate account creation when the same form command is submitted again.

### Out of scope

- Company-account archiving.
- Company-account deletion.
- Authentication middleware.
- Receivable creation.
- Dashboard aggregation.
- Customer self-service access.

## 3. Dependencies

Before starting:

- Migration `0001` is applied to `CTS_DEV`.
- `company_accounts` exists with required columns and constraints.
- `audit_events` exists.
- `idempotency_keys` exists.
- `.env` loads successfully.
- Server composition root is available or implemented as the first subtask.

## 4. Slice Structure

Create or complete:

```text
internal/slices/accounts/
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
│       ├── account-form.html
│       ├── account-row.html
│       └── validation-errors.html
└── *_test.go
```

## 5. Ordered Subtasks

### A. Confirm database contract

1. Verify `dbo.company_accounts` columns match `analysis/006-database-schema.md`.
2. Confirm required fields are non-null.
3. Confirm primary key and account indexes exist.
4. Confirm `created_at_utc` and `updated_at_utc` defaults.
5. Record any schema mismatch before writing repository code.

### B. Define the domain entity

1. Create the `CompanyAccount` domain type.
2. Add fields:
   - ID
   - Company name
   - Contact person
   - TIN number
   - Billing address
   - Delivery address
   - Contact number
   - Created/updated timestamps
3. Add a constructor or validation function.
4. Trim all text input before validation.
5. Reject blank values.
6. Apply the documented maximum lengths.
7. Keep the domain package independent from GORM and HTTP.

### C. Define application commands and queries

Create:

- `CreateAccountCommand`
- `UpdateAccountCommand`, if editing is included in this slice
- `ListAccountsQuery`
- `CompanyAccountViewModel`

Command rules:

- Accept a request ID and idempotency key.
- Validate the domain entity.
- Start one transaction.
- Insert the account.
- Insert exactly one audit event.
- Store the idempotency result.
- Commit all changes together.

Query rules:

- Return view-specific projections.
- Order accounts by company name and ID.
- Exclude no accounts because account archiving is out of scope.
- Do not return GORM models to handlers or templates.

### D. Define repository interfaces

Use interfaces owned by the accounts slice:

```go
type Repository interface {
    Create(ctx context.Context, account CompanyAccount) (CompanyAccount, error)
    FindByID(ctx context.Context, id int64) (CompanyAccount, error)
    List(ctx context.Context) ([]CompanyAccount, error)
}
```

If audit and idempotency repositories are shared, expose only the smallest ports required by the command transaction.

### E. Implement GORM models and adapter

1. Create persistence-only GORM models matching migration columns.
2. Add explicit table names.
3. Map database rows to domain entities.
4. Map GORM not-found errors to application errors.
5. Keep SQL construction inside `repository_gorm.go`.
6. Use context-aware calls.
7. Select only required columns for the list query.
8. Use the existing database transaction for create and audit writes.

### F. Implement server routes

Add:

| Method | Path | Behavior |
|---|---|---|
| `GET` | `/accounts` | Render account list page |
| `GET` | `/accounts/new` | Render create form |
| `POST` | `/accounts` | Create account or return validation fragment |
| `GET` | `/accounts/{id}` | Render account detail, if included |

Handler rules:

- Parse form values only.
- Build a command.
- Call the application command/query.
- Map validation errors to fields.
- Return full HTML for normal requests.
- Return the form or row fragment for HTMX requests.
- Use `HX-Redirect` after a successful HTMX create.
- Never place validation or database logic in the handler.

### G. Implement templates with Tailwind

Create the account list and form using the selected Windmill/Tailwind visual language.

Form requirements:

- All fields have visible labels.
- Required fields use `required` and server validation.
- Error text appears beside the field.
- Invalid fields use a visible error style and accessible description.
- Submitted values remain after validation failure.
- Form has a normal `action` and `method` fallback.
- Form has HTMX enhancement attributes.
- Form includes an idempotency key.

Example:

```html
<form
  id="account-form"
  method="post"
  action="/accounts"
  hx-post="/accounts"
  hx-target="#account-form"
  hx-swap="outerHTML"
  hx-disabled-elt="find button[type='submit']">
  <label for="company-name">Company name</label>
  <input
    id="company-name"
    name="company_name"
    value="{{ .Values.CompanyName }}"
    required
    aria-describedby="company-name-error">
  <p id="company-name-error" class="text-red-600">{{ .Errors.CompanyName }}</p>

  <input type="hidden" name="idempotency_key" value="{{ .IdempotencyKey }}">
  <button type="submit">Save company</button>
</form>
```

### H. Apply Alpine.js only where needed

Use Alpine.js for:

- Mobile navigation.
- Form section disclosure if the form becomes long.
- Local confirmation modal for future destructive actions.

Do not use Alpine.js for:

- Validation decisions.
- Account persistence.
- Audit writes.
- Idempotency.
- Server state.

### I. Add audit and idempotency behavior

For account creation:

1. Generate or accept the idempotency key.
2. Hash the normalized request payload.
3. Start a database transaction.
4. Insert or validate the idempotency record.
5. Insert the company account.
6. Insert one `create` audit event.
7. Store the resulting account ID and response status.
8. Commit.

Repeated submission with the same key and payload returns the original account result. The command must not create a second account or second audit event.

### J. Add tests before finalizing implementation

Domain tests:

- Valid complete account.
- Blank company name.
- Blank contact person.
- Blank TIN number.
- Blank billing address.
- Blank delivery address.
- Blank contact number.
- Leading/trailing whitespace trimming.
- Maximum length behavior.

Application tests:

- Successful create.
- Validation failure.
- Repository failure.
- Audit failure rolls back account creation.
- Identical idempotency replay returns the original result.
- Idempotency payload conflict is rejected.

Repository tests with `sqlmock`:

- Insert success.
- List success.
- Not found mapping.
- Transaction commit.
- Transaction rollback.
- Constraint error mapping.

HTTP tests with `httptest`:

- `GET /accounts` full page.
- `GET /accounts/new` full form.
- Valid normal `POST /accounts` redirect.
- Valid HTMX `POST /accounts` `HX-Redirect`.
- Invalid normal form response.
- Invalid HTMX fragment response.
- Unexpected application error.

## 6. Acceptance Scenarios

1. Given the account form, when all required fields are submitted, then a company account is persisted.
2. Given a blank required field, when the form is submitted, then no account is persisted and the field error is displayed.
3. Given leading/trailing spaces, when the form is submitted, then stored values are trimmed.
4. Given a successful normal form submission, then the browser is redirected to the account list or detail page.
5. Given a successful HTMX submission, then the response contains `HX-Redirect`.
6. Given the same idempotency key and payload submitted twice, then exactly one account and one audit event exist.
7. Given an audit insert failure, then the account insert is rolled back.
8. Given JavaScript is disabled, then the account form still submits and validates correctly.
9. Given a mobile viewport, then the account form remains usable without horizontal scrolling.

## 7. Definition Of Done

- The account slice follows the required vertical-slice folder structure.
- Domain rules contain no HTTP or GORM dependencies.
- Repository interfaces are owned by the accounts slice.
- GORM models are not passed to templates.
- Full-page and HTMX requests are supported.
- Go `html/template` escaping is used.
- Tailwind styles match the selected application shell.
- Alpine.js is limited to local presentation behavior.
- Required validation and accessible error messages work.
- Idempotency prevents duplicate account creation.
- Exactly one audit event is written for a successful create.
- Account and audit writes are transactional.
- Unit, `sqlmock`, application, HTTP, and template tests pass.
- `go test ./...`, `go vet ./...`, and formatting pass.
- The progress tracker is updated with the completed subtasks.

## 8. Next Slice

After this slice is complete, implement delivery receivable creation using the selected company account and the existing date/money helpers.
