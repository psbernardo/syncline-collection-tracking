# 043 Quotation Line Margin and Profit Plan

## Goal

Show internal estimated profit and margin for every quotation line and for the quotation total on the quotation create and edit screens. Cost inputs must be entered by the user and stored as quotation-time snapshots.

## Current State

- `quotations.Line` already has `SupplierCost`, but it is a hidden form field named `supplier_cost`.
- The create and edit form does not show or collect supplier cost visibly. New rows default it to zero.
- `CalculateTotals` already subtracts quantity multiplied by `SupplierCost` from the net subtotal.
- `CalculateTotals` already subtracts quotation-level commission and exposes aggregate `ProfitBeforeCommission` and `ProfitAfterCommission`.
- `Line` has no profit or margin fields. The browser only calculates quantity, unit price, line amount, and quotation tax totals.
- `supplier_delivery`, `customer_delivery`, and `other_cost` exist only as zero arguments to `CalculateTotals`; the form does not collect them and the quotation repository always passes zero.
- The database stores supplier unit cost per line and aggregate estimated supplier cost/profit, but does not store line profit/margin or quotation-level delivery/other costs.
- The detail page shows supplier cost and aggregate profitability, but not line profit/margin.
- Sales-order lines currently copy customer pricing but not supplier cost or profitability snapshots.

## Required Calculation Contract

Use the business-model definition of margin: `profit / net sales * 100`.

For each line:

```text
Supplier product cost = quantity * supplier unit cost
Net sales = line customer amount excluding VAT
Commission allocation = agreed allocation of quotation commission
Line profit before commission = net sales - supplier product cost - allocated direct costs
Line profit after commission = line profit before commission - commission allocation
Line margin % = line profit after commission / net sales * 100
```

For the quotation:

```text
Total cost = supplier product cost
           + supplier delivery cost
           + customer delivery cost
           + other cost
           + commission

Profit before commission = net sales - supplier product cost
                           - supplier delivery cost
                           - customer delivery cost
                           - other cost

Profit after commission = profit before commission - commission
Margin % = profit after commission / net sales * 100
```

The implementation must explicitly choose whether the requested displayed profit is before or after commission. Recommended: display both, with **profit/margin after commission** as the primary internal profitability measure, because the business model treats commission as a selling cost.

## Data Gaps and Decisions

### Cost inputs required from the user

1. Supplier unit cost per line. This already exists technically, but must become a visible required internal input with a clear label and currency format.
2. Supplier delivery cost per quotation. Needed because the calculation API already supports it and the business model includes it.
3. Customer delivery cost per quotation. Needed for complete estimated fulfillment cost.
4. Other cost per quotation. Needed for miscellaneous or financing-related quotation costs.

The first release can keep delivery and other costs quotation-level. Do not distribute them into line margins unless an allocation rule is agreed. The total margin can include them while line margin can be labelled **line margin before allocated quotation costs**, or the UI can show line margin only after a defined allocation method.

### Commission allocation

Commission is currently quotation-level and supports per-unit, percentage, and fixed quotation modes. A line-level after-commission margin requires an allocation rule:

- `PER_UNIT`: allocate by line quantity multiplied by commission rate.
- `PERCENTAGE`: allocate in proportion to each line's net sales, matching the quotation percentage base.
- `FIXED_QUOTATION`: allocate in proportion to each line's net sales, unless a separate fixed allocation is entered.

Recommended: use these deterministic rules and store the calculated line commission snapshot. Do not infer line commission from display-only JavaScript.

### Tax basis

Use net sales excluding VAT for profit and margin, consistent with the existing aggregate `Totals.Subtotal` and the business-model definition. Keep VAT as a separate customer total and do not treat it as revenue or profit.

### Missing persisted values

Add quotation-level fields for:

- `supplier_delivery_cost_scaled`
- `customer_delivery_cost_scaled`
- `other_cost_scaled`

Add line-level calculated snapshots for:

- supplier product cost total
- allocated commission
- profit before commission
- profit after commission
- margin percentage, or calculate it from persisted amounts when read

Prefer persisting monetary snapshots and calculating percentage from them. Store margin percentage only if reporting/query performance requires it. All monetary values must use the existing scaled `money.Amount` convention.

## Implementation Plan

### PBI 01: Domain calculations

- Extend `Line` and `Totals` with the required profitability values.
- Add a reusable margin calculation that returns zero when net sales is zero and supports negative profit.
- Refactor `CalculateTotals` to accept the three quotation-level cost inputs and return line calculations plus totals, or add a separate calculation result that contains both.
- Implement commission allocation for all supported commission types.
- Add tests for profitable, zero-profit, loss-making, VAT, and each commission type.

### PBI 02: Persistence and migration

- Add the three quotation-level cost columns.
- Add line-level cost/profit/commission snapshot columns if the result is intended to remain historically stable and available to downstream reporting.
- Update repository create, update, and read mappings.
- Recalculate all values server-side on create/update; never trust browser-calculated profit or margin.
- Backfill existing quotations with zero for newly introduced quotation-level costs and calculate line snapshots from their stored supplier cost and prices.

### PBI 03: Create/edit input UX

- Add visible `Supplier unit cost` to each quotation row beside customer rate.
- Add a quotation-level `Supplier delivery`, `Customer delivery`, and `Other cost` cost section.
- Show per-line supplier cost total, profit before/after commission, and margin percentage in the grid or an expandable internal-cost area.
- Show quotation-level net sales, total cost, commission, profit before commission, profit after commission, and margin percentage in the live summary.
- Update `app.js` to refresh only presentation values; submit the raw cost inputs as form fields.
- Preserve internal profitability visibility in create/edit while excluding it from customer-facing PDF output.
- Add clear labels that these are estimated internal costs and clarify whether displayed line margin excludes unallocated quotation-level costs.

### PBI 04: Detail and downstream snapshot

- Add line profitability columns and total margin to the quotation detail page.
- Decide whether Sales Order should copy estimated cost/profit snapshots. Recommended: copy them as read-only estimates for traceability, while later procurement/receiving work owns actual cost and actual margin.
- Ensure customer-facing quotation PDFs do not expose supplier cost, profit, margin, or commission.

### PBI 05: Validation and tests

- Reject negative cost inputs and invalid amounts.
- Require supplier unit cost explicitly, or confirm that zero is a valid estimate for unknown cost before implementation.
- Test create and edit form parsing for all cost inputs.
- Test server-side recalculation when submitted hidden/display values are tampered with.
- Test line and total calculations with VAT, discounts if later added, commission, delivery costs, and loss-making quotes.
- Test that customer PDF content does not contain internal cost/profit fields.

## Recommended Delivery Order

1. Approve the calculation basis, especially after-commission vs before-commission display and line allocation of quotation-level costs.
2. Implement domain result objects and tests.
3. Add migration and repository snapshots.
4. Add visible cost inputs and live create/edit calculations.
5. Add detail-page profitability and downstream snapshot behavior.
6. Run full quotation, sales-order, migration, and PDF regression tests.

## Acceptance Criteria

- A user can enter supplier unit cost for every quotation line during create and edit.
- A user can enter supplier delivery, customer delivery, and other estimated costs for the quotation.
- Create and edit show updated line profit/margin and total profit/margin without requiring a save.
- Saved values are recalculated and validated on the server.
- Profit and margin use net sales excluding VAT and include commission according to the approved allocation rule.
- Loss-making lines and quotations display negative profit/margin correctly.
- Internal profitability data is visible only in internal screens and is absent from the customer PDF.
