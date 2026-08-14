# Acknowledge Full Payment Tax Rule Plan

## 1. Goal

Make the tax calculation visible and explicit when the user acknowledges full payment for a receivable that was created with a tax rule.

The tax rule must **not be applied a second time** during payment acknowledgement. It was already applied when the receivable was created or edited, and the stored `AmountDue` is already the calculated net payable.

For the existing example:

```text
Gross amount: PHP 2,800.00
EWT:          PHP 25.00
Net payable:  PHP 2,775.00
```

The payment page should confirm `PHP 2,775.00` as the full amount to mark paid, show the tax breakdown for transparency, and record the payment against that net payable.

## 2. Critical Design Rule

### Do not recalculate EWT on acknowledgement

The current lifecycle is:

```text
Receivable create/edit:
gross amount -> tax rule -> EWT -> net payable -> AmountDue

Payment acknowledgement:
stored AmountDue/net payable -> mark paid
```

Applying `VAT-inclusive EWT` again during acknowledgement would produce an incorrect result:

```text
PHP 2,800.00 gross
already reduced to PHP 2,775.00
incorrectly reduced again to approximately PHP 2,750.22
```

The payment use case should therefore use the persisted tax snapshot as read-only display data and continue to mark the existing `AmountDue` as paid.

## 3. Current Payment Behavior

The current payment form:

- Displays the amount to mark as paid.
- Accepts only a payment date.
- Does not support partial payments.
- Does not accept a payment amount.
- Marks the whole receivable as paid through `ReceivePaymentCommand`.

This is compatible with the recommended design. No payment amount is needed unless the business wants to validate an externally entered settlement amount.

## 4. Payment Page Requirements

### Receivable summary

Continue displaying:

- Net payable / full amount due.
- Company, invoice, PO, delivery date, and due date.

When a tax rule exists, additionally display:

- Gross amount.
- Tax rule label.
- VAT-exclusive tax base.
- VAT amount.
- EWT amount.
- Net payable.

Example:

```text
Gross amount       PHP 2,800.00
Tax rule           VAT-inclusive, 1% EWT
VAT-exclusive base PHP 2,500.00
VAT                PHP 300.00
EWT                PHP 25.00
Net payable        PHP 2,775.00
Amount to mark paid PHP 2,775.00
```

For a no-rule receivable:

- Show the existing amount due behavior.
- Do not show misleading VAT or EWT values.
- Do not imply that the record was verified as non-taxable; it simply has no recorded tax rule.

### Confirmation language

Use explicit wording:

```text
This action acknowledges full receipt of the net payable amount.
Tax was calculated when the receivable was created or edited and will not be deducted again.
```

Keep the existing full-payment-only warning.

## 5. Domain/Application Contract

Keep `ReceivePaymentCommand` focused on payment state:

- Receivable ID.
- Payment date.
- Original row version.
- Request ID.
- Idempotency key.
- Actor ID.

Do not add a tax rate or tax-rule input to the payment command. The payment command must load the current receivable and use its persisted `AmountDue`.

At payment execution:

1. Load the receivable inside the transaction.
2. Confirm it is active and unpaid using the existing protection rules.
3. Do not call `tax.CalculateRule` to mutate an amount.
4. Validate the payment date.
5. Mark the existing receivable as paid.
6. Record the payment audit event with the net payable and tax snapshot.
7. Commit the state change and audit event together.

If a consistency check is desired, it must be read-only:

- `AmountDue` equals persisted net payable.
- Gross, EWT, and net payable satisfy the stored calculation snapshot.
- A mismatch should produce a data-integrity error and prevent acknowledgement, not silently recalculate historical tax.

## 6. Data And Persistence

No new payment tax columns are required. Reuse the receivable tax fields added by migration `0006`:

- `gross_amount_scaled`
- `tax_rule_code`
- `tax_base_scaled`
- `vat_amount_scaled`
- `ewt_amount_scaled`
- `amount_due_scaled` as net payable

The payment acknowledgement repository update only changes payment state and timestamps. It must not overwrite tax fields or amount fields.

The payment audit event should include:

- `amount_due` / net payable acknowledged.
- Gross amount.
- Tax rule code.
- Tax base.
- VAT.
- EWT.
- Payment date.

This makes it clear which financial amount was marked paid and prevents later tax-rule changes from changing the historical payment explanation.

## 7. View Models And Templates

Extend `PaymentFormViewModel` only if the payment form needs tax-specific display fields. Prefer passing the existing `ReceivableViewModel` tax display values to the payment page rather than recalculating in the template.

Recommended fields already available on `ReceivableViewModel`:

- `GrossAmountDisplay`
- `EWTDisplay`
- `NetPayableDisplay`
- `TaxRuleLabel`

Add only `TaxBaseDisplay` and `VATDisplay` if the page should show the full VAT proof. Templates must render prepared values and must not perform tax arithmetic.

The payment amount must remain `Receivable.AmountDisplay`, which is the net payable stored as `AmountDue`.

## 8. Reports And Paid Totals

The payment acknowledgement does not change amount values. It changes classification from active collection state to `Payment Received`.

