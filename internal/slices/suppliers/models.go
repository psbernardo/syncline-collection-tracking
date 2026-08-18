package suppliers

import (
	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
	"time"
)

type supplierModel struct {
	ID                                                                                        int64 `gorm:"column:supplier_id;primaryKey"`
	Name, ContactPerson, ContactNumber, Email, BillingAddress, DeliveryAddress, TaxIdentifier string
	IsActive                                                                                  bool `gorm:"column:is_active"`
	CreatedAtUTC, UpdatedAtUTC                                                                time.Time
	RowVersion                                                                                []byte `gorm:"column:row_version;->"`
}

func (supplierModel) TableName() string { return "dbo.suppliers" }
func (m supplierModel) toDomain() Supplier {
	return Supplier{m.ID, m.Name, m.ContactPerson, m.ContactNumber, m.Email, m.BillingAddress, m.DeliveryAddress, m.TaxIdentifier, m.IsActive, m.CreatedAtUTC, m.UpdatedAtUTC, m.RowVersion}
}

type supplierProductModel struct {
	ID                                                 int64 `gorm:"column:supplier_product_id;primaryKey"`
	SupplierID                                         int64 `gorm:"column:supplier_id"`
	ProductID                                          int64 `gorm:"column:product_id"`
	SupplierName, ProductSKU, ProductName, SupplierSKU string
	ReferenceCost                                      int64     `gorm:"column:reference_cost_scaled"`
	IsActive                                           bool      `gorm:"column:is_active"`
	UpdatedAtUTC                                       time.Time `gorm:"column:updated_at_utc"`
	RowVersion                                         []byte    `gorm:"column:row_version;->"`
}

func (supplierProductModel) TableName() string { return "dbo.supplier_products" }
func (m supplierProductModel) toDomain() SupplierProduct {
	return SupplierProduct{m.ID, m.SupplierID, m.ProductID, m.SupplierName, m.ProductSKU, m.ProductName, m.SupplierSKU, money.Amount(m.ReferenceCost), m.IsActive, m.UpdatedAtUTC, m.RowVersion}
}
