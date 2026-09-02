# Invoice Receivable Confirmation Plan

## Goal

Before committing an Invoice conversion, show the user a summary of both the Invoice and the Receivable that will be created.

## User Flow

1. User opens an `OPEN` Sales Order.
2. User selects `Create invoice`.
3. The Invoice form displays the Sales Order snapshot and requires an Invoice number.
4. User enters the Invoice number and selects `Create invoice`.
5. The form opens a confirmation modal.
6. The server-generated summary displays the Invoice and Receivable values.
7. User selects `Confirm and create`.
8. The combined conversion operation creates both documents atomically.
9. User is redirected to Invoice detail with links to the Receivable and Sales Order.

`Cancel` closes the modal and returns the user to the form without creating anything.

## Summary Contents

### Invoice summary

- Customer/company
- Sales Order number
- Invoice number
- Customer PO number
- Invoice date
- Invoice line count
- Invoice subtotal
- Invoice VAT
- Invoice total

### Receivable summary

- Customer/company
- Linked Invoice number
- PO number
- Gross amount
- Tax rule mapped from the Sales Order: no tax, or `VAT-inclusive, 1% EWT` when the Sales Order uses 12% VAT
- VAT amount
- EWT amount
- Net receivable amount
- Delivery date
- Payment terms in days
- Due date

The modal must label Invoice total and Receivable net amount separately. The Receivable applies EWT, so its amount due may differ from the Invoice total.

## Recommended Endpoint Design

Use a server-rendered preview endpoint rather than calculating the financial summary only in JavaScript:

```text
POST /sales-orders/{id}/invoice/preview
POST /sales-orders/{id}/invoice
```

The preview request contains the Invoice number and idempotency key. The server loads the Sales Order, validates that it is `OPEN`, and calculates a non-persisted summary using the same mapping code as the final conversion.

The final POST must recalculate all values. It must not trust:

- Submitted totals
- Submitted tax amounts
- Submitted due date
- Submitted payment terms
- Submitted company ID
- Submitted Invoice number snapshot

The Invoice number is the only user-entered commercial value in this form.

## Shared Summary Model

Add a presentation-neutral summary type, for example:

```text
InvoiceReceivablePreview {
    SalesOrderNumber
    CustomerName
    InvoiceNumber
    CustomerPONumber
    InvoiceDate
    InvoiceSubtotal
    InvoiceTax
    InvoiceTotal
    ReceivableGrossAmount
    ReceivableTaxRule
    ReceivableVATAmount
    ReceivableEWTAmount
    ReceivableNetAmount
    DeliveryDate
    PaymentTermDays
    DueDate
}
```

The summary should be derived from domain objects or a conversion calculation service, not from GORM models or browser values.

## Template Changes

Update `internal/slices/invoices/templates/form.html` to:

- Keep the Invoice number input outside the modal.
- Change the primary button to open the modal after validation.
- Add a modal with a compact Invoice summary and Receivable summary.
- Provide `Cancel` and `Confirm and create` actions.
- Submit the original form only after confirmation.
- Disable the confirmation button while the request is processing.
- Preserve the form value and modal state when preview or final validation fails.

Use the existing application styles and JavaScript conventions. The flow must remain usable without JavaScript by allowing the normal form POST to perform server validation and render a confirmation page or final validation response.

## Validation and Error Handling

Preview errors should identify the relevant issue:

- Blank or invalid Invoice number
- Sales Order is not `OPEN`
- Sales Order has no lines
- Missing or invalid Sales Order terms
- Invoice number already exists
- Existing Invoice or Receivable conflict

If the Sales Order changes between preview and confirmation, the final transaction must reject or recalculate using the locked current row. It must never commit stale preview values.

On a duplicate submission, idempotency should redirect or return the already-created Invoice and Receivable instead of creating another pair.

## Accessibility and Responsive Behavior

- Use a semantic dialog with a clear accessible label.
- Move focus into the modal when it opens.
- Return focus to the Create invoice button when cancelled.
- Support Escape to close the modal.
- Keep the summary readable on narrow screens by using stacked sections rather than a wide table.
- Ensure the confirm action remains visible on mobile.

## Tests

- Preview renders the correct Invoice and Receivable values.
- Preview uses Sales Order terms rather than a Quotation lookup.
- Preview defaults delivery date to the current business date.
- Preview maps no Sales Order tax to no receivable tax rule and maps 12% Sales Order VAT to `VAT-inclusive, 1% EWT`.
- Preview displays Invoice total and Receivable net amount distinctly.
- Final submission recalculates values after tampered preview fields.
- Cancel does not create an Invoice or Receivable.
- Confirm creates both documents and redirects correctly.
- Invalid Invoice number preserves the entered value.
- A Sales Order changed to `CONVERTED` after preview cannot be converted.
- Duplicate confirmation is safe through idempotency.
- Modal is rendered with accessible labels and responsive markup.

## Acceptance Scenario

1. An `OPEN` Sales Order has a customer, PO number, terms, lines, and total.
2. The user enters `INV-1001` and opens the confirmation dialog.
3. The dialog shows `INV-1001`, the Sales Order customer, the Sales Order PO number, current delivery date, Sales Order terms, gross Invoice total, 1% EWT, and net Receivable amount.
4. The user confirms.
5. One Invoice and one Receivable are created.
6. The Sales Order becomes `CONVERTED`.
7. Repeating the request does not create another Invoice or Receivable.
