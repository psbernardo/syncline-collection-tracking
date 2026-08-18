# 031 Trading Master Data Plan

## Epic E-02: Products, Suppliers, and Supplier Reference Pricing

This plan is the detailed implementation contract for the master-data step in the trading MVP ladder. It expands the original three PBIs without adding supplier automation, price history, purchasing transactions, or accounting behavior.

## 1. Current-plan analysis

The existing plan correctly identifies the required capabilities:

- A product master for items that may be quoted and ordered.
- A supplier master for parties used in sourcing and supplier purchase orders.
- A supplier-product relationship carrying supplier-specific identity and a current reference cost.
- Active/inactive controls so obsolete records cannot enter new work.
- A distinction between reference pricing and historical transaction cost.

The following details must be made explicit before development:

| Area | MVP decision | Reason |
|---|---|---|
| Product identity | SKU is unique among products; product name is not unique | SKU is the stable internal reference used by RFQs and quotations |
| Product UOM | Each product has one base UOM | Downstream quantities and supplier costs need one comparable unit |
| Supplier-product uniqueness | One relationship per supplier and product | The existing pricing schema uses this rule; multiple rows for the same pair would make the current price ambiguous |
| Supplier UOM conversion | Not supported in the first implementation unless a real supplier requires it | Conversion introduces pack-size, rounding, and price-normalization rules that are not needed for the current MVP workflow |
| Reference price | One current non-negative unit cost per supplier-product relationship | The next quotation can use it as a starting reference without pretending it is an agreed or historical cost |
| Price history | Not stored as a separate history table in this MVP | Commercial transactions must snapshot their source cost; a full price audit/versioning model is deferred |
| Currency | PHP only | This matches the existing money boundary and MVP scope |
| Deletion | No hard delete after use; deactivate/archive instead | Existing financial and trading plans require historical references to remain valid |
| Audit | Material create, edit, deactivate, and price-change actions are audited | A current price can change without changing prior quotation, PO, or receipt snapshots |

The original phrase "conversion factor if needed" is therefore a readiness decision, not an implementation default. If supplier pack pricing is required before RFQ development, stop and define the conversion contract before building this slice.

## 2. Scope and non-goals

### Included

- Create, list, view, and edit products.
- Activate and deactivate products.
- Create, list, view, and edit suppliers.
- Activate and deactivate suppliers.
- Link an active supplier to an active product.
- Record supplier SKU and current reference unit cost.
- Change the current reference cost with validation and audit history.
- Search and filter products, suppliers, and relationships by identity and active state.
- Use only active products, suppliers, and supplier-product relationships in new RFQs, quotations, and supplier allocations.

### Excluded

- Automatic supplier selection or lowest-price selection.
- Supplier portal or supplier price import.
- Price validity periods, effective dates, or historical price versions.
- Multi-currency or foreign-exchange rates.
- Contract pricing, quantity breaks, discounts, rebates, and negotiated price approval.
- Supplier delivery cost; that belongs to procurement/transaction records.
- Inventory, stock balances, warehouse locations, and catalog categories.
- Product variants or bundles.
- Hard deletion of records referenced by a transaction.

## 3. PBI 01: Product master

**Goal:** Maintain the products that can be requested, quoted, allocated, and ordered.

### Data contract

| Field | Rule |
|---|---|
| Product ID | Generated `BIGINT` identifier |
| SKU | Required, trimmed, case-insensitive unique value; stable after creation unless an explicit audited correction is supported |
| Name | Required, trimmed, human-readable product name |
| Description | Optional text |
| Base UOM | Required controlled value, such as `PC`, `BOX`, or `SET`; stored consistently in uppercase |
| Active state | New products are active; inactive products remain visible in history but cannot enter new transactions |
| Created/updated timestamps | UTC `DATETIME2` values |
| Row version | Required for optimistic concurrency on edits |

The product UOM is the unit used by RFQ quantities, quotation lines, supplier reference unit cost, and later transaction snapshots. Do not accept free-form UOM text in every line.

### Behavior

- Create validates required fields, SKU uniqueness, and the approved UOM value.
- Edit permits identity and descriptive corrections while the product is not locked by an accepted or posted transaction; otherwise use an audited correction policy.
- Deactivate prevents selection in new RFQs and quotations but does not alter existing lines or snapshots.
- A product with supplier relationships or transaction references is never hard-deleted.
- Product lists show active products by default and provide an explicit inactive filter.

### Implementation tasks

- Add `internal/slices/products` following the existing vertical-slice structure.
- Add domain commands and validation tests before repository implementation.
- Add repository interfaces, GORM implementation, query filtering, and SQL mock tests.
- Add create/edit/list/view handlers and server-rendered templates.
- Add migration `0007` or the next unapplied migration after the current registry, with explicit SQL Server types, constraints, indexes, and rollback.
- Register the slice in the server composition root and navigation only if the existing navigation plan requires it.

