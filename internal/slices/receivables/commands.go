package receivables

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/businessdate"
	"github.com/psbernardo/syncline-collection-tracking/internal/shared/tax"
	"github.com/psbernardo/syncline-collection-tracking/internal/slices/accounts"
	"gorm.io/gorm"
)

type AccountRepository interface {
	Exists(ctx context.Context, db *gorm.DB, id int64) (bool, error)
	List(ctx context.Context) ([]accounts.CompanyAccount, error)
}

type CreateReceivableCommand struct {
	CompanyAccountID int64
	InvoiceID        int64
	InvoiceNumber    string
	PONumber         string
	AmountInput      string
	TaxRuleCode      tax.RuleCode
	DeliveryDate     string
	PaymentTermDays  int
	RequestID        string
	IdempotencyKey   string
	ActorID          string
}

type UpdateReceivableCommand struct {
	ID               int64
	CompanyAccountID int64
	InvoiceID        int64
	InvoiceNumber    string
	PONumber         string
	AmountInput      string
	TaxRuleCode      tax.RuleCode
	DeliveryDate     string
	PaymentTermDays  int
	OriginalVersion  []byte
	RequestID        string
	IdempotencyKey   string
	ActorID          string
}

type ReceivePaymentCommand struct {
	ID              int64
	PaymentDate     string
	OriginalVersion []byte
	RequestID       string
	IdempotencyKey  string
	ActorID         string
}

type ReversePaymentAcknowledgementCommand struct {
	ID              int64
	Reason          string
	OriginalVersion []byte
	RequestID       string
	IdempotencyKey  string
	ActorID         string
}

type service struct {
	db       *gorm.DB
	repo     Repository
	accounts AccountRepository
	invoices InvoiceRepository
	now      func() time.Time
}

func NewService(db *gorm.DB, repo Repository, accountRepo AccountRepository, invoiceRepo ...InvoiceRepository) *service {
	var selected InvoiceRepository
	if len(invoiceRepo) > 0 {
		selected = invoiceRepo[0]
	}
	return &service{db: db, repo: repo, accounts: accountRepo, invoices: selected, now: func() time.Time { return time.Now().UTC() }}
}

