# Edit Company Account Implementation Plan

## 1. Goal

Add the edit workflow to the existing company-account vertical slice without creating a second feature architecture.

The edit workflow must follow `analysis/004-vertical-slice-coding-standard.md` and reuse the existing account form, repository, audit, idempotency, HTMX, Alpine.js, and test patterns.

## 2. Scope

### In scope

- Open an existing company account in edit mode.
- Display all current values in the account form.
- Validate all required fields on update.
- Trim and persist edited values.
- Detect stale edits with SQL Server `rowversion`.
- Write one audit event for each successful update.
- Prevent duplicate update commands with idempotency keys.
- Support full-page and HTMX update responses.

### Out of scope

- Company-account archiving.
- Company-account deletion.
- Receivable updates caused by account edits.
- Authentication middleware.
- Customer self-service access.

Editing account information must not recalculate or modify existing delivery receivables.

## 3. Dependencies

- Company-account create/list slice is working.
- `company_accounts` has `updated_at_utc` and SQL Server `rowversion` support if the account schema includes it.
- `audit_events` and `idempotency_keys` are available.
- The shared account form template exists.
- The server uses standard `net/http` routing.

### Schema check

The current migration contract must include a concurrency version for editable company accounts. If `dbo.company_accounts` does not have a `rowversion` column, add a new reviewed migration before implementing the update repository method.

## 4. Ordered Subtasks

### A. Confirm account concurrency schema

1. Inspect `dbo.company_accounts` for a `rowversion` column.
2. Add a migration if it is missing.
3. Confirm the repository model maps the version field correctly.
4. Confirm the update condition includes both account ID and original rowversion.

Expected update shape:

```sql
UPDATE dbo.company_accounts
SET company_name = @company_name,
    contact_person = @contact_person,
    tin_number = @tin_number,
    billing_address = @billing_address,
    delivery_address = @delivery_address,
    contact_number = @contact_number,
    updated_at_utc = SYSUTCDATETIME()
WHERE company_account_id = @id
  AND row_version = @original_row_version;
```

If zero rows are affected, return a conflict instead of overwriting another edit.

### B. Add the update domain command

Create `UpdateAccountCommand` with:

- Account ID
- Company name
- Contact person
- TIN number
- Billing address
- Delivery address
- Contact number
- Original rowversion
- Request ID
- Idempotency key
- Actor ID

Reuse the same domain validation used by account creation. Do not duplicate validation rules in the handler.

### C. Add repository update support

Extend the account repository port:

```go
Update(ctx context.Context, db *gorm.DB, account CompanyAccount, originalVersion []byte) error
FindByID(ctx context.Context, id int64) (CompanyAccount, error)
```

Repository rules:

- Select the current account and version for the edit form.
- Use the original version in the update predicate.
- Return a conflict error when no row is updated.
- Use the existing transaction supplied by the command service.
- Map not-found and conflict errors separately.
- Do not update delivery receivables.

### D. Add service transaction behavior

The update command must:

1. Validate and normalize the submitted fields.
2. Validate the idempotency key and request hash.
3. Start one database transaction.
4. Load the current account values.
5. Capture the previous values.
6. Update the account using the original rowversion.
7. Insert one audit event with previous and new values.
8. Store the idempotency result.
9. Commit all writes together.

If the account update, audit insert, or idempotency update fails, roll back the entire transaction.

### E. Add routes and handler behavior

