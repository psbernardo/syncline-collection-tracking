package invoices

import (
	"errors"
	"strings"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
)

type Status string

const (
	Posted Status = "POSTED"
	Voided Status = "VOIDED"
)

var (
	ErrInvoiceNotReady     = errors.New("sales order is not ready for invoicing")
	ErrDuplicateInvoice    = errors.New("invoice number is already used")
	ErrAlreadyInvoiced     = errors.New("sales order has already been invoiced")
	ErrIdempotencyConflict = errors.New("idempotency key was used with different invoice data")
)

func ValidateInvoiceNumber(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("invoice number is required")
	}
	if len([]rune(value)) > 100 {
		return "", errors.New("invoice number must be 100 characters or fewer")
	}
	return value, nil
}

type Line struct {
	ID, SalesOrderLineID, ProductID int64
	SKU, Name, UOM, TaxCode         string
	Quantity, UnitPrice             money.Amount
	TaxRate                         int64
	LineTotal, VATInclusiveTotal    money.Amount
}

type Invoice struct {
	ID, SalesOrderID, CompanyAccountID int64
	Number, SalesOrderNumber           string
	CustomerName, BillingAddress       string
	DeliveryAddress, ContactPerson     string
	ContactNumber, Email               string
	CustomerPONumber, SalesPerson      string
	TermsDays                          int
	InvoiceDateUTC, DueDateUTC         time.Time
	Subtotal, Tax, Total               money.Amount
	Status                             Status
	CreatedAtUTC                       time.Time
	Lines                              []Line
}