func (service *service) Create(ctx context.Context, command CreateReceivableCommand) (DeliveryReceivable, error) {
	var invoice InvoiceOption
	var err error
	invoiceNumber := command.InvoiceNumber
	if command.InvoiceID > 0 {
		if service.invoices == nil {
			return DeliveryReceivable{}, ValidationErrors{"InvoiceNumber": "Select a valid invoice for this company."}
		}
		invoice, err = service.invoices.FindSelectable(ctx, nil, command.InvoiceID, 0)
		if err != nil {
			return DeliveryReceivable{}, err
		}
		if command.CompanyAccountID != invoice.CompanyAccountID {
			return DeliveryReceivable{}, ErrInvoiceCompanyMismatch
		}
		invoiceNumber = invoice.Number
	}
	receivable, err := NewDeliveryReceivableWithTax(command.CompanyAccountID, invoiceNumber, command.PONumber, command.AmountInput, command.DeliveryDate, command.PaymentTermDays, command.TaxRuleCode)
	if err != nil {
		return DeliveryReceivable{}, err
	}
	receivable.InvoiceID = invoice.ID
	if command.IdempotencyKey == "" {
		return DeliveryReceivable{}, fmt.Errorf("idempotency key is required")
	}
	payloadHash := hashPayload(receivable)
	var result DeliveryReceivable
	err = service.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if command.InvoiceID > 0 {
			invoice, err = service.invoices.FindSelectable(ctx, tx, command.InvoiceID, 0)
			if err != nil {
				return err
			}
			if command.CompanyAccountID != invoice.CompanyAccountID {
				return ErrInvoiceCompanyMismatch
			}
			receivable.InvoiceID = invoice.ID
			receivable.InvoiceNumber = invoice.Number
		}
		var existing idempotencyModel
		findErr := tx.Where("idempotency_key = ?", command.IdempotencyKey).First(&existing).Error
		if findErr == nil {
			if existing.RequestHash != payloadHash {
				return ErrIdempotencyConflict
			}
			if existing.ResultEntityID == nil {
				return fmt.Errorf("idempotency request has no result")
			}
			result, err = service.repo.FindByID(ctx, tx, *existing.ResultEntityID)
			return err
		}
		if !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return fmt.Errorf("check idempotency key: %w", findErr)
		}
		exists, err := service.accounts.Exists(ctx, tx, command.CompanyAccountID)
		if err != nil {
			return err
		}
		if !exists {
			return ErrCompanyNotFound
		}
		if receivable.InvoiceNumber != "" {
			duplicateInvoice, err := service.repo.FindBlockingInvoiceNumber(ctx, tx, receivable.InvoiceNumber, 0)
			if err != nil {
				return err
			}
			if duplicateInvoice {
				return ErrDuplicateInvoiceNumber
			}
		}
		duplicate, err := service.repo.FindBlockingPONumber(ctx, tx, receivable.PONumberNormalized, 0)
		if err != nil {
			return err
		}
		if duplicate {
			return ErrDuplicatePO
		}
		idempotency := idempotencyModel{Key: command.IdempotencyKey, CommandType: "create_delivery_receivable", RequestHash: payloadHash, ResponseCode: 303, CreatedAtUTC: service.now(), ExpiresAtUTC: service.now().Add(24 * time.Hour)}
		if err := tx.Create(&idempotency).Error; err != nil {
			return fmt.Errorf("create idempotency record: %w", err)
		}
		result, err = service.repo.Create(ctx, tx, receivable)
		if err != nil {
			return err
		}
		if err := tx.Model(&idempotency).Updates(map[string]interface{}{"result_entity_id": result.ID}).Error; err != nil {
			return fmt.Errorf("save idempotency result: %w", err)
		}
		after, _ := json.Marshal(result)
		if err := tx.Create(&auditModel{EntityType: "delivery_receivable", EntityID: result.ID, Action: "create", ActorID: command.ActorID, OccurredAtUTC: service.now(), RequestID: command.RequestID, IdempotencyKey: command.IdempotencyKey, PreviousValues: "null", NewValues: string(after)}).Error; err != nil {
			return fmt.Errorf("create audit event: %w", err)
		}
		return nil
	})
	if err != nil {
		if isDuplicatePOError(err) {
			return DeliveryReceivable{}, ErrDuplicatePO
		}
		if isDuplicateInvoiceError(err) {
			return DeliveryReceivable{}, ErrDuplicateInvoiceNumber
		}
		return DeliveryReceivable{}, err
	}
	return result, nil
}

