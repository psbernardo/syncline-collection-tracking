package accounts

import (
	"embed"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/slices/dashboard"
	webtemplates "github.com/psbernardo/syncline-collection-tracking/internal/web/templates"
)

//go:embed templates/*.html templates/partials/*.html
var templateFiles embed.FS

type Handler struct {
	service       *commandService
	companyTotals *dashboard.CompanyTotalsService
	listTemplate  *template.Template
	formTemplate  *template.Template
}

type listPage struct {
	Title     string
	ActiveNav string
	Accounts  []AccountViewModel
}

func NewHandler(service *commandService, companyTotals ...*dashboard.CompanyTotalsService) (*Handler, error) {
	functions := template.FuncMap{"dict": dict}
	listTemplate, err := template.New("list").Funcs(functions).ParseFS(webtemplates.FS, "layout.html", "partials/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse account list templates: %w", err)
	}
	listTemplate, err = listTemplate.ParseFS(templateFiles, "templates/list.html", "templates/partials/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse account list page templates: %w", err)
	}
	formTemplate, err := template.New("form").Funcs(functions).ParseFS(webtemplates.FS, "layout.html", "partials/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse account form templates: %w", err)
	}
	formTemplate, err = formTemplate.ParseFS(templateFiles, "templates/form.html", "templates/partials/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse account form page templates: %w", err)
	}
	var totalsService *dashboard.CompanyTotalsService
	if len(companyTotals) > 0 {
		totalsService = companyTotals[0]
	}
	return &Handler{service: service, companyTotals: totalsService, listTemplate: listTemplate, formTemplate: formTemplate}, nil
}

func (handler *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /accounts", handler.list)
	mux.HandleFunc("GET /accounts/new", handler.newForm)
	mux.HandleFunc("POST /accounts", handler.create)
	mux.HandleFunc("GET /accounts/{id}/edit", handler.editForm)
	mux.HandleFunc("POST /accounts/{id}", handler.update)
}

func (handler *Handler) list(w http.ResponseWriter, r *http.Request) {
	accounts, err := handler.service.List(r.Context())
	if err != nil {
		handler.serverError(w, err)
		return
	}
	handler.render(w, handler.listTemplate, "layout", listPage{Title: "Company accounts", ActiveNav: "accounts", Accounts: accounts})
}

func (handler *Handler) newForm(w http.ResponseWriter, r *http.Request) {
	form, err := NewAccountForm()
	if err != nil {
		handler.serverError(w, err)
		return
	}
	handler.render(w, handler.formTemplate, "layout", formPage{Title: form.PageTitle, ActiveNav: "accounts", Form: form})
}

func (handler *Handler) editForm(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		http.NotFound(w, r)
		return
	}
	account, err := handler.service.Get(r.Context(), id)
	if errors.Is(err, ErrAccountNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		handler.serverError(w, err)
		return
	}
	key, err := NewIdempotencyKey()
	if err != nil {
		handler.serverError(w, err)
		return
	}
	form := AccountFormViewModel{
		Mode: "edit", Action: "/accounts/" + strconv.FormatInt(id, 10), SubmitLabel: "Save changes",
		PageTitle: "Edit company account", AccountID: id, Values: account,
		IdempotencyKey: key, RowVersion: base64.RawURLEncoding.EncodeToString(account.RowVersion),
	}
	var totals *dashboard.CompanyTotalsViewModel
	if handler.companyTotals != nil {
		companyTotals, err := handler.companyTotals.Totals(r.Context(), id)
		if err != nil {
			handler.serverError(w, err)
			return
		}
		totals = &companyTotals
	}
	handler.render(w, handler.formTemplate, "layout", formPage{Title: form.PageTitle, ActiveNav: "accounts", Form: form, CompanyTotals: totals})
}

func (handler *Handler) create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		handler.serverError(w, err)
		return
	}
	command := CreateAccountCommand{
		CompanyName:     r.FormValue("company_name"),
		ContactPerson:   r.FormValue("contact_person"),
		TINNumber:       r.FormValue("tin_number"),
		BillingAddress:  r.FormValue("billing_address"),
		DeliveryAddress: r.FormValue("delivery_address"),
		ContactNumber:   r.FormValue("contact_number"),
		RequestID:       requestID(r),
		IdempotencyKey:  r.FormValue("idempotency_key"),
		ActorID:         "local-admin",
	}
	if _, err := handler.service.Create(r.Context(), command); err != nil {
		var validation ValidationErrors
		if errors.As(err, &validation) {
			form := AccountFormViewModel{
				Values: CompanyAccount{
					CompanyName:     command.CompanyName,
					ContactPerson:   command.ContactPerson,
					TINNumber:       command.TINNumber,
					BillingAddress:  command.BillingAddress,
					DeliveryAddress: command.DeliveryAddress,
					ContactNumber:   command.ContactNumber,
				},
				Errors:         validation,
				IdempotencyKey: command.IdempotencyKey,
			}
			if form.IdempotencyKey == "" {
				form, _ = NewAccountForm()
				form.Values = CompanyAccount{
					CompanyName: command.CompanyName, ContactPerson: command.ContactPerson,
					TINNumber: command.TINNumber, BillingAddress: command.BillingAddress,
					DeliveryAddress: command.DeliveryAddress, ContactNumber: command.ContactNumber,
				}
				form.Errors = validation
			}
			if isHTMX(r) {
				w.Header().Set("X-Feedback-Message", "Please correct the highlighted company account fields.")
				w.WriteHeader(http.StatusUnprocessableEntity)
				handler.render(w, handler.formTemplate, "account-form", form)
				return
			}
			handler.render(w, handler.formTemplate, "layout", formPage{Title: "New company account", ActiveNav: "accounts", Form: form})
			return
		}
		handler.serverError(w, err)
		return
	}
	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/accounts")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, "/accounts", http.StatusSeeOther)
}

