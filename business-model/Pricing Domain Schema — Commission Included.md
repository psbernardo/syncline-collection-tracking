# Pricing Domain Schema — Commission Included

## 1. Business Context

The business operates as a **B2B buy-to-order trading and multi-supplier procurement model**.

The company:

1. Receives an RFQ from a customer.
2. Confirms product prices with suppliers.
3. Calculates the total acquisition cost.
4. Includes delivery costs and agent commissions.
5. Calculates the required profit/margin.
6. Sends a quotation to the customer.
7. Receives the customer's Purchase Order.
8. Converts the agreed order into a Sales Order.
9. Creates one or more Supplier Purchase Orders.
10. Delivers the products to the customer.
11. Invoices the customer.
12. Collects payment based on the agreed payment terms.

---

# 2. Current Pricing Scope

The pricing model intentionally remains simple.

### Supported

- PHP currency only
- Supplier product prices
- Supplier-specific product pricing
- Manual supplier price updates
- Supplier delivery cost
- Customer delivery cost
- Other applicable costs
- Agent commission
- Profit calculation
- Margin calculation
- Selling price calculation
- Quotation pricing
- Estimated profitability
- Actual profitability

### Not Required for the Current Version

- Multi-currency / FX
- Supplier price validity periods
- Supplier availability tracking
- Automated supplier price confirmation
- Discount engine
- Negotiation history
- Financing cost
- Customer-specific price lists
- Complex pricing rules
- Price versioning
- Automated supplier selection

Supplier confirmation and negotiation remain manual business processes. The system records the resulting agreed supplier price.

---

# 3. Commission Business Rule

The business may have a commission associated with a specific product.

Example:

```text
Product: ABC
Customer Price: ₱502 per box
Quantity: 100 boxes
Agent Commission: ₱2 per box
```

Calculation:

```text
Gross Customer Amount
= ₱502 × 100
= ₱50,200
```

Agent commission:

```text
Commission
= ₱2 × 100
= ₱200
```

Net sales:

```text
Net Sales
= ₱50,200 - ₱200
= ₱50,000
```

Therefore:

```text
Customer Invoice / Gross Sales    ₱50,200
Agent Commission                  ₱200
Net Sales                         ₱50,000
```

The **₱200 commission is a selling-related cost** and should be tracked separately from the supplier/product cost.

---

# 4. Pricing Flow

```text
                         PRODUCT
                            │
                            ▼
                   SUPPLIER_PRODUCT
                            │
                            ▼
                    SUPPLIER_PRICE
                            │
                            ▼
                  PRICING CALCULATION
                            │
              ┌─────────────┼─────────────┐
              │             │             │
              ▼             ▼             ▼
       Supplier Cost   Delivery Costs   Other Costs
              │             │             │
              └─────────────┼─────────────┘
                            ▼
                       TOTAL COST
                            │
                            ▼
                    PROFIT / MARGIN
                            │
                            ▼
                    NET SALES PRICE
                            │
                            ▼
                     + COMMISSION
                            │
                            ▼
                 CUSTOMER SELLING PRICE
                            │
                            ▼
                       QUOTATION
                            │
                            ▼
                    CUSTOMER PO
                            │
                            ▼
                      SALES ORDER
                            │
              ┌─────────────┼─────────────┐
              ▼             ▼             ▼
        SUPPLIER PO A  SUPPLIER PO B  SUPPLIER PO C
```

---

# 5. Financial Model

The pricing model should distinguish between:

- Supplier/product cost
- Delivery costs
- Other costs
- Agent commission
- Net sales
- Customer invoice amount
- Profit

The core relationship is:

```text
Customer Invoice Amount
    - Agent Commission
    = Net Sales

Net Sales
    - Supplier Product Cost
    - Supplier Delivery Cost
    - Customer Delivery Cost
    - Other Cost
    = Profit
```

---

# 6. Product

The `product` table contains product master data.

Pricing should not be stored directly in the product table because the same product can be supplied by multiple suppliers at different prices.

```sql
CREATE TABLE product (
    product_id      BIGINT PRIMARY KEY,
    sku             VARCHAR(50) NOT NULL UNIQUE,
    name            VARCHAR(255) NOT NULL,
    description     TEXT,
    unit_of_measure VARCHAR(20) NOT NULL,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMP NOT NULL,
    updated_at      TIMESTAMP NOT NULL
);
```

Example:

| Product ID | SKU | Name | UOM |
|---:|---|---|---|
| 1001 | LAP-001 | Dell Laptop | PC |
| 1002 | MON-001 | Dell Monitor | PC |

