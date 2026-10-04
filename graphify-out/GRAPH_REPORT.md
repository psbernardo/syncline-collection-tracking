# Graph Report - syncline-collection-tracking  (2026-10-04)

## Corpus Check
- 204 files · ~311,374 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 3013 nodes · 6242 edges · 153 communities (141 shown, 12 thin omitted)
- Extraction: 94% EXTRACTED · 6% INFERRED · 0% AMBIGUOUS · INFERRED: 390 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `207123e9`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- alpine.min.js
- htmx.min.js
- pdfPage
- Supplier
- app.js
- gorm.io/gorm.DB
- Product
- Collection Tracking System
- Pricing Domain Schema — Commission Included.md
- net/http.Request
- Handler
- context.Context
- Handler
- Payment
- r
- .Render
- Amount
- 040 Quotation PDF Screenshot Layout Plan
- Sales Order Edit and Acknowledgement Plan
- purchaseorders/pdf.go
- ListQuery
- First Vertical Slice Implementation Plan
- 054 Purchase Order Multi-Source and Direct Purchase Plan
- time.Time
- testing.T
- Handler
- CompanyAccount
- Delivery Receivable Filter Plan
- Full Payment Acknowledgment Plan
- 031 Trading Master Data Plan
- Receivable Invoice Selection Plan
- service
- Vertical Slice Page Coding Standard
- Quotation to Sales Order Conversion Plan
- B2B Trading and Procurement Business Model.md
- PurchaseOrder
- Receivable Tax Rule Plan
- 038 Supplier-Product Searchable Dropdown Plan
- 039 Quotation PDF Download Plan
- Sales Order Number Plan
- quotations/repository.go
- Tailwind UI and HTMX/Alpine Integration Plan
- Company Account Slice Implementation Plan
- 5. Ordered Subtasks
- Receivable Classification Display Plan
- 4. Ordered Subtasks
- Acknowledge Payment Page Design Plan
- Acknowledge Full Payment Tax Rule Plan
- 057 Purchase Order PDF Download Plan
- export.go
- Handler
- Shared Navigation Bar Implementation Plan
- 5. Ordered Subtasks
- Sales Order PO, Edit, and PDF Plan
- 056 Purchase Order Create/Edit Alignment Plan
- NewDeliveryReceivableWithTax
- New
- Receivable New Column Implementation Plan
- Reverse Payment Acknowledgement Plan
- Receivable Tax Computation Preview Plan
- 042 Apply Shared Sales Order Layout to Quotations
- Sales Order to Invoice Conversion Plan
- 052 Separate Quotation, Sales Order, and Invoice PDF Templates Plan
- Sales Order Invoice and Receivable Plan
- dashboard/domain.go
- Invoice
- helpers.go
- toModel
- 4. Ordered Subtasks
- VAT-Inclusive Tax Calculation Plan
- net/http.ResponseWriter
- Query
- Duplicate PO Handling Plan
- Duplicate Invoice Number Validation Plan
- 040 Reusable Text Date Input Plan
- 043 Quotation Line Margin and Profit Plan
- Sales Order PDF Header and PO Layout Plan
- 055 Supplier Bulk Product Configuration Plan
- Invoice Creation PDF New Tab Plan
- Text
- Line
- salesorders/handler_test.go
- Handler
- salesOrderPage
- html/template.Template
- Quotation
- 000-implementation-ladder.md
- Handler
- NewHandler
- Parse
- GormRepository
- Collection Tracking System: Technical and Architecture Plan
- Database Schema and Index Plan
- Reusable Feedback Snackbar Plan
- Invoice Receivable Confirmation Plan
- Dashboard Totals Implementation Plan
- Validation Sequence
- Handler
- pt
- migrations.go
- Sales Order Duplicate Process Plan
- salesorders/repository.go
- 5. Ordered Subtasks
- tax.go
- InvoiceOption
- receivables/handler.go
- 044 Branded Button Text Contrast Plan
- Config
- GormRepository
- applyFilters
- NewCompanyAccount
- Migration Implementation Plan
- run
- Confirmed Contract
- accounts/handler.go
- NewHandler
- 5. Domain Model Mapping
- SalesOrder
- supplierModel
- Epic E-03: Quote-to-Price
- Epic E-08: Reporting, Reliability, and Release
- dashboard/queries.go
- AccountViewModel
- Implementation Ladder
- 6. Vertical Slice Boundaries
- Trading MVP Implementation Ladder
- Epic E-01: Trading Domain Foundation
- Epic E-04: Customer Commitment
- Epic E-05: Multi-Supplier Procurement
- Epic E-06: Fulfillment and Invoicing
- Epic E-07: Collection Integration
- 11. Environment Separation
- 12. Testing Strategy and TDD
- 19. Technical Gap Analysis
- Architecture Graph
- 20. Development Blockers
- NewHandler
- models_test.go
- NewService
- dashboard/domain_test.go
- TestNewHandlerParsesTemplates
- Pe
- 2. Confirmed Technical Direction
- TestRegistryIsOrderedAndUnique
- Repository
- 3. Architecture Style
- 7. Database and GORM Strategy
- receivables/repository.go
- THIRD-PARTY-NOTICES.md
- github.com/psbernardo/syncline-collection-tracking

## God Nodes (most connected - your core abstractions)
1. `Amount` - 89 edges
2. `DeliveryReceivable` - 57 edges
3. `Quotation` - 46 edges
4. `run()` - 39 edges
5. `Text()` - 34 edges
6. `SalesOrder` - 33 edges
7. `He()` - 28 edges
8. `Parse()` - 27 edges
9. `Handler` - 27 edges
10. `e()` - 27 edges

## Surprising Connections (you probably didn't know these)
- `run()` --calls--> `Up()`  [EXTRACTED]
  cmd/migrate/main.go → internal/migrations/migrations.go
- `run()` --calls--> `Load()`  [EXTRACTED]
  cmd/server/main.go → internal/config/config.go
- `run()` --calls--> `Open()`  [EXTRACTED]
  cmd/server/main.go → internal/platform/database/database.go
