# Collection Tracking System: Technical and Architecture Plan

## 1. Purpose

This document defines the technical architecture, project structure, development workflow, environment strategy, migration strategy, and testing standards for the collection tracking system.

The business domain is defined in [`analysis/knowledge-base/001-business-domain.md`](./knowledge-base/001-business-domain.md). This document does not change those business decisions. If a technical choice exposes a business ambiguity, record the question in Section 18 and resolve it before implementation.

## 2. Confirmed Technical Direction

- Language: Go.
- Web interaction: server-rendered HTML enhanced with HTMX.
- Browser behavior: Alpine.js for small local interactions only.
- Database: Microsoft SQL Server, installed locally for development.
- ORM: GORM.
- Migration executable: a separate Go executable using GORM's migration facilities.
- Database unit testing: `sqlmock`.
- Configuration: environment variables loaded from `.env` during local development with `github.com/joho/godotenv`.
- Environments: separate test and production configuration.
- Architecture: vertical slice architecture with clean dependency rules inside each slice.
- Development method: test-driven development.

Confirmed platform decisions:

- Project baseline: Go 1.25.7 on Windows `amd64`.
- `go.mod` must declare the Go 1.25.7 baseline; development and CI toolchains must use the same version.
- Initial deployment: owner's local machine only.
- Local database authentication: SQL Server SQL authentication.
- Server-side templates: Go `html/template`.
- HTMX and Alpine.js assets: served locally from the application; no internet connection required at runtime.
- Local SQL Server integration tests are not required in the initial CI workflow.
- HTTP routing: standard library `net/http`; no Gin or Echo dependency.
- Authentication: required for the single-admin application.
- Authentication credentials: managed through environment variables.

### Initial dependency baseline

Pin direct dependencies in `go.mod` and review upgrades deliberately:

- `gorm.io/gorm`
- `gorm.io/driver/sqlserver`
- `github.com/microsoft/go-mssqldb`
- `github.com/DATA-DOG/go-sqlmock`
- `github.com/joho/godotenv`
- `github.com/caarlos0/env/v11`
- `golang.org/x/crypto/bcrypt`

Use the SQL Server driver supported by the selected GORM release. Do not add a dependency merely to avoid a small standard-library implementation.

## 3. Architecture Style

Use a modular monolith organized by vertical business capability. Each slice owns the complete path for one user capability, from HTTP request through application behavior to persistence, instead of grouping the entire application into global `handlers`, `services`, and `repositories` layers.

The first release should use these slices:

- `accounts`: create, edit, and view company accounts.
- `auth`: local single-admin login and session handling.
- `receivables`: create and edit delivery receivables, calculate due dates, and archive records.
- `payments`: record full payment dates and reopen or correct payment state.
- `cancellations`: cancel and reopen receivables with audit events.
- `dashboard`: calculate client-aggregated pending, near-due, overdue, and payment-received views with PO drill-down.

Each slice may contain its own handler, command/query logic, domain rules, repository interface, GORM adapter, templates, and tests. Shared code is allowed only when it is genuinely cross-cutting and stable.

### Clean rules inside each slice

```text
slice HTTP handler/template
          |
          v
slice command/query handler
          |
          v
slice domain rules and ports
          ^
          |
slice GORM / SQL Server adapter
```

Rules:

- The domain portion of a slice must not import GORM, HTTP, Alpine.js, or SQL Server packages.
- Slice handlers translate transport input into commands or queries; they do not contain business rules.
- Slice persistence adapters implement ports owned by the slice.
- Templates receive slice view models, not GORM models.
- A slice should not reach into another slice's persistence implementation.
- Cross-slice coordination happens through application-level commands, queries, or explicit shared ports.
- The composition root wires slice dependencies together.
- Do not create abstractions speculatively; extract shared code only after a real second use case exists.

## 4. Proposed Project Structure

```text
.
├── cmd/
│   ├── server/
│   │   └── main.go              # HTTP application entry point
│   ├── migrate/
│       └── main.go              # Separate migration executable
│   └── password-hash/
│       └── main.go              # Local utility for generating ADMIN_PASSWORD_HASH
├── internal/
│   ├── config/                  # Environment loading and validation
│   ├── platform/                # Shared database, clock, logging, HTTP helpers
│   │   ├── database/
│   │   ├── clock/
│   │   └── logging/
│   ├── shared/                  # Small cross-slice domain primitives
│   │   ├── businessdate/
│   │   └── money/
│   ├── slices/
│   │   ├── accounts/
│   │   │   ├── domain.go
│   │   │   ├── commands.go
│   │   │   ├── queries.go
│   │   │   ├── repository.go
│   │   │   ├── repository_gorm.go
│   │   │   ├── handler.go
│   │   │   ├── models.go
│   │   │   ├── templates/
│   │   │   └── *_test.go
│   │   ├── auth/
│   │   ├── receivables/
│   │   ├── payments/
│   │   ├── cancellations/
│   │   └── dashboard/
│   ├── migrations/              # Versioned migration definitions
│   └── web/
│       └── static/               # HTMX and Alpine.js assets
├── tests/
│   └── integration/              # SQL Server-backed tests, if added
├── .env.example
├── .env.example
├── .gitignore
├── go.mod
└── README.md
```

