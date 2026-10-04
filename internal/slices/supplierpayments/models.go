package supplierpayments

import (
	"encoding/base64"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
)

type paymentModel struct {
	ID           int64     `gorm:"column:supplier_payment_id;primaryKey"`
	SupplierID   int64     `gorm:"column:supplier_id"`
	SupplierName string    `gorm:"column:supplier_name;->"`
	CheckNumber  string    `gorm:"column:check_number"`
	IssueDateUTC time.Time `gorm:"column:issue_date_utc"`
	DueDateUTC   time.Time `gorm:"column:due_date_utc"`
	AmountScaled int64     `gorm:"column:amount_scaled"`
	Status       string    `gorm:"column:lifecycle_status"`
	CreatedAtUTC time.Time `gorm:"column:created_at_utc"`
	UpdatedAtUTC time.Time `gorm:"column:updated_at_utc"`
	RowVersion   []byte    `gorm:"column:row_version;->"`
}

func (paymentModel) TableName() string { return "dbo.supplier_payments" }

func (model paymentModel) toDomain() Payment {
	return Payment{
		ID: model.ID, SupplierID: model.SupplierID, SupplierName: model.SupplierName,
		CheckNumber: model.CheckNumber, IssueDateUTC: model.IssueDateUTC, DueDateUTC: model.DueDateUTC,
		Amount: money.Amount(model.AmountScaled), Status: model.Status,
		CreatedAtUTC: model.CreatedAtUTC, UpdatedAtUTC: model.UpdatedAtUTC, RowVersion: model.RowVersion,
	}
}

func paymentToModel(payment Payment) paymentModel {
	return paymentModel{
		ID: payment.ID, SupplierID: payment.SupplierID, CheckNumber: payment.CheckNumber,
		IssueDateUTC: payment.IssueDateUTC, DueDateUTC: payment.DueDateUTC,
		AmountScaled: payment.Amount.Int64(), Status: payment.Status,
	}
}

func encodeVersion(version []byte) string { return base64.RawURLEncoding.EncodeToString(version) }

func decodeVersion(value string) ([]byte, error) { return base64.RawURLEncoding.DecodeString(value) }
