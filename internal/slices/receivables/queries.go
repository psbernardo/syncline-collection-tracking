package receivables

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/businessdate"
	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
	"github.com/psbernardo/syncline-collection-tracking/internal/shared/tax"
)

type ListQuery struct {
	CompanyAccountIDs []int64
	Invoice           string
	PO                string
	Statuses          []string
	// CompanyFilterProvided distinguishes an omitted company filter from one
	// containing only invalid values.
	CompanyFilterProvided bool
	// StatusFilterProvided distinguishes an omitted status filter from one
	// containing only invalid values.
	StatusFilterProvided bool
	Cursor               string
	PageSize             int
	Now                  time.Time
}

func normalizeListQuery(query ListQuery) ListQuery {
	query.CompanyAccountIDs = normalizeCompanyAccountIDs(query.CompanyAccountIDs)
	query.Invoice = strings.TrimSpace(query.Invoice)
	query.PO = NormalizePONumber(query.PO)
	query.Statuses = normalizeStatuses(query.Statuses)
	if query.PageSize < 1 || query.PageSize > 100 {
		query.PageSize = 25
	}
	return query
}

func normalizeCompanyAccountIDs(values []int64) []int64 {
	seen := make(map[int64]bool)
	result := make([]int64, 0, len(values))
	for _, value := range values {
		if value > 0 && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result
}

type ListResult struct {
	Items          []ReceivableViewModel
	NextCursor     string
	RemainingCount int64
}

type cursor struct {
	DueDate string `json:"due_date"`
	ID      int64  `json:"id"`
	Filter  string `json:"filter"`
}

func encodeCursor(receivable DeliveryReceivable, query ListQuery) string {
	payload, _ := json.Marshal(cursor{DueDate: businessdate.FormatUTC(receivable.DueDateUTC), ID: receivable.ID, Filter: filterSignature(query)})
	return base64.RawURLEncoding.EncodeToString(payload)
}

func decodeCursor(value string) (cursor, error) {
	if value == "" {
		return cursor{}, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return cursor{}, fmt.Errorf("invalid cursor")
	}
	var result cursor
	if err := json.Unmarshal(decoded, &result); err != nil || result.DueDate == "" || result.ID < 1 || result.Filter == "" {
		return cursor{}, fmt.Errorf("invalid cursor")
	}
	return result, nil
}

func filterSignature(query ListQuery) string {
	payload, _ := json.Marshal(struct {
		CompanyAccountIDs []int64  `json:"company_account_ids"`
		Invoice           string   `json:"invoice"`
		PO                string   `json:"po"`
		Statuses          []string `json:"statuses"`
	}{CompanyAccountIDs: query.CompanyAccountIDs, Invoice: query.Invoice, PO: query.PO, Statuses: query.Statuses})
	return base64.RawURLEncoding.EncodeToString(payload)
}

func normalizeStatuses(values []string) []string {
	allowed := map[string]bool{"pending": true, "near_due": true, "overdue": true, "payment_received": true, "cancelled": true, "archived": true}
	seen := make(map[string]bool)
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if allowed[value] && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result
}

func parsePageSize(value string) int {
	pageSize, err := strconv.Atoi(value)
	if err != nil || pageSize < 1 || pageSize > 100 {
		return 25
	}
	return pageSize
}

type ReceivableViewModel struct {
	ID                 int64
	CompanyAccountID   int64
	CompanyName        string
	InvoiceID          int64
	InvoiceNumber      string
	PONumber           string
	AmountDisplay      string
	GrossAmountDisplay string
	TaxBaseDisplay     string
	VATDisplay         string
	EWTDisplay         string
	NetPayableDisplay  string
	TaxRuleLabel       string
	DeliveryDate       string
	DueDate            string
	PaymentTermDays    int
	PaymentDate        string
	LifecycleStatus    string
	Classification     string
	ClassificationTone string
	DaysOverdue        int
	DaysUntilDue       int
	CanReceivePayment  bool
	CanReversePayment  bool
	CanEdit            bool
	RowVersion         string
}

type AccountOption struct {
	ID          int64
	CompanyName string
}

type ReceivableFormViewModel struct {
	Mode           string
	Action         string
	SubmitLabel    string
	PageTitle      string
	ReceivableID   int64
	RowVersion     string
	Values         CreateReceivableCommand
	Errors         ValidationErrors
	Accounts       []AccountOption
	Invoices       []InvoiceOption
	TaxRules       []tax.RuleOption
	IdempotencyKey string
	TaxPreview     TaxPreviewViewModel
}

type TaxPreviewViewModel struct {
	Visible            bool
	GrossAmountDisplay string
	TaxBaseDisplay     string
	VATDisplay         string
	EWTDisplay         string
	NetPayableDisplay  string
	Error              string
}

func taxPreview(amountInput string, code tax.RuleCode) TaxPreviewViewModel {
	if code == tax.RuleNone {
		return TaxPreviewViewModel{}
	}
	amount, err := money.Parse(amountInput)
	if err != nil {
		return TaxPreviewViewModel{}
	}
	breakdown, err := tax.CalculateRule(amount, code)
	if err != nil {
		return TaxPreviewViewModel{Error: "Enter a valid amount and tax rule."}
	}
	return TaxPreviewViewModel{
		Visible:            true,
		GrossAmountDisplay: breakdown.GrossAmount.FormatPHP(),
		TaxBaseDisplay:     breakdown.TaxBase.FormatPHP(),
		VATDisplay:         breakdown.VATAmount.FormatPHP(),
		EWTDisplay:         breakdown.WithholdingAmount.FormatPHP(),
		NetPayableDisplay:  breakdown.NetAmount.FormatPHP(),
	}
}

type PaymentFormViewModel struct {
	Action         string
	ReceivableID   int64
	PaymentDate    string
	AmountDisplay  string
	TaxRuleLabel   string
	RowVersion     string
	IdempotencyKey string
	Errors         ValidationErrors
}

type ReversePaymentFormViewModel struct {
	Action         string
	ReceivableID   int64
	InvoiceNumber  string
	PaymentDate    string
	RowVersion     string
	Reason         string
	IdempotencyKey string
	Open           bool
	Errors         ValidationErrors
}

func toViewModel(receivable DeliveryReceivable, now time.Time) ReceivableViewModel {
	classification := receivable.ClassificationAt(now)
	return ReceivableViewModel{
		ID: receivable.ID, CompanyAccountID: receivable.CompanyAccountID, CompanyName: receivable.CompanyName, InvoiceID: receivable.InvoiceID, InvoiceNumber: receivable.InvoiceNumber, PONumber: receivable.PONumber,
		AmountDisplay: receivable.AmountDue.FormatPHP(), DeliveryDate: businessdate.FormatUTC(receivable.DeliveryDateUTC),
		GrossAmountDisplay: receivable.GrossAmount.FormatPHP(), TaxBaseDisplay: receivable.TaxBase.FormatPHP(), VATDisplay: receivable.VATAmount.FormatPHP(), EWTDisplay: receivable.EWTAmount.FormatPHP(), NetPayableDisplay: receivable.AmountDue.FormatPHP(), TaxRuleLabel: taxRuleLabel(receivable.TaxRuleCode),
		DueDate: businessdate.FormatUTC(receivable.DueDateUTC), PaymentTermDays: receivable.PaymentTermDays,
		PaymentDate: paymentDateDisplay(receivable.PaymentDateUTC), LifecycleStatus: receivable.LifecycleStatus, Classification: string(classification),
		ClassificationTone: classificationTone(classification), CanReceivePayment: receivable.LifecycleStatus == "Active" && receivable.PaymentDateUTC == nil,
		CanReversePayment: receivable.LifecycleStatus == "Active" && receivable.PaymentDateUTC != nil,
		CanEdit:           receivable.LifecycleStatus == "Active" && receivable.PaymentDateUTC == nil,
		RowVersion:        base64.RawURLEncoding.EncodeToString(receivable.RowVersion),
		DaysOverdue:       receivable.DaysOverdueAt(now), DaysUntilDue: receivable.DaysUntilDueAt(now),
	}
}

func taxRuleLabel(code tax.RuleCode) string {
	if code == tax.RuleNone {
		return ""
	}
	for _, option := range tax.RuleOptions() {
		if option.Code == code {
			return option.Label
		}
	}
	return ""
}

func paymentDateDisplay(value *time.Time) string {
	if value == nil {
		return ""
	}
	return businessdate.FormatUTC(*value)
}

func classificationTone(classification Classification) string {
	switch classification {
	case ClassificationNearDue:
		return "near-due"
	case ClassificationOverdue:
		return "overdue"
	case ClassificationPaymentReceived:
		return "payment-received"
	case ClassificationCancelled:
		return "cancelled"
	case ClassificationArchived:
		return "archived"
	default:
		return "pending"
	}
}
