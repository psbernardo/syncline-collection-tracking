package invoices

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
	sharedpdf "github.com/psbernardo/syncline-collection-tracking/internal/shared/pdf"
	"github.com/signintech/gopdf"
)

type InvoicePDFDocument struct {
	Seller                                                                                       sharedpdf.SellerProfile
	Number, SalesOrderNumber, CustomerPO, SalesPerson, Customer, BillingAddress, DeliveryAddress string
	ContactPerson, ContactNumber, Email                                                          string
	TermsDays                                                                                    int
	InvoiceDate, DueDate                                                                         time.Time
	Lines                                                                                        []InvoicePDFLine
	Subtotal, Tax, Total                                                                         money.Amount
	VatableSales, ZeroRatedSales, VATExemptSales, Discount, WithholdingTax, TotalAmountDue       *money.Amount
}

type InvoicePDFLine struct {
	SKU, Name, UOM              string
	Quantity, UnitPrice, Amount money.Amount
	TaxRate                     int64
}

type InvoicePDFRenderer struct {
	Seller sharedpdf.SellerProfile
	Font   string
}

const (
	invoiceFirstTableY        = 297.0
	invoiceContinuationTableY = 35.0
	invoiceDescriptionWidth   = 205.0
	invoiceFooterGap          = 3 * 28.35
	invoiceTotalsHeight       = 96.0
	invoiceFooterSpacing      = 24.0
	invoiceFooterHeight       = 42.0
	invoiceMetadataLabelWidth = 98.0
	invoiceMetadataValueX     = 445.0
	invoiceMetadataValueWidth = 126.0
	invoiceMetadataMinHeight  = 22.0
	invoiceMetadataLineHeight = 11.0
	invoiceMetadataTableGap   = 16.0
)

func NewInvoicePDFRenderer() *InvoicePDFRenderer {
	return &InvoicePDFRenderer{Seller: sharedpdf.DefaultSeller(), Font: sharedpdf.FindFont()}
}

func (r *InvoicePDFRenderer) Render(w io.Writer, document InvoicePDFDocument) error {
	if document.Number == "" || document.Customer == "" || len(document.Lines) == 0 {
		return fmt.Errorf("invoice PDF requires a number, customer, and at least one line")
	}
	pdf := &gopdf.GoPdf{}
	if err := sharedpdf.Configure(pdf, r.Font); err != nil {
		return fmt.Errorf("load invoice PDF font: %w", err)
	}
	page := invoicePage{pdf: pdf, seller: r.Seller, document: document}
	y := page.header()
	for index, line := range document.Lines {
		wrapped := sharedpdf.Wrap(pdf, invoiceDescription(line), invoiceDescriptionWidth)
		height := maxFloat(25, float64(len(wrapped))*9+3) * 1.20
		if y+height > sharedpdf.PageBottom {
			pdf.AddPage()
			page = invoicePage{pdf: pdf, seller: r.Seller, document: document}
			y = page.tableHeader(invoiceContinuationTableY)
		}
		page.line(y, height, index+1, line, wrapped)
		y += height
	}
	if !invoiceFinalContentFits(y) {
		pdf.AddPage()
		page = invoicePage{pdf: pdf, seller: r.Seller, document: document}
		y = page.tableHeader(invoiceContinuationTableY) + 20
	}
	totalsY := y + invoiceFooterGap
	page.totals(totalsY)
	page.footer(totalsY + invoiceTotalsHeight + invoiceFooterSpacing)
	return sharedpdf.Write(w, pdf, "invoice")
}

func invoiceDescription(line InvoicePDFLine) string {
	return strings.TrimSpace(line.Name)
}

func invoiceFinalContentFits(lastRowBottom float64) bool {
	return lastRowBottom+invoiceFooterGap+invoiceTotalsHeight+invoiceFooterSpacing+invoiceFooterHeight <= sharedpdf.PageBottom
}

type invoicePage struct {
	pdf      *gopdf.GoPdf
	seller   sharedpdf.SellerProfile
	document InvoicePDFDocument
}

func (p invoicePage) header() float64 {
	p.brand()
	sharedpdf.Text(p.pdf, 342, 100, 25, "INVOICE", 225, gopdf.Right, "B", sharedpdf.Purple)
	sharedpdf.Text(p.pdf, 342, 125, 8.5, p.document.Number, 225, gopdf.Right, "B", sharedpdf.Purple)
	metadataBottom := p.metadata(342, 188, []sharedpdf.Metadata{
		{Label: "INVOICE DATE", Value: p.document.InvoiceDate.Format("January 02 2006")},
		{Label: "DUE DATE", Value: p.document.DueDate.Format("January 02 2006")},
		{Label: "SALES ORDER NO.", Value: p.document.SalesOrderNumber},
		{Label: "PO NUMBER", Value: p.document.CustomerPO},
		{Label: "PAYMENT TERMS", Value: fmt.Sprintf("Net %d days", p.document.TermsDays)},
	})
	p.parties(188)
	return p.tableHeader(maxFloat(invoiceFirstTableY, maxFloat(metadataBottom, 259)+invoiceMetadataTableGap))
}

