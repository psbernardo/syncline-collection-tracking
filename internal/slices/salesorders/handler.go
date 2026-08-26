package salesorders

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
	sharedpdf "github.com/psbernardo/syncline-collection-tracking/internal/shared/pdf"
	"github.com/psbernardo/syncline-collection-tracking/internal/slices/quotations"
	webtemplates "github.com/psbernardo/syncline-collection-tracking/internal/web/templates"
)

//go:embed templates/*.html
var files embed.FS

type CustomerOption struct {
	ID   int64
	Name string
}
type ProductOption struct {
	ID             int64
	SKU, Name, UOM string
}
type CustomerOptions func(context.Context) ([]CustomerOption, error)
type ProductOptions func(context.Context) ([]ProductOption, error)

type orderFormPage struct {
	Title, ActiveNav, Error string
	Quotation               quotations.Quotation
	Allocation              []AllocationLine
	Customers               []CustomerOption
	Products                []ProductOption
	CustomerSelect          webtemplates.SearchableSelectViewModel
	OrderLines              []orderLineForm
	CustomerID              int64
	OrderID                 int64
	TermsDays               int
	Edit                    bool
	EditAcknowledged        bool
	Standalone              bool
	PO                      string
	Number                  string
}

type orderLineForm struct {
	Index               int
	ProductID           int64
	UOM                 string
	Quantity, UnitPrice money.Amount
	TaxCode             string
	TaxRate             int64
}

type detailPage struct {
	Title, ActiveNav string
	Order            SalesOrder
}
type listPage struct {
	Title, ActiveNav string
	Orders           []SalesOrder
}

type OrderLister interface {
	List(context.Context) ([]SalesOrder, error)
}

type Handler struct {
	orders     Repository
	quotations quotations.Repository
	customers  CustomerOptions
	products   ProductOptions
	number     interface {
		NextNumber(context.Context) (string, error)
	}
	renderers map[string]*template.Template
}

func NewHandler(orders Repository, quotationRepository quotations.Repository, providers ...interface{}) *Handler {
	h := &Handler{orders: orders, quotations: quotationRepository, renderers: make(map[string]*template.Template)}
	if generator, ok := orders.(interface {
		NextNumber(context.Context) (string, error)
	}); ok {
		h.number = generator
	}
	for _, provider := range providers {
		switch value := provider.(type) {
		case CustomerOptions:
			h.customers = value
		case ProductOptions:
			h.products = value
		}
	}
	return h
}

func (h *Handler) templates(name string) (*template.Template, error) {
	if value := h.renderers[name]; value != nil {
		return value, nil
	}
	t := template.New(name)
	var err error
	t, err = t.ParseFS(files, "templates/"+name+".html")
	if err != nil {
		return nil, err
	}
	// The embedded page templates use the shared layout and navigation partials.
	t, err = t.ParseFS(webtemplates.FS, "layout.html", "partials/*.html")
	return t, err
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /sales-orders", h.listPage)
	mux.HandleFunc("GET /sales-orders/new", h.newStandalone)
	mux.HandleFunc("POST /sales-orders", h.createStandalone)
	mux.HandleFunc("GET /sales-orders/{id}", h.view)
	mux.HandleFunc("GET /sales-orders/{id}/edit", h.editStandalone)
	mux.HandleFunc("POST /sales-orders/{id}", h.updateStandalone)
	mux.HandleFunc("GET /quotations/{id}/sales-order/new", h.newFromQuotation)
	mux.HandleFunc("POST /quotations/{id}/sales-order", h.createFromQuotation)
	mux.HandleFunc("GET /sales-orders/{id}/pdf", h.pdf)
}

