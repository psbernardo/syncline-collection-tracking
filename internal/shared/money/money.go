package money

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

const scale int64 = 10_000

// Amount stores PHP in ten-thousandths of a peso.
type Amount int64

func Parse(value string) (Amount, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "-") || strings.HasPrefix(value, "+") {
		return 0, fmt.Errorf("amount must be a non-negative number")
	}

	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" {
		return 0, fmt.Errorf("invalid amount %q", value)
	}
	if _, err := strconv.ParseUint(parts[0], 10, 64); err != nil {
		return 0, fmt.Errorf("invalid amount %q: %w", value, err)
	}

	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
	}
	for _, r := range fraction {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("invalid amount %q", value)
		}
	}

	if len(fraction) < 4 {
		fraction += strings.Repeat("0", 4-len(fraction))
	}
	scaledFraction, err := strconv.ParseUint(fraction[:4], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount %q: %w", value, err)
	}
	if len(fraction) > 4 && fraction[4] >= '5' {
		scaledFraction++
	}

	whole, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil || whole > uint64(math.MaxInt64)/uint64(scale) {
		return 0, fmt.Errorf("amount %q is too large", value)
	}
	if scaledFraction == uint64(scale) {
		whole++
		scaledFraction = 0
	}
	scaled := whole*uint64(scale) + scaledFraction
	if scaled > uint64(math.MaxInt64) {
		return 0, fmt.Errorf("amount %q is too large", value)
	}
	return Amount(scaled), nil
}

// Format returns a two-decimal PHP display value.
func (a Amount) Format() string {
	whole := int64(a) / scale
	remainder := int64(a) % scale
	cents := (remainder + 50) / 100
	if cents == 100 {
		whole++
		cents = 0
	}
	return fmt.Sprintf("%d.%02d", whole, cents)
}

func (a Amount) Int64() int64 { return int64(a) }
