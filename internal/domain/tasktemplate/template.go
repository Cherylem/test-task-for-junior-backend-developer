package tasktemplate

import (
	"time"

	"example.com/taskservice/internal/domain/dateonly"
)

type RecurrenceType string

const (
	RecurrenceDaily         RecurrenceType = "daily"
	RecurrenceMonthly       RecurrenceType = "monthly"
	RecurrenceSpecificDates RecurrenceType = "specific_dates"
	RecurrenceEvenOdd       RecurrenceType = "even_odd"
	RecurrenceWeekdays      RecurrenceType = "weekdays"
)

type ShortMonthStrategy string

const (
	ShortMonthSkip    ShortMonthStrategy = "skip"
	ShortMonthLastDay ShortMonthStrategy = "last_day"
)

type EvenOddMode string

const (
	EvenOddEven EvenOddMode = "even"
	EvenOddOdd  EvenOddMode = "odd"
)

type Weekday string

const (
	WeekdayMonday    Weekday = "monday"
	WeekdayTuesday   Weekday = "tuesday"
	WeekdayWednesday Weekday = "wednesday"
	WeekdayThursday  Weekday = "thursday"
	WeekdayFriday    Weekday = "friday"
	WeekdaySaturday  Weekday = "saturday"
	WeekdaySunday    Weekday = "sunday"
)

type RecurrenceSettings struct {
	EveryNDays         int
	DayOfMonth         int
	ShortMonthStrategy ShortMonthStrategy
	SpecificDates      []dateonly.Date
	EvenOddMode        EvenOddMode
	Weekdays           []Weekday
}

type Template struct {
	ID                int64
	Title             string
	Description       string
	Timezone          string
	StartDate         dateonly.Date
	EndDate           *dateonly.Date
	RecurrenceType    RecurrenceType
	RecurrenceSetting RecurrenceSettings
	IsActive          bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (r RecurrenceType) Valid() bool {
	switch r {
	case RecurrenceDaily, RecurrenceMonthly, RecurrenceSpecificDates, RecurrenceEvenOdd, RecurrenceWeekdays:
		return true
	default:
		return false
	}
}

func (s ShortMonthStrategy) Valid() bool {
	switch s {
	case ShortMonthSkip, ShortMonthLastDay:
		return true
	default:
		return false
	}
}

func (m EvenOddMode) Valid() bool {
	switch m {
	case EvenOddEven, EvenOddOdd:
		return true
	default:
		return false
	}
}

func (w Weekday) Valid() bool {
	switch w {
	case WeekdayMonday, WeekdayTuesday, WeekdayWednesday, WeekdayThursday, WeekdayFriday, WeekdaySaturday, WeekdaySunday:
		return true
	default:
		return false
	}
}

func (w Weekday) ToTimeWeekday() time.Weekday {
	switch w {
	case WeekdayMonday:
		return time.Monday
	case WeekdayTuesday:
		return time.Tuesday
	case WeekdayWednesday:
		return time.Wednesday
	case WeekdayThursday:
		return time.Thursday
	case WeekdayFriday:
		return time.Friday
	case WeekdaySaturday:
		return time.Saturday
	default:
		return time.Sunday
	}
}
