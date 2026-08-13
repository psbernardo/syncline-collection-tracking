# Reusable Feedback Snackbar Plan

## 1. Goal

Create a reusable status/snackbar component for user feedback across the application.

The first required use case is communicating duplicate PO errors when creating or editing a receivable.

Behavior:

- Appears on the left side of the screen.
- Remains visible for three seconds.
- Slowly fades out before removal.
- Works with normal full-page responses and HTMX fragments.
- Uses Alpine.js only for local visibility and timing state.
- Uses server-rendered messages as the source of truth.

## 2. Message Types

| Type | Use | Visual tone |
|---|---|---|
| `success` | Create, update, payment, or archive succeeded | Green |
| `error` | Duplicate PO, validation, or server failure | Red |
| `warning` | Attention needed but operation succeeded | Amber |
| `info` | General status or guidance | Blue |

Duplicate PO message:

```text
PO number is already used by a non-cancelled receivable.
```

The duplicate message should also appear beside the PO input so the user knows exactly what to correct. The snackbar provides global confirmation; the inline error provides field-level guidance.

## 3. Shared Template Location

Add:

```text
internal/web/templates/partials/
└── snackbar.html
```

Render the snackbar once from the shared layout, not separately in every slice.

The shared layout view model should include:

```go
type FlashMessage struct {
    ID      string
    Type    string
    Title   string
    Message string
}
```

Only trusted server-selected `Type` values may become CSS classes. `ID` must be unique per response so repeated HTMX errors are distinct DOM elements.

## 4. Alpine.js Behavior

Use Alpine.js for local state only:

```html
<div
  x-data="{ visible: true }"
  data-snackbar-id="{{ .ID }}"
  x-show="visible"
  x-init="setTimeout(() => visible = false, 3000)"
  x-transition:enter="transition ease-out duration-200"
  x-transition:enter-start="opacity-0 -translate-x-2"
  x-transition:enter-end="opacity-100 translate-x-0"
  x-transition:leave="transition ease-in duration-700"
  x-transition:leave-start="opacity-100"
  x-transition:leave-end="opacity-0"
  @transitionend.once="if (!visible) $el.remove()"
  role="{{ if eq .Type "error" }}alert{{ else }}status{{ end }}"
  aria-live="{{ if eq .Type "error" }}assertive{{ else }}polite{{ end }}">
  <strong>{{ .Title }}</strong>
  <p>{{ .Message }}</p>
  <button type="button" @click="visible = false" aria-label="Dismiss message">Close</button>
</div>
```

Rules:

- Start the three-second timer after the component is rendered.
- Use a slow leave transition of approximately 700 milliseconds.
- Allow manual dismissal immediately.
- Do not calculate business state in Alpine.js.
- Respect `prefers-reduced-motion` by disabling or shortening transitions.
- Ensure the message is announced by assistive technology.
- Remove the element after the leave transition so repeated messages do not accumulate hidden nodes.
- Use `assertive` only for errors; use `polite` for success, warning, and info.

## 5. Layout Placement and Styling

Place the snackbar container on the left side:

```html
<div id="snackbar-region" class="snackbar-container">
  {{ template "snackbar" .Flash }}
</div>
```

Recommended CSS behavior:

- `position: fixed`
- `left: 1rem` on desktop
- `bottom: 1rem`
- High z-index above page content and modals where appropriate
- Maximum width to prevent long messages from covering the page
- Full-width with page padding on mobile
- Visible border or left accent matching the message type
- Sufficient contrast for text and action controls

The snackbar must not block the primary form controls or create horizontal scrolling. The region itself should not have a live role when each message has its own announcement role.

## 6. HTMX Integration

### Duplicate PO error

When the server detects `ErrDuplicatePO`:

1. Return HTTP `422`.
2. Return the form fragment with the PO inline error.
3. Include an out-of-band snackbar fragment using `hx-swap-oob="true"`.

Because HTMX does not necessarily swap every non-2xx response by default, configure response handling once in the local HTMX bootstrap:

```js
htmx.config.responseHandling = [
  { code: "204", swap: false },
  { code: "[23]xx", swap: true },
  { code: "422", swap: true, error: true },
  { code: "[45]xx", swap: false, error: true }
]
```

This preserves `422` semantics while allowing validation fragments and OOB snackbars to render. Test it with a real HTMX request, not only an HTML snapshot.

Example response fragment:

```html
<form id="receivable-form">
  <!-- fields with the PO error -->
</form>

<div id="snackbar-region" hx-swap-oob="beforeend">
  {{ template "snackbar" .DuplicatePOMessage }}
</div>
```

The existing form target is replaced while the snackbar is appended to the shared layout region.

### Successful commands

- Continue using `HX-Redirect` after successful create/update commands.
- Do not add success snackbars until a redirect-safe flash transport exists. A response body is not available after `HX-Redirect`; use a signed flash cookie or server-side flash store later.
- Do not create duplicate snackbars from both Alpine and HTMX event handlers.

## 7. Server Message Flow

1. Domain/application code returns a typed error such as `ErrDuplicatePO`.
2. The handler maps it to a field error and a `FlashMessage`.
3. The view model passes both to the template.
4. The template renders inline validation and the snackbar OOB fragment.
5. Alpine controls only the snackbar timer, transition, and manual dismissal.

Do not expose raw database constraint messages to the user.

## 8. Subtasks

1. Define `FlashMessage` and allowed message types.
2. Add the shared snackbar layout region.
3. Add the snackbar partial with Alpine timer and transitions.
4. Add responsive left-side styling.
5. Add reduced-motion styling.
6. Extend receivable error view models with flash feedback.
7. Map duplicate PO errors to inline and snackbar messages.
8. Add HTMX `hx-swap-oob` response support.
9. Add the local HTMX response-handling bootstrap for `422` fragments.
10. Remove snackbar elements after their fade transition.
11. Add success/info/warning messages only after redirect-safe flash transport exists.
12. Add template, HTTP, HTMX, and browser behavior tests.

## 9. Tests

- Duplicate PO returns `422`.
- Duplicate PO displays inline PO error.
- Duplicate PO appends an error snackbar through `hx-swap-oob`.
- Snackbar has `role="alert"` and `aria-live`.
- Snackbar includes a manual dismiss button.
- Snackbar timer is three seconds.
- Leave transition is gradual rather than instant.
- Hidden snackbars are removed after the transition.
- `422` HTMX responses swap the form and OOB snackbar.
- Error messages use assertive announcements; non-errors use polite announcements.
- Success response does not render duplicate messages.
- Full-page responses render flash messages correctly.
- Snackbar is usable on mobile and desktop.
- Reduced-motion users do not receive forced animation.

## 10. Definition Of Done

- One shared snackbar component is used by all slices.
- Duplicate PO errors are shown inline and globally.
- HTMX form replacement and OOB snackbar insertion work together.
- HTMX response handling explicitly supports `422` validation fragments.
- Message types cannot inject arbitrary CSS classes.
- Alpine.js owns only local display state.
- The snackbar appears on the left and disappears after three seconds with a slow fade.
- Dismissed snackbars are removed from the DOM.
- Accessibility attributes and keyboard dismissal work.
- Tests pass with `go test ./...`, `go vet ./...`, and formatting.
- `analysis/progress.md` is updated.