func (p invoicePage) continuationHeader() float64 {
	p.brand()
	sharedpdf.Text(p.pdf, 342, 100, 25, "INVOICE", 225, gopdf.Right, "B", sharedpdf.Purple)
	return p.tableHeader(invoiceContinuationTableY)
}

func (p invoicePage) brand() {
	if p.seller.Logo != "" {
		_ = p.pdf.Image(p.seller.Logo, 32, 20, &gopdf.Rect{W: 102, H: 66})
	} else {
		sharedpdf.Text(p.pdf, 30, 62, 18, "LOGO", 115, gopdf.Center, "B", sharedpdf.Purple)
	}
	sharedpdf.Text(p.pdf, 30, 100, 13.5, strings.ToUpper(p.seller.Name), 300, gopdf.Left, "B", sharedpdf.Purple)
	sharedpdf.Text(p.pdf, 30, 115, 8.5, p.seller.Address, 300, gopdf.Left, "", "black")
	sharedpdf.Text(p.pdf, 30, 129, 8.5, "Phone: "+p.seller.Phone, 300, gopdf.Left, "", "black")
	sharedpdf.Text(p.pdf, 30, 143, 8.5, p.seller.Email, 300, gopdf.Left, "", "black")
}

func (p invoicePage) metadata(x, y float64, fields []sharedpdf.Metadata) float64 {
	for _, field := range fields {
		label, value := sharedpdf.Wrap(p.pdf, field.Label, invoiceMetadataLabelWidth), sharedpdf.Wrap(p.pdf, field.Value, invoiceMetadataValueWidth)
		count := len(label)
		if len(value) > count {
			count = len(value)
		}
		height := maxFloat(invoiceMetadataMinHeight, float64(count)*invoiceMetadataLineHeight+5)
		sharedpdf.MultiText(p.pdf, x, y+2, 9, strings.Join(label, "\n"), invoiceMetadataLabelWidth, height, gopdf.Right, "B", sharedpdf.Purple)
		sharedpdf.MultiText(p.pdf, invoiceMetadataValueX, y+2, 9, strings.Join(value, "\n"), invoiceMetadataValueWidth, height, gopdf.Right, "", "black")
		y += height
	}
	return y
}

func (p invoicePage) parties(y float64) {
	sharedpdf.Text(p.pdf, 30, y, 9, "BILL TO", 300, gopdf.Left, "B", sharedpdf.Purple)
	sharedpdf.Text(p.pdf, 30, y+17, 13.5, strings.ToUpper(p.document.Customer), 300, gopdf.Left, "B", sharedpdf.Purple)
	sharedpdf.MultiText(p.pdf, 30, y+32, 9, p.document.BillingAddress, 300, 28, gopdf.Left, "", "black")
	sharedpdf.Text(p.pdf, 30, y+54, 9, "SHIP TO", 300, gopdf.Left, "B", sharedpdf.Purple)
	sharedpdf.MultiText(p.pdf, 30, y+71, 9, p.document.DeliveryAddress, 300, 35, gopdf.Left, "", "black")
}

func (p invoicePage) tableHeader(y float64) float64 {
	columns := []struct {
		title string
		width float64
		align int
	}{{"#", 25, gopdf.Center}, {"QTY", 50, gopdf.Center}, {"UNIT", 45, gopdf.Center}, {"DESCRIPTION", 205, gopdf.Center}, {"RATE", 85, gopdf.Right}, {"TAX %", 50, gopdf.Right}, {"AMOUNT", 87, gopdf.Right}}
	x := sharedpdf.Margin
	p.pdf.SetFillColor(sharedpdf.RGB(sharedpdf.LightPurple))
	p.pdf.SetStrokeColor(sharedpdf.RGB(sharedpdf.LightPurple))
	p.pdf.RectFromUpperLeftWithStyle(x, y, sharedpdf.Width-2*sharedpdf.Margin, 20, "DF")
	for _, column := range columns {
		sharedpdf.Text(p.pdf, x+3, y+3, 9.5, column.title, column.width-6, column.align, "B", sharedpdf.Purple)
		x += column.width
	}
	return y + 20
}

