package tax

import (
	"fmt"
	"math/big"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
)

// Rate represents a non-negative percentage as an exact fraction.
// For example, 12% is Rate{Numerator: 12, Denominator: 100}.
type Rate struct {
	Numerator   int64
	Denominator int64
}

type RuleCode string

const (
	RuleNone             RuleCode = ""
	RuleVATInclusiveEWT1 RuleCode = "vat_inclusive_ewt_1"
)

type RuleOption struct {
	Code  RuleCode
	Label string
}

var (
	VAT12 = Rate{Numerator: 12, Denominator: 100}
	EWT1  = Rate{Numerator: 1, Denominator: 100}
)

func RuleOptions() []RuleOption {
	return []RuleOption{{Code: RuleVATInclusiveEWT1, Label: "VAT-inclusive, 1% EWT"}}
}

func CalculateRule(gross money.Amount, code RuleCode) (Breakdown, error) {
	if code == RuleNone {
		return Breakdown{GrossAmount: gross, TaxBase: gross, NetAmount: gross, VATRate: Rate{Denominator: 1}, WithholdingRate: Rate{Denominator: 1}}, nil
	}
	if code != RuleVATInclusiveEWT1 {
		return Breakdown{}, fmt.Errorf("unknown tax rule %q", code)
	}
	return CalculateVATInclusive(gross, VAT12, EWT1)
}

// Breakdown contains the VAT-inclusive amount and its tax components.
type Breakdown struct {
	GrossAmount       money.Amount
	TaxBase           money.Amount
	VATAmount         money.Amount
	WithholdingAmount money.Amount
	NetAmount         money.Amount
	VATRate           Rate
	WithholdingRate   Rate
}

// CalculateVATInclusive calculates the VAT-exclusive base, VAT, EWT, and
// amount payable from a VAT-inclusive gross amount.
//
// Withholding is calculated from the rounded tax base, not from gross or VAT.
// VAT is the remainder after the rounded tax base so the components reconcile
// exactly to gross at the money scale.
func CalculateVATInclusive(gross money.Amount, vatRate, withholdingRate Rate) (Breakdown, error) {
	if gross < 0 {
		return Breakdown{}, fmt.Errorf("gross amount must be non-negative")
	}
	if err := validateRate("VAT", vatRate, true); err != nil {
		return Breakdown{}, err
	}
	if err := validateRate("withholding", withholdingRate, true); err != nil {
		return Breakdown{}, err
	}

	taxBase, err := calculateBase(gross, vatRate)
	if err != nil {
		return Breakdown{}, err
	}
	vat := gross - taxBase
	withholding, err := applyRate(taxBase, withholdingRate)
	if err != nil {
		return Breakdown{}, err
	}
	if withholding > gross {
		return Breakdown{}, fmt.Errorf("withholding amount exceeds gross amount")
	}

	return Breakdown{
		GrossAmount:       gross,
		TaxBase:           taxBase,
		VATAmount:         vat,
		WithholdingAmount: withholding,
		NetAmount:         gross - withholding,
		VATRate:           vatRate,
		WithholdingRate:   withholdingRate,
	}, nil
}

func validateRate(name string, rate Rate, allowZero bool) error {
	if rate.Denominator <= 0 || rate.Numerator < 0 || (!allowZero && rate.Numerator == 0) {
		return fmt.Errorf("%s rate must be a non-negative fraction", name)
	}
	if rate.Numerator >= rate.Denominator {
		return fmt.Errorf("%s rate must be less than 100%%", name)
	}
	return nil
}

func calculateBase(gross money.Amount, rate Rate) (money.Amount, error) {
	// gross is already scaled. Dividing by (1 + rate) therefore requires no
	// unit conversion: gross / ((denominator + numerator) / denominator).
	numerator := new(big.Int).Mul(big.NewInt(gross.Int64()), big.NewInt(rate.Denominator))
	denominator := big.NewInt(rate.Denominator + rate.Numerator)
	scaled := roundPositive(numerator, denominator)
	return amountFromBigInt(scaled)
}

func applyRate(amount money.Amount, rate Rate) (money.Amount, error) {
	numerator := new(big.Int).Mul(big.NewInt(amount.Int64()), big.NewInt(rate.Numerator))
	denominator := big.NewInt(rate.Denominator)
	scaled := roundPositive(numerator, denominator)
	return amountFromBigInt(scaled)
}

func roundPositive(numerator, denominator *big.Int) *big.Int {
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(numerator, denominator, remainder)
	if remainder.Lsh(remainder, 1).Cmp(denominator) >= 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	return quotient
}

func amountFromBigInt(value *big.Int) (money.Amount, error) {
	if !value.IsInt64() || value.Sign() < 0 {
		return 0, fmt.Errorf("calculated amount is too large")
	}
	return money.Amount(value.Int64()), nil
}
