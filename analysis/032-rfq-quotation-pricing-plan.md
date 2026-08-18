# 032 RFQ and Quotation Pricing Plan

## Epic E-03: Quote-to-Price

### PBI 01: RFQ capture

**Goal:** Capture the customer's requested products and quantities.

**Tasks:**

- Add `rfqs` and `rfq_lines`.
- Store customer account, request date, requested delivery date, notes, and status.
- Add RFQ line product, requested quantity, and UOM.
- Add create, view, and list flows.
- Validate active customer and product references.

**Dependencies:** 031 product master and existing company-account slice.

**Ready for development when:** RFQ fields, status behavior, and whether RFQ is mandatory before quotation are approved.

### PBI 02: Quotation and quotation lines

**Goal:** Prepare the actual customer offer.

**Tasks:**

- Add `quotations` and `quotation_lines`.
- Store quotation number, customer, validity date, payment terms, delivery terms, tax default, status, and notes.
- Store customer unit price, quantity, line total, tax code, tax rate, and VAT-inclusive total.
- Support adding and removing lines while the quotation is editable.
- Calculate quotation subtotal, tax, total, and estimated margin.
- Keep quotation customer prices independent from current supplier prices.

**Dependencies:** RFQ and supplier-product reference price behavior complete.

**Ready for development when:** Price display convention, tax inclusion, discount handling, quotation status, and payment-term fields are approved.

### PBI 03: Commission calculation

**Goal:** Calculate internal commission without exposing it as a customer price component.

**Tasks:**

- Support `PER_UNIT`, `PERCENTAGE`, and `FIXED_QUOTATION`.
- Calculate per-unit commission from final quotation quantity.
- Calculate percentage commission from the agreed VAT-inclusive base.
- Define whether discounts and delivery charges are included in that base.
- Store commission amount as a quotation transaction snapshot.
- Report profit before commission and profit after commission.
- Do not include commission in supplier product cost.

**Dependencies:** Quotation line totals and tax calculations complete.

**Ready for development when:** Percentage base, discount treatment, delivery treatment, and fixed-commission reporting are approved.

### PBI 04: Quotation lifecycle

**Goal:** Control quotation editing and acceptance.

**Tasks:**

- Implement `DRAFT`, `SENT`, `ACCEPTED`, `EXPIRED`, `REJECTED`, and `CANCELLED` as required by the final workflow.
- Permit editing only before Sales Order creation.
- Lock the quotation when Sales Order creation succeeds.
- Store a direct quotation reference on the Sales Order.
- Add audit events for send, acceptance, and cancellation.

**Dependencies:** 033 Sales Order conversion contract must be agreed before acceptance is implemented.

**Ready for development when:** Locking behavior and allowed status transitions are approved.
