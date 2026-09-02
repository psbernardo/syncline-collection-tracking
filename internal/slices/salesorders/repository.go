package salesorders

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
	"github.com/psbernardo/syncline-collection-tracking/internal/slices/quotations"
	"gorm.io/gorm"
)

type Status string

const (
	Open      Status = "OPEN"
	Converted Status = "CONVERTED"
)

const FixedSalesPerson = "Alma Mae Bernardo"

var (
	ErrOrderNotReady   = errors.New("sales order is not ready for creation")
	ErrAllocationStale = errors.New("quotation quantities changed; reload and try again")
	ErrInvalidPO       = errors.New("customer PO number is required")
)

func ValidateCustomerPO(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ErrInvalidPO
	}
	if len([]rune(value)) > 100 {
		return "", errors.New("customer PO number must be 100 characters or fewer")
	}
	return value, nil
}

type LineSelection struct {
	QuotationLineID int64
	Quantity        money.Amount
}

type AllocationLine struct {
	Line      quotations.Line
	Converted money.Amount
	Current   money.Amount
	Remaining money.Amount
}

type StandaloneLineInput struct {
	ProductID int64
	Quantity  money.Amount
	UOM       string
	UnitPrice money.Amount
	TaxCode   string
	TaxRate   int64
}

type StandaloneOrderInput struct {
	Number           string
	CompanyAccountID int64
	CustomerPONumber string
	TermsDays        int
	Lines            []StandaloneLineInput
}