func (p invoicePage) line(y, height float64, number int, line InvoicePDFLine, description []string) {
	values := []struct {
		text  string
		width float64
		align int
	}{{fmt.Sprintf("%d.", number), 25, gopdf.Center}, {line.Quantity.Format(), 50, gopdf.Center}, {line.UOM, 45, gopdf.Center}, {strings.Join(description, "\n"), 205, gopdf.Left}, {sharedpdf.FormatUnitPrice(line.UnitPrice), 85, gopdf.Right}, {fmt.Sprintf("%d%%", line.TaxRate/10000), 50, gopdf.Right}, {sharedpdf.FormatUnitPrice(line.Amount), 87, gopdf.Right}}
	x := sharedpdf.Margin
	for index, value := range values {
		if index == 3 {
			sharedpdf.MultiText(p.pdf, x+3, y+4, 9, value.text, value.width-6, height-4, value.align, "", "black")
		} else {
			sharedpdf.Text(p.pdf, x+3, y+4, 9, value.text, value.width-6, value.align, "", "black")
		}
		x += value.width
	}
	sharedpdf.Line(p.pdf, sharedpdf.Margin, y+height, sharedpdf.Width-sharedpdf.Margin, y+height, "gray")
}

func (p invoicePage) totals(y float64) {
	vatableSales := p.document.VatableSales
	if vatableSales == nil && p.document.Tax != 0 {
		vatableSales = amountPointer(p.document.Subtotal)
	}
	totalAmountDue := p.document.TotalAmountDue
	if totalAmountDue == nil {
		totalAmountDue = amountPointer(p.document.Total)
	}
	left := []struct {
		label string
		value *money.Amount
	}{
		{"Vatable Sales:", vatableSales},
		{"Vat:", nonZeroAmountPointer(p.document.Tax)},
		{"Zero-Rated Sales:", p.document.ZeroRatedSales},
		{"Vat-Exempt Sales:", p.document.VATExemptSales},
	}
	right := []struct {
		label string
		value *money.Amount
	}{
		{"Total Sales:", amountPointer(p.document.Total)},
		{"Less: VAT:", nonZeroAmountPointer(p.document.Tax)},
		{"Less: Discount:", p.document.Discount},
		{"Add Vat:", nonZeroAmountPointer(p.document.Tax)},
		{"Less: Withholding Tax:", p.document.WithholdingTax},
		{"Total amount due:", totalAmountDue},
	}
	for index, row := range left {
		sharedpdf.Text(p.pdf, 30, y, 9, row.label, 125, gopdf.Left, "B", sharedpdf.Purple)
		sharedpdf.Text(p.pdf, 155, y, 9, formatOptionalAmount(row.value), 95, gopdf.Right, "B", sharedpdf.Purple)
		if index < len(right) {
			other := right[index]
			sharedpdf.Text(p.pdf, 300, y, 9, other.label, 175, gopdf.Left, "B", sharedpdf.Purple)
			sharedpdf.Text(p.pdf, 480, y, 9, formatOptionalAmount(other.value), 90, gopdf.Right, "B", sharedpdf.Purple)
		}
		y += 16
	}
	for _, row := range right[len(left):] {
		sharedpdf.Text(p.pdf, 300, y, 9, row.label, 175, gopdf.Left, "B", sharedpdf.Purple)
		sharedpdf.Text(p.pdf, 480, y, 9, formatOptionalAmount(row.value), 90, gopdf.Right, "B", sharedpdf.Purple)
		y += 16
	}
}

func amountPointer(value money.Amount) *money.Amount {
	return &value
}

func nonZeroAmountPointer(value money.Amount) *money.Amount {
	if value == 0 {
		return nil
	}
	return amountPointer(value)
}

func formatOptionalAmount(value *money.Amount) string {
	if value == nil {
		return ""
	}
	return sharedpdf.FormatAmount(*value)
}

func (p invoicePage) footer(y float64) {
	sharedpdf.Text(p.pdf, sharedpdf.Margin, y, 8, "If you have any questions about this invoice, please contact us", sharedpdf.Width-2*sharedpdf.Margin, gopdf.Center, "I", sharedpdf.Purple)
	sharedpdf.Text(p.pdf, sharedpdf.Margin, y+14, 8, fmt.Sprintf("(%s, %s)", p.seller.Email, p.seller.Phone), sharedpdf.Width-2*sharedpdf.Margin, gopdf.Center, "I", sharedpdf.Purple)
	sharedpdf.Text(p.pdf, sharedpdf.Margin, y+28, 9, "Thank You For Your Business!", sharedpdf.Width-2*sharedpdf.Margin, gopdf.Center, "I", sharedpdf.Purple)
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
