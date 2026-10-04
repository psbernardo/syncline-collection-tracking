package supplierpayments

import (
	"context"
	"embed"
	"errors"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/businessdate"
	webtemplates "github.com/psbernardo/syncline-collection-tracking/internal/web/templates"
)

//go:embed templates/*.html
var templateFiles embed.FS

type Application interface {
	List(context.Context, bool) (ListResult, error)
	Get(context.Context, int64) (Payment, error)
	Suppliers(context.Context) ([]SupplierOption, error)
	Create(context.Context, Input) (Payment, error)
	Update(context.Context, int64, Input, []byte) (Payment, error)
	Void(context.Context, int64, []byte) error
}

type Handler struct {
	service   Application
	templates map[string]*template.Template
}

type listPage struct {
	Title, ActiveNav string
	Result           ListResult
	IncludeVoided    bool
}

type formValues struct {
	ID          int64
	SupplierID  int64
	CheckNumber string
	IssueDate   string
	DueDate     string
	Amount      string
	RowVersion  string
}

type formPage struct {
	Title, ActiveNav, Error string
	Edit                    bool
	Values                  formValues
	Suppliers               []SupplierOption
}

func NewHandler(service Application) (*Handler, error) {
	templates := make(map[string]*template.Template)
	for _, name := range []string{"list", "form"} {
		t, err := template.New(name).ParseFS(webtemplates.FS, "layout.html", "partials/*.html")
		if err != nil {
			return nil, err
		}
		t, err = t.ParseFS(templateFiles, "templates/"+name+".html")
		if err != nil {
			return nil, err
		}
		templates[name] = t
	}
	return &Handler{service: service, templates: templates}, nil
}

func (handler *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /supplier-payments", handler.list)
	mux.HandleFunc("GET /supplier-payments/new", handler.newForm)
	mux.HandleFunc("POST /supplier-payments", handler.create)
	mux.HandleFunc("GET /supplier-payments/{id}/edit", handler.editForm)
	mux.HandleFunc("POST /supplier-payments/{id}", handler.update)
	mux.HandleFunc("POST /supplier-payments/{id}/delete", handler.delete)
}

func (handler *Handler) list(w http.ResponseWriter, r *http.Request) {
	includeVoided := r.URL.Query().Get("show_voided") == "1"
	result, err := handler.service.List(r.Context(), includeVoided)
	if err != nil {
		http.Error(w, "Unable to load supplier payments", http.StatusInternalServerError)
		return
	}
	page := listPage{Title: "Supplier payments", ActiveNav: "supplier-payments", Result: result, IncludeVoided: includeVoided}
	if err := handler.templates["list"].ExecuteTemplate(w, "layout", page); err != nil {
		http.Error(w, "Unable to render supplier payments", http.StatusInternalServerError)
	}
}

func (handler *Handler) newForm(w http.ResponseWriter, r *http.Request) {
	suppliers, err := handler.service.Suppliers(r.Context())
	if err != nil {
		http.Error(w, "Unable to load suppliers", http.StatusInternalServerError)
		return
	}
	issueDate := businessdate.FormatUTC(time.Now().UTC())
	handler.renderForm(w, http.StatusOK, formPage{
		Title: "Add supplier payment", ActiveNav: "supplier-payments", Suppliers: suppliers,
		Values: formValues{IssueDate: issueDate},
	})
}

func (handler *Handler) editForm(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	payment, err := handler.service.Get(r.Context(), id)
	if err != nil || payment.Status != "Active" {
		http.NotFound(w, r)
		return
	}
	suppliers, err := handler.service.Suppliers(r.Context())
	if err != nil {
		http.Error(w, "Unable to load suppliers", http.StatusInternalServerError)
		return
	}
	if !hasSupplier(suppliers, payment.SupplierID) {
		suppliers = append(suppliers, SupplierOption{ID: payment.SupplierID, Name: payment.SupplierName})
	}
	handler.renderForm(w, http.StatusOK, formPage{
		Title: "Edit supplier payment", ActiveNav: "supplier-payments", Edit: true, Suppliers: suppliers,
		Values: formValues{SupplierID: payment.SupplierID, CheckNumber: payment.CheckNumber,
			IssueDate: businessdate.FormatUTC(payment.IssueDateUTC), DueDate: businessdate.FormatUTC(payment.DueDateUTC),
			Amount: payment.Amount.Format(), RowVersion: encodeVersion(payment.RowVersion), ID: payment.ID},
	})
}

