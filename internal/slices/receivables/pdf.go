package receivables

import (
	"fmt"
	"io"
	"strings"

	sharedpdf "github.com/psbernardo/syncline-collection-tracking/internal/shared/pdf"
	"github.com/signintech/gopdf"
)

const (
	receivablesPDFWidth  = 842.0
	receivablesPDFHeight = 595.0
	receivablesPDFMargin = 24.0
	receivablesPDFBottom = 571.0
	receivablesHeaderGap = 35.0
)

type ReceivablesPDFRenderer struct {
	Seller sharedpdf.SellerProfile
	Font   string
}

func NewReceivablesPDFRenderer() *ReceivablesPDFRenderer {
	return &ReceivablesPDFRenderer{Seller: sharedpdf.DefaultSeller(), Font: sharedpdf.FindFont()}
}

func (renderer *ReceivablesPDFRenderer) Render(w io.Writer, report ExportReport) error {
	if report.RowCount == 0 {
		return fmt.Errorf("receivables PDF requires at least one receivable")
	}
	pdf := &gopdf.GoPdf{}
	page := receivablesPDFPage{pdf: pdf, seller: renderer.Seller, report: report}
	if err := page.configure(renderer.Font); err != nil {
		return fmt.Errorf("load receivables PDF font: %w", err)
	}
	y := page.header()
	for _, company := range report.Companies {
		if y+18 > receivablesPDFBottom {
			pdf.AddPage()
			y = page.tableHeader(receivablesHeaderGap)
		}
		y = page.companyBand(y, company)
		for _, row := range company.Rows {
			height := page.rowHeight(row)
			if y+height > receivablesPDFBottom {
				pdf.AddPage()
				y = page.tableHeader(receivablesHeaderGap)
			}
			y = page.line(y, height, row)
		}
		if y+18 > receivablesPDFBottom {
			pdf.AddPage()
			y = page.tableHeader(receivablesHeaderGap)
		}
		y = page.subtotal(y, company)
	}
	if y+30 > receivablesPDFBottom {
		pdf.AddPage()
		y = page.tableHeader(receivablesHeaderGap)
	}
	y = page.grandTotal(y)
	page.footer(y + 18)
	return sharedpdf.Write(w, pdf, "receivables report")
}

type receivablesPDFPage struct {
	pdf    *gopdf.GoPdf
	seller sharedpdf.SellerProfile
	report ExportReport
}

func (page *receivablesPDFPage) configure(font string) error {
	page.pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4Landscape})
	if font == "" {
		return fmt.Errorf("PDF font is not configured")
	}
	for _, family := range []string{sharedpdf.Font, sharedpdf.FontBold, sharedpdf.FontItalic, sharedpdf.FontBoldItal} {
		if err := page.pdf.AddTTFFont(family, font); err != nil {
			return err
		}
	}
	page.pdf.SetMargins(receivablesPDFMargin, receivablesPDFMargin, receivablesPDFMargin, receivablesPDFMargin)
	page.pdf.SetCompressLevel(6)
	page.pdf.AddPage()
	return nil
}

