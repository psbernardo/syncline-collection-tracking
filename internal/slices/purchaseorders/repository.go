package purchaseorders

import (
	"context"
	"encoding/json"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
	"gorm.io/gorm"
)

type Repository interface {
	NextNumber(context.Context, time.Time) (string, error)
	CreateFromSalesOrder(context.Context, CreateInput) (PurchaseOrder, error)
	FindByID(context.Context, int64) (PurchaseOrder, error)
	List(context.Context) ([]PurchaseOrder, error)
	Update(context.Context, int64, CreateInput) (PurchaseOrder, error)
}

// Creator is implemented by repositories that support both purchase modes.
type Creator interface {
	Create(context.Context, CreateInput) (PurchaseOrder, error)
}

type Lifecycle interface {
	Confirm(context.Context, int64) error
	Cancel(context.Context, int64) error
}

type ReadinessReader interface {
	AllocationReadiness(context.Context, int64) ([]AllocationStatus, error)
}

func (r *GormRepository) SupplierProducts(ctx context.Context, supplierID int64) (map[int64]SupplierProduct, error) {
	var rows []struct {
		ID, ProductID int64
		SupplierSKU   string `gorm:"column:supplier_sku"`
		ReferenceCost int64  `gorm:"column:reference_cost_scaled"`
	}
	if err := r.db.WithContext(ctx).Table("dbo.supplier_products").Select("supplier_product_id AS id, product_id, supplier_sku, reference_cost_scaled").Where("supplier_id = ? AND is_active = 1", supplierID).Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[int64]SupplierProduct, len(rows))
	for _, row := range rows {
		result[row.ProductID] = SupplierProduct{ID: row.ID, ProductID: row.ProductID, SupplierSKU: row.SupplierSKU, ReferenceCost: money.Amount(row.ReferenceCost)}
	}
	return result, nil
}

type GormRepository struct{ db *gorm.DB }

func NewGormRepository(db *gorm.DB) *GormRepository { return &GormRepository{db: db} }

type orderModel struct {
	ID               int64      `gorm:"column:purchase_order_id;primaryKey"`
	Number           string     `gorm:"column:purchase_order_number"`
	SupplierID       int64      `gorm:"column:supplier_id"`
	SalesOrderID     *int64     `gorm:"column:sales_order_id"`
	PODate           time.Time  `gorm:"column:po_date"`
	Status           string     `gorm:"column:status"`
	PaymentTerms     string     `gorm:"column:payment_terms"`
	ExpectedDelivery *time.Time `gorm:"column:expected_delivery_date"`
	Notes            string     `gorm:"column:notes"`
	Subtotal         int64      `gorm:"column:subtotal_scaled"`
	Total            int64      `gorm:"column:total_scaled"`
}

func (orderModel) TableName() string { return "dbo.purchase_orders" }

func (r *GormRepository) NextNumber(ctx context.Context, date time.Time) (string, error) {
	var sequence int64
	if err := r.db.WithContext(ctx).Raw("SELECT NEXT VALUE FOR dbo.SEQ_purchase_order_numbers").Scan(&sequence).Error; err != nil {
		return "", err
	}
	return GeneratedNumber(date, sequence), nil
}

