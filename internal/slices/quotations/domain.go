package quotations

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
	sharedtax "github.com/psbernardo/syncline-collection-tracking/internal/shared/tax"
)

type Status string

const (
	Draft     Status = "DRAFT"
	Sent      Status = "SENT"
	Approved  Status = "APPROVED"
	Accepted  Status = Approved // Legacy Go name retained for callers.
	Expired   Status = "EXPIRED"
	Rejected  Status = "REJECTED"
	Cancelled Status = "CANCELLED"
)

type CommissionType string

const (
	NoCommission   CommissionType = "NONE"
	PerUnit        CommissionType = "PER_UNIT"
	Percentage     CommissionType = "PERCENTAGE"
	FixedQuotation CommissionType = "FIXED_QUOTATION"
)

type Line struct {
	ID                                            int64
	ProductID                                     int64
	ProductSKU, ProductName                       string
	Quantity                                      money.Amount
	UOM                                           string
	UnitPrice                                     money.Amount
	TaxCode                                       string
	TaxRate                                       int64 // percentage points in 1/10000, so 12% is 120000
	SupplierCost                                  money.Amount
	LineTotal, TaxAmount, VATInclusiveTotal       money.Amount
	SupplierProductCost                           money.Amount
	CommissionAmount                              money.Amount
	ProfitBeforeCommission, ProfitAfterCommission money.Amount
	MarginBeforeCommission, MarginAfterCommission money.Amount
}

type Totals struct {
	Subtotal, Tax, Total, VATInclusiveTotal       money.Amount
	Withholding                                   money.Amount
	ProfitBeforeCommission, ProfitAfterCommission money.Amount
	Commission                                    money.Amount
	SupplierProductCost                           money.Amount
	SupplierDeliveryCost, CustomerDeliveryCost    money.Amount
	OtherCost, TotalCost                          money.Amount
	MarginBeforeCommission, MarginAfterCommission money.Amount
}

var ErrInvalidQuotation = errors.New("invalid quotation")

var allowedTermsDays = map[int]bool{7: true, 15: true, 30: true, 45: true}

func ValidTermsDays(value int) bool { return allowedTermsDays[value] }

const (
	TaxNone  = "NONE"
	TaxVAT12 = "VAT12"
)

func TaxOptions() []struct{ Code, Label string } {
	return []struct{ Code, Label string }{{TaxNone, "No tax"}, {TaxVAT12, "VAT-inclusive, 12% VAT"}}
}

func TaxLabel(code string) string {
	switch code {
	case TaxVAT12:
		return "VAT-inclusive, 12% VAT"
	default:
		return "No tax"
	}
}

func applyTaxRule(lines []Line, code string) ([]Line, error) {
	if code == "" {
		code = TaxNone
	}
	rate := int64(0)
	switch code {
	case TaxNone:
	case TaxVAT12:
		rate = 120000
	default:
		return nil, fmt.Errorf("%w: unsupported tax code %q", ErrInvalidQuotation, code)
	}
	result := make([]Line, len(lines))
	copy(result, lines)
	for index := range result {
		result[index].TaxCode = code
		result[index].TaxRate = rate
	}
	return result, nil
}

func CalculateLine(line Line) (lineTotal, tax, inclusive money.Amount, err error) {
	if line.Quantity <= 0 || line.UnitPrice < 0 || line.TaxRate < 0 {
		return 0, 0, 0, fmt.Errorf("%w: quantity, price, and tax rate must be non-negative and quantity must be positive", ErrInvalidQuotation)
	}
	lineTotal, err = multiply(line.Quantity, line.UnitPrice)
	if err != nil {
		return 0, 0, 0, err
	}
	tax, err = rate(lineTotal, line.TaxRate)
	if err != nil {
		return 0, 0, 0, err
	}
	return lineTotal, tax, lineTotal + tax, nil
}

func CalculateTotals(lines []Line, commissionType CommissionType, commissionRate money.Amount, supplierDelivery, customerDelivery, otherCost money.Amount) (Totals, error) {
	if commissionRate < 0 || supplierDelivery < 0 || customerDelivery < 0 || otherCost < 0 {
		return Totals{}, fmt.Errorf("%w: amounts cannot be negative", ErrInvalidQuotation)
	}
	_, t, err := CalculateProfitability(lines, commissionType, commissionRate, supplierDelivery, customerDelivery, otherCost)
	return t, err
}

