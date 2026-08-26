package invoices

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
	"gorm.io/gorm"
)

type Repository interface {
	CreateFromSalesOrder(context.Context, int64, string, time.Time, string, string, string) (Invoice, error)
	FindByID(context.Context, int64) (Invoice, error)
	FindBySalesOrder(context.Context, int64) (Invoice, error)
	List(context.Context) ([]Invoice, error)
}

type invoiceModel struct {
	ID                                            int64  `gorm:"column:invoice_id;primaryKey;autoIncrement"`
	Number                                        string `gorm:"column:invoice_number"`
	SalesOrderID                                  int64  `gorm:"column:sales_order_id"`
	SalesOrderNumber                              string `gorm:"column:sales_order_number"`
	CompanyAccountID                              int64  `gorm:"column:company_account_id"`
	CustomerName, BillingAddress, DeliveryAddress string
	ContactPerson, ContactNumber, Email           string
	CustomerPONumber                              string    `gorm:"column:customer_po_number"`
	SalesPerson                                   string    `gorm:"column:sales_person"`
	TermsDays                                     int       `gorm:"column:terms_days"`
	InvoiceDate                                   time.Time `gorm:"column:invoice_date_utc"`
	DueDate                                       time.Time `gorm:"column:due_date_utc"`
	Subtotal                                      int64     `gorm:"column:subtotal_scaled"`
	Tax                                           int64     `gorm:"column:tax_scaled"`
	Total                                         int64     `gorm:"column:total_scaled"`
	Status                                        Status    `gorm:"column:status"`
	CreatedAt, UpdatedAt                          time.Time `gorm:"column:created_at_utc"`
}

func (invoiceModel) TableName() string { return "dbo.invoices" }

type lineModel struct {
	ID                                     int64 `gorm:"column:invoice_line_id;primaryKey;autoIncrement"`
	InvoiceID, SalesOrderLineID, ProductID int64
	SKU, Name                              string
	Quantity                               int64 `gorm:"column:quantity_scaled"`
	UOM                                    string
	UnitPrice                              int64 `gorm:"column:unit_price_scaled"`
	TaxRate                                int64 `gorm:"column:tax_rate_scaled"`
	TaxCode                                string
	LineTotal                              int64 `gorm:"column:line_total_scaled"`
	Inclusive                              int64 `gorm:"column:vat_inclusive_total_scaled"`
}

func (lineModel) TableName() string { return "dbo.invoice_lines" }

type GormRepository struct{ db *gorm.DB }

func NewGormRepository(db *gorm.DB) *GormRepository { return &GormRepository{db: db} }