- `run()` --calls--> `NewService()`  [EXTRACTED]
  cmd/server/main.go → internal/slices/accounts/commands.go
- `run()` --calls--> `NewHandler()`  [EXTRACTED]
  cmd/server/main.go → internal/slices/accounts/handler.go

## Import Cycles
- None detected.

## Communities (153 total, 12 thin omitted)

### Community 0 - "alpine.min.js"
Cohesion: 0.03
Nodes (108): A(), ai(), an(), ao(), At(), ce(), ci(), cn() (+100 more)

### Community 1 - "htmx.min.js"
Cohesion: 0.10
Nodes (76): rn(), a(), Ae(), an(), B(), Be(), bn(), bt() (+68 more)

### Community 2 - "pdfPage"
Cohesion: 0.06
Nodes (50): decodeVersion(), encodeVersion(), formatPercent(), formatQuotationDate(), formatQuotationDateCanonical(), formatQuotationDateInput(), formatTaxRate(), Repository (+42 more)

### Community 3 - "Supplier"
Cohesion: 0.05
Nodes (48): quotationProductOptions(), audit(), Repository, SupplierProduct, NewService(), normalizeProductIDs(), NewSupplier(), NewSupplierProduct() (+40 more)

### Community 4 - "app.js"
Cohesion: 0.07
Nodes (59): addLine(), addProduct(), allVisibleSelected(), appendBlankRow(), applyTaxToRows(), clear(), clearAll(), close() (+51 more)

### Community 5 - "gorm.io/gorm.DB"
Cohesion: 0.03
Nodes (64): gorm.io/gorm.DB, down0001(), up0001(), down0002(), up0002(), down0003(), up0003(), down0004() (+56 more)

### Community 6 - "Product"
Cohesion: 0.13
Nodes (13): audit(), Repository, NewService(), toModel(), NewGormRepository(), CreateCommand, GormRepository, listPage (+5 more)

### Community 7 - "Collection Tracking System"
Cohesion: 0.05
Nodes (43): 10. Minimum Data to Confirm, 11. Permissions and Audit, 12. Operational Views, 13. MVP Acceptance Scenarios, 14. Gap Analysis, 15. Remaining Decisions Before Design, 16. Technical Direction, 17. Approval Gate (+35 more)

### Community 8 - "Pricing Domain Schema — Commission Included.md"
Cohesion: 0.05
Nodes (38): 10. Cost Calculation, 11. Agent Commission, 12. Commission Calculation, 13. Commission Types, 14. Gross Sales vs Net Sales, 15. Profit Calculation, 16. Margin Calculation, 17. Quotation Line (+30 more)

### Community 9 - "net/http.Request"
Cohesion: 0.12
Nodes (25): net/http.Request, applyLineDescriptions(), excludedLineIDs(), excludedLineSet(), formatOptionalDate(), purchaseOrderPDFRenderer, hasSubmittedLine(), manilaNow() (+17 more)

### Community 10 - "Handler"
Cohesion: 0.13
Nodes (21): applyDuplicateFormValues(), conversionError(), duplicatePage(), orderedFormLineIndices(), parseID(), quotationConvertible(), safeFilename(), salesOrderProductSelect() (+13 more)

### Community 11 - "context.Context"
Cohesion: 0.10
Nodes (8): context.Context, NewGormRepository(), DeliveryReceivable, fakeReceivableRepository, GormRepository, loadMoreReceivableRepository, paidReceivableRepository, paymentReceivableRepository

### Community 12 - "Handler"
Cohesion: 0.22
Nodes (9): RuleCode, RuleOption, RuleOptions(), NewIdempotencyKey(), ValidationErrors, isHTMX(), parseFormID(), requestID() (+1 more)

### Community 13 - "Payment"
Cohesion: 0.07
Nodes (29): Handler, ListResult, Repository, SupplierOption, isDuplicateIndexError(), NewService(), randomID(), NewPayment() (+21 more)

### Community 14 - "r"
Cohesion: 0.11
Nodes (34): ar(), Ct(), F(), ft(), H(), Hn(), mr(), nt() (+26 more)

### Community 15 - ".Render"
Cohesion: 0.18
Nodes (16): amountPointer(), formatOptionalAmount(), invoiceDescription(), invoiceFinalContentFits(), invoiceTaxPercent(), maxFloat(), NewInvoicePDFRenderer(), nonZeroAmountPointer() (+8 more)

### Community 16 - "Amount"
Cohesion: 0.24
Nodes (13): Amount, CalculateLine(), CalculateProfitability(), CalculateTotals(), Line, margin(), multiply(), NormalizeStatus() (+5 more)

### Community 17 - "040 Quotation PDF Screenshot Layout Plan"
Cohesion: 0.07
Nodes (28): 040 Quotation PDF Screenshot Layout Plan, 1. Dependency and Assets, 2. Renderer Structure, 3. Coordinate and Style Constants, 4. Text and Wrapping, 5. Pagination, 6. Download Endpoint, Acceptance Criteria (+20 more)

### Community 18 - "Sales Order Edit and Acknowledgement Plan"
Cohesion: 0.07
Nodes (28): Acknowledgement and Warning Flow, Actions, Allocation invariants, Current-State Analysis, Decisions Required Before Coding, Delivery Sequence, Detail page, Domain and Repository Contract (+20 more)

### Community 19 - "purchaseorders/pdf.go"
Cohesion: 0.14
Nodes (19): github.com/signintech/gopdf.GoPdf, io.Writer, Configure(), FormatUnitPrice(), Wrap(), Write(), formatPurchaseOrderDate(), formatPurchaseOrderOptionalDate() (+11 more)

### Community 20 - "ListQuery"
Cohesion: 0.13
Nodes (27): ListResult, TestCursorIsBoundToFilters(), parseListQuery(), classificationTone(), decodeCursor(), encodeCursor(), filterSignature(), ValidationErrors (+19 more)

### Community 21 - "First Vertical Slice Implementation Plan"
Cohesion: 0.07
Nodes (27): 1. Goal, 2. Implementation Order, 3. Route Plan, 4. HTMX UI Patterns, 5. Alpine.js UI Patterns, 6. TDD Checkpoints, 7. Definition Of Done, 8. Reference Documentation (+19 more)

