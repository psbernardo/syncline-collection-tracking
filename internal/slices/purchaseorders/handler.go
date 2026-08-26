package purchaseorders

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
	sharedpdf "github.com/psbernardo/syncline-collection-tracking/internal/shared/pdf"
	"github.com/psbernardo/syncline-collection-tracking/internal/slices/salesorders"
	webtemplates "github.com/psbernardo/syncline-collection-tracking/internal/web/templates"
)

//go:embed templates/*.html
var files embed.FS

type SupplierOption struct {
	ID   int64
	Name string
}
type SupplierOptions func(context.Context) ([]SupplierOption, error)
type SupplierProductReader interface {
	SupplierProducts(context.Context, int64) (map[int64]SupplierProduct, error)
}
type ProductOption struct {
	ID             int64
	SKU, Name, UOM string
	SupplierSKU    string
	ReferenceCost  money.Amount
	SourceLineID   int64
	Quantity       money.Amount
}
type ProductOptions func(context.Context) ([]ProductOption, error)
type page struct {
	Title, ActiveNav, Error             string
	Order                               interface{}
	Number, PODate, PaymentTerms, Notes string
	ExpectedDelivery                    string
	Suppliers                           []SupplierOption
	SupplierID                          int64
	Lines                               []formLine
	MissingProducts                     []MissingSupplierProduct
	Readiness                           []AllocationStatus
	ExcludedLineIDs                     []int64
	Mode                                Mode
	Products                            []ProductOption
	Edit                                bool
	SourceOrderNumber                   string
}
type listPage struct {
	Title, ActiveNav string
	Orders           []PurchaseOrder
}
type detailPage struct {
	Title, ActiveNav, Error string
	Order                   PurchaseOrder
}
type editPage struct {
	Title, ActiveNav, Error string
	Order                   PurchaseOrder
	Suppliers               []SupplierOption
	SupplierID              int64
	MissingProducts         []MissingSupplierProduct
	PaymentTerms, Notes     string
	ExpectedDelivery        string
	ExcludedLineIDs         []int64
}
type formLine struct {
	ID                 int64
	ProductID          int64
	SKU, Name, UOM     string
	Quantity, UnitCost money.Amount
	SupplierSKU        string
}
type Handler struct {
	orders         salesorders.Repository
	purchase       Repository
	suppliers      SupplierOptions
	products       SupplierProductReader
	productOptions ProductOptions
	templates      map[string]*template.Template
	pdfRenderer    purchaseOrderPDFRenderer
}

type purchaseOrderPDFRenderer interface {
	Render(io.Writer, PurchaseOrderPDFDocument) error
}

func manilaNow() time.Time { return time.Now().In(time.FixedZone("Asia/Manila", 8*60*60)) }

func NewHandler(orders salesorders.Repository, purchase Repository, suppliers SupplierOptions, products SupplierProductReader, options ...ProductOptions) (*Handler, error) {
	templates := make(map[string]*template.Template)
	for _, name := range []string{"form", "list", "detail"} {
		t, err := template.New(name).Funcs(template.FuncMap{"dict": purchaseTemplateDict}).ParseFS(webtemplates.FS, "layout.html", "partials/*.html")
		if err != nil {
			return nil, err
		}
		t, err = t.ParseFS(files, "templates/"+name+".html")
		if err != nil {
			return nil, err
		}
		templates[name] = t
	}
	var productOptions ProductOptions
	if len(options) > 0 {
		productOptions = options[0]
	}
	return &Handler{orders: orders, purchase: purchase, suppliers: suppliers, products: products, productOptions: productOptions, templates: templates, pdfRenderer: NewPurchaseOrderPDFRenderer()}, nil
}

