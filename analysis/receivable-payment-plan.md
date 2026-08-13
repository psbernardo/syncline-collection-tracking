# Full Payment Acknowledgment Plan

## 1. Goal

Allow a user to acknowledge that a delivery receivable was fully paid.

The existing nullable `payment_date_utc` column remains the source of truth. A receivable is classified as `Payment Received` when its payment date is non-null. No partial payments are supported.

## 2. Scope

### In scope

- A dedicated full-payment acknowledgment command.
- Recording the payment received date.
- Server-side validation of the payment date.
- Transactional audit event.
- Idempotent repeated submissions.
- Optimistic concurrency protection.
- Payment acknowledgment from the receivable list and detail pages.
- Displaying the payment date after acknowledgment.
- Refreshing receivable status and dashboard totals after success.
- Hiding normal financial edit actions for paid receivables.

### Out of scope

- Partial payments.
- Payment allocations or installments.
- Payment references, bank reconciliation, or proof uploads.
- Refunds, reversals, or reopening paid receivables.
- A separate `receivable_payments` table.
- Changing the persisted `lifecycle_status` column.

## 3. Business Rules

1. Only an `Active` receivable can be acknowledged as paid.
2. The receivable must not already have a payment date.
3. The payment amount is always the full `AmountDue`.
4. The payment received date is required.
5. The payment date cannot be earlier than the delivery date.
6. The payment date cannot be later than the current Philippines business date.
7. A successful acknowledgment sets `payment_date_utc` once.
8. A paid receivable is classified as `Payment Received` regardless of its due date.
9. A paid receivable cannot be changed through the normal receivable edit form.
10. Repeating the same request with the same idempotency key returns the original result without another update or audit event.
11. A stale request must not overwrite a newer payment or receivable edit.

The future date rule should be enforced by the service, not only by the database. The database already enforces the lower bound against the delivery date.

## 4. Existing Database Contract

No base-table migration is required because the schema already contains:

- `payment_date_utc DATETIME2(0) NULL`.
- `CK_delivery_receivables_payment_date`.
- `IX_delivery_receivables_paid_date`.
- The existing `row_version` column.
- The existing audit and idempotency tables.

If the payment command needs a new audit action constraint or lookup index, add a separate migration only for that concrete database requirement. Do not add a payment table for this full-payment-only scope.

## 5. Domain and Commands

Add a dedicated command in `internal/slices/receivables/commands.go`:

```go
type ReceivePaymentCommand struct {
    ID               int64
    PaymentDate     string
    OriginalVersion  []byte
    RequestID        string
    IdempotencyKey   string
    ActorID          string
}
```

Add a service method:

```go
func (service *service) ReceivePayment(ctx context.Context, command ReceivePaymentCommand) (DeliveryReceivable, error)
```

Add explicit domain errors as needed:

- `ErrAlreadyPaid`.
- `ErrPaymentNotAllowed` for non-active records.
- `ErrInvalidPaymentDate` or a field validation entry for invalid dates.

Use the shared `businessdate` helper to parse the date and compare it with the service clock in the `Asia/Manila` business-date contract. Store the normalized UTC date consistently with existing delivery and due dates.

## 6. Transaction Flow

The service should execute the following in one database transaction:

1. Validate the idempotency key.
2. Parse and validate the payment date.
3. Check the idempotency record.
4. Load the receivable using the transaction.
5. Reject missing, non-active, already-paid, or stale records.
6. Update only `payment_date_utc` and `updated_at_utc`, using ID and row version.
7. Reload the updated receivable.
8. Write one audit event:
   - Entity type: `delivery_receivable`
   - Action: `payment_received`
   - Previous values: unpaid receivable
   - New values: paid receivable including payment date
   - Actor, request ID, and idempotency key
9. Store the result entity ID in the idempotency record.
10. Commit.

The repository should add a narrow method rather than reuse the general financial-field update:

```go
MarkPaymentReceived(ctx context.Context, db *gorm.DB, id int64, paymentDate time.Time, originalVersion []byte) (DeliveryReceivable, error)
```

The SQL update must include `payment_date_utc IS NULL` and the supplied `row_version` in the `WHERE` clause. Zero affected rows maps to a conflict or protected-state error after reloading the record.

## 7. HTTP Routes and Handlers

