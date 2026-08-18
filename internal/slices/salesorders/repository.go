package salesorders

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
	"github.com/psbernardo/syncline-collection-tracking/internal/slices/quotations"
	"gorm.io/gorm"
)

type Status string

const (
	Open      Status = "OPEN"
	Completed Status = "COMPLETED"
)

const FixedSalesPerson = "Alma Mae Bernardo"

type SalesOrder struct {
	ID               int64
	Number           string
	QuotationID      int64
	QuotationNumber  string
	CustomerPONumber string
	SalesPerson      string
	Status           Status
	CreatedAtUTC     time.Time
	Quotation        quotations.Quotation
}

type Repository interface {
	CreateFromQuotation(context.Context, quotations.Quotation, string) (SalesOrder, error)
	FindByID(context.Context, int64) (SalesOrder, error)
}

type GormRepository struct{ db *gorm.DB }

func NewGormRepository(db *gorm.DB) *GormRepository { return &GormRepository{db: db} }

type orderModel struct {
	ID          int64     `gorm:"column:sales_order_id;primaryKey"`
	Number      string    `gorm:"column:sales_order_number"`
	QuotationID int64     `gorm:"column:quotation_id"`
	AccountID   int64     `gorm:"column:company_account_id"`
	PONumber    string    `gorm:"column:customer_po_number"`
	SalesPerson string    `gorm:"column:sales_person"`
	TermsDays   int       `gorm:"column:terms_days"`
	Status      Status    `gorm:"column:status"`
	Subtotal    int64     `gorm:"column:subtotal_scaled"`
	Tax         int64     `gorm:"column:tax_scaled"`
	Total       int64     `gorm:"column:total_scaled"`
	CreatedAt   time.Time `gorm:"column:created_at_utc"`
}

func (orderModel) TableName() string { return "dbo.sales_orders" }

type lineModel struct {
	ProductID int64  `gorm:"column:product_id"`
	Quantity  int64  `gorm:"column:quantity_scaled"`
	UOM       string `gorm:"column:uom"`
	UnitPrice int64  `gorm:"column:unit_price_scaled"`
	TaxRate   int64  `gorm:"column:tax_rate_scaled"`
	TaxCode   string `gorm:"column:tax_code"`
	LineTotal int64  `gorm:"column:line_total_scaled"`
	Inclusive int64  `gorm:"column:vat_inclusive_total_scaled"`
	SKU       string `gorm:"column:sku"`
	Name      string `gorm:"column:name"`
}

func (r *GormRepository) CreateFromQuotation(ctx context.Context, q quotations.Quotation, po string) (SalesOrder, error) {
	if q.ID <= 0 || q.CompanyAccountID <= 0 || len(q.Lines) == 0 {
		return SalesOrder{}, errors.New("quotation is not ready for sales order creation")
	}
	var result SalesOrder
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var sequence int64
		if err := tx.Raw("SELECT NEXT VALUE FOR dbo.SEQ_sales_order_numbers").Scan(&sequence).Error; err != nil {
			return err
		}
		m := orderModel{Number: fmt.Sprintf("SO-%08d", sequence), QuotationID: q.ID, AccountID: q.CompanyAccountID, PONumber: po, SalesPerson: FixedSalesPerson, TermsDays: q.TermsDays, Status: Open, Subtotal: q.Totals.Subtotal.Int64(), Tax: q.Totals.Tax.Int64(), Total: q.Totals.Total.Int64(), CreatedAt: time.Now().UTC()}
		if err := tx.Create(&m).Error; err != nil {
			return err
		}
		for _, line := range q.Lines {
			if err := tx.Table("dbo.sales_order_lines").Create(map[string]interface{}{
				"sales_order_id": m.ID, "product_id": line.ProductID, "quantity_scaled": line.Quantity.Int64(), "uom": line.UOM,
				"unit_price_scaled": line.UnitPrice.Int64(), "tax_rate_scaled": line.TaxRate, "tax_code": line.TaxCode,
				"line_total_scaled": line.LineTotal.Int64(), "vat_inclusive_total_scaled": line.VATInclusiveTotal.Int64(),
			}).Error; err != nil {
				return err
			}
		}
		result = SalesOrder{ID: m.ID, Number: m.Number, QuotationID: m.QuotationID, QuotationNumber: q.Number, CustomerPONumber: m.PONumber, SalesPerson: m.SalesPerson, Status: m.Status, CreatedAtUTC: m.CreatedAt, Quotation: q}
		result.Quotation.Number = m.Number
		result.Quotation.CreatedAtUTC = m.CreatedAt
		return nil
	})
	if err != nil {
		return SalesOrder{}, err
	}
	return r.FindByID(ctx, result.ID)
}

func (r *GormRepository) FindByID(ctx context.Context, id int64) (SalesOrder, error) {
	var m orderModel
	if err := r.db.WithContext(ctx).First(&m, "sales_order_id = ?", id).Error; err != nil {
		return SalesOrder{}, err
	}
	var q quotations.Quotation
	var account struct{ Name, Address, DeliveryAddress, ContactPerson, ContactNumber, Email string }
	if err := r.db.WithContext(ctx).Table("dbo.company_accounts").Select("company_name AS name, billing_address AS address, delivery_address, contact_person, contact_number, email").Where("company_account_id = ?", m.AccountID).Scan(&account).Error; err != nil {
		return SalesOrder{}, err
	}
	q.CompanyAccountID, q.CompanyName, q.CustomerAddress, q.CustomerDeliveryAddress, q.CustomerContactPerson, q.CustomerContactNumber, q.CustomerEmail = m.AccountID, account.Name, account.Address, account.DeliveryAddress, account.ContactPerson, account.ContactNumber, account.Email
	var source struct {
		Number string `gorm:"column:quotation_number"`
	}
	if err := r.db.WithContext(ctx).Table("dbo.quotations").Select("quotation_number").Where("quotation_id = ?", m.QuotationID).Scan(&source).Error; err != nil {
		return SalesOrder{}, err
	}
	q.Number, q.TermsDays, q.CreatedAtUTC = m.Number, m.TermsDays, m.CreatedAt
	q.Totals = quotations.Totals{Subtotal: money.Amount(m.Subtotal), Tax: money.Amount(m.Tax), Total: money.Amount(m.Total)}
	var lines []lineModel
	if err := r.db.WithContext(ctx).Table("dbo.sales_order_lines AS sol").Select("sol.*, p.sku, p.name").Joins("JOIN dbo.products AS p ON p.product_id = sol.product_id").Where("sol.sales_order_id = ?", id).Find(&lines).Error; err != nil {
		return SalesOrder{}, err
	}
	for _, line := range lines {
		q.Lines = append(q.Lines, quotations.Line{ProductID: line.ProductID, ProductSKU: line.SKU, ProductName: line.Name, Quantity: money.Amount(line.Quantity), UOM: line.UOM, UnitPrice: money.Amount(line.UnitPrice), TaxRate: line.TaxRate, TaxCode: line.TaxCode, LineTotal: money.Amount(line.LineTotal), VATInclusiveTotal: money.Amount(line.Inclusive), TaxAmount: money.Amount(line.Inclusive - line.LineTotal)})
	}
	return SalesOrder{ID: m.ID, Number: m.Number, QuotationID: m.QuotationID, QuotationNumber: source.Number, CustomerPONumber: m.PONumber, SalesPerson: m.SalesPerson, Status: m.Status, CreatedAtUTC: m.CreatedAt, Quotation: q}, nil
}