func (service *service) Update(ctx context.Context, command UpdateReceivableCommand) (DeliveryReceivable, error) {
	var invoice InvoiceOption
	var err error
	invoiceNumber := command.InvoiceNumber
	if command.InvoiceID > 0 {
		if service.invoices == nil {
			return DeliveryReceivable{}, ValidationErrors{"InvoiceNumber": "Select a valid invoice for this company."}
		}
		invoice, err = service.invoices.FindSelectable(ctx, nil, command.InvoiceID, command.ID)
		if err != nil {
			return DeliveryReceivable{}, err
		}
		if command.CompanyAccountID != invoice.CompanyAccountID {
			return DeliveryReceivable{}, ErrInvoiceCompanyMismatch
		}
		invoiceNumber = invoice.Number
	}
	receivable, err := NewDeliveryReceivableWithTax(command.CompanyAccountID, invoiceNumber, command.PONumber, command.AmountInput, command.DeliveryDate, command.PaymentTermDays, command.TaxRuleCode)
	if err != nil {
		return DeliveryReceivable{}, err
	}
	receivable.InvoiceID = invoice.ID
	receivable.ID = command.ID
	receivable.RowVersion = command.OriginalVersion
	if command.IdempotencyKey == "" {
		return DeliveryReceivable{}, fmt.Errorf("idempotency key is required")
	}
	payloadHash := hashPayload(receivable)
	var result DeliveryReceivable
	err = service.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if command.InvoiceID > 0 {
			invoice, err = service.invoices.FindSelectable(ctx, tx, command.InvoiceID, command.ID)
			if err != nil {
				return err
			}
			if command.CompanyAccountID != invoice.CompanyAccountID {
				return ErrInvoiceCompanyMismatch
			}
			receivable.InvoiceID = invoice.ID
			receivable.InvoiceNumber = invoice.Number
		}
		var existing idempotencyModel
		findErr := tx.Where("idempotency_key = ?", command.IdempotencyKey).First(&existing).Error
		if findErr == nil {
			if existing.RequestHash != payloadHash {
				return ErrIdempotencyConflict
			}
			if existing.ResultEntityID == nil {
				return fmt.Errorf("idempotency request has no result")
			}
			result, err = service.repo.FindByID(ctx, tx, *existing.ResultEntityID)
			return err
		}
		if !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return fmt.Errorf("check idempotency key: %w", findErr)
		}
		exists, err := service.accounts.Exists(ctx, tx, command.CompanyAccountID)
		if err != nil {
			return err
		}
		if !exists {
			return ErrCompanyNotFound
		}
		previous, err := service.repo.FindByID(ctx, tx, command.ID)
		if err != nil {
			return err
		}
		if previous.LifecycleStatus != "Active" || previous.PaymentDateUTC != nil {
			return ErrProtected
		}
		if receivable.InvoiceNumber != "" {
			duplicateInvoice, err := service.repo.FindBlockingInvoiceNumber(ctx, tx, receivable.InvoiceNumber, command.ID)
			if err != nil {
				return err
			}
			if duplicateInvoice {
				return ErrDuplicateInvoiceNumber
			}
		}
		duplicate, err := service.repo.FindBlockingPONumber(ctx, tx, receivable.PONumberNormalized, command.ID)
		if err != nil {
			return err
		}
		if duplicate {
			return ErrDuplicatePO
		}
		idempotency := idempotencyModel{Key: command.IdempotencyKey, CommandType: "update_delivery_receivable", RequestHash: payloadHash, ResponseCode: 303, CreatedAtUTC: service.now(), ExpiresAtUTC: service.now().Add(24 * time.Hour)}
		if err := tx.Create(&idempotency).Error; err != nil {
			return fmt.Errorf("create idempotency record: %w", err)
		}
		result, err = service.repo.Update(ctx, tx, receivable, command.OriginalVersion)
		if err != nil {
			return err
		}
		if err := tx.Model(&idempotency).Updates(map[string]interface{}{"result_entity_id": result.ID}).Error; err != nil {
			return fmt.Errorf("save idempotency result: %w", err)
		}
		before, _ := json.Marshal(previous)
		after, _ := json.Marshal(result)
		if err := tx.Create(&auditModel{EntityType: "delivery_receivable", EntityID: result.ID, Action: "update", ActorID: command.ActorID, OccurredAtUTC: service.now(), RequestID: command.RequestID, IdempotencyKey: command.IdempotencyKey, PreviousValues: string(before), NewValues: string(after)}).Error; err != nil {
			return fmt.Errorf("create audit event: %w", err)
		}
		return nil
	})
	if err != nil {
		if isDuplicatePOError(err) {
			return DeliveryReceivable{}, ErrDuplicatePO
		}
		if isDuplicateInvoiceError(err) {
			return DeliveryReceivable{}, ErrDuplicateInvoiceNumber
		}
		return DeliveryReceivable{}, err
	}
	return result, nil
}