Add:

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/receivables/{id}/payment` | Acknowledge full payment |

The handler should:

- Parse only request values.
- Decode the row version.
- Build `ReceivePaymentCommand`.
- Use `local-admin` as the current actor, matching existing commands.
- Return `303` to the detail page for normal requests.
- Return `HX-Redirect` to the detail page for successful HTMX requests.
- Return `422` with the payment form fragment for validation errors.
- Return a safe `409` response for stale, already-paid, or protected records.
- Return `404` for a missing receivable.
- Expose no raw SQL errors.

Generate a new idempotency key when rendering the acknowledgment form.

## 8. UI Changes

### List page

Update `receivable-row.html` and its view model to:

- Show `Acknowledge payment` only for active unpaid records.
- Include the current row version in the action form.
- Show `Payment received: YYYY-MM-DD` for paid rows.
- Replace or hide `Edit` for paid, cancelled, and archived records.

Use a small inline HTMX form or a confirmation disclosure. The server remains authoritative; JavaScript must not calculate or persist payment state.

### Detail page

Update `detail.html` to:

- Show the payment date when classification is `Payment Received`.
- Show an `Acknowledge full payment` action only for active unpaid records.
- Display the full amount being acknowledged.
- Hide `Edit receivable` when the record is paid, cancelled, or archived.
- Show a clear protected-state message when an action is no longer available.

### Payment form

Add a partial such as `partials/payment-form.html` containing:

- Hidden idempotency key.
- Hidden row version.
- Required payment date input.
- Full receivable amount display.
- Submit and cancel controls.
- Field-level validation output.

The form should default the payment date to the current Philippines business date when rendered. The submitted value remains server-validated.

### Dashboard and company summaries

No aggregation query change is expected. Once `payment_date_utc` is written:

- The receivable leaves Pending, Near Due, or Overdue.
- It appears in Payment Received.
- Its full amount is removed from outstanding totals.
- Global and company summary cards reflect the change after refresh.

After successful HTMX acknowledgment, refresh the relevant receivable result and dashboard summary when the dashboard is present. Full-page redirects must work without JavaScript.

### Status filter

Retain the existing `payment_received` filter. Improve the filter summary label to display `Payment received` instead of the internal value `payment_received`.

## 9. View Model Changes

Extend `ReceivableViewModel` with only the fields needed by the UI:

- `PaymentDate` formatted for display.
- `CanReceivePayment`.
- `CanEdit`.
- `RowVersion` encoded for action forms.

Derive these values server-side from the domain entity. Do not let templates infer permissions from raw lifecycle strings.

## 10. Tests

### Domain tests

- Payment date equal to delivery date is valid.
- Payment date before delivery date is rejected.
- Future payment date is rejected.
- Paid classification takes precedence over overdue status.

### Service tests

- Active unpaid receivable can be marked fully paid.
- Payment date is persisted and returned.
- Non-active receivable is rejected.
- Already-paid receivable is rejected.
- Invalid date returns field validation.
- Stale row version returns conflict.
- Same idempotency key and payload performs one update.
- Same idempotency key with a different payload returns idempotency conflict.
- Audit failure rolls back the payment update.
- Payment update never changes amount, due date, delivery date, or lifecycle status.

### Repository tests

- Payment update includes ID, row version, and `payment_date_utc IS NULL`.
- Zero affected rows maps to conflict.
- Updated record is reloaded.
- Payment date is selected in list and detail projections.

### HTTP and template tests

- Active unpaid detail shows the payment action.
- Paid detail shows the payment date and no payment action.
- Paid/cancelled/archived records do not show normal edit actions.
- Valid normal POST redirects with `303`.
- Valid HTMX POST returns `HX-Redirect`.
- Invalid HTMX POST returns `422` with the payment form.
- Already-paid and stale requests return safe `409` responses.
- List row shows payment date after acknowledgment.
- Status filters still return paid records.
- Dashboard totals move the full amount to Payment Received.

## 11. Implementation Order

1. Add payment-date parsing and business-date validation tests.
2. Add command errors, command type, and service transaction.
3. Add the narrow repository payment-update method.
4. Add service, repository, and audit/idempotency tests.
5. Add the payment route and handler responses.
6. Add payment form and list/detail action templates.
7. Add payment date, action permissions, and encoded row version to view models.
8. Hide protected edit actions and improve status filter labels.
9. Add HTTP and template coverage.
10. Run `gofmt`, `go test ./...`, and `go vet ./...`.
11. Perform SQL Server and browser verification, including dashboard movement after acknowledgment.

## 12. Definition Of Done

- A user can acknowledge a receivable as fully paid from the application.
- Only one full payment can be recorded for a receivable.
- `payment_date_utc` is the only payment state persisted.
- The payment command is transactional, audited, idempotent, and concurrency-safe.
- Paid records cannot be edited through the normal receivable form.
- List and detail views display the payment date and correct available actions.
- Dashboard and company totals classify the full amount as Payment Received.
- Full-page and HTMX flows work without requiring JavaScript.
- No partial-payment behavior or payment table is introduced.
- Automated tests and SQL Server verification pass.
