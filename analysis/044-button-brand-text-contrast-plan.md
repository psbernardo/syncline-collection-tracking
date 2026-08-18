# 044 Branded Button Text Contrast Plan

## Goal

Ensure every button or action control with a dark branded background uses white text, including hover states and responsive layouts. Review the shared design so the fix applies consistently across all application pages instead of patching individual screens.

## Current State

- The shared stylesheet defines `--brand: #3147a8` in `internal/web/static/app.css`.
- `.button-primary` uses `var(--brand)` as its background and `var(--brand-deep)` on hover.
- An earlier `.button-primary` rule sets `color: #fff`.
- A later, equally specific `.button` rule sets `color: #344054`, overriding the primary-button text color in the cascade.
- Navigation action links use a dark blue background and are later assigned `color: #c3cff5`, so they are not pure white even though they are also branded actions.
- Several trading templates use `class="button primary"` instead of `class="button button-primary"`. Those controls currently render as ordinary white buttons because no `.button.primary` style exists.
- Danger buttons already explicitly use white text and should remain unchanged.
- Select, modal-close, snackbar-dismiss, and quiet-danger controls are contextual controls with transparent or light backgrounds; they should not inherit the branded-button treatment.

## Affected Shared Design Surface

The shared layout loads `internal/web/static/app.css` for every page, so the primary-button cascade affects:

- Dashboard and dashboard summary actions.
- Company account list and form actions.
- Receivable list, form, detail, payment, and reversal actions.
- Product list and form actions.
- Supplier and supplier-product list and form actions.
- Quotation list, create/edit, and detail actions.
- Sidebar action links such as `New account`, `New receivable`, `New product`, and `New supplier`.
- Mobile versions of page-heading and detail actions, where the same controls become full-width.

## Implementation Plan

### PBI 01: Fix the shared branded-button cascade

- Keep the existing `--brand`, `--brand-deep`, and shadow tokens.
- Add an explicit `color: #fff` to the final `.button-primary` rule in the workspace-polish section, after the final generic `.button` rule, so source order cannot override it.
- Add an explicit `color: #fff` to the final `.nav-action` rule because its dark background is a branded action surface.
- Preserve the existing hover backgrounds and verify that hover text remains white.
- Do not change ordinary `.button`, `.button-danger`, quiet-danger, select-option, modal-close, or snackbar-dismiss colors unless a page inspection shows they use a branded background.

### PBI 02: Normalize primary button markup

- Replace `button primary` with `button button-primary` in the Products, Suppliers, Supplier Products, and Quotations templates.
- Review all templates for additional class variants before editing; use the shared class rather than introducing a second `.primary` selector.
- Keep ordinary navigation/back/cancel buttons neutral white.

### PBI 03: Verify all page states

- Inspect every page using the shared layout on desktop and mobile widths.
- Check default, hover, keyboard-focus, disabled, and loading states for branded buttons.
- Confirm text remains white on both `#3147a8` and the darker hover `#1d2b68` backgrounds.
- Confirm links/buttons with light or transparent backgrounds retain their intended dark or contextual text colors.
- Confirm the change does not affect customer-facing PDF styling, which is rendered separately from the web stylesheet.

## Validation

- Search the repository for all `button-primary`, `nav-action`, `class="button primary"`, and `--brand` usages after implementation.
- Run the Go test suite with `go test ./...`.
- Perform a browser smoke test for dashboard, accounts, receivables, products, suppliers, supplier products, and quotations, including a quotation detail page and a receivable payment/detail page.
- Use keyboard focus and mobile viewport checks to ensure contrast and layout remain usable.

## Acceptance Criteria

- Every control with the `#3147a8` branded background displays white text.
- Branded hover states also display white text.
- Sidebar action links display white text on their dark blue backgrounds.
- Product, supplier, supplier-product, and quotation primary actions use the same shared primary-button class and visual treatment.
- Neutral, danger, quiet-danger, dropdown, modal, and snackbar controls retain their existing intentional colors.
- No page-specific duplicate color overrides are required.
- `go test ./...` passes.
