package products

import (
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
)

type Repository interface {
	Create(context.Context, *gorm.DB, Product) (Product, error)
	List(context.Context, bool) ([]Product, error)
	FindByID(context.Context, *gorm.DB, int64) (Product, error)
	Update(context.Context, *gorm.DB, Product, []byte) (Product, error)
}
type GormRepository struct{ db *gorm.DB }

func NewGormRepository(db *gorm.DB) *GormRepository { return &GormRepository{db} }
func (r *GormRepository) Create(ctx context.Context, db *gorm.DB, p Product) (Product, error) {
	m := toModel(p)
	if err := db.WithContext(ctx).Omit("RowVersion").Create(&m).Error; err != nil {
		return Product{}, fmt.Errorf("create product: %w", err)
	}
	return m.toDomain(), nil
}
func (r *GormRepository) List(ctx context.Context, active bool) ([]Product, error) {
	var ms []productModel
	q := r.db.WithContext(ctx).Select("product_id, sku, name, description, uom, is_active, created_at_utc, updated_at_utc, row_version")
	if active {
		q = q.Where("is_active = ?", true)
	}
	if err := q.Order("name, product_id").Find(&ms).Error; err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	out := make([]Product, 0, len(ms))
	for _, m := range ms {
		out = append(out, m.toDomain())
	}
	return out, nil
}
func (r *GormRepository) FindByID(ctx context.Context, db *gorm.DB, id int64) (Product, error) {
	if db == nil {
		db = r.db
	}
	var m productModel
	if err := db.WithContext(ctx).First(&m, "product_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Product{}, ErrNotFound
		}
		return Product{}, err
	}
	return m.toDomain(), nil
}
func (r *GormRepository) Update(ctx context.Context, db *gorm.DB, p Product, v []byte) (Product, error) {
	var current productModel
	if err := db.WithContext(ctx).First(&current, "product_id = ?", p.ID).Error; err != nil {
		return Product{}, err
	}
	if current.UOM != p.UOM {
		var references int64
		if err := db.WithContext(ctx).Raw("SELECT COUNT_BIG(*) FROM dbo.quotation_lines WHERE product_id = ?", p.ID).Scan(&references).Error; err != nil {
			return Product{}, err
		}
		if references > 0 {
			return Product{}, ErrUOMLocked
		}
	}
	res := db.WithContext(ctx).Model(&productModel{}).Where("product_id = ? AND row_version = ?", p.ID, v).Updates(map[string]interface{}{"sku": p.SKU, "name": p.Name, "description": p.Description, "uom": p.UOM, "is_active": p.IsActive, "updated_at_utc": gorm.Expr("SYSUTCDATETIME()")})
	if res.Error != nil {
		return Product{}, res.Error
	}
	if res.RowsAffected != 1 {
		return Product{}, ErrConflict
	}
	return r.FindByID(ctx, db, p.ID)
}