func (page *receivablesPDFPage) header() float64 {
	if page.seller.Logo != "" {
		_ = page.pdf.Image(page.seller.Logo, 32, 20, &gopdf.Rect{W: 130, H: 84})
	}
	sharedpdf.Text(page.pdf, 30, 116, 16, strings.ToUpper(page.seller.Name), 360, gopdf.Left, "B", sharedpdf.Purple)
	sharedpdf.Text(page.pdf, 30, 133, 9, page.seller.Address, 360, gopdf.Left, "", "black")
	sharedpdf.Text(page.pdf, 30, 148, 9, "Phone: "+page.seller.Phone, 360, gopdf.Left, "", "black")
	sharedpdf.Text(page.pdf, 30, 163, 9, page.seller.Email, 360, gopdf.Left, "", "black")
	sharedpdf.Text(page.pdf, 470, 55, 22, "DELIVERY RECEIVABLES", 348, gopdf.Right, "B", sharedpdf.Purple)
	sharedpdf.Text(page.pdf, 470, 82, 9, "GROUPED RECEIVABLES REPORT", 348, gopdf.Right, "", sharedpdf.Purple)
	y := 105.0
	for _, field := range []sharedpdf.Metadata{
		{Label: "GENERATED", Value: generatedAtDisplay(page.report.GeneratedAtUTC)},
		{Label: "FILTERS", Value: filterSummary(page.report)},
	} {
		labelLines := sharedpdf.Wrap(page.pdf, field.Label, 90)
		valueLines := sharedpdf.Wrap(page.pdf, field.Value, 256)
		count := len(labelLines)
		if len(valueLines) > count {
			count = len(valueLines)
		}
		height := maxFloat(15, float64(count)*9+4)
		sharedpdf.MultiText(page.pdf, 470, y+2, 9, strings.Join(labelLines, "\n"), 90, height, gopdf.Right, "B", sharedpdf.Purple)
		sharedpdf.MultiText(page.pdf, 562, y+2, 9, strings.Join(valueLines, "\n"), 256, height, gopdf.Left, "", "black")
		y += height
	}
	sharedpdf.Line(page.pdf, receivablesPDFMargin, 205, receivablesPDFWidth-receivablesPDFMargin, 205, "gray")
	return page.tableHeader(212)
}

func (page *receivablesPDFPage) tableHeader(y float64) float64 {
	x := receivablesPDFMargin
	page.pdf.SetFillColor(sharedpdf.RGB(sharedpdf.LightPurple))
	page.pdf.SetStrokeColor(sharedpdf.RGB(sharedpdf.LightPurple))
	page.pdf.RectFromUpperLeftWithStyle(x, y, receivablesPDFWidth-2*receivablesPDFMargin, 20, "DF")
	for _, column := range receivablesPDFColumns {
		sharedpdf.Text(page.pdf, x+3, y+3, 8.5, column.title, column.width-6, column.align, "B", sharedpdf.Purple)
		x += column.width
	}
	return y + 20
}

func (page *receivablesPDFPage) companyBand(y float64, company ExportCompanyGroup) float64 {
	sharedpdf.Text(page.pdf, receivablesPDFMargin+3, y+2, 10, strings.ToUpper(fmt.Sprintf("COMPANY: %s  (%d receivable(s))", company.CompanyName, company.RowCount)), receivablesPDFWidth-2*receivablesPDFMargin, gopdf.Left, "B", sharedpdf.Purple)
	sharedpdf.Line(page.pdf, receivablesPDFMargin, y+16, receivablesPDFWidth-receivablesPDFMargin, y+16, "light-purple")
	return y + 18
}

func (page *receivablesPDFPage) rowHeight(row ExportRow) float64 {
	lines := sharedpdf.Wrap(page.pdf, row.CompanyName, receivablesPDFColumns[0].width-6)
	return maxFloat(18, float64(len(lines))*10+6)
}

func (page *receivablesPDFPage) line(y, height float64, row ExportRow) float64 {
	x := receivablesPDFMargin
	companyLines := sharedpdf.Wrap(page.pdf, row.CompanyName, receivablesPDFColumns[0].width-6)
	sharedpdf.MultiText(page.pdf, x+3, y+4, 9, strings.Join(companyLines, "\n"), receivablesPDFColumns[0].width-6, height-4, gopdf.Left, "", "black")
	x += receivablesPDFColumns[0].width
	values := []string{row.InvoiceNumber, row.PONumber, row.GrossAmountDisplay, row.DeliveryDate, row.DueDate, fmt.Sprintf("%d days", row.PaymentTermDays), row.Status}
	for index, value := range values {
		column := receivablesPDFColumns[index+1]
		sharedpdf.Text(page.pdf, x+3, y+4, 9, value, column.width-6, column.align, "", "black")
		x += column.width
	}
	sharedpdf.Line(page.pdf, receivablesPDFMargin, y+height, receivablesPDFWidth-receivablesPDFMargin, y+height, "gray")
	return y + height
}

