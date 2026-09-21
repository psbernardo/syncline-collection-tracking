package receivables

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/businessdate"
	"gorm.io/gorm"
)

type GormRepository struct{ db *gorm.DB }

func NewGormRepository(db *gorm.DB) *GormRepository { return &GormRepository{db: db} }

func (repository *GormRepository) Create(ctx context.Context, db *gorm.DB, receivable DeliveryReceivable) (DeliveryReceivable, error) {
	model := toModel(receivable)
	if err := db.WithContext(ctx).Omit("RowVersion").Create(&model).Error; err != nil {
		return DeliveryReceivable{}, fmt.Errorf("create delivery receivable: %w", err)
	}
	return model.toDomain(), nil
}

func (repository *GormRepository) FindByID(ctx context.Context, db *gorm.DB, id int64) (DeliveryReceivable, error) {
	if db == nil {
		db = repository.db
	}
	var model receivableModel
	if err := db.WithContext(ctx).Table("dbo.delivery_receivables AS r").
		Select(receivableSelect).
		Joins("INNER JOIN dbo.company_accounts AS a ON a.company_account_id = r.company_account_id").
		Where("r.delivery_receivable_id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return DeliveryReceivable{}, ErrNotFound
		}
		return DeliveryReceivable{}, fmt.Errorf("find delivery receivable: %w", err)
	}
	return model.toDomain(), nil
}

func (repository *GormRepository) List(ctx context.Context) ([]DeliveryReceivable, error) {
	var models []receivableModel
	if err := repository.db.WithContext(ctx).Table("dbo.delivery_receivables AS r").
		Select(receivableSelect).
		Joins("INNER JOIN dbo.company_accounts AS a ON a.company_account_id = r.company_account_id").
		Order("r.due_date_utc, r.delivery_receivable_id").
		Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list delivery receivables: %w", err)
	}
	result := make([]DeliveryReceivable, 0, len(models))
	for _, model := range models {
		result = append(result, model.toDomain())
	}
	return result, nil
}

func (repository *GormRepository) ListFiltered(ctx context.Context, query ListQuery) ([]DeliveryReceivable, int64, error) {
	query = normalizeListQuery(query)
	if query.Now.IsZero() {
		query.Now = time.Now().UTC()
	}
	today, nearEnd := businessdate.ClassificationBoundaries(query.Now)
	base := repository.db.WithContext(ctx).Table("dbo.delivery_receivables AS r").Joins("INNER JOIN dbo.company_accounts AS a ON a.company_account_id = r.company_account_id")
	base = applyFilters(base, query, today, nearEnd)
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count filtered receivables: %w", err)
	}
	var cursorValue cursor
	if query.Cursor != "" {
		var err error
		cursorValue, err = decodeCursor(query.Cursor)
		if err != nil {
			return nil, 0, err
		}
		cursorDate, err := businessdate.Parse(cursorValue.DueDate)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid cursor date")
		}
		if cursorValue.Filter != filterSignature(query) {
			return nil, 0, fmt.Errorf("invalid cursor")
		}
		base = base.Where("r.due_date_utc > ? OR (r.due_date_utc = ? AND r.delivery_receivable_id > ?)", cursorDate, cursorDate, cursorValue.ID)
		if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
			return nil, 0, fmt.Errorf("count remaining receivables: %w", err)
		}
	}
	var models []receivableModel
	if err := base.Select(receivableSelect).Order("r.due_date_utc, r.delivery_receivable_id").Limit(query.PageSize).Find(&models).Error; err != nil {
		return nil, 0, fmt.Errorf("list filtered receivables: %w", err)
	}
	result := make([]DeliveryReceivable, 0, len(models))
	for _, model := range models {
		result = append(result, model.toDomain())
	}
	return result, total, nil
}

