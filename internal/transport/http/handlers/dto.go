package handlers

import (
	"time"

	"example.com/taskservice/internal/domain/dateonly"
	taskdomain "example.com/taskservice/internal/domain/task"
	templatedomain "example.com/taskservice/internal/domain/tasktemplate"
)

type taskMutationDTO struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      taskdomain.Status `json:"status"`
}

type taskDTO struct {
	ID           int64             `json:"id"`
	Title        string            `json:"title"`
	Description  string            `json:"description"`
	Status       taskdomain.Status `json:"status"`
	TemplateID   *int64            `json:"template_id,omitempty"`
	ScheduledFor *string           `json:"scheduled_for,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

type recurrenceSettingsDTO struct {
	EveryNDays         int                               `json:"every_n_days,omitempty"`
	DayOfMonth         int                               `json:"day_of_month,omitempty"`
	ShortMonthStrategy templatedomain.ShortMonthStrategy `json:"short_month_strategy,omitempty"`
	SpecificDates      []string                          `json:"specific_dates,omitempty"`
	EvenOddMode        templatedomain.EvenOddMode        `json:"even_odd_mode,omitempty"`
	Weekdays           []templatedomain.Weekday          `json:"weekdays,omitempty"`
}

type taskTemplateMutationDTO struct {
	Title              string                        `json:"title"`
	Description        string                        `json:"description"`
	Timezone           string                        `json:"timezone"`
	StartDate          string                        `json:"start_date"`
	EndDate            *string                       `json:"end_date"`
	RecurrenceType     templatedomain.RecurrenceType `json:"recurrence_type"`
	RecurrenceSettings recurrenceSettingsDTO         `json:"recurrence_settings"`
}

type taskTemplateDTO struct {
	ID                 int64                         `json:"id"`
	Title              string                        `json:"title"`
	Description        string                        `json:"description"`
	Timezone           string                        `json:"timezone"`
	StartDate          string                        `json:"start_date"`
	EndDate            string                        `json:"end_date,omitempty"`
	RecurrenceType     templatedomain.RecurrenceType `json:"recurrence_type"`
	RecurrenceSettings recurrenceSettingsDTO         `json:"recurrence_settings"`
	CreatedAt          time.Time                     `json:"created_at"`
	UpdatedAt          time.Time                     `json:"updated_at"`
}

func newTaskDTO(task *taskdomain.Task) taskDTO {
	var scheduledFor string
	var scheduledForPtr *string
	if task.ScheduledFor != nil {
		scheduledFor = dateonly.FromTime(*task.ScheduledFor).String()
		scheduledForPtr = &scheduledFor
	}

	return taskDTO{
		ID:           task.ID,
		Title:        task.Title,
		Description:  task.Description,
		Status:       task.Status,
		TemplateID:   task.TemplateID,
		ScheduledFor: scheduledForPtr,
		CreatedAt:    task.CreatedAt,
		UpdatedAt:    task.UpdatedAt,
	}
}

func newTaskTemplateDTO(template *templatedomain.Template) taskTemplateDTO {
	dto := taskTemplateDTO{
		ID:             template.ID,
		Title:          template.Title,
		Description:    template.Description,
		Timezone:       template.Timezone,
		StartDate:      template.StartDate.String(),
		RecurrenceType: template.RecurrenceType,
		RecurrenceSettings: recurrenceSettingsDTO{
			EveryNDays:         template.RecurrenceSetting.EveryNDays,
			DayOfMonth:         template.RecurrenceSetting.DayOfMonth,
			ShortMonthStrategy: template.RecurrenceSetting.ShortMonthStrategy,
			EvenOddMode:        template.RecurrenceSetting.EvenOddMode,
			Weekdays:           template.RecurrenceSetting.Weekdays,
		},
		CreatedAt: template.CreatedAt,
		UpdatedAt: template.UpdatedAt,
	}

	if template.EndDate != nil {
		dto.EndDate = template.EndDate.String()
	}

	if len(template.RecurrenceSetting.SpecificDates) > 0 {
		dto.RecurrenceSettings.SpecificDates = make([]string, 0, len(template.RecurrenceSetting.SpecificDates))
		for _, value := range template.RecurrenceSetting.SpecificDates {
			dto.RecurrenceSettings.SpecificDates = append(dto.RecurrenceSettings.SpecificDates, value.String())
		}
	}

	return dto
}