### Community 22 - "054 Purchase Order Multi-Source and Direct Purchase Plan"
Cohesion: 0.07
Nodes (27): 054 Purchase Order Multi-Source and Direct Purchase Plan, Add/remove behavior, Confirmed Decisions, Current-State Impact, Data Model, Delivery Sequence, Direct mode, Direct purchase mode (+19 more)

### Community 23 - "time.Time"
Cohesion: 0.14
Nodes (21): auditModel, idempotencyModel, time.Time, ClassificationBoundaries(), DueDate(), FormatUTC(), Parse(), TestDueDateCountsDeliveryAsDayOne() (+13 more)

### Community 24 - "testing.T"
Cohesion: 0.23
Nodes (26): testing.T, NewService(), NewHandler(), receivableExportURL(), TestInvalidReceivableCreatePreservesManualInvoiceNumber(), TestInvalidReceivableCreateReturnsHTMXFragment(), TestNewReceivableFormRendersAccounts(), TestPaidReceivableDetailRendersReversalDialog() (+18 more)

### Community 25 - "Handler"
Cohesion: 0.32
Nodes (6): Handler, NewIdempotencyKey(), ValidationErrors, isHTMX(), requestID(), NewAccountForm()

### Community 26 - "CompanyAccount"
Cohesion: 0.11
Nodes (11): accountModel, CreateAccountCommand, fakeRepository, GormRepository, UpdateAccountCommand, commandService, hashPayload(), CompanyAccount (+3 more)

### Community 27 - "Delivery Receivable Filter Plan"
Cohesion: 0.08
Nodes (24): 1. Goal, 2. Filter Contract, 3. Default Scope, 4. Query Model, 5. Ordered Subtasks, 6. UX Rules, 7. Tests, 8. Acceptance Scenarios (+16 more)

### Community 28 - "Full Payment Acknowledgment Plan"
Cohesion: 0.08
Nodes (24): 10. Tests, 11. Implementation Order, 12. Definition Of Done, 1. Goal, 2. Scope, 3. Business Rules, 4. Existing Database Contract, 5. Domain and Commands (+16 more)

### Community 29 - "031 Trading Master Data Plan"
Cohesion: 0.08
Nodes (24): 031 Trading Master Data Plan, 10. Dependencies and readiness gates, 1. Current-plan analysis, 2. Scope and non-goals, 3. PBI 01: Product master, 4. PBI 02: Supplier master, 5. PBI 03: Supplier-product relationship and current reference price, 6. Persistence contract (+16 more)

### Community 30 - "Receivable Invoice Selection Plan"
Cohesion: 0.08
Nodes (24): Acceptance Scenarios, Application Changes, Current-State Findings, Decisions Required Before Implementation, Definition Of Done, Domain and commands, Domain/service tests, Existing-Data Handling (+16 more)

### Community 31 - "service"
Cohesion: 0.14
Nodes (14): Repository, hashPayload(), hashPaymentPayload(), hashReversePaymentPayload(), isDuplicateInvoiceError(), isDuplicatePOError(), AccountRepository, auditModel (+6 more)

### Community 32 - "Vertical Slice Page Coding Standard"
Cohesion: 0.08
Nodes (23): 10. Page-Specific Standards, 11. Testing Standard Per Slice, 12. Per-Page Definition Of Done, 13. References, 1. Required Slice Structure, 2. Naming Conventions, 3. Request Flow, 4. Commands and Queries (+15 more)

### Community 33 - "Quotation to Sales Order Conversion Plan"
Cohesion: 0.08
Nodes (23): Conversion state, Create Sales Order page, Current-State Gaps, Data Model, Decisions Required Before Coding, Delivery Sequence, Domain and Repository Design, Domain/repository tests (+15 more)

### Community 34 - "B2B Trading and Procurement Business Model.md"
Cohesion: 0.08
Nodes (23): 10. Customer Delivery and Invoice, 11. Accounts Receivable and Collection, 1. Client Request for Quotation, 2. Supplier Price Sourcing, 3. Price Negotiation, 4. Customer Quotation, 5. Customer Purchase Order, 6. Sales Order (+15 more)

### Community 35 - "PurchaseOrder"
Cohesion: 0.07
Nodes (38): BuildDirectLines(), BuildLines(), GeneratedNumber(), SupplierProduct, multiply(), TestBuildLinesCopiesAllSalesOrderLinesAndSupplierData(), TestBuildLinesRejectsQuantityAboveSalesLine(), TestBuildLinesRejectsUnknownSourceLine() (+30 more)

### Community 36 - "Receivable Tax Rule Plan"
Cohesion: 0.09
Nodes (22): 10. Open Decisions, 11. Definition Of Done, 1. Goal, 2. Precision Policy, 3. Tax Rule Behavior, 4. Source Of Truth And Persistence, 5. Domain And Application Changes, 6. HTTP Form And UI (+14 more)

### Community 37 - "038 Supplier-Product Searchable Dropdown Plan"
Cohesion: 0.09
Nodes (22): 038 Supplier-Product Searchable Dropdown Plan, 1. Current implementation analysis, 2. MVP decision, 3. Form behavior, 4. UI component plan, 5. Data and handler changes, 6. Suggested routes and request shape, 7. Acceptance scenarios (+14 more)

### Community 38 - "039 Quotation PDF Download Plan"
Cohesion: 0.09
Nodes (22): 039 Quotation PDF Download Plan, Acceptance Criteria, Current-State Findings, Data and Calculation Rules, Final Decision Matrix, Fonts and Assets, Go Libraries, Goal (+14 more)

### Community 39 - "Sales Order Number Plan"
Cohesion: 0.09
Nodes (22): Create-Page Behavior, Current-State Analysis, Decisions Required Before Coding, Delivery Sequence, Detail, Detail and List Pages, Domain and Repository Changes, Edit behavior (+14 more)

### Community 40 - "quotations/repository.go"
Cohesion: 0.11
Nodes (16): applyTaxRule(), TestApplyTaxRuleUsesQuotationDefault(), ValidTermsDays(), TestGeneratedQuotationNumber(), generatedQuotationNumber(), ConversionSummary, Repository, NewGormRepository() (+8 more)

