package receivables

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/businessdate"
	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
	"github.com/psbernardo/syncline-collection-tracking/internal/shared/tax"
)

var (
	ErrNotFound                            = errors.New("delivery receivable not found")
	ErrCompanyNotFound                     = errors.New("company account not found")
	ErrConflict                            = errors.New("delivery receivable was changed by another request")
	ErrProtected                           = errors.New("delivery receivable cannot be edited in its current state")
	ErrDuplicatePO                         = errors.New("PO number is already used by a non-cancelled receivable")
	ErrDuplicateInvoiceNumber              = errors.New("invoice number is already used by a non-cancelled receivable")
	ErrIdempotencyConflict                 = errors.New("idempotency key was already used with a different request")
	ErrAlreadyPaid                         = errors.New("delivery receivable has already been paid")
	ErrPaymentNotAllowed                   = errors.New("payment cannot be recorded for this receivable")
	ErrPaymentAcknowledgementNotReversible = errors.New("payment acknowledgement cannot be reversed for this receivable")
	ErrInvalidReversalReason               = errors.New("a reason is required to reverse the payment acknowledgement")
)

type ValidationErrors map[string]string

func (e ValidationErrors) Error() string { return "delivery receivable validation failed" }

type Classification string

const (
	ClassificationPending         Classification = "Pending"
	ClassificationNearDue         Classification = "Near Due"
	ClassificationOverdue         Classification = "Overdue"
	ClassificationPaymentReceived Classification = "Payment Received"
	ClassificationCancelled       Classification = "Cancelled"
	ClassificationArchived        Classification = "Archived"
)

type DeliveryReceivable struct {
	ID                 int64
	CompanyAccountID   int64
	CompanyName        string
	InvoiceNumber      string
	PONumber           string
	PONumberNormalized string
	DeliveryDateUTC    time.Time
	PaymentTermDays    int
	DueDateUTC         time.Time
	AmountDue          money.Amount
	GrossAmount        money.Amount
	TaxRuleCode        tax.RuleCode
	EWTAmount          money.Amount
	TaxBase            money.Amount
	VATAmount          money.Amount
	PaymentDateUTC     *time.Time
	LifecycleStatus    string
	CreatedAtUTC       time.Time
	UpdatedAtUTC       time.Time
	RowVersion         []byte
}

func NewDeliveryReceivable(companyID int64, invoiceNumber, poNumber, amountInput, deliveryInput string, termDays int) (DeliveryReceivable, error) {
	return NewDeliveryReceivableWithTax(companyID, invoiceNumber, poNumber, amountInput, deliveryInput, termDays, tax.RuleNone)
}

func NewDeliveryReceivableWithTax(companyID int64, invoiceNumber, poNumber, amountInput, deliveryInput string, termDays int, taxRule tax.RuleCode) (DeliveryReceivable, error) {
	errors := ValidationErrors{}
	if companyID < 1 {
		errors["CompanyAccountID"] = "Select a company."
	}
	invoiceNumber = strings.TrimSpace(invoiceNumber)
	if invoiceNumber == "" {
		errors["InvoiceNumber"] = "This field is required."
	} else if len(invoiceNumber) > 100 || !isAlphaNumeric(invoiceNumber) {
		errors["InvoiceNumber"] = "Use 1-100 ASCII letters and numbers only."
	}
	poNumber = strings.TrimSpace(poNumber)
	if poNumber == "" {
		errors["PONumber"] = "This field is required."
	} else if len(poNumber) > 100 || !isAlphaNumeric(poNumber) {
		errors["PONumber"] = "Use 1-100 ASCII letters and numbers only."
	}
	amount, err := money.Parse(amountInput)
	if err != nil {
		errors["Amount"] = "Enter a valid non-negative amount."
	}
	breakdown, taxErr := tax.CalculateRule(amount, taxRule)
	if taxErr != nil {
		errors["TaxRuleCode"] = "Select a valid tax rule."
	}
	deliveryDate, err := businessdate.Parse(strings.TrimSpace(deliveryInput))
	if err != nil {
		errors["DeliveryDate"] = "Enter a valid delivery date."
	}
	if termDays < 1 || termDays > 120 {
		errors["PaymentTermDays"] = "Payment term must be between 1 and 120 days."
	}
	if len(errors) > 0 {
		return DeliveryReceivable{}, errors
	}
	dueDate, err := businessdate.DueDate(deliveryDate, termDays)
	if err != nil {
		return DeliveryReceivable{}, fmt.Errorf("calculate due date: %w", err)
	}
	return DeliveryReceivable{
		CompanyAccountID:   companyID,
		InvoiceNumber:      invoiceNumber,
		PONumber:           poNumber,
		PONumberNormalized: NormalizePONumber(poNumber),
		DeliveryDateUTC:    deliveryDate,
		PaymentTermDays:    termDays,
		DueDateUTC:         dueDate,
		AmountDue:          breakdown.NetAmount,
		GrossAmount:        breakdown.GrossAmount,
		TaxRuleCode:        taxRule,
		EWTAmount:          breakdown.WithholdingAmount,
		TaxBase:            breakdown.TaxBase,
		VATAmount:          breakdown.VATAmount,
		LifecycleStatus:    "Active",
	}, nil
}

