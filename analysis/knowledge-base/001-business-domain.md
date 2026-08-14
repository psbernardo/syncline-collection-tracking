# Collection Tracking System

## 1. Purpose

This document defines the domain, operational process, business flow, and scope for a simple collection tracking system before application development begins.

No application implementation should start until the remaining decisions have been answered and the resulting domain model and workflows have been approved.

## 2. Planning Approach

Work through the following steps in order:

1. Define the business problem and measurable outcome.
2. Identify the people, organizations, and systems involved.
3. Establish a shared vocabulary for the domain.
4. Define the collection lifecycle and allowed state changes.
5. Describe the real-world collection process from start to finish.
6. Convert the process into business rules and exceptions.
7. Define the minimum information required at each step.
8. Define permissions, accountability, and audit requirements.
9. Select the smallest useful MVP scope.
10. Write acceptance scenarios for the agreed business flows.
11. Review and approve this document.
12. Only then design the technical solution and build the application.

## 3. Business Problem and Outcome

### Problem statement

The owner of a small company needs to track money owed by company accounts after a delivery. Each client can have a different payment term, and the owner needs to see what is overdue, what is approaching its due date, what remains pending, and what has been paid.

### Desired outcome

The system should provide one reliable view of each delivery receivable, its client, amount, due date, remaining balance, payment status, and history.

### Success measures

- The owner can identify every overdue account from the dashboard.
- The owner can identify accounts approaching their due date before they become overdue.
- Pending, overdue, and payment-received totals are accurate.
- Payment dates are traceable to the relevant delivery receivable.

## 4. Scope Definition

### Validated core workflow

The actual MVP workflow is simpler than a general collections system:

1. The owner selects or creates a client company.
2. The owner records the amount for items delivered to that client.
3. The owner records the delivery date and payment term for that receivable.
4. The system calculates the due date.
5. Each day, the dashboard calculates and displays near-due, overdue, pending, and paid receivables.
6. When money is received, the owner records the full payment and payment date.
7. The system recalculates the remaining balance and collection status.

The central domain object should therefore be a **delivery receivable**, not a generic collection task. A client can have many delivery receivables, and each receivable is settled by recording its full payment date.

### Challenge to the previous plan

The previous proposal included assignments, collection agents, contact attempts, promises to pay, escalation queues, and several workflow states. Those concepts do not match the current single-admin daily workflow and should not be part of the first MVP. They can be future extensions if the business process later requires them.

### Confirmed direction

- The first release is an internal tool for one administrator: the company owner.
- The system tracks money owed by company accounts.
- Initial data will be provided by the owner.
- Full payments are recorded in the system with a payment date; payment proof and reconciliation are out of scope.
- The system will calculate due dates from the delivery date and the term entered for that receivable.
- The initial UI should use Go, go-htmx, and Alpine.js.
- Important changes and payment entries should be traceable through history.

### In scope for the MVP

- Create and edit company accounts and money owed by each account.
- Record PO number, delivery date, payment term, calculated due date, amount due, and payment date.
- Record full payments only; partial payments are not supported.
- Search, filter, and sort accounts by status and due-date window.
- Show overdue accounts and accounts approaching their due date.
- Provide dashboard totals for overdue, pending, approaching due date, and payment received.
- View an account's payment and change history.
- Preserve the payment term and calculated due date entered for each delivery receivable.

### Explicitly out of scope until requested

- Automated payment processing or bank integration.
- Automatic legal or credit decisions.
- Complex accounting, reconciliation, or invoicing.
- Customer self-service access.
- External communications such as SMS, email, or voice automation.
- Predictive scoring or machine learning.
- Multi-organization tenancy.

## 5. Actors and Responsibilities

| Actor | Responsibility | Access needed |
|---|---|---|
| Owner / administrator | Creates accounts, records deliveries and full payments, monitors the dashboard, and resolves exceptions | Full access |
| Customer or debtor | Person or organization from whom collection is due | No access assumed |

There is one internal user in the initial release. Customer/debtor access is not required.

## 6. Domain Vocabulary

The following terms need agreed definitions before implementation:

| Term | Working definition | Decision needed |
|---|---|---|
| Company account | A client company that owes money | Company name, contact person, TIN number, billing address, delivery address, and contact number |
| Delivery receivable | A delivery made to a company account that creates an amount due | Multiple receivables may share a client, date, and amount; required alphanumeric PO number distinguishes them |
| Payment term | Number of calendar days allowed for payment after delivery | Whole number from 1 to 120 |
| Due date | Delivery date plus the term captured for that receivable | Delivery date is day one |
| Amount due | Amount expected for a delivery receivable | Philippine peso (`PHP`) |
| Payment / receipt | Full money received against an amount due | Payment date is recorded; proof is out of scope |
| Balance | Amount still outstanding before full payment | Calculated by the system |
| Status | Current collection state, such as Pending, Overdue, Payment Received, or Cancelled | Full payment only |
| Near due | A receivable whose due date is within five calendar days | Dashboard classification only |

## 7. Collection Lifecycle

### Confirmed statuses and dashboard classifications