## 4. PBI 02: Supplier master

**Goal:** Maintain suppliers used for sourcing and supplier purchase orders.

### Data contract

| Field | Rule |
|---|---|
| Supplier ID | Generated `BIGINT` identifier |
| Supplier name | Required, trimmed; duplicate names are allowed unless the business later defines a legal identity key |
| Contact person | Optional in the trading MVP; display if provided |
| Contact number | Optional, stored as text |
| Email | Optional; validate format only when supplied |
| Billing address | Optional in the master; required only if supplier PO issuance later needs it |
| Delivery address | Optional in the master; supplier-specific delivery details are not yet modeled |
| Tax/business identifier | Optional text; do not assume uniqueness without an approved business rule |
| Active state | New suppliers are active; inactive suppliers cannot be selected for new procurement |
| Created/updated timestamps | UTC `DATETIME2` values |
| Row version | Required for optimistic concurrency on edits |

The supplier master stores identity and contact data, not supplier performance, payment terms, delivery charges, or transaction status. Those belong to later procurement capabilities or transaction snapshots.

### Behavior

- Create validates the required supplier name and normalizes whitespace.
- Edit preserves the supplier ID and relationship references.
- Deactivate removes the supplier from new procurement selectors while retaining existing relationships and purchase orders.
- A supplier with relationships or transaction references is never hard-deleted.
- Supplier lists show active suppliers by default and provide search and an inactive filter.

### Implementation tasks

- Add `internal/slices/suppliers` with domain, commands/queries, repository, handlers, templates, and tests.
- Reuse the product-slice patterns for active-state filtering, row-version conflict handling, validation errors, and audit events.
- Add supplier schema and indexes in the same migration only if the migration remains independently reviewable; otherwise use a separate sequential migration.
- Confirm the future supplier PO requires any currently optional billing, delivery, or tax fields before PBI 034 starts.

## 5. PBI 03: Supplier-product relationship and current reference price

**Goal:** Record which supplier can provide which product and the current supplier reference unit cost used to prepare future quotations.

### Data contract

| Field | Rule |
|---|---|
| Supplier-product ID | Generated `BIGINT` identifier |
| Supplier ID | Required foreign key to `suppliers` |
| Product ID | Required foreign key to `products` |
| Supplier SKU | Optional supplier-specific code, trimmed text |
| Current reference unit cost | Required non-negative PHP amount, stored using the shared integer-scaled money representation |
| Active state | New relationships are active only when both master records are active |
| Updated timestamp | UTC timestamp for the latest relationship or price change |
| Row version | Required for concurrent edits |

Enforce a unique constraint on `(supplier_id, product_id)`. The current reference price belongs to the relationship, not to the product, because different suppliers may quote different costs for the same product.

The reference unit cost is understood to be the cost for one product base-UOM unit. A supplier pack UOM or conversion factor must not be added silently. If required, add an approved follow-up contract containing pack UOM, conversion direction, decimal precision, rounding behavior, and the normalized unit-cost formula.

### Behavior

- Create requires active supplier and product records and a valid non-negative reference cost.
- Duplicate supplier-product creation is rejected; the user edits the existing relationship instead.
- Edit allows supplier SKU, active state, and current reference cost changes subject to row-version checks.
- Price changes update the current value and create one audit event containing the previous and new scaled amounts.
- Deactivating the relationship prevents it from being selected for new quotation sourcing or allocation.
- Deactivating a supplier or product makes its relationships unavailable for new work without rewriting relationship history.
- A zero reference price is technically valid for a no-cost item, but the UI should require explicit confirmation rather than silently defaulting to zero.
- Negative prices, invalid money scale, missing foreign keys, and inactive parent records are rejected.
- No supplier is chosen automatically because current reference cost is informational and may be stale.

### Current-price semantics

The current reference price is not:

- A supplier commitment or accepted quotation.
- A purchase-order price.
- The actual received cost.
- A historical price record.
- A price including supplier delivery cost, tax, commission, or customer markup.

When later slices create a quotation, supplier PO, or receiving record, they must copy the applicable cost into a transaction-owned snapshot. Future edits to this relationship must not change those historical amounts.

### Implementation tasks

- Add the relationship to the supplier-product slice or create a small `supplierproducts` slice; keep product and supplier master ownership separate from relationship commands.
- Add selectors for active supplier-product relationships, including supplier, product, supplier SKU, and current reference cost.
- Add repository queries for product sourcing and supplier catalog views.
- Add domain tests for duplicate relationships, inactive parents, inactive relationship selection, zero/negative prices, and money conversion.
- Add repository tests for foreign-key failures, unique-key failures, filtered active queries, and row-version conflicts.
- Add handler tests for create/edit validation, duplicate error display, price change feedback, and optimistic-concurrency failures.
- Add migration tests verifying tables, foreign keys, check constraints, unique keys, row versions, and indexes.

