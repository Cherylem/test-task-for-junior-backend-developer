package tasktemplate

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"example.com/taskservice/internal/domain/dateonly"
	taskdomain "example.com/taskservice/internal/domain/task"
	templatedomain "example.com/taskservice/internal/domain/tasktemplate"
)

type Service struct {
	templates TemplateRepository
	tasks     GeneratedTaskRepository
	txManager TransactionManager
	now       func() time.Time
}

func NewService(templates TemplateRepository, tasks GeneratedTaskRepository, txManager TransactionManager) *Service {
	return &Service{
		templates: templates,
		tasks:     tasks,
		txManager: txManager,
		now:       func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*templatedomain.Template, error) {
	normalized, err := validateInput(input)
	if err != nil {
		return nil, err
	}

	now := s.now()
	model := &templatedomain.Template{
		Title:             normalized.Title,
		Description:       normalized.Description,
		Timezone:          normalized.Timezone,
		StartDate:         normalized.StartDate,
		EndDate:           normalized.EndDate,
		RecurrenceType:    normalized.RecurrenceType,
		RecurrenceSetting: normalized.RecurrenceSettings,
		IsActive:          true,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	var created *templatedomain.Template
	err = s.txManager.WithinTransaction(ctx, func(templates TemplateRepository, tasks GeneratedTaskRepository) error {
		var txErr error
		created, txErr = templates.Create(ctx, model)
		if txErr != nil {
			return txErr
		}

		return s.syncTemplate(ctx, *created, tasks)
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*templatedomain.Template, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.templates.GetByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, id int64, input UpdateInput) (*templatedomain.Template, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	normalized, err := validateInput(CreateInput(input))
	if err != nil {
		return nil, err
	}

	now := s.now()
	model := &templatedomain.Template{
		ID:                id,
		Title:             normalized.Title,
		Description:       normalized.Description,
		Timezone:          normalized.Timezone,
		StartDate:         normalized.StartDate,
		EndDate:           normalized.EndDate,
		RecurrenceType:    normalized.RecurrenceType,
		RecurrenceSetting: normalized.RecurrenceSettings,
		IsActive:          true,
		UpdatedAt:         now,
	}

	var updated *templatedomain.Template
	err = s.txManager.WithinTransaction(ctx, func(templates TemplateRepository, tasks GeneratedTaskRepository) error {
		var txErr error
		updated, txErr = templates.Update(ctx, model)
		if txErr != nil {
			return txErr
		}

		today, txErr := s.todayInTimezone(updated.Timezone)
		if txErr != nil {
			return txErr
		}

		if txErr := tasks.DeleteFutureByTemplate(ctx, updated.ID, today); txErr != nil {
			return txErr
		}

		return s.syncTemplate(ctx, *updated, tasks)
	})
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.txManager.WithinTransaction(ctx, func(templates TemplateRepository, tasks GeneratedTaskRepository) error {
		template, err := templates.GetByID(ctx, id)
		if err != nil {
			return err
		}

		today, err := s.todayInTimezone(template.Timezone)
		if err != nil {
			return err
		}

		if err := tasks.DeleteFutureByTemplate(ctx, id, today); err != nil {
			return err
		}

		return templates.Deactivate(ctx, id, s.now())
	})
}

func (s *Service) List(ctx context.Context) ([]templatedomain.Template, error) {
	return s.templates.List(ctx)
}

func (s *Service) SyncGeneratedTasks(ctx context.Context) error {
	templates, err := s.templates.ListActive(ctx)
	if err != nil {
		return err
	}

	var syncErr error
	for i := range templates {
		if err := s.syncTemplate(ctx, templates[i], s.tasks); err != nil {
			syncErr = errors.Join(syncErr, fmt.Errorf("sync template %d: %w", templates[i].ID, err))
		}
	}

	return syncErr
}

type validatedInput struct {
	Title              string
	Description        string
	Timezone           string
	StartDate          dateonly.Date
	EndDate            *dateonly.Date
	RecurrenceType     templatedomain.RecurrenceType
	RecurrenceSettings templatedomain.RecurrenceSettings
}

func validateInput(input CreateInput) (validatedInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	input.Timezone = strings.TrimSpace(input.Timezone)

	if input.Title == "" {
		return validatedInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if input.Timezone == "" {
		return validatedInput{}, fmt.Errorf("%w: timezone is required", ErrInvalidInput)
	}

	if _, err := time.LoadLocation(input.Timezone); err != nil {
		return validatedInput{}, fmt.Errorf("%w: invalid timezone", ErrInvalidInput)
	}

	startDate, err := dateonly.Parse(input.StartDate)
	if err != nil {
		return validatedInput{}, fmt.Errorf("%w: start_date is invalid", ErrInvalidInput)
	}

	var endDate *dateonly.Date
	if input.EndDate != nil && strings.TrimSpace(*input.EndDate) != "" {
		parsed, err := dateonly.Parse(strings.TrimSpace(*input.EndDate))
		if err != nil {
			return validatedInput{}, fmt.Errorf("%w: end_date is invalid", ErrInvalidInput)
		}

		endDate = &parsed
		if endDate.Before(startDate) {
			return validatedInput{}, fmt.Errorf("%w: end_date must not be before start_date", ErrInvalidInput)
		}
	}

	if !input.RecurrenceType.Valid() {
		return validatedInput{}, fmt.Errorf("%w: invalid recurrence_type", ErrInvalidInput)
	}

	settings, err := validateSettings(input.RecurrenceType, input.RecurrenceSettings)
	if err != nil {
		return validatedInput{}, err
	}

	return validatedInput{
		Title:              input.Title,
		Description:        input.Description,
		Timezone:           input.Timezone,
		StartDate:          startDate,
		EndDate:            endDate,
		RecurrenceType:     input.RecurrenceType,
		RecurrenceSettings: settings,
	}, nil
}

func validateSettings(recurrenceType templatedomain.RecurrenceType, input RecurrenceSettingsInput) (templatedomain.RecurrenceSettings, error) {
	settings := templatedomain.RecurrenceSettings{}

	switch recurrenceType {
	case templatedomain.RecurrenceDaily:
		if input.EveryNDays <= 0 {
			return settings, fmt.Errorf("%w: every_n_days must be positive", ErrInvalidInput)
		}
		if input.DayOfMonth != 0 || input.ShortMonthStrategy != "" || len(input.SpecificDates) > 0 || input.EvenOddMode != "" || len(input.Weekdays) > 0 {
			return settings, fmt.Errorf("%w: unsupported recurrence settings for daily type", ErrInvalidInput)
		}
		settings.EveryNDays = input.EveryNDays
	case templatedomain.RecurrenceMonthly:
		if input.DayOfMonth < 1 || input.DayOfMonth > 31 {
			return settings, fmt.Errorf("%w: day_of_month must be between 1 and 31", ErrInvalidInput)
		}
		if !input.ShortMonthStrategy.Valid() {
			return settings, fmt.Errorf("%w: invalid short_month_strategy", ErrInvalidInput)
		}
		if input.EveryNDays != 0 || len(input.SpecificDates) > 0 || input.EvenOddMode != "" || len(input.Weekdays) > 0 {
			return settings, fmt.Errorf("%w: unsupported recurrence settings for monthly type", ErrInvalidInput)
		}
		settings.DayOfMonth = input.DayOfMonth
		settings.ShortMonthStrategy = input.ShortMonthStrategy
	case templatedomain.RecurrenceSpecificDates:
		if input.EveryNDays != 0 || input.DayOfMonth != 0 || input.ShortMonthStrategy != "" || input.EvenOddMode != "" || len(input.Weekdays) > 0 {
			return settings, fmt.Errorf("%w: unsupported recurrence settings for specific_dates type", ErrInvalidInput)
		}
		if len(input.SpecificDates) == 0 {
			return settings, fmt.Errorf("%w: specific dates are required", ErrInvalidInput)
		}
		dates := make([]dateonly.Date, 0, len(input.SpecificDates))
		seen := make(map[string]struct{}, len(input.SpecificDates))
		for _, raw := range input.SpecificDates {
			parsed, err := dateonly.Parse(strings.TrimSpace(raw))
			if err != nil {
				return settings, fmt.Errorf("%w: invalid specific date", ErrInvalidInput)
			}
			if _, exists := seen[parsed.String()]; exists {
				continue
			}
			seen[parsed.String()] = struct{}{}
			dates = append(dates, parsed)
		}
		slices.SortFunc(dates, func(a, b dateonly.Date) int { return a.Compare(b) })
		settings.SpecificDates = dates
	case templatedomain.RecurrenceEvenOdd:
		if input.EveryNDays != 0 || input.DayOfMonth != 0 || input.ShortMonthStrategy != "" || len(input.SpecificDates) > 0 || len(input.Weekdays) > 0 {
			return settings, fmt.Errorf("%w: unsupported recurrence settings for even_odd type", ErrInvalidInput)
		}
		if !input.EvenOddMode.Valid() {
			return settings, fmt.Errorf("%w: invalid even_odd mode", ErrInvalidInput)
		}
		settings.EvenOddMode = input.EvenOddMode
	case templatedomain.RecurrenceWeekdays:
		if input.EveryNDays != 0 || input.DayOfMonth != 0 || input.ShortMonthStrategy != "" || len(input.SpecificDates) > 0 || input.EvenOddMode != "" {
			return settings, fmt.Errorf("%w: unsupported recurrence settings for weekdays type", ErrInvalidInput)
		}
		if len(input.Weekdays) == 0 {
			return settings, fmt.Errorf("%w: weekdays are required", ErrInvalidInput)
		}
		weekdays := make([]templatedomain.Weekday, 0, len(input.Weekdays))
		seen := make(map[templatedomain.Weekday]struct{}, len(input.Weekdays))
		for _, weekday := range input.Weekdays {
			if !weekday.Valid() {
				return settings, fmt.Errorf("%w: invalid weekday", ErrInvalidInput)
			}
			if _, exists := seen[weekday]; exists {
				continue
			}
			seen[weekday] = struct{}{}
			weekdays = append(weekdays, weekday)
		}
		slices.SortFunc(weekdays, func(a, b templatedomain.Weekday) int {
			return int(a.ToTimeWeekday()) - int(b.ToTimeWeekday())
		})
		settings.Weekdays = weekdays
	}

	return settings, nil
}

func (s *Service) syncTemplate(ctx context.Context, template templatedomain.Template, tasks GeneratedTaskRepository) error {
	if !template.IsActive {
		return nil
	}

	location, err := time.LoadLocation(template.Timezone)
	if err != nil {
		return err
	}

	today := dateonly.FromTime(s.now().In(location))
	start := today
	if template.StartDate.After(start) {
		start = template.StartDate
	}

	end := today.AddDays(30, location)
	if template.EndDate != nil && template.EndDate.Before(end) {
		end = *template.EndDate
	}
	if end.Before(start) {
		return nil
	}

	dates := buildOccurrences(template, start, end, location)
	now := s.now()
	for _, scheduledDate := range dates {
		scheduledAt := scheduledDate.ToTime(time.UTC)
		task := &taskdomain.Task{
			Title:        template.Title,
			Description:  template.Description,
			Status:       taskdomain.StatusNew,
			TemplateID:   &template.ID,
			ScheduledFor: &scheduledAt,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := tasks.CreateGenerated(ctx, task); err != nil {
			return err
		}
	}

	return nil
}

func buildOccurrences(template templatedomain.Template, start, end dateonly.Date, location *time.Location) []dateonly.Date {
	switch template.RecurrenceType {
	case templatedomain.RecurrenceDaily:
		return buildDaily(template, start, end, location)
	case templatedomain.RecurrenceMonthly:
		return buildMonthly(template, start, end, location)
	case templatedomain.RecurrenceSpecificDates:
		return buildSpecificDates(template, start, end)
	case templatedomain.RecurrenceEvenOdd:
		return buildEvenOdd(template, start, end, location)
	case templatedomain.RecurrenceWeekdays:
		return buildWeekdays(template, start, end, location)
	default:
		return nil
	}
}

func buildDaily(template templatedomain.Template, start, end dateonly.Date, location *time.Location) []dateonly.Date {
	result := make([]dateonly.Date, 0)
	for date := template.StartDate; !date.After(end); date = date.AddDays(template.RecurrenceSetting.EveryNDays, location) {
		if !date.Before(start) {
			result = append(result, date)
		}
	}
	return result
}

func buildMonthly(template templatedomain.Template, start, end dateonly.Date, location *time.Location) []dateonly.Date {
	result := make([]dateonly.Date, 0)
	current := dateonly.Date{Year: start.Year, Month: start.Month, Day: 1}
	last := dateonly.Date{Year: end.Year, Month: end.Month, Day: 1}
	for !current.After(last) {
		daysInMonth := time.Date(current.Year, current.Month+1, 0, 0, 0, 0, 0, location).Day()
		targetDay := template.RecurrenceSetting.DayOfMonth
		if targetDay > daysInMonth {
			if template.RecurrenceSetting.ShortMonthStrategy == templatedomain.ShortMonthSkip {
				current = dateonly.FromTime(current.ToTime(location).AddDate(0, 1, 0))
				continue
			}
			targetDay = daysInMonth
		}
		candidate := dateonly.Date{Year: current.Year, Month: current.Month, Day: targetDay}
		if !candidate.Before(template.StartDate) && !candidate.Before(start) && !candidate.After(end) {
			result = append(result, candidate)
		}
		current = dateonly.FromTime(current.ToTime(location).AddDate(0, 1, 0))
	}
	return result
}

func buildSpecificDates(template templatedomain.Template, start, end dateonly.Date) []dateonly.Date {
	result := make([]dateonly.Date, 0)
	for _, date := range template.RecurrenceSetting.SpecificDates {
		if date.Before(template.StartDate) || date.Before(start) || date.After(end) {
			continue
		}
		if template.EndDate != nil && date.After(*template.EndDate) {
			continue
		}
		result = append(result, date)
	}
	return result
}

func buildEvenOdd(template templatedomain.Template, start, end dateonly.Date, location *time.Location) []dateonly.Date {
	result := make([]dateonly.Date, 0)
	wantEven := template.RecurrenceSetting.EvenOddMode == templatedomain.EvenOddEven
	for date := start; !date.After(end); date = date.AddDays(1, location) {
		if date.Before(template.StartDate) {
			continue
		}
		isEven := date.Day%2 == 0
		if isEven == wantEven {
			result = append(result, date)
		}
	}
	return result
}

func buildWeekdays(template templatedomain.Template, start, end dateonly.Date, location *time.Location) []dateonly.Date {
	result := make([]dateonly.Date, 0)
	allowed := make(map[time.Weekday]struct{}, len(template.RecurrenceSetting.Weekdays))
	for _, weekday := range template.RecurrenceSetting.Weekdays {
		allowed[weekday.ToTimeWeekday()] = struct{}{}
	}
	for date := start; !date.After(end); date = date.AddDays(1, location) {
		if date.Before(template.StartDate) {
			continue
		}
		if _, ok := allowed[date.ToTime(location).Weekday()]; ok {
			result = append(result, date)
		}
	}
	return result
}

func (s *Service) todayInTimezone(timezone string) (dateonly.Date, error) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return dateonly.Date{}, err
	}
	return dateonly.FromTime(s.now().In(location)), nil
}