func (handler *Handler) create(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	input := inputFromRequest(r)
	_, err := handler.service.Create(r.Context(), input)
	if err != nil {
		handler.renderInputError(w, r, input, false, err)
		return
	}
	http.Redirect(w, r, "/supplier-payments", http.StatusSeeOther)
}

func (handler *Handler) update(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	id, ok := pathID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	input := inputFromRequest(r)
	version, err := decodeVersion(r.FormValue("row_version"))
	if err != nil || len(version) == 0 {
		handler.renderInputError(w, r, input, true, ErrConflict)
		return
	}
	if _, err := handler.service.Update(r.Context(), id, input, version); err != nil {
		handler.renderInputError(w, r, input, true, err)
		return
	}
	http.Redirect(w, r, "/supplier-payments", http.StatusSeeOther)
}

func (handler *Handler) delete(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	id, ok := pathID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	version, err := decodeVersion(r.FormValue("row_version"))
	if err != nil || len(version) == 0 {
		http.Error(w, ErrConflict.Error(), http.StatusConflict)
		return
	}
	if err := handler.service.Void(r.Context(), id, version); err != nil {
		status := http.StatusConflict
		if errors.Is(err, ErrNotFound) {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}
	http.Redirect(w, r, "/supplier-payments", http.StatusSeeOther)
}

func (handler *Handler) renderInputError(w http.ResponseWriter, r *http.Request, input Input, edit bool, err error) {
	suppliers, loadErr := handler.service.Suppliers(r.Context())
	if loadErr != nil {
		http.Error(w, "Unable to load suppliers", http.StatusInternalServerError)
		return
	}
	if edit {
		current, getErr := handler.service.Get(r.Context(), parseIDValue(r.PathValue("id")))
		if getErr == nil && current.SupplierID == input.SupplierID && !hasSupplier(suppliers, current.SupplierID) {
			suppliers = append(suppliers, SupplierOption{ID: current.SupplierID, Name: current.SupplierName})
		}
	}
	version := r.FormValue("row_version")
	pageTitle := "Add supplier payment"
	if edit {
		pageTitle = "Edit supplier payment"
	}
	handler.renderForm(w, http.StatusUnprocessableEntity, formPage{
		Title: pageTitle, ActiveNav: "supplier-payments", Edit: edit, Suppliers: suppliers,
		Error: errorMessage(err), Values: formValues{SupplierID: input.SupplierID, CheckNumber: input.CheckNumber,
			IssueDate: input.IssueDate, DueDate: input.DueDate, Amount: input.Amount, RowVersion: version, ID: parseIDValue(r.PathValue("id"))},
	})
}

func (handler *Handler) renderForm(w http.ResponseWriter, status int, page formPage) {
	w.WriteHeader(status)
	if err := handler.templates["form"].ExecuteTemplate(w, "layout", page); err != nil {
		return
	}
}

func inputFromRequest(r *http.Request) Input {
	supplierID, _ := strconv.ParseInt(r.FormValue("supplier_id"), 10, 64)
	return Input{SupplierID: supplierID, CheckNumber: r.FormValue("check_number"), IssueDate: r.FormValue("issue_date"), DueDate: r.FormValue("due_date"), Amount: r.FormValue("amount")}
}

func errorMessage(err error) string {
	var validation ValidationErrors
	if errors.As(err, &validation) {
		messages := make([]string, 0, len(validation))
		for _, field := range []string{"SupplierID", "CheckNumber", "IssueDate", "DueDate", "Amount"} {
			if message := validation[field]; message != "" {
				messages = append(messages, message)
			}
		}
		return strings.Join(messages, " ")
	}
	switch {
	case errors.Is(err, ErrSupplierNotFound):
		return "Select an active supplier."
	case errors.Is(err, ErrDuplicateCheck):
		return "That check number is already in use. Check numbers must be unique across all suppliers."
	case errors.Is(err, ErrConflict):
		return "This record changed since you opened it. Reload the page and try again."
	default:
		return err.Error()
	}
}

func hasSupplier(suppliers []SupplierOption, id int64) bool {
	for _, supplier := range suppliers {
		if supplier.ID == id {
			return true
		}
	}
	return false
}

func pathID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	return id, err == nil && id > 0
}

func parseIDValue(value string) int64 {
	id, _ := strconv.ParseInt(value, 10, 64)
	return id
}
