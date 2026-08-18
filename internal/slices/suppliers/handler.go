package suppliers

import (
	"embed"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
	webtemplates "github.com/psbernardo/syncline-collection-tracking/internal/web/templates"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

//go:embed templates/*.html
var files embed.FS

type Handler struct {
	s                                 *Service
	list, form, products, productForm *template.Template
}
type listPage struct {
	Title, ActiveNav string
	Suppliers        []Supplier
}
type formPage struct {
	Title, ActiveNav     string
	Supplier             Supplier
	Errors               ValidationErrors
	Action, Key, Version string
}
type productsPage struct {
	Title, ActiveNav string
	Products         []SupplierProduct
}

func NewHandler(s *Service) (*Handler, error) {
	l, e := template.New("list").ParseFS(webtemplates.FS, "layout.html", "partials/*.html")
	if e != nil {
		return nil, e
	}
	l, e = l.ParseFS(files, "templates/list.html")
	if e != nil {
		return nil, e
	}
	f, e := template.New("form").ParseFS(webtemplates.FS, "layout.html", "partials/*.html")
	if e != nil {
		return nil, e
	}
	f, e = f.ParseFS(files, "templates/form.html")
	if e != nil {
		return nil, e
	}
	p, e := template.New("products").ParseFS(webtemplates.FS, "layout.html", "partials/*.html")
	if e != nil {
		return nil, e
	}
	p, e = p.ParseFS(files, "templates/products.html")
	if e != nil {
		return nil, e
	}
	pf, e := template.New("product-form").ParseFS(webtemplates.FS, "layout.html", "partials/*.html")
	if e != nil {
		return nil, e
	}
	pf, e = pf.ParseFS(files, "templates/product-form.html")
	if e != nil {
		return nil, e
	}
	return &Handler{s, l, f, p, pf}, nil
}
func (h *Handler) RegisterRoutes(m *http.ServeMux) {
	m.HandleFunc("GET /suppliers", h.listPage)
	m.HandleFunc("GET /suppliers/new", h.newPage)
	m.HandleFunc("POST /suppliers", h.create)
	m.HandleFunc("GET /suppliers/{id}/edit", h.edit)
	m.HandleFunc("POST /suppliers/{id}", h.update)
	m.HandleFunc("GET /supplier-products", h.productList)
	m.HandleFunc("GET /supplier-products/new", h.newProduct)
	m.HandleFunc("POST /supplier-products", h.createProduct)
	m.HandleFunc("GET /supplier-products/{id}/edit", h.editProduct)
	m.HandleFunc("POST /supplier-products/{id}", h.updateProduct)
}
func (h *Handler) listPage(w http.ResponseWriter, r *http.Request) {
	v, e := h.s.List(r.Context(), r.URL.Query().Get("all") != "1")
	if e != nil {
		h.err(w, e)
		return
	}
	h.render(w, h.list, "layout", listPage{"Suppliers", "suppliers", v})
}
func (h *Handler) newPage(w http.ResponseWriter, r *http.Request) {
	k := fmt.Sprintf("supplier-%d", time.Now().UnixNano())
	h.render(w, h.form, "layout", formPage{"New supplier", "suppliers", Supplier{IsActive: true}, nil, "/suppliers", k, ""})
}
func (h *Handler) edit(w http.ResponseWriter, r *http.Request) {
	id, e := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if e != nil {
		http.NotFound(w, r)
		return
	}
	v, e := h.s.Get(r.Context(), id)
	if errors.Is(e, ErrSupplierNotFound) {
		http.NotFound(w, r)
		return
	}
	if e != nil {
		h.err(w, e)
		return
	}
	h.render(w, h.form, "layout", formPage{"Edit supplier", "suppliers", v, nil, "/suppliers/" + strconv.FormatInt(id, 10), fmt.Sprintf("supplier-%d", time.Now().UnixNano()), base64.RawURLEncoding.EncodeToString(v.RowVersion)})
}
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	c := CreateCommand{r.FormValue("name"), r.FormValue("contact_person"), r.FormValue("contact_number"), r.FormValue("email"), r.FormValue("billing_address"), r.FormValue("delivery_address"), r.FormValue("tax_identifier"), requestID(r), r.FormValue("idempotency_key"), "local-admin"}
	if _, e := h.s.Create(r.Context(), c); e != nil {
		var ve ValidationErrors
		if errors.As(e, &ve) {
			h.render(w, h.form, "layout", formPage{"New supplier", "suppliers", Supplier{Name: c.Name, ContactPerson: c.ContactPerson, ContactNumber: c.ContactNumber, Email: c.Email, BillingAddress: c.BillingAddress, DeliveryAddress: c.DeliveryAddress, TaxIdentifier: c.TaxIdentifier, IsActive: true}, ve, "/suppliers", c.IdempotencyKey, ""})
			return
		}
		h.err(w, e)
		return
	}
	http.Redirect(w, r, "/suppliers", 303)
}
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	id, e := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if e != nil {
		http.NotFound(w, r)
		return
	}
	v, _ := base64.RawURLEncoding.DecodeString(r.FormValue("row_version"))
	c := UpdateCommand{Supplier: Supplier{ID: id, Name: r.FormValue("name"), ContactPerson: r.FormValue("contact_person"), ContactNumber: r.FormValue("contact_number"), Email: r.FormValue("email"), BillingAddress: r.FormValue("billing_address"), DeliveryAddress: r.FormValue("delivery_address"), TaxIdentifier: r.FormValue("tax_identifier"), IsActive: r.FormValue("is_active") == "on"}, OriginalVersion: v, RequestID: requestID(r), IdempotencyKey: r.FormValue("idempotency_key"), ActorID: "local-admin"}
	if _, e = h.s.Update(r.Context(), c); e != nil {
		var ve ValidationErrors
		if errors.As(e, &ve) {
			h.render(w, h.form, "layout", formPage{"Edit supplier", "suppliers", c.Supplier, ve, r.URL.Path, c.IdempotencyKey, r.FormValue("row_version")})
			return
		}
		h.err(w, e)
		return
	}
	http.Redirect(w, r, "/suppliers", 303)
}
func (h *Handler) productList(w http.ResponseWriter, r *http.Request) {
	v, e := h.s.Products(r.Context(), r.URL.Query().Get("all") != "1")
	if e != nil {
		h.err(w, e)
		return
	}
	h.render(w, h.products, "layout", productsPage{"Supplier products", "supplier-products", v})
}