func purchaseTemplateDict(values ...interface{}) (map[string]interface{}, error) {
	if len(values)%2 != 0 {
		return nil, errors.New("dict expects key/value pairs")
	}
	result := make(map[string]interface{}, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, errors.New("dict keys must be strings")
		}
		result[key] = values[i+1]
	}
	return result, nil
}
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /purchase-orders", h.listPage)
	mux.HandleFunc("GET /purchase-orders/new", h.standaloneNewPage)
	mux.HandleFunc("POST /purchase-orders", h.standaloneCreate)
	mux.HandleFunc("GET /purchase-orders/{id}", h.detailPage)
	mux.HandleFunc("GET /purchase-orders/{id}/pdf", h.pdf)
	mux.HandleFunc("GET /purchase-orders/{id}/edit", h.editPage)
	mux.HandleFunc("POST /purchase-orders/{id}", h.update)
	mux.HandleFunc("POST /purchase-orders/{id}/confirm", h.confirm)
	mux.HandleFunc("POST /purchase-orders/{id}/cancel", h.cancel)
	mux.HandleFunc("GET /sales-orders/{id}/purchase-order/new", h.newPage)
	mux.HandleFunc("POST /sales-orders/{id}/purchase-order/supplier-preview", h.supplierPreview)
	mux.HandleFunc("POST /sales-orders/{id}/purchase-order", h.create)
}

