package salesorders

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
	sharedpdf "github.com/psbernardo/syncline-collection-tracking/internal/shared/pdf"
	"github.com/signintech/gopdf"
)

type SalesOrderPDFDocument struct {
	Seller                                                                             sharedpdf.SellerProfile
	Number, Source, CustomerPO, SalesPerson, Customer, BillingAddress, DeliveryAddress string
	TermsDays                                                                          int
	OrderDate                                                                          time.Time
	Lines                                                                              []SalesOrderPDFLine
	Subtotal, Tax, Total                                                               money.Amount
}

type SalesOrderPDFLine struct {
	SKU, Name, UOM, TaxCode     string
	Quantity, UnitPrice, Amount money.Amount
	TaxRate                     int64
}

type SalesOrderPDFRenderer struct {
	Seller sharedpdf.SellerProfile
	Font   string
}

func NewSalesOrderPDFRenderer() *SalesOrderPDFRenderer {
	return &SalesOrderPDFRenderer{Seller: sharedpdf.DefaultSeller(), Font: sharedpdf.FindFont()}
}

func (r *SalesOrderPDFRenderer) Render(w io.Writer, document SalesOrderPDFDocument) error {
	if document.Number == "" || document.Customer == "" || len(document.Lines) == 0 {
		return fmt.Errorf("sales order PDF requires a number, customer, and at least one line")
	}
	pdf := &gopdf.GoPdf{}
	if err := sharedpdf.Configure(pdf, r.Font); err != nil {
		return fmt.Errorf("load sales order PDF font: %w", err)
	}
	page := salesOrderPage{pdf: pdf, seller: r.Seller, document: document}
	y := page.header()
	for index, line := range document.Lines {
		description := strings.TrimSpace(line.SKU + " - " + line.Name)
		wrapped := sharedpdf.Wrap(pdf, description, 205)
		height := maxFloat(25, float64(len(wrapped))*9+8)
		if y+height > 690 {
			pdf.AddPage()
			page = salesOrderPage{pdf: pdf, seller: r.Seller, document: document}
			y = page.continuationHeader()
		}
		page.line(y, height, index+1, line, wrapped)
		y += height
	}
	if y+125 > sharedpdf.PageBottom {
		pdf.AddPage()
		page = salesOrderPage{pdf: pdf, seller: r.Seller, document: document}
		y = page.continuationHeader()
	}
	page.totals(y + 18)
	page.footer(y + 85)
	return sharedpdf.Write(w, pdf, "sales order")
}

type salesOrderPage struct {
	pdf      *gopdf.GoPdf
	seller   sharedpdf.SellerProfile
	document SalesOrderPDFDocument
}

func (p salesOrderPage) header() float64 {
	p.brand()
	sharedpdf.Text(p.pdf, 342, 100, 25, "SALES ORDER", 225, gopdf.Right, "B", sharedpdf.Purple)
	sharedpdf.Text(p.pdf, 342, 125, 8.5, p.document.Number, 225, gopdf.Right, "B", sharedpdf.Purple)
	metadataBottom := p.metadata(342, 188, []sharedpdf.Metadata{
		{Label: "ORDER DATE", Value: p.document.OrderDate.Format("January 02 2006")},
		{Label: "QUOTATION REF #", Value: p.document.Source},
		{Label: "SALES PERSON", Value: p.document.SalesPerson},
		{Label: "PO NUMBER", Value: p.document.CustomerPO},
		{Label: "PAYMENT TERMS", Value: fmt.Sprintf("Net %d days", p.document.TermsDays)},
	})
	p.parties(188)
	return p.tableHeader(maxFloat(330, metadataBottom+16))
}

func (p salesOrderPage) continuationHeader() float64 {
	p.brand()
	sharedpdf.Text(p.pdf, 342, 100, 25, "SALES ORDER", 225, gopdf.Right, "B", sharedpdf.Purple)
	return p.tableHeader(105)
}

