package receivables

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

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/businessdate"
	webtemplates "github.com/psbernardo/syncline-collection-tracking/internal/web/templates"
)

//go:embed templates/*.html templates/partials/*.html
var templateFiles embed.FS

type Handler struct {
	service        *service
	listTemplate   *template.Template
	formTemplate   *template.Template
	detailTemplate *template.Template
}

type listPage struct {
	Title          string
	ActiveNav      string
	Receivables    []ReceivableViewModel
	NextCursor     string
	RemainingCount int64
	Filters        ListQuery
	CompanyFilter  webtemplates.MultiSelectViewModel
	StatusFilter   webtemplates.MultiSelectViewModel
}

type statusOption struct{ Value, Label string }

type formPage struct {
	Title     string
	ActiveNav string
	Form      ReceivableFormViewModel
}

func NewHandler(service *service) (*Handler, error) {
	functions := template.FuncMap{"dict": func(values ...any) (map[string]any, error) {
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
	}, "hasStatus": func(statuses []string, value string) bool {
		for _, status := range statuses {
			if status == value {
				return true
			}
		}
		return false
	}}
	listTemplate, err := template.New("list").Funcs(functions).ParseFS(webtemplates.FS, "layout.html", "partials/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse receivable list templates: %w", err)
	}
	listTemplate, err = listTemplate.ParseFS(templateFiles, "templates/list.html", "templates/partials/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse receivable list page templates: %w", err)
	}
	formTemplate, err := template.New("form").Funcs(functions).ParseFS(webtemplates.FS, "layout.html", "partials/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse receivable form templates: %w", err)
	}
	formTemplate, err = formTemplate.ParseFS(templateFiles, "templates/form.html", "templates/partials/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse receivable form page templates: %w", err)
	}
	detailTemplate, err := template.New("detail").Funcs(functions).ParseFS(webtemplates.FS, "layout.html", "partials/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse receivable detail templates: %w", err)
	}
	detailTemplate, err = detailTemplate.ParseFS(templateFiles, "templates/detail.html", "templates/partials/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse receivable detail page templates: %w", err)
	}
	return &Handler{service: service, listTemplate: listTemplate, formTemplate: formTemplate, detailTemplate: detailTemplate}, nil
}

func (handler *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /receivables", handler.list)
	mux.HandleFunc("GET /receivables/new", handler.newForm)
	mux.HandleFunc("POST /receivables", handler.create)
	mux.HandleFunc("GET /receivables/{id}", handler.detail)
	mux.HandleFunc("GET /receivables/{id}/edit", handler.editForm)
	mux.HandleFunc("POST /receivables/{id}", handler.update)
}

func (handler *Handler) list(w http.ResponseWriter, r *http.Request) {
	query := parseListQuery(r)
	accounts, err := handler.service.Accounts(r.Context())
	if err != nil {
		handler.serverError(w, err)
		return
	}
	result, err := handler.service.ListFiltered(r.Context(), query)
	if err != nil {
		handler.serverError(w, err)
		return
	}
	page := listPage{Title: "Receivables", ActiveNav: "receivables", Receivables: result.Items, NextCursor: result.NextCursor, RemainingCount: result.RemainingCount, Filters: query, CompanyFilter: companyFilter(accounts, query.CompanyAccountIDs), StatusFilter: statusFilter(query.Statuses)}
	if isHTMX(r) {
		handler.render(w, handler.listTemplate, "receivable-results", page)
		return
	}
	handler.render(w, handler.listTemplate, "layout", page)
}

func parseListQuery(r *http.Request) ListQuery {
	companyValues := r.URL.Query()["company"]
	companyIDs := make([]int64, 0, len(companyValues))
	for _, value := range companyValues {
		companyID, err := strconv.ParseInt(value, 10, 64)
		if err == nil {
			companyIDs = append(companyIDs, companyID)
		}
	}
	statusValues := r.URL.Query()["status"]
	query := ListQuery{CompanyAccountIDs: companyIDs, CompanyFilterProvided: len(companyValues) > 0, Invoice: r.URL.Query().Get("invoice"), PO: r.URL.Query().Get("po"), Statuses: statusValues, StatusFilterProvided: len(statusValues) > 0, Cursor: r.URL.Query().Get("cursor"), PageSize: parsePageSize(r.URL.Query().Get("page_size")), Now: time.Now().UTC()}
	return normalizeListQuery(query)
}

func companyFilter(accounts []AccountOption, selected []int64) webtemplates.MultiSelectViewModel {
	viewModel := webtemplates.MultiSelectViewModel{ID: "receivable-company-filter", Name: "company", Label: "Company", Placeholder: "Select companies", Options: make([]webtemplates.MultiSelectOption, 0, len(accounts))}
	for _, account := range accounts {
		viewModel.Options = append(viewModel.Options, webtemplates.MultiSelectOption{Value: strconv.FormatInt(account.ID, 10), Label: account.CompanyName})
	}
	for _, companyID := range selected {
		viewModel.Selected = append(viewModel.Selected, strconv.FormatInt(companyID, 10))
	}
	return viewModel
}

