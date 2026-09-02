package purchaseorders

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
	sharedpdf "github.com/psbernardo/syncline-collection-tracking/internal/shared/pdf"
	"github.com/signintech/gopdf"
)

type PurchaseOrderPDFDocument struct {
	Seller           sharedpdf.SellerProfile
	Supplier         SupplierDetails
	Number           string
	PODate           time.Time
	Status           string
	PaymentTerms     string
	ExpectedDelivery *time.Time
	SourceReferences []string
	Notes            string
	Lines            []PurchaseOrderPDFLine
	Subtotal         money.Amount
	Total            money.Amount
}

type SupplierDetails struct {
	Name string
}

type PurchaseOrderPDFLine struct {
	SKU, Name, SupplierSKU, UOM string
	Quantity, UnitCost, Total   money.Amount
}

type PurchaseOrderPDFRenderer struct {
	Seller sharedpdf.SellerProfile
	Font   string
}

const (
	purchaseOrderTableStartY   = 220.0
	purchaseOrderContinueTable = 35.0
	purchaseOrderPageBottom    = 818.0
	purchaseOrderFooterGap     = 24.0
	purchaseOrderTotalsHeight  = 40.0
	purchaseOrderFooterHeight  = 42.0
	purchaseOrderRowSpacing    = 1.20
	purchaseOrderDescriptionW  = 190.0
)

func NewPurchaseOrderPDFRenderer() *PurchaseOrderPDFRenderer {
	return &PurchaseOrderPDFRenderer{Seller: sharedpdf.DefaultSeller(), Font: sharedpdf.FindFont()}
}

func purchaseOrderPDFDocument(order PurchaseOrder) PurchaseOrderPDFDocument {
	document := PurchaseOrderPDFDocument{
		Seller:           sharedpdf.DefaultSeller(),
		Supplier:         SupplierDetails{Name: order.SupplierName},
		Number:           order.Number,
		PODate:           order.PODate,
		Status:           order.Status,
		PaymentTerms:     order.PaymentTerms,
		ExpectedDelivery: order.ExpectedDelivery,
		SourceReferences: append([]string(nil), order.SalesOrderNumbers...),
		Notes:            order.Notes,
		Subtotal:         order.Subtotal,
		Total:            order.Total,
		Lines:            make([]PurchaseOrderPDFLine, 0, len(order.Lines)),
	}
	for _, line := range order.Lines {
		document.Lines = append(document.Lines, PurchaseOrderPDFLine{
			SKU: line.SKU, Name: line.Name, SupplierSKU: line.SupplierSKU, UOM: line.UOM,
			Quantity: line.Quantity, UnitCost: line.UnitCost, Total: line.Total,
		})
	}
	return document
}

func (r *PurchaseOrderPDFRenderer) Render(w io.Writer, document PurchaseOrderPDFDocument) error {
	if document.Number == "" || document.Supplier.Name == "" || len(document.Lines) == 0 {
		return fmt.Errorf("purchase order PDF requires a number, supplier, and at least one line")
	}
	pdf := &gopdf.GoPdf{}
	if err := sharedpdf.Configure(pdf, r.Font); err != nil {
		return fmt.Errorf("load purchase order PDF font: %w", err)
	}
	p := purchaseOrderPage{pdf: pdf, seller: r.Seller, document: document}
	y := p.header()
	for index, line := range document.Lines {
		wrapped := sharedpdf.Wrap(pdf, strings.TrimSpace(line.Name), purchaseOrderDescriptionW)
		height := maxPurchaseOrderFloat(25, float64(len(wrapped))*9+3) * purchaseOrderRowSpacing
		if y+height > purchaseOrderPageBottom {
			pdf.AddPage()
			p = purchaseOrderPage{pdf: pdf, seller: r.Seller, document: document}
			y = p.tableHeader(purchaseOrderContinueTable)
		}
		p.line(y, height, index+1, line, wrapped)
		y += height
	}
	if !purchaseOrderFinalContentFits(y) {
		pdf.AddPage()
		p = purchaseOrderPage{pdf: pdf, seller: r.Seller, document: document}
		y = p.tableHeader(purchaseOrderContinueTable) + 20
	}
	totalsY := y + purchaseOrderFooterGap
	if document.Notes != "" {
		p.notes(totalsY)
		totalsY += 35
	}
	p.totals(totalsY)
	p.footer(totalsY + purchaseOrderTotalsHeight + 12)
	return sharedpdf.Write(w, pdf, "purchase order")
}

