# Receivable Invoice Selection Plan

## Goal

Change the invoice number field in the Receivables add/edit forms from free text to a selection control whose options come from `dbo.invoices`, while preserving existing Receivable records and preventing incorrect invoice-to-receivable associations.

## Current-State Findings

- `delivery_receivables.invoice_number` is currently a required free-text `VARCHAR(100)` value.
- `NewDeliveryReceivableWithTax`, `CreateReceivableCommand`, `UpdateReceivableCommand`, the handler, templates, list filters, audit payloads, and duplicate checks all use that text value.
- The current add and edit routes already exist: `GET /receivables/new`, `POST /receivables`, `GET /receivables/{id}/edit`, and `POST /receivables/{id}`.
- The Receivables handler currently receives only an account repository. It will need an invoice option provider as well.
- `dbo.invoices` is now introduced by migration `0020` and has `invoice_id`, unique `invoice_number`, `company_account_id`, `status` (`POSTED` or `VOIDED`), dates, totals, and immutable Sales Order snapshots.
- No foreign key currently connects `delivery_receivables` to `invoices`.
- Existing Receivables may contain invoice numbers that have no matching row in `dbo.invoices`; they must not be deleted or assigned an invented invoice.
- The current Receivables invoice uniqueness index applies to the old text field. It does not establish that the selected value is a real invoice.

## Recommended Data Model

Add a nullable relationship rather than replacing the existing column immediately:

```sql
ALTER TABLE dbo.delivery_receivables
ADD invoice_id BIGINT NULL;

ALTER TABLE dbo.delivery_receivables
ADD CONSTRAINT FK_delivery_receivables_invoice
FOREIGN KEY (invoice_id) REFERENCES dbo.invoices(invoice_id);
```

Keep `invoice_number` during the transition as a legacy/display snapshot. The recommended long-term invariant is:

- New Receivables must have a valid `invoice_id`.
- `invoice_number` for new records is copied from the selected invoice and is not user-editable.
- Existing unresolved records may temporarily have `invoice_id IS NULL` and retain their original `invoice_number`.
- Once all legacy rows are resolved, make `invoice_id NOT NULL`, remove the old Receivable invoice-number uniqueness/index rules if they are no longer needed, and use the joined invoice number as the authoritative display value.

Do not use `invoice_number` alone as the relationship. The invoice ID is the stable foreign key; the number remains a document identifier and snapshot.

## Invoice Option Rules

Confirm these rules before coding; the plan assumes the following defaults:

- Show only `POSTED` invoices in the selectable list.
- Do not allow `VOIDED` invoices to be selected for a new Receivable.
- Show invoice number plus customer/company and invoice date to disambiguate options.
- Prefer invoices that are not already linked to another Receivable.
- Scope options by the selected company account, or derive the company from the selected invoice. Do not allow a Receivable company and invoice company to disagree.
- Preserve the current company field only if the business wants users to select company first. Otherwise selecting an invoice should set the company and make it read-only.
- Define whether one invoice may have multiple Receivables. The recommended default is one invoice to one Receivable, enforced by a filtered unique index on `delivery_receivables.invoice_id` for non-cancelled rows.

The option query should return a small read model, for example:

```text
InvoiceOption { ID, Number, CompanyAccountID, CompanyName, InvoiceDate, Total }
```

Use a server-side query/provider rather than embedding invoice data in JavaScript. If the invoice count becomes large, use a searchable HTMX endpoint instead of loading every invoice into the form.

## Existing-Data Handling

### Phase 1: Inventory before migration

Run a read-only report in the target database before applying the relationship migration:

- Total Receivables.
- Receivables grouped by `invoice_number` and lifecycle status.
- Exact matches between `delivery_receivables.invoice_number` and `invoices.invoice_number`.
- Matching rows where Receivable company differs from invoice company.
- Invoice numbers with no matching invoice.
- Multiple Receivables referring to the same invoice number.
- Receivables whose invoice number is the legacy default `0000` or otherwise invalid.
- Existing invoices already linked to a Receivable, if any relationship exists outside the schema.

Do not infer a match from amount, PO, customer name, or date alone. Only an exact, approved invoice-number match is safe for automatic backfill.

### Phase 2: Add nullable relationship and safe backfill

