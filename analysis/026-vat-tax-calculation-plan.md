# VAT-Inclusive Tax Calculation Plan

## 1. Goal

Document and implement a shared Go calculation for a Philippine VAT-inclusive amount. The calculation must isolate the VAT-exclusive tax base before any withholding tax is calculated, use the existing shared PHP money representation, and be covered by deterministic unit tests.

The canonical example is:

```text
Gross amount:       PHP 1,459.00
VAT rate:           12%
VAT-inclusive factor: 1.12
Tax base:           PHP 1,459.00 / 1.12 = PHP 1,302.678571...
Rounded tax base:   PHP 1,302.68
VAT:                PHP 1,302.68 * 12% = PHP 156.3216 -> PHP 156.32
Gross proof:        PHP 1,302.68 + PHP 156.32 = PHP 1,459.00
```

Do not calculate VAT as `gross * 12%`, and do not subtract `gross * 12%` from gross. VAT is 12% of the VAT-exclusive base, so the VAT-inclusive gross is 112% of that base.

## 2. Scope

### In scope

- A shared package-level calculation for VAT-inclusive gross amounts.
- VAT rate and tax-base derivation using integer arithmetic.
- Currency rounding and overflow/error handling.
- A returned breakdown containing gross, tax base, VAT, rate, and VAT treatment metadata.
- Documentation of the inputs required for withholding-tax calculation.
- Unit tests for the formula, rounding, boundaries, and invalid input.

### Out of scope for this slice

- Selecting a Philippine withholding-tax rate from BIR rules.
- Determining whether a supplier or transaction is subject to withholding.
- BIR Form 1601-EQ filing, remittance, reporting, or submission.
- Tax advice or legal determination of a supplier's classification.
- Non-VAT, zero-rated, exempt, mixed, or percentage-tax treatment beyond explicit validation/documentation.
- Database schema changes until the tax inputs are confirmed as part of a receivable or payment workflow.

## 3. Terms

| Term | Definition |
|---|---|
| Gross amount | The amount received from the source document or transaction. For this calculation it already includes 12% VAT. |
| VAT-inclusive | The gross amount includes VAT. At a 12% rate, gross equals tax base multiplied by `1.12`. |
| VAT-exclusive amount / tax base / net-of-VAT amount | The amount on which VAT and, subject to applicable rules, withholding are based. |
| VAT | Value-Added Tax included in the gross amount. At 12%, VAT equals tax base multiplied by `0.12`. |
| VAT rate | A percentage represented in the calculation as a rational rate, not a binary floating-point value. The default example is 12%. |
| VAT-inclusive factor | `1 + VAT rate`; for 12%, `1.12`. |
| Withholding tax | A separate amount withheld under the applicable BIR rule and rate. It must be calculated only after the VAT component is removed when the rule requires a VAT-exclusive base. |
| Rounded display amount | A two-decimal currency amount shown to users. The existing `money.Amount` retains four-decimal precision as the source of truth. |

## 4. Required Inputs

The calculation API must receive or establish these values explicitly:

- Gross amount, as `money.Amount` or a validated amount input.
- VAT treatment: VAT-inclusive, VAT-exclusive, non-VAT registered, zero-rated, exempt, or mixed. This slice accepts only VAT-inclusive for the reverse formula.
- VAT rate. Do not silently assume 12% when the rate is not known. The 12% example may be a named default/configuration value.
- Currency. This slice supports PHP only.
- Rounding policy and precision. Use the repository convention of four-decimal internal storage and two-decimal display/currency output.
- Transaction/source date where tax rules depend on date or where the applicable VAT rate may change.
- Supplier VAT registration status and invoice/document evidence when the calculation is used in a business workflow.

Additional inputs are required before withholding tax can be calculated:

- Applicable withholding-tax type and legal classification.
- Withholding rate, as a validated percentage or rational value.
- Payee/supplier classification and BIR registration information.
- Nature of income or payment and whether the payment is subject to withholding.
- Whether the withholding rule uses the VAT-exclusive tax base, another statutory base, a threshold, or an exemption.
- Relevant transaction and remittance period.

The example does not provide enough information to calculate withholding tax. The implementation must require the rate and rule classification rather than infer them.

## 5. Calculation Contract

For a VAT-inclusive gross amount `G` and VAT rate `r`:

```text
factor = 1 + r
tax base exact = G / factor
VAT exact = tax base exact * r
```

For the 12% case:

```text
factor = 1.12
tax base exact = G / 1.12
VAT exact = tax base exact * 0.12
```

The implementation must avoid `float64`. Use integer/rational arithmetic with the existing `money.Amount` scale of 10,000 units per peso. Preserve enough intermediate precision to avoid an early truncation; round only at the defined output boundary.

The returned result should contain at least:

```go
type VATBreakdown struct {
    GrossAmount money.Amount
    TaxBase     money.Amount
    VATAmount   money.Amount
    VATRate     Rate
    Treatment   Treatment
}
```

The API should return an error for unsupported treatment, invalid rates, negative values, and arithmetic overflow. A zero gross amount is valid and returns zero tax base and zero VAT.

## 6. Rounding and Reconciliation

