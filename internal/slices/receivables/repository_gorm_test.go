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
