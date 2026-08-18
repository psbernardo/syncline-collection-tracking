# 041 Shared Quotation and Sales Order PDF Decisions

## Confirmed Contract

Quotation and Sales Order PDFs use the same Syncline document layout, totals, footer, currency, and tax behavior. Only the document title changes:

| Document | Title |
|---|---|
| Quotation | `QUOTATION` |
| Sales Order | `SALES ORDER` |

### Totals

Use the same totals already implemented for quotations:

- `TOTAL SALES (VAT INC)`
- `LESS: VAT`
- `AMOUNT: NET OF VAT`

Totals must use persisted domain calculations, not PDF-side recalculation.

### Sales Order Status

The initial Sales Order statuses are:

- `OPEN`
- `COMPLETED`

### Sales Order Fields

The Sales Order uses all fields shown in the reference layout:

- order date
- sales order number
- quotation/reference number
- customer PO number
- sales person
- payment terms
- Bill To customer and billing address
- Ship To customer and delivery address
- line number
- description
- quantity and UOM
- rate
- tax percentage
- amount

### Defaults and Rules

- Sales person is temporarily fixed to `Alma Mae Bernardo`.
- Ship To uses the customer delivery address stored in the company account.
- Tax column displays `12%` when the selected tax rule is VAT-inclusive 12%.
- Tax display follows the selected quotation tax rule, matching the existing quotation behavior.
- Line amounts are tax-exclusive or tax-inclusive according to the selected tax rule, matching the existing quotation behavior.
- Currency is Philippine peso, displayed with the existing peso money formatter.
- Seller identity is Syncline, using the configured Syncline logo and seller contact details.
- The footer uses the same format as the quotation PDF.

## Implementation Consequence

Create one shared document renderer with a title/status input rather than separate quotation and Sales Order layout implementations:

```go
type DocumentTitle string

const (
    QuotationTitle  DocumentTitle = "QUOTATION"
    SalesOrderTitle DocumentTitle = "SALES ORDER"
)
```

The renderer should receive a customer-facing document projection. It must not receive internal supplier cost, commission, or profitability fields.

## Remaining Information Gaps

Before Sales Order implementation, the following data contracts still need to be defined in the application:

- Sales Order number format and sequence, unless it should reuse the quotation number generator pattern.
- How a Sales Order is created from an accepted quotation.
- Customer PO number storage and validation.
- Quotation/reference number storage on the Sales Order.
- Payment term source and allowed values. The current quotation uses 7, 15, 30, and 45 days.
- Whether an `OPEN` Sales Order can be edited and when it becomes `COMPLETED`.
- Whether the current customer PDF should expose the Sales Order status.