func defaultStatusOptions() []statusOption {
	return []statusOption{{"pending", "Pending"}, {"near_due", "Near due"}, {"overdue", "Overdue"}, {"payment_received", "Payment received"}, {"cancelled", "Cancelled"}, {"archived", "Archived"}}
}

func statusFilter(selected []string) webtemplates.MultiSelectViewModel {
	options := defaultStatusOptions()
	viewModel := webtemplates.MultiSelectViewModel{
		ID:          "receivable-status-filter",
		Name:        "status",
		Label:       "Status",
		Placeholder: "Select statuses",
		Selected:    append([]string(nil), selected...),
		Options:     make([]webtemplates.MultiSelectOption, 0, len(options)),
	}
	for _, option := range options {
		viewModel.Options = append(viewModel.Options, webtemplates.MultiSelectOption{Value: option.Value, Label: option.Label})
	}
	return viewModel
}

func (handler *Handler) newForm(w http.ResponseWriter, r *http.Request) {
	accounts, err := handler.service.Accounts(r.Context())
	if err != nil {
		handler.serverError(w, err)
		return
	}
	key, err := NewIdempotencyKey()
	if err != nil {
		handler.serverError(w, err)
		return
	}
	form := ReceivableFormViewModel{Mode: "create", Action: "/receivables", SubmitLabel: "Save receivable", PageTitle: "New delivery receivable", Accounts: accounts, IdempotencyKey: key}
	handler.render(w, handler.formTemplate, "layout", formPage{Title: "New delivery receivable", ActiveNav: "receivables", Form: form})
}

func (handler *Handler) create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		handler.serverError(w, err)
		return
	}
	companyID, _ := strconv.ParseInt(r.FormValue("company_account_id"), 10, 64)
	term, _ := strconv.Atoi(r.FormValue("payment_term_days"))
	command := CreateReceivableCommand{
		CompanyAccountID: companyID, InvoiceNumber: r.FormValue("invoice_number"), PONumber: r.FormValue("po_number"), AmountInput: r.FormValue("amount"),
		DeliveryDate: r.FormValue("delivery_date"), PaymentTermDays: term, RequestID: requestID(r),
		IdempotencyKey: r.FormValue("idempotency_key"), ActorID: "local-admin",
	}
	if _, err := handler.service.Create(r.Context(), command); err != nil {
		var validation ValidationErrors
		if errors.As(err, &validation) || errors.Is(err, ErrCompanyNotFound) || errors.Is(err, ErrDuplicatePO) || errors.Is(err, ErrDuplicateInvoiceNumber) {
			if validation == nil {
				validation = ValidationErrors{}
				if errors.Is(err, ErrCompanyNotFound) {
					validation["CompanyAccountID"] = "Select an existing company account."
				} else if errors.Is(err, ErrDuplicatePO) {
					validation["PONumber"] = "PO number is already used by a non-cancelled receivable."
				} else {
					validation["InvoiceNumber"] = "Invoice number is already used by a non-cancelled receivable."
				}
			}
			accounts, listErr := handler.service.Accounts(r.Context())
			if listErr != nil {
				handler.serverError(w, listErr)
				return
			}
			form := ReceivableFormViewModel{Mode: "create", Action: "/receivables", SubmitLabel: "Save receivable", PageTitle: "New delivery receivable", Values: command, Errors: validation, Accounts: accounts, IdempotencyKey: command.IdempotencyKey}
			if isHTMX(r) {
				if errors.Is(err, ErrDuplicatePO) {
					w.Header().Set("X-Feedback-Message", "PO number is already used by a non-cancelled receivable.")
				} else if errors.Is(err, ErrDuplicateInvoiceNumber) {
					w.Header().Set("X-Feedback-Message", "Invoice number is already used by a non-cancelled receivable.")
				} else {
					w.Header().Set("X-Feedback-Message", "Please correct the highlighted receivable fields.")
				}
				w.WriteHeader(http.StatusUnprocessableEntity)
				handler.render(w, handler.formTemplate, "receivable-form", form)
				return
			}
			handler.render(w, handler.formTemplate, "layout", formPage{Title: "New delivery receivable", ActiveNav: "receivables", Form: form})
			return
		}
		handler.serverError(w, err)
		return
	}
	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/receivables")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, "/receivables", http.StatusSeeOther)
}

