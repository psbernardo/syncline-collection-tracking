# 037 Trading MVP Release Plan

## Epic E-08: Reporting, Reliability, and Release

### PBI 01: MVP operational reporting

**Goal:** Give the owner visibility into open commercial and fulfillment work.

**Tasks:**

- Add quotation status and conversion counts.
- Add Sales Orders awaiting allocation, procurement, receipt, and invoicing.
- Add supplier PO outstanding and received views.
- Add estimated versus actual profit and margin views.
- Add commission reporting before and after commission expense.
- Keep existing receivable dashboard as the collection view.

**Dependencies:** 032 through 036 complete.

**Ready for development when:** Report definitions and source-of-truth calculations are approved.

### PBI 02: Cross-slice audit and concurrency hardening

**Goal:** Protect commercial and financial records from silent changes and duplicate commands.

**Tasks:**

- Add rowversion checks to editable trading records.
- Apply existing idempotency conventions to conversion, allocation, receiving, and invoice commands.
- Write audit events in the same transaction as state changes.
- Test duplicate submissions, stale edits, failed audit writes, and rollback behavior.
- Ensure accepted quotations, posted invoices, and historical costs are immutable.

**Dependencies:** All trading commands implemented.

**Ready for development when:** The repository's audit, idempotency, and concurrency contracts are applied consistently.

### PBI 03: Authentication and access boundary

**Goal:** Ensure the owner-only MVP is not exposed without the planned authentication baseline.

**Tasks:**

- Complete the existing technical-plan authentication requirements.
- Protect trading, invoice, and receivable routes with the single-admin middleware.
- Add CSRF protection to state-changing forms.
- Verify secrets are environment-managed and never stored in the database or source.

**Dependencies:** Trading routes exist; follow the authentication requirements in `analysis/002-technical-plan.md`.

**Ready for development when:** Login, session, CSRF, and route protection tests pass.

### PBI 04: End-to-end MVP verification

**Goal:** Verify the complete happy path and material failure paths.

**Tasks:**

- Run: customer account -> product/supplier -> RFQ -> quotation -> customer PO -> Sales Order.
- Run: Sales Order -> split supplier allocation -> supplier POs -> full receiving.
- Run: received Sales Order -> invoice -> delivery receivable -> full payment.
- Verify VAT-inclusive percentage commission and fixed quotation commission.
- Verify tax codes and the current `GOV_6` placeholder.
- Verify accepted quotation locking and transaction snapshots.
- Verify incomplete receiving blocks invoice creation.
- Run `go test ./...`, `go vet ./...`, migrations, and manual SQL Server verification.
- Reconcile implementation state in `analysis/000-implementation-ladder.md`.

**Dependencies:** All prior sequences complete and all open business decisions resolved.

**Ready for release when:** The complete acceptance suite passes and the owner signs off the actual workflow.
