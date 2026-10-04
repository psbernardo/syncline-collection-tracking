package supplierpayments

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/slices/suppliers"
	"gorm.io/gorm"
)

type SupplierReader interface {
	List(context.Context, bool) ([]suppliers.Supplier, error)
	FindByID(context.Context, *gorm.DB, int64) (suppliers.Supplier, error)
}

type Input struct {
	SupplierID  int64
	CheckNumber string
	IssueDate   string
	DueDate     string
	Amount      string
}

type Service struct {
	db        *gorm.DB
	repo      Repository
	suppliers SupplierReader
	now       func() time.Time
}

func NewService(db *gorm.DB, repository Repository, supplierReader SupplierReader) *Service {
	return &Service{db: db, repo: repository, suppliers: supplierReader, now: func() time.Time { return time.Now().UTC() }}
}

func (service *Service) List(ctx context.Context, includeVoided bool) (ListResult, error) {
	payments, err := service.repo.List(ctx)
	if err != nil {
		return ListResult{}, err
	}
	return buildListResult(payments, service.now(), includeVoided), nil
}

func (service *Service) Get(ctx context.Context, id int64) (Payment, error) {
	return service.repo.FindByID(ctx, nil, id)
}

func (service *Service) Suppliers(ctx context.Context) ([]SupplierOption, error) {
	items, err := service.suppliers.List(ctx, true)
	if err != nil {
		return nil, err
	}
	options := make([]SupplierOption, 0, len(items))
	for _, item := range items {
		options = append(options, SupplierOption{ID: item.ID, Name: item.Name})
	}
	return options, nil
}

func (service *Service) Create(ctx context.Context, input Input) (Payment, error) {
	payment, err := NewPayment(input.SupplierID, input.CheckNumber, input.IssueDate, input.DueDate, input.Amount)
	if err != nil {
		return Payment{}, err
	}
	var created Payment
	err = service.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := service.requireSupplier(ctx, tx, payment.SupplierID, false); err != nil {
			return err
		}
		duplicate, err := service.repo.HasDuplicateCheckNumber(ctx, tx, payment.CheckNumber, 0)
		if err != nil {
			return err
		}
		if duplicate {
			return ErrDuplicateCheck
		}
		created, err = service.repo.Create(ctx, tx, payment)
		if err != nil {
			return err
		}
		return service.audit(tx, "supplier_payment", created.ID, "create", nil, created)
	})
	if isDuplicateIndexError(err) {
		return Payment{}, ErrDuplicateCheck
	}
	return created, err
}

func (service *Service) Update(ctx context.Context, id int64, input Input, originalVersion []byte) (Payment, error) {
	payment, err := NewPayment(input.SupplierID, input.CheckNumber, input.IssueDate, input.DueDate, input.Amount)
	if err != nil {
		return Payment{}, err
	}
	payment.ID = id
	payment.RowVersion = originalVersion
	var updated Payment
	err = service.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		previous, err := service.repo.FindByID(ctx, tx, id)
		if err != nil {
			return err
		}
		if previous.Status != "Active" {
			return ErrVoided
		}
		if err := service.requireSupplier(ctx, tx, payment.SupplierID, payment.SupplierID == previous.SupplierID); err != nil {
			return err
		}
		duplicate, err := service.repo.HasDuplicateCheckNumber(ctx, tx, payment.CheckNumber, id)
		if err != nil {
			return err
		}
		if duplicate {
			return ErrDuplicateCheck
		}
		updated, err = service.repo.Update(ctx, tx, payment, originalVersion)
		if err != nil {
			return err
		}
		return service.audit(tx, "supplier_payment", updated.ID, "update", previous, updated)
	})
	if isDuplicateIndexError(err) {
		return Payment{}, ErrDuplicateCheck
	}
	return updated, err
}

func (service *Service) Void(ctx context.Context, id int64, originalVersion []byte) error {
	return service.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		previous, err := service.repo.FindByID(ctx, tx, id)
		if err != nil {
			return err
		}
		if previous.Status != "Active" {
			return ErrVoided
		}
		voided := previous
		voided.Status = "Voided"
		if err := service.repo.Void(ctx, tx, id, originalVersion); err != nil {
			return err
		}
		return service.audit(tx, "supplier_payment", id, "void", previous, voided)
	})
}

func (service *Service) requireSupplier(ctx context.Context, tx *gorm.DB, id int64, allowInactive bool) error {
	supplier, err := service.suppliers.FindByID(ctx, tx, id)
	if err != nil {
		if errors.Is(err, suppliers.ErrSupplierNotFound) {
			return ErrSupplierNotFound
		}
		return err
	}
	if !supplier.IsActive && !allowInactive {
		return ErrSupplierNotFound
	}
	return nil
}

func (service *Service) audit(tx *gorm.DB, entityType string, entityID int64, action string, previous, next interface{}) error {
	requestID, err := randomID()
	if err != nil {
		return err
	}
	idempotencyKey, err := randomID()
	if err != nil {
		return err
	}
	before, _ := json.Marshal(previous)
	after, _ := json.Marshal(next)
	return tx.Table("dbo.audit_events").Create(map[string]interface{}{
		"entity_type": entityType, "entity_id": entityID, "action": action, "actor_id": "local-admin",
		"occurred_at_utc": service.now(), "request_id": requestID, "idempotency_key": idempotencyKey,
		"previous_values_json": string(before), "new_values_json": string(after),
	}).Error
}

func randomID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate supplier payment audit identifier: %w", err)
	}
	return hex.EncodeToString(value), nil
}

func isDuplicateIndexError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "ux_supplier_payments_check_number")
}
