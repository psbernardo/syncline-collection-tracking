package supplierpayments

import (
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/businessdate"
	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
)

var (
	ErrNotFound         = errors.New("supplier payment not found")
	ErrSupplierNotFound = errors.New("active supplier not found")
	ErrDuplicateCheck   = errors.New("this check number is already in use")
	ErrConflict         = errors.New("supplier payment was changed by another request")
	ErrVoided           = errors.New("voided supplier payments cannot be edited")
)

type ValidationErrors map[string]string

func (ValidationErrors) Error() string { return "supplier payment validation failed" }

type Payment struct {
	ID           int64
	SupplierID   int64
	SupplierName string
	CheckNumber  string
	IssueDateUTC time.Time
	DueDateUTC   time.Time
	Amount       money.Amount
	Status       string
	CreatedAtUTC time.Time
	UpdatedAtUTC time.Time
	RowVersion   []byte
}

func NewPayment(supplierID int64, checkNumber, issueDateInput, dueDateInput, amountInput string) (Payment, error) {
	validation := ValidationErrors{}
	if supplierID < 1 {
		validation["SupplierID"] = "Select a supplier."
	}
	checkNumber = strings.TrimSpace(checkNumber)
	if checkNumber == "" {
		validation["CheckNumber"] = "This field is required."
	} else if utf8.RuneCountInString(checkNumber) > 100 {
		validation["CheckNumber"] = "Use 100 characters or fewer."
	}
	issueDate, err := businessdate.Parse(strings.TrimSpace(issueDateInput))
	if err != nil {
		validation["IssueDate"] = "Enter a valid issue date."
	}
	dueDate, err := businessdate.Parse(strings.TrimSpace(dueDateInput))
	if err != nil {
		validation["DueDate"] = "Enter a valid due date."
	} else if issueDateInput != "" && businessdate.FormatUTC(dueDate) < businessdate.FormatUTC(issueDate) {
		validation["DueDate"] = "Due date cannot be before the issue date."
	}
	amount, err := money.Parse(amountInput)
	if err != nil || amount == 0 {
		validation["Amount"] = "Enter an amount greater than zero."
	}
	if len(validation) > 0 {
		return Payment{}, validation
	}
	return Payment{SupplierID: supplierID, CheckNumber: checkNumber, IssueDateUTC: issueDate, DueDateUTC: dueDate, Amount: amount, Status: "Active"}, nil
}

type SupplierOption struct {
	ID   int64
	Name string
}

type Summary struct {
	NearDue           money.Amount
	WithinFiveDays    money.Amount
	DaysSixToFourteen money.Amount
}

type ListItem struct {
	ID           int64
	SupplierID   int64
	SupplierName string
	CheckNumber  string
	IssueDate    string
	DueDate      string
	Amount       string
	Status       string
	DueLabel     string
	RowVersion   string
}

type ListResult struct {
	Items   []ListItem
	Summary Summary
}

func buildListResult(payments []Payment, now time.Time, includeVoided bool) ListResult {
	today, _ := businessdate.Parse(businessdate.FormatUTC(now))
	result := ListResult{Items: make([]ListItem, 0, len(payments))}
	for _, payment := range payments {
		if payment.Status == "Active" {
			due, _ := businessdate.Parse(businessdate.FormatUTC(payment.DueDateUTC))
			days := int(due.Sub(today).Hours() / 24)
			if days >= 0 && days <= 14 {
				result.Summary.NearDue += payment.Amount
				if days <= 5 {
					result.Summary.WithinFiveDays += payment.Amount
				} else {
					result.Summary.DaysSixToFourteen += payment.Amount
				}
			}
		} else if !includeVoided {
			continue
		}
		due, _ := businessdate.Parse(businessdate.FormatUTC(payment.DueDateUTC))
		days := int(due.Sub(today).Hours() / 24)
		label := "Upcoming"
		switch {
		case payment.Status == "Voided":
			label = "Voided"
		case days < 0:
			label = "Overdue by " + strconv.Itoa(-days) + " day(s)"
		case days == 0:
			label = "Due today"
		case days == 1:
			label = "Due tomorrow"
		default:
			label = "Due in " + strconv.Itoa(days) + " days"
		}
		result.Items = append(result.Items, ListItem{
			ID: payment.ID, SupplierID: payment.SupplierID, SupplierName: payment.SupplierName,
			CheckNumber: payment.CheckNumber, IssueDate: businessdate.FormatUTC(payment.IssueDateUTC),
			DueDate: businessdate.FormatUTC(payment.DueDateUTC), Amount: payment.Amount.FormatPHP(),
			Status: payment.Status, DueLabel: label,
			RowVersion: encodeVersion(payment.RowVersion),
		})
	}
	return result
}