func (h *Handler) pdf(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	order, err := h.purchase.FindByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	var output bytes.Buffer
	if err := h.pdfRenderer.Render(&output, purchaseOrderPDFDocument(order)); err != nil {
		http.Error(w, "Unable to generate purchase order PDF", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="`+sharedpdf.SafeFilename(order.Number, "purchase-order")+`"`)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(output.Bytes())
}

func (h *Handler) listPage(w http.ResponseWriter, r *http.Request) {
	orders, err := h.purchase.List(r.Context())
	if err != nil {
		http.Error(w, "Internal server error", 500)
		return
	}
	_ = h.templates["list"].ExecuteTemplate(w, "layout", listPage{Title: "Purchase orders", ActiveNav: "purchase-orders", Orders: orders})
}

func (h *Handler) detailPage(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	order, err := h.purchase.FindByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	_ = h.templates["detail"].ExecuteTemplate(w, "layout", detailPage{Title: "Purchase order", ActiveNav: "purchase-orders", Order: order})
}

func (h *Handler) editPage(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	order, err := h.purchase.FindByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if order.Status != "OPEN" {
		http.Error(w, "Only open purchase orders can be edited", http.StatusConflict)
		return
	}
	suppliers, err := h.suppliers(r.Context())
	if err != nil {
		http.Error(w, "Internal server error", 500)
		return
	}
	mode := purchaseMode(order)
	products := []ProductOption(nil)
	if mode == DirectMode && h.productOptions != nil {
		products, err = h.productOptions(r.Context())
		if err != nil {
			http.Error(w, "Internal server error", 500)
			return
		}
	}
	h.render(w, page{Title: "Edit purchase order", ActiveNav: "purchase-orders", Order: order, Suppliers: suppliers, SupplierID: order.SupplierID, PaymentTerms: order.PaymentTerms, Notes: order.Notes, ExpectedDelivery: formatOptionalDate(order.ExpectedDelivery), Mode: mode, Products: products, Lines: purchaseFormLines(order), Edit: true, SourceOrderNumber: order.SalesOrderNumber})
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	current, err := h.purchase.FindByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	supplierID, err := strconv.ParseInt(r.FormValue("supplier_id"), 10, 64)
	if err != nil || supplierID < 1 {
		h.renderEditError(w, r, current, 0, ErrSupplierRequired)
		return
	}
	date, err := time.Parse("2006-01-02", r.FormValue("po_date"))
	if err != nil {
		h.renderEditError(w, r, current, supplierID, errors.New("enter a valid purchase-order date"))
		return
	}
	lines, parseErr := h.parseSubmittedLines(r, current)
	if parseErr != nil {
		h.renderEditError(w, r, current, supplierID, parseErr)
		return
	}
	mode := purchaseMode(current)
	if mode == SalesOrderMode {
		mappings, err := h.products.SupplierProducts(r.Context(), supplierID)
		if err != nil {
			h.renderEditError(w, r, current, supplierID, err)
			return
		}
		products := make([]ProductReference, 0, len(lines))
		for _, selected := range lines {
			for _, currentLine := range current.Lines {
				if currentLine.SalesOrderLineID == selected.SalesOrderLineID {
					products = append(products, ProductReference{ProductID: currentLine.ProductID, SKU: currentLine.SKU, Name: currentLine.Name})
					break
				}
			}
		}
		if err := ValidateSupplierAvailability(products, mappings); err != nil {
			h.renderEditError(w, r, current, supplierID, err)
			return
		}
	}
	updated, err := h.purchase.Update(r.Context(), id, CreateInput{Number: current.Number, SupplierID: supplierID, SalesOrderID: current.SalesOrderID, Mode: mode, PODate: date, PaymentTerms: strings.TrimSpace(r.FormValue("payment_terms")), ExpectedDelivery: parseOptionalDate(r.FormValue("expected_delivery_date")), Notes: strings.TrimSpace(r.FormValue("notes")), Lines: lines})
	if err != nil {
		h.renderEditError(w, r, current, supplierID, err)
		return
	}
	http.Redirect(w, r, "/purchase-orders/"+strconv.FormatInt(updated.ID, 10), http.StatusSeeOther)
}

func (h *Handler) renderEditError(w http.ResponseWriter, r *http.Request, order PurchaseOrder, supplierID int64, err error) {
	suppliers, _ := h.suppliers(r.Context())
	message, missing := validationMessage(err, supplierName(suppliers, supplierID))
	products := []ProductOption(nil)
	mode := purchaseMode(order)
	if mode == DirectMode && h.productOptions != nil {
		products, _ = h.productOptions(r.Context())
	}
	_ = h.templates["form"].ExecuteTemplate(w, "layout", page{Title: "Edit purchase order", ActiveNav: "purchase-orders", Error: message, Order: order, Suppliers: suppliers, SupplierID: supplierID, MissingProducts: missing, PaymentTerms: r.FormValue("payment_terms"), Notes: r.FormValue("notes"), ExpectedDelivery: r.FormValue("expected_delivery_date"), Mode: mode, Products: products, Lines: purchaseFormLines(order), Edit: true, SourceOrderNumber: order.SalesOrderNumber, ExcludedLineIDs: excludedLineIDs(r)})
}

func purchaseFormLines(order PurchaseOrder) []formLine {
	lines := make([]formLine, 0, len(order.Lines))
	for _, line := range order.Lines {
		lines = append(lines, formLine{ID: line.SalesOrderLineID, ProductID: line.ProductID, SKU: line.SKU, Name: line.Name, UOM: line.UOM, Quantity: line.Quantity, UnitCost: line.UnitCost, SupplierSKU: line.SupplierSKU})
	}
	return lines
}

func purchaseMode(order PurchaseOrder) Mode {
	if order.Mode != "" {
		return order.Mode
	}
	if order.SalesOrderID > 0 || len(order.SalesOrderNumbers) > 0 {
		return SalesOrderMode
	}
	return DirectMode
}

func (h *Handler) parseSubmittedLines(r *http.Request, order PurchaseOrder) ([]LineCost, error) {
	if order.Mode == DirectMode {
		lines := make([]LineCost, 0)
		for key := range r.Form {
			if !strings.HasPrefix(key, "products[") || !strings.HasSuffix(key, "].quantity") {
				continue
			}
			idText := strings.TrimSuffix(strings.TrimPrefix(key, "products["), "].quantity")
			productKey := "products[" + idText + "].product_id"
			productID, err := strconv.ParseInt(r.FormValue(productKey), 10, 64)
			if err != nil || productID < 1 {
				if _, hasProductField := r.Form[productKey]; hasProductField {
					continue
				}
				productID, err = strconv.ParseInt(idText, 10, 64)
				if err != nil {
					continue
				}
			}
			quantity, err := money.Parse(r.FormValue(key))
			if err != nil {
				return nil, errors.New("quantities must be valid positive numbers")
			}
			cost, err := money.Parse(r.FormValue("products[" + idText + "].unit_cost"))
			if err != nil {
				return nil, errors.New("unit costs must be valid non-negative numbers")
			}
			lines = append(lines, LineCost{ProductID: productID, Quantity: quantity, UnitCost: cost})
		}
		return lines, nil
	}
	lines := make([]LineCost, 0, len(order.Lines))
	for _, line := range order.Lines {
		if !hasSubmittedLine(r, line.SalesOrderLineID) {
			continue
		}
		id := strconv.FormatInt(line.SalesOrderLineID, 10)
		quantity := line.Quantity
		var err error
		if raw := r.FormValue("lines[" + id + "].quantity"); raw != "" {
			quantity, err = money.Parse(raw)
		}
		if err != nil {
			return nil, errors.New("quantities must be valid positive numbers")
		}
		cost, err := money.Parse(r.FormValue("lines[" + id + "].unit_cost"))
		if err != nil {
			return nil, errors.New("unit costs must be valid non-negative numbers")
		}
		lines = append(lines, LineCost{SalesOrderLineID: line.SalesOrderLineID, Quantity: quantity, UnitCost: cost})
	}
	return lines, nil
}

func validationMessage(err error, supplier string) (string, []MissingSupplierProduct) {
	var missingErr MissingSupplierProductsError
	if errors.As(err, &missingErr) {
		if supplier == "" {
			supplier = "the selected supplier"
		}
		return fmt.Sprintf("The following items are not configured for %s:", supplier), missingErr.Products
	}
	return err.Error(), nil
}

func supplierName(suppliers []SupplierOption, supplierID int64) string {
	for _, supplier := range suppliers {
		if supplier.ID == supplierID {
			return supplier.Name
		}
	}
	return ""
}

func parseID(value string) (int64, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id < 1 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}

func (h *Handler) newPage(w http.ResponseWriter, r *http.Request) { h.renderPage(w, r, "", "", nil) }

func (h *Handler) standaloneNewPage(w http.ResponseWriter, r *http.Request) {
	products, err := h.productOptions(r.Context())
	if err != nil {
		http.Error(w, "Internal server error", 500)
		return
	}
	number, err := h.purchase.NextNumber(r.Context(), manilaNow())
	if err != nil {
		http.Error(w, "Internal server error", 500)
		return
	}
	suppliers, err := h.suppliers(r.Context())
	if err != nil {
		http.Error(w, "Internal server error", 500)
		return
	}
	h.render(w, page{Title: "New purchase order", ActiveNav: "purchase-orders", Number: number, PODate: manilaNow().Format("2006-01-02"), Suppliers: suppliers, Mode: DirectMode, Products: products})
}

func (h *Handler) standaloneCreate(w http.ResponseWriter, r *http.Request) {
	creator, ok := h.purchase.(Creator)
	if !ok {
		http.Error(w, "Purchase order creation unavailable", 501)
		return
	}
	_ = r.ParseForm()
	supplierID, err := strconv.ParseInt(r.FormValue("supplier_id"), 10, 64)
	if err != nil {
		h.standaloneError(w, r, ErrSupplierRequired)
		return
	}
	date, err := time.Parse("2006-01-02", r.FormValue("po_date"))
	if err != nil {
		h.standaloneError(w, r, errors.New("enter a valid purchase-order date"))
		return
	}
	lines, parseErr := parseDirectLines(r)
	if parseErr != nil {
		h.standaloneError(w, r, parseErr)
		return
	}
	result, err := creator.Create(r.Context(), CreateInput{Number: strings.TrimSpace(r.FormValue("purchase_order_number")), SupplierID: supplierID, PODate: date, PaymentTerms: strings.TrimSpace(r.FormValue("payment_terms")), ExpectedDelivery: parseOptionalDate(r.FormValue("expected_delivery_date")), Notes: strings.TrimSpace(r.FormValue("notes")), Mode: DirectMode, Lines: lines})
	if err != nil {
		h.standaloneError(w, r, err)
		return
	}
	http.Redirect(w, r, "/purchase-orders/"+strconv.FormatInt(result.ID, 10), http.StatusSeeOther)
}

func (h *Handler) standaloneError(w http.ResponseWriter, r *http.Request, err error) {
	products, _ := h.productOptions(r.Context())
	suppliers, _ := h.suppliers(r.Context())
	h.render(w, page{Title: "New purchase order", ActiveNav: "purchase-orders", Error: err.Error(), Number: r.FormValue("purchase_order_number"), PODate: r.FormValue("po_date"), PaymentTerms: r.FormValue("payment_terms"), ExpectedDelivery: r.FormValue("expected_delivery_date"), Notes: r.FormValue("notes"), SupplierID: parseSupplierID(r.FormValue("supplier_id")), Suppliers: suppliers, Mode: DirectMode, Products: products})
}

func parseSupplierID(value string) int64 { id, _ := strconv.ParseInt(value, 10, 64); return id }

func parseDirectLines(r *http.Request) ([]LineCost, error) {
	lines := make([]LineCost, 0)
	for key := range r.Form {
		if !strings.HasPrefix(key, "products[") || !strings.HasSuffix(key, "].quantity") {
			continue
		}
		index := strings.TrimSuffix(strings.TrimPrefix(key, "products["), "].quantity")
		productID, err := strconv.ParseInt(r.FormValue("products["+index+"].product_id"), 10, 64)
		if err != nil || productID < 1 {
			continue
		}
		quantity, err := money.Parse(r.FormValue(key))
		if err != nil {
			return nil, errors.New("quantities must be valid positive numbers")
		}
		cost, err := money.Parse(r.FormValue("products[" + index + "].unit_cost"))
		if err != nil {
			return nil, errors.New("unit costs must be valid non-negative numbers")
		}
		lines = append(lines, LineCost{ProductID: productID, Quantity: quantity, UnitCost: cost})
	}
	return lines, nil
}

func (h *Handler) supplierPreview(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	supplierID, err := strconv.ParseInt(r.FormValue("supplier_id"), 10, 64)
	if err != nil || supplierID < 1 {
		h.renderPage(w, r, ErrSupplierRequired.Error(), r.FormValue("purchase_order_number"), nil)
		return
	}
	h.renderPage(w, r, "", r.FormValue("purchase_order_number"), nil)
}

func (h *Handler) renderPage(w http.ResponseWriter, r *http.Request, errorText, number string, missingProducts []MissingSupplierProduct) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		http.NotFound(w, r)
		return
	}
	order, err := h.orders.FindByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if number == "" {
		number, err = h.purchase.NextNumber(r.Context(), manilaNow())
		if err != nil {
			http.Error(w, "Internal server error", 500)
			return
		}
	}
	suppliers, err := h.suppliers(r.Context())
	if err != nil {
		http.Error(w, "Internal server error", 500)
		return
	}
	selectedSupplier, _ := strconv.ParseInt(r.URL.Query().Get("supplier_id"), 10, 64)
	if selectedSupplier < 1 {
		selectedSupplier, _ = strconv.ParseInt(r.FormValue("supplier_id"), 10, 64)
	}
	mappings := map[int64]SupplierProduct{}
	if selectedSupplier > 0 {
		mappings, err = h.products.SupplierProducts(r.Context(), selectedSupplier)
		if err != nil {
			http.Error(w, "Internal server error", 500)
			return
		}
	}
	var readiness []AllocationStatus
	if reader, ok := h.purchase.(ReadinessReader); ok {
		readiness, _ = reader.AllocationReadiness(r.Context(), id)
	}
	excluded := excludedLineSet(r)
	lines := make([]formLine, 0)
	productOptions := make([]ProductOption, 0, len(order.Quotation.Lines))
	for _, line := range order.Quotation.Lines {
		if excluded[line.ID] {
			continue
		}
		mapping := mappings[line.ProductID]
		if selectedSupplier > 0 && mapping.ProductID == line.ProductID {
			productOptions = append(productOptions, ProductOption{ID: line.ID, SourceLineID: line.ID, SKU: line.ProductSKU, Name: line.ProductName, UOM: line.UOM, Quantity: line.Quantity, ReferenceCost: mapping.ReferenceCost, SupplierSKU: mapping.SupplierSKU})
		}
	}
	if len(missingProducts) > 0 {
		errorText = fmt.Sprintf("The following items are not configured for %s:", supplierName(suppliers, selectedSupplier))
		if supplierName(suppliers, selectedSupplier) == "" {
			errorText = "The following items are not configured for the selected supplier:"
		}
	}
	h.templates["form"].ExecuteTemplate(w, "layout", page{Title: "New purchase order", ActiveNav: "sales-orders", Error: errorText, Order: order, Number: number, PODate: manilaNow().Format("2006-01-02"), Suppliers: suppliers, SupplierID: selectedSupplier, Lines: lines, Products: productOptions, MissingProducts: missingProducts, Readiness: readiness, ExcludedLineIDs: excludedLineIDs(r), Mode: SalesOrderMode})
}
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		http.NotFound(w, r)
		return
	}
	supplierID, err := strconv.ParseInt(r.FormValue("supplier_id"), 10, 64)
	if err != nil || supplierID < 1 {
		h.renderPage(w, r, ErrSupplierRequired.Error(), r.FormValue("purchase_order_number"), nil)
		return
	}
	date, err := time.Parse("2006-01-02", r.FormValue("po_date"))
	if err != nil {
		h.renderPage(w, r, "enter a valid purchase-order date", r.FormValue("purchase_order_number"), nil)
		return
	}
	order, err := h.orders.FindByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	var costs []LineCost
	for _, source := range order.Quotation.Lines {
		if !hasSubmittedLine(r, source.ID) {
			continue
		}
		value, parseErr := money.Parse(r.FormValue("lines[" + strconv.FormatInt(source.ID, 10) + "].unit_cost"))
		if parseErr != nil {
			h.renderPage(w, r, "unit costs must be valid non-negative numbers", r.FormValue("purchase_order_number"), nil)
			return
		}
		quantity := source.Quantity
		if raw := r.FormValue("lines[" + strconv.FormatInt(source.ID, 10) + "].quantity"); raw != "" {
			quantity, parseErr = money.Parse(raw)
			if parseErr != nil {
				h.renderPage(w, r, "quantities must be valid positive numbers", r.FormValue("purchase_order_number"), nil)
				return
			}
		}
		costs = append(costs, LineCost{SalesOrderLineID: source.ID, Quantity: quantity, UnitCost: value})
	}
	mappings, err := h.products.SupplierProducts(r.Context(), supplierID)
	if err != nil {
		h.renderPage(w, r, err.Error(), r.FormValue("purchase_order_number"), nil)
		return
	}
	lines, err := BuildLines(order, costs, mappings)
	if err != nil {
		message, missing := validationMessage(err, "")
		h.renderPage(w, r, message, r.FormValue("purchase_order_number"), missing)
		return
	}
	_ = lines
	result, err := h.purchase.CreateFromSalesOrder(r.Context(), CreateInput{Number: strings.TrimSpace(r.FormValue("purchase_order_number")), SupplierID: supplierID, SalesOrderID: id, PODate: date, PaymentTerms: strings.TrimSpace(r.FormValue("payment_terms")), ExpectedDelivery: parseOptionalDate(r.FormValue("expected_delivery_date")), Notes: strings.TrimSpace(r.FormValue("notes")), Lines: costs})
	if err != nil {
		message, missing := validationMessage(err, "")
		h.renderPage(w, r, message, r.FormValue("purchase_order_number"), missing)
		return
	}
	http.Redirect(w, r, "/sales-orders/"+strconv.FormatInt(result.SalesOrderID, 10), http.StatusSeeOther)
}

func (h *Handler) confirm(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	lifecycle, ok := h.purchase.(Lifecycle)
	if !ok {
		http.Error(w, "Lifecycle actions unavailable", 501)
		return
	}
	if err := lifecycle.Confirm(r.Context(), id); err != nil {
		order, _ := h.purchase.FindByID(r.Context(), id)
		_ = h.templates["detail"].ExecuteTemplate(w, "layout", detailPage{Title: "Purchase order", ActiveNav: "purchase-orders", Error: err.Error(), Order: order})
		return
	}
	http.Redirect(w, r, "/purchase-orders/"+strconv.FormatInt(id, 10), http.StatusSeeOther)
}
func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	lifecycle, ok := h.purchase.(Lifecycle)
	if !ok {
		http.Error(w, "Lifecycle actions unavailable", 501)
		return
	}
	if err := lifecycle.Cancel(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	http.Redirect(w, r, "/purchase-orders/"+strconv.FormatInt(id, 10), http.StatusSeeOther)
}
func parseOptionalDate(value string) *time.Time {
	if value == "" {
		return nil
	}
	date, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil
	}
	return &date
}
func formatOptionalDate(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format("2006-01-02")
}

func hasSubmittedLine(r *http.Request, id int64) bool {
	prefix := "lines[" + strconv.FormatInt(id, 10) + "]."
	for key := range r.Form {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

func excludedLineSet(r *http.Request) map[int64]bool {
	result := make(map[int64]bool)
	for _, value := range r.Form["excluded_line_ids"] {
		if id, err := strconv.ParseInt(value, 10, 64); err == nil && id > 0 {
			result[id] = true
		}
	}
	return result
}

func excludedLineIDs(r *http.Request) []int64 {
	result := make([]int64, 0)
	for id := range excludedLineSet(r) {
		result = append(result, id)
	}
	return result
}
func (h *Handler) render(w http.ResponseWriter, data page) {
	_ = h.templates["form"].ExecuteTemplate(w, "layout", data)
}