---

# 7. Supplier Product

A product can be supplied by multiple suppliers.

```sql
CREATE TABLE supplier_product (
    supplier_product_id BIGINT PRIMARY KEY,
    supplier_id         BIGINT NOT NULL,
    product_id          BIGINT NOT NULL,
    supplier_sku        VARCHAR(100),

    is_active            BOOLEAN NOT NULL DEFAULT TRUE,

    UNIQUE (supplier_id, product_id)
);
```

Example:

```text
Laptop
│
├── Supplier A
├── Supplier B
└── Supplier C
```

This relationship is important because one Sales Order can require products from multiple suppliers.

---

# 8. Supplier Price

The current implementation does not require price validity or price versioning.

The system stores the current supplier price and allows it to be manually updated when the supplier provides a new price.

```sql
CREATE TABLE supplier_price (
    supplier_price_id   BIGINT PRIMARY KEY,
    supplier_product_id BIGINT NOT NULL,

    unit_cost           DECIMAL(18,2) NOT NULL,

    updated_at           TIMESTAMP NOT NULL
);
```

Example:

| Supplier | Product | Current Cost |
|---|---|---:|
| Supplier A | Laptop | ₱50,000 |
| Supplier B | Laptop | ₱48,000 |
| Supplier C | Laptop | ₱51,000 |

If Supplier A calls and provides a new price:

```text
Supplier A
Laptop

Old Price: ₱50,000
New Price: ₱48,500
```

The current supplier price can simply be updated.

---

# 9. Pricing Calculation

The pricing calculation is the core of the pricing domain.

It considers:

- Supplier product cost
- Supplier delivery cost
- Customer delivery cost
- Other costs
- Agent commission
- Net sales
- Customer invoice amount
- Profit
- Margin

```sql
CREATE TABLE pricing_calculation (
    pricing_calculation_id BIGINT PRIMARY KEY,

    quotation_line_id      BIGINT NOT NULL,

    supplier_product_cost  DECIMAL(18,2) NOT NULL DEFAULT 0,

    supplier_delivery_cost DECIMAL(18,2) NOT NULL DEFAULT 0,

    customer_delivery_cost DECIMAL(18,2) NOT NULL DEFAULT 0,

    other_cost             DECIMAL(18,2) NOT NULL DEFAULT 0,

    total_cost             DECIMAL(18,2) NOT NULL,

    target_profit          DECIMAL(18,2) NOT NULL DEFAULT 0,

    target_margin_percent  DECIMAL(9,4),

    net_sales_amount       DECIMAL(18,2) NOT NULL,

    commission_amount      DECIMAL(18,2) NOT NULL DEFAULT 0,

    customer_amount        DECIMAL(18,2) NOT NULL,

    created_at             TIMESTAMP NOT NULL
);
```

---

# 10. Cost Calculation

The basic acquisition and delivery cost is:

```text
Supplier Product Cost
+ Supplier Delivery Cost
+ Customer Delivery Cost
+ Other Cost
--------------------------------
Total Cost
```

Example:

```text
Supplier Product Cost       ₱50,000
Supplier Delivery            ₱1,000
Customer Delivery              ₱500
Other Cost                     ₱200
------------------------------------
Total Cost                   ₱51,700
```

---

# 11. Agent Commission

Commission should be modeled separately from the product cost.

```sql
CREATE TABLE pricing_commission (
    pricing_commission_id  BIGINT PRIMARY KEY,

    pricing_calculation_id BIGINT NOT NULL,

    agent_id               BIGINT NOT NULL,

    commission_type        VARCHAR(30) NOT NULL,

    commission_rate        DECIMAL(18,2) NOT NULL,

    commission_quantity    DECIMAL(18,2) NOT NULL,

    commission_amount      DECIMAL(18,2) NOT NULL,

    created_at             TIMESTAMP NOT NULL
);
```

For the current business requirement:

```text
commission_type = PER_UNIT
commission_rate = ₱2.00
commission_quantity = 100
commission_amount = ₱200.00
```

---

# 12. Commission Calculation

For a per-unit commission:

```text
Commission Amount
= Commission Rate × Quantity
```

Example:

```text
Commission Rate = ₱2 / box
Quantity        = 100 boxes

Commission
= ₱2 × 100
= ₱200
```

---

# 13. Commission Types

The current requirement is **per-unit commission**.

However, the schema can support other commission types without requiring a major redesign.

```text
PER_UNIT
PERCENTAGE
FIXED_AMOUNT
```

### Per Unit

```text
₱2 × 100 boxes
= ₱200
```