`internal/` prevents application packages from being imported by external consumers. Keep slices cohesive and avoid generic `utils`, `services`, or `repositories` packages that hide feature ownership.

## 5. Domain Model Mapping

### Company account

Required business fields:

- Company name
- Contact person
- TIN number
- Billing address
- Delivery address
- Contact number

The account owns many delivery receivables.

Company-account archiving is out of scope for the current MVP. The system initially supports only 1-4 active client accounts, so company accounts remain active and are not archived through the application.

### Delivery receivable

Required business fields:

- Internal identifier
- Company account identifier
- Invoice number, optional trimmed ASCII-alphanumeric free text; may be supplied manually or sourced from a selected invoice
- PO number, mandatory alphanumeric free text
- Delivery date
- Payment term, whole number from 1 to 120 calendar days
- Amount due in PHP, stored as an integer scaled by 10,000
- Calculated due date
- Payment date, optional and empty until fully paid
- Persisted lifecycle status: `Active`, `Cancelled`, or `Archived`
- Derived dashboard classification: `Pending`, `Overdue`, `Near Due`, or `Payment Received`
- Created/updated timestamps

The due date rule is:

```text
due date = delivery date + (payment term - 1) calendar days
```

Persist timestamps in UTC and convert them to `Asia/Manila` with Go's `time.LoadLocation` and `Time.In` when records are fetched for display or business-date classification. Delivery, due, and payment dates are business dates; their persistence representation must be consistent with this UTC conversion rule.

Date conversion contract:

1. Parse an input date as a date-only value.
2. Interpret that date at midnight in `Asia/Manila`.
3. Convert the resulting instant to UTC before saving to SQL Server.
4. On read, interpret the stored value as UTC, convert it to `Asia/Manila`, and use the local calendar date for display and business rules.

### Payment representation

The MVP does not support partial payments or payment proof. The payment date is optional when the receivable is created and can be recorded later when the full payment is received. The simplest model is a nullable full-payment date on the receivable rather than a separate payment table.

Use a separate payment table only if payment history, multiple payment attempts, or future partial payments are required. This is a deliberate design decision to avoid building unsupported accounting behavior.

### Money representation and display

- Store PHP amounts as `int64` values in units of 1/10,000 PHP, with non-negative values enforced for receivables.
- Convert user input to the scaled integer in one domain/application boundary function; do not parse money independently in handlers and templates.
- Display values by dividing by 10,000 and formatting to exactly two decimal places with thousands separators.
- Example: stored `98003439` represents `9800.3439` and displays as `9,800.34`.
- Round input values with more than four decimal places to the nearest 1/10,000 PHP before storage, using one centralized conversion function. For non-negative amounts, exact half-way values round up.
- Reject negative values; test exact half-way values and boundary overflow behavior.
- The UI display is rounded to two decimal places, but the stored four-decimal value remains the source of truth.

### Derived classifications

- `Payment Received`: full payment date exists.
- `Overdue`: no payment date and the current Philippines date is after the due date.
- `Near Due`: no payment date and the due date is today through five calendar days from today.
- `Pending`: no payment date and the due date is after the near-due window.
- `Cancelled` and `Archived`: persisted lifecycle statuses, excluded from ordinary dashboard totals.

Derive `Pending`, `Overdue`, `Near Due`, and `Payment Received` only for records with lifecycle status `Active`. Persist `Cancelled` and `Archived` lifecycle status values. Do not store derived classifications that can become stale overnight.

### Dashboard aggregation and pagination