func (r *GormRepository) CreateFromSalesOrder(ctx context.Context, orderID int64, number string, invoiceDate time.Time, key, requestID, actorID string) (Invoice, error) {
	var result Invoice
	number, err := ValidateInvoiceNumber(number)
	if err != nil {
		return result, err
	}
	if key == "" {
		return result, errors.New("idempotency key is required")
	}
	hash := payloadHash(orderID, number, invoiceDate)
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing struct {
			Key, RequestHash string `gorm:"column:request_hash"`
			ResultID         *int64 `gorm:"column:result_entity_id"`
		}
		found := tx.Table("dbo.idempotency_keys").Where("idempotency_key = ?", key).First(&existing).Error
		if found == nil {
			if existing.RequestHash != hash {
				return ErrIdempotencyConflict
			}
			if existing.ResultID == nil {
				return errors.New("idempotency request has no result")
			}
			result, err = r.findByID(ctx, tx, *existing.ResultID)
			return err
		}
		if !errors.Is(found, gorm.ErrRecordNotFound) {
			return found
		}
		var order struct {
			ID          int64     `gorm:"column:sales_order_id"`
			Number      string    `gorm:"column:sales_order_number"`
			AccountID   int64     `gorm:"column:company_account_id"`
			PO          string    `gorm:"column:customer_po_number"`
			SalesPerson string    `gorm:"column:sales_person"`
			Terms       int       `gorm:"column:terms_days"`
			Subtotal    int64     `gorm:"column:subtotal_scaled"`
			Tax         int64     `gorm:"column:tax_scaled"`
			Total       int64     `gorm:"column:total_scaled"`
			Status      string    `gorm:"column:status"`
			CreatedAt   time.Time `gorm:"column:created_at_utc"`
		}
		if err := tx.Raw("SELECT * FROM dbo.sales_orders WITH (UPDLOCK, HOLDLOCK) WHERE sales_order_id = ?", orderID).Scan(&order).Error; err != nil || order.ID == 0 {
			return ErrInvoiceNotReady
		}
		if order.Status != "OPEN" && order.Status != "COMPLETED" {
			if order.Status == "CONVERTED" {
				return ErrAlreadyInvoiced
			}
			return ErrInvoiceNotReady
		}
		var account struct{ Name, Billing, Delivery, Contact, Phone, Email string }
		if err := tx.Table("dbo.company_accounts").Select("company_name AS name, billing_address AS billing, delivery_address AS delivery, contact_person AS contact, contact_number AS phone, email").Where("company_account_id = ?", order.AccountID).Scan(&account).Error; err != nil {
			return err
		}
		var sourceLines []struct {
			ID        int64  `gorm:"column:sales_order_line_id"`
			ProductID int64  `gorm:"column:product_id"`
			Quantity  int64  `gorm:"column:quantity_scaled"`
			UnitPrice int64  `gorm:"column:unit_price_scaled"`
			TaxRate   int64  `gorm:"column:tax_rate_scaled"`
			LineTotal int64  `gorm:"column:line_total_scaled"`
			Inclusive int64  `gorm:"column:vat_inclusive_total_scaled"`
			UOM       string `gorm:"column:uom"`
			TaxCode   string `gorm:"column:tax_code"`
			SKU       string `gorm:"column:sku"`
			Name      string `gorm:"column:name"`
		}
		if err := tx.Table("dbo.sales_order_lines").Where("sales_order_id = ?", orderID).Order("sales_order_line_id").Find(&sourceLines).Error; err != nil || len(sourceLines) == 0 {
			return ErrInvoiceNotReady
		}
		var duplicate int64
		if err := tx.Table("dbo.invoices").Where("invoice_number = ?", number).Count(&duplicate).Error; err != nil {
			return err
		}
		if duplicate > 0 {
			return ErrDuplicateInvoice
		}
		invoice := invoiceModel{Number: number, SalesOrderID: order.ID, SalesOrderNumber: order.Number, CompanyAccountID: order.AccountID, CustomerName: account.Name, BillingAddress: account.Billing, DeliveryAddress: account.Delivery, ContactPerson: account.Contact, ContactNumber: account.Phone, Email: account.Email, CustomerPONumber: order.PO, SalesPerson: order.SalesPerson, TermsDays: int(order.Terms), InvoiceDate: invoiceDate.UTC(), DueDate: invoiceDate.UTC().AddDate(0, 0, int(order.Terms)), Subtotal: order.Subtotal, Tax: order.Tax, Total: order.Total, Status: Posted, CreatedAt: time.Now().UTC()}
		if err := tx.Create(&invoice).Error; err != nil {
			if isUniqueError(err) {
				return ErrDuplicateInvoice
			}
			return err
		}
		for _, line := range sourceLines {
			stored := lineModel{InvoiceID: invoice.ID, SalesOrderLineID: line.ID, ProductID: line.ProductID, SKU: line.SKU, Name: line.Name, Quantity: line.Quantity, UOM: line.UOM, UnitPrice: line.UnitPrice, TaxRate: line.TaxRate, TaxCode: line.TaxCode, LineTotal: line.LineTotal, Inclusive: line.Inclusive}
			if err := tx.Create(&stored).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&struct{}{}).Table("dbo.sales_orders").Where("sales_order_id = ? AND status IN ('OPEN','COMPLETED')", orderID).Update("status", "CONVERTED").Error; err != nil {
			return err
		}
		idempotency := map[string]interface{}{"idempotency_key": key, "command_type": "create_invoice_from_sales_order", "request_hash": hash, "response_status": 303, "created_at_utc": time.Now().UTC(), "expires_at_utc": time.Now().UTC().Add(24 * time.Hour), "result_entity_id": invoice.ID}
		if err := tx.Table("dbo.idempotency_keys").Create(idempotency).Error; err != nil {
			return err
		}
		before, _ := json.Marshal(order)
		after, _ := json.Marshal(invoice)
		if err := tx.Table("dbo.audit_events").Create(map[string]interface{}{"entity_type": "invoice", "entity_id": invoice.ID, "action": "create_from_sales_order", "actor_id": actorID, "occurred_at_utc": time.Now().UTC(), "request_id": requestID, "idempotency_key": key, "previous_values_json": string(before), "new_values_json": string(after)}).Error; err != nil {
			return err
		}
		result.ID = invoice.ID
		return nil
	})
	if err != nil {
		return Invoice{}, err
	}
	return r.FindByID(ctx, result.ID)
}

