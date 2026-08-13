package receivables

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/businessdate"
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
	InvoiceNumber      string
	PONumber           string
	AmountDisplay      string
	DeliveryDate       string
	DueDate            string
	PaymentTermDays    int
	LifecycleStatus    string
	Classification     string
	ClassificationTone string
	DaysOverdue        int
	DaysUntilDue       int
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
	IdempotencyKey string
}

func toViewModel(receivable DeliveryReceivable, now time.Time) ReceivableViewModel {
	return ReceivableViewModel{
		ID: receivable.ID, CompanyAccountID: receivable.CompanyAccountID, InvoiceNumber: receivable.InvoiceNumber, PONumber: receivable.PONumber,
		CompanyName:   receivable.CompanyName,
		AmountDisplay: receivable.AmountDue.FormatPHP(), DeliveryDate: businessdate.FormatUTC(receivable.DeliveryDateUTC),
		DueDate: businessdate.FormatUTC(receivable.DueDateUTC), PaymentTermDays: receivable.PaymentTermDays,
		LifecycleStatus: receivable.LifecycleStatus, Classification: receivable.Classification(now),
		ClassificationTone: classificationTone(receivable.ClassificationAt(now)),
		DaysOverdue:        receivable.DaysOverdueAt(now), DaysUntilDue: receivable.DaysUntilDueAt(now),
	}
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
