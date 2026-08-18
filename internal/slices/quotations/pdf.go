package quotations

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/signintech/gopdf"
)

const (
	pdfWidth              = 595.0
	pdfHeight             = 842.0
	pdfMargin             = 24.0
	pdfPurple             = "purple"
	pdfLightPurple        = "light-purple"
	pdfGray               = "gray"
	pdfPageBottom         = 818.0
	pdfFirstTableY        = 297.0
	pdfContinuationTableY = 35.0
	pdfDescriptionWidth   = 205.0
	pdfRowSpacingFactor   = 1.20
	pdfFooterGap          = 3 * 28.35
	pdfTotalsHeight       = 48.0
	pdfFooterSpacing      = 24.0
	pdfFooterHeight       = 42.0
)

type SellerProfile struct {
	Name    string
	Address string
	Email   string
	Phone   string
	Logo    string
}

type QuotationPDFRenderer struct {
	Seller   SellerProfile
	Font     string
	Title    string
	Metadata []PDFMetadata
}

type PDFMetadata struct {
	Label string
	Value string
}

func NewQuotationPDFRenderer() *QuotationPDFRenderer {
	return &QuotationPDFRenderer{Seller: SellerProfile{
		Name:    envOr("SELLER_NAME", "Syncline Consumer Goods Trading"),
		Address: envOr("SELLER_ADDRESS", "402-E Marigold Street Lakeview Homes 1, Putatan Muntinlupa City"),
		Email:   envOr("SELLER_EMAIL", "syncline.mae@gmail.com"),
		Phone:   envOr("SELLER_PHONE", "63 927 670 7281"),
		Logo:    findPDFLogo(),
	}, Font: findPDFFont(), Title: "QUOTATION"}
}

func NewDocumentPDFRenderer(title string) *QuotationPDFRenderer {
	renderer := NewQuotationPDFRenderer()
	if strings.TrimSpace(title) != "" {
		renderer.Title = title
	}
	return renderer
}

func (r *QuotationPDFRenderer) Render(w io.Writer, quotation Quotation) error {
	if quotation.Number == "" || quotation.CompanyName == "" || len(quotation.Lines) == 0 {
		return fmt.Errorf("quotation PDF requires a number, customer, and at least one line")
	}

	pdf := &gopdf.GoPdf{}
	if r.Font == "" {
		return fmt.Errorf("quotation PDF font is not configured")
	}
	// Use a full standard A4 portrait page, not a half-page or landscape page.
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	for _, family := range []string{"Quotation", "QuotationBold", "QuotationItalic", "QuotationBoldItalic"} {
		if err := pdf.AddTTFFont(family, r.Font); err != nil {
			return fmt.Errorf("load quotation PDF font: %w", err)
		}
	}
	pdf.SetMargins(pdfMargin, pdfMargin, pdfMargin, pdfMargin)
	pdf.SetCompressLevel(6)
	pdf.AddPage()

	metadata := r.Metadata
	if len(metadata) == 0 {
		metadata = quotationMetadata(quotation)
	}
	page := pdfPage{pdf: pdf, seller: r.Seller, quotation: quotation, title: r.Title, metadata: metadata}
	page.drawHeader()
	page.drawSeller()
	page.drawCustomer()
	y := page.drawTableHeader(pdfFirstTableY)

	lastRowBottom := y
	for index, line := range quotation.Lines {
		lines, err := page.pdf.SplitText(asciiText(line.ProductName), pdfDescriptionWidth)
		if err != nil {
			return fmt.Errorf("wrap quotation line %d: %w", index+1, err)
		}
		rowHeight := maxFloat(25, float64(len(lines))*9+3)
		rowHeight *= pdfRowSpacingFactor
		needsPage := y+rowHeight > page.rowBottom()
		if index == len(quotation.Lines)-1 && !page.finalContentFits(y+rowHeight) {
			needsPage = true
		}
		if needsPage {
			pdf.AddPage()
			page = pdfPage{pdf: pdf, seller: r.Seller, quotation: quotation, title: r.Title, metadata: metadata}
			y = page.drawTableHeader(pdfContinuationTableY)
		}
		page.drawLine(y, rowHeight, index+1, line, lines)
		y += rowHeight
		lastRowBottom = y
	}

	// Keep the final section close to the last row instead of anchoring it to
	// the bottom of the page. The look-ahead above normally guarantees this
	// fits; retain a safe fallback for an unusually tall wrapped row.
	if !page.finalContentFits(lastRowBottom) {
		pdf.AddPage()
		page = pdfPage{pdf: pdf, seller: r.Seller, quotation: quotation, title: r.Title, metadata: metadata}
		lastRowBottom = pdfContinuationTableY + 20
	}
	totalsY := lastRowBottom + pdfFooterGap
	page.drawTotals(totalsY)
	page.drawFooter(totalsY + pdfTotalsHeight + pdfFooterSpacing)
	bytes, err := pdf.GetBytesPdfReturnErr()
	if err != nil {
		return fmt.Errorf("build quotation PDF: %w", err)
	}
	if _, err := w.Write(bytes); err != nil {
		return fmt.Errorf("write quotation PDF: %w", err)
	}
	return nil
}