- Dashboard rows are aggregated by company account and include receivable count, outstanding amount in scaled PHP units, nearest due date, and a drill-down link for PO numbers.
- Query DTOs should expose only dashboard data required by the template; they should not expose GORM models.
- Use a deterministic sort order with a stable tie-breaker such as company account ID.
- Use a load-more interaction through HTMX rather than loading every client at once.
- Prefer cursor-based pagination for stable results. The cursor must contain the last sort key and tie-breaker, and must be validated server-side.
- Return `items`, `next_cursor`, and `remaining_count` from the dashboard query. The UI displays how many rows remain and disables the load-more control when no cursor remains.
- Use a configurable default page size, initially 25 rows, with a server-enforced maximum.
- The remaining count may use a separate aggregate count query; it must use exactly the same filters as the page query.
- Apply filters at the receivable level before grouping by company, and exclude archived and cancelled receivables from ordinary totals.
- Sum scaled integer amounts before formatting the aggregate; do not sum values already rounded for display.

### Dashboard user experience

The dashboard is the owner's daily starting point. It should answer, without opening every client:

- How much is overdue and how many receivables are overdue?
- Which clients need attention first?
- Which receivables will become due within five days?
- How much remains pending?
- How much has been paid?

Recommended dashboard components:

- Summary cards for overdue, near due, pending, and payment received, each showing both client/receivable counts and PHP amounts.
- A default overdue client list ordered by greatest days overdue, then outstanding amount, then company name.
- Near-due and pending views ordered by nearest due date.
- Search by company name, TIN number, contact person, and PO number.
- Filters for classification, delivery-date range, due-date range, payment-term range, amount range, and archived/cancelled inclusion.
- A client row showing company name, matching receivable count, total outstanding amount, earliest due date, and most urgent classification.
- A client drill-down showing each PO number, amount, delivery date, payment term, due date, payment date, lifecycle status, and derived classification.
- Direct actions from a receivable row to record payment, cancel, reopen, edit, or archive according to its lifecycle state.
- Clear filter state, a reset-filters action, and a load-more control that displays the remaining row count.

Use server-side filtering and aggregation. Alpine.js may control filter visibility, but it must not calculate totals or classifications in the browser.

## 6. Vertical Slice Boundaries

Each slice contains the domain and application behavior needed for its feature. The boundaries below apply inside every slice rather than creating application-wide layers.

### Domain rules inside a slice

Contains entities, value objects, invariants, and pure business calculations:

- Validate payment terms from 1 through 120.
- Validate the PO number as non-empty alphanumeric text.
- Calculate due dates.
- Determine active classification from a supplied current date.
- Prevent payment dates before delivery dates or after the current date.
- Represent cancellation and reopening rules.

Domain code should be deterministic. Pass the current date or a clock interface rather than calling `time.Now()` directly inside business rules.

### Commands and queries inside a slice

Commands change state; queries read state. Keep them separate when their behavior, transaction needs, or performance characteristics differ.

Typical commands and queries include:

- Create/update company account.
- Create delivery receivable.
- Record full payment date.
- Cancel and reopen receivable.
- Archive records.
- List overdue, near-due, pending, and paid dashboard data.
- View client-level aggregation and PO-level drill-down.

Commands own transaction boundaries when a change writes multiple records or an audit event. They return application errors that handlers can map to user-facing validation messages. Queries should use read-specific projections and avoid loading complete domain entities when a dashboard only needs aggregates.

### Adapters inside a slice

- HTTP handlers parse form data, invoke use cases, and return full pages or HTMX fragments.
- GORM repositories map between domain objects and persistence models.
- SQL Server-specific query syntax remains in the persistence adapter.
- Templates render view models and must not call repositories.

### Cross-slice behavior

- A receivable payment command may use a shared transaction and emit an audit event without importing the payments slice internals.
- The dashboard reads the data it needs through explicit query ports; it should not call another slice's GORM repository directly.
- Shared domain concepts should live in a small shared package only when their ownership is genuinely cross-slice.
- Avoid a distributed transaction between slices unless one database transaction is required by a concrete business rule.

### Composition root

`cmd/server/main.go` should load configuration, create logging, connect to SQL Server, construct each slice's dependencies, register slice routes, and start the HTTP server. Keep construction explicit rather than relying on global service locators.

## 7. Database and GORM Strategy

Use GORM with the SQL Server driver and a configured `database/sql` connection pool.

Recommended responsibilities:

- GORM models are persistence-only and may differ from domain entities.
- Explicitly configure connection pool limits and connection lifetime.
- Use context-aware database operations.
- Use transactions for commands that must update multiple tables or write audit events.
- Select only required columns for dashboard aggregation.
- Add indexes for company, PO number, delivery date, due date, payment date, status/cancelled state, and archive state based on actual query plans.
- Store money as an integer scaled by 10,000. For example, `9800.3439` is stored as `98003439`; the UI displays it as `9,800.34` using two decimal places.
- Do not use floating-point types for money. Define the scale constant in the domain and use integer arithmetic for calculations.
- Persist all timestamps and normalized business dates in UTC as SQL Server `datetime2`. Interpret date input at midnight in `Asia/Manila`, convert to UTC before persistence, then convert back to `Asia/Manila` and reduce to a date when fetched.
- Keep all date classification queries based on one `Asia/Manila` business date supplied by the application; do not mix server-local timezone behavior into SQL predicates.
- Define field lengths, nullability, collation, and check constraints in migrations rather than relying on GORM defaults.
- Do not add a unique constraint on PO number alone; PO traceability is scoped to a company and may require future support for repeated POs.

### Suggested initial tables

- `company_accounts`
- `delivery_receivables`
- `audit_events`
- `idempotency_keys`
- `schema_migrations` or the migration table required by the migration runner

`delivery_receivables` should retain the term and calculated due date used at creation. Client account edits must not recalculate historical receivables.

Detailed table and index definitions for migration `0001` are documented in [`analysis/006-database-schema.md`](./006-database-schema.md).

The no-argument migration configuration and `.env` loading plan is documented in [`analysis/007-migration-implementation-plan.md`](./007-migration-implementation-plan.md).

Initial persistence contract:

- Money column: SQL Server `bigint`, storing PHP multiplied by 10,000, with a non-negative check constraint.
- Date/timestamp columns: SQL Server `datetime2`, persisted as UTC according to the conversion contract above.
- Payment date: nullable until full payment is recorded.
- Lifecycle status: constrained to `Active`, `Cancelled`, or `Archived`.
- Concurrency: SQL Server `rowversion` column on editable records.
- Audit event timestamp: UTC `datetime2`.
- Audit previous/new values: structured JSON stored in an appropriate SQL Server text/JSON-compatible column.
- Dashboard indexes: lifecycle status, due-date UTC value, payment-date UTC value, company account ID, PO number, and rowversion-related update paths.

The initial schema should explicitly define how archived and cancelled records are filtered. Dashboard queries must exclude both from ordinary totals and must expose an intentional drill-down filter when historical records are needed. Archive, restoration, and cancellation operations update the record status; they do not delete or move the record to another table.

Restoring an archived receivable changes its lifecycle status from `Archived` to `Active`; its dashboard classification is recalculated from its payment date and due date.

## 8. Separate Migration Executable

The migration tool must be a separately built executable, independent from server startup.

Suggested commands:

```text
go run ./cmd/migrate
```

Rules:

- Server startup must not silently mutate the schema.
- Every migration has a monotonically increasing version and a clear description.
- Migration files are immutable after they have been applied; changes require a new migration.
- The migration runner uses a database lock or equivalent guard so two migration processes cannot run concurrently.
- Migration status reports the current version, pending versions, and failed state clearly.
- Migrations are applied in a transaction where SQL Server supports the operation safely.
- Destructive migrations require an explicit command and a review.
- `down` migrations must be treated as development/recovery tools, not routine production operations.
- The migration executable validates configuration and fails before making changes when the target environment is invalid.
- Migration tests should verify important schema behavior against a local SQL Server instance or a controlled integration database.

GORM's `AutoMigrate` should not be used as an implicit production schema management mechanism. If GORM's migrator is used, wrap it in versioned, reviewed migration definitions and keep the migration history explicit.

### Migration failure and locking behavior

- Use a `schema_migrations` table containing version, description, applied timestamp, and success state.
- Acquire a SQL Server application lock before reading or changing migration state, then release it in a `defer` path.
- Apply one migration at a time inside a transaction where supported.
- On failure, roll back the transaction, retain a clear failed status in the command output, and stop without applying later migrations.
- Recovery requires fixing the migration or database condition, then rerunning the same version; never edit an already-applied migration.
- `down` requires an explicit version/step argument and confirmation in production-like environments.

## 9. HTTP, HTMX, and Alpine.js

Use standard `net/http`. Go 1.25's method/path routing is sufficient for this small local application, and avoiding Gin or Echo reduces dependencies and keeps the architecture transparent. Keep handlers thin.

Recommended request flow:

1. Browser requests a page or submits a form.
2. Handler validates transport-level input and converts it to a use-case command.
3. Use case validates domain rules and performs the operation.
4. Handler returns a redirect after successful normal form submission, or an HTMX fragment when requested.
5. Validation errors re-render the relevant form with field-level messages.

HTMX conventions:

- Use progressive enhancement: core forms should remain understandable without JavaScript.
- Return HTML fragments for partial updates, not JSON unless an actual API is required.
- Use `HX-Redirect` or normal redirects after successful commands to avoid duplicate submissions.
- Keep server-rendered state authoritative.
- Use stable element IDs and clear `hx-target`/`hx-swap` behavior.
- Do not hide validation or authorization rules in browser code.

Alpine.js should be limited to local UI state such as modal visibility, confirmation dialogs, and filter controls. Do not duplicate domain calculations in Alpine.js.

Start with the latest stable HTMX and Alpine.js releases, record their exact versions and licenses, and commit those files under `internal/web/static`. Do not load them from a CDN at runtime.

## 10. Configuration and `.env`

Use a typed configuration struct loaded once at startup.

Suggested variables:

```text
HOST=localhost
PORT=1433
USER=sa
PASSWORD=Admin@12345
DATABASE=CTS_DEV
LOG_LEVEL=info
```

Rules:

- Call `godotenv.Load()` only for local development/test convenience; production should receive environment variables from the process manager or deployment environment.
- The migration executable accepts no arguments and uses `godotenv.Load()` to load `.env`.
- The migration executable uses `.env` for both development and the current shared test database.
- Existing process environment variables should take precedence over values loaded from files.
- Use `.env.example` as a committed documentation template.
- Do not commit `.env`, production secrets, certificates, or database backups.
- Require authentication configuration at startup in environments where login is enabled.
- Prefer `ADMIN_PASSWORD_HASH` over a plain-text password variable. If a plain-text local development password is temporarily supported, never log it and never commit it.
- Require a long, random `SESSION_SECRET`; never use a default value.
- Set `SESSION_COOKIE_SECURE=false` only for local HTTP development. Use `true` whenever HTTPS is used.
- Test configuration parsing without requiring a database.
- Keep test and production database names/credentials separate.
- Never let tests use the production connection string.

Recommended environment files:

- `.env.example`: safe variable names and example values.
- Local `.env`: ignored and developer-specific.
- Production: injected environment variables or a secret manager, not a committed file.

## 11. Environment Separation

### Development

- Local SQL Server database, for example `collection_tracking_dev`.
- Development `.env`.
- Debug logging and local HTTP server.
- Migrations run explicitly with the migration executable.

### Test

- Development and test currently use the same local SQL Server database, `CTS_DEV`, with the same SQL authentication credentials.
- The same `.env` values are used for the current shared `CTS_DEV` development/test database.
- Because the database is shared temporarily, tests must not run destructive migration or cleanup operations against data used by development.
- Unit tests should not require a running database; integration tests may require SQL Server.
- A separate test database should be introduced before automated integration testing or concurrent development/testing is needed.

### Production

- Separate database and credentials.
- Production environment variables supplied outside source control.
- Migrations run as an explicit deployment step before the compatible server version starts.
- Do not run destructive migration commands automatically.
- Configure backups and recovery when production scope is expanded, even though backup behavior is outside the current MVP business scope.

### Local authentication

- Show a login page before accessing application routes.
- Store `ADMIN_USERNAME` and a bcrypt `ADMIN_PASSWORD_HASH` in the environment. Never store the password or hash in source code or SQL Server.
- Provide `cmd/password-hash`, an interactive local utility that reads a password without echoing it and prints only the generated hash.
- Compare the submitted password with `ADMIN_PASSWORD_HASH` using bcrypt verification. Use a generic login failure message for unknown username and wrong password.
- Store only a signed session cookie after successful login; never store the password in the session.
- Keep the cookie `HttpOnly`, `SameSite=Lax`, and `Secure=true` under HTTPS. Use `Secure=false` only for local HTTP development.
- Use an eight-hour idle timeout as the initial recommendation; refresh the expiry only after authenticated activity.
- Logout clears the cookie. Changing `SESSION_SECRET` invalidates all existing sessions.
- Record failed-login events without passwords or sensitive request data. A short uniform delay is recommended; account lockout is unnecessary for the local-only MVP but should be added before network exposure.
- Use a CSRF token for state-changing forms, bound to the authenticated session.
- Keep authentication credentials in environment variables, never in source code or the database for the single-admin MVP.

## 12. Testing Strategy and TDD

Write a failing test before the smallest implementation that makes it pass, then refactor while keeping the test suite green.

### Unit tests

Use standard Go `testing` and table-driven tests for:

- Due-date calculation, including term 1, term 120, and invalid values.
- Philippines date boundaries.
- Near-due boundary: today, five days ahead, and six days ahead.
- Overdue boundary: due date yesterday versus today.
- Payment-date validation.
- UTC/`Asia/Manila` conversion at midnight and date boundaries.
- Cancellation and reopening.
- Archive and restoration as lifecycle status changes.
- PO validation.
- Scaled-money input, four-decimal storage, and two-decimal display behavior.
- Authentication success, failure, session expiry, logout, and CSRF validation.
- Cursor pagination and exact remaining-count behavior.

