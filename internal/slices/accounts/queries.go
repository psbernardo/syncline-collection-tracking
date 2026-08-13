package accounts

import (
	"context"
	"time"
)

type AccountViewModel struct {
	ID              int64
	CompanyName     string
	ContactPerson   string
	TINNumber       string
	BillingAddress  string
	DeliveryAddress string
	ContactNumber   string
}

func (service *commandService) List(ctx context.Context) ([]AccountViewModel, error) {
	accounts, err := service.repository.List(ctx)
	if err != nil {
		return nil, err
	}
	viewModels := make([]AccountViewModel, 0, len(accounts))
	for _, account := range accounts {
		viewModels = append(viewModels, AccountViewModel{
			ID:              account.ID,
			CompanyName:     account.CompanyName,
			ContactPerson:   account.ContactPerson,
			TINNumber:       account.TINNumber,
			BillingAddress:  account.BillingAddress,
			DeliveryAddress: account.DeliveryAddress,
			ContactNumber:   account.ContactNumber,
		})
	}
	return viewModels, nil
}

type AccountFormViewModel struct {
	Mode           string
	Action         string
	SubmitLabel    string
	PageTitle      string
	AccountID      int64
	Values         CompanyAccount
	Errors         ValidationErrors
	IdempotencyKey string
	RowVersion     string
}

func NewAccountForm() (AccountFormViewModel, error) {
	key, err := NewIdempotencyKey()
	if err != nil {
		return AccountFormViewModel{}, err
	}
	return AccountFormViewModel{Mode: "create", Action: "/accounts", SubmitLabel: "Save company", PageTitle: "New company account", IdempotencyKey: key}, nil
}

func formatTime(value time.Time) string { return value.Format("2006-01-02") }

func (service *commandService) Get(ctx context.Context, id int64) (CompanyAccount, error) {
	return service.repository.FindByID(ctx, nil, id)
}
