# B2B Trading and Procurement Business Model

## Business Model Classification

**Primary Business Model:**  
B2B Trading / Procurement and Distribution

**Specific Model:**  
Buy-to-Order B2B Reselling with Multi-Supplier Procurement

**Business Process Model:**  
Quote-to-Cash + Procure-to-Pay

---

## Business Overview

The business operates as a **B2B sourcing, trading, and procurement intermediary**.

The company receives a request for quotation (RFQ) from a client, sources and negotiates supplier prices, calculates the required selling price and profit margin, provides a quotation to the client, and upon receiving the customer's Purchase Order (PO), creates a Sales Order and one or more Purchase Orders for different suppliers.

The company earns revenue through the **margin between the customer's selling price and the total cost of acquiring and fulfilling the order**.

The business primarily operates on a **buy-to-order model**, meaning supplier purchases are triggered by actual customer demand rather than maintaining large amounts of inventory.

---

# End-to-End Business Flow

```text
CLIENT
  │
  │ 1. Request for Quotation (RFQ)
  ▼
YOUR COMPANY
  │
  ├── 2. Source Supplier Prices
  │       ├── Supplier A
  │       ├── Supplier B
  │       └── Supplier C
  │
  ├── 3. Negotiate Supplier / Customer Pricing
  │
  ├── 4. Calculate Cost, Expenses, Margin & Profit
  │
  ▼
QUOTATION TO CLIENT
  │
  │ Payment Terms: 7 / 15 / 30 / 45 Days
  │
  ▼
CUSTOMER PURCHASE ORDER
  │
  │ 5. Validate agreed items and prices
  │
  ▼
SALES ORDER
  │
  ├──────────────┬──────────────┐
  ▼              ▼              ▼
PO Supplier A  PO Supplier B  PO Supplier C
  │              │              │
  ▼              ▼              ▼
Supplier A     Supplier B     Supplier C
  │              │              │
  └──────────────┴──────────────┘
                 │
                 ▼
          Delivery to Client
                 │
                 ▼
              INVOICE
                 │
                 │ 7 / 15 / 30 / 45 Days
                 ▼
        ACCOUNTS RECEIVABLE
                 │
                 ▼
             COLLECTION
```

---

# 1. Client Request for Quotation

The client submits an **RFQ (Request for Quotation)** containing a list of requested items.

Example:

| Item | Quantity |
|---|---:|
| Laptop | 10 |
| Monitor | 20 |
| Printer | 5 |

The RFQ becomes the starting point of the commercial process.

---

# 2. Supplier Price Sourcing

The company checks prices from one or more suppliers.

A single client RFQ may require multiple suppliers.

Example:

| Item | Supplier | Supplier Cost |
|---|---|---:|
| Laptop | Supplier A | ₱50,000 |
| Monitor | Supplier B | ₱8,000 |
| Printer | Supplier C | ₱15,000 |

Supplier pricing may involve:

- Price negotiation
- Supplier discounts
- Volume discounts
- Payment terms
- Availability
- Lead time
- Delivery charges
- Supplier-specific conditions

---

# 3. Price Negotiation

Price negotiation happens between the company, suppliers, and the customer.

The company needs to determine a selling price that provides sufficient margin and covers the expected expenses.

### Pricing calculation

```text
Supplier Cost
+ Shipping / Freight
+ Delivery Cost
+ Handling Cost
+ Financing Cost
+ Other Expenses
+ Desired Profit
--------------------------------
= Customer Selling Price
```

Example:

```text
Supplier Cost       ₱50,000
Freight              ₱2,000
Other Expenses       ₱1,000
Desired Profit       ₱7,000
--------------------------------
Selling Price       ₱60,000
```

### Margin

```text
Gross Profit = Selling Price - Total Cost

Gross Profit = ₱60,000 - ₱53,000
             = ₱7,000
```

```text
Gross Margin = Gross Profit / Selling Price × 100

Gross Margin = ₱7,000 / ₱60,000 × 100
             = 11.67%
```

---

# 4. Customer Quotation

After supplier pricing and internal profitability calculations, the company sends a **Quotation** to the client.

The quotation may contain:

- Item
- Quantity
- Unit price
- Total price
- Delivery terms
- Payment terms
- Validity period
- Lead time
- Warranty
- Commercial conditions

Example payment terms:

```text
Payment Terms:
7 Days
15 Days
30 Days
45 Days
```

---

# 5. Customer Purchase Order

If the customer accepts the quotation, they send a **Purchase Order (PO)**.

The customer's PO becomes the customer's formal commitment to purchase.

However, the final order may differ from the original quotation.

For example:

```text
Quotation
──────────
Laptop       10
Monitor      20
Printer       5

Customer PO
──────────
Laptop        8
Monitor      20
Printer       3
```

Therefore, the system should not assume:

```text
Quotation = Customer PO
```

Instead:

```text
Quotation
    ↓
Customer PO
    ↓
Sales Order
```

The Sales Order represents the **final agreed commercial transaction**.

---

# 6. Sales Order

The customer's PO is converted into a **Sales Order**.

The Sales Order contains the final agreed:

- Items
- Quantities
- Selling prices
- Discounts
- Payment terms
- Delivery terms
- Customer information
- Expected delivery date

