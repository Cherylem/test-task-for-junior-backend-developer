package tasktemplate

import (
	"context"
	"time"

	"example.com/taskservice/internal/domain/dateonly"
	taskdomain "example.com/taskservice/internal/domain/task"
	templatedomain "example.com/taskservice/internal/domain/tasktemplate"
)

type TemplateRepository interface {
	Create(ctx context.Context, template *templatedomain.Template) (*templatedomain.Template, error)
	GetByID(ctx context.Context, id int64) (*templatedomain.Template, error)
	Update(ctx context.Context, template *templatedomain.Template) (*templatedomain.Template, error)
	Deactivate(ctx context.Context, id int64, updatedAt time.Time) error
	List(ctx context.Context) ([]templatedomain.Template, error)
	ListActive(ctx context.Context) ([]templatedomain.Template, error)
}

type GeneratedTaskRepository interface {
	CreateGenerated(ctx context.Context, task *taskdomain.Task) error
	DeleteFutureByTemplate(ctx context.Context, templateID int64, fromDate dateonly.Date) error
}

type TransactionManager interface {
	WithinTransaction(ctx context.Context, fn func(TemplateRepository, GeneratedTaskRepository) error) error
}

type Usecase interface {
	Create(ctx context.Context, input CreateInput) (*templatedomain.Template, error)
	GetByID(ctx context.Context, id int64) (*templatedomain.Template, error)
	Update(ctx context.Context, id int64, input UpdateInput) (*templatedomain.Template, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context) ([]templatedomain.Template, error)
	SyncGeneratedTasks(ctx context.Context) error
}

type RecurrenceSettingsInput struct {
	EveryNDays         int
	DayOfMonth         int
	ShortMonthStrategy templatedomain.ShortMonthStrategy
	SpecificDates      []string
	EvenOddMode        templatedomain.EvenOddMode
	Weekdays           []templatedomain.Weekday
}

type CreateInput struct {
	Title              string
	Description        string
	Timezone           string
	StartDate          string
	EndDate            *string
	RecurrenceType     templatedomain.RecurrenceType
	RecurrenceSettings RecurrenceSettingsInput
}

type UpdateInput struct {
	Title              string
	Description        string
	Timezone           string
	StartDate          string
	EndDate            *string
	RecurrenceType     templatedomain.RecurrenceType
	RecurrenceSettings RecurrenceSettingsInput
}