func (repository *GormRepository) ListFilteredAll(ctx context.Context, query ListQuery) ([]DeliveryReceivable, error) {
	query = normalizeListQuery(query)
	if query.Now.IsZero() {
		query.Now = time.Now().UTC()
	}
	today, nearEnd := businessdate.ClassificationBoundaries(query.Now)
	base := repository.db.WithContext(ctx).Table("dbo.delivery_receivables AS r").Joins("INNER JOIN dbo.company_accounts AS a ON a.company_account_id = r.company_account_id")
	base = applyFilters(base, query, today, nearEnd)
	var models []receivableModel
	if err := base.Select(receivableSelect).Order("a.company_name, r.due_date_utc, r.delivery_receivable_id").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list all filtered receivables: %w", err)
	}
	result := make([]DeliveryReceivable, 0, len(models))
	for _, model := range models {
		result = append(result, model.toDomain())
	}
	return result, nil
}

func applyFilters(query *gorm.DB, input ListQuery, today, nearEnd time.Time) *gorm.DB {
	if len(input.CompanyAccountIDs) > 0 {
		query = query.Where("r.company_account_id IN ?", input.CompanyAccountIDs)
	} else if input.CompanyFilterProvided {
		query = query.Where("1 = 0")
	}
	if invoice := input.Invoice; invoice != "" {
		query = query.Where("r.invoice_number LIKE ? ESCAPE '\\'", "%"+escapeLike(invoice)+"%")
	}
	if po := input.PO; po != "" {
		query = query.Where("r.po_number_normalized LIKE ? ESCAPE '\\'", "%"+escapeLike(po)+"%")
	}
	statuses := input.Statuses
	if len(statuses) == 0 {
		if input.StatusFilterProvided {
			return query.Where("1 = 0")
		}
		return query.Where("r.lifecycle_status = ?", "Active")
	}
	conditions := make([]string, 0, len(statuses))
	args := make([]interface{}, 0, len(statuses)*2)
	for _, status := range statuses {
		switch status {
		case "cancelled":
			conditions = append(conditions, "r.lifecycle_status = ?")
			args = append(args, "Cancelled")
		case "archived":
			conditions = append(conditions, "r.lifecycle_status = ?")
			args = append(args, "Archived")
		case "pending":
			conditions = append(conditions, "(r.lifecycle_status = 'Active' AND r.payment_date_utc IS NULL AND r.due_date_utc > ?)")
			args = append(args, nearEnd)
		case "near_due":
			conditions = append(conditions, "(r.lifecycle_status = 'Active' AND r.payment_date_utc IS NULL AND r.due_date_utc >= ? AND r.due_date_utc <= ?)")
			args = append(args, today, nearEnd)
		case "overdue":
			conditions = append(conditions, "(r.lifecycle_status = 'Active' AND r.payment_date_utc IS NULL AND r.due_date_utc < ?)")
			args = append(args, today)
		case "payment_received":
			conditions = append(conditions, "(r.lifecycle_status = 'Active' AND r.payment_date_utc IS NOT NULL)")
		}
	}
	return query.Where("("+strings.Join(conditions, " OR ")+")", args...)
}

func escapeLike(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "%", "\\%")
	value = strings.ReplaceAll(value, "_", "\\_")
	value = strings.ReplaceAll(value, "[", "\\[")
	return value
}