func (p pdfPage) rowBottom() float64 { return pdfPageBottom }

func (p pdfPage) finalContentFits(lastRowBottom float64) bool {
	return lastRowBottom+pdfFooterGap+pdfTotalsHeight+pdfFooterSpacing+pdfFooterHeight <= pdfPageBottom
}

type pdfPage struct {
	pdf       *gopdf.GoPdf
	seller    SellerProfile
	quotation Quotation
	title     string
	metadata  []PDFMetadata
}

func (p pdfPage) drawHeader() {
	if p.seller.Logo != "" {
		if err := p.pdf.Image(p.seller.Logo, 32, 20, &gopdf.Rect{W: 102, H: 66}); err != nil {
			p.text(30, 62, 18, "LOGO", 115, gopdf.Center, "B", pdfPurple)
		}
	} else {
		p.text(30, 62, 18, "LOGO", 115, gopdf.Center, "B", pdfPurple)
	}
	if p.title == "QUOTATION" {
		p.text(342, 100, 25, p.title, 225, gopdf.Right, "B", pdfPurple)
		p.text(342, 125, 8.5, "QUOTE NUMBER: "+p.quotation.Number, 225, gopdf.Right, "B", pdfPurple)
	} else {
		p.text(342, 100, 25, p.title, 225, gopdf.Right, "B", pdfPurple)
	}
}

func quotationMetadata(q Quotation) []PDFMetadata {
	validity := "Not specified"
	if q.ValidityDate != nil {
		validity = formatQuotationDate(*q.ValidityDate)
	}
	return []PDFMetadata{
		{Label: "DATE", Value: formatQuotationDate(q.CreatedAtUTC)},
		{Label: "QUOTE NUMBER", Value: q.Number},
		{Label: "VALID UNTIL", Value: validity},
		{Label: "TERMS", Value: fmt.Sprintf("%d Days", q.TermsDays)},
	}
}

func (p pdfPage) drawSeller() {
	p.text(30, 100, 13.5, strings.ToUpper(asciiText(p.seller.Name)), 300, gopdf.Left, "B", pdfPurple)
	p.text(30, 115, 8.5, asciiText(p.seller.Address), 300, gopdf.Left, "", "black")
	p.text(30, 129, 8.5, "Phone: "+asciiText(p.seller.Phone), 300, gopdf.Left, "", "black")
	p.text(30, 143, 8.5, asciiText(p.seller.Email), 300, gopdf.Left, "", "black")
}

func (p pdfPage) drawCustomer() {
	p.drawParty(30, 188, "BILL TO", p.quotation.CustomerAddress, true)
	p.drawParty(30, 242, "SHIP TO", p.quotation.CustomerDeliveryAddress, false)
	metadataIndex := 0
	for _, field := range p.metadata {
		if p.title == "QUOTATION" && field.Label == "QUOTE NUMBER" {
			continue
		}
		y := 188 + float64(metadataIndex)*27
		p.text(342, y, 9, field.Label, 95, gopdf.Right, "B", pdfPurple)
		p.text(445, y, 9, field.Value, 126, gopdf.Right, "", "black")
		metadataIndex++
	}
}

func (p pdfPage) drawParty(x, y float64, title, address string, includeCompany bool) {
	p.text(x, y, 9, title, 370, gopdf.Left, "B", pdfPurple)
	addressY := y + 17
	if includeCompany {
		p.text(x, y+17, 13.5, strings.ToUpper(asciiText(p.quotation.CompanyName)), 370, gopdf.Left, "B", pdfPurple)
		addressY = y + 32
	}
	p.text(x, addressY, 9, asciiText(address), 370, gopdf.Left, "", "black")
}