The Sales Order is the central transaction connecting the **sales side** and **procurement side**.

---

# 7. One Sales Order → Multiple Supplier Purchase Orders

One of the key characteristics of this business model is that a single Sales Order can generate multiple Supplier POs.

Example:

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
      ├── Supplier PO #1
      ├── Supplier PO #2
      └── Supplier PO #3
```

This is a **multi-supplier fulfillment model**.

---

# 8. Supplier Purchase Order

The company creates Purchase Orders for suppliers based on the Sales Order.

Supplier payment conditions may differ.

Example:

| Supplier | Payment Term |
|---|---|
| Supplier A | Cash |
| Supplier B | 15 Days |
| Supplier C | 30 Days |

The company must track:

- Supplier price
- Quantity
- Payment terms
- Delivery date
- Supplier confirmation
- Supplier invoice
- Supplier payment
- Outstanding payable

---

# 9. Supplier Delivery

Suppliers deliver the ordered items.

Depending on the business process, the goods may:

```text
Supplier
   ↓
Your Warehouse
   ↓
Customer
```

or:

```text
Supplier
   ↓
Customer
```

The second approach is effectively a **direct-ship / drop-ship fulfillment pattern**, if the supplier ships directly to the customer on your company's behalf.

---

# 10. Customer Delivery and Invoice

After the items are delivered to the customer, the company issues an invoice.

Example:

```text
Delivery Date: August 16
Payment Terms: 30 Days

Invoice Date: August 16
Due Date: September 15
```

The payment term determines when the customer's receivable becomes due.

---

# 11. Accounts Receivable and Collection

The company tracks the customer's outstanding balance.

Example:

| Invoice | Amount | Terms | Due Date | Status |
|---|---:|---|---|---|
| INV-1001 | ₱500,000 | 30 Days | Sep 15 | Outstanding |
| INV-1002 | ₱250,000 | 15 Days | Aug 31 | Outstanding |
| INV-1003 | ₱100,000 | 7 Days | Aug 23 | Paid |

The collection process includes:

```text
Invoice
   ↓
Due Date Monitoring
   ↓
Collection Reminder
   ↓
Customer Payment
   ↓
Payment Recording
   ↓
Invoice Reconciliation
   ↓
Invoice Closed
```

---

# Financial Model

The business has two major financial flows.

## Customer Side

```text
Customer Invoice
        ↓
Accounts Receivable
        ↓
Customer Payment
```

## Supplier Side

```text
Supplier Invoice
        ↓
Accounts Payable
        ↓
Supplier Payment
```

The company makes money from the difference between these transactions.

```text
Customer Revenue
       -
Supplier Cost
       -
Operating Expenses
       -
Financing Cost
       =
Net Profit
```

---

# Working Capital Consideration

The payment terms between customers and suppliers can create a working-capital requirement.

Example:

```text
Supplier Terms: Cash
Customer Terms: 45 Days
```

The company may need to pay the supplier immediately but wait 45 days before receiving customer payment.

Example:

```text
Day 0
Customer PO
   ↓
Day 2
Supplier requires ₱500,000
   ↓
Company pays supplier
   ↓
Day 5
Customer receives goods
   ↓
45-day payment term starts
   ↓
Day 50
Customer pays ₱600,000
```

In this example, the company finances approximately **₱500,000 for the period between supplier payment and customer collection**.

Therefore, the business needs to manage:

- Working capital
- Accounts receivable
- Accounts payable
- Customer credit limits
- Supplier payment terms
- Cash flow
- Financing costs
- Collection efficiency

---

# Business Model Summary

| Area | Model |
|---|---|
| Industry Model | B2B Trading / Procurement |
| Sales Model | B2B Reselling |
| Fulfillment Model | Buy-to-Order |
| Procurement Model | Multi-Supplier Sourcing |
| Inventory Model | Low/No Stock or Order-Driven Inventory |
| Pricing Model | Cost + Margin |
| Customer Payment | Credit / Trade Terms |
| Supplier Payment | Cash or Credit Terms |
| Revenue | Product/Service Resale |
| Profit | Selling Price − Total Cost |
| Customer Process | Quote-to-Cash |
| Supplier Process | Procure-to-Pay |
| Key Transaction | Sales Order |
| Key Relationship | 1 Sales Order → Multiple Supplier POs |

---

# Recommended Business Model Name

For documentation, I would use:

> **B2B Buy-to-Order Trading and Multi-Supplier Procurement Model**

For an ERP/software project:

> **B2B Quote-to-Cash and Procure-to-Pay Trading System**

The defining characteristic is:

```text
                CUSTOMER
                   │
                  RFQ
                   ↓
              QUOTATION
                   ↓
              CUSTOMER PO
                   ↓
              SALES ORDER
                   │
          ┌────────┼────────┐
          ↓        ↓        ↓
       SUPPLIER  SUPPLIER  SUPPLIER
          A        B        C
          │        │        │
          └────────┼────────┘
                   ↓
               DELIVERY
                   ↓
                INVOICE
                   ↓
              COLLECTION
```

The **Sales Order is the commercial hub**, while the Supplier Purchase Orders represent the procurement execution required to fulfill that Sales Order.