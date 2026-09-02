package quotations

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
	"gorm.io/gorm"
)

type Quotation struct {
	ID                      int64
	Number                  string
	CompanyAccountID        int64
	CompanyName             string
	CustomerAddress         string
	CustomerDeliveryAddress string
	CustomerContactPerson   string
	CustomerContactNumber   string
	CustomerEmail           string
	ValidityDate            *time.Time
	TermsDays               int
	TaxDefaultCode, Notes   string
	Status                  Status
	CommissionType          CommissionType
	CommissionRate          money.Amount
	CommissionAmount        money.Amount
	EstimatedSupplierCost   money.Amount
	SupplierDeliveryCost    money.Amount
	CustomerDeliveryCost    money.Amount
	OtherCost               money.Amount
	Totals                  Totals
	Lines                   []Line
	CreatedAtUTC            time.Time
	RowVersion              []byte
}

var (
	ErrQuotationNotEditable = errors.New("quotation is not editable")
	ErrQuotationConflict    = errors.New("quotation was changed by another request")
)

type Repository interface {
	Create(context.Context, Quotation) (Quotation, error)
	List(context.Context) ([]Quotation, error)
	FindByID(context.Context, int64) (Quotation, error)
	Update(context.Context, Quotation, []byte) (Quotation, error)
}

type Approver interface {
	Approve(context.Context, int64) (Quotation, error)
}

type ConversionSummary struct {
	OrderCount int
	Converted  money.Amount
	Total      money.Amount
	Remaining  money.Amount
}

type ConversionSummaryProvider interface {
	ConversionSummary(context.Context, int64) (ConversionSummary, error)
}

type NumberGenerator interface {
	NextNumber(context.Context) (string, error)
}
type GormRepository struct{ db *gorm.DB }

func NewGormRepository(db *gorm.DB) *GormRepository { return &GormRepository{db: db} }

type quotationModel struct {
	ID                    int64          `gorm:"column:quotation_id;primaryKey"`
	Number                string         `gorm:"column:quotation_number"`
	CompanyAccountID      int64          `gorm:"column:company_account_id"`
	ValidityDate          *time.Time     `gorm:"column:validity_date"`
	TermsDays             int            `gorm:"column:terms_days"`
	TaxDefaultCode        string         `gorm:"column:tax_default_code"`
	Status                Status         `gorm:"column:status"`
	Notes                 string         `gorm:"column:notes"`
	CommissionType        CommissionType `gorm:"column:commission_type"`
	CommissionRate        int64          `gorm:"column:commission_rate_scaled"`
	CommissionAmount      int64          `gorm:"column:commission_amount_scaled"`
	EstimatedSupplierCost int64          `gorm:"column:estimated_supplier_cost_scaled"`
	SupplierDeliveryCost  int64          `gorm:"column:supplier_delivery_cost_scaled"`
	CustomerDeliveryCost  int64          `gorm:"column:customer_delivery_cost_scaled"`
	OtherCost             int64          `gorm:"column:other_cost_scaled"`
	Subtotal              int64          `gorm:"column:subtotal_scaled"`
	Tax                   int64          `gorm:"column:tax_scaled"`
	Total                 int64          `gorm:"column:total_scaled"`
	Withholding           int64          `gorm:"column:withholding_tax_scaled"`
	ProfitBefore          int64          `gorm:"column:estimated_profit_before_commission_scaled"`
	ProfitAfter           int64          `gorm:"column:estimated_profit_after_commission_scaled"`
	CreatedAtUTC          time.Time      `gorm:"column:created_at_utc"`
	RowVersion            []byte         `gorm:"column:row_version;->"`
}

func (quotationModel) TableName() string { return "dbo.quotations" }

