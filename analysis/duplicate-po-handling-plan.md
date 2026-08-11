# Duplicate PO Handling Plan

## 1. Goal

Prevent creating or editing a delivery receivable with a PO number that is already used by a receivable whose lifecycle status is not `Cancelled`.

The latest rule is:

```text
PO can be reused only when every existing receivable with that PO is Cancelled.
```

The system uses the spelling `Cancelled` for the lifecycle status.

## 2. Scope Assumption

This plan assumes PO uniqueness is **global across the system**, regardless of company account.

That means the following are blocked:

- Same PO with the same company and an active record.
- Same PO with a different company and an active record.
- Same PO with a paid record.
- Same PO with an archived record.

Reuse is allowed when all previous receivables using that PO are `Cancelled`.

If PO numbers are only unique per company, change every lookup and database index in this plan to use `(company_account_id, po_number)` instead of `po_number` alone.

## 3. Create Behavior

Before creating a receivable:

1. Trim the PO number.
2. Validate that it is non-empty and alphanumeric.
3. Normalize it for comparison, initially using uppercase ASCII.
4. Search for an existing receivable with the normalized PO and `lifecycle_status <> 'Cancelled'`.
5. If one exists, reject creation with a field-level PO error.
6. If no blocking match exists, continue with creation.

The duplicate check and insert must happen in the same transaction. Application validation alone is not sufficient because two requests can pass the check at the same time.

## 4. Edit Behavior

When editing a receivable:

- If the PO is unchanged, allow the update without treating the current row as a duplicate.
- If the PO changes, run the duplicate check against other receivables.
- Exclude the current `delivery_receivable_id` from the duplicate lookup.
- Block a change when another receivable with the normalized PO has a status other than `Cancelled`.
- If the current receivable is `Cancelled`, it may be edited only according to the lifecycle/reopen rules; its PO may be reused only when no other non-cancelled match exists.
- A paid or archived receivable remains protected by the existing edit rules.

## 5. Database Protection

Application checks provide a friendly error; the database constraint provides race-condition protection.

Add a new migration with:

- A normalized PO column, or a deterministic normalized lookup strategy.
- A filtered unique index for non-cancelled records.

Recommended schema direction:

```text
po_number_normalized VARCHAR(100) NOT NULL
```

Recommended index direction for global uniqueness:

```sql
CREATE UNIQUE INDEX UX_delivery_receivables_po_not_cancelled
ON dbo.delivery_receivables (po_number_normalized)
WHERE lifecycle_status <> 'Cancelled';
```

If SQL Server filtered-index limitations or existing data make this unsuitable, use a normalized active-PO registry table with one row per blocking PO. The registry row is created/deleted in the same transaction as the receivable lifecycle change.

Do not silently merge duplicate records.

## 6. Ordered Subtasks

### A. Confirm data and migration impact

1. Query existing PO values and normalize them.
2. Identify duplicate PO values with non-cancelled records.
3. Resolve existing conflicts before adding a unique filtered index. Do not silently cancel, merge, or delete historical records.
4. Decide whether the uniqueness scope is global or per company.
5. Create the migration for the normalized PO field/index.

Migration `0003` must fail safely when existing conflicts are found and must report the conflicting PO values for manual resolution.

### B. Add domain normalization

1. Add one shared `NormalizePONumber` function.
2. Trim whitespace.
3. Validate ASCII alphanumeric characters.
4. Normalize comparison value to uppercase.
5. Preserve the original entered value for display only if casing preservation is required; otherwise store the normalized value consistently.

### C. Add repository lookup

Add a slice-owned repository method:

```go
FindBlockingPONumber(
    ctx context.Context,
    db *gorm.DB,
    normalizedPO string,
    excludeReceivableID int64,
) (bool, error)
```

The query must:

- Match the normalized PO.
- Exclude `Cancelled` records.
- Exclude the current receivable ID for edits.
- Run inside the create/update transaction.

### D. Update create command

1. Normalize and validate the PO.
2. Check for a blocking duplicate.
3. Return `ErrDuplicatePO` mapped to the PO field.
4. Continue with idempotency, insert, and audit only when no duplicate exists.
5. Map a database unique-index violation to the same `ErrDuplicatePO` error.

### E. Update edit command

1. Normalize the submitted PO.
2. Compare it with the current normalized PO.
3. Skip duplicate lookup when unchanged.
4. Exclude the current receivable when changed.
5. Reject blocking duplicates before update.
6. Map a database race/unique violation to a user-safe duplicate error.
7. Write an audit event only for a successful update.

### F. Update templates and HTMX responses

- Show `PO number already exists on an active receivable.` beside the PO input.
- Also show the duplicate error through the reusable snackbar defined in `analysis/feedback-snackbar-plan.md`.
- Preserve the submitted form values.
- Return HTTP `422` with the form fragment for HTMX validation failures.
- Return the full form for normal requests.
- Do not reveal details of another company's receivable beyond the duplicate message.
- Keep Alpine.js out of duplicate validation; the server is authoritative.

### G. Add audit and idempotency tests

- Duplicate rejection must not create an audit event.
- Reusing a cancelled PO creates a new receivable and one audit event.
- A duplicate idempotent replay returns the original result.
- A different payload with the same idempotency key remains an idempotency conflict.
- A database unique conflict is mapped to the same user-facing PO error.

## 7. Test Matrix

| Scenario | Expected result |
|---|---|
| PO does not exist | Create succeeds |
| PO exists on `Active` record | Create rejected |
| PO exists on `Payment Received` record | Create rejected |
| PO exists on `Archived` record | Create rejected |
| PO exists only on `Cancelled` records | Create succeeds |
| Edit keeps the same PO | Update succeeds |
| Edit changes to another active PO | Update rejected |
| Edit changes to a PO used only by cancelled records | Update succeeds |
| Two concurrent creates use the same PO | Only one succeeds |
| Duplicate form submit repeats same idempotency key | Original result returned |
| PO includes spaces, hyphens, or symbols | Rejected according to alphanumeric rule |
| PO differs only by casing | Treated as duplicate after normalization |

## 8. Definition Of Done

- PO normalization is centralized and tested.
- Create and edit commands use the same duplicate rule.
- Current receivable is excluded from its own edit lookup.
- Non-cancelled duplicates are rejected.
- Cancelled PO reuse is allowed.
- Database race protection exists through a reviewed migration/index.
- Duplicate errors render correctly in full-page and HTMX forms.
- Duplicate rejection creates no audit event.
- Successful reuse/create/update creates exactly one audit event.
- Unit, repository, service, HTTP, and migration tests pass.
- `go test ./...`, `go vet ./...`, and formatting pass.
- The progress tracker is updated.

## 9. Open Decision

Confirm whether PO uniqueness is global across all companies or scoped to each company account. This plan currently assumes global uniqueness based on the phrase “already added to the system.”
