package receivables

import (
	"testing"
	"time"
)

func TestNormalizeListQuery(t *testing.T) {
	query := normalizeListQuery(ListQuery{
		CompanyAccountIDs: []int64{2, 2, 1},
		Invoice:           " 0220 ",
		PO:                " po-123 ",
		Statuses:          []string{"OVERDUE", "overdue", "invalid", " near_due "},
		PageSize:          999,
	})
	if len(query.CompanyAccountIDs) != 2 || query.CompanyAccountIDs[0] != 2 || query.CompanyAccountIDs[1] != 1 || query.Invoice != "0220" || query.PO != "PO-123" {
		t.Fatalf("normalized search values = %+v", query)
	}
	if len(query.Statuses) != 2 || query.Statuses[0] != "overdue" || query.Statuses[1] != "near_due" {
		t.Fatalf("normalized statuses = %#v", query.Statuses)
	}
	if query.PageSize != 25 {
		t.Fatalf("page size = %d, want 25", query.PageSize)
	}
}

func TestNormalizeListQueryPreservesStatusFilterPresence(t *testing.T) {
	query := normalizeListQuery(ListQuery{Statuses: []string{"not-a-status"}, StatusFilterProvided: true})
	if len(query.Statuses) != 0 {
		t.Fatalf("statuses = %#v, want no valid statuses", query.Statuses)
	}
	if !query.StatusFilterProvided {
		t.Fatal("status filter presence was lost")
	}
}

func TestCursorIsBoundToFilters(t *testing.T) {
	query := normalizeListQuery(ListQuery{CompanyAccountIDs: []int64{1}, Invoice: "0220", PO: "po-1", Statuses: []string{"overdue"}})
	receivable := DeliveryReceivable{ID: 1, DueDateUTC: query.Now}
	value := encodeCursor(receivable, query)
	decoded, err := decodeCursor(value)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Filter != filterSignature(query) {
		t.Fatalf("cursor filter = %q, want %q", decoded.Filter, filterSignature(query))
	}
	other := query
	other.CompanyAccountIDs = []int64{2}
	if decoded.Filter == filterSignature(other) {
		t.Fatal("cursor unexpectedly matches a different filter")
	}
	other = query
	other.Invoice = "0127"
	if decoded.Filter == filterSignature(other) {
		t.Fatal("cursor unexpectedly matches a different invoice filter")
	}
}

func TestNormalizeListQueryPreservesValidPageSize(t *testing.T) {
	query := normalizeListQuery(ListQuery{PageSize: 50, Now: time.Now()})
	if query.PageSize != 50 {
		t.Fatalf("page size = %d, want 50", query.PageSize)
	}
}

func TestNewDeliveryReceivableCalculatesDueDate(t *testing.T) {
	receivable, err := NewDeliveryReceivable(1, "0220", "PO123", "9800.3439", "2026-08-10", 5)
	if err != nil {
		t.Fatal(err)
	}
	if receivable.PONumber != "PO123" || receivable.AmountDue.Int64() != 98_003_439 {
		t.Fatalf("unexpected receivable: %+v", receivable)
	}
	if receivable.Classification(receivable.DueDateUTC) != "Near Due" {
		t.Fatalf("expected near-due classification at due date")
	}
}

func TestNewDeliveryReceivableRejectsInvalidInput(t *testing.T) {
	_, err := NewDeliveryReceivable(0, "", "PO-123", "-1", "bad-date", 121)
	validation, ok := err.(ValidationErrors)
	if !ok {
		t.Fatalf("error type = %T, want ValidationErrors", err)
	}
	for _, field := range []string{"CompanyAccountID", "InvoiceNumber", "PONumber", "Amount", "DeliveryDate", "PaymentTermDays"} {
		if validation[field] == "" {
			t.Errorf("missing validation error for %s", field)
		}
	}
}

func TestClassificationBoundaries(t *testing.T) {
	receivable, err := NewDeliveryReceivable(1, "0127", "PO123", "100", "2026-08-10", 5)
	if err != nil {
		t.Fatal(err)
	}
	if got := receivable.Classification(receivable.DeliveryDateUTC); got != "Near Due" {
		t.Fatalf("classification at delivery = %q, want Near Due", got)
	}
	if got := receivable.Classification(receivable.DueDateUTC.AddDate(0, 0, 1)); got != "Overdue" {
		t.Fatalf("classification after due date = %q, want Overdue", got)
	}
	if got := receivable.DaysUntilDueAt(receivable.DeliveryDateUTC); got != 4 {
		t.Fatalf("days until due = %d, want 4", got)
	}
	if got := receivable.DaysOverdueAt(receivable.DueDateUTC.AddDate(0, 0, 3)); got != 3 {
		t.Fatalf("days overdue = %d, want 3", got)
	}
}
