package quotations

import (
	"bytes"
	"context"
	"embed"
	"encoding/base64"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
	webtemplates "github.com/psbernardo/syncline-collection-tracking/internal/web/templates"
)

//go:embed templates/*.html
var files embed.FS

type Handler struct {
	repo               Repository
	list, form, detail *template.Template
	accountOptions     webtemplates.OptionsProvider
	productOptions     webtemplates.OptionsProvider
	numberGenerator    NumberGenerator
	pdfRenderer        *QuotationPDFRenderer
}
type listPage struct {
	Title, ActiveNav string
	Quotations       []Quotation
}
type formPage struct {
	Title, ActiveNav, Error string
	Quotation               Quotation
	Mode, Action, Version   string
	CustomerSelect          webtemplates.SearchableSelectViewModel
	ProductOptions          []webtemplates.SearchableSelectOption
}
type detailPage struct {
	Title, ActiveNav string
	Quotation        Quotation
}

const quotationDateLayout = "01/02/2006"

func NewHandler(repo Repository, providers ...webtemplates.OptionsProvider) (*Handler, error) {
	var accountOptions, productOptions webtemplates.OptionsProvider
	if len(providers) > 0 {
		accountOptions = providers[0]
	}
	if len(providers) > 1 {
		productOptions = providers[1]
	}
	functions := template.FuncMap{"dict": templateDict, "taxRate": formatTaxRate, "taxLabel": TaxLabel, "percent": formatPercent, "quotationDate": formatQuotationDate, "quotationDateInput": formatQuotationDateInput}
	l, err := template.New("list").Funcs(functions).ParseFS(webtemplates.FS, "layout.html", "partials/*.html")
	if err != nil {
		return nil, err
	}
	l, err = l.ParseFS(files, "templates/list.html")
	if err != nil {
		return nil, err
	}
	f, err := template.New("form").Funcs(functions).ParseFS(webtemplates.FS, "layout.html", "partials/*.html")
	if err != nil {
		return nil, err
	}
	f, err = f.ParseFS(files, "templates/form.html")
	if err != nil {
		return nil, err
	}
	d, err := template.New("detail").Funcs(functions).ParseFS(webtemplates.FS, "layout.html", "partials/*.html")
	if err != nil {
		return nil, err
	}
	d, err = d.ParseFS(files, "templates/detail.html")
	if err != nil {
		return nil, err
	}
	numberGenerator, _ := repo.(NumberGenerator)
	return &Handler{repo: repo, list: l, form: f, detail: d, accountOptions: accountOptions, productOptions: productOptions, numberGenerator: numberGenerator, pdfRenderer: NewQuotationPDFRenderer()}, nil
}
func (h *Handler) RegisterRoutes(m *http.ServeMux) {
	m.HandleFunc("GET /quotations", h.listPage)
	m.HandleFunc("GET /quotations/new", h.newPage)
	m.HandleFunc("POST /quotations", h.create)
	m.HandleFunc("GET /quotations/{id}", h.view)
	m.HandleFunc("GET /quotations/{id}/pdf", h.pdf)
	m.HandleFunc("GET /quotations/{id}/edit", h.edit)
	m.HandleFunc("POST /quotations/{id}", h.update)
}
func (h *Handler) listPage(w http.ResponseWriter, r *http.Request) {
	values, err := h.repo.List(r.Context())
	if err != nil {
		http.Error(w, "Internal server error", 500)
		return
	}
	h.render(w, h.list, listPage{"Quotations", "quotations", values})
}
func (h *Handler) newPage(w http.ResponseWriter, r *http.Request) {
	quotation := Quotation{Status: Draft, Lines: []Line{}}
	if h.numberGenerator != nil {
		number, err := h.numberGenerator.NextNumber(r.Context())
		if err != nil {
			http.Error(w, "Internal server error", 500)
			return
		}
		quotation.Number = number
	}
	page, err := h.prepareForm(r.Context(), formPage{Title: "New quotation", ActiveNav: "quotations", Quotation: quotation, Mode: "new", Action: "/quotations"})
	if err != nil {
		http.Error(w, "Internal server error", 500)
		return
	}
	h.render(w, h.form, page)
}
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	value, err := quotationFromForm(r)
	if err != nil {
		page, _ := h.prepareForm(r.Context(), formPage{Title: "New quotation", ActiveNav: "quotations", Quotation: value, Error: err.Error(), Mode: "new", Action: "/quotations"})
		h.render(w, h.form, page)
		return
	}
	created, err := h.repo.Create(r.Context(), value)
	if err != nil {
		page, _ := h.prepareForm(r.Context(), formPage{Title: "New quotation", ActiveNav: "quotations", Quotation: value, Error: err.Error(), Mode: "new", Action: "/quotations"})
		h.render(w, h.form, page)
		return
	}
	http.Redirect(w, r, "/quotations/"+strconv.FormatInt(created.ID, 10), http.StatusSeeOther)
}
func (h *Handler) view(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	value, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	h.render(w, h.detail, detailPage{"Quotation", "quotations", value})
}