func (r *GormRepository) Create(ctx context.Context, input CreateInput) (PurchaseOrder, error) {
	if input.Mode == "" {
		if input.SalesOrderID > 0 {
			input.Mode = SalesOrderMode
		} else {
			input.Mode = DirectMode
		}
	}
	if err := input.Validate(); err != nil {
		return PurchaseOrder{}, err
	}
	var result PurchaseOrder
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Table("dbo.suppliers").Where("supplier_id = ? AND is_active = 1", input.SupplierID).Count(&count).Error; err != nil || count != 1 {
			return ErrSupplierRequired
		}
		var salesOrderID *int64
		if input.Mode == SalesOrderMode {
			salesOrderID = &input.SalesOrderID
		}
		m := orderModel{Number: input.Number, SupplierID: input.SupplierID, SalesOrderID: salesOrderID, PODate: input.PODate, Status: "OPEN", PaymentTerms: input.PaymentTerms, ExpectedDelivery: input.ExpectedDelivery, Notes: input.Notes}
		if err := tx.Create(&m).Error; err != nil {
			return err
		}
		var subtotal money.Amount
		for _, line := range input.Lines {
			total, _ := multiply(line.Quantity, line.UnitCost)
			var insert *gorm.DB
			if input.Mode == DirectMode {
				insert = tx.Raw(`INSERT INTO dbo.purchase_order_lines (purchase_order_id, sales_order_id, sales_order_line_id, product_id, supplier_product_id, sku, name, supplier_sku, uom, quantity_scaled, unit_cost_scaled, line_total_scaled) OUTPUT INSERTED.purchase_order_line_id SELECT ?, NULL, NULL, p.product_id, sp.supplier_product_id, p.sku, p.name, sp.supplier_sku, p.uom, ?, ?, (? * ?) / 10000 FROM dbo.products p JOIN dbo.supplier_products sp ON sp.product_id = p.product_id AND sp.supplier_id = ? AND sp.is_active = 1 WHERE p.product_id = ? AND p.is_active = 1`, m.ID, line.Quantity.Int64(), line.UnitCost.Int64(), line.Quantity.Int64(), line.UnitCost.Int64(), input.SupplierID, line.ProductID)
			} else {
				soID := line.SalesOrderID
				if soID == 0 {
					soID = input.SalesOrderID
				}
				var sourceQuantity int64
				if err := tx.Raw("SELECT quantity_scaled FROM dbo.sales_order_lines WITH (UPDLOCK, HOLDLOCK) WHERE sales_order_line_id = ? AND sales_order_id = ?", line.SalesOrderLineID, soID).Scan(&sourceQuantity).Error; err != nil {
					return err
				}
				if sourceQuantity == 0 {
					return ErrNotReady
				}
				var available int64
				if err := tx.Raw(`SELECT ? - COALESCE((SELECT SUM(a.quantity_scaled) FROM dbo.purchase_order_allocations a JOIN dbo.purchase_order_lines pol ON pol.purchase_order_line_id = a.purchase_order_line_id JOIN dbo.purchase_orders p ON p.purchase_order_id = pol.purchase_order_id WHERE a.sales_order_line_id = ? AND p.status <> 'CANCELLED'), 0)`, sourceQuantity, line.SalesOrderLineID).Scan(&available).Error; err != nil {
					return err
				}
				if line.Quantity.Int64() > available {
					return ErrAllocationExceeded
				}
				insert = tx.Raw(`INSERT INTO dbo.purchase_order_lines (purchase_order_id, sales_order_id, sales_order_line_id, product_id, supplier_product_id, sku, name, supplier_sku, uom, quantity_scaled, unit_cost_scaled, line_total_scaled) OUTPUT INSERTED.purchase_order_line_id SELECT ?, sol.sales_order_id, sol.sales_order_line_id, sol.product_id, sp.supplier_product_id, sol.sku, sol.name, sp.supplier_sku, sol.uom, ?, ?, (? * ?) / 10000 FROM dbo.sales_order_lines sol JOIN dbo.supplier_products sp ON sp.product_id = sol.product_id AND sp.supplier_id = ? AND sp.is_active = 1 WHERE sol.sales_order_line_id = ? AND sol.sales_order_id = ?`, m.ID, line.Quantity.Int64(), line.UnitCost.Int64(), line.Quantity.Int64(), line.UnitCost.Int64(), input.SupplierID, line.SalesOrderLineID, soID)
			}
			var lineID int64
			if err := insert.Scan(&lineID).Error; err != nil {
				return err
			}
			if lineID == 0 {
				return ErrMissingSupplierProduct
			}
			if input.Mode == SalesOrderMode {
				soID := line.SalesOrderID
				if soID == 0 {
					soID = input.SalesOrderID
				}
				if err := tx.Exec(`INSERT INTO dbo.purchase_order_allocations (purchase_order_line_id, sales_order_line_id, quantity_scaled) VALUES (?, ?, ?)`, lineID, line.SalesOrderLineID, line.Quantity.Int64()).Error; err != nil {
					return err
				}
				_ = soID
			}
			subtotal += total
		}
		if err := tx.Model(&orderModel{}).Where("purchase_order_id = ?", m.ID).Updates(map[string]interface{}{"subtotal_scaled": subtotal.Int64(), "total_scaled": subtotal.Int64(), "updated_at_utc": gorm.Expr("SYSUTCDATETIME()")}).Error; err != nil {
			return err
		}
		if err := recordAudit(tx, m.ID, "create", nil, map[string]interface{}{"number": m.Number, "supplier_id": m.SupplierID, "mode": input.Mode, "line_count": len(input.Lines), "total_scaled": subtotal.Int64()}); err != nil {
			return err
		}
		result = PurchaseOrder{ID: m.ID, Number: m.Number, SupplierID: m.SupplierID, PODate: m.PODate, PaymentTerms: m.PaymentTerms, ExpectedDelivery: m.ExpectedDelivery, Notes: m.Notes, Status: "OPEN", Mode: input.Mode, Subtotal: subtotal, Total: subtotal}
		return nil
	})
	if err != nil {
		return PurchaseOrder{}, err
	}
	return r.FindByID(ctx, result.ID)
}

