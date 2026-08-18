# 035 Receiving and Invoice Plan

## Epic E-06: Fulfillment and Invoicing

### PBI 01: Full receiving

**Goal:** Confirm that the complete Sales Order quantity has been received or is ready for customer delivery.

**Tasks:**

- Add receipt header and receipt lines, or an equivalent transaction-owned receiving record.
- Record received date, received quantity, condition, delivery method, and proof/reference.
- Require received quantity to equal the confirmed Sales Order quantity because partial delivery is out of scope.
- Record actual supplier, supplier delivery, customer delivery, and other costs.
- Prevent duplicate receiving confirmation.
- Add receiving audit events.

**Dependencies:** 034 procurement allocation and 033 Sales Order status complete.

**Ready for development when:** “Received” versus “ready for customer delivery” is defined and full-quantity validation is approved.

### PBI 02: Actual profitability

**Goal:** Compare quoted profitability with actual transaction profitability.

**Tasks:**

- Snapshot estimated supplier, delivery, other, and commission costs at quotation/Sales Order time.
- Record actual supplier and fulfillment costs at receiving/invoice time.
- Calculate profit before commission and profit after commission.
- Calculate margin using the agreed VAT-inclusive commission rule and VAT-separated reporting.
- Display estimated, actual, and variance values.
- Add tests for quantity, tax, commission, and cost variance calculations.

**Dependencies:** 032 commission and tax calculations, 034 supplier costs, and 035 receiving cost entries.

**Ready for development when:** Cost categories, VAT treatment, and profitability formulas are approved.

### PBI 03: Invoice creation from Sales Order

**Goal:** Create the customer invoice only from the final Sales Order.

**Tasks:**

- Add invoice number, invoice date, customer payment terms, due date, tax breakdown, total, and Sales Order reference.
- Provide a UI action to create an invoice from a fully received Sales Order.
- Allow add/remove line adjustments only before invoice posting, if this business rule remains required.
- Ensure invoice values match the final invoice snapshot and do not change with product or tax configuration updates.
- Prevent invoice creation when receiving is incomplete.
- Add invoice status and posting behavior.

**Dependencies:** Full receiving and actual profitability complete; 030 cross-slice receivables command approved.

**Ready for development when:** Invoice posting, line adjustment, numbering, tax, and payment-term rules are approved.
