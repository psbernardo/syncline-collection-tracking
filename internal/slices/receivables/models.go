package receivables

import (
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
)

type receivableModel struct {
	ID                 int64      `gorm:"column:delivery_receivable_id;primaryKey"`
	CompanyAccountID   int64      `gorm:"column:company_account_id"`
	CompanyName        string     `gorm:"column:company_name;->"`
	PONumber           string     `gorm:"column:po_number"`
	PONumberNormalized string     `gorm:"column:po_number_normalized"`
	DeliveryDateUTC    time.Time  `gorm:"column:delivery_date_utc"`
	PaymentTermDays    int        `gorm:"column:payment_term_days"`
	DueDateUTC         time.Time  `gorm:"column:due_date_utc"`
	AmountDueScaled    int64      `gorm:"column:amount_due_scaled"`
	PaymentDateUTC     *time.Time `gorm:"column:payment_date_utc"`
	LifecycleStatus    string     `gorm:"column:lifecycle_status"`
	CreatedAtUTC       time.Time  `gorm:"column:created_at_utc"`
	UpdatedAtUTC       time.Time  `gorm:"column:updated_at_utc"`
	RowVersion         []byte     `gorm:"column:row_version;->"`
}

func (receivableModel) TableName() string { return "dbo.delivery_receivables" }

func (model receivableModel) toDomain() DeliveryReceivable {
	return DeliveryReceivable{
		ID: model.ID, CompanyAccountID: model.CompanyAccountID, PONumber: model.PONumber,
		CompanyName:        model.CompanyName,
		PONumberNormalized: model.PONumberNormalized,
		DeliveryDateUTC:    model.DeliveryDateUTC, PaymentTermDays: model.PaymentTermDays,
		DueDateUTC: model.DueDateUTC, AmountDue: money.Amount(model.AmountDueScaled),
		PaymentDateUTC: model.PaymentDateUTC, LifecycleStatus: model.LifecycleStatus,
		CreatedAtUTC: model.CreatedAtUTC, UpdatedAtUTC: model.UpdatedAtUTC, RowVersion: model.RowVersion,
	}
}

func toModel(receivable DeliveryReceivable) receivableModel {
	return receivableModel{
		ID: receivable.ID, CompanyAccountID: receivable.CompanyAccountID, PONumber: receivable.PONumber,
		PONumberNormalized: receivable.PONumberNormalized,
		DeliveryDateUTC:    receivable.DeliveryDateUTC, PaymentTermDays: receivable.PaymentTermDays,
		DueDateUTC: receivable.DueDateUTC, AmountDueScaled: receivable.AmountDue.Int64(),
		PaymentDateUTC: receivable.PaymentDateUTC, LifecycleStatus: receivable.LifecycleStatus,
		CreatedAtUTC: receivable.CreatedAtUTC, UpdatedAtUTC: receivable.UpdatedAtUTC, RowVersion: receivable.RowVersion,
	}
}
