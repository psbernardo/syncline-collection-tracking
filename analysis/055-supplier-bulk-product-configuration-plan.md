# 055 Supplier Bulk Product Configuration Plan

## Goal

Allow an owner to configure multiple products for one supplier from a supplier-scoped page. The process creates a `supplier_products` relationship for each selected product with a default reference cost of `PHP 0.00`.

This is a master-data convenience workflow. It does not set a supplier price, choose a supplier for a transaction, or change existing supplier-product pricing.

## Current-state analysis

- Supplier master records are maintained in `internal/slices/suppliers`.
- Products are maintained in `internal/slices/products`.
- Supplier-product relationships currently have to be created one at a time at `POST /supplier-products`.
- The database already enforces one relationship per `(supplier_id, product_id)`.
- Existing relationship creation requires active supplier and active product records and accepts a non-negative scaled money value.
- The supplier list has edit actions, but no supplier-scoped product configuration page.
- The global supplier-product list displays relationships but is not a suitable bulk-selection workflow.

## MVP decisions

| Area | Decision | Reason |
|---|---|---|
| Entry point | Add `Configure products` from the supplier list/edit page | The operation is scoped to one supplier and should not require repeatedly selecting that supplier |
| Eligible products | Active products only | Inactive products must not enter new sourcing configuration under the existing master-data rules |
| Selection | Explicit multi-select with a `Select all visible` action | Avoid accidentally linking every product when the catalog grows |
| Default cost | Always `PHP 0.00` for newly created relationships | The request is to establish coverage before real supplier pricing is known |
| Existing relationships | Leave them unchanged, including cost, supplier SKU, active state, and timestamps | The operation must be safe to repeat and must never overwrite commercial data |
| Existing inactive relationship | Leave it unchanged and report it as already configured | Re-activating it silently would change an explicit business decision |
| Supplier state | The supplier must be active when the page is opened and submitted | New relationships cannot be created for an inactive supplier |
| Supplier SKU | Blank on bulk-created relationships | No supplier-specific code is supplied by this workflow |
| Partial failure | All-or-nothing transaction | A bulk configuration should not leave an unknown partial result |
| Audit | Record one bulk operation audit event plus relationship-level create audit events for rows inserted | The operation is traceable while each relationship retains the existing audit contract |
| Price confirmation | No confirmation per zero-cost row; show a prominent explanation and final summary | Zero is an intentional default in this workflow, unlike manual price entry where it may be accidental |

If the business intends to configure inactive products too, that must be an explicit follow-up decision because it conflicts with the current active-parent creation rule.

## User flow

1. The owner opens the supplier list or supplier edit page.
2. The owner selects `Configure products` for an active supplier.
3. The page displays the supplier identity and a searchable list of active products.
4. Products already linked to the supplier are marked `Configured` and are not selectable.
5. The owner selects one or more unconfigured products and submits `Configure selected products`.
6. The page explains that each new relationship starts with reference cost `PHP 0.00` and requires later review.
7. The server rechecks supplier and product state inside the write transaction.
8. On success, the page reports the number configured and the number skipped as already configured, then links to the supplier catalog and global supplier-product list.
9. The owner can edit each new relationship later to enter supplier SKU and reference cost.

Recommended route shape:

```text
GET  /suppliers/{id}/products
POST /suppliers/{id}/products
```

The POST body should contain selected product IDs, an idempotency key, and optionally a row/version token for the page. The supplier ID must come from the route and must not be trusted from a hidden form field.

## Page behavior

Add a supplier-scoped catalog page rather than placing a very large form on the supplier edit screen.

The page should show:

- Supplier name and active/inactive status.
- Count of active products available.
- Count already configured for this supplier.
- Search by product SKU or name.
- Product base UOM.
- Configured status and current reference cost for already-linked products.
- Checkboxes only for products with no existing relationship.
- Empty state when all active products are configured.
- Clear warning: `New relationships will start at PHP 0.00. Review the reference cost before using them for quotations.`

Do not post the complete product list as trusted business data. The selected IDs are only an input hint; the server must query and validate them again.

## Domain and application contract

Add a dedicated service operation, for example:

```go
type ConfigureProductsCommand struct {
    SupplierID     int64
    ProductIDs     []int64
    RequestID      string
    IdempotencyKey string
    ActorID        string
}

type ConfigureProductsResult struct {
    CreatedCount int
    SkippedCount int
    CreatedIDs   []int64
}
```

The command should:

- Reject a non-positive supplier ID or empty product selection.
- Normalize and deduplicate submitted product IDs before persistence.
- Require a non-empty idempotency key.
- Require the supplier to exist and be active.
- Require every selected product to exist and be active.
- Create each missing relationship with `ReferenceCost: money.Amount(0)`, blank supplier SKU, and active state.
- Treat an existing relationship as skipped, regardless of its active state.
- Return a stable result suitable for a success message.

Keep this operation in the supplier-product application boundary. Do not call the products repository repeatedly for each checkbox; use a dedicated bulk query/command.

## Repository and persistence approach

Extend the supplier repository with dedicated methods rather than composing the existing single-row `CreateProduct` call in a loop:

- Load the supplier-scoped product catalog with configured state and current cost.
- Insert missing `(supplier_id, product_id)` pairs in one transaction.
- Return inserted IDs and skipped count.

The SQL implementation should use a set-based `INSERT ... SELECT` or equivalent transaction-safe approach with `NOT EXISTS`. The existing unique index remains the final protection against duplicates. If concurrent requests race, convert a unique-key conflict into a safe retry or a clear conflict response; never report success for rows that were not committed.

The current schema needs no migration. `reference_cost_scaled = 0` already satisfies the check constraint, and `(supplier_id, product_id)` is already unique.

## Audit and idempotency

- Write the bulk operation audit record in the same transaction as relationship inserts.
- Include supplier ID, requested product IDs, created IDs, skipped count, and default cost in the operation audit payload.
- Preserve relationship-level `supplier_product create` audit events for each inserted row, or document an approved exception if event volume makes that impractical.
- Repeating the same request must not create duplicate rows or change existing values.
- Reusing an idempotency key after a committed request should return the original result according to the repository's existing idempotency convention. If no shared idempotency lookup exists yet, add one before exposing this as a retryable bulk action.

## Handler and template changes

- Add the two supplier-scoped routes to `Handler.RegisterRoutes`.
- Parse the route supplier ID strictly and return `404` for an unknown supplier.
- Add a page view model containing supplier, catalog rows, filters, errors, and selected IDs.
- Preserve selected IDs when validation fails.
- Use the existing multi-select visual language only where appropriate; this workflow needs per-product configured/disabled state and may be better represented by a checkbox table with Alpine filtering.
- Add a `Configure products` link only for active suppliers. Inactive suppliers can remain viewable but must not offer the action.
- Render a result summary after POST without requiring a second submission.
- Map inactive supplier/product, empty selection, duplicate/concurrent, and stale-state errors to useful page messages.

## Tests

### Domain/service tests

- Empty product selection is rejected.
- Duplicate submitted product IDs are deduplicated.
- Zero reference cost is applied to every newly created relationship.
- Existing active relationship is skipped without mutation.
- Existing inactive relationship is skipped without reactivation.
- Inactive supplier is rejected.
- Inactive product is rejected and no relationship is written.
- A failed batch does not leave partial relationships.
- Repeated execution is idempotent.

### Repository tests

- Catalog query returns active products and configured state.
- Catalog query includes current cost for configured rows.
- Bulk insert creates all missing rows with scaled cost `0`.
- Unique constraint/concurrent execution does not create duplicates.
- Transaction rollback removes all rows when one validation or insert fails.
- Audit rows are committed or rolled back with the relationship changes.

### Handler/template tests

- Active supplier exposes the configuration route/action.
- Inactive supplier does not expose a usable configuration action.
- GET renders configured products as unavailable and unconfigured products as selectable.
- Search/filter input is present and labels include SKU, name, and UOM.
- POST with no selection returns a field/page error.
- POST redirects or renders a summary with created and skipped counts.
- Invalid route and stale/inactive submissions are handled without a 500 response.

## Delivery sequence

1. Confirm that the desired scope is active products only and that existing inactive relationships must remain inactive.
2. Add catalog row/view-model types and a supplier-scoped read query.
3. Add the bulk configuration command and repository transaction using zero scaled cost.
4. Add audit and idempotency handling consistent with existing mutation flows.
5. Add GET/POST routes, templates, supplier list/edit links, and responsive selection UI.
6. Add domain, repository, handler, and template tests.
7. Run `gofmt`, `go test -count=1 ./...`, and `go vet ./...`.
8. Verify manually with: no configured products, some configured products, all configured products, repeated submission, inactive supplier, inactive product, and concurrent/retry behavior.
9. Update `analysis/000-implementation-ladder.md` with implementation evidence after verification.

## Acceptance scenarios

1. Given an active supplier and active products, when the owner selects three unconfigured products, then three active supplier-product rows are created with reference cost `PHP 0.00`.
2. Given one product is already configured with cost `PHP 125.00`, when the owner runs the process again, then its cost remains `PHP 125.00` and it is reported as skipped.
3. Given an inactive existing relationship, when the owner runs the process, then the relationship remains inactive and is not reactivated.
4. Given no products are selected, when the owner submits, then no data is written and a useful validation message is shown.
5. Given an inactive supplier, when the owner submits a stale form, then no relationships are created.
6. Given a selected product was deactivated after GET, when the owner submits, then the transaction is rejected or the product is excluded according to the chosen all-or-nothing policy, with no partial write.
7. Given the request is retried, when it is processed again, then no duplicate relationships are created and existing costs are unchanged.
8. Given a bulk operation succeeds, when an auditor inspects the audit data, then the operation and created relationships identify the supplier, products, actor, request, and default cost.

## Out of scope

- Entering a different cost per product during bulk configuration.
- Supplier SKU import or bulk SKU entry.
- Automatic supplier selection for quotations or purchase orders.
- Price history, effective dates, discounts, quantity breaks, or currency conversion.
- Configuring inactive products without a separate approved lifecycle rule.