## 6. Persistence contract

Use explicit SQL Server schema names and the repository's existing conventions:

```text
dbo.products
dbo.suppliers
dbo.supplier_products
```

Recommended minimum constraints:

- Primary keys on each generated ID.
- Foreign keys from `supplier_products` to both master tables.
- Unique case-insensitive SKU key for products.
- Unique `(supplier_id, product_id)` key for supplier products.
- Check constraints for active-state values and non-negative reference cost.
- `ROWVERSION` on editable master and relationship rows.
- UTC created/updated timestamps.

Recommended query indexes:

- Products by active state, name, and ID; SKU lookup.
- Suppliers by active state, name, and ID.
- Supplier products by product and active state for sourcing.
- Supplier products by supplier and active state for supplier catalog views.
- Include only columns justified by the actual list/select query.

Money must use `internal/shared/money`; do not use `DECIMAL` in application models or floating-point arithmetic. The database should store the same scaled integer convention used by receivables and future trading transactions.

## 7. Cross-slice rules

- Product and supplier master data are reference data owned by this epic.
- RFQ creation may select active products only.
- Quotation sourcing may select active supplier-product relationships whose supplier and product are active.
- Procurement allocation may select only active relationships at the time of allocation.
- Existing quotations, Sales Orders, supplier POs, and received costs own their snapshots and remain readable after master-data deactivation.
- Master-data repositories must not call RFQ, quotation, or receivable repositories directly. Use application-level ports or commands at integration boundaries.
- Audit events use the foundation audit contract and are committed in the same transaction as the master-data mutation.

## 8. Acceptance scenarios

1. Given a unique SKU and valid UOM, when the owner creates a product, then it appears in the active product list.
2. Given a duplicate SKU ignoring case and surrounding whitespace, when the owner creates a product, then creation is rejected without a second product.
3. Given an inactive product, when the owner creates an RFQ, then that product is unavailable for selection.
4. Given valid supplier identity data, when the owner creates a supplier, then it appears in the active supplier list.
5. Given an inactive supplier, when the owner starts procurement, then that supplier is unavailable for selection.
6. Given active supplier and product records, when the owner creates a supplier-product relationship with a non-negative PHP cost, then it is available for sourcing.
7. Given an existing supplier-product relationship, when the owner attempts to create the same supplier/product pair, then the operation is rejected and the existing record is preserved.
8. Given a supplier-product relationship, when the owner changes its reference price, then the current value changes and one audit event records the old and new values.
9. Given a negative or invalid reference cost, when the owner submits the relationship, then validation fails and no data is written.
10. Given an accepted quotation or supplier PO sourced from a reference price, when the reference price changes, then the historical transaction cost remains unchanged.
11. Given two suppliers for one product, when both relationships are active, then each retains its own SKU and current reference cost.
12. Given a stale edit based on an old row version, when the owner saves it, then the system rejects the update and preserves the newer data.

## 9. Delivery sequence

1. Approve the decisions in the current-plan analysis, especially base-UOM pricing and no price-history table for MVP.
2. Define shared controlled UOM values and money conversion behavior.
3. Implement product schema, domain, repository, handlers, templates, and tests.
4. Implement supplier schema, domain, repository, handlers, templates, and tests.
5. Implement supplier-product relationship and current reference-price behavior.
6. Register the next migration after the currently applied migration `0006`; never edit an applied migration.
7. Wire active selectors for the RFQ and quotation slices.
8. Run migration metadata verification, `go test ./...`, `go vet ./...`, and manual SQL Server checks.
9. Record completion in `analysis/000-implementation-ladder.md` before starting plan 032.

## 10. Dependencies and readiness gates

**Dependencies:**

- `030-trading-foundation-plan.md` PBI 01 through PBI 03.
- Existing company-account, migration, audit, money, row-version, handler, and template conventions.
- SQL Server development and test databases.

**Ready for development when:**

- Product base UOM values are approved.
- SKU normalization and edit policy are approved.
- Supplier required fields are approved.
- One supplier-product row per supplier/product is approved.
- Reference cost is explicitly per product base-UOM unit in PHP.
- Supplier pack conversion is either deferred or fully specified.
- Current-price-only behavior and transaction snapshot rules are approved.
- Active/inactive selection rules and audit expectations are approved.

**Ready for plan 032 when:**

- Product and supplier masters can be created and selected.
- Active supplier-product relationships can be maintained and queried.
- Current reference cost can be updated safely and audited.
- RFQ and quotation code can consume active records without reaching into implementation details of these slices.
- All acceptance scenarios pass, including historical snapshot protection at the first downstream integration point.