func (r *GormRepository) CreateFromSalesOrder(ctx context.Context, input CreateInput) (PurchaseOrder, error) {
	input.Mode = SalesOrderMode
	for i := range input.Lines {
		input.Lines[i].SalesOrderID = input.SalesOrderID
	}
	return r.Create(ctx, input)
	/*
	   	if input.SupplierID < 1 || input.SalesOrderID < 1 || !ValidNumber(input.Number) || input.PODate.IsZero() {
	   		return PurchaseOrder{}, ErrNotReady
	   	}

	   var result PurchaseOrder

	   	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
	   		var supplierCount int64
	   		if err := tx.Table("dbo.suppliers").Where("supplier_id = ? AND is_active = 1", input.SupplierID).Count(&supplierCount).Error; err != nil || supplierCount != 1 {
	   			return ErrSupplierRequired
	   		}
	   		var sourceCount int64
	   		if err := tx.Table("dbo.sales_orders").Where("sales_order_id = ? AND status = ?", input.SalesOrderID, "OPEN").Count(&sourceCount).Error; err != nil || sourceCount != 1 {
	   			return ErrNotReady
	   		}
	   		m := orderModel{Number: input.Number, SupplierID: input.SupplierID, SalesOrderID: input.SalesOrderID, PODate: input.PODate, Status: "OPEN", PaymentTerms: input.PaymentTerms, ExpectedDelivery: input.ExpectedDelivery, Notes: input.Notes}
	   		if err := tx.Create(&m).Error; err != nil {
	   			return err
	   		}
	   		var subtotal money.Amount
	   		for _, line := range input.Lines {
	   			if line.SalesOrderLineID < 1 || line.Quantity <= 0 {
	   				return ErrNotReady
	   			}
	   			total, ok := multiply(line.Quantity, line.UnitCost)
	   			if !ok {
	   				return ErrNotReady
	   			}
	   			// Serialize allocations for this sales line so concurrent supplier POs
	   			// cannot both observe the same remaining quantity.
	   			var sourceQuantity int64
	   			if err := tx.Raw("SELECT quantity_scaled FROM dbo.sales_order_lines WITH (UPDLOCK, HOLDLOCK) WHERE sales_order_line_id = ? AND sales_order_id = ?", line.SalesOrderLineID, input.SalesOrderID).Scan(&sourceQuantity).Error; err != nil {
	   				return err
	   			}
	   			if sourceQuantity == 0 {
	   				return ErrNotReady
	   			}
	   			var available int64
	   			if err := tx.Raw(`SELECT ? - COALESCE((SELECT SUM(a.quantity_scaled) FROM dbo.purchase_order_allocations a JOIN dbo.purchase_order_lines pol ON pol.purchase_order_line_id = a.purchase_order_line_id JOIN dbo.purchase_orders p ON p.purchase_order_id = pol.purchase_order_id WHERE a.sales_order_line_id = ? AND p.status <> 'CANCELLED'), 0)`, sourceQuantity, line.SalesOrderLineID).Scan(&available).Error; err != nil {
	   				return err
	   			}
	   			if line.Quantity.Int64() > available {
	   				return ErrAllocationExceeded
	   			}
	   			var lineID int64
	   			insert := tx.Raw(`INSERT INTO dbo.purchase_order_lines (purchase_order_id, sales_order_line_id, product_id, supplier_product_id, sku, name, supplier_sku, uom, quantity_scaled, unit_cost_scaled, line_total_scaled) OUTPUT INSERTED.purchase_order_line_id SELECT ?, sol.sales_order_line_id, sol.product_id, sp.supplier_product_id, sol.sku, sol.name, sp.supplier_sku, sol.uom, ?, ?, ( ? * ? ) / 10000 FROM dbo.sales_order_lines sol JOIN dbo.supplier_products sp ON sp.product_id = sol.product_id AND sp.supplier_id = ? AND sp.is_active = 1 WHERE sol.sales_order_line_id = ? AND sol.sales_order_id = ?`, m.ID, line.Quantity.Int64(), line.UnitCost.Int64(), line.Quantity.Int64(), line.UnitCost.Int64(), input.SupplierID, line.SalesOrderLineID, input.SalesOrderID).Scan(&lineID)
	   			if insert.Error != nil {
	   				return insert.Error
	   			}
	   			if lineID == 0 {
	   				return ErrMissingSupplierProduct
	   			}
	   			if err := tx.Exec(`INSERT INTO dbo.purchase_order_allocations (purchase_order_line_id, sales_order_line_id, quantity_scaled) VALUES (?, ?, ?)`, lineID, line.SalesOrderLineID, line.Quantity.Int64()).Error; err != nil {
	   				return err
	   			}
	   			subtotal += total
	   		}
	   		if len(input.Lines) == 0 {
	   			return ErrNotReady
	   		}
	   		if err := tx.Model(&orderModel{}).Where("purchase_order_id = ?", m.ID).Updates(map[string]interface{}{"subtotal_scaled": subtotal.Int64(), "total_scaled": subtotal.Int64(), "updated_at_utc": gorm.Expr("SYSUTCDATETIME()")}).Error; err != nil {
	   			return err
	   		}
	   		if err := recordAudit(tx, m.ID, "create", nil, map[string]interface{}{"number": m.Number, "supplier_id": m.SupplierID, "sales_order_id": m.SalesOrderID, "status": m.Status, "line_count": len(input.Lines), "total_scaled": subtotal.Int64()}); err != nil {
	   			return err
	   		}
	   		result = PurchaseOrder{ID: m.ID, Number: input.Number, SupplierID: input.SupplierID, SalesOrderID: input.SalesOrderID, PODate: input.PODate, PaymentTerms: input.PaymentTerms, ExpectedDelivery: input.ExpectedDelivery, Notes: input.Notes, Status: "OPEN", Subtotal: subtotal, Total: subtotal}
	   		return nil
	   	})

	   	if err != nil {
	   		if errors.Is(err, gorm.ErrDuplicatedKey) {
	   			return PurchaseOrder{}, ErrNotReady
	   		}
	   		return PurchaseOrder{}, err
	   	}

	   return result, nil
	*/
}