func (h *Handler) newStandalone(w http.ResponseWriter, r *http.Request) {
	page := orderFormPage{Title: "New sales order", ActiveNav: "sales-orders", Standalone: true, TermsDays: 30, OrderLines: []orderLineForm{{Index: 0}}}
	if h.number != nil {
		page.Number, _ = h.number.NextNumber(r.Context())
	}
	if err := h.loadOptions(r.Context(), &page); err != nil {
		http.Error(w, "Internal server error", 500)
		return
	}
	h.render(w, "form", page)
}

func (h *Handler) editStandalone(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	order, err := h.orders.FindByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if order.Status != Open {
		http.Error(w, "Only open sales orders can be edited", http.StatusConflict)
		return
	}
	if order.QuotationID != 0 {
		q, err := h.quotations.FindByID(r.Context(), order.QuotationID)
		if err != nil || !quotationConvertible(q) {
			http.Error(w, "This sales order can no longer be edited", http.StatusConflict)
			return
		}
		var allocation []AllocationLine
		if reader, ok := h.orders.(OrderAllocationReader); ok {
			allocation, err = reader.AllocationLinesForOrder(r.Context(), q, order.ID)
		} else if reader, ok := h.orders.(AllocationReader); ok {
			allocation, err = reader.AllocationLines(r.Context(), q)
		}
		if err != nil {
			http.Error(w, "Unable to load quotation quantities", 500)
			return
		}
		h.render(w, "form", orderFormPage{Title: "Edit sales order", ActiveNav: "sales-orders", Edit: true, OrderID: order.ID, PO: order.CustomerPONumber, Quotation: q, Allocation: allocation})
		return
	}
	page := orderFormPage{Title: "Edit sales order", ActiveNav: "sales-orders", Standalone: true, Edit: true, OrderID: order.ID, CustomerID: order.Quotation.CompanyAccountID, TermsDays: order.Quotation.TermsDays, PO: order.CustomerPONumber, Number: order.Number}
	for index, line := range order.Quotation.Lines {
		page.OrderLines = append(page.OrderLines, orderLineForm{Index: index, ProductID: line.ProductID, UOM: line.UOM, Quantity: line.Quantity, UnitPrice: line.UnitPrice, TaxCode: line.TaxCode, TaxRate: line.TaxRate})
	}
	if err := h.loadOptions(r.Context(), &page); err != nil {
		http.Error(w, "Internal server error", 500)
		return
	}
	page.CustomerSelect.Selected = strconv.FormatInt(page.CustomerID, 10)
	h.render(w, "form", page)
}

func (h *Handler) newFromQuotation(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	q, err := h.quotations.FindByID(r.Context(), id)
	if err != nil || !quotationConvertible(q) {
		http.Error(w, "Quotation is not available for conversion", http.StatusConflict)
		return
	}
	allocation := make([]AllocationLine, 0, len(q.Lines))
	if reader, ok := h.orders.(AllocationReader); ok {
		allocation, err = reader.AllocationLines(r.Context(), q)
	} else {
		for _, line := range q.Lines {
			allocation = append(allocation, AllocationLine{Line: line, Remaining: line.Quantity})
		}
	}
	if err != nil {
		http.Error(w, "Unable to load quotation quantities", 500)
		return
	}
	page := orderFormPage{Title: "Create sales order", ActiveNav: "quotations", Quotation: q, Allocation: allocation}
	if h.number != nil {
		page.Number, _ = h.number.NextNumber(r.Context())
	}
	h.render(w, "form", page)
}

