# Collection Tracking Development Progress

This tracker follows the approved business plan in `analysis/plan.md` and technical plan in `analysis/technical-plan.md`.

Statuses:

- `Completed`: finished and verified.
- `In Progress`: currently being implemented.
- `Pending`: not started.
- `Blocked`: waiting on an external decision, environment, or dependency.
- `Deferred`: intentionally moved to a later phase.

## Current Priority

**Priority 1: Database migration foundation**

The next development work is the separate migration executable and the first SQL Server schema migration. Do not build GORM repositories before the migration schema is reviewed and applied to the development database.

## Progress Summary

| Priority | Task | Status | Dependency / Next action |
|---|---|---|---|
| P0 | Business domain and workflow plan | Completed | Approved in `analysis/plan.md` |
| P0 | Technical and architecture plan | Completed | Approved in `analysis/technical-plan.md` |
| P0 | First vertical slice implementation plan | Completed | Defined in `analysis/first-vertical-slice-plan.md` |
| P0 | Tailwind UI integration plan | Completed | Windmill Dashboard HTML selected in `analysis/tailwind-ui-plan.md` |
| P0 | Vertical slice coding standard | Completed | Defined in `analysis/vertical-slice-coding-standard.md` |
| P1 | Company account slice implementation plan | Completed | Defined in `analysis/company-account-slice-plan.md` |
| P1 | Edit company account implementation plan | Completed | Defined in `analysis/company-account-edit-plan.md` |
| P1 | Create delivery receivable implementation plan | Completed | Defined in `analysis/delivery-receivable-slice-plan.md` |
| P1 | Shared navigation bar plan | Completed | Defined in `analysis/navigation-bar-plan.md` |
| P0 | Go 1.25.7 module baseline | Completed | Declared in `go.mod` |
| P0 | UTC and Philippines date helpers | Completed | Unit tests passing |
| P0 | Scaled PHP money helpers | Completed | Unit tests passing |
| P0 | Local HTMX and Alpine.js assets | Completed | HTMX 2.0.10 and Alpine.js 3.15.12 vendored locally |
| P0 | `.env` setup | Completed | File exists and is ignored by Git |
| P0 | Development/test SQL Server database | Completed | Development and testing currently share local database `CTS_DEV` |
| P0 | SQL authentication credentials | Completed | Configured in ignored `.env` |
| P1 | Migration executable | Completed | `cmd/migrate` is working with `.env` defaults and versioned migrations |
| P1 | Migration configuration plan | Completed | No-argument `.env` loading plan in `analysis/migration-implementation-plan.md` |
| P1 | First database schema migration | Completed | Migration `0001` implemented and working |
| P1 | Migration status/up verification | Completed | Migration versions `0001-0003` verified against `CTS_DEV` |
| P1 | Database schema review | Completed | Defined in `analysis/database-schema.md` and implemented in migration `0001` |
| P1 | Server composition root | In Progress | `cmd/server` and local account routes implemented; browser verification pending |
| P2 | Company account slice | Completed | Create, list, edit, validation, rowversion, templates, audit/idempotency flow, and tests implemented |
| P2 | Delivery receivable slice | In Progress | Domain, create/list/detail flow, helpers, audit/idempotency, templates, and tests implemented |
| P2 | Dashboard aggregation slice | Pending | Depends on receivable persistence |
| P2 | Audit transaction implementation | In Progress | Company-account create command writes audit atomically |
| P2 | Idempotent command implementation | In Progress | Company-account create command uses idempotency keys |
| P2 | Edit company account implementation | Completed | Update route, rowversion conflict handling, audit, idempotency, and tests implemented |
| P2 | Create delivery receivable implementation | In Progress | Create/list/detail route, due-date/money validation, audit/idempotency, and tests implemented |
| P2 | Shared navigation bar implementation | Completed | Shared embedded layout/navigation used by account and receivable pages |
| P2 | Receivable classification display plan | Completed | Defined in `analysis/receivable-classification-plan.md` |
| P2 | Receivable classification display implementation | Completed | Typed server-side classification, day counters, status badges, and boundary tests implemented |
| P2 | Dashboard totals plan | Completed | Defined in `analysis/dashboard-totals-plan.md` |
| P2 | Dashboard totals implementation | In Progress | Aggregation repository, summary cards, HTMX refresh, and tests implemented; live SQL verification pending |
| P2 | Receivable filters plan | Completed | Defined in `analysis/receivable-filter-plan.md` |
| P2 | Receivable filters implementation | Completed | Company, PO, multi-status filters, server-side query, normalized URL state, HTMX results, and cursor UI implemented; live SQL verification pending |
| P2 | Reusable multi-select dropdown plan | Completed | Defined in `analysis/multi-select-dropdown-plan.md` |
| P2 | Reusable multi-select dropdown implementation | Completed | Shared server-rendered component, Alpine add/remove behavior, native fallback, receivables status integration, and template coverage implemented |
| P2 | Receivable list company-name and edit plan | Completed | Defined in `analysis/receivable-list-edit-plan.md` |
| P2 | Duplicate PO handling plan | Completed | Defined in `analysis/duplicate-po-handling-plan.md`; global uniqueness is the current assumption |
| P2 | Reusable feedback snackbar plan | Completed | Defined in `analysis/feedback-snackbar-plan.md`; duplicate PO is the first use case |
| P2 | Reusable feedback snackbar implementation | Completed | Global Alpine queue, HTMX non-2xx handling, response feedback headers, and fade/dismiss behavior implemented |
| P1 | Duplicate PO data cleanup | Completed | Existing PO `001` conflict resolved before migration `0003` |
| P2 | Duplicate PO handling implementation | In Progress | Application checks and migration are complete; user feedback snackbar remains to be implemented |
| P2 | Receivable list company-name and edit implementation | In Progress | Company-name projections and edit flow implemented; live SQL/browser verification pending |
| P3 | Authentication middleware | Deferred | Add during the middle of development before wider use |
| P3 | Manual SQL Server integration verification | Completed | `go run ./cmd/migrate` connected to `CTS_DEV`; rollback remains intentionally unrun on shared data |
| P3 | Production deployment hardening | Deferred | Local-only MVP; backups and graceful shutdown remain out of scope |