func purchaseOrderFinalContentFits(lastRowBottom float64) bool {
	return lastRowBottom+purchaseOrderFooterGap+35+purchaseOrderTotalsHeight+12+purchaseOrderFooterHeight <= purchaseOrderPageBottom
}

type purchaseOrderPage struct {
	pdf      *gopdf.GoPdf
	seller   sharedpdf.SellerProfile
	document PurchaseOrderPDFDocument
}

func (p purchaseOrderPage) header() float64 {
	p.brand()
	sharedpdf.Text(p.pdf, 342, 100, 25, "PURCHASE ORDER", 225, gopdf.Right, "B", sharedpdf.Purple)
	sharedpdf.Text(p.pdf, 342, 125, 8.5, p.document.Number, 225, gopdf.Right, "B", sharedpdf.Purple)
	metadata := []sharedpdf.Metadata{
		{Label: "PO DATE", Value: formatPurchaseOrderDate(p.document.PODate)},
		{Label: "STATUS", Value: p.document.Status},
		{Label: "PAYMENT TERMS", Value: p.document.PaymentTerms},
		{Label: "EXPECTED DELIVERY", Value: formatPurchaseOrderOptionalDate(p.document.ExpectedDelivery)},
		{Label: "SOURCE", Value: purchaseOrderSource(p.document.SourceReferences)},
	}
	y := 165.0
	for _, field := range metadata {
		sharedpdf.Text(p.pdf, 342, y, 8.5, field.Label, 98, gopdf.Right, "B", sharedpdf.Purple)
		sharedpdf.Text(p.pdf, 445, y, 8.5, field.Value, 126, gopdf.Right, "", "black")
		y += 16
	}
	sharedpdf.Text(p.pdf, 30, 165, 9, "SUPPLIER", 300, gopdf.Left, "B", sharedpdf.Purple)
	sharedpdf.Text(p.pdf, 30, 182, 13.5, strings.ToUpper(p.document.Supplier.Name), 300, gopdf.Left, "B", "black")
	return p.tableHeader(purchaseOrderTableStartY)
}

func (p purchaseOrderPage) brand() {
	if p.seller.Logo != "" {
		if err := p.pdf.Image(p.seller.Logo, 32, 20, &gopdf.Rect{W: 102, H: 66}); err != nil {
			sharedpdf.Text(p.pdf, 30, 62, 18, "LOGO", 115, gopdf.Center, "B", sharedpdf.Purple)
		}
	} else {
		sharedpdf.Text(p.pdf, 30, 62, 18, "LOGO", 115, gopdf.Center, "B", sharedpdf.Purple)
	}
	sharedpdf.Text(p.pdf, 30, 100, 13.5, strings.ToUpper(p.seller.Name), 300, gopdf.Left, "B", sharedpdf.Purple)
	sharedpdf.Text(p.pdf, 30, 115, 8.5, p.seller.Address, 300, gopdf.Left, "", "black")
	sharedpdf.Text(p.pdf, 30, 129, 8.5, "Phone: "+p.seller.Phone, 300, gopdf.Left, "", "black")
	sharedpdf.Text(p.pdf, 30, 143, 8.5, p.seller.Email, 300, gopdf.Left, "", "black")
}

func (p purchaseOrderPage) tableHeader(y float64) float64 {
	columns := purchaseOrderColumns()
	x := sharedpdf.Margin
	p.pdf.SetFillColor(sharedpdf.RGB(sharedpdf.LightPurple))
	p.pdf.SetStrokeColor(sharedpdf.RGB(sharedpdf.LightPurple))
	p.pdf.RectFromUpperLeftWithStyle(x, y, sharedpdf.Width-2*sharedpdf.Margin, 20, "DF")
	for _, column := range columns {
		sharedpdf.Text(p.pdf, x+3, y+3, 8.5, column.title, column.width-6, column.align, "B", sharedpdf.Purple)
		x += column.width
	}
	return y + 20
}