### Community 41 - "Tailwind UI and HTMX/Alpine Integration Plan"
Cohesion: 0.09
Nodes (21): 10. Implementation Sequence, 11. Acceptance Criteria, 12. References, 1. Recommended Template, 2. Scope Of Template Reuse, 3. Proposed Visual Language, 4. Asset and Build Plan, 5. Go Template Integration (+13 more)

### Community 42 - "Company Account Slice Implementation Plan"
Cohesion: 0.18
Nodes (10): 1. Goal, 2. Scope, 3. Dependencies, 4. Slice Structure, 6. Acceptance Scenarios, 7. Definition Of Done, 8. Next Slice, Company Account Slice Implementation Plan (+2 more)

### Community 43 - "5. Ordered Subtasks"
Cohesion: 0.09
Nodes (21): 1. Goal, 2. Scope, 3. Dependencies, 4. Slice Structure, 5. Ordered Subtasks, 6. Acceptance Scenarios, 7. Definition Of Done, 8. Next Slice (+13 more)

### Community 44 - "Receivable Classification Display Plan"
Cohesion: 0.09
Nodes (21): 1. Goal, 2. Classification Rules, 3. Scope, 4. Ordered Subtasks, 5. Status Styling, 6. TDD Subtasks, 7. Acceptance Scenarios, 8. Definition Of Done (+13 more)

### Community 45 - "4. Ordered Subtasks"
Cohesion: 0.09
Nodes (21): 1. Current Behavior, 2. Goal, 3. Proposed Edit Rules, 4. Ordered Subtasks, 5. Tests, 6. Acceptance Scenarios, 7. Definition Of Done, A. Add company name to list/detail projections (+13 more)

### Community 46 - "Acknowledge Payment Page Design Plan"
Cohesion: 0.09
Nodes (21): 10. Tests, 11. Implementation Order, 12. Definition Of Done, 1. Goal, 2. Current UI Assessment, 3. Recommended Experience, 4. Route and Navigation Changes, 5. Page Structure (+13 more)

### Community 47 - "Acknowledge Full Payment Tax Rule Plan"
Cohesion: 0.09
Nodes (21): 10. Implementation Steps, 11. Test Plan, 12. Decisions Required, 13. Definition Of Done, 1. Goal, 2. Critical Design Rule, 3. Current Payment Behavior, 4. Payment Page Requirements (+13 more)

### Community 48 - "057 Purchase Order PDF Download Plan"
Cohesion: 0.09
Nodes (21): 057 Purchase Order PDF Download Plan, Acceptance Criteria, Alignment With Quotation Download, Current-State Findings, Data Contract Decisions, Goal, Handler tests, HTTP and UI Changes (+13 more)

### Community 49 - "export.go"
Cohesion: 0.19
Nodes (17): excelize.File, excelize.Fill, buildExportReport(), exportStatusLabel(), filterSummary(), generatedAtDisplay(), solidFill(), TestBuildExportReportGroupsByCompanyWithSubtotals() (+9 more)

### Community 50 - "Handler"
Cohesion: 0.17
Nodes (14): errorMessage(), ListResult, SupplierOption, hasSupplier(), inputFromRequest(), NewHandler(), parseIDValue(), pathID() (+6 more)

### Community 51 - "Shared Navigation Bar Implementation Plan"
Cohesion: 0.10
Nodes (20): 1. Goal, 2. Scope, 3. Navigation Contract, 4. Template Structure, 5. Ordered Subtasks, 6. HTMX Rules, 7. Testing Tasks, 8. Definition Of Done (+12 more)

### Community 52 - "5. Ordered Subtasks"
Cohesion: 0.10
Nodes (20): 1. Goal, 2. Scope, 3. Component Contract, 4. Template Structure, 5. Ordered Subtasks, 6. Acceptance Scenarios, 7. Definition Of Done, 8. Implementation Decision Gate (+12 more)

### Community 53 - "Sales Order PO, Edit, and PDF Plan"
Cohesion: 0.10
Nodes (20): Business Rules, Create sales-order page, Current-State Analysis, Database Migration, Decisions Required Before Coding, Delivery Sequence, Domain and Repository Changes, Edit behavior (+12 more)

### Community 54 - "056 Purchase Order Create/Edit Alignment Plan"
Cohesion: 0.10
Nodes (20): 056 Purchase Order Create/Edit Alignment Plan, Create, Create context, Current-State Analysis, Decisions Required Before Coding, Delivery Sequence, Domain and Repository Changes, Domain/repository tests (+12 more)

### Community 55 - "NewDeliveryReceivableWithTax"
Cohesion: 0.10
Nodes (26): TestDuplicateInvoiceErrorRecognized(), TestNewDeliveryReceivableAllowsMissingInvoiceNumber(), TestNewDeliveryReceivableRejectsInvalidInvoiceNumber(), TestNewDeliveryReceivableRejectsNonAlphanumericPO(), TestNewDeliveryReceivableTrimsManualInvoiceNumber(), TestNormalizePONumber(), isAlphaNumeric(), NewDeliveryReceivable() (+18 more)

### Community 56 - "New"
Cohesion: 0.18
Nodes (12): Code, Normalize(), TestNormalizeAndValidate(), TestOptionsOrder(), Valid(), New(), required(), TestNewProductNormalizesAndValidates() (+4 more)

### Community 57 - "Receivable New Column Implementation Plan"
Cohesion: 0.10
Nodes (19): 1. Current State, 2. Approved Field Contract, 3. Recommended Scope Classification, 4. Ordered Implementation Steps, 5. Test Plan, 6. Acceptance Criteria, 7. Documentation Updates, 8. Migration Behavior (+11 more)

### Community 58 - "Reverse Payment Acknowledgement Plan"
Cohesion: 0.10
Nodes (19): 10. Tests, 11. Implementation Order, 12. Definition Of Done, 1. Purpose, 2. Current State, 3. Challenge To The Proposal, 4. Recommended Business Rules, 5. Decisions Required Before Coding (+11 more)

