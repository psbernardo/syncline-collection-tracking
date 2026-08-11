# Tailwind UI and HTMX/Alpine Integration Plan

## 1. Recommended Template

Use **Windmill Dashboard HTML** as the starting visual template:

- Repository: https://github.com/estevanmaito/windmill-dashboard
- Live HTML dashboard: https://windmillui.com/dashboard-html
- License: MIT
- Styling: Tailwind CSS
- Browser behavior included: Alpine.js
- Not included: HTMX, which will be integrated into the server-rendered Go application

Windmill is a better fit than a React/Vue admin template because the project uses Go `html/template`, HTMX, and Alpine.js. Its accessible dashboard shell, responsive layout, cards, tables, forms, modal patterns, and light/dark theme support are relevant to this application.

### Alternatives reviewed

- Tailwind Plus: high-quality HTML application components, but paid and its interactive examples are not Alpine/HTMX-specific.
- HyperUI: free MIT Tailwind components, useful for individual UI pieces but not as complete an admin shell.
- TailAdmin: not selected because the referenced repository/template availability and framework variants should be verified before adopting it.

## 2. Scope Of Template Reuse

Reuse only the visual and structural parts needed by the collection tracker:

- Application shell
- Sidebar or compact mobile navigation
- Page headings
- Summary cards
- Status badges
- Responsive tables
- Form layouts
- Validation states
- Modal confirmation pattern
- Empty states
- Alerts and flash messages
- Light/dark theme tokens if retained

Do not copy template business logic, demo data, chart code, React code, or unnecessary dependencies.

## 3. Proposed Visual Language

The application is an operational finance tool, so the UI should prioritize scan speed and clear urgency:

- Overdue: red status, strong contrast, days overdue visible.
- Near due: amber status, days remaining visible.
- Pending: blue or slate status.
- Payment received: green status.
- Cancelled/archived: muted slate status.
- PHP values: right-aligned and formatted consistently to two decimals.
- PO numbers: visible in detail rows and drill-down views.
- Dense desktop tables with a stacked mobile card representation where tables become unreadable.
- The first dashboard panel should be overdue work, not charts.

Avoid decorative charts until the core data tables and daily workflow are reliable.

## 4. Asset and Build Plan

Add Tailwind CSS as a local build step. The final application should not require internet access at runtime.

Recommended structure:

```text
internal/web/
├── templates/
│   ├── layout.html
│   ├── dashboard/
│   ├── accounts/
│   ├── receivables/
│   └── partials/
├── styles/
│   └── input.css
└── static/
    ├── app.css
    ├── htmx.min.js
    ├── alpine.min.js
    └── THIRD-PARTY-NOTICES.md
```

Build rules:

- Copy the selected Windmill HTML patterns into Go templates instead of serving the demo application.
- Configure Tailwind content/source scanning for `internal/web/templates/**/*.html` and any Go-generated class sources.
- Compile only used utilities into `internal/web/static/app.css`.
- Pin the Tailwind CLI version in the project tooling.
- Keep HTMX and Alpine.js locally vendored as already established.
- Record Windmill, Tailwind, HTMX, and Alpine.js licenses and source versions.
- Do not use a CDN at runtime.

The build should fail if the generated CSS is missing or empty. The server should serve the committed generated CSS during local development.

## 5. Go Template Integration

Use a shared layout with named blocks:

```gotemplate
{{ define "layout" }}
<!doctype html>
<html lang="en" class="h-full">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <link rel="stylesheet" href="/static/app.css">
    <script defer src="/static/alpine.min.js"></script>
    <script src="/static/htmx.min.js"></script>
    <title>{{ .Title }} | Collection Tracking</title>
  </head>
  <body class="min-h-full bg-gray-50 text-gray-900 dark:bg-gray-900 dark:text-gray-100">
    {{ template "navigation" . }}
    <main class="px-4 py-6 sm:px-6 lg:px-8">
      {{ template "flash" . }}
      {{ block "content" . }}{{ end }}
    </main>
  </body>
</html>
{{ end }}
```

Rules:

- Pass view models, not GORM models, to templates.
- Use `html/template` escaping by default.
- Keep formatting functions explicit, such as PHP amount formatting and PH date formatting.
- Render full pages for normal requests and partial fragments for HTMX requests.
- Give every fragment a stable root element so it can be replaced predictably.

## 6. HTMX Integration

HTMX owns server communication and partial page replacement.

### Dashboard filters

```html
<form
  id="dashboard-filters"
  hx-get="/dashboard/clients"
  hx-target="#dashboard-results"
  hx-trigger="change, keyup changed delay:300ms from:input[name='query']"
  hx-push-url="true">
  <input
    name="query"
    type="search"
    placeholder="Search company or PO"
    class="rounded-lg border-gray-300">

  <select name="classification" class="rounded-lg border-gray-300">
    <option value="">All active</option>
    <option value="overdue">Overdue</option>
    <option value="near_due">Near due</option>
    <option value="pending">Pending</option>
    <option value="payment_received">Payment received</option>
  </select>
</form>

<div id="dashboard-results">
  {{ template "dashboard-client-results" . }}
</div>
```