```text
Pending --within warning window--> Near Due (dashboard view)
Pending -> Overdue
Pending -> Payment Received
Overdue -> Payment Received
Pending -> Cancelled
Overdue -> Cancelled
Payment Received -> Cancelled
Cancelled -> Pending / Overdue / Payment Received (derived)
```

This is the agreed MVP status model. `Near Due` remains a dashboard classification rather than a stored status.

### State definitions

| State | Meaning | Entry condition | Exit condition |
|---|---|---|---|
| Pending | Amount is owed and its due date has not passed | Delivery receivable created | Payment received or due date passes |
| Overdue | Due date has passed and the balance remains unpaid | System date is after due date | Payment received or correction |
| Payment Received | Full payment has been recorded with a payment date | Payment date entered and payment equals amount due | Reopened only by explicit correction |
| Cancelled | Receivable is excluded from collection tracking | Owner manually sets status, with an optional note | Owner can reopen it explicitly |

`Pending`, `Overdue`, `Payment Received`, and `Cancelled` are statuses. `Near Due` is a calculated dashboard classification, not a permanent lifecycle status. `Pending`, `Overdue`, and `Near Due` are derived from the due date and current date. `Payment Received` is based on a full payment and payment date. `Cancelled` is manually set by the owner and excluded from ordinary overdue and pending totals.

## 8. Business Process

### 8.1 Account and delivery receivable intake

1. The owner creates or selects a company account.
2. The owner records the company, PO number, delivered amount, delivery date, and payment term.
3. The system validates the required fields.
4. The system calculates and stores the due date from the delivery date and captured payment term.
5. The receivable starts as `Pending` unless it is already fully paid.
6. The system validates that the PO number is alphanumeric and warns about possible duplicates.

### 8.2 Due-date monitoring

1. The system compares today's date in Philippines time with each calculated due date.
2. Pending records with a due date from today through five calendar days from today appear as near due.
3. Records past their due date with an unpaid balance appear as overdue.
4. The dashboard summarizes pending, near due, overdue, and payment-received records.

### 8.3 Daily owner workflow

1. Open the dashboard.
2. Review overdue receivables first, ordered by days overdue or amount outstanding.
3. Review near-due receivables, ordered by nearest due date.
4. Record payments received since the previous review.
5. Review each recorded payment date.
6. Review pending and outstanding totals.
7. Correct or investigate exceptions such as disputed, returned, or incorrectly entered receivables.

### 8.4 Payment or other collection outcome

1. The owner records the full payment and payment date.
2. The system updates the payment date and remaining balance.
3. When the full amount is recorded, the status becomes `Payment Received`.

### 8.5 Exception handling

1. The owner corrects an incorrect company, PO number, delivery date, term, amount, or payment.
2. The system preserves the previous value and correction timestamp.
3. The owner can reopen a payment-received or cancelled record explicitly, with the action recorded in history.

## 9. Core Business Rules

The following rules should be confirmed or changed:

- Each delivery receivable belongs to one company account.
- Multiple receivables may belong to the same company, including with the same delivery date and amount.
- The PO number distinguishes receivables that otherwise have the same company, date, and amount.
- PO number is mandatory and accepts alphanumeric free text.
- A PO number is used to trace a receivable; the system warns about possible duplicate company, PO, date, and amount combinations rather than silently merging records.
- The due date is calculated from the delivery date and the term entered for that receivable.
- Payment terms accept any whole number from 1 to 120 calendar days.
- Terms use calendar days, with the delivery date counted as day one.
- Due date formula: `delivery date + (term days - 1)` calendar days.
- All date calculations use the Philippines timezone (`Asia/Manila`).
- A payment is considered received when the full amount is recorded with a payment date.
- Partial payments are not supported in the MVP.
- `Overdue` is calculated when the due date has passed and no full payment date has been recorded.
- `Payment Received` is based on the payment date, not a separate check date.
- `Cancelled` receivables are excluded from overdue and pending totals.
- A cancelled receivable may be reopened explicitly by the owner.
- Reopening a cancelled receivable returns it to a derived active status based on its due date and payment date.
- Corrections to a payment, amount, delivery date, term, company, or PO number must be audited.
- A payment date cannot be earlier than the delivery date.
- A payment date cannot be later than today in Philippines time.
- Financial records are archived rather than deleted.
- Amounts use Philippine peso (`PHP`).
- The system must preserve an immutable history of material changes.

The following rules are not yet defined:

- Detailed handling of cancelled records beyond the manually assigned status.

Legal, regulatory, retention, privacy, and approval rules are out of scope for this MVP.

## 10. Minimum Data to Confirm

### Delivery receivable

- Identifier
- Company account reference
- PO number
- Delivery date
- Payment term in days
- Amount due
- Payment date
- Currency (`PHP`)
- Due date
- Status
- Created date and creator
- Cancellation note, if provided

### Company account

- Company account name
- Contact person
- TIN number
- Billing address
- Delivery address
- Contact number

### Activity

- Date and time
- Activity type, such as payment recorded, cancellation, or correction
- Actor
- Previous value and new value when data changes
- Notes

