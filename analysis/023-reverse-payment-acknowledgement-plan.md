# Reverse Payment Acknowledgement Plan

## 1. Purpose

Allow an authorized user to correct a receivable that was incorrectly marked as `Payment Received` by returning it to the active, unpaid classification.

This plan covers correction of an acknowledgement error. It does **not** cover a refund, chargeback, returned cheque, payment cancellation by a bank, or any other movement of money after a real payment was received.

## 2. Current State

The receivable slice currently uses:

- `payment_date_utc` as the sole payment-state field.
- `lifecycle_status = 'Active'` for ordinary receivables.
- A non-null payment date to classify an active receivable as `Payment Received`.
- `row_version` for optimistic concurrency.
- Idempotency records for commands.
- `audit_events` containing before and after entity snapshots.

Therefore, a correction technically clears `payment_date_utc`; it must not change `lifecycle_status`, amount, tax snapshot, dates, or payment terms.

## 3. Challenge To The Proposal

“Un-acknowledge payment” is ambiguous and potentially unsafe.

- If the payment was never actually received and the acknowledgement was a data-entry mistake, a reversal is appropriate.
- If the payment was received and later returned, rejected, refunded, or charged back, clearing the date falsely makes the receivable look as though it was never paid. That needs a payment/reversal transaction model, not a toggle.
- If a paid receivable is edited after clearing the date, the original financial state can become difficult to reconstruct unless the audit event is retained and reviewed.
- A popup alone is not a control. Users can confirm the wrong operation, and a stale page can still reverse a newer payment unless the server enforces the original row version.

Recommended product wording:

> Reverse payment acknowledgement

Use helper text:

> This corrects an acknowledgement recorded by mistake. It does not record a refund or reverse money already received.

Require a reason and an appropriate permission. Do not expose this action to every user merely because the record is paid.

## 4. Recommended Business Rules

1. Only an `Active` receivable with a non-null payment date can be reversed through this correction flow.
2. The action requires an explicit confirmation dialog and a non-empty reason.
3. The reason must be stored in the audit event, not only displayed in the browser.
4. The server clears only `payment_date_utc` and updates the row timestamp/version.
5. Amount, gross amount, tax rule, tax amounts, delivery date, due date, payment terms, company, invoice, PO, and lifecycle status remain unchanged.
6. A stale row version returns `409` and does not modify the record.
7. Repeating the same request with the same idempotency key returns the original result without another update or audit event.
8. A different request using the same idempotency key returns an idempotency conflict.
9. A successful reversal creates one audit event with action `payment_acknowledgement_reversed`, the paid snapshot in `previous_values_json`, the unpaid snapshot in `new_values_json`, and the reason in a controlled audit payload or `note`.
10. The result is reclassified from `Payment Received` to `Pending`, `Near Due`, or `Overdue` using the existing due-date rules.
11. The record becomes eligible for normal edit and payment acknowledgement again, subject to the resulting active/unpaid state.
12. Cancelled and archived records cannot be reversed by this command.

## 5. Decisions Required Before Coding

These decisions materially affect the implementation:

- Who may reverse an acknowledgement: all admins, a collection supervisor, or a finance role?
- Is a reason mandatory, and what minimum/maximum length should it have?
- Should the original payment date be retained as a separate `acknowledged_payment_date_utc` field, or is immutable audit history sufficient for this correction-only workflow?
- Should reversal be allowed indefinitely, or only before period close / within a configurable time window?
- Is the intended feature actually a refund or chargeback workflow? If yes, stop this implementation and design a payment transaction model instead.
- Should a reversal require a second-person approval for financial controls?
- Should dashboard totals refresh immediately after the reversal? The expected answer is yes.

Recommended defaults for this application:

- Supervisor/admin permission only.
- Mandatory reason, trimmed and limited to 1-500 characters.
- No new date column for the first correction-only version; preserve the full paid state in the immutable audit event.
- No time limit until period-close behavior exists, but display a warning that closed-period corrections may require an accounting process.
- No second approval in the first slice unless the business already has role/approval infrastructure.

## 6. Domain and Command Changes

Add explicit errors in `internal/slices/receivables/domain.go`:

- `ErrPaymentAcknowledgementNotReversible` for non-active or unpaid records.
- `ErrInvalidReversalReason` or a `ValidationErrors` entry for a missing/invalid reason.

Add a command in `internal/slices/receivables/commands.go`:

```go
type ReversePaymentAcknowledgementCommand struct {
    ID               int64
    Reason           string
    OriginalVersion  []byte
    RequestID        string
    IdempotencyKey   string
    ActorID          string
}
```

Add `ReversePaymentAcknowledgement` to the service. Its transaction should:

1. Validate the idempotency key and reason.
2. Look up the idempotency record and replay the original result when applicable.
3. Load the receivable inside the transaction.
4. Require `Active` and a non-null payment date.
5. Update the payment date to `NULL` using ID, row version, active status, and `payment_date_utc IS NOT NULL` predicates.
6. Reload the entity.
7. Create the reversal audit event with actor, request ID, idempotency key, before/after snapshots, and reason.
8. Save the result entity ID to the idempotency record and commit.

Keep the repository method narrow:

```go
ReversePaymentAcknowledgement(ctx context.Context, db *gorm.DB, id int64, originalVersion []byte) (DeliveryReceivable, error)
```

