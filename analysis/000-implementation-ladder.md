# Implementation Ladder

This is the canonical implementation tracker for the project. The three-digit filename prefix is the sequence identifier. Sequence numbers describe the intended dependency and observed implementation order, not a claim that every step was committed separately. The early plans were introduced together in one commit, so their exact intra-commit order is inferred from dependencies and code structure.

## How To Read This File

- `KB`: approved durable knowledge. Read it as a constraint.
- `Implemented`: code exists in the current worktree and the plan is substantially reflected in it.
- `Partial`: some code exists, but verification or the full plan is incomplete.
- `Planned`: documentation exists, but implementation evidence is not present.
- `Active worktree`: changes are currently uncommitted; do not treat them as an implemented milestone.
- Sequence numbers are stable identifiers. Add new work at the end; do not renumber history.

## Ladder

| Sequence | State | Capability / plan | Evidence | Confidence |
|---:|---|---|---|---|
| 001 | KB | Approved business domain | [`knowledge-base/001-business-domain.md`](./knowledge-base/001-business-domain.md) | High |
| 002 | Implemented | Technical architecture and vertical-slice constraints | [`002-technical-plan.md`](./002-technical-plan.md); `internal/` follows slice structure | High |
| 003 | Implemented | First vertical slice implementation plan | [`003-first-vertical-slice-plan.md`](./003-first-vertical-slice-plan.md); account and receivable slices exist | Medium |
| 004 | Implemented | Vertical-slice coding standard | [`004-vertical-slice-coding-standard.md`](./004-vertical-slice-coding-standard.md); code follows the documented boundaries | High |
| 005 | Implemented | Tailwind UI foundation | [`005-tailwind-ui-plan.md`](./005-tailwind-ui-plan.md); shared templates and local assets exist | Medium |
| 006 | Implemented | Database schema contract | [`006-database-schema.md`](./006-database-schema.md); schema migrations implement the core contract | High |
| 007 | Implemented | Migration foundation | [`007-migration-implementation-plan.md`](./007-migration-implementation-plan.md); migrations `0001` through `0005` are registered | High |
| 008 | Implemented | Company account create/list slice | [`008-company-account-slice-plan.md`](./008-company-account-slice-plan.md); `internal/slices/accounts` | High |
| 009 | Implemented | Delivery receivable create/list/detail slice | [`009-delivery-receivable-slice-plan.md`](./009-delivery-receivable-slice-plan.md); `internal/slices/receivables` | High |
| 010 | Implemented | Shared navigation bar | [`010-navigation-bar-plan.md`](./010-navigation-bar-plan.md); shared navigation template exists | High |
| 011 | Implemented | Receivable classification | [`011-receivable-classification-plan.md`](./011-receivable-classification-plan.md); typed classification and tests exist | High |
| 012 | Partial | Dashboard totals | [`012-dashboard-totals-plan.md`](./012-dashboard-totals-plan.md); repository, cards, HTMX, and tests exist; live SQL/browser verification remains | High |
| 013 | Partial | Receivable filters | [`013-receivable-filter-plan.md`](./013-receivable-filter-plan.md); filtering and cursor UI exist; live SQL verification remains | High |
| 014 | Implemented | Reusable multi-select dropdown | [`014-multi-select-dropdown-plan.md`](./014-multi-select-dropdown-plan.md); shared component and tests exist | High |
| 015 | Implemented | Company account editing | [`015-company-account-edit-plan.md`](./015-company-account-edit-plan.md); update route, concurrency, audit/idempotency, and tests exist | High |
| 016 | Partial | Duplicate PO handling | [`016-duplicate-po-handling-plan.md`](./016-duplicate-po-handling-plan.md); application and migration work exists | Medium |
| 017 | Partial | Feedback snackbar | [`017-feedback-snackbar-plan.md`](./017-feedback-snackbar-plan.md); reusable behavior exists, but tracker/browser verification needs reconciliation | Medium |
| 018 | Partial | Receivable list editing | [`018-receivable-list-edit-plan.md`](./018-receivable-list-edit-plan.md); code exists, live verification remains | High |
| 019 | Planned | Duplicate invoice-number validation | [`019-duplicate-invoice-number-validation-plan.md`](./019-duplicate-invoice-number-validation-plan.md); no separate committed implementation evidence | High |
| 020 | Partial | Receivable invoice-number column | [`020-receivable-new-column-plan.md`](./020-receivable-new-column-plan.md); code and migrations `0004`/`0005` exist, live verification remains | High |
| 021 | Implemented | Full payment acknowledgement command | [`021-receivable-payment-plan.md`](./021-receivable-payment-plan.md); committed in `b7f5399` and present in receivables code | High |
| 022 | Implemented | Full payment acknowledgement page | [`022-receivable-payment-page-plan.md`](./022-receivable-payment-page-plan.md); dedicated payment page exists | High |
| 023 | Planned | Reverse payment acknowledgement | [`023-reverse-payment-acknowledgement-plan.md`](./023-reverse-payment-acknowledgement-plan.md); no committed implementation evidence | High |
| 024 | Planned | Receivable tax rules | [`024-receivable-tax-rule-plan.md`](./024-receivable-tax-rule-plan.md); current tax work is uncommitted | High |
| 025 | Planned | Receivable tax preview | [`025-receivable-tax-preview-plan.md`](./025-receivable-tax-preview-plan.md); current tax work is uncommitted | High |
| 026 | Planned | VAT calculation | [`026-vat-tax-calculation-plan.md`](./026-vat-tax-calculation-plan.md); current tax work is uncommitted | High |
| 027 | Planned | Payment acknowledgement tax rule | [`027-acknowledge-payment-tax-rule-plan.md`](./027-acknowledge-payment-tax-rule-plan.md); current tax work is uncommitted | High |
| 029 | Planned | Trading MVP ladder and scope | [`029-trading-mvp-ladder.md`](./029-trading-mvp-ladder.md); dependency-ordered MVP plan | High |
| 030 | Planned | Trading foundation and cross-slice contract | [`030-trading-foundation-plan.md`](./030-trading-foundation-plan.md); decisions, snapshots, and receivable handoff boundary | High |
| 031 | Partial | Product, supplier, and supplier-price master data | [`031-trading-master-data-plan.md`](./031-trading-master-data-plan.md); `internal/slices/products`, `internal/slices/suppliers`, migration `0007`, route/UI tests, `go test -count=1 ./...`, and `go vet ./...` pass; live migration verification is blocked by SQL Server authentication for `CTS_DEV` | High |
| 032 | Planned | RFQ, quotation, tax, and commission pricing | [`032-rfq-quotation-pricing-plan.md`](./032-rfq-quotation-pricing-plan.md); no `rfqs`/`quotations` slices, migration, or implementation evidence exists | High |
| 033 | Planned | Customer PO and Sales Order conversion | [`033-sales-order-plan.md`](./033-sales-order-plan.md); final commercial transaction plan | High |
| 034 | Planned | Multi-supplier procurement and allocation | [`034-procurement-allocation-plan.md`](./034-procurement-allocation-plan.md); supplier PO and split-allocation plan | High |
| 035 | Planned | Full receiving and invoice creation | [`035-receiving-invoice-plan.md`](./035-receiving-invoice-plan.md); fulfillment and invoice plan | High |
| 036 | Planned | Invoice-to-receivable collection handoff | [`036-receivable-handoff-plan.md`](./036-receivable-handoff-plan.md); integration with existing receivables | High |
| 037 | Planned | Trading MVP reporting, hardening, and release | [`037-trading-mvp-release-plan.md`](./037-trading-mvp-release-plan.md); final verification plan | High |
| 038 | Partial | Supplier-product searchable selectors | [`038-supplier-product-searchable-dropdown-plan.md`](./038-supplier-product-searchable-dropdown-plan.md); searchable single-select UI, active option loading, strict ID validation, and relationship edit identity rules exist; browser verification remains | High |
| 046 | Partial | Quotation allocation and repeat sales-order creation | [`046-quotation-sales-order-conversion-plan.md`](./046-quotation-sales-order-conversion-plan.md); migration `0018`, allocation-aware quotation conversion, standalone repeat-order route, shared sales-order pages, and tests exist in the active worktree; live SQL/browser verification remains | High |
| 049 | Partial | Auto-generated sales-order numbering | [`049-sales-order-number-plan.md`](./049-sales-order-number-plan.md); sequence preview, immutable edit display, and PDF sales-order metadata are implemented in the active worktree; live SQL/browser verification remains | High |
| 054 | Planned | Purchase-order multi-source and direct purchasing | [`054-purchase-order-multi-source-direct-purchase-plan.md`](./054-purchase-order-multi-source-direct-purchase-plan.md); implementation not started | High |
| 055 | Partial | Supplier-scoped bulk product configuration | [`055-supplier-bulk-product-configuration-plan.md`](./055-supplier-bulk-product-configuration-plan.md); supplier catalog routes/UI, set-based bulk creation, zero-cost defaults, audit events, and automated tests pass; live SQL/browser verification remains | High |
| 061 | Planned | Sales-order duplicate process | [`061-sales-order-duplicate-plan.md`](./061-sales-order-duplicate-plan.md); source-order duplication plan, no implementation started | High |

