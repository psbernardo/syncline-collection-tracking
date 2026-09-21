package receivables

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
	"github.com/xuri/excelize/v2"
)

type ExportRow struct {
	CompanyName        string
	InvoiceNumber      string
	PONumber           string
	GrossAmountDisplay string
	DeliveryDate       string
	DueDate            string
	PaymentTermDays    int
	Status             string
	PaymentDate        string
}

type ExportCompanyGroup struct {
	CompanyName   string
	Rows          []ExportRow
	RowCount      int
	GrossSubtotal money.Amount
}

type ExportReport struct {
	Companies      []ExportCompanyGroup
	Filters        ListQuery
	GeneratedAtUTC time.Time
	RowCount       int
	GrandGross     money.Amount
}

func buildExportReport(receivables []DeliveryReceivable, query ListQuery, now time.Time) ExportReport {
	report := ExportReport{Filters: query, GeneratedAtUTC: now}
	groups := make(map[int64]*exportCompanyBuilder)
	order := make([]int64, 0, len(receivables))
	for _, receivable := range receivables {
		group, ok := groups[receivable.CompanyAccountID]
		if !ok {
			group = &exportCompanyBuilder{name: receivable.CompanyName}
			groups[receivable.CompanyAccountID] = group
			order = append(order, receivable.CompanyAccountID)
		}
		group.receivables = append(group.receivables, receivable)
	}
	for _, companyID := range order {
		group := groups[companyID]
		company := ExportCompanyGroup{CompanyName: group.name, Rows: make([]ExportRow, 0, len(group.receivables))}
		for _, receivable := range group.receivables {
			row := exportRow(receivable, now)
			company.Rows = append(company.Rows, row)
			company.GrossSubtotal += receivable.GrossAmount
			report.GrandGross += receivable.GrossAmount
			report.RowCount++
		}
		company.RowCount = len(group.receivables)
		report.Companies = append(report.Companies, company)
	}
	return report
}

type exportCompanyBuilder struct {
	name        string
	receivables []DeliveryReceivable
}

func exportRow(receivable DeliveryReceivable, now time.Time) ExportRow {
	view := toViewModel(receivable, now)
	return ExportRow{
		CompanyName:        view.CompanyName,
		InvoiceNumber:      view.InvoiceNumber,
		PONumber:           view.PONumber,
		GrossAmountDisplay: view.GrossAmountDisplay,
		DeliveryDate:       view.DeliveryDate,
		DueDate:            view.DueDate,
		PaymentTermDays:    view.PaymentTermDays,
		Status:             view.Classification,
		PaymentDate:        view.PaymentDate,
	}
}

func filterSummary(report ExportReport) string {
	parts := make([]string, 0, 4)
	if len(report.Filters.CompanyAccountIDs) > 0 {
		names := make([]string, 0, len(report.Companies))
		for _, company := range report.Companies {
			names = append(names, company.CompanyName)
		}
		parts = append(parts, "Companies: "+strings.Join(names, ", "))
	}
	if report.Filters.Invoice != "" {
		parts = append(parts, fmt.Sprintf("Invoice contains %q", report.Filters.Invoice))
	}
	if report.Filters.PO != "" {
		parts = append(parts, fmt.Sprintf("PO starts with %q", report.Filters.PO))
	}
	if len(report.Filters.Statuses) > 0 {
		labels := make([]string, 0, len(report.Filters.Statuses))
		for _, status := range report.Filters.Statuses {
			labels = append(labels, exportStatusLabel(status))
		}
		parts = append(parts, "Status: "+strings.Join(labels, ", "))
	}
	if len(parts) == 0 {
		return "All delivery receivables"
	}
	return strings.Join(parts, "  ·  ")
}

func exportStatusLabel(value string) string {
	for _, option := range defaultStatusOptions() {
		if option.Value == value {
			return option.Label
		}
	}
	return value
}

func generatedAtDisplay(value time.Time) string {
	return value.Format("January 02, 2006 15:04")
}

const (
	xlsxTitleRow     = 1
	xlsxGeneratedRow = 2
	xlsxFilterRow    = 3
	xlsxHeaderRow    = 5
)

var xlsxColumns = []struct {
	title string
	width float64
}{
	{"Company", 28},
	{"Invoice #", 16},
	{"PO #", 18},
	{"Total Amount", 16},
	{"Delivery Date", 14},
	{"Due Date", 14},
	{"Term (Days)", 12},
	{"Status", 18},
	{"Payment Date", 14},
}

