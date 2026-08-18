# 036 Receivable Handoff Plan

## Epic E-07: Collection Integration

### PBI 01: Create delivery receivable from invoice

**Goal:** Make a posted trading invoice visible in the existing collection-tracking workflow.

**Tasks:**

- Map invoice customer to existing `company_accounts`.
- Map invoice number, Sales Order/customer PO reference, delivery date, payment term, and amount due.
- Preserve invoice ID and Sales Order ID for drill-down.
- Create the existing delivery receivable in the same transaction as invoice posting where required.
- Reuse existing tax amount mapping instead of duplicating tax calculation rules.
- Add idempotency and rollback tests.

**Dependencies:** 035 invoice posting and 030 integration boundary complete.

**Ready for development when:** Field mapping and transaction ownership are approved.

### PBI 02: Existing collection workflow regression

**Goal:** Ensure trading invoices do not break current manual receivable entry and payment tracking.

**Tasks:**

- Keep existing manual receivable creation working.
- Keep full payment acknowledgement working.
- Keep payment reversal/correction behavior working where implemented.
- Confirm dashboard totals include invoices created by trading and manually created receivables according to the agreed scope.
- Add tests for invoice-created receivable classification and payment date behavior.

**Dependencies:** 036 PBI 01 complete.

**Ready for development when:** Existing `accounts`, `receivables`, and `dashboard` tests remain green and the combined totals are approved.

### PBI 03: Trading-to-collection navigation

**Goal:** Allow the owner to trace a collection record back to its commercial transaction.

**Tasks:**

- Add Sales Order, invoice, quotation, and customer PO references to the receivable view.
- Add links from invoice/receivable to Sales Order and customer account.
- Keep collection screens focused on amount due, due date, payment status, and history.
- Do not duplicate quotation or procurement editing in the receivables slice.

**Dependencies:** 036 PBI 01 and existing receivable detail page.

**Ready for development when:** Navigation ownership and missing-reference behavior are approved.