func (r *GormRepository) FindByID(ctx context.Context, id int64) (PurchaseOrder, error) {
	var row struct {
		ID               int64      `gorm:"column:purchase_order_id"`
		Number           string     `gorm:"column:purchase_order_number"`
		SupplierID       int64      `gorm:"column:supplier_id"`
		SalesOrderID     int64      `gorm:"column:sales_order_id"`
		PODate           time.Time  `gorm:"column:po_date"`
		Status           string     `gorm:"column:status"`
		PaymentTerms     string     `gorm:"column:payment_terms"`
		ExpectedDelivery *time.Time `gorm:"column:expected_delivery_date"`
		Notes            string     `gorm:"column:notes"`
		Subtotal         int64      `gorm:"column:subtotal_scaled"`
		Total            int64      `gorm:"column:total_scaled"`
		SupplierName     string     `gorm:"column:supplier_name"`
		SalesOrderNo     string     `gorm:"column:sales_order_number"`
	}
	query := r.db.WithContext(ctx).Table("dbo.purchase_orders AS po").Select("po.purchase_order_id AS purchase_order_id, po.purchase_order_number AS purchase_order_number, po.supplier_id AS supplier_id, COALESCE(po.sales_order_id, 0) AS sales_order_id, po.po_date AS po_date, po.status AS status, po.payment_terms, po.expected_delivery_date, po.notes, po.subtotal_scaled AS subtotal_scaled, po.total_scaled AS total_scaled, s.name AS supplier_name, COALESCE(so.sales_order_number, '') AS sales_order_number").Joins("JOIN dbo.suppliers s ON s.supplier_id = po.supplier_id").Joins("LEFT JOIN dbo.sales_orders so ON so.sales_order_id = po.sales_order_id").Where("po.purchase_order_id = ?", id).Scan(&row)
	if query.Error != nil {
		return PurchaseOrder{}, query.Error
	}
	if query.RowsAffected == 0 {
		return PurchaseOrder{}, gorm.ErrRecordNotFound
	}
	var lines []struct {
		ID, SalesOrderID, SalesOrderLineID, ProductID, SupplierProductID int64
		SKU, Name, SupplierSKU, UOM                                      string
		Quantity, UnitCost, Total                                        int64
	}
	if err := r.db.WithContext(ctx).Table("dbo.purchase_order_lines").Select("purchase_order_line_id AS id, COALESCE(sales_order_id, 0) AS sales_order_id, COALESCE(sales_order_line_id, 0) AS sales_order_line_id, product_id, COALESCE(supplier_product_id, 0) AS supplier_product_id, sku, name, COALESCE(supplier_sku, '') AS supplier_sku, uom, quantity_scaled AS quantity, unit_cost_scaled AS unit_cost, line_total_scaled AS total").Where("purchase_order_id = ?", id).Order("purchase_order_line_id").Find(&lines).Error; err != nil {
		return PurchaseOrder{}, err
	}
	result := PurchaseOrder{ID: row.ID, Number: row.Number, SupplierID: row.SupplierID, SupplierName: row.SupplierName, SalesOrderID: row.SalesOrderID, SalesOrderNumber: row.SalesOrderNo, PODate: row.PODate, PaymentTerms: row.PaymentTerms, ExpectedDelivery: row.ExpectedDelivery, Notes: row.Notes, Status: row.Status, Subtotal: money.Amount(row.Subtotal), Total: money.Amount(row.Total), Mode: DirectMode}
	var sourceNumbers []string
	if err := r.db.WithContext(ctx).Table("dbo.purchase_order_lines AS pol").Joins("JOIN dbo.sales_orders so ON so.sales_order_id = pol.sales_order_id").Where("pol.purchase_order_id = ?", id).Distinct().Order("so.sales_order_number").Pluck("so.sales_order_number", &sourceNumbers).Error; err != nil {
		return PurchaseOrder{}, err
	}
	result.SalesOrderNumbers = sourceNumbers
	if len(sourceNumbers) > 0 || result.SalesOrderID != 0 {
		result.Mode = SalesOrderMode
	}
	for _, line := range lines {
		result.Lines = append(result.Lines, Line{ID: line.ID, SalesOrderID: line.SalesOrderID, SalesOrderLineID: line.SalesOrderLineID, ProductID: line.ProductID, SupplierProductID: line.SupplierProductID, SKU: line.SKU, Name: line.Name, SupplierSKU: line.SupplierSKU, UOM: line.UOM, Quantity: money.Amount(line.Quantity), UnitCost: money.Amount(line.UnitCost), Total: money.Amount(line.Total)})
	}
	return result, nil
}