Initial data will be provided by the owner. The mandatory fields still need to be agreed.

## 11. Permissions and Audit

The initial release has one administrator with full access:

- Create and edit company accounts and delivery receivables.
- Record full payments and payment dates.
- Correct financial data with an audit reason.
- View data; exports are not required for the MVP.

Future multi-user permissions are out of scope for the MVP.

Every material action should record actor, timestamp, previous value, new value, and reason where relevant.

Exports are not required for the first release.

## 12. Operational Views

The initial product should be evaluated against these views:

- Dashboard totals: overdue, pending, near due, and payment received.
- Overdue clients aggregated by outstanding amount and receivable count, with drill-down to PO numbers.
- Near-due clients aggregated by outstanding amount and receivable count, with days remaining.
- Pending clients and outstanding balance.
- Payment-received clients and collected totals.
- Account detail with deliveries, payments, balance, and history.

The overdue view is the most important. The dashboard should also make upcoming due dates visible before they become overdue.

## 13. MVP Acceptance Scenarios

These scenarios become acceptance criteria after the decisions above are finalized:

1. Given a company account and delivery date, when an amount and payment term are entered, then the system calculates and displays the due date.
2. Given a pending record whose due date has passed, then it appears in the overdue dashboard view.
3. Given a pending record within the configured warning window, then it appears in the near-due view.
4. Given a receivable, when the owner records the full payment and payment date, then it becomes `Payment Received`.
5. Given an overdue receivable, when its full payment date is recorded, then it leaves the overdue view and appears in payment-received results.
6. Given two receivables for the same company, date, and amount, when they have different PO numbers, then both remain separately traceable.
7. Given a material edit or payment correction, then the prior value remains available in the audit history.
8. Given the dashboard, then overdue, pending, near-due, and payment-received totals are visible and accurate.
9. Given a payment, cancellation, reopening, archive, or correction, when the command succeeds, then exactly one immutable audit event records the change.
10. Given a failed business change or audit write, then neither the business change nor its audit event is committed.

## 14. Gap Analysis

The described workflow is sufficient to define the happy path, but these gaps could produce incorrect balances or misleading dashboard results:

### Financial decisions now confirmed

- Each delivery receivable is tracked separately, and multiple receivables can belong to one company.
- PO number distinguishes receivables with otherwise identical company, date, and amount data.
- Partial payments, overpayments, discounts, taxes, credit notes, refunds, and payment proof are out of scope.
- Payment status is based on the payment date.

### Date and term decisions now confirmed

- Terms use calendar days.
- The delivery date is day one.
- Dates use Philippines time (`Asia/Manila`).
- The near-due warning window is five days before the due date.
- Terms are entered per collection item/receivable, not at company level.

### Operational gaps

- The system needs an explicit correction process for wrong client, amount, delivery date, term, or payment data.
- Cancelled receivables are manually tagged with `Cancelled` and excluded from overdue totals.
- Initial records are entered by the owner using company name, amount, term, delivery date, and PO number.
- Company-level dashboard rows link to their underlying receivables; status, due date, and PO filters apply at receivable level.

### Reporting gaps

- Dashboard cards show both amount totals and receivable counts.
- The overdue view needs a sort order and filters, such as client, amount, and days overdue.
- Dashboard totals are aggregated by client, with drill-down to individual receivables and PO numbers.
- A client can appear in more than one dashboard classification when it has multiple receivables with different statuses.
- The system uses the Philippines date boundary for daily classification.

### Security and reliability gaps

- Backup, restore, and data-loss recovery are out of scope for the MVP.
- Audit history should prevent silent deletion of financial records.
- Currency is Philippine peso (`PHP`).

### Recommended MVP boundary

For the first version, implement only:

- Company accounts.
- Delivery receivables with amount, delivery date, term, and calculated due date.
- Full payments with payment dates and a calculated outstanding balance.
- Derived `Pending`, `Overdue`, `Payment Received`, and `Near Due` dashboard classifications.
- Daily dashboard and filtered lists.
- Corrections with audit history.

Defer contact management, reminders by email/SMS, promises to pay, automatic escalation, multiple users, accounting integration, and advanced reports until the basic numbers are trusted.

## 15. Remaining Decisions Before Design

No business decisions are currently outstanding for the MVP. Any future decisions should be recorded here before implementation changes the agreed scope.

## 16. Technical Direction

The planned application stack is:

- Backend: Go.
- Server-rendered interaction: go-htmx.
- Lightweight client-side behavior: Alpine.js.

Technical design, persistence, deployment, and testing should be selected after the remaining business decisions are approved.

## 17. Approval Gate

The plan is ready for technical design when:

- The business problem and success measures are agreed.
- Actors and responsibilities are confirmed.
- The glossary has no ambiguous core terms.
- The lifecycle states and transitions are approved.
- Required data and business rules are confirmed.
- Permissions and audit expectations are known.
- MVP scope and acceptance scenarios are approved.
- Open questions are either answered or explicitly deferred.

### Decision log

| Date | Decision | Reason | Owner |
|---|---|---|---|
| TBD | Planning document created | Establish domain and process before implementation | TBD |
