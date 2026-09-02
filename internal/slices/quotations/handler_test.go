package quotations

import (
	"bytes"
	"context"
	webtemplates "github.com/psbernardo/syncline-collection-tracking/internal/web/templates"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

type handlerRepository struct{ created Quotation }

func (r *handlerRepository) Create(_ context.Context, value Quotation) (Quotation, error) {
	value.ID = 7
	if value.CreatedAtUTC.IsZero() {
		value.CreatedAtUTC = time.Now().UTC()
	}
	r.created = value
	return value, nil
}
func (r *handlerRepository) List(context.Context) ([]Quotation, error) { return nil, nil }
func (r *handlerRepository) FindByID(context.Context, int64) (Quotation, error) {
	return r.created, nil
}
func (r *handlerRepository) Update(context.Context, Quotation, []byte) (Quotation, error) {
	return r.created, nil
}
func (r *handlerRepository) NextNumber(context.Context) (string, error) { return "QT-00000001", nil }

func TestQuotationFromFormParsesMultipleLines(t *testing.T) {
	form := url.Values{
		"quotation_number": {"Q-100"}, "company_account_id": {"42"}, "terms_days": {"30"}, "commission_type": {"NONE"}, "commission_rate": {"0"},
		"lines[0].product_id": {"1"}, "lines[0].quantity": {"2"}, "lines[0].uom": {"PC"}, "lines[0].unit_price": {"10"}, "lines[0].supplier_cost": {"7"},
		"lines[1].product_id": {"2"}, "lines[1].quantity": {"3"}, "lines[1].uom": {"BOX"}, "lines[1].unit_price": {"20"}, "lines[1].supplier_cost": {"15"},
	}
	r := httptest.NewRequest("POST", "/quotations", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := r.ParseForm(); err != nil {
		t.Fatal(err)
	}
	value, err := quotationFromForm(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(value.Lines) != 2 || value.Lines[1].ProductID != 2 {
		t.Fatalf("unexpected lines: %+v", value.Lines)
	}
	if value.Number != "Q-100" {
		t.Fatalf("quotation number was not preserved for the generated form value: %q", value.Number)
	}
}

func TestGeneratedQuotationNumber(t *testing.T) {
	if got := generatedQuotationNumber(1); got != "QT-00000001" {
		t.Fatalf("unexpected first number: %s", got)
	}
	if got := generatedQuotationNumber(42); got != "QT-00000042" {
		t.Fatalf("unexpected number: %s", got)
	}
}

func TestQuotationFromFormParsesValidityAndTerms(t *testing.T) {
	form := url.Values{"company_account_id": {"42"}, "validity_date": {"12/31/2099"}, "terms_days": {"30"}, "commission_type": {"NONE"}, "commission_rate": {"0"}, "lines[0].product_id": {"1"}, "lines[0].quantity": {"1"}, "lines[0].unit_price": {"10"}, "lines[0].supplier_cost": {"0"}}
	r := httptest.NewRequest("POST", "/quotations", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := r.ParseForm(); err != nil {
		t.Fatal(err)
	}
	value, err := quotationFromForm(r)
	if err != nil {
		t.Fatal(err)
	}
	if value.ValidityDate == nil || value.TermsDays != 30 {
		t.Fatalf("unexpected quotation metadata: %+v", value)
	}
}

func TestQuotationFromFormTreatsBlankSupplierCostAsZero(t *testing.T) {
	form := url.Values{
		"company_account_id": {"42"}, "terms_days": {"30"}, "commission_type": {"NONE"}, "commission_rate": {"0"},
		"lines[0].product_id": {"1"}, "lines[0].quantity": {"1"}, "lines[0].unit_price": {"10"},
		"lines[1].product_id": {"2"}, "lines[1].quantity": {"2"}, "lines[1].unit_price": {"20"},
		"lines[1].supplier_cost": {""},
	}
	r := httptest.NewRequest("POST", "/quotations", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := r.ParseForm(); err != nil {
		t.Fatal(err)
	}
	value, err := quotationFromForm(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(value.Lines) != 2 || value.Lines[1].SupplierCost != 0 {
		t.Fatalf("unexpected lines: %+v", value.Lines)
	}
}

func TestQuotationFromFormParsesDatePickerValue(t *testing.T) {
	form := url.Values{"validity_date": {"2099-12-31"}, "terms_days": {"30"}, "commission_type": {"NONE"}, "commission_rate": {"0"}, "lines[0].product_id": {"1"}, "lines[0].quantity": {"1"}, "lines[0].unit_price": {"10"}, "lines[0].supplier_cost": {"0"}}
	r := httptest.NewRequest("POST", "/quotations", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := r.ParseForm(); err != nil {
		t.Fatal(err)
	}
	value, err := quotationFromForm(r)
	if err != nil {
		t.Fatal(err)
	}
	if value.ValidityDate == nil || value.ValidityDate.Format("01/02/2006") != "12/31/2099" {
		t.Fatalf("unexpected validity date: %v", value.ValidityDate)
	}
}

func TestQuotationDateFormattingUsesStandardFormat(t *testing.T) {
	value := time.Date(2099, time.December, 31, 0, 0, 0, 0, time.UTC)
	if got := formatQuotationDate(value); got != "12/31/2099" {
		t.Fatalf("unexpected quotation date format: %q", got)
	}
	if got := formatQuotationDateInput(&value); got != "12/31/2099" {
		t.Fatalf("unexpected quotation date input format: %q", got)
	}
	if got := formatQuotationDateInput(nil); got != "" {
		t.Fatalf("expected empty quotation date input, got %q", got)
	}
	if got := formatQuotationDateCanonical(&value); got != "2099-12-31" {
		t.Fatalf("unexpected canonical quotation date: %q", got)
	}
}

func TestQuotationEditFormUsesStandardDateValue(t *testing.T) {
	validityDate := time.Date(2099, time.December, 31, 0, 0, 0, 0, time.UTC)
	repo := &handlerRepository{created: Quotation{ID: 7, Number: "Q-100", Status: Draft, ValidityDate: &validityDate}}
	h, err := NewHandler(repo)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/quotations/7/edit", nil))
	body := w.Body.String()
	if !strings.Contains(body, `value="12/31/2099"`) {
		t.Fatalf("quotation edit form did not render the standard date: %s", body)
	}
	if !strings.Contains(body, `name="validity_date" value="2099-12-31"`) {
		t.Fatalf("quotation edit form did not render the canonical date: %s", body)
	}
	for _, expected := range []string{`data-date-display`, `data-date-picker`, `data-date-picker-trigger`, `type="date"`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("quotation edit form does not contain %q", expected)
		}
	}
}

func TestQuotationFormUsesSearchableRelationshipSelectors(t *testing.T) {
	provider := func(_ context.Context) ([]webtemplates.SearchableSelectOption, error) {
		return []webtemplates.SearchableSelectOption{{Value: "1", Label: "ACME - Widget (PC)", Description: "Long widget description", Search: "acme widget long widget description"}}, nil
	}
	h, err := NewHandler(&handlerRepository{}, provider, provider)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.newPage(w, httptest.NewRequest("GET", "/quotations/new", nil))
	body := w.Body.String()
	for _, expected := range []string{"QT-00000001", "Search customer...", "ACME - Widget (PC)", "Long widget description", "Quoted items", "quotation-row-template", "class=\"product-cell\" colspan=\"2\"", "quotation-line-profitability", "Cost &amp; profit", "data-field=\"supplier-cost\"", "quotation-grid-value", `value="VAT12" selected`, "VAT-inclusive, 12% VAT"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("quotation form does not contain %q", expected)
		}
	}
	if strings.Contains(body, ">Description</th>") || strings.Contains(body, "data-field=\"description\"") {
		t.Fatal("quotation form still contains a separate description column")
	}
	if strings.Contains(body, "class=\"cost-column\"") || strings.Contains(body, ">Profit / margin</th>") {
		t.Fatal("quotation form still renders cost or profit as grid columns")
	}
}

func TestCreatePassesAllQuotationLinesToRepository(t *testing.T) {
	repo := &handlerRepository{}
	h, err := NewHandler(repo)
	if err != nil {
		t.Fatal(err)
	}
	form := url.Values{"quotation_number": {"Q-100"}, "company_account_id": {"42"}, "terms_days": {"30"}, "commission_type": {"NONE"}, "commission_rate": {"0"},
		"lines[0].product_id": {"1"}, "lines[0].quantity": {"2"}, "lines[0].uom": {"PC"}, "lines[0].unit_price": {"10"}, "lines[0].supplier_cost": {"7"},
		"lines[1].product_id": {"2"}, "lines[1].quantity": {"3"}, "lines[1].uom": {"BOX"}, "lines[1].unit_price": {"20"}, "lines[1].supplier_cost": {"15"}}
	r := httptest.NewRequest("POST", "/quotations", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.create(w, r)
	if len(repo.created.Lines) != 2 {
		t.Fatalf("expected two persisted lines, got %d", len(repo.created.Lines))
	}
	if repo.created.CreatedAtUTC.IsZero() {
		t.Fatal("expected created timestamp to be set when quotation is created")
	}
}

func TestQuotationPDFDownloadReturnsPDFAttachment(t *testing.T) {
	repo := &handlerRepository{created: Quotation{
		ID:                    7,
		Number:                "QT-00000007",
		CompanyName:           "ACME Corporation",
		CustomerAddress:       "1 Main Street",
		CustomerContactPerson: "Jane Doe",
		CustomerContactNumber: "555-0100",
		CreatedAtUTC:          time.Date(2026, time.March, 13, 0, 0, 0, 0, time.UTC),
		TermsDays:             15,
		Lines:                 []Line{{ProductName: "HP 680 TRI-COLOR", Quantity: 500000, UOM: "PC", UnitPrice: 6750000, VATInclusiveTotal: 337500000}},
		Totals:                Totals{Subtotal: 669642900, Tax: 80357100, Total: 750000000},
	}}
	h, err := NewHandler(repo)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/quotations/7/pdf", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Content-Type"); got != "application/pdf" {
		t.Fatalf("unexpected content type: %q", got)
	}
	if got := w.Header().Get("Content-Disposition"); got != `attachment; filename="QT-00000007.pdf"` {
		t.Fatalf("unexpected content disposition: %q", got)
	}
	if !bytes.HasPrefix(w.Body.Bytes(), []byte("%PDF-")) || !bytes.HasSuffix(bytes.TrimSpace(w.Body.Bytes()), []byte("%%EOF")) {
		t.Fatalf("response is not a PDF: %q", w.Body.Bytes()[:minInt(12, len(w.Body.Bytes()))])
	}
	if got := w.Header().Get("Content-Length"); got != strconv.Itoa(w.Body.Len()) {
		t.Fatalf("unexpected content length: header=%q body=%d", got, w.Body.Len())
	}
}

func TestQuotationPDFPaginatesItemsWithoutRepeatingFullHeader(t *testing.T) {
	lines := make([]Line, 11)
	for index := range lines {
		lines[index] = Line{
			ProductName:       "Short product description",
			Quantity:          10000,
			UOM:               "PC",
			UnitPrice:         10000,
			VATInclusiveTotal: 10000,
		}
	}
	quotation := Quotation{
		Number:       "QT-00000008",
		CompanyName:  "ACME Corporation",
		CreatedAtUTC: time.Date(2026, time.March, 13, 0, 0, 0, 0, time.UTC),
		Lines:        lines,
	}

	var output bytes.Buffer
	if err := NewQuotationPDFRenderer().Render(&output, quotation); err != nil {
		t.Fatal(err)
	}
	if got := pdfPageCount(output.Bytes()); got != 2 {
		t.Fatalf("expected final item section and dynamic totals page, got %d pages", got)
	}
	if pdfWidth != 595 || pdfHeight != 842 {
		t.Fatalf("expected standard A4 portrait dimensions, got %.0fx%.0f", pdfWidth, pdfHeight)
	}
}

func TestQuotationPDFFitsLongMetadata(t *testing.T) {
	quotation := Quotation{Number: "QT-00000024", CompanyName: "ACME Corporation", CustomerAddress: "Customer address", Lines: []Line{{ProductName: "Product", Quantity: 10000, UOM: "PC", UnitPrice: 10000, VATInclusiveTotal: 10000}}}
	renderer := NewQuotationPDFRenderer()
	renderer.Metadata = []PDFMetadata{
		{Label: "SALES ORDER NO.", Value: "SO-00000003"},
		{Label: "ORDER DATE", Value: "August 23 2026"},
		{Label: "QUOTATION REF #", Value: "QT-00000024"},
		{Label: "SALES PERSON", Value: "A very long sales person name for layout testing"},
		{Label: "PO NUMBER", Value: "CUSTOMER-PO-2026-000001-LONG-VALUE"},
		{Label: "PAYMENT TERMS #", Value: "Net 30 days from customer receipt"},
	}
	var output bytes.Buffer
	if err := renderer.Render(&output, quotation); err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(output.Bytes(), []byte("%PDF-")) {
		t.Fatal("response is not a PDF")
	}
}

func pdfPageCount(value []byte) int {
	count := bytes.Count(value, []byte("/Type /Page"))
	return count - bytes.Count(value, []byte("/Type /Pages"))
}

func TestSafePDFFileNameRemovesPathCharacters(t *testing.T) {
	if got := safePDFFileName(`../QT/7`); got != "QT7.pdf" {
		t.Fatalf("unexpected safe filename: %q", got)
	}
}

func TestFormatQuantityUsesWholeNumbers(t *testing.T) {
	if got := formatQuantity(1500000); got != "150" {
		t.Fatalf("expected whole-number quantity, got %q", got)
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