Existing reports should therefore continue to use:

- `amount_due_scaled` for the payment-received amount.
- Net payable for outstanding and paid totals.
- Gross/EWT separate fields only for tax-specific reporting.

Do not subtract EWT from paid totals at acknowledgement. EWT has already been reflected in `amount_due_scaled`.

If a future tax report is required, provide separate totals:

- Gross invoice total.
- VAT total.
- EWT total.
- Net payable total.
- Paid net payable total.

Do not label net payable as gross revenue.

## 9. Edge Cases

- No tax rule: acknowledge the existing amount due without tax fields.
- VAT-inclusive EWT rule: show the persisted breakdown and acknowledge net payable.
- Receivable has a tax-rule code but missing tax snapshot fields: fail safely or run a controlled data repair; do not recalculate silently during payment.
- Stored gross, EWT, and net payable do not reconcile: prevent acknowledgement and log a data-integrity error.
- Tax rule configuration changes after creation: use the persisted historical snapshot; never apply the new rule to an existing payment.
- Active unpaid receivable edited before payment: payment page must show the newly persisted tax snapshot and net payable.
- Paid receivable: retain existing protection against duplicate acknowledgement.
- Cancelled or archived receivable: retain existing payment-not-allowed behavior.
- Payment date validation remains unchanged.
- Idempotent payment replay: return the original paid result without recalculating tax or creating a second audit event.
- Concurrent edit/payment: row-version conflict must prevent acknowledgement using stale tax or amount data.

## 10. Implementation Steps

1. Extend payment page view data with tax-base and VAT display values if full breakdown is approved.
2. Update the payment summary to label the amount as net payable/full amount due.
3. Add the tax breakdown and no-double-deduction notice when a tax rule exists.
4. Keep `ReceivePaymentCommand` unchanged with respect to tax fields.
5. Ensure `MarkPaymentReceived` updates only payment state and does not update tax or amount columns.
6. Include the tax snapshot and net payable in payment audit JSON.
7. Add an optional read-only consistency validator for persisted tax fields.
8. Add unit and HTTP/template tests.
9. Run `gofmt`, `go test ./...`, and `go vet ./...`.

## 11. Test Plan

### Domain/service tests

- Taxed receivable payment preserves `AmountDue` at `PHP 2,775.00`.
- Payment acknowledgement does not invoke a second EWT calculation.
- No-rule receivable payment preserves its original amount.
- Tax snapshot remains unchanged after payment acknowledgement.
- Taxed receivable becomes `Payment Received` after valid acknowledgement.
- Paid, cancelled, and archived records remain protected.
- Stale row version is rejected.
- Idempotent payment replay does not duplicate the audit event.

### Repository tests

- Payment update SQL changes only payment date and update timestamp.
- Payment query reads tax fields for the returned domain entity.
- Payment update does not write `amount_due_scaled`, gross, EWT, tax base, VAT, or rule code.
- Payment audit contains net payable and the tax snapshot.
- Tax snapshot mismatch is handled according to the approved integrity policy.

### HTTP/template tests

- Taxed payment page shows gross, tax base, VAT, EWT, net payable, and amount to mark paid.
- Taxed payment page states that tax is not deducted again.
- No-rule payment page does not show fake VAT/EWT values.
- Payment form submits only the payment date and idempotency/version fields.
- Validation failure retains the same tax summary.
- Successful acknowledgement redirects and preserves the net payable display on the detail page.

### Acceptance scenarios

1. Given a receivable created with gross `PHP 2,800.00` and the VAT-inclusive EWT rule, the payment page shows EWT `PHP 25.00` and amount to mark paid `PHP 2,775.00`.
2. When the user acknowledges payment, the receivable becomes paid without changing gross, EWT, tax base, VAT, or net payable.
3. Dashboard payment-received totals include `PHP 2,775.00`, not `PHP 2,800.00` and not `PHP 2,750.22`.
4. Given a no-rule receivable, acknowledgement behaves exactly as it does today.
5. Given a stale payment page after an edit, acknowledgement fails with a conflict instead of marking an old amount paid.

## 12. Decisions Required

- Should the payment page show only EWT and net payable, or the complete VAT proof including tax base and VAT?
- Is the payment amount always assumed to equal stored net payable, or should the user enter/confirm an actual received amount?
- If an actual payment amount is entered and differs from net payable, should the system reject it, allow a variance, or support partial/adjusted payment later?
- Should tax snapshot inconsistency block payment acknowledgement immediately, or should an admin repair workflow be created first?
- Should payment audit data include the tax snapshot as a full copy or reference the receivable's current fields?

## 13. Definition Of Done

- Payment acknowledgement never applies EWT twice.
- Taxed payment pages show the persisted tax breakdown and net payable.
- No-rule payment behavior remains unchanged.
- Payment updates do not mutate tax or amount fields.
- Payment audit records the amount acknowledged and tax snapshot.
- Reports continue using net payable for paid totals.
- Stale, duplicate, protected, and inconsistent-state cases are tested.
- `go test ./...`, `go vet ./...`, and formatting pass.
