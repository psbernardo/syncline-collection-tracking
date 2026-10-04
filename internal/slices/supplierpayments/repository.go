package supplierpayments

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

type Repository interface {
	Create(context.Context, *gorm.DB, Payment) (Payment, error)
	FindByID(context.Context, *gorm.DB, int64) (Payment, error)
	List(context.Context) ([]Payment, error)
	HasDuplicateCheckNumber(context.Context, *gorm.DB, string, int64) (bool, error)
	Update(context.Context, *gorm.DB, Payment, []byte) (Payment, error)
	Void(context.Context, *gorm.DB, int64, []byte) error
}

type GormRepository struct{ db *gorm.DB }

func NewGormRepository(db *gorm.DB) *GormRepository { return &GormRepository{db: db} }

func (repository *GormRepository) Create(ctx context.Context, db *gorm.DB, payment Payment) (Payment, error) {
	db = repository.useDB(db)
	model := paymentToModel(payment)
	if err := db.WithContext(ctx).Omit("SupplierName", "RowVersion").Create(&model).Error; err != nil {
		return Payment{}, fmt.Errorf("create supplier payment: %w", err)
	}
	return repository.FindByID(ctx, db, model.ID)
}

func (repository *GormRepository) FindByID(ctx context.Context, db *gorm.DB, id int64) (Payment, error) {
	db = repository.useDB(db)
	var model paymentModel
	err := db.WithContext(ctx).Table("dbo.supplier_payments AS p").
		Select("p.supplier_payment_id, p.supplier_id, s.name AS supplier_name, p.check_number, p.issue_date_utc, p.due_date_utc, p.amount_scaled, p.lifecycle_status, p.created_at_utc, p.updated_at_utc, p.row_version").
		Joins("JOIN dbo.suppliers s ON s.supplier_id = p.supplier_id").
		Where("p.supplier_payment_id = ?", id).Take(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Payment{}, ErrNotFound
	}
	if err != nil {
		return Payment{}, fmt.Errorf("find supplier payment: %w", err)
	}
	return model.toDomain(), nil
}

func (repository *GormRepository) List(ctx context.Context) ([]Payment, error) {
	var models []paymentModel
	err := repository.db.WithContext(ctx).Table("dbo.supplier_payments AS p").
		Select("p.supplier_payment_id, p.supplier_id, s.name AS supplier_name, p.check_number, p.issue_date_utc, p.due_date_utc, p.amount_scaled, p.lifecycle_status, p.created_at_utc, p.updated_at_utc, p.row_version").
		Joins("JOIN dbo.suppliers s ON s.supplier_id = p.supplier_id").
		Order("p.due_date_utc, p.supplier_payment_id").Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("list supplier payments: %w", err)
	}
	payments := make([]Payment, 0, len(models))
	for _, model := range models {
		payments = append(payments, model.toDomain())
	}
	return payments, nil
}

func (repository *GormRepository) HasDuplicateCheckNumber(ctx context.Context, db *gorm.DB, checkNumber string, excludeID int64) (bool, error) {
	db = repository.useDB(db)
	query := db.WithContext(ctx).Table("dbo.supplier_payments").
		Where("check_number = ?", checkNumber)
	if excludeID > 0 {
		query = query.Where("supplier_payment_id <> ?", excludeID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, fmt.Errorf("check duplicate supplier check number: %w", err)
	}
	return count > 0, nil
}

func (repository *GormRepository) Update(ctx context.Context, db *gorm.DB, payment Payment, originalVersion []byte) (Payment, error) {
	db = repository.useDB(db)
	result := db.WithContext(ctx).Model(&paymentModel{}).
		Where("supplier_payment_id = ? AND row_version = ? AND lifecycle_status = 'Active'", payment.ID, originalVersion).
		Updates(map[string]interface{}{
			"supplier_id": payment.SupplierID, "check_number": payment.CheckNumber,
			"issue_date_utc": payment.IssueDateUTC, "due_date_utc": payment.DueDateUTC,
			"amount_scaled": payment.Amount.Int64(), "updated_at_utc": gorm.Expr("SYSUTCDATETIME()"),
		})
	if result.Error != nil {
		return Payment{}, fmt.Errorf("update supplier payment: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return Payment{}, ErrConflict
	}
	return repository.FindByID(ctx, db, payment.ID)
}

func (repository *GormRepository) Void(ctx context.Context, db *gorm.DB, id int64, originalVersion []byte) error {
	db = repository.useDB(db)
	result := db.WithContext(ctx).Model(&paymentModel{}).
		Where("supplier_payment_id = ? AND row_version = ? AND lifecycle_status = 'Active'", id, originalVersion).
		Updates(map[string]interface{}{"lifecycle_status": "Voided", "updated_at_utc": gorm.Expr("SYSUTCDATETIME()")})
	if result.Error != nil {
		return fmt.Errorf("void supplier payment: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return ErrConflict
	}
	return nil
}

func (repository *GormRepository) useDB(db *gorm.DB) *gorm.DB {
	if db == nil {
		return repository.db
	}
	return db
}