func (p pdfPage) drawTableHeader(y float64) float64 {
	// Ship To address ends at approximately 326 pt; keep a small gap before the table.
	columns := []struct {
		title string
		width float64
		align int
	}{
		{"#", 25, gopdf.Center},
		{"QTY", 50, gopdf.Center},
		{"UNIT", 45, gopdf.Center},
		{"DESCRIPTION", 205, gopdf.Center},
		{"UNIT PRICE", 85, gopdf.Right},
		{"TAX %", 50, gopdf.Right},
		{"AMOUNT", 87, gopdf.Right},
	}
	x := pdfMargin
	p.pdf.SetFillColor(rgb(pdfLightPurple))
	p.pdf.SetStrokeColor(rgb(pdfLightPurple))
	p.pdf.RectFromUpperLeftWithStyle(x, y, pdfWidth-2*pdfMargin, 20, "DF")
	headerTextY := y + (20-(9.5+4))/2
	for _, column := range columns {
		p.text(x+3, headerTextY, 9.5, column.title, column.width-6, column.align, "B", pdfPurple)
		x += column.width
	}
	return y + 20
}

func (p pdfPage) drawLine(y, height float64, number int, line Line, description []string) {
	values := []struct {
		text  string
		width float64
		align int
	}{
		{fmt.Sprintf("%d.", number), 25, gopdf.Center},
		{formatQuantity(line.Quantity.Int64()), 50, gopdf.Center},
		{line.UOM, 45, gopdf.Center},
		{strings.Join(description, "\n"), 205, gopdf.Center},
		{formatUnitPrice(line.UnitPrice), 85, gopdf.Right},
		{taxPercent(line.TaxRate), 50, gopdf.Right},
		{formatUnitPrice(line.VATInclusiveTotal), 87, gopdf.Right},
	}
	x := pdfMargin
	for index, value := range values {
		textHeight := valueTextHeight(index, len(description))
		textY := y + maxFloat(0, (height-textHeight)/2)
		color := "black"
		if index == 0 {
			color = pdfPurple
		}
		if index == 3 {
			p.multiText(x+3, textY, 9, value.text, value.width-6, value.align, height)
		} else {
			p.text(x+3, textY, 9, value.text, value.width-6, value.align, "", color)
		}
		x += value.width
	}
	p.subtleLine(pdfMargin, y+height, pdfWidth-pdfMargin, y+height)
}

func taxPercent(rate int64) string {
	return fmt.Sprintf("%d%%", rate/10000)
}

func formatQuantity(value int64) string {
	return fmt.Sprintf("%d", value/10000)
}

func (p pdfPage) drawTotals(y float64) {
	x := 305.0
	rows := []struct{ label, value string }{
		{"TOTAL SALES (VAT INC)", formatAmount(p.quotation.Totals.Total)},
		{"LESS: VAT", formatAmount(p.quotation.Totals.Tax)},
		{"AMOUNT: NET OF VAT", formatAmount(p.quotation.Totals.Subtotal)},
	}
	for _, row := range rows {
		p.text(x, y, 9, row.label, 170, gopdf.Left, "B", pdfPurple)
		p.text(x+175, y, 9, row.value, 90, gopdf.Right, "B", pdfPurple)
		y += 16
	}
}

func (p pdfPage) drawFooter(y float64) {
	p.text(pdfMargin, y, 8, "If you have any questions about this price quote, please contact us", pdfWidth-2*pdfMargin, gopdf.Center, "I", pdfPurple)
	p.text(pdfMargin, y+14, 8, fmt.Sprintf("(%s, %s)", p.seller.Email, p.seller.Phone), pdfWidth-2*pdfMargin, gopdf.Center, "I", pdfPurple)
	p.text(pdfMargin, y+28, 9, "Thank You For Your Business!", pdfWidth-2*pdfMargin, gopdf.Center, "I", pdfPurple)
}

func (p pdfPage) labelValue(y float64, label, value string) {
	p.text(pdfMargin+3, y, 9.5, label+":", 125, gopdf.Left, "B", pdfPurple)
	p.text(pdfMargin+132, y, 9.5, asciiText(value), 680, gopdf.Left, "", "black")
}

func (p pdfPage) cell(x, y, width, height float64, value string, align int, style, color string) {
	p.pdf.SetXY(x, y+3)
	p.pdf.SetTextColor(rgb(color))
	_ = p.pdf.SetFont(fontFamily(style), "", 9.5)
	_ = p.pdf.MultiCellWithOption(&gopdf.Rect{W: width - 6, H: height - 4}, asciiText(value), gopdf.CellOption{Align: align, Border: gopdf.AllBorders, Float: gopdf.Bottom})
}

