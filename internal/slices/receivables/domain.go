package receivables

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/businessdate"
	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
)

var (
	ErrNotFound               = errors.New("delivery receivable not found")
	ErrCompanyNotFound        = errors.New("company account not found")
	ErrConflict               = errors.New("delivery receivable was changed by another request")
	ErrProtected              = errors.New("delivery receivable cannot be edited in its current state")
	ErrDuplicatePO            = errors.New("PO number is already used by a non-cancelled receivable")
	ErrDuplicateInvoiceNumber = errors.New("invoice number is already used by a non-cancelled receivable")
	ErrIdempotencyConflict    = errors.New("idempotency key was already used with a different request")
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
	PaymentDateUTC     *time.Time
	LifecycleStatus    string
	CreatedAtUTC       time.Time
	UpdatedAtUTC       time.Time
	RowVersion         []byte
}

func NewDeliveryReceivable(companyID int64, invoiceNumber, poNumber, amountInput, deliveryInput string, termDays int) (DeliveryReceivable, error) {
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
		AmountDue:          amount,
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

func isAlphaNumeric(value string) bool {
	for _, char := range value {
		if (char < 'A' || char > 'Z') && (char < 'a' || char > 'z') && (char < '0' || char > '9') {
			return false
		}
	}
	return true
}
