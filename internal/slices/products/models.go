package products

import "time"

type productModel struct {
	ID                          int64 `gorm:"column:product_id;primaryKey"`
	SKU, Name, Description, UOM string
	IsActive                    bool      `gorm:"column:is_active"`
	CreatedAtUTC                time.Time `gorm:"column:created_at_utc"`
	UpdatedAtUTC                time.Time `gorm:"column:updated_at_utc"`
	RowVersion                  []byte    `gorm:"column:row_version;->"`
}

func (productModel) TableName() string { return "dbo.products" }
func (m productModel) toDomain() Product {
	return Product{m.ID, m.SKU, m.Name, m.Description, m.UOM, m.IsActive, m.CreatedAtUTC, m.UpdatedAtUTC, m.RowVersion}
}
func toModel(p Product) productModel {
	return productModel{ID: p.ID, SKU: p.SKU, Name: p.Name, Description: p.Description, UOM: p.UOM, IsActive: p.IsActive, CreatedAtUTC: p.CreatedAtUTC, UpdatedAtUTC: p.UpdatedAtUTC, RowVersion: p.RowVersion}
}
