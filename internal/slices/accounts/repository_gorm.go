package accounts

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

type GormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) *GormRepository { return &GormRepository{db: db} }

func (repository *GormRepository) Create(ctx context.Context, db *gorm.DB, account CompanyAccount) (CompanyAccount, error) {
	model := toModel(account)
	if err := db.WithContext(ctx).Omit("RowVersion").Create(&model).Error; err != nil {
		return CompanyAccount{}, fmt.Errorf("create company account: %w", err)
	}
	return model.toDomain(), nil
}

func (repository *GormRepository) List(ctx context.Context) ([]CompanyAccount, error) {
	var models []accountModel
	if err := repository.db.WithContext(ctx).
		Select("company_account_id, company_name, contact_person, tin_number, billing_address, delivery_address, contact_number, email, created_at_utc, updated_at_utc, row_version").
		Order("company_name, company_account_id").
		Find(&models).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []CompanyAccount{}, nil
		}
		return nil, fmt.Errorf("list company accounts: %w", err)
	}
	accounts := make([]CompanyAccount, 0, len(models))
	for _, model := range models {
		accounts = append(accounts, model.toDomain())
	}
	return accounts, nil
}

func (repository *GormRepository) FindByID(ctx context.Context, db *gorm.DB, id int64) (CompanyAccount, error) {
	if db == nil {
		db = repository.db
	}
	var model accountModel
	if err := db.WithContext(ctx).First(&model, "company_account_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return CompanyAccount{}, ErrAccountNotFound
		}
		return CompanyAccount{}, fmt.Errorf("find company account: %w", err)
	}
	return model.toDomain(), nil
}

func (repository *GormRepository) Exists(ctx context.Context, db *gorm.DB, id int64) (bool, error) {
	if db == nil {
		db = repository.db
	}
	var count int64
	if err := db.WithContext(ctx).Model(&accountModel{}).Where("company_account_id = ?", id).Count(&count).Error; err != nil {
		return false, fmt.Errorf("check company account: %w", err)
	}
	return count == 1, nil
}

func (repository *GormRepository) Update(ctx context.Context, db *gorm.DB, account CompanyAccount, originalVersion []byte) (CompanyAccount, error) {
	result := db.WithContext(ctx).Model(&accountModel{}).
		Where("company_account_id = ? AND row_version = ?", account.ID, originalVersion).
		Updates(map[string]interface{}{
			"company_name":     account.CompanyName,
			"contact_person":   account.ContactPerson,
			"tin_number":       account.TINNumber,
			"billing_address":  account.BillingAddress,
			"delivery_address": account.DeliveryAddress,
			"contact_number":   account.ContactNumber,
			"email":            account.Email,
			"updated_at_utc":   gorm.Expr("SYSUTCDATETIME()"),
		})
	if result.Error != nil {
		return CompanyAccount{}, fmt.Errorf("update company account: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return CompanyAccount{}, ErrAccountConflict
	}
	var model accountModel
	if err := db.WithContext(ctx).First(&model, "company_account_id = ?", account.ID).Error; err != nil {
		return CompanyAccount{}, fmt.Errorf("reload updated company account: %w", err)
	}
	return model.toDomain(), nil
}