type SalesOrder struct {
	ID               int64
	InvoiceID        int64
	InvoiceNumber    string
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

type SelectionCreator interface {
	CreateFromQuotationSelection(context.Context, quotations.Quotation, string, []LineSelection) (SalesOrder, error)
}

type NumberedSelectionCreator interface {
	CreateFromQuotationSelectionWithNumber(context.Context, quotations.Quotation, string, []LineSelection, string) (SalesOrder, error)
}

type StandaloneCreator interface {
	CreateStandalone(context.Context, StandaloneOrderInput) (SalesOrder, error)
}

type StandaloneUpdater interface {
	UpdateStandalone(context.Context, int64, StandaloneOrderInput) (SalesOrder, error)
}

type QuotationUpdater interface {
	UpdateFromQuotation(context.Context, int64, quotations.Quotation, string, []LineSelection) (SalesOrder, error)
}

type AllocationReader interface {
	AllocationLines(context.Context, quotations.Quotation) ([]AllocationLine, error)
}

type OrderAllocationReader interface {
	AllocationLinesForOrder(context.Context, quotations.Quotation, int64) ([]AllocationLine, error)
}

type GormRepository struct{ db *gorm.DB }

func NewGormRepository(db *gorm.DB) *GormRepository { return &GormRepository{db: db} }

func (r *GormRepository) NextNumber(ctx context.Context) (string, error) {
	var sequence int64
	if err := r.db.WithContext(ctx).Raw("SELECT NEXT VALUE FOR dbo.SEQ_sales_order_numbers").Scan(&sequence).Error; err != nil {
		return "", err
	}
	return GeneratedSalesOrderNumber(sequence), nil
}

func GeneratedSalesOrderNumber(sequence int64) string { return fmt.Sprintf("SO-%08d", sequence) }

func ValidSalesOrderNumber(value string) bool {
	if len(value) != len("SO-00000000") || value[:3] != "SO-" {
		return false
	}
	for _, char := range value[3:] {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

type orderModel struct {
	ID          int64     `gorm:"column:sales_order_id;primaryKey;autoIncrement"`
	Number      string    `gorm:"column:sales_order_number"`
	QuotationID *int64    `gorm:"column:quotation_id"`
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
	ID              int64  `gorm:"column:sales_order_line_id;primaryKey;autoIncrement"`
	QuotationLineID *int64 `gorm:"column:quotation_line_id"`
	ProductID       int64  `gorm:"column:product_id"`
	Quantity        int64  `gorm:"column:quantity_scaled"`
	UOM             string `gorm:"column:uom"`
	UnitPrice       int64  `gorm:"column:unit_price_scaled"`
	TaxRate         int64  `gorm:"column:tax_rate_scaled"`
	TaxCode         string `gorm:"column:tax_code"`
	LineTotal       int64  `gorm:"column:line_total_scaled"`
	Inclusive       int64  `gorm:"column:vat_inclusive_total_scaled"`
	SKU             string `gorm:"column:sku"`
	Name            string `gorm:"column:name"`
}

func (lineModel) TableName() string { return "dbo.sales_order_lines" }

func (r *GormRepository) CreateFromQuotation(ctx context.Context, q quotations.Quotation, po string) (SalesOrder, error) {
	selections := make([]LineSelection, 0, len(q.Lines))
	for _, line := range q.Lines {
		selections = append(selections, LineSelection{QuotationLineID: line.ID, Quantity: line.Quantity})
	}
	return r.CreateFromQuotationSelection(ctx, q, po, selections)
}

func (r *GormRepository) CreateFromQuotationSelectionWithNumber(ctx context.Context, q quotations.Quotation, po string, selections []LineSelection, number string) (SalesOrder, error) {
	return r.createFromQuotationSelection(ctx, q, po, selections, number)
}

func (r *GormRepository) CreateFromQuotationSelection(ctx context.Context, q quotations.Quotation, po string, selections []LineSelection) (SalesOrder, error) {
	return r.createFromQuotationSelection(ctx, q, po, selections, "")
}

func (r *GormRepository) createFromQuotationSelection(ctx context.Context, q quotations.Quotation, po string, selections []LineSelection, number string) (SalesOrder, error) {
	var err error
	po, err = ValidateCustomerPO(po)
	if err != nil {
		return SalesOrder{}, err
	}
	if q.ID <= 0 || q.CompanyAccountID <= 0 || len(q.Lines) == 0 || len(selections) == 0 || (q.Status != quotations.Approved && q.Status != "") {
		return SalesOrder{}, ErrOrderNotReady
	}
	byID := make(map[int64]quotations.Line, len(q.Lines))
	for _, line := range q.Lines {
		byID[line.ID] = line
	}
	var result SalesOrder
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		allocated, err := r.lockAllocations(tx, q.ID)
		if err != nil {
			return err
		}
		requested := make(map[int64]money.Amount, len(selections))
		for _, selection := range selections {
			requested[selection.QuotationLineID] += selection.Quantity
		}
		for lineID, quantity := range requested {
			line, ok := byID[lineID]
			if !ok || quantity <= 0 || quantity > line.Quantity-allocated[lineID] {
				return ErrAllocationStale
			}
		}
		selected := make([]quotations.Line, 0, len(requested))
		for _, selection := range selections {
			line, ok := byID[selection.QuotationLineID]
			if !ok || selection.Quantity <= 0 {
				return ErrAllocationStale
			}
			line.Quantity = selection.Quantity
			line.LineTotal, line.TaxAmount, line.VATInclusiveTotal, err = quotations.CalculateLine(line)
			if err != nil {
				return err
			}
			selected = append(selected, line)
		}
		if len(selected) == 0 {
			return ErrOrderNotReady
		}
		return r.createOrderTx(tx, q.CompanyAccountID, q.TermsDays, po, &q, selected, selections, &result, number)
	})
	if err != nil {
		return SalesOrder{}, err
	}
	return r.FindByID(ctx, result.ID)
}

func (r *GormRepository) CreateStandalone(ctx context.Context, input StandaloneOrderInput) (SalesOrder, error) {
	var poErr error
	input.CustomerPONumber, poErr = ValidateCustomerPO(input.CustomerPONumber)
	if poErr != nil {
		return SalesOrder{}, poErr
	}
	if input.CompanyAccountID <= 0 || len(input.Lines) == 0 || !quotations.ValidTermsDays(input.TermsDays) {
		return SalesOrder{}, ErrOrderNotReady
	}
	if input.Number != "" && !ValidSalesOrderNumber(input.Number) {
		return SalesOrder{}, ErrOrderNotReady
	}
	lines := make([]quotations.Line, 0, len(input.Lines))
	for _, inputLine := range input.Lines {
		if inputLine.ProductID <= 0 || inputLine.Quantity <= 0 || inputLine.UnitPrice < 0 {
			return SalesOrder{}, ErrOrderNotReady
		}
		line := quotations.Line{ProductID: inputLine.ProductID, Quantity: inputLine.Quantity, UOM: inputLine.UOM, UnitPrice: inputLine.UnitPrice, TaxCode: inputLine.TaxCode, TaxRate: inputLine.TaxRate}
		var lineErr error
		line.LineTotal, line.TaxAmount, line.VATInclusiveTotal, lineErr = quotations.CalculateLine(line)
		if lineErr != nil {
			return SalesOrder{}, lineErr
		}
		if line.TaxCode == "" {
			line.TaxCode = quotations.TaxNone
		}
		lines = append(lines, line)
	}
	calculated, totals, err := quotations.CalculateProfitability(lines, quotations.NoCommission, 0, 0, 0, 0)
	if err != nil {
		return SalesOrder{}, err
	}
	var result SalesOrder
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var accountCount int64
		if err := tx.Table("dbo.company_accounts").Where("company_account_id = ?", input.CompanyAccountID).Count(&accountCount).Error; err != nil || accountCount != 1 {
			return ErrOrderNotReady
		}
		for index := range calculated {
			var product struct {
				SKU, Name, UOM string
				Active         bool `gorm:"column:is_active"`
			}
			if err := tx.Table("dbo.products").Select("sku, name, uom, is_active").Where("product_id = ?", calculated[index].ProductID).First(&product).Error; err != nil || !product.Active || (calculated[index].UOM != "" && product.UOM != calculated[index].UOM) {
				return ErrOrderNotReady
			}
			calculated[index].ProductSKU, calculated[index].ProductName, calculated[index].UOM = product.SKU, product.Name, product.UOM
		}
		return r.createOrderTx(tx, input.CompanyAccountID, input.TermsDays, input.CustomerPONumber, nil, calculated, nil, &result, totals, input.Number)
	})
	if err != nil {
		return SalesOrder{}, err
	}
	return r.FindByID(ctx, result.ID)
}

