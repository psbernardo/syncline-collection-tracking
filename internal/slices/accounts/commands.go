package accounts

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

var ErrIdempotencyConflict = errors.New("idempotency key was already used with a different request")

type CreateAccountCommand struct {
	CompanyName     string
	ContactPerson   string
	TINNumber       string
	BillingAddress  string
	DeliveryAddress string
	ContactNumber   string
	RequestID       string
	IdempotencyKey  string
	ActorID         string
}

type UpdateAccountCommand struct {
	ID              int64
	CompanyName     string
	ContactPerson   string
	TINNumber       string
	BillingAddress  string
	DeliveryAddress string
	ContactNumber   string
	OriginalVersion []byte
	RequestID       string
	IdempotencyKey  string
	ActorID         string
}

type commandService struct {
	db         *gorm.DB
	repository Repository
	now        func() time.Time
}

func NewService(db *gorm.DB, repository Repository) *commandService {
	return &commandService{db: db, repository: repository, now: func() time.Time { return time.Now().UTC() }}
}

func (service *commandService) Create(ctx context.Context, command CreateAccountCommand) (CompanyAccount, error) {
	account, err := NewCompanyAccount(CompanyAccount{
		CompanyName:     command.CompanyName,
		ContactPerson:   command.ContactPerson,
		TINNumber:       command.TINNumber,
		BillingAddress:  command.BillingAddress,
		DeliveryAddress: command.DeliveryAddress,
		ContactNumber:   command.ContactNumber,
	})
	if err != nil {
		return CompanyAccount{}, err
	}
	if command.IdempotencyKey == "" {
		return CompanyAccount{}, fmt.Errorf("idempotency key is required")
	}

	payloadHash := hashPayload(account)
	var result CompanyAccount
	err = service.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing idempotencyModel
		findErr := tx.Where("idempotency_key = ?", command.IdempotencyKey).First(&existing).Error
		if findErr == nil {
			if existing.RequestHash != payloadHash {
				return ErrIdempotencyConflict
			}
			if existing.ResultEntityID == nil {
				return fmt.Errorf("idempotency request has no result")
			}
			var model accountModel
			if err := tx.First(&model, *existing.ResultEntityID).Error; err != nil {
				return fmt.Errorf("load idempotent result: %w", err)
			}
			result = model.toDomain()
			return nil
		}
		if !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return fmt.Errorf("check idempotency key: %w", findErr)
		}

		idempotency := idempotencyModel{
			Key:          command.IdempotencyKey,
			CommandType:  "create_company_account",
			RequestHash:  payloadHash,
			ResponseCode: 303,
			CreatedAtUTC: service.now(),
			ExpiresAtUTC: service.now().Add(24 * time.Hour),
		}
		if err := tx.Create(&idempotency).Error; err != nil {
			return fmt.Errorf("create idempotency record: %w", err)
		}
		account, err = service.repository.Create(ctx, tx, account)
		if err != nil {
			return err
		}
		result = account
		idempotency.ResultEntityID = &account.ID
		if err := tx.Model(&idempotency).Updates(map[string]interface{}{"result_entity_id": account.ID}).Error; err != nil {
			return fmt.Errorf("save idempotency result: %w", err)
		}
		before, _ := json.Marshal(nil)
		after, _ := json.Marshal(account)
		if err := tx.Create(&auditModel{
			EntityType:     "company_account",
			EntityID:       account.ID,
			Action:         "create",
			ActorID:        command.ActorID,
			OccurredAtUTC:  service.now(),
			RequestID:      command.RequestID,
			IdempotencyKey: command.IdempotencyKey,
			PreviousValues: string(before),
			NewValues:      string(after),
		}).Error; err != nil {
			return fmt.Errorf("create audit event: %w", err)
		}
		return nil
	})
	if err != nil {
		return CompanyAccount{}, err
	}
	return result, nil
}

