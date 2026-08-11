package accounts

import (
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, db *gorm.DB, account CompanyAccount) (CompanyAccount, error)
	List(ctx context.Context) ([]CompanyAccount, error)
	FindByID(ctx context.Context, db *gorm.DB, id int64) (CompanyAccount, error)
	Exists(ctx context.Context, db *gorm.DB, id int64) (bool, error)
	Update(ctx context.Context, db *gorm.DB, account CompanyAccount, originalVersion []byte) (CompanyAccount, error)
}
