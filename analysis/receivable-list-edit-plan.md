# Receivable List and Edit Enhancement Plan

## 1. Current Behavior

The current receivable slice supports create, list, and detail. It does not yet show the company name in the receivable list and does not provide an edit route.

Receivable editing was intentionally excluded from the initial create-receivable slice so the first implementation could establish creation, due-date calculation, classification, audit, and idempotency first. The database already includes `row_version`, so editing can now be added safely with optimistic concurrency.

## 2. Goal

Enhance delivery receivables so that:

- Every list/grid row displays the company name.
- Receivable detail displays the company name.
- Active unpaid receivables can be edited.
- Corrections are audited and idempotent.
- Stale edits cannot overwrite newer changes.
- Due dates and classifications are recalculated after edits.

## 3. Proposed Edit Rules

### Editable for active unpaid receivables

- Company account
- PO number
- Amount
- Delivery date
- Payment term

Changing delivery date or payment term recalculates and persists the due date.

### Not editable through the normal edit form

- Payment date
- Derived classification
- Audit history
- Rowversion

Payment date belongs to the payment slice. Classification is always derived.

### Protected states

- `Payment Received`: requires an explicit correction/reopen flow before financial fields can change.
- `Cancelled`: requires reopening before editing.
- `Archived`: requires restoration to `Active` before editing.

Whether paid receivables can be corrected directly or only after a separate reopen action should be confirmed before implementing the protected-state flow.

## 4. Ordered Subtasks

### A. Add company name to list/detail projections

1. Add `CompanyName` to `ReceivableViewModel`.
2. Update the repository list query to join `dbo.company_accounts`.
3. Select only `company_account_id` and `company_name` from the account table.
4. Update the detail query to include company name.
5. Display company name before PO number in the list/grid.
6. Display company name in the detail heading and summary.
7. Add repository and template tests.

Recommended query shape:

```sql
SELECT
    r.delivery_receivable_id,
    r.company_account_id,
    a.company_name,
    r.po_number,
    r.amount_due_scaled,
    r.delivery_date_utc,
    r.due_date_utc,
    r.payment_term_days,
    r.payment_date_utc,
    r.lifecycle_status,
    r.row_version
FROM dbo.delivery_receivables AS r
INNER JOIN dbo.company_accounts AS a
    ON a.company_account_id = r.company_account_id
ORDER BY r.due_date_utc, r.delivery_receivable_id;
```

### B. Define the edit domain command

Create `UpdateReceivableCommand` with:

- Receivable ID
- Company account ID
- PO number
- Amount input
- Delivery date input
- Payment term
- Original rowversion
- Request ID
- Idempotency key
- Actor ID

Reuse the create validation and shared money/date helpers.

### C. Add repository update support

Add:

```go
FindByID(ctx context.Context, db *gorm.DB, id int64) (DeliveryReceivable, error)
Update(ctx context.Context, db *gorm.DB, receivable DeliveryReceivable, originalVersion []byte) (DeliveryReceivable, error)
```

Update rules:

- Use `delivery_receivable_id` and `row_version` in the `WHERE` clause.
- Exclude the generated `row_version` column from `UPDATE`.
- Return a conflict when zero rows are affected.
- Reload the updated row inside the same transaction.
- Do not update `payment_date_utc` through this command.
- Do not update audit events through the repository.

### D. Implement the transactional update service

1. Parse and validate the incoming values.
2. Verify the company account exists.
3. Calculate the new due date.
4. Check the idempotency key and request hash.
5. Load the previous receivable inside the transaction.
6. Reject protected lifecycle states according to the approved rules.
7. Update with the original rowversion.
8. Reload the updated record.
9. Write one `update` audit event with previous and new values.
10. Store the idempotency result.
11. Commit the transaction.

If any step fails, roll back the receivable update, audit event, and idempotency record together.

### E. Add edit routes and handlers

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/receivables/{id}/edit` | Render edit form |
| `POST` | `/receivables/{id}` | Update receivable |

Handler rules:

- Return `404` for a missing receivable.
- Return `409` for a stale rowversion.
- Return `422` with the form fragment for HTMX validation errors.
- Return `303` for successful normal POST.
- Return `HX-Redirect` for successful HTMX POST.
- Never expose raw SQL errors.

### F. Update templates

Reuse the create form with edit mode:

- Populate company select with the current company selected.
- Populate PO, amount, delivery date, and payment term.
- Include an encoded rowversion hidden input.
- Include a new idempotency key.
- Change heading to `Edit delivery receivable`.
- Change submit label to `Save changes`.
- Show the calculated due date and classification after the update.

List/grid layout:

- Company name
- PO number
- Amount
- Delivery date
- Due date
- Payment term
- Classification
- Edit action

The company name should be the primary identity in the row, with PO number as the secondary traceability field.

### G. HTMX and Alpine behavior

HTMX:

- Use standard form fallback attributes.
- Return form fragments for validation errors.
- Use `HX-Redirect` after successful update.
- Preserve the shared navigation and page shell.

Alpine.js:

- Use only for a local confirmation modal or unsaved-changes warning.
- Do not calculate due dates, balances, or classifications.
- Do not own the rowversion or authoritative form state.

## 5. Tests

### Domain/service tests

- Valid receivable update.
- Invalid PO, amount, date, or term.
- Due date recalculates after delivery date change.
- Due date recalculates after term change.
- Stale rowversion returns conflict.
- Protected lifecycle state is rejected.
- Idempotent replay returns the original result.
- Idempotency payload conflict is rejected.
- Audit failure rolls back the update.
- Existing payment date is never changed by the edit command.

### Repository tests with `sqlmock`

- List query returns company name from the join.
- Detail query returns company name.
- Update uses ID and rowversion predicates.
- Generated rowversion is excluded from update values.
- Zero affected rows maps to conflict.
- Transaction commit and rollback behavior.

### HTTP/template tests

- List row displays company name.
- Detail page displays company name.
- Edit page displays current values.
- Missing receivable returns `404`.
- Valid normal edit redirects.
- Valid HTMX edit returns `HX-Redirect`.
- Invalid HTMX edit returns a `422` fragment.
- Stale edit returns a safe `409` response.
- Status and company name render without JavaScript.

## 6. Acceptance Scenarios

1. Given a receivable, when the list loads, then its company name is visible.
2. Given a receivable detail page, then the company name and PO number are both visible.
3. Given an active unpaid receivable, when valid fields are edited, then the record is updated.
4. Given a changed delivery date or term, then the due date is recalculated.
5. Given a stale edit form, then newer data is not overwritten.
6. Given a successful edit, then exactly one immutable audit event is created.
7. Given the same edit command submitted twice with the same idempotency key, then only one update occurs.
8. Given a payment date, then the normal receivable edit form cannot change it.
9. Given an archived or cancelled receivable, then normal editing is blocked until the proper lifecycle action occurs.

## 7. Definition Of Done

- Company name appears in receivable list/grid and detail views.
- Edit routes and forms are implemented inside the receivables slice.
- Shared validation, money, and date helpers are used.
- Due date is recalculated when relevant input changes.
- `rowversion` prevents stale overwrites.
- Audit and idempotency are transactional.
- Payment date remains protected from normal edits.
- Full-page and HTMX flows work.
- Alpine.js remains presentation-only.
- Unit, repository, service, HTTP, and template tests pass.
- `go test ./...`, `go vet ./...`, and formatting pass.
- `analysis/progress.md` is updated.