func (page *receivablesPDFPage) subtotal(y float64, company ExportCompanyGroup) float64 {
	sharedpdf.Text(page.pdf, receivablesPDFMargin+3, y+3, 9, "SUBTOTAL", 200, gopdf.Left, "B", sharedpdf.Purple)
	sharedpdf.Text(page.pdf, page.columnStart(3)+3, y+3, 9, sharedpdf.FormatAmount(company.GrossSubtotal), receivablesPDFColumns[3].width-6, gopdf.Right, "B", "black")
	sharedpdf.Text(page.pdf, page.columnStart(4)+3, y+3, 9, fmt.Sprintf("%d receivable(s)", company.RowCount), receivablesPDFColumns[4].width+receivablesPDFColumns[5].width+receivablesPDFColumns[6].width-6, gopdf.Left, "", "black")
	sharedpdf.Line(page.pdf, receivablesPDFMargin, y+16, receivablesPDFWidth-receivablesPDFMargin, y+16, "light-purple")
	return y + 18
}

func (page *receivablesPDFPage) grandTotal(y float64) float64 {
	page.pdf.SetFillColor(sharedpdf.RGB(sharedpdf.Purple))
	page.pdf.SetStrokeColor(sharedpdf.RGB(sharedpdf.Purple))
	page.pdf.RectFromUpperLeftWithStyle(receivablesPDFMargin, y, receivablesPDFWidth-2*receivablesPDFMargin, 20, "DF")
	sharedpdf.Text(page.pdf, receivablesPDFMargin+3, y+4, 9.5, "GRAND TOTAL", 260, gopdf.Left, "B", "white")
	sharedpdf.Text(page.pdf, page.columnStart(3)+3, y+4, 9.5, sharedpdf.FormatAmount(page.report.GrandGross), receivablesPDFColumns[3].width-6, gopdf.Right, "B", "white")
	sharedpdf.Text(page.pdf, page.columnStart(4)+3, y+4, 9, fmt.Sprintf("%d receivable(s)", page.report.RowCount), receivablesPDFColumns[4].width+receivablesPDFColumns[5].width+receivablesPDFColumns[6].width-6, gopdf.Left, "", "white")
	return y + 20
}

func (page *receivablesPDFPage) footer(y float64) {
	sharedpdf.Line(page.pdf, receivablesPDFMargin, y-9, receivablesPDFWidth-receivablesPDFMargin, y-9, "gray")
	sharedpdf.Text(page.pdf, receivablesPDFMargin, y, 8, "Delivery receivables report generated by Syncline", receivablesPDFWidth-2*receivablesPDFMargin, gopdf.Center, "I", sharedpdf.Purple)
	sharedpdf.Text(page.pdf, receivablesPDFMargin, y+13, 8, fmt.Sprintf("(%s, %s)", page.seller.Email, page.seller.Phone), receivablesPDFWidth-2*receivablesPDFMargin, gopdf.Center, "I", sharedpdf.Purple)
}

func (page *receivablesPDFPage) columnStart(index int) float64 {
	x := receivablesPDFMargin
	for columnIndex := 0; columnIndex < index; columnIndex++ {
		x += receivablesPDFColumns[columnIndex].width
	}
	return x
}

var receivablesPDFColumns = []struct {
	title string
	width float64
	align int
}{
	{"COMPANY", 210, gopdf.Left},
	{"INVOICE #", 65, gopdf.Left},
	{"PO #", 70, gopdf.Left},
	{"TOTAL AMOUNT", 70, gopdf.Right},
	{"DELIVERY", 55, gopdf.Center},
	{"DUE DATE", 55, gopdf.Center},
	{"TERM", 35, gopdf.Center},
	{"STATUS", 90, gopdf.Left},
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