func (p salesOrderPage) brand() {
	if p.seller.Logo != "" {
		_ = p.pdf.Image(p.seller.Logo, 32, 20, &gopdf.Rect{W: 102, H: 66})
	}
	if p.seller.Logo == "" {
		sharedpdf.Text(p.pdf, 30, 62, 18, "LOGO", 115, gopdf.Center, "B", sharedpdf.Purple)
	}
	sharedpdf.Text(p.pdf, 30, 100, 13.5, strings.ToUpper(p.seller.Name), 300, gopdf.Left, "B", sharedpdf.Purple)
	sharedpdf.Text(p.pdf, 30, 115, 8.5, p.seller.Address, 300, gopdf.Left, "", "black")
	sharedpdf.Text(p.pdf, 30, 129, 8.5, "Phone: "+p.seller.Phone, 300, gopdf.Left, "", "black")
	sharedpdf.Text(p.pdf, 30, 143, 8.5, p.seller.Email, 300, gopdf.Left, "", "black")
}

func (p salesOrderPage) metadata(x, y float64, fields []sharedpdf.Metadata) float64 {
	for _, field := range fields {
		label := sharedpdf.Wrap(p.pdf, field.Label, 98)
		value := sharedpdf.Wrap(p.pdf, field.Value, 126)
		count := len(label)
		if len(value) > count {
			count = len(value)
		}
		height := maxFloat(22, float64(count)*11+5)
		sharedpdf.MultiText(p.pdf, x, y+2, 9, strings.Join(label, "\n"), 98, height, gopdf.Right, "B", sharedpdf.Purple)
		sharedpdf.MultiText(p.pdf, 445, y+2, 9, strings.Join(value, "\n"), 126, height, gopdf.Right, "", "black")
		y += height
	}
	return y
}

func (p salesOrderPage) parties(y float64) {
	sharedpdf.Text(p.pdf, 30, y, 9, "BILL TO", 300, gopdf.Left, "B", sharedpdf.Purple)
	sharedpdf.Text(p.pdf, 30, y+17, 13.5, strings.ToUpper(p.document.Customer), 300, gopdf.Left, "B", sharedpdf.Purple)
	sharedpdf.MultiText(p.pdf, 30, y+32, 9, p.document.BillingAddress, 300, 28, gopdf.Left, "", "black")
	sharedpdf.Text(p.pdf, 30, y+54, 9, "SHIP TO", 300, gopdf.Left, "B", sharedpdf.Purple)
	sharedpdf.MultiText(p.pdf, 30, y+71, 9, p.document.DeliveryAddress, 300, 35, gopdf.Left, "", "black")
}

func (p salesOrderPage) tableHeader(y float64) float64 {
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

func (p salesOrderPage) line(y, height float64, number int, line SalesOrderPDFLine, description []string) {
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

func (p salesOrderPage) totals(y float64) {
	for _, row := range []struct{ label, value string }{{"TOTAL SALES (VAT INC)", sharedpdf.FormatAmount(p.document.Total)}, {"LESS: VAT", sharedpdf.FormatAmount(p.document.Tax)}, {"AMOUNT: NET OF VAT", sharedpdf.FormatAmount(p.document.Subtotal)}} {
		sharedpdf.Text(p.pdf, 305, y, 9, row.label, 170, gopdf.Left, "B", sharedpdf.Purple)
		sharedpdf.Text(p.pdf, 480, y, 9, row.value, 90, gopdf.Right, "B", sharedpdf.Purple)
		y += 16
	}
}

func (p salesOrderPage) footer(y float64) {
	sharedpdf.Text(p.pdf, sharedpdf.Margin, y, 8, "If you have any questions about this sales order, please contact us", sharedpdf.Width-2*sharedpdf.Margin, gopdf.Center, "I", sharedpdf.Purple)
	sharedpdf.Text(p.pdf, sharedpdf.Margin, y+14, 8, fmt.Sprintf("(%s, %s)", p.seller.Email, p.seller.Phone), sharedpdf.Width-2*sharedpdf.Margin, gopdf.Center, "I", sharedpdf.Purple)
	sharedpdf.Text(p.pdf, sharedpdf.Margin, y+28, 9, "Thank You For Your Business!", sharedpdf.Width-2*sharedpdf.Margin, gopdf.Center, "I", sharedpdf.Purple)
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