### Load more

```html
{{ if .NextCursor }}
<button
  hx-get="/dashboard/clients?cursor={{ .NextCursor }}"
  hx-target="#load-more"
  hx-swap="outerHTML"
  class="rounded-lg bg-primary-600 px-4 py-2 text-white">
  Load more
  <span class="text-sm">({{ .RemainingCount }} remaining)</span>
</button>
{{ end }}
```

### Form commands

- Use normal `POST` forms as the fallback.
- Add `hx-post` for partial validation and submission.
- Return the form fragment with validation errors.
- Return `HX-Redirect` after successful commands.
- Disable the submit control while a request is active.
- Keep idempotency keys in the form as hidden inputs.
- Never calculate due dates, totals, or classifications in the browser.

## 7. Alpine.js Integration

Alpine.js owns local presentation state only:

- Mobile sidebar open/closed state
- Filter drawer open/closed state
- Confirmation modal visibility
- Light/dark theme preference if retained
- Local disclosure panels
- Focus and escape-key behavior

Example mobile navigation:

```html
<div x-data="{ open: false }" class="lg:hidden">
  <button
    type="button"
    @click="open = !open"
    :aria-expanded="open.toString()"
    aria-controls="mobile-navigation"
    class="rounded-lg p-2">
    <span class="sr-only">Open navigation</span>
    Menu
  </button>

  <nav
    id="mobile-navigation"
    x-cloak
    x-show="open"
    @keydown.escape.window="open = false"
    class="fixed inset-y-0 left-0 z-40 w-72 bg-white shadow-xl dark:bg-gray-800">
    {{ template "navigation-links" . }}
  </nav>
</div>
```

Alpine must not own server state. After an HTMX response replaces a component, the returned HTML is authoritative.

## 8. Page Plan

### Dashboard

- Summary cards at the top.
- Overdue clients first.
- Near-due clients second.
- Pending and payment-received sections below.
- Search and filter bar above results.
- Client row opens or loads PO drill-down.
- Amount and count visible together.

### Company accounts

- List with company name, TIN, contact person, and contact number.
- Create/edit form using Windmill form patterns.
- Required-field validation inline.
- No company-account archive control in the MVP.

### Receivable form

- Company select.
- PO number input.
- Amount input with PHP prefix.
- Delivery date input.
- Term input with range guidance `1-120 days`.
- Server-rendered due-date result after save.
- Payment date omitted from initial create form or shown as optional.

### Receivable detail

- Client information.
- PO and delivery information.
- Amount and due date.
- Current lifecycle status and derived classification.
- Payment action.
- Cancel, reopen, archive, and restore actions according to lifecycle state.
- Audit timeline.

## 9. Accessibility and Responsive Rules

- Use semantic headings and landmarks.
- Every input has a visible label.
- Every status badge includes text, not color alone.
- Tables must have headers and responsive fallback behavior.
- Modal dialogs must support Escape and focus management.
- HTMX loading states must be visible to keyboard and screen-reader users.
- Use `aria-live` for validation and dashboard update messages.
- Test at narrow mobile widths and desktop widths.

## 10. Implementation Sequence

1. Copy and simplify the Windmill HTML shell.
2. Add Tailwind build configuration and generate `app.css`.
3. Add Go layout and partial templates.
4. Implement dashboard read-only page with fixture view models.
5. Wire dashboard filters and load-more HTMX fragments.
6. Implement company account create/edit.
7. Implement receivable create and due-date display.
8. Add payment, cancellation, archive, restore, and audit actions.
9. Replace fixture data with slice queries and repositories.
10. Add responsive, accessibility, and browser workflow tests.

## 11. Acceptance Criteria

- The application renders a complete dashboard without external network access.
- Tailwind CSS, HTMX, and Alpine.js assets are locally served.
- Dashboard filters update server-rendered results without a full page reload.
- Load-more displays the remaining count and appends the next result page.
- A company and receivable can be created through server-rendered forms.
- Validation works with JavaScript disabled.
- Alpine interactions do not duplicate server-side business rules.
- The dashboard displays accurate client totals and PO-level details.
- The visual shell works on mobile and desktop.

## 12. References

- Windmill Dashboard: https://github.com/estevanmaito/windmill-dashboard
- Windmill HTML dashboard: https://windmillui.com/dashboard-html
- Tailwind CSS: https://tailwindcss.com/docs
- HTMX documentation: https://htmx.org/docs/
- Alpine.js documentation: https://alpinejs.dev/start-here