func (r *GormRepository) UpdateStandalone(ctx context.Context, id int64, input StandaloneOrderInput) (SalesOrder, error) {
	var poErr error
	input.CustomerPONumber, poErr = ValidateCustomerPO(input.CustomerPONumber)
	if poErr != nil {
		return SalesOrder{}, poErr
	}
	if input.CompanyAccountID <= 0 || len(input.Lines) == 0 || !quotations.ValidTermsDays(input.TermsDays) {
		return SalesOrder{}, ErrOrderNotReady
	}
	lines := make([]quotations.Line, 0, len(input.Lines))
	for _, item := range input.Lines {
		if item.ProductID <= 0 || item.Quantity <= 0 || item.UnitPrice < 0 {
			return SalesOrder{}, ErrOrderNotReady
		}
		if item.TaxCode == "" {
			item.TaxCode = quotations.TaxNone
		}
		line := quotations.Line{ProductID: item.ProductID, Quantity: item.Quantity, UOM: item.UOM, UnitPrice: item.UnitPrice, TaxCode: item.TaxCode, TaxRate: item.TaxRate}
		var err error
		line.LineTotal, line.TaxAmount, line.VATInclusiveTotal, err = quotations.CalculateLine(line)
		if err != nil {
			return SalesOrder{}, err
		}
		lines = append(lines, line)
	}
	calculated, totals, err := quotations.CalculateProfitability(lines, quotations.NoCommission, 0, 0, 0, 0)
	if err != nil {
		return SalesOrder{}, err
	}
	var result SalesOrder
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current orderModel
		if err := tx.First(&current, "sales_order_id = ?", id).Error; err != nil || current.QuotationID != nil || current.Status != Open {
			return ErrOrderNotReady
		}
		var count int64
		if err := tx.Table("dbo.company_accounts").Where("company_account_id = ?", input.CompanyAccountID).Count(&count).Error; err != nil || count != 1 {
			return ErrOrderNotReady
		}
		for index := range calculated {
			var product struct {
				SKU, Name, UOM string
				Active         bool `gorm:"column:is_active"`
			}
			if err := tx.Table("dbo.products").Select("sku, name, uom, is_active").Where("product_id = ?", calculated[index].ProductID).First(&product).Error; err != nil || !product.Active || (calculated[index].UOM != "" && calculated[index].UOM != product.UOM) {
				return ErrOrderNotReady
			}
			calculated[index].ProductSKU, calculated[index].ProductName, calculated[index].UOM = product.SKU, product.Name, product.UOM
		}
		if err := tx.Model(&orderModel{}).Where("sales_order_id = ?", id).Updates(map[string]interface{}{"company_account_id": input.CompanyAccountID, "customer_po_number": input.CustomerPONumber, "terms_days": input.TermsDays, "subtotal_scaled": totals.Subtotal.Int64(), "tax_scaled": totals.Tax.Int64(), "total_scaled": totals.Total.Int64(), "updated_at_utc": gorm.Expr("SYSUTCDATETIME()")}).Error; err != nil {
			return err
		}
		if err := tx.Where("sales_order_id = ?", id).Delete(&lineModel{}).Error; err != nil {
			return err
		}
		for _, line := range calculated {
			if err := tx.Table("dbo.sales_order_lines").Create(map[string]interface{}{"sales_order_id": id, "product_id": line.ProductID, "quantity_scaled": line.Quantity.Int64(), "uom": line.UOM, "unit_price_scaled": line.UnitPrice.Int64(), "tax_rate_scaled": line.TaxRate, "tax_code": line.TaxCode, "line_total_scaled": line.LineTotal.Int64(), "vat_inclusive_total_scaled": line.VATInclusiveTotal.Int64(), "sku": line.ProductSKU, "name": line.ProductName}).Error; err != nil {
				return err
			}
		}
		result.ID = id
		return nil
	})
	if err != nil {
		return SalesOrder{}, err
	}
	return r.FindByID(ctx, result.ID)
}

