package purchaseorders

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
	"github.com/psbernardo/syncline-collection-tracking/internal/slices/salesorders"
)

var (
	ErrNotReady               = errors.New("purchase order is not ready for creation")
	ErrSupplierRequired       = errors.New("select an active supplier")
	ErrMissingSupplierProduct = errors.New("one or more sales-order products are not configured for this supplier")
	ErrAllocationExceeded     = errors.New("purchase quantity exceeds the unallocated sales quantity")
	ErrNotFullyAllocated      = errors.New("all sales-order quantities must be allocated before confirmation")
	ErrMixedModes             = errors.New("purchase order cannot mix sales-order and direct purchase lines")
	ErrDuplicateLine          = errors.New("purchase order contains a duplicate line")
)

type Mode string

const (
	SalesOrderMode Mode = "SALES_ORDER"
	DirectMode     Mode = "DIRECT"
)

type ProductReference struct {
	ProductID int64
	SKU, Name, UOM string
}

type MissingSupplierProduct struct {
	ProductID int64
	SKU, Name string
}

type MissingSupplierProductsError struct {
	Products []MissingSupplierProduct
}

func (e MissingSupplierProductsError) Error() string { return ErrMissingSupplierProduct.Error() }
func (e MissingSupplierProductsError) Is(target error) bool {
	return target == ErrMissingSupplierProduct
}

type Line struct {
	ID, SalesOrderID, SalesOrderLineID, ProductID, SupplierProductID int64
	SKU, Name, SupplierSKU, UOM                                      string
	Quantity, UnitCost, Total                                        money.Amount
}

type PurchaseOrder struct {
	ID, SupplierID, SalesOrderID                   int64
	Number, SupplierName, SalesOrderNumber, Status string
	PODate                                         time.Time
	PaymentTerms, Notes                            string
	ExpectedDelivery                               *time.Time
	Subtotal, Total                                money.Amount
	Mode                                           Mode
	SalesOrderNumbers                              []string
	Lines                                          []Line
}

type AllocationStatus struct {
	SalesOrderLineID                int64
	SKU, Name                       string
	Ordered, Allocated, Unallocated money.Amount
}

type CreateInput struct {
	Number           string
	SupplierID       int64
	SalesOrderID     int64
	Mode             Mode
	PODate           time.Time
	PaymentTerms     string
	ExpectedDelivery *time.Time
	Notes            string
	Lines            []LineCost
}

type LineCost struct {
	ProductID        int64
	SalesOrderID     int64
	SalesOrderLineID int64
	Quantity         money.Amount
	UnitCost         money.Amount
}

type PurchaseOrderLineInput = LineCost

func (i CreateInput) Validate() error {
	if i.SupplierID < 1 || !ValidNumber(i.Number) || i.PODate.IsZero() || len(i.Lines) == 0 {
		return ErrNotReady
	}
	mode := i.Mode
	if mode == "" {
		if i.SalesOrderID > 0 {
			mode = SalesOrderMode
		} else {
			mode = DirectMode
		}
	}
	if mode != SalesOrderMode && mode != DirectMode {
		return ErrNotReady
	}
	seenSources, seenProducts := map[int64]bool{}, map[int64]bool{}
	for _, line := range i.Lines {
		if line.Quantity <= 0 || line.UnitCost < 0 {
			return ErrNotReady
		}
		if _, ok := multiply(line.Quantity, line.UnitCost); !ok {
			return ErrNotReady
		}
		if mode == SalesOrderMode {
			if line.SalesOrderLineID < 1 || (line.SalesOrderID < 1 && i.SalesOrderID < 1) || seenSources[line.SalesOrderLineID] {
				return ErrDuplicateLine
			}
			seenSources[line.SalesOrderLineID] = true
		} else {
			if line.ProductID < 1 || line.SalesOrderID != 0 || line.SalesOrderLineID != 0 || seenProducts[line.ProductID] {
				return ErrDuplicateLine
			}
			seenProducts[line.ProductID] = true
		}
	}
	return nil
}

var numberPattern = regexp.MustCompile(`^PO-[0-9]{8}-[0-9]{4,}$`)

func GeneratedNumber(date time.Time, sequence int64) string {
	return fmt.Sprintf("PO-%s-%04d", date.Format("20060102"), sequence)
}
func ValidNumber(value string) bool { return numberPattern.MatchString(value) }