func (service *commandService) Update(ctx context.Context, command UpdateAccountCommand) (CompanyAccount, error) {
	account, err := NewCompanyAccount(CompanyAccount{
		ID:              command.ID,
		CompanyName:     command.CompanyName,
		ContactPerson:   command.ContactPerson,
		TINNumber:       command.TINNumber,
		BillingAddress:  command.BillingAddress,
		DeliveryAddress: command.DeliveryAddress,
		ContactNumber:   command.ContactNumber,
		RowVersion:      command.OriginalVersion,
	})
	if err != nil {
		return CompanyAccount{}, err
	}
	if command.IdempotencyKey == "" {
		return CompanyAccount{}, fmt.Errorf("idempotency key is required")
	}
	payloadHash := hashPayload(account)
	var result CompanyAccount
	err = service.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing idempotencyModel
		findErr := tx.Where("idempotency_key = ?", command.IdempotencyKey).First(&existing).Error
		if findErr == nil {
			if existing.RequestHash != payloadHash {
				return ErrIdempotencyConflict
			}
			if existing.ResultEntityID == nil {
				return fmt.Errorf("idempotency request has no result")
			}
			returnValue, loadErr := service.repository.FindByID(ctx, tx, *existing.ResultEntityID)
			if loadErr != nil {
				return loadErr
			}
			result = returnValue
			return nil
		}
		if !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return fmt.Errorf("check idempotency key: %w", findErr)
		}

		idempotency := idempotencyModel{
			Key: command.IdempotencyKey, CommandType: "update_company_account", RequestHash: payloadHash,
			ResponseCode: 303, CreatedAtUTC: service.now(), ExpiresAtUTC: service.now().Add(24 * time.Hour),
		}
		if err := tx.Create(&idempotency).Error; err != nil {
			return fmt.Errorf("create idempotency record: %w", err)
		}
		previous, err := service.repository.FindByID(ctx, tx, command.ID)
		if err != nil {
			return err
		}
		updated, err := service.repository.Update(ctx, tx, account, command.OriginalVersion)
		if err != nil {
			return err
		}
		result = updated
		if err := tx.Model(&idempotency).Updates(map[string]interface{}{"result_entity_id": updated.ID}).Error; err != nil {
			return fmt.Errorf("save idempotency result: %w", err)
		}
		before, _ := json.Marshal(previous)
		after, _ := json.Marshal(updated)
		if err := tx.Create(&auditModel{
			EntityType: "company_account", EntityID: updated.ID, Action: "update", ActorID: command.ActorID,
			OccurredAtUTC: service.now(), RequestID: command.RequestID, IdempotencyKey: command.IdempotencyKey,
			PreviousValues: string(before), NewValues: string(after),
		}).Error; err != nil {
			return fmt.Errorf("create audit event: %w", err)
		}
		return nil
	})
	if err != nil {
		return CompanyAccount{}, err
	}
	return result, nil
}

func hashPayload(account CompanyAccount) string {
	payload, _ := json.Marshal(account)
	hash := sha256.Sum256(payload)
	return hex.EncodeToString(hash[:])
}

func NewIdempotencyKey() (string, error) {
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

type idempotencyModel struct {
	Key            string    `gorm:"column:idempotency_key;primaryKey"`
	CommandType    string    `gorm:"column:command_type"`
	RequestHash    string    `gorm:"column:request_hash"`
	ResultEntityID *int64    `gorm:"column:result_entity_id"`
	ResponseCode   int       `gorm:"column:response_status"`
	CreatedAtUTC   time.Time `gorm:"column:created_at_utc"`
	ExpiresAtUTC   time.Time `gorm:"column:expires_at_utc"`
}

func (idempotencyModel) TableName() string { return "dbo.idempotency_keys" }

type auditModel struct {
	EventID        int64     `gorm:"column:event_id;primaryKey"`
	EntityType     string    `gorm:"column:entity_type"`
	EntityID       int64     `gorm:"column:entity_id"`
	Action         string    `gorm:"column:action"`
	ActorID        string    `gorm:"column:actor_id"`
	OccurredAtUTC  time.Time `gorm:"column:occurred_at_utc"`
	RequestID      string    `gorm:"column:request_id"`
	IdempotencyKey string    `gorm:"column:idempotency_key"`
	PreviousValues string    `gorm:"column:previous_values_json"`
	NewValues      string    `gorm:"column:new_values_json"`
}

func (auditModel) TableName() string { return "dbo.audit_events" }