- Keep calculations in scaled integer units; never use binary floating point for money.
- Use the existing non-negative half-up money rounding convention for fractional units.
- Tax base and VAT must be rounded according to one documented policy, consistently applied in every caller.
- The displayed proof uses two-decimal values. Because independently rounded components can differ by one cent, the implementation must define reconciliation behavior.
- Preferred reconciliation rule for a VAT-inclusive invoice: calculate the rounded tax base from gross, calculate VAT as `gross - rounded tax base`, and expose the VAT-rate calculation as a diagnostic/check. This guarantees `tax base + VAT = gross` at currency precision.
- For the example, rounded tax base is `PHP 1,302.68`, VAT is `PHP 156.32`, and the proof total is exactly `PHP 1,459.00`.
- If statutory or accounting policy requires VAT to be independently rounded from the tax base, record the one-cent difference explicitly and test the selected policy. Do not allow callers to choose inconsistent policies ad hoc.
- The four-decimal stored values remain the source of truth; UI formatting must use `money.Amount.Format`/`FormatPHP`.

## 7. Edge Cases

- Gross amount is zero.
- Gross amount is one cent or one ten-thousandth of a peso; rounding can produce a zero tax base and must be defined.
- Gross amount is exactly divisible by the VAT-inclusive factor.
- Tax base or VAT is exactly at a half-unit rounding boundary.
- Gross amount has more than four decimal places and is rounded by `money.Parse` before calculation.
- VAT rate is zero, negative, or equal to/exceeds 100%; reject rates outside the supported business range.
- VAT rate has more precision than the money scale; represent it as a rational/decimal rate and validate it without `float64`.
- Unsupported VAT treatment is passed to a VAT-inclusive function.
- Non-VAT registered supplier: do not reverse out VAT; the gross amount is not automatically a VAT-inclusive base. This requires a separate explicit path.
- VAT-exempt or zero-rated transaction: do not apply the 12% formula; VAT is zero only when the treatment is explicitly confirmed.
- Mixed VAT treatments: calculate each line/component separately before aggregation; never reverse the total blindly if the total combines treatments.
- Gross amount, factor, intermediate product, or result exceeds `int64` capacity.
- Missing withholding rate or classification: return a validation error, not zero withholding.
- Tax base is used for withholding: ensure withholding is not calculated on VAT or added to the gross amount.
- Recalculation after edits: preserve the source gross, rate, treatment, and calculation version/audit context if the result is persisted.

## 8. Proposed Implementation Steps

1. Confirm the business decisions in Section 9, especially rate source and rounding/reconciliation policy.
2. Add a shared package under `internal/shared/tax` (or an equivalent explicitly approved shared location) that imports `internal/shared/money` only.
3. Add typed `Rate` and VAT `Treatment` values, validation, and a VAT-inclusive breakdown function.
4. Implement integer/rational arithmetic with overflow checks and no `float64` conversion.
5. Add table-driven unit tests before or alongside implementation.
6. Add a test for the canonical PHP 1,459.00 example, including exact formatted output and gross reconciliation.
7. Add documentation or domain-facing validation showing that withholding requires a separate configured rate/classification.
8. Integrate the breakdown into the first actual receivable/payment use case only after the owning slice defines where gross, tax base, and VAT are persisted/displayed.
9. Run `gofmt`, `go test ./...`, and `go vet ./...`.
10. Update `analysis/028-progress.md` after implementation.

## 9. Decisions Required Before Coding

- Is the application calculating VAT for invoices, payments, withholding certificates, or all three?
- Is the supported VAT rate always 12%, or must rates be effective-dated/configurable?
- Should the public result store/display four-decimal tax values, or only two-decimal currency values?
- Should VAT be derived as `gross - rounded tax base` to guarantee reconciliation, or independently rounded as `rounded tax base * rate`?
- Which exact withholding-tax types and rates must the product support?
- Does the applicable withholding rule always use the VAT-exclusive base, or are there payment-specific exceptions?
- Should non-VAT, zero-rated, exempt, and mixed transactions be implemented now or rejected until a later slice?
- Where should the source document date, supplier VAT status, tax treatment, and rate be stored for auditability?
- Is a tax calculation snapshot required so later rate/configuration changes do not alter historical records?

## 10. Unit-Test Matrix

### Formula and example

- PHP 1,459.00 at 12% returns tax base PHP 1,302.68.
- The result returns VAT PHP 156.32 under the reconciliation policy.
- Tax base plus VAT equals gross at two-decimal currency precision.
- The direct subtraction error (`gross - gross * rate`) is not used.

### Rates and treatments

- Zero gross returns zero values.
- Valid supported rate succeeds.
- Zero/negative/over-limit/invalid rates fail according to the confirmed policy.
- Non-VAT, exempt, zero-rated, and mixed treatments do not accidentally use the 1.12 formula.

### Precision and boundaries

- Exact division.
- Half-up rounding at the internal four-decimal boundary.
- Two-decimal display rounding.
- One-cent reconciliation boundary.
- Very small amounts.
- Maximum representable amount and overflow.

### Withholding contract

- Missing withholding rate fails.
- Missing classification fails.
- A supplied withholding rate is applied to the approved base, not gross, VAT, or gross-minus-gross-rate.
- A withholding result never exceeds the approved base without an explicit rule allowing it.

## 11. Definition Of Done

- The formula and terminology are documented in this file.
- The exact input contract and unresolved tax decisions are explicit.
- Shared Go code uses the existing `money.Amount` representation and no floating-point money arithmetic.
- The canonical example and all agreed edge cases have unit tests.
- Tax-base/VAT output reconciles to gross under the selected rounding policy.
- Withholding tax is not calculated without an explicit applicable rate and classification.
- Unsupported tax treatments fail safely rather than being silently treated as 12% VAT.
- `go test ./...`, `go vet ./...`, and formatting pass.