### Percentage

```text
2% × ₱50,000
= ₱1,000
```

### Fixed Amount

```text
Commission = ₱500
```

Only `PER_UNIT` needs to be implemented initially if that is the only current business requirement.

---

# 14. Gross Sales vs Net Sales

The system should distinguish between **Gross Customer Amount** and **Net Sales**.

Example:

```text
Customer Price = ₱502 / box
Quantity       = 100 boxes
```

Gross customer amount:

```text
₱502 × 100
= ₱50,200
```

Commission:

```text
₱2 × 100
= ₱200
```

Net sales:

```text
₱50,200 - ₱200
= ₱50,000
```

Therefore:

```text
Gross Sales / Customer Amount    ₱50,200
Less: Agent Commission             ₱200
-----------------------------------------
Net Sales                         ₱50,000
```

---

# 15. Profit Calculation

Profit should be calculated using **Net Sales**, not the gross customer amount.

```text
Net Sales
- Supplier Product Cost
- Supplier Delivery Cost
- Customer Delivery Cost
- Other Cost
--------------------------------
Profit
```

Example:

```text
Net Sales                   ₱50,000
Supplier Product Cost      -₱40,000
Supplier Delivery            -₱2,000
Customer Delivery            -₱1,000
Other Cost                     -₱500
------------------------------------
Profit                       ₱6,500
```

The commission has already been deducted when calculating Net Sales.

---

# 16. Margin Calculation

Margin should also use Net Sales.

```text
Margin %
= Profit / Net Sales × 100
```

Example:

```text
Profit       = ₱6,500
Net Sales    = ₱50,000

Margin
= ₱6,500 / ₱50,000 × 100

= 13%
```

This prevents the agent commission from artificially inflating the reported margin.

---

# 17. Quotation Line

The quotation line stores the actual customer-facing price.

```sql
CREATE TABLE quotation_line (
    quotation_line_id BIGINT PRIMARY KEY,

    quotation_id      BIGINT NOT NULL,
    product_id        BIGINT NOT NULL,

    quantity          DECIMAL(18,2) NOT NULL,

    unit_price        DECIMAL(18,2) NOT NULL,
    line_total        DECIMAL(18,2) NOT NULL,

    created_at        TIMESTAMP NOT NULL
);
```

For the example:

```text
Product: ABC
Quantity: 100
Unit Price: ₱502

Line Total:
100 × ₱502
= ₱50,200
```

---

# 18. Calculated Price vs Quoted Price

The calculated selling price and the actual quotation price should remain separate.

Example:

```text
Total Cost                ₱43,500
Target Profit              ₱6,500
--------------------------------
Calculated Net Sales      ₱50,000

Commission                   ₱200
--------------------------------
Customer Price             ₱50,200
```

The quotation stores:

```text
Unit Price = ₱502
```

The pricing calculation stores the underlying financial calculation.

This separation allows the business to understand **why the customer price was set at ₱502**.

---

# 19. Sales Order Pricing

The customer's Purchase Order may change the original quotation.

Example:

```text
Quotation
----------------
ABC        100 boxes
Price      ₱502

Customer PO
----------------
ABC         80 boxes
Price      ₱502
```

The Sales Order represents the final agreed commercial transaction.

```sql
CREATE TABLE sales_order_line (
    sales_order_line_id BIGINT PRIMARY KEY,

    sales_order_id      BIGINT NOT NULL,
    product_id          BIGINT NOT NULL,

    quantity            DECIMAL(18,2) NOT NULL,

    unit_price          DECIMAL(18,2) NOT NULL,
    line_total          DECIMAL(18,2) NOT NULL,

    created_at          TIMESTAMP NOT NULL
);
```

The Sales Order must retain its own transaction price.

---

# 20. Multi-Supplier Procurement

One Sales Order can generate multiple Supplier Purchase Orders.

```text
Sales Order SO-1001
│
├── Laptop
│      └── PO-1001 → Supplier A
│
├── Monitor
│      └── PO-1002 → Supplier B
│
└── Printer
       └── PO-1003 → Supplier C
```

Therefore:

```text
1 Sales Order
       │
       ├── Supplier PO A
       ├── Supplier PO B
       └── Supplier PO C
```

---

# 21. Estimated vs Actual Profitability

This distinction should remain in the model.

## At quotation time

```text
Estimated Supplier Cost       ₱40,000
Estimated Supplier Delivery    ₱2,000
Estimated Customer Delivery    ₱1,000
Other Cost                       ₱500
Commission                       ₱200
--------------------------------------
Estimated Total Cost          ₱43,700

Net Sales                     ₱50,000
Estimated Profit               ₱6,300
```

