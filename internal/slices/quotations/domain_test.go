package quotations

import (
	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
	"testing"
)

func TestCalculateTotals(t *testing.T) {
	price, _ := money.Parse("502")
	commission, _ := money.Parse("2")
	totals, err := CalculateTotals([]Line{{Quantity: 1000000, UnitPrice: price, SupplierCost: money.Amount(400000)}}, PerUnit, commission, 0, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if totals.Total != 502000000 || totals.Commission != 2000000 {
		t.Fatalf("unexpected totals: %+v", totals)
	}
}

func TestApplyTaxRuleUsesQuotationDefault(t *testing.T) {
	lines, err := applyTaxRule([]Line{{}}, TaxVAT12)
	if err != nil {
		t.Fatal(err)
	}
	if lines[0].TaxCode != TaxVAT12 || lines[0].TaxRate != 120000 {
		t.Fatalf("unexpected tax snapshot: %+v", lines[0])
	}
	if _, err := applyTaxRule([]Line{{}}, "UNKNOWN"); err == nil {
		t.Fatal("expected unknown tax code error")
	}
}

func TestCalculateTotalsAppliesTaxToAggregateSubtotal(t *testing.T) {
	priceA, _ := money.Parse("100")
	priceB, _ := money.Parse("200")
	lines, err := applyTaxRule([]Line{{Quantity: 10000, UnitPrice: priceA}, {Quantity: 10000, UnitPrice: priceB}}, TaxVAT12)
	if err != nil {
		t.Fatal(err)
	}
	totals, err := CalculateTotals(lines, NoCommission, 0, 0, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if totals.Subtotal != 2678571 || totals.Tax != 321429 || totals.Total != 3000000 {
		t.Fatalf("unexpected aggregate totals: %+v", totals)
	}
}

func TestCalculateProfitabilityReturnsLineAndQuotationMargins(t *testing.T) {
	price, _ := money.Parse("100")
	cost, _ := money.Parse("60")
	delivery, _ := money.Parse("10")
	lines, totals, err := CalculateProfitability([]Line{{Quantity: 10000, UnitPrice: price, SupplierCost: cost}}, NoCommission, 0, delivery, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if lines[0].ProfitAfterCommission != 400000 || lines[0].MarginAfterCommission != 400000 {
		t.Fatalf("unexpected line profitability: %+v", lines[0])
	}
	if totals.ProfitAfterCommission != 300000 || totals.MarginAfterCommission != 300000 {
		t.Fatalf("unexpected quotation profitability: %+v", totals)
	}
}

func TestCalculateProfitabilityAllocatesFixedCommissionAcrossLines(t *testing.T) {
	price, _ := money.Parse("100")
	commission, _ := money.Parse("10")
	lines, totals, err := CalculateProfitability([]Line{
		{Quantity: 10000, UnitPrice: price},
		{Quantity: 30000, UnitPrice: price},
	}, FixedQuotation, commission, 0, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if totals.Commission != commission || lines[0].CommissionAmount != 25000 || lines[1].CommissionAmount != 75000 {
		t.Fatalf("unexpected commission allocation: totals=%+v lines=%+v", totals, lines)
	}
}