func (h *Handler) pdf(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		http.NotFound(w, r)
		return
	}
	value, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	var output bytes.Buffer
	if err := h.pdfRenderer.Render(&output, value); err != nil {
		http.Error(w, "Unable to generate quotation PDF", http.StatusInternalServerError)
		return
	}
	filename := safePDFFileName(value.Number)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(output.Bytes())
}

func safePDFFileName(number string) string {
	var builder strings.Builder
	for _, r := range number {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			builder.WriteRune(r)
		}
	}
	name := builder.String()
	if name == "" {
		name = "quotation"
	}
	return name + ".pdf"
}

func (h *Handler) edit(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	value, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if value.Status != Draft {
		http.Error(w, "Quotation is not editable", http.StatusConflict)
		return
	}
	page, err := h.prepareForm(r.Context(), formPage{Title: "Edit quotation", ActiveNav: "quotations", Quotation: value, Mode: "edit", Action: "/quotations/" + strconv.FormatInt(id, 10), Version: encodeVersion(value.RowVersion)})
	if err != nil {
		http.Error(w, "Internal server error", 500)
		return
	}
	h.render(w, h.form, page)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	current, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	_ = r.ParseForm()
	value, err := quotationFromForm(r)
	value.ID = id
	value.Number = current.Number
	value.Status = current.Status
	value.RowVersion = current.RowVersion
	page := formPage{Title: "Edit quotation", ActiveNav: "quotations", Quotation: value, Mode: "edit", Action: r.URL.Path, Version: r.FormValue("row_version")}
	if err != nil {
		page.Error = err.Error()
		page, _ = h.prepareForm(r.Context(), page)
		h.render(w, h.form, page)
		return
	}
	version, err := decodeVersion(r.FormValue("row_version"))
	if err != nil {
		page.Error = "The quotation version is invalid."
		page, _ = h.prepareForm(r.Context(), page)
		h.render(w, h.form, page)
		return
	}
	updated, err := h.repo.Update(r.Context(), value, version)
	if err != nil {
		page.Error = err.Error()
		page, _ = h.prepareForm(r.Context(), page)
		h.render(w, h.form, page)
		return
	}
	h.render(w, h.detail, detailPage{"Quotation", "quotations", updated})
}

func (h *Handler) prepareForm(ctx context.Context, page formPage) (formPage, error) {
	if h.accountOptions != nil {
		options, err := h.accountOptions(ctx)
		if err != nil {
			return page, err
		}
		page.CustomerSelect = webtemplates.SearchableSelectViewModel{ID: "quotation-customer-select", Name: "company_account_id", Label: "Customer", Placeholder: "Search customer...", Options: options, Selected: strconv.FormatInt(page.Quotation.CompanyAccountID, 10)}
	}
	if h.productOptions != nil {
		options, err := h.productOptions(ctx)
		if err != nil {
			return page, err
		}
		page.ProductOptions = quotationProductOptions(options)
	}
	return page, nil
}

func quotationProductOptions(options []webtemplates.SearchableSelectOption) []webtemplates.SearchableSelectOption {
	result := make([]webtemplates.SearchableSelectOption, len(options))
	copy(result, options)
	for index := range result {
		suffix := " (" + result[index].UOM + ")"
		result[index].Label = strings.TrimSuffix(result[index].Label, suffix)
	}
	return result
}

func templateDict(values ...any) (map[string]any, error) {
	if len(values)%2 != 0 {
		return nil, fmt.Errorf("template dict requires pairs")
	}
	result := make(map[string]any, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, fmt.Errorf("template dict key must be a string")
		}
		result[key] = values[i+1]
	}
	return result, nil
}

func formatTaxRate(rate int64) string         { return money.Amount(rate).Format() }
func formatPercent(value money.Amount) string { return value.Format() + "%" }

