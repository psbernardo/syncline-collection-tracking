package receivables

import (
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
)

func TestApplyFiltersDefaultsToActiveRecords(t *testing.T) {
	db := dryRunSQLServer(t)
	query := db.Table("dbo.delivery_receivables AS r").Joins("INNER JOIN dbo.company_accounts AS a ON a.company_account_id = r.company_account_id")
	statement := applyFilters(query, ListQuery{}, time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC), time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC)).Find(&[]receivableModel{}).Statement

	if !strings.Contains(statement.SQL.String(), "r.lifecycle_status = @") {
		t.Fatalf("default filter SQL = %s, want active lifecycle predicate", statement.SQL.String())
	}
	if len(statement.Vars) != 1 || statement.Vars[0] != "Active" {
		t.Fatalf("default filter vars = %#v, want [Active]", statement.Vars)
	}
}

func TestApplyFiltersInvalidOnlyStatusReturnsNoRows(t *testing.T) {
	db := dryRunSQLServer(t)
	statement := applyFilters(db.Table("dbo.delivery_receivables AS r"), ListQuery{StatusFilterProvided: true}, time.Time{}, time.Time{}).Find(&[]receivableModel{}).Statement

	if !strings.Contains(statement.SQL.String(), "1 = 0") {
		t.Fatalf("invalid status SQL = %s, want no-row predicate", statement.SQL.String())
	}
}

func TestApplyFiltersAddsCompanyAccountPredicate(t *testing.T) {
	db := dryRunSQLServer(t)
	query := db.Table("dbo.delivery_receivables AS r")
	statement := applyFilters(query, ListQuery{CompanyAccountIDs: []int64{2, 5}}, time.Time{}, time.Time{}).Find(&[]receivableModel{}).Statement
	if !strings.Contains(statement.SQL.String(), "r.company_account_id IN") {
		t.Fatalf("company filter SQL = %s, want company predicate", statement.SQL.String())
	}
	if len(statement.Vars) != 3 || statement.Vars[0] != int64(2) || statement.Vars[1] != int64(5) || statement.Vars[2] != "Active" {
		t.Fatalf("company filter vars = %#v, want [2 5 Active]", statement.Vars)
	}
}

func TestApplyFiltersEmptyProvidedCompanyFilterReturnsNoRows(t *testing.T) {
	db := dryRunSQLServer(t)
	statement := applyFilters(db.Table("dbo.delivery_receivables AS r"), ListQuery{CompanyFilterProvided: true}, time.Time{}, time.Time{}).Find(&[]receivableModel{}).Statement
	if !strings.Contains(statement.SQL.String(), "1 = 0") {
		t.Fatalf("empty company filter SQL = %s, want no-row predicate", statement.SQL.String())
	}
}

func TestMarkPaymentReceivedUsesProtectedPredicates(t *testing.T) {
	db := dryRunSQLServer(t)
	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Table("dbo.delivery_receivables").
			Where("delivery_receivable_id = ? AND row_version = ? AND lifecycle_status = ? AND payment_date_utc IS NULL", int64(7), []byte("version"), "Active").
			Updates(map[string]interface{}{"payment_date_utc": time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC), "updated_at_utc": gorm.Expr("SYSUTCDATETIME()")})
	})
	for _, expected := range []string{"delivery_receivable_id =", "row_version =", "lifecycle_status =", "payment_date_utc IS NULL"} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("payment update SQL = %s, missing %q", sql, expected)
		}
	}
}

func TestApplyFiltersAddsInvoicePredicate(t *testing.T) {
	db := dryRunSQLServer(t)
	query := db.Table("dbo.delivery_receivables AS r")
	statement := applyFilters(query, ListQuery{Invoice: "0220"}, time.Time{}, time.Time{}).Find(&[]receivableModel{}).Statement
	if !strings.Contains(statement.SQL.String(), "r.invoice_number LIKE @") {
		t.Fatalf("invoice filter SQL = %s, want invoice predicate", statement.SQL.String())
	}
	if len(statement.Vars) < 2 || statement.Vars[0] != "%0220%" {
		t.Fatalf("invoice filter vars = %#v, want invoice pattern", statement.Vars)
	}
}

func dryRunSQLServer(t *testing.T) *gorm.DB {
	t.Helper()
	sqlDB, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	db, err := gorm.Open(sqlserver.New(sqlserver.Config{Conn: sqlDB}), &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	return db
}