func (service *service) ReceivePayment(ctx context.Context, command ReceivePaymentCommand) (DeliveryReceivable, error) {
	if command.IdempotencyKey == "" {
		return DeliveryReceivable{}, fmt.Errorf("idempotency key is required")
	}
	paymentDate, err := parsePaymentDate(command.PaymentDate)
	if err != nil {
		return DeliveryReceivable{}, ValidationErrors{"PaymentDate": "Enter a valid payment date."}
	}
	if businessdate.FormatUTC(paymentDate) > businessdate.FormatUTC(service.now()) {
		return DeliveryReceivable{}, ValidationErrors{"PaymentDate": "Payment date cannot be in the future."}
	}
	payloadHash := hashPaymentPayload(command.ID, paymentDate)
	var result DeliveryReceivable
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
			result, err = service.repo.FindByID(ctx, tx, *existing.ResultEntityID)
			return err
		}
		if !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return fmt.Errorf("check idempotency key: %w", findErr)
		}
		previous, err := service.repo.FindByID(ctx, tx, command.ID)
		if err != nil {
			return err
		}
		if previous.LifecycleStatus != "Active" {
			return ErrPaymentNotAllowed
		}
		if previous.PaymentDateUTC != nil {
			return ErrAlreadyPaid
		}
		if _, err := ValidatePaymentDate(previous, command.PaymentDate, service.now()); err != nil {
			return err
		}
		idempotency := idempotencyModel{Key: command.IdempotencyKey, CommandType: "receive_delivery_receivable_payment", RequestHash: payloadHash, ResponseCode: 303, CreatedAtUTC: service.now(), ExpiresAtUTC: service.now().Add(24 * time.Hour)}
		if err := tx.Create(&idempotency).Error; err != nil {
			return fmt.Errorf("create idempotency record: %w", err)
		}
		result, err = service.repo.MarkPaymentReceived(ctx, tx, command.ID, paymentDate, command.OriginalVersion)
		if err != nil {
			return err
		}
		if err := tx.Model(&idempotency).Updates(map[string]interface{}{"result_entity_id": result.ID}).Error; err != nil {
			return fmt.Errorf("save idempotency result: %w", err)
		}
		before, _ := json.Marshal(previous)
		after, _ := json.Marshal(result)
		if err := tx.Create(&auditModel{EntityType: "delivery_receivable", EntityID: result.ID, Action: "payment_received", ActorID: command.ActorID, OccurredAtUTC: service.now(), RequestID: command.RequestID, IdempotencyKey: command.IdempotencyKey, PreviousValues: string(before), NewValues: string(after)}).Error; err != nil {
			return fmt.Errorf("create audit event: %w", err)
		}
		return nil
	})
	if err != nil {
		return DeliveryReceivable{}, err
	}
	return result, nil
}

func (service *service) ReversePaymentAcknowledgement(ctx context.Context, command ReversePaymentAcknowledgementCommand) (DeliveryReceivable, error) {
	if command.IdempotencyKey == "" {
		return DeliveryReceivable{}, fmt.Errorf("idempotency key is required")
	}
	reason, err := ValidateReversalReason(command.Reason)
	if err != nil {
		return DeliveryReceivable{}, err
	}
	payloadHash := hashReversePaymentPayload(command.ID, reason)
	var result DeliveryReceivable
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
			result, err = service.repo.FindByID(ctx, tx, *existing.ResultEntityID)
			return err
		}
		if !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return fmt.Errorf("check idempotency key: %w", findErr)
		}
		previous, err := service.repo.FindByID(ctx, tx, command.ID)
		if err != nil {
			return err
		}
		if previous.LifecycleStatus != "Active" || previous.PaymentDateUTC == nil {
			return ErrPaymentAcknowledgementNotReversible
		}
		idempotency := idempotencyModel{Key: command.IdempotencyKey, CommandType: "reverse_payment_acknowledgement", RequestHash: payloadHash, ResponseCode: 303, CreatedAtUTC: service.now(), ExpiresAtUTC: service.now().Add(24 * time.Hour)}
		if err := tx.Create(&idempotency).Error; err != nil {
			return fmt.Errorf("create idempotency record: %w", err)
		}
		result, err = service.repo.ReversePaymentAcknowledgement(ctx, tx, command.ID, command.OriginalVersion)
		if err != nil {
			return err
		}
		if err := tx.Model(&idempotency).Updates(map[string]interface{}{"result_entity_id": result.ID}).Error; err != nil {
			return fmt.Errorf("save idempotency result: %w", err)
		}
		before, _ := json.Marshal(previous)
		after, _ := json.Marshal(result)
		if err := tx.Create(&auditModel{EntityType: "delivery_receivable", EntityID: result.ID, Action: "payment_acknowledgement_reversed", ActorID: command.ActorID, OccurredAtUTC: service.now(), RequestID: command.RequestID, IdempotencyKey: command.IdempotencyKey, PreviousValues: string(before), NewValues: string(after), Note: reason}).Error; err != nil {
			return fmt.Errorf("create audit event: %w", err)
		}
		return nil
	})
	if err != nil {
		return DeliveryReceivable{}, err
	}
	return result, nil
}

