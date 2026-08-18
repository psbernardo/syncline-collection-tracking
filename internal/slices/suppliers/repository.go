package suppliers

import (
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
)

type Repository interface {
	Create(context.Context, *gorm.DB, Supplier) (Supplier, error)
	List(context.Context, bool) ([]Supplier, error)
	FindByID(context.Context, *gorm.DB, int64) (Supplier, error)
	Update(context.Context, *gorm.DB, Supplier, []byte) (Supplier, error)
	CreateProduct(context.Context, *gorm.DB, SupplierProduct) (SupplierProduct, error)
	ListProducts(context.Context, bool) ([]SupplierProduct, error)
	UpdateProduct(context.Context, *gorm.DB, SupplierProduct, []byte) (SupplierProduct, error)
	FindProduct(context.Context, *gorm.DB, int64) (SupplierProduct, error)
}
type GormRepository struct{ db *gorm.DB }

func NewGormRepository(db *gorm.DB) *GormRepository { return &GormRepository{db} }
func (r *GormRepository) Create(ctx context.Context, db *gorm.DB, s Supplier) (Supplier, error) {
	m := supplierModel{ID: s.ID, Name: s.Name, ContactPerson: s.ContactPerson, ContactNumber: s.ContactNumber, Email: s.Email, BillingAddress: s.BillingAddress, DeliveryAddress: s.DeliveryAddress, TaxIdentifier: s.TaxIdentifier, IsActive: true}
	if e := db.WithContext(ctx).Omit("RowVersion").Create(&m).Error; e != nil {
		return Supplier{}, fmt.Errorf("create supplier: %w", e)
	}
	return m.toDomain(), nil
}
func (r *GormRepository) List(ctx context.Context, active bool) ([]Supplier, error) {
	var ms []supplierModel
	q := r.db.WithContext(ctx)
	if active {
		q = q.Where("is_active = ?", true)
	}
	if e := q.Order("name, supplier_id").Find(&ms).Error; e != nil {
		return nil, e
	}
	o := make([]Supplier, 0, len(ms))
	for _, m := range ms {
		o = append(o, m.toDomain())
	}
	return o, nil
}
func (r *GormRepository) FindByID(ctx context.Context, db *gorm.DB, id int64) (Supplier, error) {
	if db == nil {
		db = r.db
	}
	var m supplierModel
	if e := db.WithContext(ctx).First(&m, "supplier_id = ?", id).Error; e != nil {
		if errors.Is(e, gorm.ErrRecordNotFound) {
			return Supplier{}, ErrSupplierNotFound
		}
		return Supplier{}, e
	}
	return m.toDomain(), nil
}
func (r *GormRepository) Update(ctx context.Context, db *gorm.DB, s Supplier, v []byte) (Supplier, error) {
	q := db.WithContext(ctx).Model(&supplierModel{}).Where("supplier_id = ? AND row_version = ?", s.ID, v).Updates(map[string]interface{}{"name": s.Name, "contact_person": s.ContactPerson, "contact_number": s.ContactNumber, "email": s.Email, "billing_address": s.BillingAddress, "delivery_address": s.DeliveryAddress, "tax_identifier": s.TaxIdentifier, "is_active": s.IsActive, "updated_at_utc": gorm.Expr("SYSUTCDATETIME()")})
	if q.Error != nil {
		return Supplier{}, q.Error
	}
	if q.RowsAffected != 1 {
		return Supplier{}, ErrSupplierConflict
	}
	return r.FindByID(ctx, db, s.ID)
}
func (r *GormRepository) CreateProduct(ctx context.Context, db *gorm.DB, p SupplierProduct) (SupplierProduct, error) {
	var count int64
	if e := db.WithContext(ctx).Table("dbo.supplier_products").Where("supplier_id = ? AND product_id = ?", p.SupplierID, p.ProductID).Count(&count).Error; e != nil {
		return SupplierProduct{}, e
	}
	if count == 1 {
		return SupplierProduct{}, ErrDuplicateSupplierProduct
	}
	if e := db.WithContext(ctx).Table("dbo.suppliers").Where("supplier_id = ? AND is_active = 1", p.SupplierID).Count(&count).Error; e != nil {
		return SupplierProduct{}, e
	}
	if count != 1 {
		return SupplierProduct{}, ErrSupplierNotFound
	}
	if e := db.WithContext(ctx).Table("dbo.products").Where("product_id = ? AND is_active = 1", p.ProductID).Count(&count).Error; e != nil {
		return SupplierProduct{}, e
	}
	if count != 1 {
		return SupplierProduct{}, ErrProductNotFound
	}
	m := supplierProductModel{SupplierID: p.SupplierID, ProductID: p.ProductID, SupplierSKU: p.SupplierSKU, ReferenceCost: p.ReferenceCost.Int64(), IsActive: true}
	if e := db.WithContext(ctx).Omit("RowVersion", "SupplierName", "ProductSKU", "ProductName", "UpdatedAtUTC").Create(&m).Error; e != nil {
		return SupplierProduct{}, fmt.Errorf("create supplier product: %w", e)
	}
	return r.product(ctx, db, m.ID)
}
func (r *GormRepository) ListProducts(ctx context.Context, active bool) ([]SupplierProduct, error) {
	var ms []supplierProductModel
	q := r.db.WithContext(ctx).Table("dbo.supplier_products AS sp").Select("sp.supplier_product_id, sp.supplier_id, sp.product_id, s.name supplier_name, p.sku product_sku, p.name product_name, sp.supplier_sku, sp.reference_cost_scaled, sp.is_active, sp.updated_at_utc, sp.row_version").Joins("JOIN dbo.suppliers s ON s.supplier_id = sp.supplier_id").Joins("JOIN dbo.products p ON p.product_id = sp.product_id")
	if active {
		q = q.Where("sp.is_active = 1 AND s.is_active = 1 AND p.is_active = 1")
	}
	if e := q.Order("p.name, s.name").Find(&ms).Error; e != nil {
		return nil, e
	}
	o := make([]SupplierProduct, 0, len(ms))
	for _, m := range ms {
		o = append(o, m.toDomain())
	}
	return o, nil
}
func (r *GormRepository) product(ctx context.Context, db *gorm.DB, id int64) (SupplierProduct, error) {
	var m supplierProductModel
	q := db.WithContext(ctx).Table("dbo.supplier_products AS sp").Select("sp.supplier_product_id, sp.supplier_id, sp.product_id, s.name supplier_name, p.sku product_sku, p.name product_name, sp.supplier_sku, sp.reference_cost_scaled, sp.is_active, sp.updated_at_utc, sp.row_version").Joins("JOIN dbo.suppliers s ON s.supplier_id = sp.supplier_id").Joins("JOIN dbo.products p ON p.product_id = sp.product_id").Where("sp.supplier_product_id = ?", id)
	if e := q.First(&m).Error; e != nil {
		if errors.Is(e, gorm.ErrRecordNotFound) {
			return SupplierProduct{}, ErrSupplierProductNotFound
		}
		return SupplierProduct{}, e
	}
	return m.toDomain(), nil
}
func (r *GormRepository) UpdateProduct(ctx context.Context, db *gorm.DB, p SupplierProduct, v []byte) (SupplierProduct, error) {
	q := db.WithContext(ctx).Model(&supplierProductModel{}).Where("supplier_product_id = ? AND row_version = ?", p.ID, v).Updates(map[string]interface{}{"supplier_sku": p.SupplierSKU, "reference_cost_scaled": p.ReferenceCost.Int64(), "is_active": p.IsActive, "updated_at_utc": gorm.Expr("SYSUTCDATETIME()")})
	if q.Error != nil {
		return SupplierProduct{}, q.Error
	}
	if q.RowsAffected != 1 {
		return SupplierProduct{}, ErrSupplierConflict
	}
	return r.product(ctx, db, p.ID)
}

func (r *GormRepository) FindProduct(ctx context.Context, db *gorm.DB, id int64) (SupplierProduct, error) {
	return r.product(ctx, db, id)
}
