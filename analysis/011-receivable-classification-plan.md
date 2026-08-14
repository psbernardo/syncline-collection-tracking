# Receivable Classification Display Plan

## 1. Goal

Display the current derived classification for every active delivery receivable:

- `Pending`
- `Near Due`
- `Overdue`

The classification must be calculated from the due date, payment date, lifecycle status, and current Philippines business date. It must not be stored as a value that can become stale overnight.

This plan follows `analysis/004-vertical-slice-coding-standard.md`.

## 2. Classification Rules

Apply rules in this order:

1. If lifecycle status is `Cancelled`, display `Cancelled`.
2. If lifecycle status is `Archived`, display `Archived`.
3. If a full payment date exists, display `Payment Received`.
4. If the current Philippines date is after the due date, display `Overdue`.
5. If the due date is today through five calendar days from today, display `Near Due`.
6. Otherwise, display `Pending`.

The classification is derived in the domain/application layer using a supplied clock or current time. Do not call `time.Now()` directly inside the domain rule.

## 3. Scope

### In scope

- Display classification in the receivable list.
- Display classification in receivable detail.
- Display days overdue for overdue records.
- Display days remaining for near-due records.
- Display a readable status badge with text and color.
- Apply the classification consistently in full-page and HTMX responses.
- Test all date boundaries using `Asia/Manila`.

### Out of scope

- Dashboard client aggregation.
- Automatic reminders.
- Email or SMS notifications.
- Payment recording UI.
- Partial payment classification.
- Persisting a separate derived classification column.

## 4. Ordered Subtasks

### A. Confirm the domain classification function

1. Add or confirm a pure function on the receivable domain type.
2. Accept the current time as an argument.
3. Convert current time and due date through the shared Philippines date helper.
4. Apply the rule order from Section 2.
5. Return a typed classification value rather than an arbitrary string where practical.
6. Add helpers for days overdue and days remaining.

Suggested API:

```go
type Classification string

const (
    ClassificationPending        Classification = "Pending"
    ClassificationNearDue        Classification = "Near Due"
    ClassificationOverdue        Classification = "Overdue"
    ClassificationPaymentReceived Classification = "Payment Received"
    ClassificationCancelled      Classification = "Cancelled"
    ClassificationArchived       Classification = "Archived"
)

func (r DeliveryReceivable) ClassificationAt(now time.Time) Classification
```

### B. Add classification to query view models

Extend `ReceivableViewModel` with:

- `Classification`
- `ClassificationTone` or `ClassificationClass`
- `DaysOverdue`
- `DaysUntilDue`
- `DueDateDisplay`

The handler should receive fully prepared display values. Templates should not perform date arithmetic.

### C. Update repository projections

Receivable list and detail queries must select:

- Lifecycle status
- Payment date
- Due date
- Company account ID
- PO number
- Amount
- Delivery date
- Payment term

Do not query a stored derived classification. The query should provide the source data needed by the service to derive it.

### D. Update the receivable service

1. Inject a clock or `now` function into the service.
2. Load receivables through the repository.
3. Derive the classification for each record.
4. Calculate days overdue or days remaining.
5. Build the view model.
6. Return the view model to the handler.

For later dashboard queries, move the same rule into a shared query specification so list and dashboard classifications cannot diverge.

### E. Update list and detail templates

List row example:

```gotemplate
<td>
  <span class="status-badge status-{{ .ClassificationTone }}">
    {{ .Classification }}
  </span>
  {{ if eq .Classification "Overdue" }}
    <span class="status-detail">{{ .DaysOverdue }} days overdue</span>
  {{ end }}
  {{ if eq .Classification "Near Due" }}
    <span class="status-detail">{{ .DaysUntilDue }} days remaining</span>
  {{ end }}
</td>
```

Rules:

- Always display text, not color alone.
- Use accessible contrast.
- Keep status styling consistent across list, detail, and future dashboard pages.
- Use server-rendered values.
- Do not add Alpine calculations.

### F. Add HTMX behavior

For the current list/detail pages:

- Classification is rendered on the initial server response.
- HTMX form responses include the updated classification when a command changes the record.
- Do not poll the server in this slice.
- When dashboard refresh is implemented, use an explicit HTMX refresh request rather than browser-side date calculations.

## 5. Status Styling

Recommended semantic tones:

| Classification | Visual tone | Meaning |
|---|---|---|
| Pending | Blue/slate | Not yet near due |
| Near Due | Amber | Due within five calendar days |
| Overdue | Red | Due date has passed |
| Payment Received | Green | Full payment date exists |
| Cancelled | Gray | Excluded from active collection |
| Archived | Gray/muted | Historical record |

Example CSS classes:

```css
.status-badge { border-radius: 999px; padding: .25rem .6rem; font-size: .75rem; font-weight: 700; }
.status-pending { background: #dbeafe; color: #1e40af; }
.status-near-due { background: #fef3c7; color: #92400e; }
.status-overdue { background: #fee2e2; color: #991b1b; }
.status-payment-received { background: #dcfce7; color: #166534; }
.status-cancelled, .status-archived { background: #e2e8f0; color: #475569; }
```

The class should be generated by trusted server-side mapping, not directly from unvalidated user input.

## 6. TDD Subtasks

### Domain tests

- Active receivable due six days from today is `Pending`.
- Active receivable due five days from today is `Near Due`.
- Active receivable due today is `Near Due`.
- Active receivable due yesterday is `Overdue`.
- Paid receivable is `Payment Received` regardless of due date.
- Cancelled receivable is `Cancelled` regardless of due date.
- Archived receivable is `Archived` regardless of due date.
- Philippines midnight boundary is handled correctly.
- Days overdue is correct.
- Days remaining is correct.

### Service tests

- List view models contain classification and display details.
- Detail view model contains classification and display details.
- A fixed test clock produces deterministic output.

### HTTP/template tests

- Pending badge renders.
- Near-due badge and days remaining render.
- Overdue badge and days overdue render.
- Payment-received, cancelled, and archived text render.
- Status text remains available without JavaScript.
- HTMX fragment contains the same classification as the full-page response.

## 7. Acceptance Scenarios

1. Given an active receivable due more than five days from today, then the page displays `Pending`.
2. Given an active receivable due within five calendar days, then the page displays `Near Due` and days remaining.
3. Given an active receivable whose due date passed, then the page displays `Overdue` and days overdue.
4. Given a fully paid receivable, then the page displays `Payment Received`.
5. Given a cancelled receivable, then the page displays `Cancelled` and excludes it from active classification views.
6. Given an archived receivable, then the page displays `Archived` and excludes it from ordinary active views.
7. Given the same record viewed at different times, then its derived classification changes without a database update.

## 8. Definition Of Done

- Classification is derived server-side.
- No derived classification column is added to SQL Server.
- The shared Philippines timezone helper is used.
- The current time is injectable in tests.
- List and detail pages display consistent classification values.
- Days remaining and days overdue are correct at boundaries.
- Status badges are accessible and text-based.
- HTMX responses preserve classification behavior.
- Alpine.js contains no classification logic.
- Unit, service, HTTP, and template tests pass.
- `go test ./...`, `go vet ./...`, and formatting pass.
- `analysis/028-progress.md` is updated.

## 9. Next Slice

Reuse this classification service in the dashboard aggregation slice. Dashboard queries should aggregate after applying the same active-record classification rules.