func (service *service) Accounts(ctx context.Context) ([]AccountOption, error) {
	accountsList, err := service.accounts.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]AccountOption, 0, len(accountsList))
	for _, account := range accountsList {
		result = append(result, AccountOption{ID: account.ID, CompanyName: account.CompanyName})
	}
	return result, nil
}

func (service *service) InvoiceOptions(ctx context.Context, companyAccountID, excludeReceivableID int64) ([]InvoiceOption, error) {
	if service.invoices == nil {
		return []InvoiceOption{}, nil
	}
	return service.invoices.ListSelectable(ctx, companyAccountID, excludeReceivableID)
}

func (service *service) List(ctx context.Context) ([]ReceivableViewModel, error) {
	result, err := service.ListFiltered(ctx, ListQuery{PageSize: 100, Now: service.now()})
	return result.Items, err
}

func (service *service) ListFiltered(ctx context.Context, query ListQuery) (ListResult, error) {
	query = normalizeListQuery(query)
	if query.Now.IsZero() {
		query.Now = service.now()
	}
	receivables, total, err := service.repo.ListFiltered(ctx, query)
	if err != nil {
		return ListResult{}, err
	}
	result := ListResult{Items: make([]ReceivableViewModel, 0, len(receivables)), RemainingCount: total}
	for _, receivable := range receivables {
		result.Items = append(result.Items, toViewModel(receivable, query.Now))
	}
	if len(receivables) > 0 && total > int64(len(receivables)) {
		result.NextCursor = encodeCursor(receivables[len(receivables)-1], query)
		result.RemainingCount = total - int64(len(receivables))
	} else {
		result.RemainingCount = 0
	}
	return result, nil
}

func (service *service) Get(ctx context.Context, id int64) (ReceivableViewModel, error) {
	receivable, err := service.GetEntity(ctx, id)
	if err != nil {
		return ReceivableViewModel{}, err
	}
	return toViewModel(receivable, service.now()), nil
}

func (service *service) GetEntity(ctx context.Context, id int64) (DeliveryReceivable, error) {
	return service.repo.FindByID(ctx, nil, id)
}

func hashPayload(receivable DeliveryReceivable) string {
	payload, _ := json.Marshal(receivable)
	hash := sha256.Sum256(payload)
	return hex.EncodeToString(hash[:])
}

func hashPaymentPayload(id int64, paymentDate time.Time) string {
	payload, _ := json.Marshal(struct {
		ID          int64     `json:"id"`
		PaymentDate time.Time `json:"payment_date"`
	}{ID: id, PaymentDate: paymentDate})
	hash := sha256.Sum256(payload)
	return hex.EncodeToString(hash[:])
}

func hashReversePaymentPayload(id int64, reason string) string {
	payload, _ := json.Marshal(struct {
		ID     int64  `json:"id"`
		Reason string `json:"reason"`
	}{ID: id, Reason: reason})
	hash := sha256.Sum256(payload)
	return hex.EncodeToString(hash[:])
}

func isDuplicatePOError(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "ux_delivery_receivables_po_not_cancelled")
}

func isDuplicateInvoiceError(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "ux_delivery_receivables_invoice_not_cancelled")
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
	Note           string    `gorm:"column:note"`
}

func (auditModel) TableName() string { return "dbo.audit_events" }
