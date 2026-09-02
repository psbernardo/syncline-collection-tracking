package invoices

import (
	"errors"
	"testing"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
	"github.com/psbernardo/syncline-collection-tracking/internal/shared/tax"
	"github.com/psbernardo/syncline-collection-tracking/internal/slices/quotations"
	"github.com/psbernardo/syncline-collection-tracking/internal/slices/salesorders"
)

func TestReceivableTaxRule(t *testing.T) {
	tests := []struct {
		name  string
		codes []string
		want  tax.RuleCode
		err   error
	}{
		{name: "no tax", codes: []string{quotations.TaxNone, quotations.TaxNone}, want: tax.RuleNone},
		{name: "twelve percent VAT", codes: []string{quotations.TaxVAT12}, want: tax.RuleVATInclusiveEWT1},
		{name: "mixed treatment", codes: []string{quotations.TaxNone, quotations.TaxVAT12}, err: ErrMixedTaxTreatment},
		{name: "unsupported treatment", codes: []string{"EXEMPT"}, err: ErrUnsupportedTaxRule},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ReceivableTaxRule(test.codes)
			if !errors.Is(err, test.err) || got != test.want {
				t.Fatalf("got rule=%q, err=%v; want rule=%q, err=%v", got, err, test.want, test.err)
			}
		})
	}
}

func TestBuildPreviewMapsSalesOrderTaxRule(t *testing.T) {
	noTax := salesorders.SalesOrder{Number: "SO-NONE", CustomerPONumber: "PO-NONE", Quotation: quotations.Quotation{
		TermsDays: 30,
		Lines:     []quotations.Line{{TaxCode: quotations.TaxNone}},
		Totals:    quotations.Totals{Total: money.Amount(28000000)},
	}}
	vat := noTax
	vat.Number = "SO-VAT"
	vat.Quotation.Lines = []quotations.Line{{TaxCode: quotations.TaxVAT12}}

	noTaxPreview := buildPreview(noTax, "INV-NONE", time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC))
	if noTaxPreview.TaxRule != "No tax rule" || noTaxPreview.ReceivableEWT != "₱0.00" || noTaxPreview.ReceivableNet != noTaxPreview.ReceivableGross {
		t.Fatalf("unexpected no-tax preview: %+v", noTaxPreview)
	}
	vatPreview := buildPreview(vat, "INV-VAT", time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC))
	if vatPreview.TaxRule != "VAT-inclusive, 1% EWT" || vatPreview.ReceivableEWT == "₱0.00" {
		t.Fatalf("unexpected VAT preview: %+v", vatPreview)
	}
}