The repository must not reuse the general receivable update method.

## 7. Persistence

No new table is required for the correction-only version because `payment_date_utc` is already nullable and audit snapshots already exist.

Add a migration only if a database constraint limits `audit_events.action` or if a required index is missing. Do not add a payment table merely to support clearing the existing date.

The update must change only:

- `payment_date_utc = NULL`
- `updated_at_utc`

The database update must include:

```text
delivery_receivable_id = ?
row_version = ?
lifecycle_status = 'Active'
payment_date_utc IS NOT NULL
```

Zero affected rows must be mapped to a safe conflict/protected-state response after the service reloads the record where possible.

## 8. HTTP and UI Flow

### Entry points

Add the reverse action on the paid receivable detail page first. Add it to list rows only after the detail flow is proven; placing a destructive financial correction in every row increases accidental activation risk.

The action should include the encoded current row version and a generated idempotency key.

### Confirmation popup

Use an Alpine.js modal, not `window.confirm()`.

The modal should:

- Have `role="dialog"`, `aria-modal="true"`, and a labelled heading.
- State the invoice and company being affected.
- State that the record will return to unpaid collection status.
- State that this is not a refund or bank reversal.
- Show the original payment date.
- Require a reason field.
- Provide `Cancel` and `Reverse acknowledgement` buttons.
- Support Escape, keyboard focus, and mobile layout.
- Use `x-cloak` to prevent an initialization flash.

Alpine may control only modal visibility and focus behavior. The server remains responsible for authorization, validation, state checks, and persistence.

Recommended confirmation copy:

> Reverse payment acknowledgement for invoice INV-001?

> This will remove the payment acknowledgement and return the receivable to its due-date-based unpaid status. It does not record a refund or reverse money already received.

### Request handling

Add:

```text
POST /receivables/{id}/payment-acknowledgement/reverse
```

For a normal request, redirect to the detail page with `303`. For HTMX, return `HX-Redirect` to the detail page. Return `409` for stale, protected, unauthorized, or already-changed records. Return `422` with the form/modal fragment for reason validation errors. Never expose raw database errors.

Do not rely on a success snackbar after `HX-Redirect` until the application has a redirect-safe flash mechanism. The destination page must visibly show the new unpaid status.

## 9. Audit and Reporting

The audit event is the minimum historical record for this correction flow. It should capture:

- Original payment date.
- Reversal timestamp.
- Actor ID.
- Reason.
- Original paid snapshot.
- Resulting unpaid snapshot.
- Request and idempotency identifiers.

Dashboard and receivable totals should update because the payment date becomes null. No amount or tax calculation should run during reversal.

If finance requires a complete payment ledger, audit snapshots are not enough. Replace this approach with a `receivable_payments`/payment-events design before implementation.

## 10. Tests

### Domain and service

- Active paid receivable can be reversed with a valid reason.
- Unpaid, cancelled, archived, and missing receivables are rejected.
- Missing and overlong reasons return field validation.
- Reversal clears only payment state and preserves all financial fields.
- Reversal classification follows the due date.
- Stale row version returns conflict.
- Concurrent acknowledgement/reversal cannot overwrite the newer state.
- Same idempotency key replays without a second update or audit event.
- Same key with a different payload returns idempotency conflict.
- Audit failure rolls back the database update.
- Actor/authorization policy is enforced by the service or authorization boundary.

### Repository

- The SQL update includes ID, row version, active status, and non-null payment date.
- The SQL update sets only payment date and timestamp.
- Zero affected rows returns conflict.
- The updated entity is reloaded.

### HTTP and templates

- Paid detail shows the reverse action.
- Unpaid, cancelled, and archived details do not show the reverse action.
- The modal includes the warning, original payment date, hidden row version, and idempotency key.
- Reason validation returns `422` and preserves entered text.
- Valid normal POST redirects with `303`.
- Valid HTMX POST returns `HX-Redirect`.
- Stale and protected requests return safe `409` responses.
- The detail page after success shows unpaid classification and restores the appropriate edit/acknowledge actions.
- The modal is keyboard accessible and does not submit when cancelled.

## 11. Implementation Order

1. Confirm correction-versus-refund scope, permission, reason, and period-close policy.
2. Add domain errors and reason validation tests.
3. Add the narrow repository reversal method and SQL tests.
4. Add the idempotent, audited, row-version-protected service command.
5. Add service tests for protected states, concurrency, replay, and rollback.
6. Add the reverse route and handler responses.
7. Add the paid-detail action and accessible Alpine confirmation modal.
8. Add HTTP/template tests and dashboard classification coverage.
9. Run `gofmt`, `go test ./...`, and `go vet ./...`.
10. Verify SQL Server behavior and browser keyboard/mobile behavior.

## 12. Definition Of Done

- An authorized user can correct a mistaken payment acknowledgement.
- Confirmation requires an explicit action and reason.
- The server prevents stale, duplicate, non-paid, non-active, and unauthorized reversals.
- The original paid state and reason remain auditable.
- Only `payment_date_utc` and the update timestamp change.
- Receivable and dashboard classifications move back to unpaid correctly.
- The feature is clearly not presented as a refund or chargeback.
- Automated tests and accessibility/browser verification pass.