func (r *GormRepository) List(ctx context.Context) ([]PurchaseOrder, error) {
	var ids []int64
	if err := r.db.WithContext(ctx).Table("dbo.purchase_orders").Order("purchase_order_id DESC").Pluck("purchase_order_id", &ids).Error; err != nil {
		return nil, err
	}
	result := make([]PurchaseOrder, 0, len(ids))
	for _, id := range ids {
		item, err := r.FindByID(ctx, id)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func (r *GormRepository) Update(ctx context.Context, id int64, input CreateInput) (PurchaseOrder, error) {
	if input.Mode == "" {
		if input.SalesOrderID > 0 {
			input.Mode = SalesOrderMode
		} else {
			input.Mode = DirectMode
		}
	}
	if id < 1 || input.SupplierID < 1 || !ValidNumber(input.Number) || input.PODate.IsZero() || len(input.Lines) == 0 {
		return PurchaseOrder{}, ErrNotReady
	}
	if err := input.Validate(); err != nil {
		return PurchaseOrder{}, err
	}
	if input.Mode == DirectMode {
		return r.updateDirect(ctx, id, input)
	}
	if input.SalesOrderID < 1 {
		return PurchaseOrder{}, ErrNotReady
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current orderModel
		if err := tx.Raw("SELECT * FROM dbo.purchase_orders WITH (UPDLOCK, HOLDLOCK) WHERE purchase_order_id = ?", id).Scan(&current).Error; err != nil || current.ID == 0 || current.Status != "OPEN" || current.SalesOrderID == nil || *current.SalesOrderID != input.SalesOrderID {
			return ErrNotReady
		}
		var count int64
		if err := tx.Table("dbo.suppliers").Where("supplier_id = ? AND is_active = 1", input.SupplierID).Count(&count).Error; err != nil || count != 1 {
			return ErrSupplierRequired
		}
		if err := tx.Model(&orderModel{}).Where("purchase_order_id = ?", id).Updates(map[string]interface{}{"supplier_id": input.SupplierID, "po_date": input.PODate, "payment_terms": input.PaymentTerms, "expected_delivery_date": input.ExpectedDelivery, "notes": input.Notes, "subtotal_scaled": 0, "total_scaled": 0, "updated_at_utc": gorm.Expr("SYSUTCDATETIME()")}).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM dbo.purchase_order_allocations WHERE purchase_order_line_id IN (SELECT purchase_order_line_id FROM dbo.purchase_order_lines WHERE purchase_order_id = ?)", id).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM dbo.purchase_order_lines WHERE purchase_order_id = ?", id).Error; err != nil {
			return err
		}
		var subtotal money.Amount
		for _, line := range input.Lines {
			if line.SalesOrderLineID < 1 || line.Quantity <= 0 {
				return ErrNotReady
			}
			total, ok := multiply(line.Quantity, line.UnitCost)
			if !ok {
				return ErrNotReady
			}
			subtotal += total
			var lineID int64
			var sourceQuantity int64
			if err := tx.Raw("SELECT quantity_scaled FROM dbo.sales_order_lines WITH (UPDLOCK, HOLDLOCK) WHERE sales_order_line_id = ? AND sales_order_id = ?", line.SalesOrderLineID, input.SalesOrderID).Scan(&sourceQuantity).Error; err != nil {
				return err
			}
			if sourceQuantity == 0 || line.Quantity.Int64() > sourceQuantity {
				return ErrAllocationExceeded
			}
			var alreadyAllocated int64
			if err := tx.Raw(`SELECT COALESCE(SUM(a.quantity_scaled), 0) FROM dbo.purchase_order_allocations a JOIN dbo.purchase_order_lines pol ON pol.purchase_order_line_id = a.purchase_order_line_id JOIN dbo.purchase_orders p ON p.purchase_order_id = pol.purchase_order_id WHERE a.sales_order_line_id = ? AND p.status <> 'CANCELLED'`, line.SalesOrderLineID).Scan(&alreadyAllocated).Error; err != nil {
				return err
			}
			if line.Quantity.Int64() > sourceQuantity-alreadyAllocated {
				return ErrAllocationExceeded
			}
			insert := tx.Raw(`INSERT INTO dbo.purchase_order_lines (purchase_order_id, sales_order_id, sales_order_line_id, product_id, supplier_product_id, sku, name, supplier_sku, uom, quantity_scaled, unit_cost_scaled, line_total_scaled) OUTPUT INSERTED.purchase_order_line_id SELECT ?, sol.sales_order_id, sol.sales_order_line_id, sol.product_id, sp.supplier_product_id, sol.sku, sol.name, sp.supplier_sku, sol.uom, ?, ?, (? * ?) / 10000 FROM dbo.sales_order_lines sol JOIN dbo.supplier_products sp ON sp.product_id = sol.product_id AND sp.supplier_id = ? AND sp.is_active = 1 WHERE sol.sales_order_line_id = ? AND sol.sales_order_id = ?`, id, line.Quantity.Int64(), line.UnitCost.Int64(), line.Quantity.Int64(), line.UnitCost.Int64(), input.SupplierID, line.SalesOrderLineID, input.SalesOrderID).Scan(&lineID)
			if insert.Error != nil {
				return insert.Error
			}
			if lineID == 0 {
				return ErrMissingSupplierProduct
			}
			if err := tx.Exec(`INSERT INTO dbo.purchase_order_allocations (purchase_order_line_id, sales_order_line_id, quantity_scaled) VALUES (?, ?, ?)`, lineID, line.SalesOrderLineID, line.Quantity.Int64()).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&orderModel{}).Where("purchase_order_id = ?", id).Updates(map[string]interface{}{"subtotal_scaled": subtotal.Int64(), "total_scaled": subtotal.Int64(), "updated_at_utc": gorm.Expr("SYSUTCDATETIME()")}).Error; err != nil {
			return err
		}
		return recordAudit(tx, id, "update", map[string]string{"status": "OPEN"}, map[string]interface{}{"status": "OPEN", "line_count": len(input.Lines), "total_scaled": subtotal.Int64()})
	})
	if err != nil {
		return PurchaseOrder{}, err
	}
	return r.FindByID(ctx, id)
}