type productFormPage struct {
	Title, ActiveNav, Action, Key, Version string
	Product                                SupplierProduct
	Errors                                 ValidationErrors
	Suppliers                              []Supplier
	Products                               []ProductOption
	Editing                                bool
	SupplierSelect                         webtemplates.SearchableSelectViewModel
	ProductSelect                          webtemplates.SearchableSelectViewModel
}

func (h *Handler) newProduct(w http.ResponseWriter, r *http.Request) {
	h.renderProductForm(w, r, productFormPage{Title: "New supplier product", ActiveNav: "supplier-products", Action: "/supplier-products", Key: fmt.Sprintf("supplier-product-%d", time.Now().UnixNano())})
}
func (h *Handler) createProduct(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	sid, sidErr := parseID(r.FormValue("supplier_id"), "SupplierID")
	pid, pidErr := parseID(r.FormValue("product_id"), "ProductID")
	if len(sidErr)+len(pidErr) > 0 {
		h.renderProductForm(w, r, productFormPage{Title: "New supplier product", ActiveNav: "supplier-products", Action: "/supplier-products", Key: r.FormValue("idempotency_key"), Product: SupplierProduct{SupplierID: sid, ProductID: pid, SupplierSKU: r.FormValue("supplier_sku")}, Errors: mergeErrors(sidErr, pidErr)})
		return
	}
	c := CreateProductCommand{sid, pid, r.FormValue("supplier_sku"), r.FormValue("reference_cost"), requestID(r), r.FormValue("idempotency_key"), "local-admin"}
	if _, e := h.s.CreateProduct(r.Context(), c); e != nil {
		var ve ValidationErrors
		if errors.As(e, &ve) {
			h.renderProductForm(w, r, productFormPage{Title: "New supplier product", ActiveNav: "supplier-products", Action: "/supplier-products", Key: c.IdempotencyKey, Product: SupplierProduct{SupplierID: sid, ProductID: pid, SupplierSKU: c.SupplierSKU}, Errors: ve})
			return
		}
		if errors.Is(e, ErrDuplicateSupplierProduct) {
			h.renderProductForm(w, r, productFormPage{Title: "New supplier product", ActiveNav: "supplier-products", Action: "/supplier-products", Key: c.IdempotencyKey, Product: SupplierProduct{SupplierID: sid, ProductID: pid, SupplierSKU: c.SupplierSKU}, Errors: ValidationErrors{"Relationship": "This supplier and product relationship already exists."}})
			return
		}
		if errors.Is(e, ErrSupplierNotFound) || errors.Is(e, ErrProductNotFound) {
			h.renderProductForm(w, r, productFormPage{Title: "New supplier product", ActiveNav: "supplier-products", Action: "/supplier-products", Key: c.IdempotencyKey, Product: SupplierProduct{SupplierID: sid, ProductID: pid, SupplierSKU: c.SupplierSKU}, Errors: ValidationErrors{"Relationship": "Select active supplier and product records."}})
			return
		}
		h.err(w, e)
		return
	}
	http.Redirect(w, r, "/supplier-products", 303)
}
func (h *Handler) editProduct(w http.ResponseWriter, r *http.Request) {
	id, e := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if e != nil {
		http.NotFound(w, r)
		return
	}
	p, e := h.s.GetProduct(r.Context(), id)
	if e != nil {
		h.err(w, e)
		return
	}
	h.renderProductForm(w, r, productFormPage{Title: "Edit supplier product", ActiveNav: "supplier-products", Action: "/supplier-products/" + strconv.FormatInt(id, 10), Key: fmt.Sprintf("supplier-product-%d", time.Now().UnixNano()), Version: base64.RawURLEncoding.EncodeToString(p.RowVersion), Product: p, Editing: true})
}
func (h *Handler) updateProduct(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	id, e := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if e != nil {
		http.NotFound(w, r)
		return
	}
	current, e := h.s.GetProduct(r.Context(), id)
	if e != nil {
		h.err(w, e)
		return
	}
	cost, pe := money.Parse(r.FormValue("reference_cost"))
	if pe != nil {
		current.SupplierSKU = r.FormValue("supplier_sku")
		h.renderProductForm(w, r, productFormPage{Title: "Edit supplier product", ActiveNav: "supplier-products", Action: r.URL.Path, Key: r.FormValue("idempotency_key"), Version: r.FormValue("row_version"), Product: current, Errors: ValidationErrors{"ReferenceCost": pe.Error()}, Editing: true})
		return
	}
	v, _ := base64.RawURLEncoding.DecodeString(r.FormValue("row_version"))
	c := UpdateProductCommand{SupplierProduct: SupplierProduct{ID: id, SupplierID: current.SupplierID, ProductID: current.ProductID, SupplierSKU: r.FormValue("supplier_sku"), ReferenceCost: cost, IsActive: r.FormValue("is_active") == "on"}, OriginalVersion: v, RequestID: requestID(r), IdempotencyKey: r.FormValue("idempotency_key"), ActorID: "local-admin"}
	if _, e = h.s.UpdateProduct(r.Context(), c); e != nil {
		if errors.Is(e, ErrSupplierConflict) {
			http.Error(w, "This relationship was changed by another request. Reload and try again.", http.StatusConflict)
			return
		}
		h.err(w, e)
		return
	}
	http.Redirect(w, r, "/supplier-products", 303)
}
func (h *Handler) renderProductForm(w http.ResponseWriter, r *http.Request, page productFormPage) {
	if !page.Editing {
		suppliers, products, err := h.s.FormOptions(r.Context())
		if err != nil {
			h.err(w, err)
			return
		}
		page.Suppliers, page.Products = suppliers, products
		page.SupplierSelect = supplierSelect(suppliers, page.Product.SupplierID)
		page.ProductSelect = productSelect(products, page.Product.ProductID)
	}
	h.render(w, h.productForm, "layout", page)
}
func supplierSelect(items []Supplier, selected int64) webtemplates.SearchableSelectViewModel {
	options := make([]webtemplates.SearchableSelectOption, 0, len(items))
	for _, item := range items {
		options = append(options, webtemplates.SearchableSelectOption{Value: strconv.FormatInt(item.ID, 10), Label: item.Name, Search: strings.ToLower(item.Name + " " + item.ContactNumber)})
	}
	return webtemplates.SearchableSelectViewModel{ID: "supplier-select", Name: "supplier_id", Label: "Supplier", Placeholder: "Search supplier...", Options: options, Selected: strconv.FormatInt(selected, 10)}
}
func productSelect(items []ProductOption, selected int64) webtemplates.SearchableSelectViewModel {
	options := make([]webtemplates.SearchableSelectOption, 0, len(items))
	for _, item := range items {
		label := item.SKU + " - " + item.Name + " (" + item.UOM + ")"
		options = append(options, webtemplates.SearchableSelectOption{Value: strconv.FormatInt(item.ID, 10), Label: label, Search: strings.ToLower(label)})
	}
	return webtemplates.SearchableSelectViewModel{ID: "product-select", Name: "product_id", Label: "Product", Placeholder: "Search by SKU or product name...", Options: options, Selected: strconv.FormatInt(selected, 10)}
}
func parseID(value, field string) (int64, ValidationErrors) {
	id, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || id < 1 {
		return 0, ValidationErrors{field: "Select a valid option."}
	}
	return id, nil
}
func mergeErrors(all ...ValidationErrors) ValidationErrors {
	out := ValidationErrors{}
	for _, e := range all {
		for k, v := range e {
			out[k] = v
		}
	}
	return out
}
func (h *Handler) render(w http.ResponseWriter, t *template.Template, n string, d any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = t.ExecuteTemplate(w, n, d)
}
func (h *Handler) err(w http.ResponseWriter, e error) {
	log.Printf("suppliers request failed: %v", e)
	http.Error(w, "Internal server error", 500)
}
func requestID(r *http.Request) string {
	if x := r.Header.Get("X-Request-ID"); x != "" {
		return x
	}
	return fmt.Sprintf("request-%d", time.Now().UnixNano())
}
