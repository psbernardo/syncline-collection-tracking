package products

import (
	"embed"
	"encoding/base64"
	"errors"
	"fmt"
	webtemplates "github.com/psbernardo/syncline-collection-tracking/internal/web/templates"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/uom"
)

//go:embed templates/*.html
var files embed.FS

type Handler struct {
	s          *Service
	list, form *template.Template
}
type listPage struct {
	Title, ActiveNav string
	Products         []Product
}
type formPage struct {
	Title, ActiveNav           string
	Product                    Product
	Errors                     ValidationErrors
	UOMOptions                 []uom.Code
	Mode, Action, Key, Version string
}

func NewHandler(s *Service) (*Handler, error) {
	f := template.FuncMap{}
	l, e := template.New("list").Funcs(f).ParseFS(webtemplates.FS, "layout.html", "partials/*.html")
	if e != nil {
		return nil, e
	}
	l, e = l.ParseFS(files, "templates/list.html")
	if e != nil {
		return nil, e
	}
	ft, e := template.New("form").ParseFS(webtemplates.FS, "layout.html", "partials/*.html")
	if e != nil {
		return nil, e
	}
	ft, e = ft.ParseFS(files, "templates/form.html")
	if e != nil {
		return nil, e
	}
	return &Handler{s, l, ft}, nil
}
func (h *Handler) RegisterRoutes(m *http.ServeMux) {
	m.HandleFunc("GET /products", h.listPage)
	m.HandleFunc("GET /products/new", h.newPage)
	m.HandleFunc("POST /products", h.create)
	m.HandleFunc("GET /products/{id}/edit", h.edit)
	m.HandleFunc("POST /products/{id}", h.update)
}
func (h *Handler) listPage(w http.ResponseWriter, r *http.Request) {
	p, e := h.s.List(r.Context(), r.URL.Query().Get("all") != "1")
	if e != nil {
		h.err(w, e)
		return
	}
	h.render(w, h.list, "layout", listPage{"Products", "products", p})
}
func (h *Handler) newPage(w http.ResponseWriter, r *http.Request) {
	k, _ := newKey()
	h.render(w, h.form, "layout", formPage{"New product", "products", Product{IsActive: true}, nil, uom.Options(), "new", "/products", k, ""})
}
func (h *Handler) edit(w http.ResponseWriter, r *http.Request) {
	id, e := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if e != nil {
		http.NotFound(w, r)
		return
	}
	p, e := h.s.Get(r.Context(), id)
	if errors.Is(e, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if e != nil {
		h.err(w, e)
		return
	}
	k, _ := newKey()
	h.render(w, h.form, "layout", formPage{"Edit product", "products", p, nil, uom.Options(), "edit", "/products/" + strconv.FormatInt(id, 10), k, base64.RawURLEncoding.EncodeToString(p.RowVersion)})
}
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	c := CreateCommand{r.FormValue("sku"), r.FormValue("name"), r.FormValue("description"), r.FormValue("uom"), requestID(r), r.FormValue("idempotency_key"), "local-admin"}
	if _, e := h.s.Create(r.Context(), c); e != nil {
		var v ValidationErrors
		if errors.As(e, &v) {
			h.render(w, h.form, "layout", formPage{"New product", "products", Product{SKU: c.SKU, Name: c.Name, Description: c.Description, UOM: c.UOM, IsActive: true}, v, uom.Options(), "new", "/products", c.IdempotencyKey, ""})
			return
		}
		h.err(w, e)
		return
	}
	http.Redirect(w, r, "/products", http.StatusSeeOther)
}
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	id, e := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if e != nil {
		http.NotFound(w, r)
		return
	}
	v, _ := base64.RawURLEncoding.DecodeString(r.FormValue("row_version"))
	c := UpdateCommand{Product: Product{ID: id, SKU: r.FormValue("sku"), Name: r.FormValue("name"), Description: r.FormValue("description"), UOM: r.FormValue("uom"), IsActive: r.FormValue("is_active") == "on"}, OriginalVersion: v, RequestID: requestID(r), IdempotencyKey: r.FormValue("idempotency_key"), ActorID: "local-admin"}
	if _, e = h.s.Update(r.Context(), c); e != nil {
		if errors.Is(e, ErrUOMLocked) {
			http.Error(w, "This product's UOM cannot change after it has been used in a transaction.", http.StatusConflict)
			return
		}
		var ve ValidationErrors
		if errors.As(e, &ve) {
			h.render(w, h.form, "layout", formPage{"Edit product", "products", c.Product, ve, uom.Options(), "edit", r.URL.Path, r.FormValue("idempotency_key"), r.FormValue("row_version")})
			return
		}
		h.err(w, e)
		return
	}
	http.Redirect(w, r, "/products", http.StatusSeeOther)
}
func (h *Handler) render(w http.ResponseWriter, t *template.Template, n string, d any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = t.ExecuteTemplate(w, n, d)
}
func (h *Handler) err(w http.ResponseWriter, e error) {
	log.Printf("products request failed: %v", e)
	http.Error(w, "Internal server error", 500)
}
func requestID(r *http.Request) string {
	if x := r.Header.Get("X-Request-ID"); x != "" {
		return x
	}
	return fmt.Sprintf("request-%d", time.Now().UnixNano())
}
func newKey() (string, error) { return fmt.Sprintf("product-%d", time.Now().UnixNano()), nil }
