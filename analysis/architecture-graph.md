# Architecture Graph

This graph reflects the current implementation. The server is composed from vertical slices; each slice owns its handlers, domain logic, and repository boundary. The migration executable shares configuration and database infrastructure with the server.

## Runtime Flow

```mermaid
flowchart LR
    browser[Browser]
    server[cmd/server]
    mux[net/http ServeMux]
    templates[web/templates<br/>html/template + HTMX + Alpine.js]
    static[web/static<br/>embedded CSS + JavaScript]
    database[(SQL Server)]

    browser --> mux
    server --> config[config.Load]
    server --> dbopen[platform/database.Open]
    server --> mux
    mux --> accounts[accounts slice]
    mux --> products[products slice]
    mux --> suppliers[suppliers slice]
    mux --> receivables[receivables slice]
    mux --> quotations[quotations slice]
    mux --> salesorders[salesorders slice]
    mux --> dashboard[dashboard slice]
    mux --> templates
    mux --> static
    accounts --> database
    products --> database
    suppliers --> database
    receivables --> database
    quotations --> database
    salesorders --> database
    dashboard --> database
    templates --> browser
    static --> browser

    migrate[cmd/migrate] --> config
    migrate --> migrations[migrations]
    migrations --> database
```

## Package Dependencies

```mermaid
flowchart TB
    subgraph EntryPoints
        server[cmd/server]
        migrate[cmd/migrate]
    end

    subgraph Infrastructure
        config[internal/config]
        db[internal/platform/database]
        migrations[internal/migrations]
        templates[internal/web/templates]
        static[internal/web/static]
    end

    subgraph Shared
        money[shared/money]
        tax[shared/tax]
        date[shared/businessdate]
        uom[shared/uom]
    end

    subgraph VerticalSlices
        accounts[slices/accounts]
        dashboard[slices/dashboard]
        products[slices/products]
        suppliers[slices/suppliers]
        receivables[slices/receivables]
        quotations[slices/quotations]
        salesorders[slices/salesorders]
    end

    server --> config
    server --> db
    server --> templates
    server --> static
    server --> accounts
    server --> dashboard
    server --> products
    server --> suppliers
    server --> receivables
    server --> quotations
    server --> salesorders

    migrate --> config
    migrate --> migrations
    migrations --> config
    migrations --> db

    accounts --> dashboard
    accounts --> db
    products --> db
    products --> uom
    suppliers --> db
    suppliers --> money
    receivables --> accounts
    receivables --> db
    receivables --> date
    receivables --> money
    receivables --> tax
    quotations --> products
    quotations --> templates
    quotations --> money
    quotations --> tax
    quotations --> uom
    salesorders --> quotations
    salesorders --> db
    dashboard --> db
    dashboard --> date
    dashboard --> money

    accounts --> templates
    products --> templates
    suppliers --> templates
    receivables --> templates
    salesorders --> templates
```

## Conventions

- HTTP handlers register routes with the shared `http.ServeMux`.
- Repositories hide GORM and SQL Server details from slice services and handlers.
- Templates and static assets are embedded into the binary.
- `analysis/000-implementation-ladder.md` remains the source of truth for implementation status.
