package supplierpayments

import (
	"testing"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/businessdate"
	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
)

func TestNewPaymentValidatesRequiredFieldsAndDates(t *testing.T) {
	_, err := NewPayment(0, " ", "bad", "", "0")
	validation, ok := err.(ValidationErrors)
	if !ok {
		t.Fatalf("error = %T, want ValidationErrors", err)
	}
	for _, field := range []string{"SupplierID", "CheckNumber", "IssueDate", "DueDate", "Amount"} {
		if validation[field] == "" {
			t.Errorf("missing validation error for %s", field)
		}
	}

	_, err = NewPayment(3, "001-ABC", "2026-05-12", "2026-05-11", "25.50")
	validation, ok = err.(ValidationErrors)
	if !ok || validation["DueDate"] == "" {
		t.Fatalf("earlier due date error = %#v", err)
	}
}

func TestNewPaymentTrimsNumberAndAcceptsPositiveAmount(t *testing.T) {
	payment, err := NewPayment(9, " 0000123 ", "2026-05-12", "2026-05-12", "12.3456")
	if err != nil {
		t.Fatalf("NewPayment() error = %v", err)
	}
	if payment.CheckNumber != "0000123" {
		t.Errorf("check number = %q", payment.CheckNumber)
	}
	if payment.Amount != money.Amount(123456) {
		t.Errorf("amount = %d, want 123456", payment.Amount)
	}
	if businessdate.FormatUTC(payment.IssueDateUTC) != "2026-05-12" || businessdate.FormatUTC(payment.DueDateUTC) != "2026-05-12" {
		t.Errorf("dates = %s / %s", businessdate.FormatUTC(payment.IssueDateUTC), businessdate.FormatUTC(payment.DueDateUTC))
	}
}

func TestBuildListResultDueWindowBoundariesAndVoidedRecords(t *testing.T) {
	today, err := businessdate.Parse("2026-05-12")
	if err != nil {
		t.Fatal(err)
	}
	dateAt := func(days int) time.Time {
		local := today.In(time.FixedZone("Asia/Manila", 8*60*60)).AddDate(0, 0, days)
		parsed, parseErr := businessdate.Parse(local.Format("2006-01-02"))
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		return parsed
	}
	amount := func(value string) money.Amount {
		parsed, parseErr := money.Parse(value)
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		return parsed
	}
	payments := []Payment{
		{ID: 1, SupplierID: 1, SupplierName: "Supplier", CheckNumber: "TODAY", IssueDateUTC: today, DueDateUTC: dateAt(0), Amount: amount("10"), Status: "Active"},
		{ID: 2, SupplierID: 1, SupplierName: "Supplier", CheckNumber: "FIVE", IssueDateUTC: today, DueDateUTC: dateAt(5), Amount: amount("20"), Status: "Active"},
		{ID: 3, SupplierID: 1, SupplierName: "Supplier", CheckNumber: "SIX", IssueDateUTC: today, DueDateUTC: dateAt(6), Amount: amount("30"), Status: "Active"},
		{ID: 4, SupplierID: 1, SupplierName: "Supplier", CheckNumber: "FOURTEEN", IssueDateUTC: today, DueDateUTC: dateAt(14), Amount: amount("40"), Status: "Active"},
		{ID: 5, SupplierID: 1, SupplierName: "Supplier", CheckNumber: "FIFTEEN", IssueDateUTC: today, DueDateUTC: dateAt(15), Amount: amount("50"), Status: "Active"},
		{ID: 6, SupplierID: 1, SupplierName: "Supplier", CheckNumber: "OVERDUE", IssueDateUTC: dateAt(-2), DueDateUTC: dateAt(-1), Amount: amount("60"), Status: "Active"},
		{ID: 7, SupplierID: 1, SupplierName: "Supplier", CheckNumber: "VOID", IssueDateUTC: today, DueDateUTC: dateAt(1), Amount: amount("70"), Status: "Voided"},
	}

	result := buildListResult(payments, today, false)
	if result.Summary.NearDue != amount("100") {
		t.Errorf("near-due total = %s, want 100", result.Summary.NearDue.Format())
	}
	if result.Summary.WithinFiveDays != amount("30") {
		t.Errorf("within-five-days total = %s, want 30", result.Summary.WithinFiveDays.Format())
	}
	if result.Summary.DaysSixToFourteen != amount("70") {
		t.Errorf("days-six-to-fourteen total = %s, want 70", result.Summary.DaysSixToFourteen.Format())
	}
	if len(result.Items) != 6 {
		t.Errorf("active list length = %d, want 6", len(result.Items))
	}
	if result.Items[0].DueLabel != "Due today" || result.Items[5].DueLabel != "Overdue by 1 day(s)" {
		t.Errorf("due labels = %q ... %q", result.Items[0].DueLabel, result.Items[5].DueLabel)
	}
	withVoided := buildListResult(payments, today, true)
	if len(withVoided.Items) != 7 || withVoided.Summary.NearDue != amount("100") {
		t.Errorf("show-voided result items=%d near-due=%s", len(withVoided.Items), withVoided.Summary.NearDue.Format())
	}
}
