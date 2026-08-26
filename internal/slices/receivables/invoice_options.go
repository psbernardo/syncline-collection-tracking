package receivables

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type InvoiceOption struct {
	ID               int64
	Number           string
	CompanyAccountID int64
	CompanyName      string
	InvoiceDateUTC   time.Time
	TotalScaled      int64
	Status           string
}

type InvoiceRepository interface {
	ListSelectable(ctx context.Context, companyAccountID, excludeReceivableID int64) ([]InvoiceOption, error)
	FindSelectable(ctx context.Context, db *gorm.DB, id, excludeReceivableID int64) (InvoiceOption, error)
}

type GormInvoiceRepository struct{ db *gorm.DB }

func NewGormInvoiceRepository(db *gorm.DB) *GormInvoiceRepository {
	return &GormInvoiceRepository{db: db}
}

func (repository *GormInvoiceRepository) ListSelectable(ctx context.Context, companyAccountID, excludeReceivableID int64) ([]InvoiceOption, error) {
	return repository.listSelectable(ctx, repository.db, companyAccountID, excludeReceivableID)
}

func (repository *GormInvoiceRepository) listSelectable(ctx context.Context, db *gorm.DB, companyAccountID, excludeReceivableID int64) ([]InvoiceOption, error) {
	var result []InvoiceOption
	query := db.WithContext(ctx).Table("dbo.invoices AS i").Select("i.invoice_id, i.invoice_number, i.company_account_id, a.company_name, i.invoice_date_utc, i.total_scaled, i.status").Joins("INNER JOIN dbo.company_accounts AS a ON a.company_account_id = i.company_account_id").Where("i.status = ?", "POSTED")
	if companyAccountID > 0 {
		query = query.Where("i.company_account_id = ?", companyAccountID)
	}
	if excludeReceivableID > 0 {
		query = query.Where("(NOT EXISTS (SELECT 1 FROM dbo.delivery_receivables AS r WHERE r.invoice_id = i.invoice_id AND r.lifecycle_status <> ?) OR EXISTS (SELECT 1 FROM dbo.delivery_receivables AS current_r WHERE current_r.invoice_id = i.invoice_id AND current_r.delivery_receivable_id = ?))", "Cancelled", excludeReceivableID)
	}
	if err := query.Order("i.invoice_date_utc DESC, i.invoice_number, i.invoice_id").Find(&result).Error; err != nil {
		return nil, fmt.Errorf("list selectable invoices: %w", err)
	}
	return result, nil
}

func (repository *GormInvoiceRepository) FindSelectable(ctx context.Context, db *gorm.DB, id, excludeReceivableID int64) (InvoiceOption, error) {
	if db == nil {
		db = repository.db
	}
	var result InvoiceOption
	query := db.WithContext(ctx).Table("dbo.invoices AS i").Select("i.invoice_id, i.invoice_number, i.company_account_id, a.company_name, i.invoice_date_utc, i.total_scaled, i.status").Joins("INNER JOIN dbo.company_accounts AS a ON a.company_account_id = i.company_account_id").Where("i.invoice_id = ?", id).Where("i.status = ?", "POSTED")
	if excludeReceivableID > 0 {
		query = query.Where("(NOT EXISTS (SELECT 1 FROM dbo.delivery_receivables AS r WHERE r.invoice_id = i.invoice_id AND r.lifecycle_status <> ?) OR EXISTS (SELECT 1 FROM dbo.delivery_receivables AS current_r WHERE current_r.invoice_id = i.invoice_id AND current_r.delivery_receivable_id = ?))", "Cancelled", excludeReceivableID)
	}
	if err := query.First(&result).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return InvoiceOption{}, ErrInvoiceNotFound
		}
		return InvoiceOption{}, fmt.Errorf("find selectable invoice: %w", err)
	}
	return result, nil
}
