# 033 Customer PO and Sales Order Plan

## Epic E-04: Customer Commitment

### PBI 01: Customer PO capture

**Goal:** Record the customer's formal commitment without assuming it exactly matches the quotation.

**Tasks:**

- Add customer PO number, PO date, attachment/reference, and received date.
- Capture the customer PO against a quotation.
- Validate customer PO quantities, prices, payment terms, and delivery terms.
- Display differences between quotation and customer PO before conversion.
- Allow the owner to resolve differences manually.

**Dependencies:** 032 quotation lifecycle complete.

**Ready for development when:** PO fields, discrepancy behavior, and whether a PO attachment is required are approved.

### PBI 02: Sales Order creation

**Goal:** Create the final commercial transaction used by procurement and invoicing.

**Tasks:**

- Add `sales_orders` and `sales_order_lines`.
- Copy final customer, product, quantity, price, tax, payment, and delivery terms.
- Preserve quotation and customer PO references.
- Copy commission configuration and calculated amount as a snapshot.
- Lock the source quotation after successful creation.
- Make Sales Order creation idempotent.
- Add domain, repository, handler, template, migration, and acceptance tests.

**Dependencies:** Customer PO capture and 032 pricing calculation complete.

**Ready for development when:** Sales Order ownership of final commercial values is approved.

### PBI 03: Sales Order status and changes

**Goal:** Prevent invalid procurement and invoicing actions.

**Tasks:**

- Define statuses such as `DRAFT`, `CONFIRMED`, `PROCUREMENT_IN_PROGRESS`, `RECEIVED`, `INVOICED`, and `CANCELLED`.
- Allow only valid edits before procurement confirmation.
- Require a Sales Order to be confirmed before supplier POs are created.
- Prevent invoice creation before all required quantities are received.
- Add cancellation rules for unprocured orders.

**Dependencies:** 034 procurement status contract and 035 receiving contract must be agreed.

**Ready for development when:** Status transitions and edit locks are approved.
