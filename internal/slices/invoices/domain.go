package invoices

import (
	"errors"
	"strings"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
	"github.com/psbernardo/syncline-collection-tracking/internal/shared/tax"
	"github.com/psbernardo/syncline-collection-tracking/internal/slices/quotations"
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
	ErrMixedTaxTreatment   = errors.New("sales order has mixed tax treatment")
	ErrUnsupportedTaxRule  = errors.New("sales order has an unsupported tax rule")
)

// ReceivableTaxRule maps Sales Order line tax treatment to the receivable rule.
// Receivables use EWT when the Sales Order was configured for 12% VAT.
func ReceivableTaxRule(taxCodes []string) (tax.RuleCode, error) {
	if len(taxCodes) == 0 {
		return tax.RuleNone, ErrUnsupportedTaxRule
	}
	rule := tax.RuleNone
	for _, code := range taxCodes {
		if code == "" || code == quotations.TaxNone {
			if rule != tax.RuleNone {
				return tax.RuleNone, ErrMixedTaxTreatment
			}
			continue
		}
		if code != quotations.TaxVAT12 {
			return tax.RuleNone, ErrUnsupportedTaxRule
		}
		if rule == tax.RuleNone && hasTaxCode(taxCodes, quotations.TaxNone) {
			return tax.RuleNone, ErrMixedTaxTreatment
		}
		rule = tax.RuleVATInclusiveEWT1
	}
	return rule, nil
}

func hasTaxCode(codes []string, wanted string) bool {
	for _, code := range codes {
		if code == "" || code == wanted {
			return true
		}
	}
	return false
}

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
	ID, SalesOrderID, CompanyAccountID, ReceivableID int64
	Number, SalesOrderNumber                         string
	CustomerName, BillingAddress                     string
	DeliveryAddress, ContactPerson                   string
	ContactNumber, Email                             string
	CustomerPONumber, SalesPerson                    string
	TermsDays                                        int
	InvoiceDateUTC, DueDateUTC                       time.Time
	Subtotal, Tax, Total                             money.Amount
	Status                                           Status
	CreatedAtUTC                                     time.Time
	Lines                                            []Line
}