func CalculateProfitability(lines []Line, commissionType CommissionType, commissionRate money.Amount, supplierDelivery, customerDelivery, otherCost money.Amount) ([]Line, Totals, error) {
	if commissionRate < 0 || supplierDelivery < 0 || customerDelivery < 0 || otherCost < 0 {
		return nil, Totals{}, fmt.Errorf("%w: amounts cannot be negative", ErrInvalidQuotation)
	}
	result := make([]Line, len(lines))
	copy(result, lines)
	var t Totals
	var grossSubtotal money.Amount
	for index, line := range result {
		lineTotal, lineTax, inclusive, err := CalculateLine(line)
		if err != nil {
			return nil, Totals{}, err
		}
		result[index].LineTotal = lineTotal
		result[index].TaxAmount = lineTax
		result[index].VATInclusiveTotal = inclusive
		grossSubtotal += lineTotal
	}
	t.Subtotal = grossSubtotal
	taxCode := TaxNone
	if len(lines) > 0 && lines[0].TaxCode != "" {
		taxCode = lines[0].TaxCode
	}
	rule := sharedtax.RuleNone
	switch taxCode {
	case TaxVAT12:
		rule = sharedtax.RuleVATInclusive12
	case TaxNone:
	default:
		return nil, Totals{}, fmt.Errorf("%w: unsupported tax code %q", ErrInvalidQuotation, taxCode)
	}
	breakdown, err := sharedtax.CalculateRule(t.Subtotal, rule)
	if err != nil {
		return nil, Totals{}, err
	}
	t.Subtotal = breakdown.TaxBase
	t.Tax = breakdown.VATAmount
	t.Withholding = breakdown.WithholdingAmount
	t.Total = breakdown.GrossAmount
	t.VATInclusiveTotal = breakdown.GrossAmount
	t.SupplierDeliveryCost = supplierDelivery
	t.CustomerDeliveryCost = customerDelivery
	t.OtherCost = otherCost
	var commissionBase money.Amount
	switch commissionType {
	case NoCommission:
	case PerUnit:
		for index, line := range result {
			value, err := multiply(line.Quantity, commissionRate)
			if err != nil {
				return nil, Totals{}, err
			}
			t.Commission += value
			result[index].CommissionAmount = value
		}
	case Percentage:
		var err error
		t.Commission, err = rate(t.Total, commissionRate.Int64())
		if err != nil {
			return nil, Totals{}, err
		}
		commissionBase = t.Subtotal
	case FixedQuotation:
		t.Commission = commissionRate
		commissionBase = t.Subtotal
	default:
		return nil, Totals{}, fmt.Errorf("%w: unsupported commission type %q", ErrInvalidQuotation, commissionType)
	}
	var supplierTotal money.Amount
	for index, line := range result {
		value, err := multiply(line.Quantity, line.SupplierCost)
		if err != nil {
			return nil, Totals{}, err
		}
		supplierTotal += value
		result[index].SupplierProductCost = value
	}
	if commissionBase > 0 {
		for index, line := range result {
			lineBase := line.LineTotal
			if grossSubtotal > 0 {
				lineBase = money.Amount(new(big.Int).Quo(new(big.Int).Mul(big.NewInt(line.LineTotal.Int64()), big.NewInt(t.Subtotal.Int64())), big.NewInt(grossSubtotal.Int64())).Int64())
			}
			value, err := multiply(t.Commission, money.Amount(new(big.Int).Quo(new(big.Int).Mul(big.NewInt(lineBase.Int64()), big.NewInt(10000)), big.NewInt(commissionBase.Int64())).Int64()))
			if err != nil {
				return nil, Totals{}, err
			}
			result[index].CommissionAmount = value
		}
	}
	t.SupplierProductCost = supplierTotal
	t.TotalCost = supplierTotal + supplierDelivery + customerDelivery + otherCost + t.Commission
	t.ProfitBeforeCommission = t.Subtotal - supplierTotal - supplierDelivery - customerDelivery - otherCost
	t.ProfitAfterCommission = t.ProfitBeforeCommission - t.Commission
	for index, line := range result {
		lineNetSales := line.LineTotal
		if grossSubtotal > 0 {
			lineNetSales = money.Amount(new(big.Int).Quo(new(big.Int).Mul(big.NewInt(line.LineTotal.Int64()), big.NewInt(t.Subtotal.Int64())), big.NewInt(grossSubtotal.Int64())).Int64())
		}
		result[index].ProfitBeforeCommission = lineNetSales - line.SupplierProductCost
		result[index].ProfitAfterCommission = result[index].ProfitBeforeCommission - result[index].CommissionAmount
		result[index].MarginBeforeCommission = margin(result[index].ProfitBeforeCommission, lineNetSales)
		result[index].MarginAfterCommission = margin(result[index].ProfitAfterCommission, lineNetSales)
	}
	t.MarginBeforeCommission = margin(t.ProfitBeforeCommission, t.Subtotal)
	t.MarginAfterCommission = margin(t.ProfitAfterCommission, t.Subtotal)
	return result, t, nil
}

func margin(profit, sales money.Amount) money.Amount {
	if sales == 0 {
		return 0
	}
	v := new(big.Int).Mul(big.NewInt(profit.Int64()), big.NewInt(1000000))
	v.Quo(v, big.NewInt(sales.Int64()))
	return money.Amount(v.Int64())
}

func multiply(a, b money.Amount) (money.Amount, error) {
	v := new(big.Int).Mul(big.NewInt(a.Int64()), big.NewInt(b.Int64()))
	v.Quo(v, big.NewInt(10000))
	if !v.IsInt64() {
		return 0, fmt.Errorf("%w: amount is too large", ErrInvalidQuotation)
	}
	return money.Amount(v.Int64()), nil
}
func rate(amount money.Amount, scaledRate int64) (money.Amount, error) {
	if scaledRate < 0 {
		return 0, fmt.Errorf("%w: tax rate cannot be negative", ErrInvalidQuotation)
	}
	v := new(big.Int).Mul(big.NewInt(amount.Int64()), big.NewInt(scaledRate))
	v.Quo(v, big.NewInt(1000000))
	if !v.IsInt64() {
		return 0, fmt.Errorf("%w: amount is too large", ErrInvalidQuotation)
	}
	return money.Amount(v.Int64()), nil
}

func NormalizeStatus(value string) (Status, error) {
	s := Status(strings.ToUpper(strings.TrimSpace(value)))
	switch s {
	case Draft, Sent, Approved, Expired, Rejected, Cancelled:
		return s, nil
	}
	return "", fmt.Errorf("%w: invalid status", ErrInvalidQuotation)
}
