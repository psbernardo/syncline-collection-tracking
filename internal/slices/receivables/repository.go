package receivables

import (
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, db *gorm.DB, receivable DeliveryReceivable) (DeliveryReceivable, error)
	FindByID(ctx context.Context, db *gorm.DB, id int64) (DeliveryReceivable, error)
	List(ctx context.Context) ([]DeliveryReceivable, error)
	ListFiltered(ctx context.Context, query ListQuery) ([]DeliveryReceivable, int64, error)
	Update(ctx context.Context, db *gorm.DB, receivable DeliveryReceivable, originalVersion []byte) (DeliveryReceivable, error)
	FindBlockingPONumber(ctx context.Context, db *gorm.DB, normalizedPO string, excludeID int64) (bool, error)
	FindBlockingInvoiceNumber(ctx context.Context, db *gorm.DB, invoiceNumber string, excludeID int64) (bool, error)
}
