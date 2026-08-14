# Receivable Tax Rule Plan

## 1. Goal

Allow a user to select an optional tax rule while creating or editing a delivery receivable. The initial state has no tax rule selected. The first available rule is the VAT-inclusive EWT rule already implemented in `internal/shared/tax`:

```text
VAT rate: 12%
EWT rate: 1%
EWT base: VAT-exclusive tax base
```

When the rule is selected, calculate and show:

- Original amount / gross amount entered by the user.
- EWT amount.
- Net payable.

The net payable becomes the receivable's outstanding amount. Existing grids, dashboard totals, payment amount, filters, and collection calculations continue to use the stored net payable through the existing `AmountDue` field.

## 2. Precision Policy

The canonical example uses a two-decimal gross amount:

```text
Input amount:  PHP 2,800.00
EWT:          PHP 25.00
Net payable:  PHP 2,775.00
```

The repository retains up to four decimal places internally and formats currency to two decimals. Inputs with more than two decimals remain supported, but their calculated EWT and net payable must be tested and displayed according to that existing precision contract. The implementation must not silently truncate fractional values to force whole-peso results.

## 3. Tax Rule Behavior

### No rule selected

- Selection value is empty/null.
- Gross amount is the net payable.
- EWT is zero.
- VAT/tax-base calculation is not applied.
- Existing receivables remain compatible and display no tax rule.

### VAT-inclusive EWT rule selected

Given gross amount `G`:

```text
tax base = round(G / 1.12)
VAT = G - tax base
EWT = round(tax base * 1%)
net payable = G - EWT
```

Use `tax.CalculateVATInclusive` as the only calculation path. Do not duplicate the formula in the receivables domain, handler, template, SQL, or JavaScript.

Examples at two-decimal display precision:

| Gross amount | Tax base | EWT | Net payable |
|---:|---:|---:|---:|
| PHP 2,800.00 | PHP 2,500.00 | PHP 25.00 | PHP 2,775.00 |
| PHP 13,400.00 | PHP 11,964.29 | PHP 119.64 | PHP 13,280.36 |

The second requested value, `PHP 13,280`, is a whole-peso presentation only. It must not replace the precise stored net payable unless whole-peso truncation is explicitly approved.

## 4. Source Of Truth And Persistence

Do not store only the calculated net amount. That would lose the user's original invoice amount and make tax recalculation, audit review, and editing ambiguous.

Extend `delivery_receivables` with:

- `gross_amount_scaled BIGINT NULL` or `NOT NULL` after a backfill strategy is approved.
- `tax_rule_code VARCHAR(64) NULL`.
- `ewt_amount_scaled BIGINT NOT NULL DEFAULT 0`.
- `net_payable_scaled BIGINT NOT NULL`.
- Optionally `tax_base_scaled BIGINT NULL` and `vat_amount_scaled BIGINT NULL` for audit/report transparency.
- Optionally `tax_calculation_version VARCHAR(32) NOT NULL` when historical reproducibility is required.

Recommended initial model:

- Preserve existing `amount_due_scaled` as the stored net payable used by all current collection queries.
- Add `gross_amount_scaled` to preserve the entered source amount.
- Add `tax_rule_code` to preserve the selected rule, including no selection.
- Add `ewt_amount_scaled` to preserve the applied deduction.
- Add `tax_base_scaled` and `vat_amount_scaled` if the UI/report needs to explain the calculation without recalculating historical values.
- Treat `amount_due_scaled` and `net_payable_scaled` as one concept; avoid storing both unless a migration or external contract requires both names.

For new records:

```text
gross_amount_scaled = parsed user amount
tax_rule_code       = selected rule or NULL
ewt_amount_scaled   = calculated EWT or 0
amount_due_scaled   = net payable
```

For historical records, the migration should set `gross_amount_scaled = amount_due_scaled`, `ewt_amount_scaled = 0`, and `tax_rule_code = NULL`. Historical records then mean “no tax rule recorded,” not that the original invoice was proven non-taxable.

Add non-negative check constraints for every scaled amount and a check that `amount_due_scaled = gross_amount_scaled - ewt_amount_scaled` for the supported rule model. If tax adjustments may later exist, do not add a constraint that assumes all future rules have this exact shape.

