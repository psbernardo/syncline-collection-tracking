# Shared Navigation Bar Implementation Plan

## 1. Goal

Create one shared responsive navigation bar that lets the owner move between:

- Dashboard
- Company accounts
- Delivery receivables
- New company account
- New delivery receivable

The navigation must follow `analysis/vertical-slice-coding-standard.md` and work with Go `html/template`, HTMX, Alpine.js, and the local Tailwind asset plan.

## 2. Scope

### In scope

- Shared navigation partial used by every full page.
- Desktop navigation links.
- Mobile navigation menu.
- Active route styling.
- Accessible labels and keyboard behavior.
- Links to company and receivable pages.
- Local Alpine.js behavior for mobile menu state.

### Out of scope

- Authentication controls.
- User profile menu.
- Notifications.
- Breadcrumbs for every page.
- Client-side route handling.

## 3. Navigation Contract

| Label | Path | Active when |
|---|---|---|
| Dashboard | `/dashboard` | Path starts with `/dashboard` |
| Company accounts | `/accounts` | Path starts with `/accounts` |
| Receivables | `/receivables` | Path starts with `/receivables` |
| New account | `/accounts/new` | Exact path `/accounts/new` |
| New receivable | `/receivables/new` | Exact path `/receivables/new` |

The dashboard link may remain a placeholder until the dashboard slice is implemented, but it should be part of the shared navigation contract.

## 4. Template Structure

Create shared templates:

```text
internal/web/templates/
├── layout.html
└── partials/
    └── navigation.html
```

The accounts and receivables slices should stop maintaining separate layout headers. Both should use the shared layout and navigation partial.

The page view model should provide the current path or active navigation key:

```go
type LayoutViewModel struct {
    Title       string
    CurrentPath string
    ActiveNav   string
    Content     any
}
```

Prefer calculating active navigation in the handler or layout view-model builder. Do not infer active state with Alpine.js.

## 5. Ordered Subtasks

### A. Define shared layout data

1. Add `LayoutViewModel` or an equivalent shared view model.
2. Pass `CurrentPath` or `ActiveNav` from every page handler.
3. Define active matching rules for list and detail pages.
4. Keep the view model independent from GORM models.

### B. Create the navigation partial

1. Add desktop links for Dashboard, Company accounts, and Receivables.
2. Add primary action links for New account and New receivable.
3. Apply active styling to the current section.
4. Add visible labels and accessible focus styles.
5. Use `aria-current="page"` for the active link.

### C. Add Alpine.js mobile behavior

Use Alpine only for local menu state:

```html
<div x-data="{ open: false }" class="navigation-shell">
  <button
    type="button"
    @click="open = !open"
    :aria-expanded="open.toString()"
    aria-controls="mobile-navigation">
    <span class="sr-only">Toggle navigation</span>
    Menu
  </button>

  <nav id="mobile-navigation" x-cloak x-show="open">
    <!-- shared links -->
  </nav>
</div>
```

Rules:

- Close the menu after selecting a link.
- Close the menu on Escape.
- Keep links usable if Alpine.js fails to load.
- Do not use Alpine to decide the current route.
- Do not use HTMX for normal navigation links unless a specific partial-navigation requirement exists.

### D. Integrate every page

Update:

- Account list page.
- New account page.
- Edit account page.
- Receivable list page.
- New receivable page.
- Receivable detail page.
- Dashboard page when implemented.

All pages must render the same shared navigation partial.

### E. Add responsive styling

Use the Tailwind/Windmill visual language:

- Desktop horizontal navigation or sidebar according to the selected shell.
- Mobile menu button below the desktop breakpoint.
- Clear active background and text contrast.
- Visible focus ring.
- No horizontal scrolling on narrow screens.
- Keep primary create actions visible without overwhelming the page.

## 6. HTMX Rules

- Normal navigation uses standard links.
- Do not replace the entire application shell during ordinary HTMX form updates.
- HTMX responses should target page content or a local fragment, not the navigation bar.
- After a successful command redirect, render the destination page with the correct active navigation state.
- Use `HX-Redirect` for successful HTMX commands when the destination changes.

## 7. Testing Tasks

### Template tests

- Dashboard link renders.
- Company accounts link renders.
- Receivables link renders.
- New account and new receivable links render.
- Active state is applied for `/accounts` and account detail/edit pages.
- Active state is applied for `/receivables` and receivable detail pages.
- `aria-current="page"` is present only on the active link.

### HTTP tests

- Account pages render the shared navigation.
- Receivable pages render the shared navigation.
- Navigation links point to the correct routes.
- Successful HTMX commands redirect to a page with the correct active item.

### Manual checks

- Desktop navigation works with keyboard only.
- Mobile menu opens and closes.
- Escape closes the mobile menu.
- Selecting a mobile link closes the menu.
- Navigation works with JavaScript disabled.
- Active state remains correct on detail and edit pages.

## 8. Definition Of Done

- One shared navigation partial is used by all full pages.
- Company account and receivable pages are reachable from the navigation.
- Dashboard link is included in the shared contract.
- Active route styling is server-rendered.
- Mobile behavior uses Alpine.js only for local presentation state.
- Standard links work without JavaScript.
- HTMX commands preserve the shared shell and redirect correctly.
- Navigation is keyboard accessible and responsive.
- Template and HTTP tests pass.
- `go test ./...`, `go vet ./...`, and formatting pass.
- `analysis/progress.md` is updated.

## 9. Next Task

Implement the shared layout and navigation partial before adding more page-specific headers.
