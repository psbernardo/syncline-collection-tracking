package receivables

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
	"github.com/xuri/excelize/v2"
)

func TestBuildExportReportGroupsByCompanyWithSubtotals(t *testing.T) {
	now := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	receivables := []DeliveryReceivable{
		{ID: 1, CompanyAccountID: 1, CompanyName: "Acme Corp", InvoiceNumber: "INV1", PONumber: "PO1", DeliveryDateUTC: time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC), DueDateUTC: time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC), PaymentTermDays: 5, AmountDue: 1000, GrossAmount: 2000, LifecycleStatus: "Active"},
		{ID: 2, CompanyAccountID: 1, CompanyName: "Acme Corp", InvoiceNumber: "INV2", PONumber: "PO2", DeliveryDateUTC: time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC), DueDateUTC: time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC), PaymentTermDays: 5, AmountDue: 2000, GrossAmount: 4000, LifecycleStatus: "Active"},
		{ID: 3, CompanyAccountID: 2, CompanyName: "Beta Inc", InvoiceNumber: "INV3", PONumber: "PO3", DeliveryDateUTC: time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC), DueDateUTC: time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC), PaymentTermDays: 5, AmountDue: 3000, GrossAmount: 6000, LifecycleStatus: "Active"},
	}
	report := buildExportReport(receivables, ListQuery{Now: now}, now)
	if report.RowCount != 3 || report.GeneratedAtUTC != now {
		t.Fatalf("unexpected report header: rows=%d generated=%v", report.RowCount, report.GeneratedAtUTC)
	}
	if len(report.Companies) != 2 {
		t.Fatalf("company groups = %d, want 2", len(report.Companies))
	}
	acme := report.Companies[0]
	if acme.CompanyName != "Acme Corp" || acme.RowCount != 2 {
		t.Fatalf("unexpected first group: %+v", acme)
	}
	if acme.GrossSubtotal != money.Amount(6000) {
		t.Fatalf("acme gross subtotal = %d, want 6000", acme.GrossSubtotal)
	}
	if len(acme.Rows) != 2 || acme.Rows[0].Status != "Overdue" {
		t.Fatalf("unexpected acme rows: %+v", acme.Rows)
	}
	beta := report.Companies[1]
	if beta.CompanyName != "Beta Inc" || beta.RowCount != 1 {
		t.Fatalf("unexpected second group: %+v", beta)
	}
	if report.GrandGross != money.Amount(12000) {
		t.Fatalf("grand gross = %d, want 12000", report.GrandGross)
	}
}

func TestBuildExportReportPreservesCompanyOrderAndEmptyReport(t *testing.T) {
	now := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	empty := buildExportReport(nil, ListQuery{Now: now}, now)
	if empty.RowCount != 0 || len(empty.Companies) != 0 || empty.GrandGross != 0 {
		t.Fatalf("unexpected empty report: %+v", empty)
	}
	report := buildExportReport([]DeliveryReceivable{
		{ID: 1, CompanyAccountID: 2, CompanyName: "Beta Inc", LifecycleStatus: "Active"},
		{ID: 2, CompanyAccountID: 1, CompanyName: "Acme Corp", LifecycleStatus: "Active"},
	}, ListQuery{Now: now}, now)
	if report.Companies[0].CompanyName != "Beta Inc" || report.Companies[1].CompanyName != "Acme Corp" {
		t.Fatalf("company order was not preserved: %+v", report.Companies)
	}
}

func TestFilterSummaryDescribesAppliedFilters(t *testing.T) {
	report := ExportReport{
		Filters:   ListQuery{CompanyAccountIDs: []int64{1}, Invoice: "0220", PO: "PO-1", Statuses: []string{"overdue", "near_due"}},
		Companies: []ExportCompanyGroup{{CompanyName: "Acme Corp"}},
	}
	summary := filterSummary(report)
	for _, expected := range []string{"Companies: Acme Corp", `Invoice contains "0220"`, `PO starts with "PO-1"`, "Status: Overdue, Near due"} {
		if !strings.Contains(summary, expected) {
			t.Fatalf("filter summary %q does not contain %q", summary, expected)
		}
	}
	if got := filterSummary(ExportReport{}); got != "All delivery receivables" {
		t.Fatalf("unfiltered summary = %q, want %q", got, "All delivery receivables")
	}
}

func TestWriteReceivablesXLSXProducesOpenableWorkbook(t *testing.T) {
	now := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	report := buildExportReport([]DeliveryReceivable{
		{ID: 1, CompanyAccountID: 1, CompanyName: "Acme Corp", InvoiceNumber: "INV1", PONumber: "PO1", DeliveryDateUTC: time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC), DueDateUTC: time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC), PaymentTermDays: 5, AmountDue: 1000, GrossAmount: 2000, LifecycleStatus: "Active"},
		{ID: 2, CompanyAccountID: 1, CompanyName: "Acme Corp", InvoiceNumber: "INV2", PONumber: "PO2", DeliveryDateUTC: time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC), DueDateUTC: time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC), PaymentTermDays: 5, AmountDue: 2000, GrossAmount: 4000, LifecycleStatus: "Active"},
	}, ListQuery{Now: now}, now)
	var output bytes.Buffer
	if err := writeReceivablesXLSX(&output, report); err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(output.Bytes(), []byte("PK")) {
		t.Fatal("workbook output is not a zip/xlsx file")
	}
	file, err := excelize.OpenReader(bytes.NewReader(output.Bytes()))
	if err != nil {
		t.Fatalf("open workbook: %v", err)
	}
	defer file.Close()
	sheet := file.GetSheetName(0)
	rows, err := file.GetRows(sheet)
	if err != nil {
		t.Fatalf("read workbook rows: %v", err)
	}
	if len(rows) < 5 {
		t.Fatalf("workbook has only %d rows", len(rows))
	}
	if rows[0][0] != "DELIVERY RECEIVABLES REPORT" {
		t.Fatalf("title cell = %q", rows[0][0])
	}
	headerHints := []string{"Company", "Invoice #", "PO #", "Total Amount", "Delivery Date", "Due Date", "Term (Days)", "Status", "Payment Date"}
	for index, hint := range headerHints {
		if !strings.Contains(rows[4][index], hint) {
			t.Fatalf("header cell %d = %q, want it to contain %q", index, rows[4][index], hint)
		}
	}
	var grandTotal []string
	for index := range rows {
		if len(rows[index]) > 0 && rows[index][0] == "GRAND TOTAL" {
			grandTotal = rows[index]
			break
		}
	}
	if grandTotal == nil {
		t.Fatal("grand total row was not written")
	}
	if len(grandTotal) < 5 || grandTotal[3] == "" || grandTotal[4] == "" {
		t.Fatalf("grand total row incomplete: %+v", grandTotal)
	}
}