Domain unit tests should not require GORM, SQL Server, HTTP, or environment files.

### Repository tests

Use `go-sqlmock` for deterministic GORM repository tests:

- Verify SQL behavior and expected arguments.
- Construct GORM over the sqlmock `database/sql` connection using the SQL Server dialector configuration; do not open a real SQL Server connection in unit tests.
- Test successful reads/writes.
- Test `sql.ErrNoRows` or equivalent not-found mapping.
- Test constraint and database error mapping.
- Test transaction commit and rollback behavior.
- Test that queries exclude archived and cancelled records where required.
- Test aggregate dashboard queries and PO drill-down results.

Keep SQL mock expectations focused on behavior rather than copying every incidental ORM-generated detail when possible. Add SQL Server integration tests for queries whose behavior cannot be reliably represented by `sqlmock`.

Keep tests close to their slice. For example, `internal/slices/receivables/` should contain the receivable domain tests, command/query tests, repository tests, and handler tests. Cross-slice integration tests belong in `tests/integration/`.

### HTTP tests

Use `net/http/httptest` with mocked application ports/use cases:

- Valid form submission.
- Invalid form submission and field errors.
- Redirect after successful command.
- HTMX fragment response.
- Dashboard rendering.
- Error response behavior.

### Integration tests

Run against a dedicated local SQL Server test database when validating:

- Actual migrations.
- SQL Server data types and constraints.
- Date and decimal behavior.
- Dashboard aggregation queries.
- Transaction behavior.

Do not replace unit tests with integration tests. Keep integration tests explicit and environment-gated.

## 13. Idempotent Commands and Concurrency

Use an idempotent command pattern for all state-changing form actions, especially create receivable, record payment, cancel, reopen, and archive.

- Generate a unique command key when a form is rendered and submit it as a hidden field or request header.
- Store the key, command type, normalized request hash, result resource identifier, response status, and expiry in `idempotency_keys`.
- Add a unique database constraint on the command key.
- Process the idempotency record, business mutation, and audit event in one transaction.
- If the same key is submitted again with the same request hash, return the original result without repeating the mutation or audit event.
- If the same key is submitted with a different request hash, reject it as a request conflict.
- Expire old idempotency records only after the command replay window is safely complete.
- Keep SQL Server `rowversion` checks for stale edits; idempotency prevents repeated commands, while `rowversion` detects a different edit made after the form was loaded.
- Retry a deadlocked transaction only when the command is safe to retry and the idempotency key is present. Use a small bounded retry count with jitter and return a clear error after the limit.

Tests must cover first submission, identical replay, payload conflict, rollback before the idempotency record commits, stale `rowversion`, and deadlock retry behavior.

## 14. Audit and Data Integrity

Audit events should capture:

- Event identifier.
- Entity type and identifier.
- Action, such as create, update, payment, cancel, reopen, or archive.
- Actor identifier; initially the single administrator.
- Timestamp.
- Previous and new values, preferably as structured JSON or a defined change format.
- Optional note.

The initial `audit_events` schema should include `event_id`, `entity_type`, `entity_id`, `action`, `actor_id`, `occurred_at_utc`, `request_id`, `idempotency_key`, `previous_values_json`, `new_values_json`, and `note`.

Audit is the immutable financial change history. It is not a separate payment ledger in the MVP; payment date remains on the receivable and the payment action is recorded as an audit event.

For every material command, write the business change and its audit event in the same database transaction:

1. Validate the command and load the current record.
2. Capture the previous values.
3. Apply the business change.
4. Insert the audit event with previous values, new values, actor, UTC timestamp, request ID, and idempotency key.
5. Commit both writes together.

If either write fails, roll back both. This prevents a payment, cancellation, or correction from existing without a history entry, or a history entry from claiming a change that did not commit.

Audit events are append-only through the application. The application must expose no update or delete operation for audit events. Corrections create new events instead of rewriting previous events. Avoid hard deletion of financial records; archive records and exclude them from ordinary queries.

## 15. Logging and Error Handling

- Use structured logs with request ID, operation, and error context.
- Never log passwords, connection strings, or sensitive customer data unnecessarily.
- Return user-safe messages from handlers while logging technical details server-side.
- Wrap errors with context using Go's `%w` convention.
- Map expected domain errors to validation responses; do not expose raw SQL errors.
- Recover from unexpected HTTP handler panics at the server boundary and return a safe error response.
- Set request and database timeouts. Graceful shutdown is outside the first-release scope.