Create migration `0021` after `0020`:

1. Add nullable `invoice_id`.
2. Add the foreign key to `dbo.invoices`.
3. Backfill `invoice_id` only for exact invoice-number matches with an unambiguous company match.
4. Leave unmatched, ambiguous, duplicate, and company-mismatch rows as `NULL`.
5. Record/report the unresolved IDs and counts. Fail the migration only for schema errors, not because business data requires review.
6. Add an index on `invoice_id`.

Do not automatically modify legacy `invoice_number`, create invoice records, merge Receivables, or assign a guessed invoice.

### Phase 3: Reconcile unresolved rows

Provide an operational reconciliation procedure or admin-only workflow:

- Select the correct existing invoice for each unresolved Receivable.
- Validate company compatibility and one-to-one rules.
- Update `invoice_id` and synchronize the snapshot number.
- Write an audit event identifying the old value, selected invoice ID/number, actor, and reason.
- Keep a documented exception path for Receivables that were intentionally created without a system Invoice.

Until reconciliation is complete, existing unresolved records must remain viewable and payable. Their edit form should either show a clearly labelled legacy value with a required migration choice, or prevent ordinary edits with a safe message. Do not silently submit a blank selection.

### Phase 4: Tighten constraints

After the unresolved count reaches zero and the business approves the invariant:

- Make `invoice_id` `NOT NULL`.
- Add the filtered unique index if one invoice may have only one non-cancelled Receivable.
- Remove or retain the old invoice-number index based on whether the snapshot column remains authoritative.
- Consider renaming the old column to make its snapshot/legacy role explicit, but only as a separate migration after code has stopped treating it as user input.

## Application Changes

### Invoice repository/read model

Add a method to the invoice repository or a dedicated Receivable option query, such as:

```go
ListSelectableOptions(ctx context.Context, companyAccountID int64, excludeReceivableID int64) ([]InvoiceOption, error)
```

The query must:

- Join `invoices` to `company_accounts`.
- Filter by the approved invoice status.
- Exclude invoices already assigned to another blocking Receivable.
- Exclude the current Receivable only when editing.
- Return the current invoice even if it is now non-selectable, so an edit form can display the existing association safely, or return a specific stale/voided-state error.
- Use bound parameters and stable ordering by invoice date/number/ID.

Wire the provider into `receivables.NewService`/`NewHandler` without making the Receivables slice depend on invoice GORM models. Use a small interface and option type.

### Domain and commands

Add `InvoiceID int64` to the Receivable domain and create/update commands. Keep `InvoiceNumber` only as an internal snapshot/legacy input during the transition.

On create and update:

1. Validate that an invoice was selected.
2. Load the invoice inside the transaction.
3. Reject missing, voided, incompatible-company, or already-linked invoices.
4. Copy the invoice number, company account ID, payment terms, due date, and amount fields according to the approved ownership rule.
5. Decide explicitly whether Receivable-specific delivery date, tax, amount, and payment term edits remain allowed. The safer default is that values derived from an immutable posted invoice are read-only; Receivable payment state remains owned by Receivables.
6. Persist `invoice_id` and the copied `invoice_number` together.

The invoice lookup and final update must be in the same transaction. A preloaded dropdown is only a usability feature; it is not authorization or concurrency protection.

Keep row-version, idempotency, duplicate PO, lifecycle protection, audit, and rollback behavior from the existing edit flow. Change audit payloads to include both invoice ID and invoice number.

### Handler and templates

Replace the invoice text input in `internal/slices/receivables/templates/partials/receivable-form.html` with a searchable select or native select using `InvoiceOption` values.

- Create form: require a selectable invoice.
- Edit form: select the current invoice.
- Legacy unresolved edit: show the preserved legacy number and require an explicit invoice mapping, or block editing with an actionable reconciliation message.
- Validation errors must preserve the selected ID and submitted form values.
- For HTMX, return the same `422` form fragment behavior already used by Receivables.
- Do not trust a hidden invoice number; submit the invoice ID and resolve the number server-side.

Update list/detail links and labels to use the joined invoice number, and add a link to invoice detail when the relationship exists.

### Server wiring

Construct the invoice repository before the Receivables handler or expose a small invoice-options adapter from the invoice slice. Avoid importing invoice persistence models into Receivables. Update handler/service test fakes and constructors accordingly.