func (handler *Handler) editForm(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		http.NotFound(w, r)
		return
	}
	receivable, err := handler.service.GetEntity(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		handler.serverError(w, err)
		return
	}
	if receivable.LifecycleStatus != "Active" {
		http.Error(w, "This receivable cannot be edited in its current state.", http.StatusConflict)
		return
	}
	accounts, err := handler.service.Accounts(r.Context())
	if err != nil {
		handler.serverError(w, err)
		return
	}
	key, err := NewIdempotencyKey()
	if err != nil {
		handler.serverError(w, err)
		return
	}
	form := ReceivableFormViewModel{
		Mode: "edit", Action: "/receivables/" + strconv.FormatInt(id, 10), SubmitLabel: "Save changes", PageTitle: "Edit delivery receivable",
		ReceivableID: id, RowVersion: base64.RawURLEncoding.EncodeToString(receivable.RowVersion), Accounts: accounts,
		Values:         CreateReceivableCommand{CompanyAccountID: receivable.CompanyAccountID, InvoiceNumber: receivable.InvoiceNumber, PONumber: receivable.PONumber, AmountInput: receivable.AmountDue.Format(), DeliveryDate: businessdate.FormatUTC(receivable.DeliveryDateUTC), PaymentTermDays: receivable.PaymentTermDays},
		IdempotencyKey: key,
	}
	handler.render(w, handler.formTemplate, "layout", formPage{Title: form.PageTitle, ActiveNav: "receivables", Form: form})
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
	companyID, _ := strconv.ParseInt(r.FormValue("company_account_id"), 10, 64)
	term, _ := strconv.Atoi(r.FormValue("payment_term_days"))
	command := UpdateReceivableCommand{ID: id, CompanyAccountID: companyID, InvoiceNumber: r.FormValue("invoice_number"), PONumber: r.FormValue("po_number"), AmountInput: r.FormValue("amount"), DeliveryDate: r.FormValue("delivery_date"), PaymentTermDays: term, OriginalVersion: version, RequestID: requestID(r), IdempotencyKey: r.FormValue("idempotency_key"), ActorID: "local-admin"}
	if _, err := handler.service.Update(r.Context(), command); err != nil {
		var validation ValidationErrors
		if errors.As(err, &validation) || errors.Is(err, ErrCompanyNotFound) || errors.Is(err, ErrDuplicatePO) || errors.Is(err, ErrDuplicateInvoiceNumber) {
			if validation == nil {
				validation = ValidationErrors{}
				if errors.Is(err, ErrCompanyNotFound) {
					validation["CompanyAccountID"] = "Select an existing company account."
				} else if errors.Is(err, ErrDuplicatePO) {
					validation["PONumber"] = "PO number is already used by a non-cancelled receivable."
				} else {
					validation["InvoiceNumber"] = "Invoice number is already used by a non-cancelled receivable."
				}
			}
			handler.renderUpdateValidation(w, r, command, validation, version)
			return
		}
		if errors.Is(err, ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		if errors.Is(err, ErrConflict) || errors.Is(err, ErrProtected) {
			w.Header().Set("X-Feedback-Message", "This receivable cannot be updated in its current state.")
			http.Error(w, "This receivable cannot be updated in its current state. Reload and try again.", http.StatusConflict)
			return
		}
		handler.serverError(w, err)
		return
	}
	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/receivables/"+strconv.FormatInt(id, 10))
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, "/receivables/"+strconv.FormatInt(id, 10), http.StatusSeeOther)
}

func (handler *Handler) renderUpdateValidation(w http.ResponseWriter, r *http.Request, command UpdateReceivableCommand, validation ValidationErrors, version []byte) {
	accounts, err := handler.service.Accounts(r.Context())
	if err != nil {
		handler.serverError(w, err)
		return
	}
	form := ReceivableFormViewModel{Mode: "edit", Action: "/receivables/" + strconv.FormatInt(command.ID, 10), SubmitLabel: "Save changes", PageTitle: "Edit delivery receivable", ReceivableID: command.ID, RowVersion: base64.RawURLEncoding.EncodeToString(version), Values: CreateReceivableCommand{CompanyAccountID: command.CompanyAccountID, InvoiceNumber: command.InvoiceNumber, PONumber: command.PONumber, AmountInput: command.AmountInput, DeliveryDate: command.DeliveryDate, PaymentTermDays: command.PaymentTermDays}, Errors: validation, Accounts: accounts, IdempotencyKey: command.IdempotencyKey}
	if isHTMX(r) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		handler.render(w, handler.formTemplate, "receivable-form", form)
		return
	}
	handler.render(w, handler.formTemplate, "layout", formPage{Title: form.PageTitle, ActiveNav: "receivables", Form: form})
}

func (handler *Handler) detail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		http.NotFound(w, r)
		return
	}
	receivable, err := handler.service.Get(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		handler.serverError(w, err)
		return
	}
	handler.render(w, handler.detailTemplate, "layout", struct {
		Title      string
		ActiveNav  string
		Receivable ReceivableViewModel
	}{Title: "Receivable detail", ActiveNav: "receivables", Receivable: receivable})
}

func (handler *Handler) render(w http.ResponseWriter, parsed *template.Template, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = parsed.ExecuteTemplate(w, name, data)
}

func (handler *Handler) serverError(w http.ResponseWriter, err error) {
	log.Printf("receivables request failed: %v", err)
	w.Header().Set("X-Feedback-Message", "The receivable request could not be completed.")
	http.Error(w, "Internal server error", http.StatusInternalServerError)
}

func requestID(r *http.Request) string {
	if value := r.Header.Get("X-Request-ID"); value != "" {
		return value
	}
	return fmt.Sprintf("request-%d", time.Now().UnixNano())
}

func isHTMX(r *http.Request) bool { return r.Header.Get("HX-Request") == "true" }
