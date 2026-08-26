package invoices

import (
	"bytes"
	"crypto/rand"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	sharedpdf "github.com/psbernardo/syncline-collection-tracking/internal/shared/pdf"
	"github.com/psbernardo/syncline-collection-tracking/internal/slices/salesorders"
	webtemplates "github.com/psbernardo/syncline-collection-tracking/internal/web/templates"
)

//go:embed templates/*.html
var files embed.FS

type Handler struct {
	orders    salesorders.Repository
	repo      Repository
	templates map[string]*template.Template
}

type conversionPage struct {
	Title, ActiveNav, Error, InvoiceNumber string
	Order                                  salesorders.SalesOrder
	IdempotencyKey                         string
}
type detailPage struct {
	Title, ActiveNav string
	Invoice          Invoice
}
type listPage struct {
	Title, ActiveNav string
	Invoices         []Invoice
}

func NewHandler(orders salesorders.Repository, repo Repository) *Handler {
	return &Handler{orders: orders, repo: repo, templates: make(map[string]*template.Template)}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /sales-orders/{id}/invoice/new", h.newConversion)
	mux.HandleFunc("POST /sales-orders/{id}/invoice", h.create)
	mux.HandleFunc("GET /invoices", h.list)
	mux.HandleFunc("GET /invoices/{id}", h.detail)
	mux.HandleFunc("GET /invoices/{id}/pdf", h.pdf)
}

func (h *Handler) newConversion(w http.ResponseWriter, r *http.Request) {
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
	if order.Status == salesorders.Converted {
		http.Error(w, "This sales order has already been converted", http.StatusConflict)
		return
	}
	if len(order.Quotation.Lines) == 0 {
		http.Error(w, "This sales order has no items to invoice", http.StatusConflict)
		return
	}
	key, err := newIdempotencyKey()
	if err != nil {
		http.Error(w, "Internal server error", 500)
		return
	}
	h.render(w, "form", conversionPage{Title: "Create invoice", ActiveNav: "sales-orders", Order: order, IdempotencyKey: key})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
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
	number, numberErr := ValidateInvoiceNumber(r.FormValue("invoice_number"))
	if numberErr != nil {
		h.render(w, "form", conversionPage{Title: "Create invoice", ActiveNav: "sales-orders", Order: order, Error: numberErr.Error(), InvoiceNumber: strings.TrimSpace(r.FormValue("invoice_number")), IdempotencyKey: r.FormValue("idempotency_key")})
		return
	}
	key := r.FormValue("idempotency_key")
	if key == "" {
		http.Error(w, "idempotency key is required", http.StatusBadRequest)
		return
	}
	invoice, err := h.repo.CreateFromSalesOrder(r.Context(), id, number, time.Now().UTC(), key, requestID(r), "local-admin")
	if err != nil {
		if errors.Is(err, ErrDuplicateInvoice) || errors.Is(err, ErrAlreadyInvoiced) || errors.Is(err, ErrInvoiceNotReady) || errors.Is(err, ErrIdempotencyConflict) {
			h.render(w, "form", conversionPage{Title: "Create invoice", ActiveNav: "sales-orders", Order: order, Error: err.Error(), InvoiceNumber: number, IdempotencyKey: key})
			return
		}
		http.Error(w, "Internal server error", 500)
		return
	}
	http.Redirect(w, r, "/invoices/"+strconv.FormatInt(invoice.ID, 10), http.StatusSeeOther)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.List(r.Context())
	if err != nil {
		http.Error(w, "Internal server error", 500)
		return
	}
	h.render(w, "list", listPage{Title: "Invoices", ActiveNav: "invoices", Invoices: items})
}
func (h *Handler) detail(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	item, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	h.render(w, "detail", detailPage{Title: "Invoice", ActiveNav: "invoices", Invoice: item})
}

func (h *Handler) pdf(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	invoice, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	document := InvoicePDFDocument{Seller: sharedpdf.DefaultSeller(), Number: invoice.Number, SalesOrderNumber: invoice.SalesOrderNumber, CustomerPO: invoice.CustomerPONumber, SalesPerson: invoice.SalesPerson, Customer: invoice.CustomerName, BillingAddress: invoice.BillingAddress, DeliveryAddress: invoice.DeliveryAddress, ContactPerson: invoice.ContactPerson, ContactNumber: invoice.ContactNumber, Email: invoice.Email, TermsDays: invoice.TermsDays, InvoiceDate: invoice.InvoiceDateUTC, DueDate: invoice.DueDateUTC, Subtotal: invoice.Subtotal, Tax: invoice.Tax, Total: invoice.Total}
	for _, line := range invoice.Lines {
		document.Lines = append(document.Lines, InvoicePDFLine{SKU: line.SKU, Name: line.Name, UOM: line.UOM, Quantity: line.Quantity, UnitPrice: line.UnitPrice, Amount: line.VATInclusiveTotal, TaxRate: line.TaxRate})
	}
	var output bytes.Buffer
	if err := NewInvoicePDFRenderer().Render(&output, document); err != nil {
		http.Error(w, "Unable to generate invoice PDF", 500)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="`+safeFilename(invoice.Number)+`"`)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(output.Bytes())
}

func (h *Handler) render(w http.ResponseWriter, name string, value any) {
	t, err := h.template(name)
	if err != nil {
		http.Error(w, "Internal server error", 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = t.ExecuteTemplate(w, "layout", value)
}
func (h *Handler) template(name string) (*template.Template, error) {
	if t := h.templates[name]; t != nil {
		return t, nil
	}
	t, err := template.New(name).ParseFS(files, "templates/"+name+".html")
	if err != nil {
		return nil, err
	}
	t, err = t.ParseFS(webtemplates.FS, "layout.html", "partials/*.html")
	if err != nil {
		return nil, err
	}
	h.templates[name] = t
	return t, nil
}
func parseID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}
func newIdempotencyKey() (string, error) {
	b := make([]byte, 24)
	_, err := rand.Read(b)
	return fmt.Sprintf("%x", b), err
}
func requestID(r *http.Request) string {
	if value := r.Header.Get("X-Request-ID"); value != "" {
		return value
	}
	return fmt.Sprintf("request-%d", time.Now().UnixNano())
}
func safeFilename(value string) string {
	var b strings.Builder
	for _, r := range value {
		if strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_", r) {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "invoice.pdf"
	}
	return b.String() + ".pdf"
}