func (r *GormRepository) UpdateFromQuotation(ctx context.Context, id int64, q quotations.Quotation, po string, selections []LineSelection) (SalesOrder, error) {
	po, err := ValidateCustomerPO(po)
	if err != nil || len(selections) == 0 || (q.Status != quotations.Approved && q.Status != "") {
		return SalesOrder{}, ErrOrderNotReady
	}
	byID := make(map[int64]quotations.Line, len(q.Lines))
	for _, line := range q.Lines {
		byID[line.ID] = line
	}
	var result SalesOrder
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order orderModel
		if err := tx.Raw("SELECT * FROM dbo.sales_orders WITH (UPDLOCK, HOLDLOCK) WHERE sales_order_id = ?", id).Scan(&order).Error; err != nil || order.ID == 0 || order.Status != Open || order.QuotationID == nil || *order.QuotationID != q.ID {
			return ErrOrderNotReady
		}
		all, current, err := r.lockAllocationSets(tx, q.ID, id)
		if err != nil {
			return err
		}
		requested := make(map[int64]money.Amount, len(selections))
		for _, selection := range selections {
			requested[selection.QuotationLineID] += selection.Quantity
		}
		selected := make([]quotations.Line, 0, len(requested))
		ordered := make([]LineSelection, 0, len(requested))
		for lineID, quantity := range requested {
			line, ok := byID[lineID]
			available := line.Quantity - all[lineID] + current[lineID]
			if !ok || quantity <= 0 || quantity > available {
				return ErrAllocationStale
			}
			line.Quantity = quantity
			line.LineTotal, line.TaxAmount, line.VATInclusiveTotal, err = quotations.CalculateLine(line)
			if err != nil {
				return err
			}
			selected = append(selected, line)
			ordered = append(ordered, LineSelection{QuotationLineID: lineID, Quantity: quantity})
		}
		_, totals, err := quotations.CalculateProfitability(selected, quotations.NoCommission, 0, 0, 0, 0)
		if err != nil {
			return err
		}
		if err := tx.Model(&orderModel{}).Where("sales_order_id = ?", id).Updates(map[string]interface{}{"customer_po_number": po, "subtotal_scaled": totals.Subtotal.Int64(), "tax_scaled": totals.Tax.Int64(), "total_scaled": totals.Total.Int64(), "updated_at_utc": gorm.Expr("SYSUTCDATETIME()")}).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM dbo.quotation_line_allocations WHERE sales_order_id = ?", id).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM dbo.sales_order_lines WHERE sales_order_id = ?", id).Error; err != nil {
			return err
		}
		for index, line := range selected {
			var storedID int64
			if err := tx.Raw(`INSERT INTO dbo.sales_order_lines (sales_order_id, quotation_line_id, product_id, quantity_scaled, uom, unit_price_scaled, tax_rate_scaled, tax_code, line_total_scaled, vat_inclusive_total_scaled, sku, name) OUTPUT INSERTED.sales_order_line_id VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, id, ordered[index].QuotationLineID, line.ProductID, line.Quantity.Int64(), line.UOM, line.UnitPrice.Int64(), line.TaxRate, line.TaxCode, line.LineTotal.Int64(), line.VATInclusiveTotal.Int64(), line.ProductSKU, line.ProductName).Scan(&storedID).Error; err != nil || storedID == 0 {
				return errors.New("sales order line identity was not generated")
			}
			if err := tx.Table("dbo.quotation_line_allocations").Create(map[string]interface{}{"quotation_id": q.ID, "quotation_line_id": ordered[index].QuotationLineID, "sales_order_id": id, "sales_order_line_id": storedID, "quantity_scaled": line.Quantity.Int64()}).Error; err != nil {
				return err
			}
		}
		result.ID = id
		return nil
	})
	if err != nil {
		return SalesOrder{}, err
	}
	return r.FindByID(ctx, result.ID)
}

func (r *GormRepository) createOrderTx(tx *gorm.DB, accountID int64, terms int, po string, source *quotations.Quotation, lines []quotations.Line, selections []LineSelection, result *SalesOrder, totalsOverride ...interface{}) error {
	var totals quotations.Totals
	var err error
	number := ""
	for _, value := range totalsOverride {
		switch typed := value.(type) {
		case quotations.Totals:
			totals = typed
		case string:
			number = typed
		}
	}
	if totals == (quotations.Totals{}) {
		_, totals, err = quotations.CalculateProfitability(lines, quotations.NoCommission, 0, 0, 0, 0)
		if err != nil {
			return err
		}
	} else {
		if number != "" && !ValidSalesOrderNumber(number) {
			return ErrOrderNotReady
		}
	}
	var sequence int64
	if number == "" {
		if err := tx.Raw("SELECT NEXT VALUE FOR dbo.SEQ_sales_order_numbers").Scan(&sequence).Error; err != nil {
			return err
		}
		number = GeneratedSalesOrderNumber(sequence)
	} else if !ValidSalesOrderNumber(number) {
		return ErrOrderNotReady
	}
	var quotationID *int64
	if source != nil {
		quotationID = &source.ID
	}
	m := orderModel{Number: number, QuotationID: quotationID, AccountID: accountID, PONumber: po, SalesPerson: FixedSalesPerson, TermsDays: terms, Status: Open, Subtotal: totals.Subtotal.Int64(), Tax: totals.Tax.Int64(), Total: totals.Total.Int64(), CreatedAt: time.Now().UTC()}
	if err := tx.Create(&m).Error; err != nil {
		return err
	}
	for index, line := range lines {
		var sourceLineID *int64
		if source != nil {
			id := selections[index].QuotationLineID
			sourceLineID = &id
		}
		stored := lineModel{QuotationLineID: sourceLineID, ProductID: line.ProductID, Quantity: line.Quantity.Int64(), UOM: line.UOM, UnitPrice: line.UnitPrice.Int64(), TaxRate: line.TaxRate, TaxCode: line.TaxCode, LineTotal: line.LineTotal.Int64(), Inclusive: line.VATInclusiveTotal.Int64()}
		stored.SKU, stored.Name = line.ProductSKU, line.ProductName
		if err := tx.Raw(`INSERT INTO dbo.sales_order_lines
            (sales_order_id, quotation_line_id, product_id, quantity_scaled, uom, unit_price_scaled, tax_rate_scaled, tax_code, line_total_scaled, vat_inclusive_total_scaled, sku, name)
            OUTPUT INSERTED.sales_order_line_id
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			m.ID, stored.QuotationLineID, stored.ProductID, stored.Quantity, stored.UOM, stored.UnitPrice,
			stored.TaxRate, stored.TaxCode, stored.LineTotal, stored.Inclusive, stored.SKU, stored.Name).Scan(&stored.ID).Error; err != nil || stored.ID == 0 {
			return errors.New("sales order line identity was not generated")
		}
		if source != nil {
			if err := tx.Table("dbo.quotation_line_allocations").Create(map[string]interface{}{"quotation_id": source.ID, "quotation_line_id": selections[index].QuotationLineID, "sales_order_id": m.ID, "sales_order_line_id": stored.ID, "quantity_scaled": line.Quantity.Int64()}).Error; err != nil {
				return err
			}
		}
	}
	*result = SalesOrder{ID: m.ID, Number: m.Number, QuotationID: sourceID(quotationID), QuotationNumber: sourceNumber(source), CustomerPONumber: po, SalesPerson: m.SalesPerson, Status: m.Status, CreatedAtUTC: m.CreatedAt}
	return nil
}