func (h *Handler) createFromQuotation(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	q, err := h.quotations.FindByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if !quotationConvertible(q) {
		http.Error(w, "Quotation is not available for conversion", http.StatusConflict)
		return
	}
	po, poErr := ValidateCustomerPO(r.FormValue("customer_po_number"))
	if poErr != nil {
		h.render(w, "form", orderFormPage{Title: "Create sales order", ActiveNav: "quotations", Quotation: q, Error: poErr.Error(), PO: strings.TrimSpace(r.FormValue("customer_po_number")), Number: strings.TrimSpace(r.FormValue("sales_order_number"))})
		return
	}
	selections, parseErr := selectionsFromForm(r)
	if parseErr != nil {
		h.render(w, "form", orderFormPage{Title: "Create sales order", ActiveNav: "quotations", Quotation: q, Error: parseErr.Error(), PO: strings.TrimSpace(r.FormValue("customer_po_number")), Number: strings.TrimSpace(r.FormValue("sales_order_number"))})
		return
	}
	var order SalesOrder
	if creator, ok := h.orders.(NumberedSelectionCreator); ok {
		order, err = creator.CreateFromQuotationSelectionWithNumber(r.Context(), q, po, selections, r.FormValue("sales_order_number"))
	} else if creator, ok := h.orders.(SelectionCreator); ok {
		order, err = creator.CreateFromQuotationSelection(r.Context(), q, po, selections)
	} else {
		// Keep older repository implementations usable while the allocation workflow rolls out.
		order, err = h.orders.CreateFromQuotation(r.Context(), q, po)
	}
	if err != nil {
		h.render(w, "form", orderFormPage{Title: "Create sales order", ActiveNav: "quotations", Quotation: q, Error: conversionError(err), PO: strings.TrimSpace(r.FormValue("customer_po_number")), Number: strings.TrimSpace(r.FormValue("sales_order_number"))})
		return
	}
	h.redirectOrder(w, r, order)
}

func (h *Handler) createStandalone(w http.ResponseWriter, r *http.Request) {
	input, err := standaloneFromForm(r)
	if err != nil {
		h.renderStandaloneError(w, r, err)
		return
	}
	creator, ok := h.orders.(StandaloneCreator)
	if !ok {
		http.Error(w, "Standalone sales orders are not available", http.StatusNotImplemented)
		return
	}
	order, err := creator.CreateStandalone(r.Context(), input)
	if err != nil {
		h.renderStandaloneError(w, r, err)
		return
	}
	h.redirectOrder(w, r, order)
}