func (r *GormRepository) FindByID(ctx context.Context, id int64) (Invoice, error) {
	return r.findByID(ctx, r.db.WithContext(ctx), id)
}
func (r *GormRepository) FindBySalesOrder(ctx context.Context, id int64) (Invoice, error) {
	var m invoiceModel
	if err := r.db.WithContext(ctx).Where("sales_order_id = ?", id).First(&m).Error; err != nil {
		return Invoice{}, err
	}
	return r.FindByID(ctx, m.ID)
}
func (r *GormRepository) List(ctx context.Context) ([]Invoice, error) {
	var rows []invoiceModel
	if err := r.db.WithContext(ctx).Order("invoice_id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]Invoice, 0, len(rows))
	for _, row := range rows {
		item, err := r.FindByID(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func (r *GormRepository) findByID(ctx context.Context, db *gorm.DB, id int64) (Invoice, error) {
	var m invoiceModel
	if err := db.First(&m, "invoice_id = ?", id).Error; err != nil {
		return Invoice{}, err
	}
	var lines []lineModel
	if err := db.Where("invoice_id = ?", id).Order("invoice_line_id").Find(&lines).Error; err != nil {
		return Invoice{}, err
	}
	result := Invoice{ID: m.ID, SalesOrderID: m.SalesOrderID, SalesOrderNumber: m.SalesOrderNumber, Number: m.Number, CompanyAccountID: m.CompanyAccountID, CustomerName: m.CustomerName, BillingAddress: m.BillingAddress, DeliveryAddress: m.DeliveryAddress, ContactPerson: m.ContactPerson, ContactNumber: m.ContactNumber, Email: m.Email, CustomerPONumber: m.CustomerPONumber, SalesPerson: m.SalesPerson, TermsDays: m.TermsDays, InvoiceDateUTC: m.InvoiceDate, DueDateUTC: m.DueDate, Subtotal: moneyAmount(m.Subtotal), Tax: moneyAmount(m.Tax), Total: moneyAmount(m.Total), Status: m.Status, CreatedAtUTC: m.CreatedAt, Lines: make([]Line, 0, len(lines))}
	for _, line := range lines {
		result.Lines = append(result.Lines, Line{ID: line.ID, SalesOrderLineID: line.SalesOrderLineID, ProductID: line.ProductID, SKU: line.SKU, Name: line.Name, Quantity: moneyAmount(line.Quantity), UOM: line.UOM, UnitPrice: moneyAmount(line.UnitPrice), TaxRate: line.TaxRate, TaxCode: line.TaxCode, LineTotal: moneyAmount(line.LineTotal), VATInclusiveTotal: moneyAmount(line.Inclusive)})
	}
	return result, nil
}
func moneyAmount(value int64) money.Amount { return money.Amount(value) }
func payloadHash(id int64, number string, date time.Time) string {
	value, _ := json.Marshal(struct {
		ID     int64
		Number string
		Date   time.Time
	}{id, number, date.UTC()})
	hash := sha256.Sum256(value)
	return hex.EncodeToString(hash[:])
}
func isUniqueError(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "uq_invoices")
}