func NormalizePONumber(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func (receivable DeliveryReceivable) Classification(now time.Time) string {
	return string(receivable.ClassificationAt(now))
}

func (receivable DeliveryReceivable) ClassificationAt(now time.Time) Classification {
	if receivable.LifecycleStatus != "Active" {
		return Classification(receivable.LifecycleStatus)
	}
	if receivable.PaymentDateUTC != nil {
		return ClassificationPaymentReceived
	}
	today, nearEnd := businessdate.ClassificationBoundaries(now)
	todayValue := businessdate.FormatUTC(today)
	dueDate := businessdate.FormatUTC(receivable.DueDateUTC)
	if dueDate < todayValue {
		return ClassificationOverdue
	}
	if dueDate <= businessdate.FormatUTC(nearEnd) {
		return ClassificationNearDue
	}
	return ClassificationPending
}

func (receivable DeliveryReceivable) DaysOverdueAt(now time.Time) int {
	if receivable.ClassificationAt(now) != ClassificationOverdue {
		return 0
	}
	today, _ := businessdate.Parse(businessdate.FormatUTC(now))
	due, _ := businessdate.Parse(businessdate.FormatUTC(receivable.DueDateUTC))
	return int(today.Sub(due).Hours() / 24)
}

func (receivable DeliveryReceivable) DaysUntilDueAt(now time.Time) int {
	if receivable.ClassificationAt(now) != ClassificationNearDue {
		return 0
	}
	today, _ := businessdate.Parse(businessdate.FormatUTC(now))
	due, _ := businessdate.Parse(businessdate.FormatUTC(receivable.DueDateUTC))
	return int(due.Sub(today).Hours() / 24)
}

func ValidatePaymentDate(receivable DeliveryReceivable, input string, now time.Time) (time.Time, error) {
	validation := ValidationErrors{}
	paymentDate, err := businessdate.Parse(strings.TrimSpace(input))
	if err != nil {
		validation["PaymentDate"] = "Enter a valid payment date."
	} else {
		paymentValue := businessdate.FormatUTC(paymentDate)
		if paymentValue < businessdate.FormatUTC(receivable.DeliveryDateUTC) {
			validation["PaymentDate"] = "Payment date cannot be before the delivery date."
		} else if paymentValue > businessdate.FormatUTC(now) {
			validation["PaymentDate"] = "Payment date cannot be in the future."
		}
	}
	if len(validation) > 0 {
		return time.Time{}, validation
	}
	return paymentDate, nil
}

func ValidateReversalReason(input string) (string, error) {
	reason := strings.TrimSpace(input)
	if reason == "" || utf8.RuneCountInString(reason) > 500 {
		return "", ValidationErrors{"Reason": "Enter a reason of 1-500 characters."}
	}
	return reason, nil
}

func isAlphaNumeric(value string) bool {
	for _, char := range value {
		if (char < 'A' || char > 'Z') && (char < 'a' || char > 'z') && (char < '0' || char > '9') {
			return false
		}
	}
	return true
}
