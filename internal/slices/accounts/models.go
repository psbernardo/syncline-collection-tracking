package accounts

import "time"

type accountModel struct {
	ID              int64     `gorm:"column:company_account_id;primaryKey"`
	CompanyName     string    `gorm:"column:company_name"`
	ContactPerson   string    `gorm:"column:contact_person"`
	TINNumber       string    `gorm:"column:tin_number"`
	BillingAddress  string    `gorm:"column:billing_address"`
	DeliveryAddress string    `gorm:"column:delivery_address"`
	ContactNumber   string    `gorm:"column:contact_number"`
	CreatedAtUTC    time.Time `gorm:"column:created_at_utc"`
	UpdatedAtUTC    time.Time `gorm:"column:updated_at_utc"`
	RowVersion      []byte    `gorm:"column:row_version;->"`
}

func (accountModel) TableName() string { return "dbo.company_accounts" }

func toModel(account CompanyAccount) accountModel {
	return accountModel{
		ID:              account.ID,
		CompanyName:     account.CompanyName,
		ContactPerson:   account.ContactPerson,
		TINNumber:       account.TINNumber,
		BillingAddress:  account.BillingAddress,
		DeliveryAddress: account.DeliveryAddress,
		ContactNumber:   account.ContactNumber,
		CreatedAtUTC:    account.CreatedAtUTC,
		UpdatedAtUTC:    account.UpdatedAtUTC,
		RowVersion:      account.RowVersion,
	}
}

func (model accountModel) toDomain() CompanyAccount {
	return CompanyAccount{
		ID:              model.ID,
		CompanyName:     model.CompanyName,
		ContactPerson:   model.ContactPerson,
		TINNumber:       model.TINNumber,
		BillingAddress:  model.BillingAddress,
		DeliveryAddress: model.DeliveryAddress,
		ContactNumber:   model.ContactNumber,
		CreatedAtUTC:    model.CreatedAtUTC,
		UpdatedAtUTC:    model.UpdatedAtUTC,
		RowVersion:      model.RowVersion,
	}
}