## 5. Domain And Application Changes

### Tax rule type

Add a typed rule code in the shared tax package or receivables domain, for example:

```go
type RuleCode string

const (
    RuleNone             RuleCode = ""
    RuleVATInclusiveEWT RuleCode = "vat_inclusive_ewt_1"
)
```

Expose a trusted rule option list for the form:

- Empty: `No tax rule`
- `vat_inclusive_ewt_1`: `VAT-inclusive, 1% EWT`

Do not accept arbitrary rates or formulas from the browser. The posted code must be validated against the server-side rule registry.

### Receivable domain

Extend `DeliveryReceivable` with:

- `GrossAmount money.Amount`
- `TaxRuleCode tax.RuleCode`
- `EWTAmount money.Amount`
- Optional `TaxBase` and `VATAmount` if persisted/displayed
- Keep `AmountDue` as net payable for existing collection behavior.

Change `NewDeliveryReceivable` and the update path to accept the amount and tax-rule selection, then:

1. Parse the gross amount once.
2. Validate the tax rule.
3. Apply no-rule identity behavior or the shared VAT-inclusive EWT calculation.
4. Assign `AmountDue = NetAmount`.
5. Return field validation for invalid rule/input.

The same function must be used by create and edit so behavior cannot diverge.

### Commands

Add `TaxRuleCode` to `CreateReceivableCommand` and `UpdateReceivableCommand`. Include the selected rule and calculated values in the idempotency payload hash. A replay with a changed tax rule must be rejected as an idempotency conflict.

Protect paid records from edit as today. Editing an active unpaid receivable recalculates tax and changes the stored amount used by dashboard totals; the audit event must include old and new gross, rule, EWT, net payable, tax base, and VAT where available.

## 6. HTTP Form And UI

Add a required-by-UI-but-empty-allowed select after the amount field:

```text
Tax rule: [ No tax rule                       v ]
           [ VAT-inclusive, 1% EWT            ]
```

The initial create form must select no rule. The edit form must select the persisted rule. On validation errors, preserve the posted selection.

Show server-rendered calculated fields when a rule is selected:

- Original amount: formatted gross.
- EWT: formatted withholding amount.
- Net payable: formatted net amount.

The browser may provide a preview, but it must not be authoritative. The server recalculates on POST and persists only the server result. Prefer an HTMX preview endpoint or server-rendered response if live preview is required; do not duplicate tax arithmetic in Alpine/JavaScript.

The form must clearly label the amount as gross/source amount once a tax rule exists. Avoid calling both gross and net “Amount.”

Update receivable list/detail/payment displays deliberately:

- Existing amount column displays net payable because it is the collection amount.
- Detail view displays gross, tax rule, EWT, and net payable when tax data exists.
- Payment acknowledgement uses net payable.
- No-rule records continue to show the existing amount without tax fields.

## 7. Reports, Grid, And Calculations

Because `amount_due_scaled` remains net payable, the following existing behavior automatically uses net payable after repository mapping is updated:

- Receivable list/grid amount.
- Dashboard classification totals.
- Company totals.
- Outstanding amount.
- Payment page amount.
- Any amount filters or sort order added later.

Explicitly review and test every query selecting or summing `amount_due_scaled`. Do not change dashboard SQL to sum gross or EWT. Add tax columns to projections only where the UI needs tax detail.

Reports that need tax analysis should expose separate aggregates:

- Gross/source total.
- EWT total.
- Net payable/outstanding total.

Do not label a net-payable collection total as gross revenue. This is a reporting semantic boundary.

## 8. Migration Plan

Add a new versioned migration, not an edit to the applied baseline:

1. Add nullable tax columns or add columns with safe defaults.
2. Backfill existing rows: gross equals existing amount due, EWT zero, no rule selected.
3. Make columns non-null only after the backfill is verified, if that is the chosen schema.
4. Add check constraints and indexes only where required by queries.
5. Update model projections and repository create/update/select statements.
6. Test migration rollback and existing-row compatibility.

