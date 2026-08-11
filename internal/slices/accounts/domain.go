package accounts

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrAccountNotFound = errors.New("company account not found")
	ErrAccountConflict = errors.New("company account was changed by another request")
)

const (
	maxNameLength    = 255
	maxTINLength     = 50
	maxAddressLength = 255
	maxContactLength = 50
)

type CompanyAccount struct {
	ID              int64
	CompanyName     string
	ContactPerson   string
	TINNumber       string
	BillingAddress  string
	DeliveryAddress string
	ContactNumber   string
	CreatedAtUTC    time.Time
	UpdatedAtUTC    time.Time
	RowVersion      []byte
}

type ValidationErrors map[string]string

func (e ValidationErrors) Error() string { return "company account validation failed" }

func NewCompanyAccount(input CompanyAccount) (CompanyAccount, error) {
	account := input
	account.CompanyName = strings.TrimSpace(account.CompanyName)
	account.ContactPerson = strings.TrimSpace(account.ContactPerson)
	account.TINNumber = strings.TrimSpace(account.TINNumber)
	account.BillingAddress = strings.TrimSpace(account.BillingAddress)
	account.DeliveryAddress = strings.TrimSpace(account.DeliveryAddress)
	account.ContactNumber = strings.TrimSpace(account.ContactNumber)

	errors := ValidationErrors{}
	validateRequired(errors, "CompanyName", account.CompanyName, maxNameLength)
	validateRequired(errors, "ContactPerson", account.ContactPerson, maxNameLength)
	validateRequired(errors, "TINNumber", account.TINNumber, maxTINLength)
	validateRequired(errors, "BillingAddress", account.BillingAddress, maxAddressLength)
	validateRequired(errors, "DeliveryAddress", account.DeliveryAddress, maxAddressLength)
	validateRequired(errors, "ContactNumber", account.ContactNumber, maxContactLength)
	if len(errors) > 0 {
		return CompanyAccount{}, errors
	}
	return account, nil
}

func validateRequired(errors ValidationErrors, field, value string, maxLength int) {
	if value == "" {
		errors[field] = "This field is required."
		return
	}
	if len([]rune(value)) > maxLength {
		errors[field] = fmt.Sprintf("Must be %d characters or fewer.", maxLength)
	}
}