type purchaseOrderColumn struct {
	title string
	width float64
	align int
}

func purchaseOrderColumns() []purchaseOrderColumn {
	return []purchaseOrderColumn{
		{"#", 22, gopdf.Center}, {"QTY", 45, gopdf.Right}, {"UNIT", 40, gopdf.Center},
		{"DESCRIPTION", 190, gopdf.Left}, {"SUPPLIER SKU", 100, gopdf.Left},
		{"UNIT COST", 75, gopdf.Right}, {"AMOUNT", 75, gopdf.Right},
	}
}

func (p purchaseOrderPage) line(y, height float64, number int, line PurchaseOrderPDFLine, description []string) {
	values := []string{
		fmt.Sprintf("%d.", number), line.Quantity.Format(), line.UOM, strings.Join(description, "\n"),
		line.SupplierSKU, sharedpdf.FormatUnitPrice(line.UnitCost), sharedpdf.FormatUnitPrice(line.Total),
	}
	x := sharedpdf.Margin
	for index, value := range values {
		column := purchaseOrderColumns()[index]
		if index == 3 {
			sharedpdf.MultiText(p.pdf, x+3, y+4, 9, value, column.width-6, height-4, column.align, "", "black")
		} else {
			sharedpdf.Text(p.pdf, x+3, y+4, 9, value, column.width-6, column.align, "", "black")
		}
		x += column.width
	}
	sharedpdf.Line(p.pdf, sharedpdf.Margin, y+height, sharedpdf.Width-sharedpdf.Margin, y+height, "gray")
}

func (p purchaseOrderPage) notes(y float64) {
	sharedpdf.Text(p.pdf, 30, y, 9, "NOTES", 100, gopdf.Left, "B", sharedpdf.Purple)
	sharedpdf.MultiText(p.pdf, 30, y+14, 9, p.document.Notes, 300, 28, gopdf.Left, "", "black")
}

func (p purchaseOrderPage) totals(y float64) {
	for _, row := range []struct{ label, value string }{
		{"SUBTOTAL", sharedpdf.FormatAmount(p.document.Subtotal)},
		{"TOTAL", sharedpdf.FormatAmount(p.document.Total)},
	} {
		sharedpdf.Text(p.pdf, 390, y, 9, row.label, 100, gopdf.Left, "B", sharedpdf.Purple)
		sharedpdf.Text(p.pdf, 495, y, 9, row.value, 75, gopdf.Right, "B", sharedpdf.Purple)
		y += 16
	}
}

func (p purchaseOrderPage) footer(y float64) {
	sharedpdf.Text(p.pdf, sharedpdf.Margin, y, 8, "Purchase order generated by Syncline", sharedpdf.Width-2*sharedpdf.Margin, gopdf.Center, "I", sharedpdf.Purple)
	sharedpdf.Text(p.pdf, sharedpdf.Margin, y+14, 8, fmt.Sprintf("(%s, %s)", p.seller.Email, p.seller.Phone), sharedpdf.Width-2*sharedpdf.Margin, gopdf.Center, "I", sharedpdf.Purple)
	sharedpdf.Text(p.pdf, sharedpdf.Margin, y+28, 9, "Thank You For Your Business!", sharedpdf.Width-2*sharedpdf.Margin, gopdf.Center, "I", sharedpdf.Purple)
}

func purchaseOrderSource(references []string) string {
	if len(references) == 0 {
		return "Direct purchase"
	}
	return strings.Join(references, ", ")
}

func formatPurchaseOrderDate(value time.Time) string {
	if value.IsZero() {
		return "Not specified"
	}
	return value.Format("January 02 2006")
}

func formatPurchaseOrderOptionalDate(value *time.Time) string {
	if value == nil {
		return "Not specified"
	}
	return formatPurchaseOrderDate(*value)
}

func maxPurchaseOrderFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