## After fulfillment

Actual costs may be:

```text
Actual Supplier Cost          ₱40,500
Actual Supplier Delivery       ₱2,500
Actual Customer Delivery       ₱1,200
Actual Other Cost                 ₱0
Commission                       ₱200
--------------------------------------
Actual Total Cost             ₱44,400

Net Sales                     ₱50,000
Actual Profit                  ₱5,600
```

Variance:

```text
Estimated Profit    ₱6,300
Actual Profit       ₱5,600
---------------------------
Variance             -₱700
```

This allows the business to determine whether actual delivery and procurement costs are reducing the expected margin.

---

# 22. Recommended Domain Model

```text
PRODUCT
   │
   └── SUPPLIER_PRODUCT
          │
          └── SUPPLIER_PRICE


QUOTATION
   │
   └── QUOTATION_LINE
          │
          └── PRICING_CALCULATION
                  │
                  ├── Supplier Product Cost
                  ├── Supplier Delivery Cost
                  ├── Customer Delivery Cost
                  ├── Other Cost
                  │
                  ├── PRICING_COMMISSION
                  │      ├── Agent
                  │      ├── Commission Type
                  │      ├── Commission Rate
                  │      ├── Quantity
                  │      └── Commission Amount
                  │
                  ├── Total Cost
                  ├── Target Profit
                  ├── Net Sales
                  └── Customer Amount


CUSTOMER PO
   │
   ▼
SALES ORDER
   │
   ├── SUPPLIER PO A
   ├── SUPPLIER PO B
   └── SUPPLIER PO C
```

---

# 23. Recommended Tables

The current MVP pricing domain consists of:

```text
product
supplier_product
supplier_price

pricing_calculation
pricing_commission

quotation
quotation_line

sales_order
sales_order_line

purchase_order
purchase_order_line
```

---

# 24. Core Financial Relationship

The most important financial relationship in the system is:

```text
Customer Amount
    - Agent Commission
    = Net Sales

Net Sales
    - Supplier Product Cost
    - Supplier Delivery Cost
    - Customer Delivery Cost
    - Other Cost
    = Profit
```

Example:

```text
Customer Amount                 ₱50,200
Agent Commission                  -₱200
---------------------------------------
Net Sales                       ₱50,000

Supplier Product Cost          -₱40,000
Supplier Delivery               -₱2,000
Customer Delivery               -₱1,000
Other Cost                        -₱500
---------------------------------------
Profit                           ₱6,500

Margin                             13%
```

---

# 25. Final Pricing Architecture

```text
                  PRODUCT MASTER
                       │
                       ▼
              SUPPLIER PRODUCT
                       │
                       ▼
               CURRENT SUPPLIER
                     PRICE
                       │
                       ▼
              PRICING CALCULATION
                       │
             ┌─────────┼─────────┐
             │         │         │
             ▼         ▼         ▼
          Product   Supplier   Customer
           Cost     Delivery   Delivery
             │         │         │
             └─────────┼─────────┘
                       │
                       ▼
                  OTHER COST
                       │
                       ▼
               AGENT COMMISSION
                       │
                       ▼
                  TOTAL COST
                       │
                       ▼
                 TARGET PROFIT
                       │
                       ▼
                   NET SALES
                       │
                       ▼
                 + COMMISSION
                       │
                       ▼
             CUSTOMER AMOUNT
                       │
                       ▼
                  QUOTATION
                       │
                       ▼
                CUSTOMER PO
                       │
                       ▼
                  SALES ORDER
                       │
              ┌────────┼────────┐
              ▼        ▼        ▼
          SUPPLIER A SUPPLIER B SUPPLIER C
              │        │        │
              ▼        ▼        ▼
            PO A      PO B      PO C
```

## Core Design Principle

The system must distinguish between:

### Supplier/reference pricing

```text
supplier_price.unit_cost
```

### Transaction pricing

```text
quotation_line.unit_price
sales_order_line.unit_price
purchase_order_line.unit_cost
```

### Commission

```text
pricing_commission
```

### Profitability

```text
Net Sales
- Actual Costs
= Actual Profit
```

The commission is **not part of the supplier product cost**. It is a separate selling-related cost that is deducted from the gross customer amount to determine **Net Sales**.

For the current business requirement, the commission can primarily be implemented as **per-unit commission**, e.g.:

```text
₱2 commission / box × 100 boxes = ₱200 commission
```

while keeping the schema extensible for future commission types.