func (r *GormRepository) updateDirect(ctx context.Context, id int64, input CreateInput) (PurchaseOrder, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current orderModel
		if err := tx.Raw("SELECT * FROM dbo.purchase_orders WITH (UPDLOCK, HOLDLOCK) WHERE purchase_order_id = ?", id).Scan(&current).Error; err != nil || current.ID == 0 || current.Status != "OPEN" {
			return ErrNotReady
		}
		var count int64
		if err := tx.Table("dbo.suppliers").Where("supplier_id = ? AND is_active = 1", input.SupplierID).Count(&count).Error; err != nil || count != 1 {
			return ErrSupplierRequired
		}
		if current.SalesOrderID != nil {
			return ErrMixedModes
		}
		if err := tx.Exec("DELETE FROM dbo.purchase_order_allocations WHERE purchase_order_line_id IN (SELECT purchase_order_line_id FROM dbo.purchase_order_lines WHERE purchase_order_id = ?)", id).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM dbo.purchase_order_lines WHERE purchase_order_id = ?", id).Error; err != nil {
			return err
		}
		if err := tx.Model(&orderModel{}).Where("purchase_order_id = ?", id).Updates(map[string]interface{}{"supplier_id": input.SupplierID, "po_date": input.PODate, "payment_terms": input.PaymentTerms, "expected_delivery_date": input.ExpectedDelivery, "notes": input.Notes, "subtotal_scaled": 0, "total_scaled": 0, "updated_at_utc": gorm.Expr("SYSUTCDATETIME()")}).Error; err != nil {
			return err
		}
		var subtotal money.Amount
		for _, line := range input.Lines {
			total, ok := multiply(line.Quantity, line.UnitCost)
			if !ok {
				return ErrNotReady
			}
			subtotal += total
			var lineID int64
			insert := tx.Raw(`INSERT INTO dbo.purchase_order_lines (purchase_order_id, sales_order_id, sales_order_line_id, product_id, supplier_product_id, sku, name, supplier_sku, uom, quantity_scaled, unit_cost_scaled, line_total_scaled) OUTPUT INSERTED.purchase_order_line_id SELECT ?, NULL, NULL, p.product_id, sp.supplier_product_id, p.sku, p.name, sp.supplier_sku, p.uom, ?, ?, (? * ?) / 10000 FROM dbo.products p JOIN dbo.supplier_products sp ON sp.product_id = p.product_id AND sp.supplier_id = ? AND sp.is_active = 1 WHERE p.product_id = ? AND p.is_active = 1`, id, line.Quantity.Int64(), line.UnitCost.Int64(), line.Quantity.Int64(), line.UnitCost.Int64(), input.SupplierID, line.ProductID).Scan(&lineID)
			if insert.Error != nil {
				return insert.Error
			}
			if lineID == 0 {
				return ErrNotReady
			}
		}
		if err := tx.Model(&orderModel{}).Where("purchase_order_id = ?", id).Updates(map[string]interface{}{"subtotal_scaled": subtotal.Int64(), "total_scaled": subtotal.Int64(), "updated_at_utc": gorm.Expr("SYSUTCDATETIME()")}).Error; err != nil {
			return err
		}
		return recordAudit(tx, id, "update", map[string]string{"status": "OPEN"}, map[string]interface{}{"status": "OPEN", "mode": DirectMode, "line_count": len(input.Lines), "total_scaled": subtotal.Int64()})
	})
	if err != nil {
		return PurchaseOrder{}, err
	}
	return r.FindByID(ctx, id)
}