### Community 59 - "Receivable Tax Computation Preview Plan"
Cohesion: 0.10
Nodes (19): 1. Current State, 2. Goal, 3. Recommended Interaction, 4. Server Changes, 5. Template And Styling Changes, 6. Tests, 7. Acceptance Criteria, 8. Open Product Decision (+11 more)

### Community 60 - "042 Apply Shared Sales Order Layout to Quotations"
Cohesion: 0.10
Nodes (19): 042 Apply Shared Sales Order Layout to Quotations, 1. Extend Quotation Customer Projection, 2. Create Shared PDF View Model, 3. Update Quotation Mapping, 4. Update Renderer Layout, 5. Update Quotation Detail Download, 6. Verify Sales Order Regression, Acceptance Criteria (+11 more)

### Community 61 - "Sales Order to Invoice Conversion Plan"
Cohesion: 0.10
Nodes (19): Conversion eligibility, Current-State Findings, Data Model, Decisions Required Before Coding, Delivery Sequence, Domain and Transaction Design, Domain/repository/service tests, Goal (+11 more)

### Community 62 - "052 Separate Quotation, Sales Order, and Invoice PDF Templates Plan"
Cohesion: 0.10
Nodes (19): 052 Separate Quotation, Sales Order, and Invoice PDF Templates Plan, Acceptance Criteria, Current-State Findings, Decisions Superseding Earlier Analysis, Desired Boundary, Document Contracts, Files Expected To Change, Goal (+11 more)

### Community 63 - "Sales Order Invoice and Receivable Plan"
Cohesion: 0.10
Nodes (19): Application Changes, Confirmed Business Rules, Cross-links, Current-State Findings, Database Changes, Definition of Done, Delivery Sequence, Domain and service tests (+11 more)

### Community 64 - "dashboard/domain.go"
Cohesion: 0.20
Nodes (15): ClassificationTotal, ClientReceivableSummary, DashboardTotals, DashboardViewModel, SalesDashboard, SalesDashboardViewModel, SalesLeader, SalesLeaderViewModel (+7 more)

### Community 65 - "Invoice"
Cohesion: 0.11
Nodes (17): hasTaxCode(), ReceivableTaxRule(), TestBuildPreviewMapsSalesOrderTaxRule(), TestReceivableTaxRule(), isUniqueError(), moneyAmount(), NewGormRepository(), nullableTaxRule() (+9 more)

### Community 66 - "helpers.go"
Cohesion: 0.14
Nodes (17): DefaultSeller(), envOr(), FindFont(), findLogo(), SafeFilename(), PurchaseOrderPDFRenderer, NewPurchaseOrderPDFRenderer(), TestPurchaseOrderPDFRendererPaginatesLongOrders() (+9 more)

### Community 67 - "toModel"
Cohesion: 0.36
Nodes (6): nullableInvoiceID(), nullableInvoiceNumber(), nullableRule(), toModel(), valueOrEmpty(), receivableModel

### Community 68 - "4. Ordered Subtasks"
Cohesion: 0.11
Nodes (18): 1. Goal, 2. Scope, 3. Dependencies, 4. Ordered Subtasks, 5. Acceptance Scenarios, 6. Definition Of Done, A. Confirm account concurrency schema, B. Add the update domain command (+10 more)

### Community 69 - "VAT-Inclusive Tax Calculation Plan"
Cohesion: 0.11
Nodes (18): 10. Unit-Test Matrix, 11. Definition Of Done, 1. Goal, 2. Scope, 3. Terms, 4. Required Inputs, 5. Calculation Contract, 6. Rounding and Reconciliation (+10 more)

### Community 70 - "net/http.ResponseWriter"
Cohesion: 0.37
Nodes (3): net/http.ResponseWriter, requestID(), Handler

### Community 71 - "Query"
Cohesion: 0.19
Nodes (10): fakeCompanyTotalsRepository, aggregateRow, clientAggregateRow, GormRepository, salesLeaderRow, salesPeriodRow, boundaries(), Query (+2 more)

### Community 72 - "Duplicate PO Handling Plan"
Cohesion: 0.11
Nodes (17): 1. Goal, 2. Scope Assumption, 3. Create Behavior, 4. Edit Behavior, 5. Database Protection, 6. Ordered Subtasks, 7. Test Matrix, 8. Definition Of Done (+9 more)

### Community 73 - "Duplicate Invoice Number Validation Plan"
Cohesion: 0.11
Nodes (17): 1. Add domain error, 2. Extend the repository contract, 3. Enforce the check in both commands, 4. Map errors in the handler, 5. Add a database uniqueness migration, Current State, Domain/service tests, Duplicate Invoice Number Validation Plan (+9 more)

### Community 74 - "040 Reusable Text Date Input Plan"
Cohesion: 0.11
Nodes (17): 040 Reusable Text Date Input Plan, 1. Current implementation analysis, 2. MVP decisions, 3. Reusable component design, 4. Server-side contract, 5. Migration sequence, 6. Accessibility and responsive behavior, 7. Tests (+9 more)

### Community 75 - "043 Quotation Line Margin and Profit Plan"
Cohesion: 0.11
Nodes (17): 043 Quotation Line Margin and Profit Plan, Acceptance Criteria, Commission allocation, Cost inputs required from the user, Current State, Data Gaps and Decisions, Goal, Implementation Plan (+9 more)

### Community 76 - "Sales Order PDF Header and PO Layout Plan"
Cohesion: 0.11
Nodes (17): Current-State Analysis, Decisions Required Before Coding, Delivery Sequence, Dynamic header boundary, Goal, Header column widths, Implementation Changes, Layout Strategy (+9 more)

### Community 77 - "055 Supplier Bulk Product Configuration Plan"
Cohesion: 0.11
Nodes (17): 055 Supplier Bulk Product Configuration Plan, Acceptance scenarios, Audit and idempotency, Current-state analysis, Delivery sequence, Domain and application contract, Domain/service tests, Goal (+9 more)

### Community 78 - "Invoice Creation PDF New Tab Plan"
Cohesion: 0.11
Nodes (17): Browser acceptance tests, Current State, Definition of Done, Delivery Sequence, Form and Browser Behavior, Goal, Handler tests, HTTP Changes (+9 more)