## Remaining Implementation Tasks

### P1: Usable Application Foundation

| Order | Task | Status | Completion condition |
|---:|---|---|---|
| 1 | Confirm migration schema manually | In Progress | Tables, constraints, indexes, and rollback behavior verified in SQL Server |
| 2 | Create server composition root | Pending | `cmd/server` loads `.env`, opens GORM, registers routes, and serves HTTP on `127.0.0.1:8080` |
| 3 | Add shared templates/layout | Pending | `html/template` layout, navigation, flash/validation messages, and local assets wired |

### P2: First Usable Vertical Slice

| Order | Task | Status | Completion condition |
|---:|---|---|---|
| 4 | Company account slice | Pending | Create, edit, view, validate, and list required company fields |
| 5 | Delivery receivable slice | Pending | Enter company, PO, amount, delivery date, and term; calculate and persist due date |
| 6 | Daily classification logic | Completed | Date and money helpers already implemented and tested |
| 7 | Dashboard slice | Pending | Show client totals for pending, near due, overdue, and payment received |
| 8 | PO drill-down | Pending | Show receivables for a client with filtering and load-more pagination |

### P2: Collection Operations

| Order | Task | Status | Completion condition |
|---:|---|---|---|
| 9 | Full payment command | Pending | Record payment date and derive `Payment Received` |
| 10 | Cancel and reopen commands | Pending | Update lifecycle status only and recalculate active classification after reopen |
| 11 | Receivable archive and restore | Pending | Update `Archived`/`Active` status only and audit both actions |
| 12 | Audit transaction implementation | Pending | Every material command commits its audit event atomically |
| 13 | Idempotent command implementation | Pending | Repeated submissions return the original result without duplicate changes |

### P3: Verification and Safety

| Order | Task | Status | Completion condition |
|---:|---|---|---|
| 14 | Repository tests with `sqlmock` | Pending | CRUD, filters, aggregates, errors, and transactions covered |
| 15 | HTTP and HTMX tests | Pending | Forms, validation, redirects, fragments, and dashboard rendering covered |
| 16 | Manual SQL Server integration verification | Completed | Migration connected to `CTS_DEV` and the safe `up` path completed; destructive rollback remains unrun |
| 17 | Authentication middleware | Deferred | Add before the system is used beyond the owner's local development workflow |

## Next Task

Start with **Task 1: Confirm migration schema manually**, then implement **Task 2: Server composition root** and proceed through the first usable vertical slice.

## Migration Foundation Status

- SQL authentication values are configured in ignored `.env`.
- Migration version `0001` is implemented with the approved tables, constraints, indexes, audit schema, idempotency schema, and rowversion.
- `cmd/migrate` has no arguments and applies pending migrations using `.env` and its tagged defaults.
- Migration locking, rollback behavior, and registry tests are implemented.
- Manual SQL Server verification against shared `CTS_DEV` is complete for the safe `up` path; rollback remains intentionally unrun.

## Definition Of Done: Migration Foundation

The migration foundation is complete when:

- `cmd/migrate` builds and applies pending migrations with no arguments.
- Development and testing use the shared local `CTS_DEV` database for now.
- Migration version `0001` applies successfully to `CTS_DEV`.
- The schema contains all required tables, constraints, indexes, and concurrency fields.
- A failed migration rolls back safely and reports a clear error.
- The migration registry and internal status/down behavior are covered by tests; destructive rollback is not run against shared `CTS_DEV`.
- The migration does not run automatically when the server starts.
- Manual verification results are recorded in this file.

## Next Milestone

After the migration foundation is complete, implement the first vertical slice:

```text
Create company account
-> Create delivery receivable
-> Calculate due date
-> Persist receivable
-> Display Pending / Near Due / Overdue classification
```

## Decision And Verification Log

| Date | Item | Result | Notes |
|---|---|---|---|
| TBD | Development tracker created | Open | Database migration is Priority 1 |
| 2026-08-10 | Migration status connection check | Blocked | `CTS_DEV` configuration loaded, but `localhost:1433` refused the TCP connection |