func quotationFromForm(r *http.Request) (Quotation, error) {
	account, err := strconv.ParseInt(r.FormValue("company_account_id"), 10, 64)
	if err != nil {
		account = 0
	}
	commissionRate, _ := money.Parse(r.FormValue("commission_rate"))
	termsDays, termsErr := strconv.Atoi(r.FormValue("terms_days"))
	supplierDelivery, supplierDeliveryErr := parseOptionalAmount(r.FormValue("supplier_delivery_cost"))
	customerDelivery, customerDeliveryErr := parseOptionalAmount(r.FormValue("customer_delivery_cost"))
	otherCost, otherCostErr := parseOptionalAmount(r.FormValue("other_cost"))
	value := Quotation{Number: strings.TrimSpace(r.FormValue("quotation_number")), CompanyAccountID: account, TermsDays: termsDays, TaxDefaultCode: strings.TrimSpace(r.FormValue("tax_default_code")), CommissionType: CommissionType(r.FormValue("commission_type")), CommissionRate: commissionRate, SupplierDeliveryCost: supplierDelivery, CustomerDeliveryCost: customerDelivery, OtherCost: otherCost}
	if supplierDeliveryErr != nil || customerDeliveryErr != nil || otherCostErr != nil {
		return value, fmt.Errorf("quotation costs contain an invalid amount")
	}
	if termsErr != nil || !ValidTermsDays(termsDays) {
		return value, fmt.Errorf("terms must be 7, 15, 30, or 45 days")
	}
	if rawDate := strings.TrimSpace(r.FormValue("validity_date")); rawDate != "" {
		validUntil, dateErr := parseValidityDate(rawDate)
		if dateErr != nil {
			return value, fmt.Errorf("valid until must be a valid date")
		}
		today := time.Now().UTC().Truncate(24 * time.Hour)
		if validUntil.Before(today) {
			return value, fmt.Errorf("valid until cannot be before today")
		}
		value.ValidityDate = &validUntil
	}
	indices := map[int]bool{}
	for key := range r.PostForm {
		if strings.HasPrefix(key, "lines[") {
			end := strings.Index(key, "]")
			if end > 6 {
				index, parseErr := strconv.Atoi(key[6:end])
				if parseErr == nil {
					indices[index] = true
				}
			}
		}
	}
	ordered := make([]int, 0, len(indices))
	for index := range indices {
		ordered = append(ordered, index)
	}
	sort.Ints(ordered)
	seenProducts := map[int64]bool{}
	for _, index := range ordered {
		prefix := "lines[" + strconv.Itoa(index) + "]."
		rawProduct := strings.TrimSpace(r.FormValue(prefix + "product_id"))
		if rawProduct == "" {
			continue
		}
		product, _ := strconv.ParseInt(rawProduct, 10, 64)
		if product <= 0 {
			return value, fmt.Errorf("line %d requires a valid product", index+1)
		}
		if seenProducts[product] {
			return value, fmt.Errorf("product %d appears more than once", product)
		}
		seenProducts[product] = true
		quantity, quantityErr := money.Parse(r.FormValue(prefix + "quantity"))
		price, priceErr := money.Parse(r.FormValue(prefix + "unit_price"))
		cost, costErr := money.Parse(r.FormValue(prefix + "supplier_cost"))
		if quantityErr != nil || priceErr != nil || costErr != nil {
			return value, fmt.Errorf("line %d contains an invalid amount", index+1)
		}
		rate := int64(0)
		rateErr := error(nil)
		if rawRate := r.FormValue(prefix + "tax_rate"); rawRate != "" {
			parsed, parseErr := money.Parse(rawRate)
			rate, rateErr = parsed.Int64(), parseErr
		}
		if rateErr != nil || rate < 0 {
			return value, fmt.Errorf("line %d contains an invalid tax rate", index+1)
		}
		value.Lines = append(value.Lines, Line{ProductID: product, Quantity: quantity, UnitPrice: price, SupplierCost: cost, UOM: strings.TrimSpace(r.FormValue(prefix + "uom")), TaxCode: strings.TrimSpace(r.FormValue(prefix + "tax_code")), TaxRate: rate})
	}
	if len(value.Lines) == 0 {
		return value, fmt.Errorf("at least one quotation line is required")
	}
	return value, nil
}

func parseOptionalAmount(value string) (money.Amount, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	return money.Parse(value)
}

func formatQuotationDate(value time.Time) string {
	return value.Format(quotationDateLayout)
}

func formatQuotationDateInput(value *time.Time) string {
	if value == nil {
		return ""
	}
	return formatQuotationDate(*value)
}

func parseValidityDate(value string) (time.Time, error) {
	for _, layout := range []string{quotationDateLayout, "2006-01-02"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid validity date")
}
func (h *Handler) render(w http.ResponseWriter, t *template.Template, value any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = t.ExecuteTemplate(w, "layout", value)
}

func encodeVersion(version []byte) string        { return base64.RawURLEncoding.EncodeToString(version) }
func decodeVersion(value string) ([]byte, error) { return base64.RawURLEncoding.DecodeString(value) }