func (h *Handler) updateStandalone(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	current, err := h.orders.FindByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if current.QuotationID != 0 {
		if err := ValidateEditAcknowledgement(r.FormValue("edit_acknowledged")); err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		q, err := h.quotations.FindByID(r.Context(), current.QuotationID)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if !quotationConvertible(q) {
			http.Error(w, "This sales order can no longer be edited", http.StatusConflict)
			return
		}
		selections, err := selectionsFromForm(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		updater, ok := h.orders.(QuotationUpdater)
		if !ok {
			http.Error(w, "Quotation sales-order editing is not available", http.StatusNotImplemented)
			return
		}
		order, err := updater.UpdateFromQuotation(r.Context(), id, q, r.FormValue("customer_po_number"), selections)
		if err != nil {
			http.Error(w, conversionError(err), http.StatusUnprocessableEntity)
			return
		}
		h.redirectOrder(w, r, order)
		return
	}
	input, err := standaloneFromForm(r)
	if err != nil {
		h.renderStandaloneError(w, r, err, id)
		return
	}
	updater, ok := h.orders.(StandaloneUpdater)
	if !ok {
		http.Error(w, "Sales-order editing is not available", http.StatusNotImplemented)
		return
	}
	order, err := updater.UpdateStandalone(r.Context(), id, input)
	if err != nil {
		h.renderStandaloneError(w, r, err, id)
		return
	}
	h.redirectOrder(w, r, order)
}

func ValidateEditAcknowledgement(value string) error {
	value = strings.ToLower(strings.TrimSpace(value))
	if value != "1" && value != "on" && value != "true" {
		return errors.New("Please acknowledge the editing warning before saving.")
	}
	return nil
}

func (h *Handler) renderStandaloneError(w http.ResponseWriter, r *http.Request, err error, editID ...int64) {
	page := orderFormPage{Title: "New sales order", ActiveNav: "sales-orders", Standalone: true, Error: err.Error(), PO: strings.TrimSpace(r.FormValue("customer_po_number")), Number: strings.TrimSpace(r.FormValue("sales_order_number")), TermsDays: 30, OrderLines: []orderLineForm{{Index: 0}}}
	if len(editID) > 0 {
		page.Title, page.Edit, page.OrderID = "Edit sales order", true, editID[0]
	}
	page.CustomerID, _ = strconv.ParseInt(r.FormValue("company_account_id"), 10, 64)
	if loadErr := h.loadOptions(r.Context(), &page); loadErr != nil {
		http.Error(w, "Internal server error", 500)
		return
	}
	h.render(w, "form", page)
}

func (h *Handler) loadOptions(ctx context.Context, page *orderFormPage) error {
	var err error
	if h.customers != nil {
		page.Customers, err = h.customers(ctx)
		if err != nil {
			return err
		}
		options := make([]webtemplates.SearchableSelectOption, 0, len(page.Customers))
		for _, item := range page.Customers {
			options = append(options, webtemplates.SearchableSelectOption{Value: strconv.FormatInt(item.ID, 10), Label: item.Name, Search: item.Name})
		}
		page.CustomerSelect = webtemplates.SearchableSelectViewModel{ID: "sales-order-customer", Name: "company_account_id", Label: "Customer", Placeholder: "Search customer...", Options: options}
		page.CustomerSelect.Selected = strconv.FormatInt(page.CustomerID, 10)
	}
	if h.products != nil {
		page.Products, err = h.products(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) listPage(w http.ResponseWriter, r *http.Request) {
	lister, ok := h.orders.(OrderLister)
	if !ok {
		h.render(w, "list", listPage{Title: "Sales orders", ActiveNav: "sales-orders"})
		return
	}
	orders, err := lister.List(r.Context())
	if err != nil {
		http.Error(w, "Internal server error", 500)
		return
	}
	h.render(w, "list", listPage{Title: "Sales orders", ActiveNav: "sales-orders", Orders: orders})
}

func (h *Handler) view(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	order, err := h.orders.FindByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	h.render(w, "detail", detailPage{Title: "Sales order", ActiveNav: "sales-orders", Order: order})
}

func parseID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}
func quotationConvertible(q quotations.Quotation) bool {
	return q.Status == quotations.Approved || q.Status == quotations.Accepted || q.Status == ""
}
func conversionError(err error) string {
	if errors.Is(err, ErrAllocationStale) {
		return "Some quantities were converted in another order. Reload the page and review the remaining quantities."
	}
	return "Unable to create sales order: " + err.Error()
}

func selectionsFromForm(r *http.Request) ([]LineSelection, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}
	var selections []LineSelection
	for key, values := range r.PostForm {
		if !strings.HasPrefix(key, "lines[") || !strings.HasSuffix(key, "].quantity") || len(values) == 0 || strings.TrimSpace(values[0]) == "" {
			continue
		}
		start := strings.Index(key, "[") + 1
		end := strings.Index(key, "]")
		id, err := strconv.ParseInt(r.FormValue("lines["+key[start:end]+"].id"), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("a quotation line is invalid")
		}
		quantity, err := money.Parse(values[0])
		if err != nil {
			return nil, fmt.Errorf("selected quantities must be positive numbers")
		}
		if quantity == 0 {
			continue
		}
		selections = append(selections, LineSelection{QuotationLineID: id, Quantity: quantity})
	}
	return selections, nil
}

func standaloneFromForm(r *http.Request) (StandaloneOrderInput, error) {
	if err := r.ParseForm(); err != nil {
		return StandaloneOrderInput{}, err
	}
	account, err := strconv.ParseInt(r.FormValue("company_account_id"), 10, 64)
	if err != nil || account < 1 {
		return StandaloneOrderInput{}, errors.New("select a customer")
	}
	var lines []StandaloneLineInput
	for key, values := range r.PostForm {
		if !strings.HasPrefix(key, "lines[") || !strings.HasSuffix(key, "].product_id") || len(values) == 0 {
			continue
		}
		index := key[len("lines["):strings.Index(key, "]")]
		product, parseErr := strconv.ParseInt(values[0], 10, 64)
		if parseErr != nil || product < 1 {
			return StandaloneOrderInput{}, errors.New("select a valid product")
		}
		quantity, parseErr := money.Parse(r.FormValue("lines[" + index + "].quantity"))
		if parseErr != nil || quantity <= 0 {
			return StandaloneOrderInput{}, errors.New("quantities must be positive numbers")
		}
		price, parseErr := money.Parse(r.FormValue("lines[" + index + "].unit_price"))
		if parseErr != nil {
			return StandaloneOrderInput{}, errors.New("unit prices must be valid numbers")
		}
		rate, _ := money.Parse(r.FormValue("lines[" + index + "].tax_rate"))
		lines = append(lines, StandaloneLineInput{ProductID: product, Quantity: quantity, UOM: r.FormValue("lines[" + index + "].uom"), UnitPrice: price, TaxCode: r.FormValue("lines[" + index + "].tax_code"), TaxRate: rate.Int64()})
	}
	if len(lines) == 0 {
		return StandaloneOrderInput{}, errors.New("add at least one product")
	}
	po, err := ValidateCustomerPO(r.FormValue("customer_po_number"))
	if err != nil {
		return StandaloneOrderInput{}, err
	}
	terms, _ := strconv.Atoi(r.FormValue("terms_days"))
	return StandaloneOrderInput{Number: strings.TrimSpace(r.FormValue("sales_order_number")), CompanyAccountID: account, CustomerPONumber: po, TermsDays: terms, Lines: lines}, nil
}

func (h *Handler) redirectOrder(w http.ResponseWriter, r *http.Request, order SalesOrder) {
	http.Redirect(w, r, "/sales-orders/"+strconv.FormatInt(order.ID, 10), http.StatusSeeOther)
}

func (h *Handler) render(w http.ResponseWriter, name string, value any) {
	t, err := h.templates(name)
	if err != nil {
		http.Error(w, "Internal server error", 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = t.ExecuteTemplate(w, "layout", value)
}

func (h *Handler) pdf(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	order, err := h.orders.FindByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	quotationReference := order.QuotationNumber
	if quotationReference == "" {
		quotationReference = "Repeat order"
	}
	document := SalesOrderPDFDocument{Seller: sharedpdf.DefaultSeller(), Number: order.Number, Source: quotationReference, CustomerPO: order.CustomerPONumber, SalesPerson: order.SalesPerson, Customer: order.Quotation.CompanyName, BillingAddress: order.Quotation.CustomerAddress, DeliveryAddress: order.Quotation.CustomerDeliveryAddress, TermsDays: order.Quotation.TermsDays, OrderDate: order.CreatedAtUTC, Subtotal: order.Quotation.Totals.Subtotal, Tax: order.Quotation.Totals.Tax, Total: order.Quotation.Totals.Total}
	for _, line := range order.Quotation.Lines {
		document.Lines = append(document.Lines, SalesOrderPDFLine{SKU: line.ProductSKU, Name: line.ProductName, UOM: line.UOM, TaxCode: line.TaxCode, Quantity: line.Quantity, UnitPrice: line.UnitPrice, Amount: line.VATInclusiveTotal, TaxRate: line.TaxRate})
	}
	var output bytes.Buffer
	if err := NewSalesOrderPDFRenderer().Render(&output, document); err != nil {
		http.Error(w, "Unable to generate sales order PDF", 500)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="`+safeFilename(order.Number)+`"`)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(output.Bytes())
}

func safeFilename(number string) string {
	var b strings.Builder
	for _, r := range number {
		if strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_", r) {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "sales-order.pdf"
	}
	return b.String() + ".pdf"
}