func (r *GormRepository) Confirm(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current orderModel
		if err := tx.Raw("SELECT * FROM dbo.purchase_orders WITH (UPDLOCK, HOLDLOCK) WHERE purchase_order_id = ?", id).Scan(&current).Error; err != nil || current.ID == 0 || current.Status != "OPEN" {
			return ErrNotReady
		}
		var lockedLines []int64
		_ = lockedLines
		if err := tx.Table("dbo.purchase_orders").Where("purchase_order_id = ?", id).Update("status", "CONFIRMED").Error; err != nil {
			return err
		}
		return recordAudit(tx, id, "confirm", map[string]string{"status": "OPEN"}, map[string]string{"status": "CONFIRMED"})
	})
}

func (r *GormRepository) Cancel(ctx context.Context, id int64) error {
	var previous string
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current orderModel
		if err := tx.Raw("SELECT * FROM dbo.purchase_orders WITH (UPDLOCK, HOLDLOCK) WHERE purchase_order_id = ?", id).Scan(&current).Error; err != nil || current.ID == 0 || (current.Status != "OPEN" && current.Status != "CONFIRMED") {
			return ErrNotReady
		}
		previous = current.Status
		if err := tx.Table("dbo.purchase_orders").Where("purchase_order_id = ?", id).Update("status", "CANCELLED").Error; err != nil {
			return err
		}
		return recordAudit(tx, id, "cancel", map[string]string{"status": previous}, map[string]string{"status": "CANCELLED"})
	})
	if err != nil {
		return err
	}
	return nil
}