func (repository *GormRepository) Update(ctx context.Context, db *gorm.DB, receivable DeliveryReceivable, originalVersion []byte) (DeliveryReceivable, error) {
	result := db.WithContext(ctx).Model(&receivableModel{}).
		Where("delivery_receivable_id = ? AND row_version = ?", receivable.ID, originalVersion).
		Updates(map[string]interface{}{
			"company_account_id":   receivable.CompanyAccountID,
			"invoice_id":           nullableInvoiceID(receivable.InvoiceID),
			"invoice_number":       nullableInvoiceNumber(receivable.InvoiceNumber),
			"po_number":            receivable.PONumber,
			"po_number_normalized": receivable.PONumberNormalized,
			"delivery_date_utc":    receivable.DeliveryDateUTC,
			"payment_term_days":    receivable.PaymentTermDays,
			"due_date_utc":         receivable.DueDateUTC,
			"amount_due_scaled":    receivable.AmountDue.Int64(),
			"gross_amount_scaled":  receivable.GrossAmount.Int64(),
			"tax_rule_code":        nullableRule(receivable.TaxRuleCode),
			"ewt_amount_scaled":    receivable.EWTAmount.Int64(),
			"tax_base_scaled":      receivable.TaxBase.Int64(),
			"vat_amount_scaled":    receivable.VATAmount.Int64(),
			"updated_at_utc":       gorm.Expr("SYSUTCDATETIME()"),
		})
	if result.Error != nil {
		return DeliveryReceivable{}, fmt.Errorf("update delivery receivable: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return DeliveryReceivable{}, ErrConflict
	}
	return repository.FindByID(ctx, db, receivable.ID)
}

const receivableSelect = "r.delivery_receivable_id, r.company_account_id, a.company_name, r.invoice_id, r.invoice_number, r.po_number, r.po_number_normalized, r.delivery_date_utc, r.payment_term_days, r.due_date_utc, r.amount_due_scaled, r.gross_amount_scaled, r.tax_rule_code, r.ewt_amount_scaled, r.tax_base_scaled, r.vat_amount_scaled, r.payment_date_utc, r.lifecycle_status, r.created_at_utc, r.updated_at_utc, r.row_version"

func (repository *GormRepository) MarkPaymentReceived(ctx context.Context, db *gorm.DB, id int64, paymentDate time.Time, originalVersion []byte) (DeliveryReceivable, error) {
	result := db.WithContext(ctx).Model(&receivableModel{}).
		Where("delivery_receivable_id = ? AND row_version = ? AND lifecycle_status = ? AND payment_date_utc IS NULL", id, originalVersion, "Active").
		Updates(map[string]interface{}{"payment_date_utc": paymentDate, "updated_at_utc": gorm.Expr("SYSUTCDATETIME()")})
	if result.Error != nil {
		return DeliveryReceivable{}, fmt.Errorf("mark payment received: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return DeliveryReceivable{}, ErrConflict
	}
	return repository.FindByID(ctx, db, id)
}

func (repository *GormRepository) ReversePaymentAcknowledgement(ctx context.Context, db *gorm.DB, id int64, originalVersion []byte) (DeliveryReceivable, error) {
	result := db.WithContext(ctx).Model(&receivableModel{}).
		Where("delivery_receivable_id = ? AND row_version = ? AND lifecycle_status = ? AND payment_date_utc IS NOT NULL", id, originalVersion, "Active").
		Updates(map[string]interface{}{"payment_date_utc": nil, "updated_at_utc": gorm.Expr("SYSUTCDATETIME()")})
	if result.Error != nil {
		return DeliveryReceivable{}, fmt.Errorf("reverse payment acknowledgement: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return DeliveryReceivable{}, ErrConflict
	}
	return repository.FindByID(ctx, db, id)
}

func (repository *GormRepository) FindBlockingPONumber(ctx context.Context, db *gorm.DB, normalizedPO string, excludeID int64) (bool, error) {
	var count int64
	query := db.WithContext(ctx).Model(&receivableModel{}).
		Where("po_number_normalized = ? AND lifecycle_status <> ?", normalizedPO, "Cancelled")
	if excludeID > 0 {
		query = query.Where("delivery_receivable_id <> ?", excludeID)
	}
	if err := query.Count(&count).Error; err != nil {
		return false, fmt.Errorf("check duplicate PO: %w", err)
	}
	return count > 0, nil
}

func (repository *GormRepository) FindBlockingInvoiceNumber(ctx context.Context, db *gorm.DB, invoiceNumber string, excludeID int64) (bool, error) {
	var count int64
	query := db.WithContext(ctx).Model(&receivableModel{}).
		Where("invoice_number = ? AND lifecycle_status IN (?, ?)", invoiceNumber, "Active", "Archived")
	if excludeID > 0 {
		query = query.Where("delivery_receivable_id <> ?", excludeID)
	}
	if err := query.Count(&count).Error; err != nil {
		return false, fmt.Errorf("check duplicate invoice number: %w", err)
	}
	return count > 0, nil
}
