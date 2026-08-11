package money

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Amount
	}{
		{name: "whole", input: "9800", want: 98_000_000},
		{name: "four decimals", input: "9800.3439", want: 98_003_439},
		{name: "round up", input: "1.23495", want: 12_350},
		{name: "round down", input: "1.23494", want: 12_349},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("Parse() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestParseRejectsInvalidAmounts(t *testing.T) {
	for _, input := range []string{"", "-1", "+1", ".50", "1.2.3", "1,x"} {
		t.Run(input, func(t *testing.T) {
			if _, err := Parse(input); err == nil {
				t.Fatalf("Parse(%q) expected an error", input)
			}
		})
	}
}

func TestFormat(t *testing.T) {
	for _, tt := range []struct {
		amount Amount
		want   string
	}{
		{amount: 98_003_439, want: "9800.34"},
		{amount: 12_345, want: "1.23"},
		{amount: 12_350, want: "1.24"},
	} {
		if got := tt.amount.Format(); got != tt.want {
			t.Fatalf("Format() = %q, want %q", got, tt.want)
		}
	}
}