func writeReceivablesXLSX(w io.Writer, report ExportReport) error {
	file := excelize.NewFile()
	defer file.Close()
	sheet := "Receivables"
	if err := file.SetSheetName("Sheet1", sheet); err != nil {
		return fmt.Errorf("rename receivables sheet: %w", err)
	}
	styles, err := xlsxStyles(file)
	if err != nil {
		return err
	}
	lastColumn := len(xlsxColumns)
	span := func(row int) (string, string) {
		return xlsxCell(1, row), xlsxCell(lastColumn, row)
	}
	first, last := span(xlsxTitleRow)
	if err := file.MergeCell(sheet, first, last); err != nil {
		return fmt.Errorf("merge receivables title: %w", err)
	}
	if err := file.SetCellValue(sheet, xlsxCell(1, xlsxTitleRow), "DELIVERY RECEIVABLES REPORT"); err != nil {
		return fmt.Errorf("write receivables title: %w", err)
	}
	if err := file.SetCellStyle(sheet, first, last, styles.title); err != nil {
		return fmt.Errorf("style receivables title: %w", err)
	}
	first, last = span(xlsxGeneratedRow)
	if err := file.MergeCell(sheet, first, last); err != nil {
		return fmt.Errorf("merge receivables generated row: %w", err)
	}
	if err := file.SetCellValue(sheet, xlsxCell(1, xlsxGeneratedRow), "Generated: "+generatedAtDisplay(report.GeneratedAtUTC)); err != nil {
		return fmt.Errorf("write receivables generated row: %w", err)
	}
	if err := file.SetCellStyle(sheet, first, last, styles.meta); err != nil {
		return fmt.Errorf("style receivables generated row: %w", err)
	}
	first, last = span(xlsxFilterRow)
	if err := file.MergeCell(sheet, first, last); err != nil {
		return fmt.Errorf("merge receivables filter row: %w", err)
	}
	if err := file.SetCellValue(sheet, xlsxCell(1, xlsxFilterRow), "Filters: "+filterSummary(report)); err != nil {
		return fmt.Errorf("write receivables filter row: %w", err)
	}
	if err := file.SetCellStyle(sheet, first, last, styles.meta); err != nil {
		return fmt.Errorf("style receivables filter row: %w", err)
	}
	for index, column := range xlsxColumns {
		if err := file.SetCellValue(sheet, xlsxCell(index+1, xlsxHeaderRow), column.title); err != nil {
			return fmt.Errorf("write receivables header: %w", err)
		}
	}
	first, last = span(xlsxHeaderRow)
	if err := file.SetCellStyle(sheet, first, last, styles.header); err != nil {
		return fmt.Errorf("style receivables header: %w", err)
	}
	if err := file.SetRowHeight(sheet, xlsxHeaderRow, 22); err != nil {
		return fmt.Errorf("set receivables header height: %w", err)
	}
	row := xlsxHeaderRow + 1
	for _, company := range report.Companies {
		first, last = span(row)
		if err := file.MergeCell(sheet, first, last); err != nil {
			return fmt.Errorf("merge company band: %w", err)
		}
		if err := file.SetCellValue(sheet, xlsxCell(1, row), fmt.Sprintf("COMPANY: %s — %d receivable(s)", company.CompanyName, company.RowCount)); err != nil {
			return fmt.Errorf("write company band: %w", err)
		}
		if err := file.SetCellStyle(sheet, first, last, styles.company); err != nil {
			return fmt.Errorf("style company band: %w", err)
		}
		row++
		for _, receivable := range company.Rows {
			values := []string{receivable.CompanyName, receivable.InvoiceNumber, receivable.PONumber, receivable.GrossAmountDisplay, receivable.DeliveryDate, receivable.DueDate, fmt.Sprintf("%d", receivable.PaymentTermDays), receivable.Status, receivable.PaymentDate}
			for index, value := range values {
				if err := file.SetCellValue(sheet, xlsxCell(index+1, row), value); err != nil {
					return fmt.Errorf("write receivables row: %w", err)
				}
			}
			if err := file.SetCellStyle(sheet, xlsxCell(4, row), xlsxCell(4, row), styles.money); err != nil {
				return fmt.Errorf("style receivables total amount: %w", err)
			}
			row++
		}
		if err := file.SetCellValue(sheet, xlsxCell(1, row), fmt.Sprintf("Subtotal — %s", company.CompanyName)); err != nil {
			return fmt.Errorf("write company subtotal: %w", err)
		}
		if err := file.SetCellValue(sheet, xlsxCell(4, row), company.GrossSubtotal.FormatPHP()); err != nil {
			return fmt.Errorf("write company subtotal total amount: %w", err)
		}
		if err := file.SetCellValue(sheet, xlsxCell(5, row), fmt.Sprintf("%d receivable(s)", company.RowCount)); err != nil {
			return fmt.Errorf("write company subtotal count: %w", err)
		}
		first, last = span(row)
		if err := file.SetCellStyle(sheet, first, last, styles.subtotal); err != nil {
			return fmt.Errorf("style company subtotal: %w", err)
		}
		row += 2
	}
	if err := file.SetCellValue(sheet, xlsxCell(1, row), "GRAND TOTAL"); err != nil {
		return fmt.Errorf("write grand total label: %w", err)
	}
	if err := file.SetCellValue(sheet, xlsxCell(4, row), report.GrandGross.FormatPHP()); err != nil {
		return fmt.Errorf("write grand total total amount: %w", err)
	}
	if err := file.SetCellValue(sheet, xlsxCell(5, row), fmt.Sprintf("%d receivable(s)", report.RowCount)); err != nil {
		return fmt.Errorf("write grand total count: %w", err)
	}
	first, last = span(row)
	if err := file.SetCellStyle(sheet, first, last, styles.grandTotal); err != nil {
		return fmt.Errorf("style grand total: %w", err)
	}
	for index, column := range xlsxColumns {
		name, err := excelize.ColumnNumberToName(index + 1)
		if err != nil {
			return fmt.Errorf("resolve receivables column name: %w", err)
		}
		if err := file.SetColWidth(sheet, name, name, column.width); err != nil {
			return fmt.Errorf("set receivables column width: %w", err)
		}
	}
	output, err := file.WriteToBuffer()
	if err != nil {
		return fmt.Errorf("build receivables workbook: %w", err)
	}
	if _, err := w.Write(output.Bytes()); err != nil {
		return fmt.Errorf("write receivables workbook: %w", err)
	}
	return nil
}