func recordAudit(tx *gorm.DB, entityID int64, action string, previous, current interface{}) error {
	before, err := json.Marshal(previous)
	if err != nil {
		return err
	}
	after, err := json.Marshal(current)
	if err != nil {
		return err
	}
	return tx.Exec(`INSERT INTO dbo.audit_events (entity_type, entity_id, action, actor_id, request_id, idempotency_key, previous_values_json, new_values_json) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, "purchase_order", entityID, action, "local-admin", "", "", string(before), string(after)).Error
}

func (r *GormRepository) AllocationReadiness(ctx context.Context, salesOrderID int64) ([]AllocationStatus, error) {
	var rows []struct {
		ID                 int64 `gorm:"column:id"`
		SKU, Name          string
		Ordered, Allocated int64
	}
	err := r.db.WithContext(ctx).Raw(`SELECT sol.sales_order_line_id AS id, sol.sku, sol.name, sol.quantity_scaled AS ordered, COALESCE((SELECT SUM(a.quantity_scaled) FROM dbo.purchase_order_allocations a JOIN dbo.purchase_order_lines pol ON pol.purchase_order_line_id = a.purchase_order_line_id JOIN dbo.purchase_orders po ON po.purchase_order_id = pol.purchase_order_id WHERE a.sales_order_line_id = sol.sales_order_line_id AND po.status <> 'CANCELLED'), 0) AS allocated FROM dbo.sales_order_lines sol WHERE sol.sales_order_id = ? ORDER BY sol.sales_order_line_id`, salesOrderID).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make([]AllocationStatus, 0, len(rows))
	for _, row := range rows {
		result = append(result, AllocationStatus{SalesOrderLineID: row.ID, SKU: row.SKU, Name: row.Name, Ordered: money.Amount(row.Ordered), Allocated: money.Amount(row.Allocated), Unallocated: money.Amount(row.Ordered - row.Allocated)})
	}
	return result, nil
}
