package tax

import (
	"testing"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
)

func TestCalculateVATInclusive(t *testing.T) {
	tests := []struct {
		name     string
		gross    string
		wantBase string
		wantVAT  string
		wantEWT  string
		wantNet  string
	}{
		{
			name:     "2800 gross with 25 EWT",
			gross:    "2800",
			wantBase: "2500",
			wantVAT:  "300",
			wantEWT:  "25",
			wantNet:  "2775",
		},
		{
			name:     "13400 gross with 119.64 EWT",
			gross:    "13400",
			wantBase: "11964.29",
			wantVAT:  "1435.71",
			wantEWT:  "119.64",
			wantNet:  "13280.36",
		},
		{
			name:     "canonical VAT example without withholding",
			gross:    "1459",
			wantBase: "1302.68",
			wantVAT:  "156.32",
			wantEWT:  "0",
			wantNet:  "1459",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gross, err := money.Parse(tt.gross)
			if err != nil {
				t.Fatalf("money.Parse() error = %v", err)
			}
			withholdingRate := EWT1
			if tt.wantEWT == "0" {
				withholdingRate = Rate{Denominator: 1}
			}
			got, err := CalculateVATInclusive(gross, VAT12, withholdingRate)
			if err != nil {
				t.Fatalf("CalculateVATInclusive() error = %v", err)
			}
			assertAmount(t, "tax base", got.TaxBase, tt.wantBase)
			assertAmount(t, "VAT", got.VATAmount, tt.wantVAT)
			assertAmount(t, "EWT", got.WithholdingAmount, tt.wantEWT)
			assertAmount(t, "net amount", got.NetAmount, tt.wantNet)
			if got.TaxBase+got.VATAmount != got.GrossAmount {
				t.Fatalf("tax base + VAT = %s, want gross %s", (got.TaxBase + got.VATAmount).Format(), got.GrossAmount.Format())
			}
		})
	}
}

func TestCalculateVATInclusiveRejectsInvalidRates(t *testing.T) {
	gross, err := money.Parse("2800")
	if err != nil {
		t.Fatal(err)
	}
	for _, rate := range []Rate{
		{Numerator: -1, Denominator: 100},
		{Numerator: 12, Denominator: 0},
		{Numerator: 100, Denominator: 100},
	} {
		if _, err := CalculateVATInclusive(gross, rate, EWT1); err == nil {
			t.Fatalf("CalculateVATInclusive(%+v) expected an error", rate)
		}
	}
}

func TestCalculateVATInclusiveZero(t *testing.T) {
	got, err := CalculateVATInclusive(0, VAT12, EWT1)
	if err != nil {
		t.Fatal(err)
	}
	if got.TaxBase != 0 || got.VATAmount != 0 || got.WithholdingAmount != 0 || got.NetAmount != 0 {
		t.Fatalf("zero calculation = %+v, want all zero amounts", got)
	}
}

func assertAmount(t *testing.T, label string, got money.Amount, want string) {
	t.Helper()
	wantAmount, err := money.Parse(want)
	if err != nil {
		t.Fatalf("money.Parse(%q) error = %v", want, err)
	}
	if got.Format() != wantAmount.Format() {
		t.Fatalf("%s = %s, want %s", label, got.Format(), wantAmount.Format())
	}
}
