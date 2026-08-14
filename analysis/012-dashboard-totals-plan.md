# Dashboard Totals Implementation Plan

## 1. Goal

Implement the first dashboard read slice that shows the owner accurate collection totals without opening individual receivables.

The dashboard must follow `analysis/004-vertical-slice-coding-standard.md` and reuse the existing receivable classification rules.

## 2. Dashboard Metrics

Display four summary cards:

| Card | Receivable count | Client count | Amount |
|---|---:|---:|---:|
| Pending | Active pending receivables | Clients with pending receivables | Sum of outstanding receivable amounts |
| Near Due | Active near-due receivables | Clients with near-due receivables | Sum of outstanding receivable amounts |
| Overdue | Active overdue receivables | Clients with overdue receivables | Sum of outstanding receivable amounts |
| Payment Received | Fully paid receivables | Clients with paid receivables | Sum of fully paid receivable amounts |

Every card must show both the number of receivables and the number of clients, plus the PHP amount.

## 3. Aggregation Rules

- Use the Philippines business date from the application clock.
- Include only delivery receivables with lifecycle status `Active`.
- Exclude `Cancelled` and `Archived` records from ordinary totals.
- A payment-received record has a non-null full payment date.
- An overdue record has no payment date and a due date before today.
- A near-due record has no payment date and a due date from today through five calendar days ahead.
- A pending record has no payment date and a due date after the near-due window.
- Sum scaled PHP integers before formatting for display.
- Do not store a derived classification in SQL Server.
- A client may contribute to more than one card when it has receivables with different classifications.
- Payment-received amount equals the full receivable amount because partial payments are not supported.

## 4. Slice Structure

Create or complete:

```text
internal/slices/dashboard/
├── queries.go
├── repository.go
├── repository_gorm.go
├── handler.go
├── view_models.go
├── templates/
│   ├── dashboard.html
│   ├── summary-cards.html
│   └── partials/
│       └── summary-cards.html
└── *_test.go
```

The dashboard slice owns read models and aggregation queries. It must not call the receivables slice's GORM repository directly.

## 5. Ordered Subtasks

### A. Define the dashboard query model

Create:

```go
type DashboardTotals struct {
    Pending        ClassificationTotal
    NearDue        ClassificationTotal
    Overdue        ClassificationTotal
    PaymentReceived ClassificationTotal
}

type ClassificationTotal struct {
    ReceivableCount int64
    ClientCount     int64
    AmountScaled    int64
}
```

Rules:

- Keep amounts in scaled integer units until the view-model boundary.
- Format PHP amounts in the view model, not SQL or templates.
- Return only values required by the dashboard template.
- Do not expose GORM models.

### B. Define the clock boundary

1. Add a dashboard query input containing the current UTC time or Philippines business date.
2. Convert it through the shared date helper.
3. Use the same date boundary for every card in one request.
4. Do not call `time.Now()` separately for each card.

Suggested query boundary:

```go
type GetDashboardTotalsQuery struct {
    Now time.Time
}

type QueryRepository interface {
    GetTotals(ctx context.Context, query GetDashboardTotalsQuery) (DashboardTotals, error)
}
```

### C. Define the SQL aggregation query

The query should:

- Join `delivery_receivables` to `company_accounts`.
- Filter `lifecycle_status = 'Active'`.
- Classify records using due date and payment date.
- Aggregate `COUNT_BIG(*)`, distinct company count, and scaled amount.
- Return one row per classification or map rows into the dashboard DTO.

The implementation must use parameterized values for the current business date and near-due boundary. Keep SQL Server-specific query syntax inside `repository_gorm.go`.

Do not duplicate the classification rules in unrelated handlers. Centralize the SQL classification expression or query specification and cover it with boundary tests.

### D. Implement the repository adapter

1. Define a persistence projection for aggregate rows.
2. Execute the aggregation with GORM `Raw` or a carefully scoped query builder.
3. Map classification rows into `DashboardTotals`.
4. Initialize missing classifications with zero totals.
5. Map SQL errors to application errors.
6. Select only aggregate columns.
7. Verify the query uses the intended due-date and lifecycle indexes.

### E. Implement the dashboard service