type xlsxStyleIDs struct {
	title      int
	meta       int
	header     int
	company    int
	subtotal   int
	grandTotal int
	money      int
}

func xlsxStyles(file *excelize.File) (xlsxStyleIDs, error) {
	title, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 16, Color: purpleHex}})
	if err != nil {
		return xlsxStyleIDs{}, fmt.Errorf("create title style: %w", err)
	}
	meta, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Size: 10, Color: "403752"}, Alignment: &excelize.Alignment{WrapText: true}})
	if err != nil {
		return xlsxStyleIDs{}, fmt.Errorf("create meta style: %w", err)
	}
	header, err := file.NewStyle(&excelize.Style{Fill: solidFill(purpleHex), Font: &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11}, Alignment: &excelize.Alignment{Vertical: "center", Horizontal: "center"}})
	if err != nil {
		return xlsxStyleIDs{}, fmt.Errorf("create header style: %w", err)
	}
	company, err := file.NewStyle(&excelize.Style{Fill: solidFill(lightPurpleHex), Font: &excelize.Font{Bold: true, Color: purpleHex, Size: 11}})
	if err != nil {
		return xlsxStyleIDs{}, fmt.Errorf("create company style: %w", err)
	}
	subtotal, err := file.NewStyle(&excelize.Style{Fill: solidFill(lightPurpleHex), Font: &excelize.Font{Bold: true, Color: "17121F"}})
	if err != nil {
		return xlsxStyleIDs{}, fmt.Errorf("create subtotal style: %w", err)
	}
	grandTotal, err := file.NewStyle(&excelize.Style{Fill: solidFill(purpleHex), Font: &excelize.Font{Bold: true, Color: "FFFFFF"}})
	if err != nil {
		return xlsxStyleIDs{}, fmt.Errorf("create grand total style: %w", err)
	}
	money, err := file.NewStyle(&excelize.Style{Alignment: &excelize.Alignment{Horizontal: "right"}})
	if err != nil {
		return xlsxStyleIDs{}, fmt.Errorf("create money style: %w", err)
	}
	return xlsxStyleIDs{title: title, meta: meta, header: header, company: company, subtotal: subtotal, grandTotal: grandTotal, money: money}, nil
}

const (
	purpleHex      = "371473"
	lightPurpleHex = "BEAADC"
)

func solidFill(color string) excelize.Fill {
	return excelize.Fill{Type: "pattern", Color: []string{color}, Pattern: 1}
}

func xlsxCell(column, row int) string {
	name, err := excelize.CoordinatesToCellName(column, row)
	if err != nil {
		return ""
	}
	return name
}