func (p pdfPage) text(x, y, size float64, value string, width float64, align int, style, color string) {
	p.pdf.SetXY(x, y)
	p.pdf.SetTextColor(rgb(color))
	_ = p.pdf.SetFont(fontFamily(style), "", size)
	_ = p.pdf.CellWithOption(&gopdf.Rect{W: width, H: size + 4}, asciiText(value), gopdf.CellOption{Align: align})
}

func valueTextHeight(column, descriptionLines int) float64 {
	if column == 3 {
		return float64(descriptionLines) * 9
	}
	return 13
}

func (p pdfPage) multiText(x, y, size float64, value string, width float64, align int, height float64) {
	p.pdf.SetXY(x, y)
	p.pdf.SetTextColor(rgb("black"))
	_ = p.pdf.SetFont(fontFamily(""), "", size)
	_ = p.pdf.MultiCellWithOption(&gopdf.Rect{W: width, H: height}, value, gopdf.CellOption{Align: align})
}

func (p pdfPage) rect(x, y, width, height float64, fill bool) {
	style := "D"
	if fill {
		p.pdf.SetFillColor(217, 217, 217)
		style = "DF"
	} else {
		p.pdf.SetFillColor(255, 255, 255)
	}
	p.pdf.SetStrokeColor(rgb(pdfPurple))
	p.pdf.SetLineWidth(0.7)
	p.pdf.RectFromUpperLeftWithStyle(x, y, width, height, style)
}

func (p pdfPage) line(x1, y1, x2, y2 float64) {
	p.pdf.SetStrokeColor(rgb(pdfPurple))
	p.pdf.SetLineWidth(0.7)
	p.pdf.Line(x1, y1, x2, y2)
}

func (p pdfPage) subtleLine(x1, y1, x2, y2 float64) {
	p.pdf.SetStrokeColor(218, 218, 218)
	p.pdf.SetLineWidth(0.5)
	p.pdf.Line(x1, y1, x2, y2)
}

func formatAmount(value interface{ FormatPHP() string }) string {
	return strings.ReplaceAll(value.FormatPHP(), "₱", "PHP ")
}

func formatUnitPrice(value interface{ FormatPHP() string }) string {
	return strings.TrimPrefix(formatAmount(value), "PHP ")
}

func rgb(name string) (uint8, uint8, uint8) {
	if name == pdfPurple {
		return 55, 20, 115
	}
	if name == pdfLightPurple {
		return 190, 170, 220
	}
	if name == "white" {
		return 255, 255, 255
	}
	return 0, 0, 0
}

func asciiText(value string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' || (r >= 32 && r <= 126) {
			return r
		}
		return '?'
	}, value)
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func findPDFFont() string {
	if value := strings.TrimSpace(os.Getenv("PDF_FONT_PATH")); value != "" {
		return value
	}
	candidates := []string{
		"/usr/share/fonts/truetype/montserrat/Montserrat-Regular.ttf",
		"/usr/share/fonts/truetype/montserrat/montserrat.ttf",
		"/usr/local/share/fonts/Montserrat-Regular.ttf",
		"/usr/share/fonts/truetype/msttcorefonts/Verdana.ttf",
		"/usr/share/fonts/truetype/msttcorefonts/verdana.ttf",
		"/usr/share/fonts/truetype/msttcorefonts/verdana.TTF",
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		"/usr/share/fonts/truetype/liberation2/LiberationSans-Regular.ttf",
	}
	if windir := os.Getenv("WINDIR"); windir != "" {
		candidates = append([]string{filepath.Join(windir, "Fonts", "Montserrat-Regular.ttf"), filepath.Join(windir, "Fonts", "Montserrat.ttf"), filepath.Join(windir, "Fonts", "verdana.ttf")}, candidates...)
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func findPDFLogo() string {
	if value := strings.TrimSpace(os.Getenv("SELLER_LOGO_PATH")); value != "" {
		if _, err := os.Stat(value); err == nil {
			return value
		}
	}
	for _, candidate := range []string{"/files/syncline-logo.png", "files/syncline-logo.png"} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func fontFamily(style string) string {
	switch style {
	case "B":
		return "QuotationBold"
	case "I":
		return "QuotationItalic"
	case "BI", "IB":
		return "QuotationBoldItalic"
	default:
		return "Quotation"
	}
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
