package suppliers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
	"gorm.io/gorm"
	"time"
)

type CreateCommand struct{ Name, ContactPerson, ContactNumber, Email, BillingAddress, DeliveryAddress, TaxIdentifier, RequestID, IdempotencyKey, ActorID string }
type UpdateCommand struct {
	Supplier
	OriginalVersion                    []byte
	RequestID, IdempotencyKey, ActorID string
}
type CreateProductCommand struct {
	SupplierID, ProductID                                          int64
	SupplierSKU, ReferenceCost, RequestID, IdempotencyKey, ActorID string
}
type UpdateProductCommand struct {
	SupplierProduct
	OriginalVersion                    []byte
	RequestID, IdempotencyKey, ActorID string
}
type ProductOption struct {
	ID             int64
	SKU, Name, UOM string
}
type ProductOptionLoader func(context.Context) ([]ProductOption, error)
type Service struct {
	db             *gorm.DB
	repo           Repository
	now            func() time.Time
	productOptions ProductOptionLoader
}

func NewService(db *gorm.DB, r Repository, loaders ...ProductOptionLoader) *Service {
	var loader ProductOptionLoader
	if len(loaders) > 0 {
		loader = loaders[0]
	}
	return &Service{db: db, repo: r, now: func() time.Time { return time.Now().UTC() }, productOptions: loader}
}
func (s *Service) List(ctx context.Context, active bool) ([]Supplier, error) {
	return s.repo.List(ctx, active)
}
func (s *Service) Get(ctx context.Context, id int64) (Supplier, error) {
	return s.repo.FindByID(ctx, nil, id)
}
func (s *Service) Products(ctx context.Context, active bool) ([]SupplierProduct, error) {
	return s.repo.ListProducts(ctx, active)
}
func (s *Service) GetProduct(ctx context.Context, id int64) (SupplierProduct, error) {
	return s.repo.FindProduct(ctx, nil, id)
}
func (s *Service) FormOptions(ctx context.Context) ([]Supplier, []ProductOption, error) {
	suppliers, err := s.repo.List(ctx, true)
	if err != nil {
		return nil, nil, err
	}
	if s.productOptions == nil {
		return suppliers, []ProductOption{}, nil
	}
	products, err := s.productOptions(ctx)
	if err != nil {
		return nil, nil, err
	}
	return suppliers, products, nil
}
func (s *Service) Create(ctx context.Context, c CreateCommand) (Supplier, error) {
	v, e := NewSupplier(Supplier{Name: c.Name, ContactPerson: c.ContactPerson, ContactNumber: c.ContactNumber, Email: c.Email, BillingAddress: c.BillingAddress, DeliveryAddress: c.DeliveryAddress, TaxIdentifier: c.TaxIdentifier})
	if e != nil {
		return Supplier{}, e
	}
	if c.IdempotencyKey == "" {
		return Supplier{}, fmt.Errorf("idempotency key is required")
	}
	var o Supplier
	e = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		o, e = s.repo.Create(ctx, tx, v)
		if e != nil {
			return e
		}
		return audit(tx, "supplier", o.ID, "create", c.ActorID, c.RequestID, c.IdempotencyKey, nil, o, s.now())
	})
	return o, e
}
func (s *Service) Update(ctx context.Context, c UpdateCommand) (Supplier, error) {
	v, e := NewSupplier(c.Supplier)
	if e != nil {
		return Supplier{}, e
	}
	var o Supplier
	e = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		old, e := s.repo.FindByID(ctx, tx, c.ID)
		if e != nil {
			return e
		}
		o, e = s.repo.Update(ctx, tx, v, c.OriginalVersion)
		if e != nil {
			return e
		}
		return audit(tx, "supplier", o.ID, "update", c.ActorID, c.RequestID, c.IdempotencyKey, old, o, s.now())
	})
	return o, e
}
func (s *Service) CreateProduct(ctx context.Context, c CreateProductCommand) (SupplierProduct, error) {
	cost, e := money.Parse(c.ReferenceCost)
	if e != nil {
		return SupplierProduct{}, ValidationErrors{"ReferenceCost": e.Error()}
	}
	v, e := NewSupplierProduct(SupplierProduct{SupplierID: c.SupplierID, ProductID: c.ProductID, ReferenceCost: cost})
	if e != nil {
		return SupplierProduct{}, e
	}
	if c.IdempotencyKey == "" {
		return SupplierProduct{}, fmt.Errorf("idempotency key is required")
	}
	var o SupplierProduct
	e = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		o, e = s.repo.CreateProduct(ctx, tx, v)
		if e != nil {
			return e
		}
		return audit(tx, "supplier_product", o.ID, "create", c.ActorID, c.RequestID, c.IdempotencyKey, nil, o, s.now())
	})
	return o, e
}
func (s *Service) UpdateProduct(ctx context.Context, c UpdateProductCommand) (SupplierProduct, error) {
	v, e := NewSupplierProduct(c.SupplierProduct)
	if e != nil {
		return SupplierProduct{}, e
	}
	var o SupplierProduct
	e = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		old, e := s.repo.FindProduct(ctx, tx, c.ID)
		if e != nil {
			return e
		}
		updated, e := s.repo.UpdateProduct(ctx, tx, v, c.OriginalVersion)
		if e != nil {
			return e
		}
		o = updated
		return audit(tx, "supplier_product", o.ID, "update", c.ActorID, c.RequestID, c.IdempotencyKey, old, o, s.now())
	})
	return o, e
}

type auditRow struct {
	EntityType     string    `gorm:"column:entity_type"`
	EntityID       int64     `gorm:"column:entity_id"`
	Action         string    `gorm:"column:action"`
	ActorID        string    `gorm:"column:actor_id"`
	RequestID      string    `gorm:"column:request_id"`
	IdempotencyKey string    `gorm:"column:idempotency_key"`
	PreviousValues string    `gorm:"column:previous_values_json"`
	NewValues      string    `gorm:"column:new_values_json"`
	OccurredAtUTC  time.Time `gorm:"column:occurred_at_utc"`
}

func (auditRow) TableName() string { return "dbo.audit_events" }
func audit(tx *gorm.DB, typ string, id int64, act, actor, req, key string, old, new any, t time.Time) error {
	b, _ := json.Marshal(old)
	n, _ := json.Marshal(new)
	return tx.Create(&auditRow{typ, id, act, actor, req, key, string(b), string(n), t}).Error
}