## Migration and Rollout Sequence

1. Confirm invoice eligibility, one-to-one versus many-to-one association, company behavior, and whether invoice-derived financial fields are editable.
2. Run and review the inventory report against development/staging/production data.
3. Add migration `0021` with nullable `invoice_id`, foreign key, index, and conservative exact-match backfill.
4. Deploy read support that displays linked invoices while retaining legacy fallback display.
5. Deploy the new create form requiring invoice selection.
6. Reconcile unresolved legacy rows using approved mappings and audit events.
7. Deploy edit behavior, including the explicit unresolved-row path.
8. Verify counts and referential integrity, then apply the final non-null/unique constraint migration only after approval.

If a rollback is required before final tightening, remove the new UI requirement first, retain `invoice_id` and backfilled data, and do not restore free-text behavior in a way that creates new orphan values.

## Tests

### Migration/data tests

- Migration `0021` is registered once after `0020`.
- Exact matching backfills the correct invoice ID.
- No-match, ambiguous, default `0000`, and company-mismatch rows remain unresolved.
- Backfill is idempotent and does not alter legacy invoice numbers.
- Foreign-key and final uniqueness constraints are created only in their approved phase.

### Domain/service tests

- Create accepts a valid posted invoice.
- Create rejects missing, voided, mismatched-company, and already-linked invoices.
- Update can retain the current invoice without treating it as a duplicate.
- Update cannot switch to an invoice linked to another Receivable.
- Submitted invoice number cannot override the invoice record.
- Existing unresolved records follow the approved mapping/blocking behavior.
- Payment, row-version, idempotency, audit, and rollback behavior remains unchanged.

### Repository tests

- Option query filters status and linked Receivables.
- Current edit invoice is included when appropriate.
- Company filtering and stable ordering are correct.
- Create/update persists `invoice_id` and snapshot number together.
- Detail/list queries return invoice relationship data.

### HTTP/template tests

- New form renders invoice options and no free-text invoice input.
- Edit form selects the current invoice.
- Invalid selection returns field-level error and preserves the selection.
- Legacy unresolved behavior is clear and safe.
- Successful normal and HTMX create/edit flows redirect correctly.
- Invoice detail link appears for linked Receivables.

## Acceptance Scenarios

1. A user opening Add Receivable sees invoice options sourced from system invoices, not manually typed numbers.
2. Selecting an invoice cannot associate it with a different company.
3. A voided or already-linked invoice cannot be newly selected.
4. Existing Receivables remain visible and payable after deployment.
5. Existing exact invoice-number matches are linked automatically only when unambiguous and company-compatible.
6. Existing unmatched records are reported and are never assigned guessed invoices.
7. Editing a linked Receivable shows its current invoice and cannot replace it with a conflicting invoice.
8. Concurrent submissions cannot associate the same invoice twice when one-to-one is approved.
9. Invoice number changes in the invoice system cannot silently change the Receivable's historical snapshot or audit record.

## Decisions Required Before Implementation

- Are only `POSTED` invoices selectable, or are `VOIDED` invoices also valid historical associations?
- May one invoice have multiple Receivables, or is the relationship one-to-one?
- Should company be selected separately, or derived/locked from the selected invoice?
- Which Receivable fields remain editable after selecting an immutable invoice?
- Must every old Receivable eventually map to an Invoice, or are legacy/manual Receivables a supported permanent exception?
- Should unresolved legacy rows be editable through a mapping workflow or temporarily read-only?
- Should the current invoice-number uniqueness index be retained during and after migration?

## Definition Of Done

- Add and edit use invoice options sourced from `dbo.invoices`.
- Server-side validation resolves and verifies invoice IDs; submitted display numbers are not trusted.
- Existing Receivables are preserved, inventoried, and either safely linked or explicitly left as documented legacy exceptions.
- No guessed, deleted, or silently overwritten invoice data is introduced.
- Foreign-key and approved cardinality constraints protect new associations.
- Full-page, HTMX, concurrency, audit, migration, and legacy-data tests pass.
- `go test ./...`, `go vet ./...`, and formatting pass.
- The implementation ladder and progress documentation are updated after the feature is delivered.
