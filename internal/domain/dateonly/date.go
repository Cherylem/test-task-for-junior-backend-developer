package dateonly

import (
	"fmt"
	"time"
)

const layout = "2006-01-02"

type Date struct {
	Year  int
	Month time.Month
	Day   int
}

func Parse(value string) (Date, error) {
	if value == "" {
		return Date{}, fmt.Errorf("date is required")
	}

	parsed, err := time.Parse(layout, value)
	if err != nil {
		return Date{}, fmt.Errorf("invalid date %q: %w", value, err)
	}

	return FromTime(parsed), nil
}

func FromTime(value time.Time) Date {
	year, month, day := value.Date()

	return Date{
		Year:  year,
		Month: month,
		Day:   day,
	}
}

func (d Date) String() string {
	if d.IsZero() {
		return ""
	}

	return fmt.Sprintf("%04d-%02d-%02d", d.Year, d.Month, d.Day)
}

func (d Date) IsZero() bool {
	return d.Year == 0 && d.Month == 0 && d.Day == 0
}

func (d Date) Compare(other Date) int {
	if d.Year != other.Year {
		if d.Year < other.Year {
			return -1
		}

		return 1
	}

	if d.Month != other.Month {
		if d.Month < other.Month {
			return -1
		}

		return 1
	}

	if d.Day != other.Day {
		if d.Day < other.Day {
			return -1
		}

		return 1
	}

	return 0
}

func (d Date) Before(other Date) bool {
	return d.Compare(other) < 0
}

func (d Date) After(other Date) bool {
	return d.Compare(other) > 0
}

func (d Date) Equal(other Date) bool {
	return d.Compare(other) == 0
}

func (d Date) ToTime(location *time.Location) time.Time {
	if location == nil {
		location = time.UTC
	}

	return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, location)
}

func (d Date) AddDays(days int, location *time.Location) Date {
	return FromTime(d.ToTime(location).AddDate(0, 0, days))
}
