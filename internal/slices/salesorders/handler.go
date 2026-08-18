package salesorders

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/psbernardo/syncline-collection-tracking/internal/slices/quotations"
)

type Handler struct {
	orders     Repository
	quotations quotations.Repository
	renderer   *quotations.QuotationPDFRenderer
}

func NewHandler(orders Repository, quotationRepository quotations.Repository) *Handler {
	return &Handler{orders: orders, quotations: quotationRepository, renderer: quotations.NewDocumentPDFRenderer("SALES ORDER")}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /quotations/{id}/sales-order/new", h.newPage)
	mux.HandleFunc("POST /quotations/{id}/sales-order", h.create)
	mux.HandleFunc("GET /sales-orders/{id}/pdf", h.pdf)
}

func (h *Handler) newPage(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		http.NotFound(w, r)
		return
	}
	q, err := h.quotations.FindByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	tmpl := template.Must(template.New("sales-order").Parse(`<!doctype html><html><head><meta charset="utf-8"><title>Create Sales Order</title></head><body><h1>Create Sales Order</h1><p>Quotation: {{.Number}}</p><form method="post" action="/quotations/{{.ID}}/sales-order"><label>Customer PO number <input name="customer_po_number" maxlength="100"></label><button type="submit">Create Sales Order</button></form></body></html>`))
	_ = tmpl.Execute(w, q)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		http.NotFound(w, r)
		return
	}
	q, err := h.quotations.FindByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	order, err := h.orders.CreateFromQuotation(r.Context(), q, strings.TrimSpace(r.FormValue("customer_po_number")))
	if err != nil {
		http.Error(w, "Unable to create sales order", http.StatusUnprocessableEntity)
		return
	}
	http.Redirect(w, r, "/sales-orders/"+strconv.FormatInt(order.ID, 10)+"/pdf", http.StatusSeeOther)
}

func (h *Handler) pdf(w http.ResponseWriter, r *http.Request) {
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
	var output bytes.Buffer
	renderer := *h.renderer
	renderer.Metadata = []quotations.PDFMetadata{
		{Label: "ORDER DATE", Value: order.CreatedAtUTC.Format("January 02 2006")},
		{Label: "REF #", Value: order.QuotationNumber},
		{Label: "SALES PERSON", Value: order.SalesPerson},
		{Label: "PO NUMBER", Value: order.CustomerPONumber},
		{Label: "PAYMENT TERMS #", Value: fmt.Sprintf("Net %d", order.Quotation.TermsDays)},
	}
	if err := renderer.Render(&output, order.Quotation); err != nil {
		http.Error(w, "Unable to generate sales order PDF", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="`+safeFilename(order.Number)+`"`)
	w.Header().Set("X-Content-Type-Options", "nosniff")
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
	return fmt.Sprintf("%s.pdf", b.String())
}
