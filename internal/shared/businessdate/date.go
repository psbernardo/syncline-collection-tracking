package businessdate

import (
	"fmt"
	"time"
)

const layout = "2006-01-02"

var location, _ = time.LoadLocation("Asia/Manila")

func Parse(value string) (time.Time, error) {
	localDate, err := time.ParseInLocation(layout, value, location)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid business date %q: %w", value, err)
	}
	return localDate.UTC(), nil
}

func FormatUTC(value time.Time) string {
	return value.In(location).Format(layout)
}

func DueDate(deliveryUTC time.Time, termDays int) (time.Time, error) {
	if termDays < 1 || termDays > 120 {
		return time.Time{}, fmt.Errorf("payment term must be between 1 and 120 days")
	}
	localDelivery := deliveryUTC.In(location)
	dueLocal := time.Date(localDelivery.Year(), localDelivery.Month(), localDelivery.Day(), 0, 0, 0, 0, location)
	dueLocal = dueLocal.AddDate(0, 0, termDays-1)
	return dueLocal.UTC(), nil
}

// ClassificationBoundaries returns the start of today and the end of the
// near-due window using the application's business timezone.
func ClassificationBoundaries(now time.Time) (today, nearEnd time.Time) {
	today, _ = Parse(FormatUTC(now))
	nearEnd, _ = DueDate(today, 6)
	return today, nearEnd
}
