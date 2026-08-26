package dashboard

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"

	webtemplates "github.com/psbernardo/syncline-collection-tracking/internal/web/templates"
)

//go:embed templates/*.html templates/partials/*.html
var templateFiles embed.FS

type Handler struct {
	service       *Service
	template      *template.Template
	salesTemplate *template.Template
}

type page struct {
	Title     string
	ActiveNav string
	Totals    DashboardViewModel
}

type salesPage struct {
	Title, ActiveNav string
	Sales            SalesDashboardViewModel
}

func NewHandler(service *Service) (*Handler, error) {
	parsed, err := template.New("dashboard").ParseFS(webtemplates.FS, "layout.html", "partials/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse dashboard layout: %w", err)
	}
	parsed, err = parsed.ParseFS(templateFiles, "templates/dashboard.html", "templates/partials/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse dashboard templates: %w", err)
	}
	salesParsed, err := template.New("sales-dashboard").ParseFS(webtemplates.FS, "layout.html", "partials/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse sales dashboard layout: %w", err)
	}
	salesParsed, err = salesParsed.ParseFS(templateFiles, "templates/sales-dashboard.html")
	if err != nil {
		return nil, fmt.Errorf("parse sales dashboard template: %w", err)
	}
	return &Handler{service: service, template: parsed, salesTemplate: salesParsed}, nil
}

func (handler *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /dashboard", handler.dashboard)
	mux.HandleFunc("GET /dashboard/summary", handler.summary)
}

func (handler *Handler) RegisterSalesRoutes(mux *http.ServeMux, service *SalesService) {
	mux.HandleFunc("GET /dashboard/sales", func(w http.ResponseWriter, r *http.Request) {
		data, err := service.Dashboard(r.Context())
		if err != nil {
			handler.serverError(w, err)
			return
		}
		handler.renderSales(w, salesPage{Title: "Sales dashboard", ActiveNav: "sales-dashboard", Sales: data})
	})
}

func (handler *Handler) renderSales(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = handler.salesTemplate.ExecuteTemplate(w, "layout", data)
}

func (handler *Handler) dashboard(w http.ResponseWriter, r *http.Request) {
	totals, err := handler.service.Totals(r.Context())
	if err != nil {
		handler.serverError(w, err)
		return
	}
	handler.render(w, "layout", page{Title: "Collection dashboard", ActiveNav: "collection-dashboard", Totals: totals})
}

func (handler *Handler) summary(w http.ResponseWriter, r *http.Request) {
	totals, err := handler.service.Totals(r.Context())
	if err != nil {
		handler.serverError(w, err)
		return
	}
	handler.render(w, "dashboard-data", totals)
}

func (handler *Handler) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = handler.template.ExecuteTemplate(w, name, data)
}

func (handler *Handler) serverError(w http.ResponseWriter, err error) {
	log.Printf("dashboard request failed: %v", err)
	w.Header().Set("X-Feedback-Message", "The dashboard totals could not be loaded.")
	http.Error(w, "Internal server error", http.StatusInternalServerError)
}
