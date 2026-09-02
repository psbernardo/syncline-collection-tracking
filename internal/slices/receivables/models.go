package receivables

import (
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
	"github.com/psbernardo/syncline-collection-tracking/internal/shared/tax"
)

type receivableModel struct {
	ID                 int64      `gorm:"column:delivery_receivable_id;primaryKey"`
	CompanyAccountID   int64      `gorm:"column:company_account_id"`
	CompanyName        string     `gorm:"column:company_name;->"`
	InvoiceID          *int64     `gorm:"column:invoice_id"`
	InvoiceNumber      *string    `gorm:"column:invoice_number"`
	PONumber           string     `gorm:"column:po_number"`
	PONumberNormalized string     `gorm:"column:po_number_normalized"`
	DeliveryDateUTC    time.Time  `gorm:"column:delivery_date_utc"`
	PaymentTermDays    int        `gorm:"column:payment_term_days"`
	DueDateUTC         time.Time  `gorm:"column:due_date_utc"`
	AmountDueScaled    int64      `gorm:"column:amount_due_scaled"`
	GrossAmountScaled  int64      `gorm:"column:gross_amount_scaled"`
	TaxRuleCode        *string    `gorm:"column:tax_rule_code"`
	EWTAmountScaled    int64      `gorm:"column:ewt_amount_scaled"`
	TaxBaseScaled      int64      `gorm:"column:tax_base_scaled"`
	VATAmountScaled    int64      `gorm:"column:vat_amount_scaled"`
	PaymentDateUTC     *time.Time `gorm:"column:payment_date_utc"`
	LifecycleStatus    string     `gorm:"column:lifecycle_status"`
	CreatedAtUTC       time.Time  `gorm:"column:created_at_utc"`
	UpdatedAtUTC       time.Time  `gorm:"column:updated_at_utc"`
	RowVersion         []byte     `gorm:"column:row_version;->"`
}

func (receivableModel) TableName() string { return "dbo.delivery_receivables" }

func (model receivableModel) toDomain() DeliveryReceivable {
	var invoiceID int64
	if model.InvoiceID != nil {
		invoiceID = *model.InvoiceID
	}
	return DeliveryReceivable{
		ID: model.ID, CompanyAccountID: model.CompanyAccountID, InvoiceID: invoiceID, InvoiceNumber: valueOrEmpty(model.InvoiceNumber), PONumber: model.PONumber,
		CompanyName:        model.CompanyName,
		PONumberNormalized: model.PONumberNormalized,
		DeliveryDateUTC:    model.DeliveryDateUTC, PaymentTermDays: model.PaymentTermDays,
		DueDateUTC: model.DueDateUTC, AmountDue: money.Amount(model.AmountDueScaled), GrossAmount: money.Amount(model.GrossAmountScaled),
		TaxRuleCode: tax.RuleCode(valueOrEmpty(model.TaxRuleCode)), EWTAmount: money.Amount(model.EWTAmountScaled), TaxBase: money.Amount(model.TaxBaseScaled), VATAmount: money.Amount(model.VATAmountScaled),
		PaymentDateUTC: model.PaymentDateUTC, LifecycleStatus: model.LifecycleStatus,
		CreatedAtUTC: model.CreatedAtUTC, UpdatedAtUTC: model.UpdatedAtUTC, RowVersion: model.RowVersion,
	}
}

func toModel(receivable DeliveryReceivable) receivableModel {
	return receivableModel{
		ID: receivable.ID, CompanyAccountID: receivable.CompanyAccountID, InvoiceID: nullableInvoiceID(receivable.InvoiceID), InvoiceNumber: nullableInvoiceNumber(receivable.InvoiceNumber), PONumber: receivable.PONumber,
		PONumberNormalized: receivable.PONumberNormalized,
		DeliveryDateUTC:    receivable.DeliveryDateUTC, PaymentTermDays: receivable.PaymentTermDays,
		DueDateUTC: receivable.DueDateUTC, AmountDueScaled: receivable.AmountDue.Int64(), GrossAmountScaled: receivable.GrossAmount.Int64(),
		TaxRuleCode: nullableRule(receivable.TaxRuleCode), EWTAmountScaled: receivable.EWTAmount.Int64(), TaxBaseScaled: receivable.TaxBase.Int64(), VATAmountScaled: receivable.VATAmount.Int64(),
		PaymentDateUTC: receivable.PaymentDateUTC, LifecycleStatus: receivable.LifecycleStatus,
		CreatedAtUTC: receivable.CreatedAtUTC, UpdatedAtUTC: receivable.UpdatedAtUTC, RowVersion: receivable.RowVersion,
	}
}

func nullableInvoiceID(value int64) *int64 {
	if value < 1 {
		return nil
	}
	return &value
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func nullableInvoiceNumber(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func nullableRule(value tax.RuleCode) *string {
	if value == tax.RuleNone {
		return nil
	}
	result := string(value)
	return &result
}