type quotationLineModel struct {
	ID                  int64  `gorm:"column:quotation_line_id;primaryKey"`
	QuotationID         int64  `gorm:"column:quotation_id"`
	ProductID           int64  `gorm:"column:product_id"`
	ProductSKU          string `gorm:"column:product_sku;->"`
	ProductName         string `gorm:"column:product_name;->"`
	Quantity            int64  `gorm:"column:quantity_scaled"`
	UnitPrice           int64  `gorm:"column:unit_price_scaled"`
	LineTotal           int64  `gorm:"column:line_total_scaled"`
	TaxRate             int64  `gorm:"column:tax_rate_scaled"`
	VATInclusiveTotal   int64  `gorm:"column:vat_inclusive_total_scaled"`
	SupplierCost        int64  `gorm:"column:supplier_cost_scaled"`
	SupplierProductCost int64  `gorm:"column:supplier_product_cost_scaled"`
	CommissionAmount    int64  `gorm:"column:commission_amount_scaled"`
	ProfitBefore        int64  `gorm:"column:profit_before_commission_scaled"`
	ProfitAfter         int64  `gorm:"column:profit_after_commission_scaled"`
	UOM                 string `gorm:"column:uom"`
	TaxCode             string `gorm:"column:tax_code"`
}

func (quotationLineModel) TableName() string { return "dbo.quotation_lines" }

func (r *GormRepository) Create(ctx context.Context, input Quotation) (Quotation, error) {
	if input.CompanyAccountID <= 0 || len(input.Lines) == 0 {
		return Quotation{}, fmt.Errorf("customer and at least one line are required")
	}
	if !ValidTermsDays(input.TermsDays) {
		return Quotation{}, fmt.Errorf("terms must be 7, 15, 30, or 45 days")
	}
	if input.Status == "" {
		input.Status = Draft
	}
	if _, err := NormalizeStatus(string(input.Status)); err != nil {
		return Quotation{}, err
	}
	if input.TaxDefaultCode == "" {
		input.TaxDefaultCode = TaxNone
	}
	var err error
	input.Lines, err = applyTaxRule(input.Lines, input.TaxDefaultCode)
	if err != nil {
		return Quotation{}, err
	}
	calculatedLines, calculated, err := CalculateProfitability(input.Lines, input.CommissionType, input.CommissionRate, input.SupplierDeliveryCost, input.CustomerDeliveryCost, input.OtherCost)
	if err != nil {
		return Quotation{}, err
	}
	input.Lines, input.Totals = calculatedLines, calculated
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if input.Number == "" {
			var sequence int64
			if err := tx.Raw("SELECT NEXT VALUE FOR dbo.SEQ_quotation_numbers").Scan(&sequence).Error; err != nil {
				return err
			}
			input.Number = generatedQuotationNumber(sequence)
		}
		if !validQuotationNumber(input.Number) {
			return fmt.Errorf("invalid generated quotation number")
		}
		var count int64
		if err := tx.Table("dbo.company_accounts").Where("company_account_id = ?", input.CompanyAccountID).Count(&count).Error; err != nil {
			return err
		}
		if count != 1 {
			return fmt.Errorf("customer account not found")
		}
		for index := range input.Lines {
			var product struct{ UOM string }
			if err := tx.Table("dbo.products").Select("uom").Where("product_id = ? AND is_active = 1", input.Lines[index].ProductID).First(&product).Error; err != nil {
				return fmt.Errorf("product %d is not active", input.Lines[index].ProductID)
			}
			input.Lines[index].UOM = product.UOM
		}
		m := quotationModel{Number: input.Number, CompanyAccountID: input.CompanyAccountID, ValidityDate: input.ValidityDate, TermsDays: input.TermsDays, TaxDefaultCode: input.TaxDefaultCode, Status: input.Status, Notes: input.Notes, CommissionType: input.CommissionType, CommissionRate: input.CommissionRate.Int64(), CommissionAmount: calculated.Commission.Int64(), EstimatedSupplierCost: calculated.SupplierProductCost.Int64(), SupplierDeliveryCost: input.SupplierDeliveryCost.Int64(), CustomerDeliveryCost: input.CustomerDeliveryCost.Int64(), OtherCost: input.OtherCost.Int64(), Subtotal: calculated.Subtotal.Int64(), Tax: calculated.Tax.Int64(), Total: calculated.Total.Int64(), Withholding: calculated.Withholding.Int64(), ProfitBefore: calculated.ProfitBeforeCommission.Int64(), ProfitAfter: calculated.ProfitAfterCommission.Int64(), CreatedAtUTC: time.Now().UTC()}
		if err := tx.Create(&m).Error; err != nil {
			return err
		}
		for _, line := range input.Lines {
			if err := tx.Create(&quotationLineModel{QuotationID: m.ID, ProductID: line.ProductID, Quantity: line.Quantity.Int64(), UOM: line.UOM, UnitPrice: line.UnitPrice.Int64(), LineTotal: line.LineTotal.Int64(), TaxCode: line.TaxCode, TaxRate: line.TaxRate, VATInclusiveTotal: line.VATInclusiveTotal.Int64(), SupplierCost: line.SupplierCost.Int64(), SupplierProductCost: line.SupplierProductCost.Int64(), CommissionAmount: line.CommissionAmount.Int64(), ProfitBefore: line.ProfitBeforeCommission.Int64(), ProfitAfter: line.ProfitAfterCommission.Int64()}).Error; err != nil {
				return err
			}
		}
		input.ID = m.ID
		return nil
	})
	if err != nil {
		return Quotation{}, err
	}
	return r.FindByID(ctx, input.ID)
}