Do not store the tax rule only in JSON audit data; it is needed for edit/display/report queries.

## 9. Testing Plan

### Shared tax tests

Retain and extend `internal/shared/tax` tests for:

- PHP 2,800.00 with 12% VAT and 1% EWT: EWT 25.00, net 2,775.00.
- PHP 13,400.00: EWT 119.64, net 13,280.36.
- An input with more than two decimal places under the selected four-decimal precision policy.
- No withholding rate/no-rule identity calculation.
- Invalid rule/rate and negative amounts.
- Tax base/VAT/EWT reconciliation and overflow.

### Receivable domain tests

- No rule stores gross as `AmountDue`, EWT zero, and null/empty rule.
- Selected rule stores net payable as `AmountDue`.
- Selected rule stores gross and EWT separately.
- Invalid tax-rule code returns validation errors.
- Create and edit produce identical calculations.
- Editing a tax rule changes amount due only for active unpaid records.
- Paid receivable remains protected from tax-rule edits.
- Four-decimal source values are retained and formatted correctly.

### Service and repository tests

- Create/update command payload includes tax rule and calculation result.
- Repository inserts/updates all tax columns.
- Repository reads old and new tax columns correctly.
- Existing rows with no rule map to identity behavior.
- Dashboard sums `amount_due_scaled` as net payable.
- Audit before/after JSON includes tax fields.
- Idempotent replay with identical tax inputs returns the original result.
- Same idempotency key with changed rule or amount fails.

### HTTP/template tests

- Create form defaults to no selected rule.
- Edit form restores the selected rule.
- Validation failure preserves selection.
- Tax summary displays gross, EWT, and net payable.
- List/detail/payment pages display net payable.
- HTMX and full-page responses are consistent.
- No-rule records do not show misleading VAT/EWT values.

### End-to-end acceptance scenarios

1. Create a receivable for `PHP 2,800.00` with no rule. The stored and reported amount is `PHP 2,800.00`.
2. Create a receivable for `PHP 2,800.00` with VAT-inclusive EWT. The grid and dashboard use `PHP 2,775.00`; detail shows EWT `PHP 25.00`.
3. Create a receivable for `PHP 13,400.00` with the rule. The grid uses `PHP 13,280.36`; EWT shows `PHP 119.64`.
4. Edit an unpaid receivable from no rule to the EWT rule. The amount due and dashboard totals change to net payable, with an audit event.
5. Edit an unpaid receivable's gross amount. Gross, EWT, net payable, grid, and dashboard all update consistently.
6. Attempt to edit a paid receivable's tax rule. The existing protected-state behavior rejects the change.

## 10. Open Decisions

- Confirm that inputs with more than two decimal places should retain the existing four-decimal precision behavior.
- Confirm whether amount input is an invoice gross amount or a pre-tax/net amount when no rule is selected.
- Confirm the exact label for the only tax rule and whether it is legally correct to call it “1% EWT.”
- Confirm whether tax rule configuration must be effective-dated/versioned before additional rules are introduced.
- Confirm whether tax base and VAT must be persisted or can be derived from gross/rule for display.
- Confirm whether historical records need a tax snapshot for audit and filing.
- Confirm whether reports should add separate gross/EWT/net columns or only continue using net payable.
- Confirm treatment of cancelled, archived, and paid receivables in gross/EWT reporting.
- Confirm whether payment is based on net payable even if the supplier invoice or official receipt has a different settlement amount.

## 11. Definition Of Done

- Tax rule is optional and defaults to no selection.
- Only server-registered rule codes are accepted.
- Create and edit apply the same shared VAT-inclusive EWT calculation.
- Original gross, tax rule, EWT, and net payable are preserved.
- `AmountDue` is net payable and all existing collection totals use it.
- Detail/report surfaces distinguish gross, EWT, and net payable.
- Existing records remain valid and behave as no-rule records.
- Four-decimal precision for inputs with more than two decimals is explicitly approved and tested.
- Audit and idempotency data include tax changes.
- Migration, domain, repository, service, HTTP, template, and dashboard tests pass.
- `go test ./...`, `go vet ./...`, and formatting pass.
