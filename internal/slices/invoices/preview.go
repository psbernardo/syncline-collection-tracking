package invoices

import (
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/businessdate"
	"github.com/psbernardo/syncline-collection-tracking/internal/shared/tax"
	"github.com/psbernardo/syncline-collection-tracking/internal/slices/salesorders"
)

type InvoiceReceivablePreview struct {
	CustomerName, SalesOrderNumber, InvoiceNumber, CustomerPONumber string
	InvoiceDate, DeliveryDate, DueDate                              string
	InvoiceSubtotal, InvoiceTax, InvoiceTotal                       string
	ReceivableGross, ReceivableVAT, ReceivableEWT, ReceivableNet    string
	PaymentTerms                                                    int
	TaxRule                                                         string
}

func buildPreview(order salesorders.SalesOrder, number string, now time.Time) InvoiceReceivablePreview {
	delivery, _ := businessdate.Parse(businessdate.FormatUTC(now))
	due, _ := businessdate.DueDate(delivery, order.Quotation.TermsDays)
	codes := make([]string, 0, len(order.Quotation.Lines))
	for _, line := range order.Quotation.Lines {
		codes = append(codes, line.TaxCode)
	}
	receivableRule, err := ReceivableTaxRule(codes)
	if err != nil {
		receivableRule = tax.RuleNone
	}
	breakdown, _ := tax.CalculateRule(order.Quotation.Totals.Total, receivableRule)
	taxLabel := "No tax rule"
	if receivableRule == tax.RuleVATInclusiveEWT1 {
		taxLabel = "VAT-inclusive, 1% EWT"
	}
	return InvoiceReceivablePreview{
		CustomerName: order.Quotation.CompanyName, SalesOrderNumber: order.Number, InvoiceNumber: number,
		CustomerPONumber: order.CustomerPONumber, InvoiceDate: businessdate.FormatUTC(now), DeliveryDate: businessdate.FormatUTC(delivery), DueDate: businessdate.FormatUTC(due),
		InvoiceSubtotal: order.Quotation.Totals.Subtotal.FormatPHP(), InvoiceTax: order.Quotation.Totals.Tax.FormatPHP(), InvoiceTotal: order.Quotation.Totals.Total.FormatPHP(),
		ReceivableGross: breakdown.GrossAmount.FormatPHP(), ReceivableVAT: breakdown.VATAmount.FormatPHP(), ReceivableEWT: breakdown.WithholdingAmount.FormatPHP(), ReceivableNet: breakdown.NetAmount.FormatPHP(),
		PaymentTerms: order.Quotation.TermsDays, TaxRule: taxLabel,
	}
}