func ValidateSupplierAvailability(products []ProductReference, supplierProducts map[int64]SupplierProduct) error {
	missing := make([]MissingSupplierProduct, 0)
	seen := make(map[int64]bool, len(products))
	for _, product := range products {
		mapping := supplierProducts[product.ProductID]
		if mapping.ProductID != product.ProductID && !seen[product.ProductID] {
			missing = append(missing, MissingSupplierProduct{ProductID: product.ProductID, SKU: product.SKU, Name: product.Name})
			seen[product.ProductID] = true
		}
	}
	if len(missing) > 0 {
		return MissingSupplierProductsError{Products: missing}
	}
	return nil
}

func BuildLines(order salesorders.SalesOrder, costs []LineCost, supplierProducts map[int64]SupplierProduct) ([]Line, error) {
	byID := make(map[int64]LineCost, len(costs))
	for _, cost := range costs {
		if cost.SalesOrderLineID > 0 {
			byID[cost.SalesOrderLineID] = cost
		}
	}
	if len(order.Quotation.Lines) == 0 {
		return nil, ErrNotReady
	}
	products := make([]ProductReference, 0, len(byID))
	for _, source := range order.Quotation.Lines {
		if _, selected := byID[source.ID]; selected {
			products = append(products, ProductReference{ProductID: source.ProductID, SKU: source.ProductSKU, Name: source.ProductName})
		}
	}
	if err := ValidateSupplierAvailability(products, supplierProducts); err != nil {
		return nil, err
	}
	lines := make([]Line, 0, len(byID))
	var subtotal money.Amount
	for _, source := range order.Quotation.Lines {
		cost, ok := byID[source.ID]
		if !ok {
			continue
		}
		if cost.UnitCost < 0 || cost.Quantity <= 0 || cost.Quantity > source.Quantity {
			return nil, ErrNotReady
		}
		mapping := supplierProducts[source.ProductID]
		total, ok := multiply(cost.Quantity, cost.UnitCost)
		if !ok {
			return nil, ErrNotReady
		}
		lines = append(lines, Line{SalesOrderLineID: source.ID, ProductID: source.ProductID, SupplierProductID: mapping.ID, SKU: source.ProductSKU, Name: source.ProductName, SupplierSKU: mapping.SupplierSKU, UOM: source.UOM, Quantity: cost.Quantity, UnitCost: cost.UnitCost, Total: total})
		subtotal += total
	}
	for id := range byID {
		found := false
		for _, source := range order.Quotation.Lines {
			if source.ID == id {
				found = true
				break
			}
		}
		if !found {
			return nil, ErrNotReady
		}
	}
	return lines, nil
}

// BuildDirectLines creates supplier snapshots without introducing a sales-order allocation.
func BuildDirectLines(products []ProductReference, costs []LineCost, supplierProducts map[int64]SupplierProduct) ([]Line, error) {
	byID := make(map[int64]ProductReference, len(products))
	for _, product := range products { if product.ProductID < 1 { return nil, ErrNotReady }; byID[product.ProductID] = product }
	seen := make(map[int64]bool, len(costs))
	lines := make([]Line, 0, len(costs))
	for _, cost := range costs {
		product, exists := byID[cost.ProductID]
		mapping, mapped := supplierProducts[cost.ProductID]
		if !exists || !mapped || seen[cost.ProductID] || cost.SalesOrderID != 0 || cost.SalesOrderLineID != 0 || cost.Quantity <= 0 || cost.UnitCost < 0 { return nil, ErrNotReady }
		total, ok := multiply(cost.Quantity, cost.UnitCost); if !ok { return nil, ErrNotReady }
		seen[cost.ProductID] = true
		lines = append(lines, Line{ProductID: cost.ProductID, SupplierProductID: mapping.ID, SKU: product.SKU, Name: product.Name, SupplierSKU: mapping.SupplierSKU, UOM: product.UOM, Quantity: cost.Quantity, UnitCost: cost.UnitCost, Total: total})
	}
	if len(lines) == 0 { return nil, ErrNotReady }
	return lines, nil
}

type SupplierProduct struct {
	ID, ProductID int64
	SupplierSKU   string
	ReferenceCost money.Amount
}

func multiply(quantity, unitCost money.Amount) (money.Amount, bool) {
	if quantity < 0 || unitCost < 0 || quantity > money.Amount(math.MaxInt64)/unitCost {
		return 0, false
	}
	return quantity * unitCost / 10000, true
}