func (handler *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		handler.serverError(w, err)
		return
	}
	version, err := base64.RawURLEncoding.DecodeString(r.FormValue("row_version"))
	if err != nil {
		http.Error(w, "Invalid edit version", http.StatusBadRequest)
		return
	}
	command := UpdateAccountCommand{
		ID: id, CompanyName: r.FormValue("company_name"), ContactPerson: r.FormValue("contact_person"),
		TINNumber: r.FormValue("tin_number"), BillingAddress: r.FormValue("billing_address"),
		DeliveryAddress: r.FormValue("delivery_address"), ContactNumber: r.FormValue("contact_number"),
		OriginalVersion: version, RequestID: requestID(r), IdempotencyKey: r.FormValue("idempotency_key"), ActorID: "local-admin",
	}
	updated, err := handler.service.Update(r.Context(), command)
	if err != nil {
		var validation ValidationErrors
		if errors.As(err, &validation) {
			handler.renderUpdateValidation(w, r, command, validation, version)
			return
		}
		if errors.Is(err, ErrAccountNotFound) {
			http.NotFound(w, r)
			return
		}
		if errors.Is(err, ErrAccountConflict) {
			http.Error(w, "This account was changed by another request. Reload and try again.", http.StatusConflict)
			return
		}
		handler.serverError(w, err)
		return
	}
	_ = updated
	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/accounts")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, "/accounts", http.StatusSeeOther)
}

func (handler *Handler) renderUpdateValidation(w http.ResponseWriter, r *http.Request, command UpdateAccountCommand, validation ValidationErrors, version []byte) {
	form := AccountFormViewModel{
		Mode: "edit", Action: "/accounts/" + strconv.FormatInt(command.ID, 10), SubmitLabel: "Save changes",
		PageTitle: "Edit company account", AccountID: command.ID,
		Values: CompanyAccount{ID: command.ID, CompanyName: command.CompanyName, ContactPerson: command.ContactPerson, TINNumber: command.TINNumber, BillingAddress: command.BillingAddress, DeliveryAddress: command.DeliveryAddress, ContactNumber: command.ContactNumber},
		Errors: validation, IdempotencyKey: command.IdempotencyKey, RowVersion: base64.RawURLEncoding.EncodeToString(version),
	}
	if isHTMX(r) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		handler.render(w, handler.formTemplate, "account-form", form)
		return
	}
	handler.render(w, handler.formTemplate, "layout", formPage{Title: form.PageTitle, ActiveNav: "accounts", Form: form})
}

type formPage struct {
	Title         string
	ActiveNav     string
	Form          AccountFormViewModel
	CompanyTotals *dashboard.CompanyTotalsViewModel
}

func (handler *Handler) render(w http.ResponseWriter, parsed *template.Template, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := parsed.ExecuteTemplate(w, name, data); err != nil {
		return
	}
}

func (handler *Handler) serverError(w http.ResponseWriter, err error) {
	log.Printf("accounts request failed: %v", err)
	w.Header().Set("X-Feedback-Message", "The company account request could not be completed.")
	http.Error(w, "Internal server error", http.StatusInternalServerError)
}

func requestID(r *http.Request) string {
	if value := r.Header.Get("X-Request-ID"); value != "" {
		return value
	}
	return fmt.Sprintf("request-%d", time.Now().UnixNano())
}

func isHTMX(r *http.Request) bool { return r.Header.Get("HX-Request") == "true" }

func dict(values ...any) (map[string]any, error) {
	if len(values)%2 != 0 {
		return nil, fmt.Errorf("dict requires key/value pairs")
	}
	result := make(map[string]any, len(values)/2)
	for index := 0; index < len(values); index += 2 {
		key, ok := values[index].(string)
		if !ok {
			return nil, fmt.Errorf("dict key must be a string")
		}
		result[key] = values[index+1]
	}
	return result, nil
}