func sourceID(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func sourceNumber(value *quotations.Quotation) string {
	if value == nil {
		return ""
	}
	return value.Number
}

func (r *GormRepository) lockAllocations(tx *gorm.DB, quotationID int64) (map[int64]money.Amount, error) {
	var rows []struct {
		QuotationLineID int64 `gorm:"column:quotation_line_id"`
		Quantity        int64 `gorm:"column:quantity_scaled"`
	}
	var locks []int64
	if err := tx.Raw("SELECT quotation_line_id FROM dbo.quotation_lines WITH (UPDLOCK, HOLDLOCK) WHERE quotation_id = ?", quotationID).Scan(&locks).Error; err != nil {
		return nil, err
	}
	err := tx.Raw("SELECT quotation_line_id, SUM(quantity_scaled) AS quantity_scaled FROM dbo.quotation_line_allocations WITH (UPDLOCK, HOLDLOCK) WHERE quotation_id = ? GROUP BY quotation_line_id", quotationID).Scan(&rows).Error
	allocated := make(map[int64]money.Amount, len(rows))
	for _, row := range rows {
		allocated[row.QuotationLineID] = money.Amount(row.Quantity)
	}
	return allocated, err
}

func (r *GormRepository) lockAllocationSets(tx *gorm.DB, quotationID, orderID int64) (map[int64]money.Amount, map[int64]money.Amount, error) {
	var locks []int64
	if err := tx.Raw("SELECT quotation_line_id FROM dbo.quotation_lines WITH (UPDLOCK, HOLDLOCK) WHERE quotation_id = ?", quotationID).Scan(&locks).Error; err != nil {
		return nil, nil, err
	}
	all, current := make(map[int64]money.Amount), make(map[int64]money.Amount)
	var rows []struct {
		QuotationLineID int64 `gorm:"column:quotation_line_id"`
		Quantity        int64 `gorm:"column:quantity_scaled"`
	}
	if err := tx.Raw("SELECT quotation_line_id, SUM(quantity_scaled) AS quantity_scaled FROM dbo.quotation_line_allocations WITH (UPDLOCK, HOLDLOCK) WHERE quotation_id = ? GROUP BY quotation_line_id", quotationID).Scan(&rows).Error; err != nil {
		return nil, nil, err
	}
	for _, row := range rows {
		all[row.QuotationLineID] = money.Amount(row.Quantity)
	}
	rows = nil
	if err := tx.Raw("SELECT quotation_line_id, SUM(quantity_scaled) AS quantity_scaled FROM dbo.quotation_line_allocations WITH (UPDLOCK, HOLDLOCK) WHERE sales_order_id = ? GROUP BY quotation_line_id", orderID).Scan(&rows).Error; err != nil {
		return nil, nil, err
	}
	for _, row := range rows {
		current[row.QuotationLineID] = money.Amount(row.Quantity)
	}
	return all, current, nil
}

func (r *GormRepository) AllocationLines(ctx context.Context, q quotations.Quotation) ([]AllocationLine, error) {
	allocated, err := r.lockAllocations(r.db.WithContext(ctx), q.ID)
	if err != nil {
		return nil, err
	}
	result := make([]AllocationLine, 0, len(q.Lines))
	for _, line := range q.Lines {
		converted := allocated[line.ID]
		result = append(result, AllocationLine{Line: line, Converted: converted, Remaining: line.Quantity - converted})
	}
	return result, nil
}

func (r *GormRepository) AllocationLinesForOrder(ctx context.Context, q quotations.Quotation, orderID int64) ([]AllocationLine, error) {
	all, current, err := r.lockAllocationSets(r.db.WithContext(ctx), q.ID, orderID)
	if err != nil {
		return nil, err
	}
	result := make([]AllocationLine, 0, len(q.Lines))
	for _, line := range q.Lines {
		result = append(result, AllocationLine{Line: line, Converted: all[line.ID] - current[line.ID], Current: current[line.ID], Remaining: line.Quantity - all[line.ID] + current[line.ID]})
	}
	return result, nil
}

func (r *GormRepository) ConversionSummary(ctx context.Context, quotationID int64) (quotations.ConversionSummary, error) {
	var orderCount int64
	if err := r.db.WithContext(ctx).Model(&orderModel{}).Where("quotation_id = ?", quotationID).Count(&orderCount).Error; err != nil {
		return quotations.ConversionSummary{}, err
	}
	var converted, total int64
	if err := r.db.WithContext(ctx).Table("dbo.quotation_line_allocations").Where("quotation_id = ?", quotationID).Select("COALESCE(SUM(quantity_scaled), 0)").Scan(&converted).Error; err != nil {
		return quotations.ConversionSummary{}, err
	}
	if err := r.db.WithContext(ctx).Table("dbo.quotation_lines").Where("quotation_id = ?", quotationID).Select("COALESCE(SUM(quantity_scaled), 0)").Scan(&total).Error; err != nil {
		return quotations.ConversionSummary{}, err
	}
	return quotations.ConversionSummary{OrderCount: int(orderCount), Converted: money.Amount(converted), Total: money.Amount(total), Remaining: money.Amount(total - converted)}, nil
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
	if m.QuotationID != nil {
		var source struct {
			Number         string `gorm:"column:quotation_number"`
			TaxDefaultCode string `gorm:"column:tax_default_code"`
		}
		if err := r.db.WithContext(ctx).Table("dbo.quotations").Select("quotation_number, tax_default_code").Where("quotation_id = ?", *m.QuotationID).Scan(&source).Error; err != nil {
			return SalesOrder{}, err
		}
		q.Number, q.TaxDefaultCode = source.Number, source.TaxDefaultCode
	}
	q.TermsDays, q.CreatedAtUTC = m.TermsDays, m.CreatedAt
	q.Totals = quotations.Totals{Subtotal: money.Amount(m.Subtotal), Tax: money.Amount(m.Tax), Total: money.Amount(m.Total)}
	var lines []lineModel
	if err := r.db.WithContext(ctx).Table("dbo.sales_order_lines AS sol").Select("sol.*").Where("sol.sales_order_id = ?", id).Find(&lines).Error; err != nil {
		return SalesOrder{}, err
	}
	for _, line := range lines {
		q.Lines = append(q.Lines, quotations.Line{ID: line.ID, ProductID: line.ProductID, ProductSKU: line.SKU, ProductName: line.Name, Quantity: money.Amount(line.Quantity), UOM: line.UOM, UnitPrice: money.Amount(line.UnitPrice), TaxRate: line.TaxRate, TaxCode: line.TaxCode, LineTotal: money.Amount(line.LineTotal), VATInclusiveTotal: money.Amount(line.Inclusive), TaxAmount: money.Amount(line.Inclusive - line.LineTotal)})
	}
	var invoice struct {
		ID     int64  `gorm:"column:invoice_id"`
		Number string `gorm:"column:invoice_number"`
	}
	if err := r.db.WithContext(ctx).Table("dbo.invoices").Select("invoice_id, invoice_number").Where("sales_order_id = ?", id).Scan(&invoice).Error; err != nil {
		return SalesOrder{}, err
	}
	return SalesOrder{ID: m.ID, InvoiceID: invoice.ID, InvoiceNumber: invoice.Number, Number: m.Number, QuotationID: sourceID(m.QuotationID), QuotationNumber: q.Number, CustomerPONumber: m.PONumber, SalesPerson: m.SalesPerson, Status: m.Status, CreatedAtUTC: m.CreatedAt, Quotation: q}, nil
}

func (r *GormRepository) List(ctx context.Context) ([]SalesOrder, error) {
	var rows []orderModel
	if err := r.db.WithContext(ctx).Order("sales_order_id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]SalesOrder, 0, len(rows))
	for _, row := range rows {
		order, err := r.FindByID(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, order)
	}
	return result, nil
}
