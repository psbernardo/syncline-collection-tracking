# 030 Trading Foundation Plan

## Epic E-01: Trading Domain Foundation

### PBI 01: Approve the trading MVP contract

**Goal:** Freeze the minimum business rules before schema and UI work.

**Tasks:**

- Record principal-reseller behavior: the company buys and resells goods.
- Record the lifecycle: RFQ, quotation, Sales Order, supplier PO, receipt, invoice, collection.
- Confirm accepted quotation is locked when the Sales Order is created.
- Confirm invoice is created from the Sales Order only after all goods are received or marked ready for customer delivery.
- Confirm partial delivery and partial payment are excluded.
- Confirm commission types: per-unit, percentage, and fixed quotation amount.
- Confirm percentage commission uses a VAT-inclusive base.
- Record tax codes `VAT_12`, `ZERO_RATED`, and `GOV_6` as configurable values.
- Confirm the existing receivable slice is the invoice collection destination.

**Ready for development when:** The decisions are recorded in the business knowledge base and no core transaction rule is ambiguous.

### PBI 02: Define transaction snapshots and immutability

**Goal:** Prevent reference-data changes from changing historical transactions.

**Tasks:**

- Define quotation line customer-price snapshots.
- Define quotation supplier-cost snapshots.
- Define Sales Order price, tax, and commission snapshots.
- Define supplier PO cost snapshots.
- Define received-cost snapshots used for actual profit.
- Define which records are editable by status.
- Define audit actions for creation, acceptance, allocation, receipt, invoice, and cancellation.

**Ready for development when:** Each historical amount has a transaction-owned source of truth and accepted quotations/Sales Orders cannot be silently changed.

### PBI 03: Define the cross-slice integration boundary

**Goal:** Integrate trading invoices with existing receivables without coupling slices to each other's repositories.

**Tasks:**

- Define an application command or explicit port for creating a delivery receivable from an invoice.
- Map Sales Order customer, invoice number, PO number, delivery date, payment term, and total due.
- Preserve the Sales Order and invoice identifiers on the receivable.
- Ensure invoice creation and receivable creation use one transaction where required.
- Keep the existing payment and dashboard behavior unchanged.

**Ready for development when:** The command contract, transaction boundary, failure behavior, and ownership of each field are approved.
