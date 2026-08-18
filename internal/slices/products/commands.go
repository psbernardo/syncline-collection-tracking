package products

import (
	"context"
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"time"
)

type CreateCommand struct{ SKU, Name, Description, UOM, RequestID, IdempotencyKey, ActorID string }
type UpdateCommand struct {
	Product
	OriginalVersion                    []byte
	RequestID, IdempotencyKey, ActorID string
}
type Service struct {
	db   *gorm.DB
	repo Repository
	now  func() time.Time
}

func NewService(db *gorm.DB, r Repository) *Service {
	return &Service{db: db, repo: r, now: func() time.Time { return time.Now().UTC() }}
}
func (s *Service) List(ctx context.Context, active bool) ([]Product, error) {
	return s.repo.List(ctx, active)
}
func (s *Service) Get(ctx context.Context, id int64) (Product, error) {
	return s.repo.FindByID(ctx, nil, id)
}
func (s *Service) Create(ctx context.Context, c CreateCommand) (Product, error) {
	p, e := New(Product{SKU: c.SKU, Name: c.Name, Description: c.Description, UOM: c.UOM, IsActive: true})
	if e != nil {
		return Product{}, e
	}
	if c.IdempotencyKey == "" {
		return Product{}, fmt.Errorf("idempotency key is required")
	}
	var out Product
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		out, e = s.repo.Create(ctx, tx, p)
		if e != nil {
			return e
		}
		return audit(tx, "product", out.ID, "create", c.ActorID, c.RequestID, c.IdempotencyKey, nil, out, s.now())
	})
	return out, err
}
func (s *Service) Update(ctx context.Context, c UpdateCommand) (Product, error) {
	p, e := New(c.Product)
	if e != nil {
		return Product{}, e
	}
	var out Product
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		old, e := s.repo.FindByID(ctx, tx, c.ID)
		if e != nil {
			return e
		}
		out, e = s.repo.Update(ctx, tx, p, c.OriginalVersion)
		if e != nil {
			return e
		}
		return audit(tx, "product", out.ID, "update", c.ActorID, c.RequestID, c.IdempotencyKey, old, out, s.now())
	})
	return out, err
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
func audit(tx *gorm.DB, typ string, id int64, action, actor, request, key string, old, now any, t time.Time) error {
	b, _ := json.Marshal(old)
	n, _ := json.Marshal(now)
	return tx.Create(&auditRow{typ, id, action, actor, request, key, string(b), string(n), t}).Error
}
