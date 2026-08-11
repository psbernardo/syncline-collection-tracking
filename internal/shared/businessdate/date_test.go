package businessdate

import (
	"testing"
	"time"
)

func TestParseAndFormatUTC(t *testing.T) {
	got, err := Parse("2026-08-10")
	if err != nil {
		t.Fatal(err)
	}
	if got.Location() != time.UTC {
		t.Fatalf("Parse() location = %v, want UTC", got.Location())
	}
	if formatted := FormatUTC(got); formatted != "2026-08-10" {
		t.Fatalf("FormatUTC() = %q, want 2026-08-10", formatted)
	}
}

func TestDueDateCountsDeliveryAsDayOne(t *testing.T) {
	delivery, err := Parse("2026-08-10")
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		term int
		days string
	}{
		{term: 1, days: "2026-08-10"},
		{term: 5, days: "2026-08-14"},
		{term: 120, days: "2026-12-07"},
	} {
		due, err := DueDate(delivery, tt.term)
		if err != nil {
			t.Fatal(err)
		}
		if got := FormatUTC(due); got != tt.days {
			t.Errorf("term %d: due date = %s, want %s", tt.term, got, tt.days)
		}
	}
}

func TestDueDateRejectsInvalidTerms(t *testing.T) {
	delivery, _ := Parse("2026-08-10")
	for _, term := range []int{0, 121} {
		if _, err := DueDate(delivery, term); err == nil {
			t.Errorf("DueDate(%d) expected an error", term)
		}
	}
}