### Community 79 - "Text"
Cohesion: 0.29
Nodes (5): Family(), MultiText(), RGB(), Text(), invoicePage

### Community 80 - "Line"
Cohesion: 0.25
Nodes (7): FormatAmount(), Line(), exportRow(), maxFloat(), ExportCompanyGroup, ExportRow, receivablesPDFPage

### Community 81 - "salesorders/handler_test.go"
Cohesion: 0.25
Nodes (17): Repository, NewHandler(), TestConvertedSalesOrderCanBeDuplicated(), TestCreateDuplicateSalesOrderUsesEditedPOAndQuantities(), TestCreateSalesOrderUsesFixedSalesPersonAndPO(), TestCustomerPOIsRequiredAndTrimmed(), TestDuplicateSalesOrderPageCopiesSourceContext(), testQuotation() (+9 more)

### Community 83 - "salesOrderPage"
Cohesion: 0.30
Nodes (4): maxFloat(), salesOrderPage, SalesOrderPDFDocument, SalesOrderPDFLine

### Community 84 - "html/template.Template"
Cohesion: 0.22
Nodes (9): html/template.Template, ValidateInvoiceNumber(), newIdempotencyKey(), parseID(), requestID(), safeFilename(), detailPage, Handler (+1 more)

### Community 85 - "Quotation"
Cohesion: 0.29
Nodes (3): Quotation, handlerRepository, quotationRepo

### Community 86 - "000-implementation-ladder.md"
Cohesion: 0.21
Nodes (7): Project Progress, Knowledge Base, Knowledge-base rule, Reading order, Project Analysis, Run with Docker Desktop, syncline-collection-tracking

### Community 87 - "Handler"
Cohesion: 0.34
Nodes (6): Options(), Service, NewHandler(), newKey(), requestID(), Handler

### Community 88 - "NewHandler"
Cohesion: 0.28
Nodes (15): Repository, NewHandler(), TestDirectPurchaseOrderUsesCanonicalEditFormAndUpdate(), TestNewDirectPurchaseOrderKeepsDirectProductGrid(), testPurchaseOrder(), TestPurchaseOrderCreateListsUnavailableSupplierProducts(), TestPurchaseOrderEditRejectsUnavailableSupplierProduct(), TestPurchaseOrderGridUsesInlineAction() (+7 more)

### Community 89 - "Parse"
Cohesion: 0.16
Nodes (14): Parse(), TestFormat(), TestFormatPHP(), TestParse(), TestParseRejectsInvalidAmounts(), assertAmount(), TestCalculateVATInclusive(), TestCalculateVATInclusiveRejectsInvalidRates() (+6 more)

### Community 90 - "GormRepository"
Cohesion: 0.26
Nodes (4): SupplierProduct, NewGormRepository(), GormRepository, Repository

### Community 91 - "Collection Tracking System: Technical and Architecture Plan"
Cohesion: 0.14
Nodes (14): 10. Configuration and `.env`, 13. Idempotent Commands and Concurrency, 14. Audit and Data Integrity, 15. Logging and Error Handling, 16. Security and Operational Baseline, 17. Development Workflow, 18. Technical Decisions, 1. Purpose (+6 more)

### Community 92 - "Database Schema and Index Plan"
Cohesion: 0.14
Nodes (14): 1. SQL Script Standards, 2. Table Contract, 3. Index Plan, 4. Migration Verification Queries, 5. References, Audit events, Company accounts, Database Schema and Index Plan (+6 more)

### Community 93 - "Reusable Feedback Snackbar Plan"
Cohesion: 0.14
Nodes (13): 10. Definition Of Done, 1. Goal, 2. Message Types, 3. Shared Template Location, 4. Alpine.js Behavior, 5. Layout Placement and Styling, 6. HTMX Integration, 7. Server Message Flow (+5 more)

### Community 94 - "Invoice Receivable Confirmation Plan"
Cohesion: 0.14
Nodes (13): Acceptance Scenario, Accessibility and Responsive Behavior, Goal, Invoice Receivable Confirmation Plan, Invoice summary, Receivable summary, Recommended Endpoint Design, Shared Summary Model (+5 more)

### Community 95 - "Dashboard Totals Implementation Plan"
Cohesion: 0.09
Nodes (21): 1. Goal, 2. Dashboard Metrics, 3. Aggregation Rules, 4. Slice Structure, 5. Ordered Subtasks, 6. Testing Subtasks, 7. Acceptance Scenarios, 8. Definition Of Done (+13 more)

### Community 96 - "Validation Sequence"
Cohesion: 0.15
Nodes (12): 045 PDF Print Failure Validation Plan, Acceptance Criteria, Current Validation Gap, Goal, Phase 1: Capture the Exact Download, Phase 2: Validate PDF Structure, Phase 3: Inspect Document Features, Phase 4: Test Renderer Edge Cases (+4 more)

### Community 98 - "pt"
Cohesion: 0.23
Nodes (13): ae(), be(), br(), Bt(), D(), fr(), Oe(), pt() (+5 more)

### Community 99 - "migrations.go"
Cohesion: 0.36
Nodes (11): applyMigration(), Down(), ensureRegistry(), find(), isApplied(), reverseMigration(), Status(), Up() (+3 more)

### Community 100 - "Sales Order Duplicate Process Plan"
Cohesion: 0.17
Nodes (11): Acceptance Criteria, Confirmed Decisions, Current-State Impact Analysis, Data and Application Impact, Delivery Sequence, Domain and Repository Plan, Goal, Recommended User Flow (+3 more)

### Community 101 - "salesorders/repository.go"
Cohesion: 0.12
Nodes (17): duplicateInputFromForm(), ValidateCustomerPO(), AllocationReader, DuplicateCreator, DuplicateLineSelection, DuplicateOrderInput, lineModel, NumberedSelectionCreator (+9 more)