## 16. Security and Operational Baseline

- Use parameterized queries through GORM; never concatenate user input into SQL.
- Escape template output by default.
- Add CSRF protection for state-changing form submissions before production exposure.
- Restrict database credentials to the required database and permissions.
- Protect every application route with single-admin authentication middleware except login and static assets.
- Validate and limit request body sizes.
- Use secure cookie settings if sessions are introduced.
- Keep dependencies updated and run `go vet` and a static analyzer in CI.

## 17. Development Workflow

1. Confirm or update the business rule in `analysis/knowledge-base/001-business-domain.md`.
2. Select one vertical slice and write its domain or command/query test first.
3. Implement the smallest passing behavior.
4. Add repository tests with `sqlmock` for persistence behavior.
5. Add handler/template tests for the user flow.
6. Add or update a versioned migration.
7. Run formatting, tests, static analysis, and migration verification.
8. Review the change against the business acceptance scenarios.

The first recommended vertical slice is: create a delivery receivable -> calculate and persist its due date -> display it in the appropriate dashboard classification.

Suggested local commands:

```text
go test ./...
go vet ./...
gofmt -w .
go run ./cmd/migrate
go run ./cmd/server
```

Use a Makefile or PowerShell task file later if repeated commands need standardization. Do not hide important migration or test behavior behind opaque scripts.

## 18. Technical Decisions

The following technical decisions are confirmed:

- Go 1.25.7 on Windows `amd64`.
- The first release runs on the owner's local machine only.
- The migration executable supports `up`, `status`, and `down`.
- Local SQL Server uses SQL authentication.
- Payment date is optional at receivable creation and is stored directly on `delivery_receivables` for the MVP.
- All company fields are mandatory.
- Go `html/template` is used.
- HTMX and Alpine.js are served locally from `internal/web/static`.
- SQL Server integration testing is not part of the initial CI workflow.
- Standard library `net/http` is used instead of Gin or Echo.
- Login is required for the single-admin application.
- Credentials are managed through environment variables, preferably with `ADMIN_PASSWORD_HASH`.

No technical decisions are currently outstanding for the documented MVP architecture.

## 19. Technical Gap Analysis

The architecture is suitable for the MVP. The following implementation requirements must be completed before the first vertical slice is considered production-ready because mistakes here can cause security problems, data migration work, or inconsistent dashboard results later.

### Critical gaps

- **Authentication implementation:** Implement and test the signed session cookie, `SESSION_SECRET` rotation, eight-hour idle timeout, failed-login behavior, CSRF token, and `cmd/password-hash` utility.
- **Date storage contract:** Implement and test UTC `datetime2` persistence with `Asia/Manila` conversion, especially midnight and date-boundary cases.
- **Money contract:** Implement scaled `int64` values with four decimal places and round inputs with more than four decimals to the nearest 1/10,000 PHP.
- **Migration runner design:** Implement and test the version registry, SQL Server lock, transaction rollback, failed-state reporting, and immutable migration rules. GORM `AutoMigrate` alone does not provide this operational contract.
- **Audit schema:** Define and migrate the event table columns, serialization format for previous/new values, retention behavior, and links to the initiating request.
- **Dashboard query contract:** Implement and test client aggregation, PO drill-down, cursor pagination, remaining-count calculation, filters, and mixed-status clients.

### Important gaps

- **Company field validation:** Default lengths and required-value validation are defined, but actual production data should be reviewed before the first migration fixes the column sizes.
- **PO validation:** ASCII alphanumeric validation and duplicate-warning behavior are defined; maximum length and case-normalization behavior should be covered by tests.
- **Receivable archive behavior:** Archive and restoration are status-only updates; a restored receivable returns to `Active` and receives a derived dashboard classification. Company-account archiving is separately out of scope.
- **Account archive semantics:** Out of scope for the current MVP because the system will initially support only 1-4 active client accounts.
- **Cancellation behavior:** The plan allows cancellation and reopening; the UI and audit scenarios must confirm how a cancelled paid receivable is handled.
- **Error and concurrency behavior:** A baseline now exists for stale edits, duplicate form submissions, SQL deadlocks, and optimistic concurrency, but these require implementation tests.
- **Static assets:** Latest stable HTMX and Alpine.js versions will be selected, pinned locally, and recorded with source/license information.
- **Operational startup:** Graceful shutdown and SQL Server-unavailable behavior are explicitly out of scope for the first release; request and database timeouts remain recommended safeguards.
- **Test database lifecycle:** Integration tests are excluded from initial CI; the documented local test database and migration verification command must be used before database changes are accepted.
- **Production boundary:** The first release is local-only, but CSRF, session security, and secret handling should remain enabled so later network exposure does not require an architectural rewrite.