1. Receive the query and clock value.
2. Normalize the current date once.
3. Call the dashboard repository.
4. Format each scaled amount to two decimal places.
5. Build a page view model.
6. Return a stable error for database failures.

### F. Implement dashboard routes

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/dashboard` | Render full dashboard page |
| `GET` | `/dashboard/summary` | Render summary-card HTMX fragment |

Handler rules:

- Full page requests render the shared layout and summary cards.
- HTMX requests to `/dashboard/summary` return only the summary-card fragment.
- The handler does not calculate totals.
- Database errors are logged server-side and hidden behind a safe response.
- Dashboard output is read-only and creates no audit events.

### G. Implement templates

Use four semantic summary cards:

```gotemplate
<section class="summary-grid" aria-label="Collection totals">
  {{ template "summary-card" .Totals.Overdue "Overdue" "status-overdue" }}
  {{ template "summary-card" .Totals.NearDue "Near due" "status-near-due" }}
  {{ template "summary-card" .Totals.Pending "Pending" "status-pending" }}
  {{ template "summary-card" .Totals.PaymentReceived "Payment received" "status-payment-received" }}
</section>
```

Each card should show:

- Status label.
- Formatted PHP amount.
- Receivable count.
- Client count.
- Link to the corresponding filtered list when available.

Use text and icons only as reinforcement. Color must not be the only status signal.

### H. Add HTMX refresh behavior

The initial dashboard loads normally. Add an explicit refresh action:

```html
<section
  id="dashboard-summary"
  hx-get="/dashboard/summary"
  hx-trigger="refresh-dashboard from:body"
  hx-target="this"
  hx-swap="outerHTML">
  {{ template "summary-cards" . }}
</section>
```

Rules:

- Do not poll in the first version.
- Use HTMX to refresh the summary after a successful payment or receivable command.
- Alpine.js may show a local loading indicator but must not calculate totals.
- Keep the page usable when JavaScript is disabled.

## 6. Testing Subtasks

### Domain/query tests

- Pending classification is included in pending totals.
- Due today is included in near-due totals.
- Due five days ahead is included in near-due totals.
- Due six days ahead is pending.
- Due yesterday is overdue.
- Payment date moves the amount to payment received.
- Cancelled records are excluded.
- Archived records are excluded.
- A client with multiple classifications contributes to each relevant card once at the client count level per card.
- Amounts are summed in scaled units before display formatting.

### Repository tests with `sqlmock`

- Query binds current date and near-due boundary.
- Aggregate rows map to the correct cards.
- Missing classification rows become zero totals.
- Database errors are mapped safely.
- Query does not load individual receivable rows.

### HTTP and template tests

- `/dashboard` renders all four cards.
- `/dashboard/summary` returns only the summary fragment.
- Cards display PHP amounts, receivable counts, and client counts.
- Status classes and text are correct.
- Database failure returns a safe response.
- Dashboard navigation is active.

## 7. Acceptance Scenarios

1. Given active receivables in each classification, when the dashboard opens, then all four cards show correct counts and amounts.
2. Given cancelled and archived receivables, when the dashboard opens, then they are absent from ordinary totals.
3. Given one client with overdue and near-due receivables, then that client is counted once in each applicable client card.
4. Given a payment-received receivable, then its full amount appears only in the payment-received card.
5. Given no receivables, then all cards display zero values and the page remains usable.
6. Given a successful state-changing command, when the page requests a summary refresh, then the updated totals render through HTMX.
7. Given JavaScript is disabled, then the dashboard still renders accurate server-side totals.

## 8. Definition Of Done

- Dashboard totals are derived server-side from active receivable data.
- No derived classification column is added to the database.
- Each card shows PHP amount, receivable count, and client count.
- Classification boundary rules use the shared Philippines date handling.
- Aggregates sum scaled integer money before formatting.
- Cancelled and archived records are excluded.
- Full-page and HTMX summary responses work.
- Alpine.js contains no business calculations.
- Repository, service, HTTP, and template tests pass.
- `go test ./...`, `go vet ./...`, and formatting pass.
- The progress tracker is updated.

## 9. Next Slice

After totals are complete, implement client-level dashboard rows, filters, PO drill-down, and cursor load-more pagination.