### Community 102 - "5. Ordered Subtasks"
Cohesion: 0.18
Nodes (11): 5. Ordered Subtasks, A. Confirm database contract, B. Define the domain entity, C. Define application commands and queries, D. Define repository interfaces, E. Implement GORM models and adapter, F. Implement server routes, G. Implement templates with Tailwind (+3 more)

### Community 103 - "tax.go"
Cohesion: 0.47
Nodes (10): math/big.Int, amountFromBigInt(), applyRate(), calculateBase(), CalculateRule(), CalculateVATInclusive(), roundPositive(), validateRate() (+2 more)

### Community 104 - "InvoiceOption"
Cohesion: 0.26
Nodes (5): NewGormInvoiceRepository(), fakeInvoiceRepository, GormInvoiceRepository, InvoiceOption, InvoiceRepository

### Community 105 - "receivables/handler.go"
Cohesion: 0.27
Nodes (9): companyFilter(), defaultStatusOptions(), receivableLoadMoreURL(), statusFilter(), AccountOption, listPage, statusOption, MultiSelectOption (+1 more)

### Community 106 - "044 Branded Button Text Contrast Plan"
Cohesion: 0.18
Nodes (10): 044 Branded Button Text Contrast Plan, Acceptance Criteria, Affected Shared Design Surface, Current State, Goal, Implementation Plan, PBI 01: Fix the shared branded-button cascade, PBI 02: Normalize primary button markup (+2 more)

### Community 107 - "Config"
Cohesion: 0.25
Nodes (6): main(), run(), Config, Load(), Read(), Open()

### Community 108 - "GormRepository"
Cohesion: 0.19
Nodes (7): GeneratedSalesOrderNumber(), NewGormRepository(), sourceID(), sourceNumber(), ValidSalesOrderNumber(), GormRepository, LineSelection

### Community 109 - "applyFilters"
Cohesion: 0.35
Nodes (10): applyFilters(), escapeLike(), dryRunSQLServer(), TestApplyFiltersAddsCompanyAccountPredicate(), TestApplyFiltersAddsInvoicePredicate(), TestApplyFiltersDefaultsToActiveRecords(), TestApplyFiltersEmptyProvidedCompanyFilterReturnsNoRows(), TestApplyFiltersInvalidOnlyStatusReturnsNoRows() (+2 more)

### Community 110 - "NewCompanyAccount"
Cohesion: 0.33
Nodes (6): ValidationErrors, NewCompanyAccount(), TestNewCompanyAccountRejectsTooLongValue(), TestNewCompanyAccountReportsRequiredFields(), TestNewCompanyAccountTrimsAndValidates(), validateRequired()

### Community 111 - "Migration Implementation Plan"
Cohesion: 0.22
Nodes (9): 1. Requested Direction, 2. Important Compatibility Detail, 3. No-Argument Migration Flow, 4. Configuration Contract, 5. Database and Testing Consequences, 6. Security Risk Requiring Explicit Acceptance, 7. Implementation Tasks, 8. Approval Questions (+1 more)

### Community 112 - "run"
Cohesion: 0.28
Nodes (8): main(), requestTimeouts(), run(), net/http.Handler, ProductOptions, SupplierOptions, CustomerOptions, ProductOptions

### Community 113 - "Confirmed Contract"
Cohesion: 0.22
Nodes (8): 041 Shared Quotation and Sales Order PDF Decisions, Confirmed Contract, Defaults and Rules, Implementation Consequence, Remaining Information Gaps, Sales Order Fields, Sales Order Status, Totals

### Community 114 - "accounts/handler.go"
Cohesion: 0.29
Nodes (4): AccountFormViewModel, formPage, formatTime(), ValidationErrors

### Community 115 - "NewHandler"
Cohesion: 0.38
Nodes (9): Repository, NewHandler(), TestConversionFormIsReadOnly(), TestConvertedSalesOrderCannotOpenInvoiceForm(), TestCreateInvoiceRedirectsToInvoiceDetail(), TestInvoiceDetailShowsTotalComputation(), TestInvoicePDFReturnsInlinePDF(), testOrder() (+1 more)

### Community 116 - "5. Domain Model Mapping"
Cohesion: 0.25
Nodes (8): 5. Domain Model Mapping, Company account, Dashboard aggregation and pagination, Dashboard user experience, Delivery receivable, Derived classifications, Money representation and display, Payment representation

### Community 117 - "SalesOrder"
Cohesion: 0.31
Nodes (4): SalesOrder, orderRepo, testOrders, orderRepo

### Community 118 - "supplierModel"
Cohesion: 0.25
Nodes (3): SupplierProduct, supplierModel, supplierProductModel

### Community 119 - "Epic E-03: Quote-to-Price"
Cohesion: 0.29
Nodes (6): 032 RFQ and Quotation Pricing Plan, Epic E-03: Quote-to-Price, PBI 01: RFQ capture, PBI 02: Quotation and quotation lines, PBI 03: Commission calculation, PBI 04: Quotation lifecycle

### Community 120 - "Epic E-08: Reporting, Reliability, and Release"
Cohesion: 0.29
Nodes (6): 037 Trading MVP Release Plan, Epic E-08: Reporting, Reliability, and Release, PBI 01: MVP operational reporting, PBI 02: Cross-slice audit and concurrency hardening, PBI 03: Authentication and access boundary, PBI 04: End-to-end MVP verification

### Community 121 - "dashboard/queries.go"
Cohesion: 0.29
Nodes (9): CompanyTotalsRepository, QueryRepository, SalesQueryRepository, SalesService, Service, CompanyTotalsService, NewCompanyTotalsService(), NewSalesService() (+1 more)

### Community 122 - "AccountViewModel"
Cohesion: 0.40
Nodes (3): AccountViewModel, listPage, commandService

### Community 123 - "Implementation Ladder"
Cohesion: 0.33
Nodes (6): AI State Contract, Current State, How To Read This File, Implementation Ladder, Ladder, Recording New Work

### Community 124 - "6. Vertical Slice Boundaries"
Cohesion: 0.33
Nodes (6): 6. Vertical Slice Boundaries, Adapters inside a slice, Commands and queries inside a slice, Composition root, Cross-slice behavior, Domain rules inside a slice