func (r *GormRepository) NextNumber(ctx context.Context) (string, error) {
	var sequence int64
	if err := r.db.WithContext(ctx).Raw("SELECT NEXT VALUE FOR dbo.SEQ_quotation_numbers").Scan(&sequence).Error; err != nil {
		return "", err
	}
	return generatedQuotationNumber(sequence), nil
}

func generatedQuotationNumber(sequence int64) string { return fmt.Sprintf("QT-%08d", sequence) }

func validQuotationNumber(value string) bool {
	if len(value) != len("QT-00000000") || !strings.HasPrefix(value, "QT-") {
		return false
	}
	for _, char := range value[3:] {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func (r *GormRepository) Update(ctx context.Context, input Quotation, version []byte) (Quotation, error) {
	if input.Status != Draft {
		return Quotation{}, ErrQuotationNotEditable
	}
	if input.Number == "" || input.CompanyAccountID <= 0 || len(input.Lines) == 0 {
		return Quotation{}, fmt.Errorf("quotation number, customer, and at least one line are required")
	}
	if !ValidTermsDays(input.TermsDays) {
		return Quotation{}, fmt.Errorf("terms must be 7, 15, 30, or 45 days")
	}
	if input.TaxDefaultCode == "" {
		input.TaxDefaultCode = TaxNone
	}
	var err error
	input.Lines, err = applyTaxRule(input.Lines, input.TaxDefaultCode)
	if err != nil {
		return Quotation{}, err
	}
	calculatedLines, calculated, err := CalculateProfitability(input.Lines, input.CommissionType, input.CommissionRate, input.SupplierDeliveryCost, input.CustomerDeliveryCost, input.OtherCost)
	if err != nil {
		return Quotation{}, err
	}
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Table("dbo.company_accounts").Where("company_account_id = ?", input.CompanyAccountID).Count(&count).Error; err != nil {
			return err
		}
		if count != 1 {
			return fmt.Errorf("customer account not found")
		}
		for index := range input.Lines {
			var product struct{ UOM string }
			if err := tx.Table("dbo.products").Select("uom").Where("product_id = ? AND is_active = 1", input.Lines[index].ProductID).First(&product).Error; err != nil {
				return fmt.Errorf("product %d is not active", input.Lines[index].ProductID)
			}
			input.Lines[index].UOM = product.UOM
		}
		result := tx.Model(&quotationModel{}).Where("quotation_id = ? AND row_version = ? AND status = ?", input.ID, version, Draft).Updates(map[string]interface{}{
			"quotation_number": input.Number, "company_account_id": input.CompanyAccountID, "validity_date": input.ValidityDate,
			"terms_days": input.TermsDays, "tax_default_code": input.TaxDefaultCode,
			"notes": input.Notes, "commission_type": input.CommissionType, "commission_rate_scaled": input.CommissionRate.Int64(),
			"commission_amount_scaled": calculated.Commission.Int64(), "estimated_supplier_cost_scaled": calculated.SupplierProductCost.Int64(), "supplier_delivery_cost_scaled": input.SupplierDeliveryCost.Int64(), "customer_delivery_cost_scaled": input.CustomerDeliveryCost.Int64(), "other_cost_scaled": input.OtherCost.Int64(), "subtotal_scaled": calculated.Subtotal.Int64(), "tax_scaled": calculated.Tax.Int64(),
			"total_scaled": calculated.Total.Int64(), "withholding_tax_scaled": calculated.Withholding.Int64(), "estimated_profit_before_commission_scaled": calculated.ProfitBeforeCommission.Int64(),
			"estimated_profit_after_commission_scaled": calculated.ProfitAfterCommission.Int64(), "updated_at_utc": gorm.Expr("SYSUTCDATETIME()"),
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrQuotationConflict
		}
		if err := tx.Where("quotation_id = ?", input.ID).Delete(&quotationLineModel{}).Error; err != nil {
			return err
		}
		for _, line := range calculatedLines {
			if err := tx.Create(&quotationLineModel{QuotationID: input.ID, ProductID: line.ProductID, Quantity: line.Quantity.Int64(), UOM: line.UOM, UnitPrice: line.UnitPrice.Int64(), LineTotal: line.LineTotal.Int64(), TaxCode: line.TaxCode, TaxRate: line.TaxRate, VATInclusiveTotal: line.VATInclusiveTotal.Int64(), SupplierCost: line.SupplierCost.Int64(), SupplierProductCost: line.SupplierProductCost.Int64(), CommissionAmount: line.CommissionAmount.Int64(), ProfitBefore: line.ProfitBeforeCommission.Int64(), ProfitAfter: line.ProfitAfterCommission.Int64()}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return Quotation{}, err
	}
	return r.FindByID(ctx, input.ID)
}

func (r *GormRepository) Approve(ctx context.Context, id int64) (Quotation, error) {
	result := r.db.WithContext(ctx).Model(&quotationModel{}).Where("quotation_id = ? AND status IN ?", id, []Status{Draft, Sent}).Updates(map[string]interface{}{"status": Approved, "updated_at_utc": gorm.Expr("SYSUTCDATETIME()")})
	if result.Error != nil {
		return Quotation{}, result.Error
	}
	if result.RowsAffected != 1 {
		return Quotation{}, ErrQuotationNotEditable
	}
	return r.FindByID(ctx, id)
}
func (r *GormRepository) List(ctx context.Context) ([]Quotation, error) {
	var rows []quotationModel
	if err := r.db.WithContext(ctx).Order("quotation_id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]Quotation, 0, len(rows))
	for _, row := range rows {
		value, err := r.fromModel(ctx, row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}
func (r *GormRepository) FindByID(ctx context.Context, id int64) (Quotation, error) {
	var row quotationModel
	if err := r.db.WithContext(ctx).First(&row, "quotation_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Quotation{}, gorm.ErrRecordNotFound
		}
		return Quotation{}, err
	}
	return r.fromModel(ctx, row)
}
func (r *GormRepository) fromModel(ctx context.Context, row quotationModel) (Quotation, error) {
	var company struct {
		Name            string `gorm:"column:name"`
		Address         string `gorm:"column:address"`
		DeliveryAddress string `gorm:"column:delivery_address"`
		ContactPerson   string `gorm:"column:contact_person"`
		ContactNumber   string `gorm:"column:contact_number"`
		Email           string `gorm:"column:email"`
	}
	if err := r.db.WithContext(ctx).Table("dbo.company_accounts").Select("company_name AS name, billing_address AS address, delivery_address, contact_person, contact_number, email").Where("company_account_id = ?", row.CompanyAccountID).Scan(&company).Error; err != nil {
		return Quotation{}, err
	}
	var lines []quotationLineModel
	if err := r.db.WithContext(ctx).Table("dbo.quotation_lines AS ql").Select("ql.quotation_line_id, ql.quotation_id, ql.product_id, p.sku AS product_sku, p.name AS product_name, ql.quantity_scaled, ql.unit_price_scaled, ql.line_total_scaled, ql.tax_rate_scaled, ql.vat_inclusive_total_scaled, ql.supplier_cost_scaled, ql.supplier_product_cost_scaled, ql.commission_amount_scaled, ql.profit_before_commission_scaled, ql.profit_after_commission_scaled, ql.uom, ql.tax_code").Joins("JOIN dbo.products AS p ON p.product_id = ql.product_id").Where("ql.quotation_id = ?", row.ID).Find(&lines).Error; err != nil {
		return Quotation{}, err
	}
	value := Quotation{ID: row.ID, Number: row.Number, CompanyAccountID: row.CompanyAccountID, CompanyName: company.Name, CustomerAddress: company.Address, CustomerDeliveryAddress: company.DeliveryAddress, CustomerContactPerson: company.ContactPerson, CustomerContactNumber: company.ContactNumber, CustomerEmail: company.Email, ValidityDate: row.ValidityDate, TermsDays: row.TermsDays, TaxDefaultCode: row.TaxDefaultCode, Notes: row.Notes, Status: row.Status, CommissionType: row.CommissionType, CommissionRate: money.Amount(row.CommissionRate), CommissionAmount: money.Amount(row.CommissionAmount), EstimatedSupplierCost: money.Amount(row.EstimatedSupplierCost), SupplierDeliveryCost: money.Amount(row.SupplierDeliveryCost), CustomerDeliveryCost: money.Amount(row.CustomerDeliveryCost), OtherCost: money.Amount(row.OtherCost), CreatedAtUTC: row.CreatedAtUTC, RowVersion: row.RowVersion, Totals: Totals{Subtotal: money.Amount(row.Subtotal), Tax: money.Amount(row.Tax), Total: money.Amount(row.Total), Withholding: money.Amount(row.Withholding), ProfitBeforeCommission: money.Amount(row.ProfitBefore), ProfitAfterCommission: money.Amount(row.ProfitAfter), SupplierProductCost: money.Amount(row.EstimatedSupplierCost), SupplierDeliveryCost: money.Amount(row.SupplierDeliveryCost), CustomerDeliveryCost: money.Amount(row.CustomerDeliveryCost), OtherCost: money.Amount(row.OtherCost), TotalCost: money.Amount(row.EstimatedSupplierCost + row.SupplierDeliveryCost + row.CustomerDeliveryCost + row.OtherCost + row.CommissionAmount), MarginBeforeCommission: margin(money.Amount(row.ProfitBefore), money.Amount(row.Subtotal)), MarginAfterCommission: margin(money.Amount(row.ProfitAfter), money.Amount(row.Subtotal))}}
	for _, line := range lines {
		netSales := money.Amount(line.LineTotal)
		if line.TaxCode == TaxVAT12 {
			netSales = money.Amount((line.LineTotal * 100) / 112)
		}
		value.Lines = append(value.Lines, Line{ID: line.ID, ProductID: line.ProductID, ProductSKU: line.ProductSKU, ProductName: line.ProductName, Quantity: money.Amount(line.Quantity), UOM: line.UOM, UnitPrice: money.Amount(line.UnitPrice), TaxCode: line.TaxCode, TaxRate: line.TaxRate, SupplierCost: money.Amount(line.SupplierCost), SupplierProductCost: money.Amount(line.SupplierProductCost), CommissionAmount: money.Amount(line.CommissionAmount), ProfitBeforeCommission: money.Amount(line.ProfitBefore), ProfitAfterCommission: money.Amount(line.ProfitAfter), MarginBeforeCommission: margin(money.Amount(line.ProfitBefore), netSales), MarginAfterCommission: margin(money.Amount(line.ProfitAfter), netSales), LineTotal: money.Amount(line.LineTotal), TaxAmount: money.Amount(line.VATInclusiveTotal - line.LineTotal), VATInclusiveTotal: money.Amount(line.VATInclusiveTotal)})
	}
	return value, nil
}
