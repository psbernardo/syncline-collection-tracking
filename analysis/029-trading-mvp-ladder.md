# Trading MVP Implementation Ladder

This is the dependency-ordered plan for the B2B trading and procurement MVP. The existing collection-tracking implementation remains the foundation. New work must follow the repository's vertical-slice architecture, TDD workflow, SQL Server migration rules, integer-scaled PHP money, and server-rendered HTMX UI conventions.

## Current Repository Baseline

- `internal/slices/accounts` already manages customer company accounts.
- `internal/slices/receivables` already manages invoices/POs, tax amounts, due dates, full payment acknowledgement, and collection status.
- `internal/slices/dashboard` already provides collection aggregation.
- `internal/shared/money` is the money boundary and must be reused.
- `internal/shared/tax` is the tax boundary and must be extended only when a new rule is required.
- `internal/migrations` uses ordered SQL Server migrations registered in `migrations.go`.
- `cmd/server/main.go` is the composition root for slice registration.

## MVP Boundary

Included:

- Product and supplier master data
- Supplier-product relationship and current reference price
- RFQ capture
- Quotation with line pricing, VAT/tax selection, and commission
- Customer PO reference and Sales Order creation
- Quotation acceptance and locking when a Sales Order is created
- Multiple supplier POs for one Sales Order
- Sales-line to supplier-line allocation through the UI
- Full receipt only; partial delivery is not supported
- Invoice creation from a fully received Sales Order
- Handoff of the invoice to the existing receivables workflow
- Collection status and full payment tracking through the existing slice
- Estimated versus actual profitability at transaction level

Excluded:

- Partial delivery
- Partial payment
- Credit approval and risk scoring
- Automatic supplier selection
- Supplier price automation
- Multi-currency
- Quotation revision history
- Customer self-service
- Accounting integration
- Agent commission settlement workflow

## Ladder

| Sequence | Plan | Dependency gate |
|---:|---|---|
| 029 | MVP decisions and architecture gate | Business rules and integration boundary approved |
| 030 | Shared trading foundation and persistence contract | 029 complete |
| 031 | Product, supplier, and supplier-price master data | 030 complete |
| 032 | RFQ and quotation pricing slice | 031 complete |
| 033 | Customer PO and Sales Order conversion | 032 complete |
| 034 | Multi-supplier procurement and allocation | 033 complete |
| 035 | Full receiving and invoice creation | 034 complete |
| 036 | Receivables handoff and collection integration | 035 complete |
| 037 | MVP reporting, audit, security, and release verification | 030 through 036 complete |

## Development Rule

Do not start a PBI until its listed readiness dependencies are complete. Each PBI is developed as a vertical slice: domain tests, command/query behavior, repository tests, handler tests, templates, migration, and acceptance verification.
