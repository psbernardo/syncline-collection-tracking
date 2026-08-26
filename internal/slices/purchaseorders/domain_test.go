package purchaseorders

import (
	"errors"
	"testing"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
	"github.com/psbernardo/syncline-collection-tracking/internal/slices/quotations"
	"github.com/psbernardo/syncline-collection-tracking/internal/slices/salesorders"
)

func TestGeneratedNumberUsesDateAndSequence(t *testing.T) {
	got := GeneratedNumber(time.Date(2026, time.August, 23, 0, 0, 0, 0, time.UTC), 12)
	if got != "PO-20260823-0012" || !ValidNumber(got) {
		t.Fatalf("unexpected purchase order number: %q", got)
	}
}

func TestBuildLinesCopiesAllSalesOrderLinesAndSupplierData(t *testing.T) {
	order := salesorders.SalesOrder{Quotation: quotations.Quotation{Lines: []quotations.Line{
		{ID: 10, ProductID: 1, ProductSKU: "SKU-1", ProductName: "Widget", UOM: "PC", Quantity: money.Amount(2 * 10000)},
		{ID: 11, ProductID: 2, ProductSKU: "SKU-2", ProductName: "Gadget", UOM: "BOX", Quantity: money.Amount(3 * 10000)},
	}}}
	lines, err := BuildLines(order, []LineCost{{SalesOrderLineID: 10, Quantity: 20000, UnitCost: money.Amount(50000)}, {SalesOrderLineID: 11, Quantity: 30000, UnitCost: money.Amount(70000)}}, map[int64]SupplierProduct{
		1: {ID: 21, ProductID: 1, SupplierSKU: "SUP-1"}, 2: {ID: 22, ProductID: 2, SupplierSKU: "SUP-2"},
	})
	if err != nil || len(lines) != 2 || lines[0].SupplierSKU != "SUP-1" || lines[1].Total != money.Amount(210000) {
		t.Fatalf("unexpected purchase lines: %+v, error=%v", lines, err)
	}
}

func TestBuildLinesRequiresSupplierProductMapping(t *testing.T) {
	order := salesorders.SalesOrder{Quotation: quotations.Quotation{Lines: []quotations.Line{{ID: 10, ProductID: 1, Quantity: 10000}}}}
	_, err := BuildLines(order, []LineCost{{SalesOrderLineID: 10, Quantity: 10000}}, map[int64]SupplierProduct{})
	if !errors.Is(err, ErrMissingSupplierProduct) {
		t.Fatalf("expected missing supplier product error, got %v", err)
	}
}

func TestValidateSupplierAvailabilityRequiresActiveMappings(t *testing.T) {
	err := ValidateSupplierAvailability([]ProductReference{{ProductID: 1, SKU: "SKU-1", Name: "Widget"}, {ProductID: 2, SKU: "SKU-2", Name: "Gadget"}}, map[int64]SupplierProduct{
		1: {ID: 21, ProductID: 1},
	})
	var missing MissingSupplierProductsError
	if !errors.As(err, &missing) || len(missing.Products) != 1 || missing.Products[0].SKU != "SKU-2" {
		t.Fatalf("expected missing supplier product error, got %v", err)
	}
}

func TestValidateSupplierAvailabilityAcceptsAllMappings(t *testing.T) {
	if err := ValidateSupplierAvailability([]ProductReference{{ProductID: 1}, {ProductID: 2}}, map[int64]SupplierProduct{
		1: {ID: 21, ProductID: 1},
		2: {ID: 22, ProductID: 2},
	}); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestValidateSupplierAvailabilityDeduplicatesProducts(t *testing.T) {
	err := ValidateSupplierAvailability([]ProductReference{
		{ProductID: 1, SKU: "SKU-1", Name: "Widget"},
		{ProductID: 1, SKU: "SKU-1", Name: "Widget"},
	}, map[int64]SupplierProduct{})
	var missing MissingSupplierProductsError
	if !errors.As(err, &missing) || len(missing.Products) != 1 {
		t.Fatalf("expected one missing product, got %v", err)
	}
}

func TestBuildLinesSupportsPartialSelection(t *testing.T) {
	order := salesorders.SalesOrder{Quotation: quotations.Quotation{Lines: []quotations.Line{
		{ID: 10, ProductID: 1, ProductSKU: "SKU-1", Quantity: 10000},
		{ID: 11, ProductID: 2, ProductSKU: "SKU-2", Quantity: 20000},
	}}}
	lines, err := BuildLines(order, []LineCost{{SalesOrderLineID: 10, Quantity: 5000, UnitCost: 10000}}, map[int64]SupplierProduct{1: {ID: 21, ProductID: 1}})
	if err != nil || len(lines) != 1 || lines[0].Quantity != 5000 {
		t.Fatalf("unexpected partial selection: %+v, error=%v", lines, err)
	}
}

func TestBuildLinesRejectsQuantityAboveSalesLine(t *testing.T) {
	order := salesorders.SalesOrder{Quotation: quotations.Quotation{Lines: []quotations.Line{{ID: 10, ProductID: 1, Quantity: 10000}}}}
	_, err := BuildLines(order, []LineCost{{SalesOrderLineID: 10, Quantity: 10001, UnitCost: 10000}}, map[int64]SupplierProduct{1: {ID: 21, ProductID: 1}})
	if !errors.Is(err, ErrNotReady) {
		t.Fatalf("expected quantity validation error, got %v", err)
	}
}

func TestBuildLinesRejectsUnknownSourceLine(t *testing.T) {
	order := salesorders.SalesOrder{Quotation: quotations.Quotation{Lines: []quotations.Line{{ID: 10, ProductID: 1, Quantity: 10000}}}}
	_, err := BuildLines(order, []LineCost{{SalesOrderLineID: 99, Quantity: 10000, UnitCost: 10000}}, map[int64]SupplierProduct{1: {ID: 21, ProductID: 1}})
	if !errors.Is(err, ErrNotReady) {
		t.Fatalf("expected source ownership error, got %v", err)
	}
}