Add:

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/accounts/{id}/edit` | Render edit form |
| `POST` | `/accounts/{id}` | Update account |

Handler rules:

- Parse the account ID from the route.
- Load the account and version for the edit form.
- Return a not-found response when the account does not exist.
- Preserve submitted values and errors on validation failure.
- Return an HTMX form fragment with status `422` for validation errors.
- Return `HX-Redirect: /accounts` after a successful HTMX update.
- Redirect normal successful POST requests with `303 See Other`.
- Return a conflict response when the rowversion is stale.
- Do not expose raw database errors.

### F. Reuse and update templates

Reuse the create form partial with an explicit mode:

```go
type AccountFormViewModel struct {
    Mode           string
    AccountID      int64
    Values         CompanyAccount
    Errors         ValidationErrors
    IdempotencyKey string
    RowVersion     string
}
```

The form should:

- Use `method="post"` and the account-specific action URL.
- Include the original rowversion as a hidden value.
- Include a new idempotency key for each rendered edit form.
- Change the heading and submit label to `Edit company account` and `Save changes`.
- Use the same HTMX attributes as the create form.
- Preserve all submitted values after validation failure.

Example:

```html
<form
  id="account-form"
  method="post"
  action="/accounts/{{ .AccountID }}"
  hx-post="/accounts/{{ .AccountID }}"
  hx-target="#account-form"
  hx-swap="outerHTML"
  hx-disabled-elt="find button[type='submit']">
  <input type="hidden" name="row_version" value="{{ .RowVersion }}">
  <input type="hidden" name="idempotency_key" value="{{ .IdempotencyKey }}">
  <!-- shared account fields -->
  <button type="submit">Save changes</button>
</form>
```

Alpine.js may control a local confirmation or unsaved-changes warning, but it must not own the account values or update decision.

### G. Add audit and idempotency behavior

Audit action:

- `update`

Audit data must include:

- Account entity type and ID.
- Actor ID.
- UTC timestamp.
- Previous account values.
- New account values.
- Request ID.
- Idempotency key.

Idempotency behavior:

- Same key and same normalized request returns the original update result.
- Same key with a different request is rejected.
- A replay must not create another audit event.

### H. Add tests

Domain tests:

- Same validation behavior as create.
- Trimming edited fields.
- Maximum length validation.

Application tests:

- Successful update.
- Account not found.
- Stale rowversion conflict.
- Audit failure rolls back the account update.
- Idempotent replay does not update twice.
- Idempotency payload conflict is rejected.

Repository tests with `sqlmock`:

- Find account and rowversion.
- Successful update with rowversion predicate.
- Zero affected rows maps to conflict.
- Transaction commit and rollback.
- Database error mapping.

HTTP tests with `httptest`:

- `GET /accounts/{id}/edit` renders current values.
- Missing account returns `404`.
- Invalid normal POST re-renders the full form.
- Invalid HTMX POST returns a `422` form fragment.
- Successful normal POST redirects with `303`.
- Successful HTMX POST returns `HX-Redirect`.
- Stale rowversion returns a user-safe conflict response.

## 5. Acceptance Scenarios

1. Given an existing account, when the edit page opens, then all current values are displayed.
2. Given valid changed values, when the form is submitted, then the account is updated.
3. Given a blank required field, when the form is submitted, then no update occurs and the field error is displayed.
4. Given a stale edit form, when it is submitted after another update, then the update is rejected without overwriting newer data.
5. Given a successful update, then exactly one immutable audit event is created.
6. Given an audit failure, then the account update is rolled back.
7. Given the same idempotency key and payload submitted twice, then the update and audit event occur only once.
8. Given an edit to a company account, then existing delivery receivables remain unchanged.
9. Given JavaScript is disabled, then the edit form still works through normal HTTP.

## 6. Definition Of Done

- The edit workflow remains inside `internal/slices/accounts`.
- The create and edit forms share reusable field partials.
- All validation rules are domain-owned and not duplicated in handlers.
- GORM models remain separate from domain entities and view models.
- The update uses `rowversion` optimistic concurrency.
- Stale updates never overwrite newer data.
- The command is idempotent.
- The account update and audit event commit atomically.
- HTMX and normal form flows both work.
- Alpine.js remains presentation-only.
- Errors are accessible and preserve submitted values.
- Unit, repository, service, HTTP, and template tests pass.
- `go test ./...`, `go vet ./...`, and formatting pass.
- The progress tracker is updated.