### Community 125 - "Trading MVP Implementation Ladder"
Cohesion: 0.33
Nodes (5): Current Repository Baseline, Development Rule, Ladder, MVP Boundary, Trading MVP Implementation Ladder

### Community 126 - "Epic E-01: Trading Domain Foundation"
Cohesion: 0.33
Nodes (5): 030 Trading Foundation Plan, Epic E-01: Trading Domain Foundation, PBI 01: Approve the trading MVP contract, PBI 02: Define transaction snapshots and immutability, PBI 03: Define the cross-slice integration boundary

### Community 127 - "Epic E-04: Customer Commitment"
Cohesion: 0.33
Nodes (5): 033 Customer PO and Sales Order Plan, Epic E-04: Customer Commitment, PBI 01: Customer PO capture, PBI 02: Sales Order creation, PBI 03: Sales Order status and changes

### Community 128 - "Epic E-05: Multi-Supplier Procurement"
Cohesion: 0.33
Nodes (5): 034 Procurement and Allocation Plan, Epic E-05: Multi-Supplier Procurement, PBI 01: Supplier PO creation, PBI 02: Sales-line to supplier-line allocation, PBI 03: Procurement status tracking

### Community 129 - "Epic E-06: Fulfillment and Invoicing"
Cohesion: 0.33
Nodes (5): 035 Receiving and Invoice Plan, Epic E-06: Fulfillment and Invoicing, PBI 01: Full receiving, PBI 02: Actual profitability, PBI 03: Invoice creation from Sales Order

### Community 130 - "Epic E-07: Collection Integration"
Cohesion: 0.33
Nodes (5): 036 Receivable Handoff Plan, Epic E-07: Collection Integration, PBI 01: Create delivery receivable from invoice, PBI 02: Existing collection workflow regression, PBI 03: Trading-to-collection navigation

### Community 131 - "11. Environment Separation"
Cohesion: 0.40
Nodes (5): 11. Environment Separation, Development, Local authentication, Production, Test

### Community 132 - "12. Testing Strategy and TDD"
Cohesion: 0.40
Nodes (5): 12. Testing Strategy and TDD, HTTP tests, Integration tests, Repository tests, Unit tests

### Community 133 - "19. Technical Gap Analysis"
Cohesion: 0.40
Nodes (5): 19. Technical Gap Analysis, Critical gaps, Default validation and data-integrity baseline, Important gaps, Recommended pre-implementation decisions

### Community 134 - "Architecture Graph"
Cohesion: 0.40
Nodes (4): Architecture Graph, Conventions, Package Dependencies, Runtime Flow

### Community 135 - "20. Development Blockers"
Cohesion: 0.50
Nodes (4): 20. Development Blockers, Blockers before production-ready MVP, Blockers before the first runnable vertical slice, Important but not development blockers

### Community 136 - "NewHandler"
Cohesion: 0.22
Nodes (7): fakeRepository, page, salesPage, Service, NewHandler(), TestDashboardRendersSummaryCards(), TestDashboardSummaryReturnsFragment()

### Community 137 - "models_test.go"
Cohesion: 0.50
Nodes (3): TestReceivableModelMapsInvoiceNumberValue(), TestReceivableModelMapsMissingInvoiceAsNull(), TestReceivableModelMapsNullInvoiceNumberAsEmpty()

### Community 138 - "NewService"
Cohesion: 0.39
Nodes (8): commandService, Repository, NewService(), NewHandler(), TestEditFormRendersCompanyReceivablesSummary(), TestEditFormRendersCurrentAccount(), TestInvalidCreateReturnsHTMXValidationFragment(), TestNewAccountFormRenders()

### Community 141 - "Pe"
Cohesion: 0.43
Nodes (7): De(), k(), me(), Mt(), Pe(), xe(), ye()

## Knowledge Gaps
- **1004 isolated node(s):** `github.com/psbernardo/syncline-collection-tracking`, `migrationRow`, `Repository`, `aggregateRow`, `clientAggregateRow` (+999 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **12 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Amount` connect `Amount` to `pdfPage`, `Supplier`, `net/http.Request`, `Handler`, `context.Context`, `Payment`, `.Render`, `purchaseorders/pdf.go`, `PurchaseOrder`, `quotations/repository.go`, `export.go`, `dashboard/domain.go`, `Invoice`, `helpers.go`, `toModel`, `Line`, `salesOrderPage`, `Quotation`, `NewHandler`, `Parse`, `salesorders/repository.go`, `tax.go`, `GormRepository`, `NewHandler`, `supplierModel`?**
  _High betweenness centrality (0.039) - this node is a cross-community bridge._
- **Why does `Quotation` connect `Quotation` to `pdfPage`, `migrations.go`, `Supplier`, `quotations/repository.go`, `Handler`, `GormRepository`, `Amount`, `Line`, `salesorders/handler_test.go`, `SalesOrder`, `time.Time`?**
  _High betweenness centrality (0.012) - this node is a cross-community bridge._
- **Why does `run()` connect `run` to `pdfPage`, `Supplier`, `gorm.io/gorm.DB`, `Product`, `NewHandler`, `NewService`, `context.Context`, `Payment`, `testing.T`, `CompanyAccount`, `PurchaseOrder`, `quotations/repository.go`, `Handler`, `Invoice`, `Query`, `salesorders/handler_test.go`, `Handler`, `NewHandler`, `GormRepository`, `InvoiceOption`, `Config`, `GormRepository`, `NewHandler`, `dashboard/queries.go`?**
  _High betweenness centrality (0.009) - this node is a cross-community bridge._
- **What connects `github.com/psbernardo/syncline-collection-tracking`, `migrationRow`, `Repository` to the rest of the system?**
  _1004 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `alpine.min.js` be split into smaller, more focused modules?**
  _Cohesion score 0.029277739804055593 - nodes in this community are weakly interconnected._
- **Should `htmx.min.js` be split into smaller, more focused modules?**
  _Cohesion score 0.09740259740259741 - nodes in this community are weakly interconnected._
- **Should `pdfPage` be split into smaller, more focused modules?**
  _Cohesion score 0.05775638652350981 - nodes in this community are weakly interconnected._