### Recommended pre-implementation decisions

1. Implement the session mechanism and cookie/security settings.
2. Define SQL Server column types, lengths, constraints, and indexes in the first migration.
3. Define and migrate the audit event schema and transaction rules.
4. Define dashboard query DTOs and representative aggregation scenarios.
5. Define local SQL Server test setup and a manual integration-test command.
6. Pin local HTMX and Alpine.js versions.

### Default validation and data-integrity baseline

- Trim all text input before validation and persistence.
- Require all company fields defined in the business plan; reject blank values.
- Use bounded lengths in both application validation and SQL Server columns. Initial defaults may be 255 characters for names and addresses, 50 for TIN/contact number, and 100 for PO number unless actual data requires more.
- Accept PO numbers using ASCII alphanumeric characters only, preserve the entered casing for display, and use a normalized value for duplicate warnings.
- Use a SQL Server `rowversion` column for optimistic concurrency on editable financial records.
- Use Post/Redirect/Get and HTMX request handling that prevents accidental double submission; add a request token when a command cannot safely be repeated.
- Retry transient SQL deadlocks a small, bounded number of times with jitter, only around safe transactions, and log the final failure.
- Add indexes based on actual dashboard predicates and verify them with SQL Server execution plans.

## 20. Development Blockers

These items can stop the system from being developed or run successfully. They should be resolved before the related vertical slice begins.

### Blockers before the first runnable vertical slice

1. **Go toolchain:** Install Go 1.25.7 for Windows `amd64` and declare the same version in `go.mod`.
2. **SQL Server access:** Create separate development and test databases, confirm SQL authentication, and provide host, port, database names, username, and passwords through ignored environment files.
3. **Configuration file:** Create `.env` from the example; development and the current shared test database use the same `CTS_DEV` values.
4. **Migration baseline:** Implement the separate migration executable and the first versioned schema before repository or integration work. The first schema must settle `bigint` scaled money, UTC `datetime2`, lifecycle status, nullable payment date, `rowversion`, and audit events.
5. **Date and money helpers:** Implement and test the UTC/Philippines date conversion and scaled-money rounding functions before creating receivable commands.
6. **Local frontend assets:** Pin and add HTMX and Alpine.js files locally before building HTMX pages.

Current status:

- Go 1.25.7: installed.
- `.env`: created and ignored by Git.
- SQL Server manual verification: planned by the owner.
- SQL authentication credentials: configured in ignored `.env`.
- First migration: planned as part of the development timeline.
- Date and money helpers: implemented and passing unit tests.
- HTMX 2.0.10 and Alpine.js 3.15.12: downloaded to `internal/web/static`.
- Authentication: intentionally deferred to the middle of development; must be completed before the application is shared beyond the owner's local development workflow.

### Blockers before production-ready MVP

1. **Audit schema and transactions:** Define which commands write audit events and ensure the business change and audit event commit or roll back together.
2. **Dashboard query contract:** Implement aggregation, filters, cursor pagination, PO drill-down, and exact remaining counts.
3. **Concurrency behavior:** Add `rowversion` checks, duplicate-submit protection, and bounded deadlock retry tests.
4. **Local SQL Server verification:** Run migrations and SQL Server integration tests against the test database manually, even though they are excluded from initial CI.

### Important but not development blockers

- Graceful shutdown and SQL Server-unavailable behavior are out of scope for the first release.
- Production backups and recovery are out of scope for the MVP.
- Cancelled-paid edge cases can be completed with the cancellation vertical slice; receivable archive restoration behavior is already defined as a status-only update.
- Exact company field length adjustments can be made before the first production migration, using the documented defaults during development.
- Authentication can be developed after the initial domain and dashboard slices, but must be completed before wider use.

## 21. Architecture Approval Gate

Begin implementation only after:

- The database and domain mapping is approved.
- The migration executable behavior is approved.
- Test and production configuration boundaries are approved.
- The authentication and deployment assumptions are explicit.
- The payment persistence choice is approved.
- The technical decisions in Section 18 are recorded and approved.
- The critical gaps in Section 19 are answered or intentionally deferred.
- The blockers in Section 20 are resolved for the relevant development stage.
- The first vertical slice is selected, preferably create receivable -> calculate due date -> display dashboard classification.