## Current State

- The strongest implemented boundary is full-payment acknowledgement at sequence `021`.
- Sequences `012`, `013`, `016`, and `018` require verification or tracker reconciliation before being called complete.
- Sequence `024` must not be described as implemented until the current worktree changes are reviewed, tested, and committed.
- Authentication is required by the technical plan but is not represented by an implementation sequence yet; it is a remaining prerequisite before wider use.
- Migrations `0006` and `0007`, the tax package, and trading master-data slices are active worktree changes and are intentionally excluded from the completed history.
- Verification on 2026-08-16: `go test -count=1 ./...`, `go vet ./...`, migration registry tests, and `GET /health` passed; live migration execution was attempted but blocked by SQL Server login failure for `CTS_DEV`.

## AI State Contract

When describing the project, use this precedence:

1. Current source code and tests show what exists.
2. This ladder gives the sequence and implementation status.
3. Approved knowledge-base documents constrain business behavior.
4. Individual plans describe intended work and may be stale or incomplete.
5. Git status distinguishes committed milestones from active worktree changes.

Never infer implementation from a plan filename, an `Approved` sentence, or a completed plan row alone.

## Recording New Work

Add one row at the end with a new sequence number, link the plan, name concrete code/test evidence, and state what is still unverified. Update the row when implementation changes; do not silently change an earlier sequence's meaning.
