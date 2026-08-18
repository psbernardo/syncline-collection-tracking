# 034 Procurement and Allocation Plan

## Epic E-05: Multi-Supplier Procurement

### PBI 01: Supplier PO creation

**Goal:** Create one or more supplier POs from a confirmed Sales Order.

**Tasks:**

- Add `purchase_orders` and `purchase_order_lines`.
- Store supplier, PO number, order date, payment terms, expected delivery date, status, and notes.
- Store supplier unit cost and line total as transaction snapshots.
- Support multiple supplier POs for one Sales Order.
- Prevent procurement from inactive suppliers or inactive supplier-product relationships.
- Add create, view, list, confirm, and cancel actions.

**Dependencies:** 033 Sales Order status and 031 supplier-product data complete.

**Ready for development when:** Supplier PO statuses, numbering, payment terms, and cancellation rules are approved.

### PBI 02: Sales-line to supplier-line allocation

**Goal:** Allow the owner to decide how each Sales Order quantity is procured.

**Tasks:**

- Add an allocation relationship between Sales Order lines and Purchase Order lines.
- Add UI to allocate one Sales Order line to one or more suppliers.
- Validate allocated quantity does not exceed Sales Order quantity.
- Display ordered, allocated, and unallocated quantities.
- Prevent confirmation when required customer quantities remain unallocated.
- Preserve the allocation used for actual cost and fulfillment reporting.

**Dependencies:** Purchase Order line model complete.

**Ready for development when:** Split allocation behavior, quantity validation, and allocation edit locks are approved.

### PBI 03: Procurement status tracking

**Goal:** Track supplier commitment without implementing supplier automation.

**Tasks:**

- Track `DRAFT`, `SENT`, `CONFIRMED`, `REJECTED`, `RECEIVING`, `RECEIVED`, and `CANCELLED` as needed.
- Allow manual supplier confirmation and expected delivery updates.
- Show procurement readiness for each Sales Order.
- Record supplier cost changes as explicit adjustments, not silent overwrites.
- Add audit events for PO creation, confirmation, changes, and cancellation.

**Dependencies:** 034 PBI 01 and PBI 02 complete.

**Ready for development when:** Manual supplier status behavior and cost-change rules are approved.
